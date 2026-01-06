package gitlab

import (
	"strings"
	"testing"
	"unicode/utf8"

	gl "gitlab.com/gitlab-org/api/client-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTruncator_TruncateString(t *testing.T) {
	truncator := NewTextTruncator(300)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Short string - no truncation",
			input:    "Short description",
			expected: "Short description",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Exactly max length - no truncation",
			input:    strings.Repeat("a", 300),
			expected: strings.Repeat("a", 300),
		},
		{
			name:     "One over max length - should truncate",
			input:    strings.Repeat("a", 301),
			expected: strings.Repeat("a", 300) + "...",
		},
		{
			name:     "Much longer than max length",
			input:    strings.Repeat("a", 500),
			expected: strings.Repeat("a", 300) + "...",
		},
		{
			name:     "Unicode characters - Chinese",
			input:    "你好世界你好世界" + strings.Repeat("a", 350),
			expected: "你好世界你好世界" + strings.Repeat("a", 292) + "...",
		},
		{
			name:     "Unicode characters - Emoji",
			input:    "😀😁😂🤣😃😄😅😆" + strings.Repeat("a", 350),
			expected: "😀😁😂🤣😃😄😅😆" + strings.Repeat("a", 292) + "...",
		},
		{
			name:     "Mixed unicode and ASCII",
			input:    "Hello世界🌍" + strings.Repeat("b", 350),
			expected: "Hello世界🌍" + strings.Repeat("b", 292) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncator.truncateString(tt.input)
			assert.Equal(t, tt.expected, result)

			// Verify that if truncated, it ends with suffix and is correct length in runes
			if utf8.RuneCountInString(result) > 300 {
				assert.True(t, strings.HasSuffix(result, "..."), "Truncated string should end with '...'")
				assert.Equal(t, 303, utf8.RuneCountInString(result), "Truncated string should be 303 runes (300 + '...')")
			}
		})
	}
}

func TestTruncator_TruncateListResponse_Issues(t *testing.T) {
	truncator := NewTextTruncator(300)

	longDesc := strings.Repeat("a", 500)
	shortDesc := "Short description"
	issues := []*gl.Issue{
		{
			ID:          1,
			IID:         101,
			Title:       "Test Issue",
			Description: longDesc,
		},
		{
			ID:          2,
			IID:         102,
			Title:       "Short Issue",
			Description: shortDesc,
		},
		{
			ID:          3,
			IID:         103,
			Title:       "Empty Description Issue",
			Description: "",
		},
	}

	result, err := truncator.TruncateListResponse(issues, IssueFields)
	require.NoError(t, err)

	// Verify result is a slice
	resultSlice, ok := result.([]map[string]interface{})
	require.True(t, ok, "Result should be a slice of maps")
	require.Len(t, resultSlice, 3, "Should have 3 issues")

	// Check first issue (long description truncated)
	desc1, exists := resultSlice[0]["description"]
	require.True(t, exists, "description field should exist")
	desc1Str, ok := desc1.(string)
	require.True(t, ok, "description should be a string")
	assert.Equal(t, 303, utf8.RuneCountInString(desc1Str), "Long description should be 303 runes")
	assert.True(t, strings.HasSuffix(desc1Str, "..."), "Should end with '...'")
	assert.Equal(t, longDesc[:300]+"...", desc1Str, "Should match first 300 chars + '...'")

	// Check second issue (short description unchanged)
	desc2, exists := resultSlice[1]["description"]
	require.True(t, exists)
	desc2Str, ok := desc2.(string)
	require.True(t, ok)
	assert.Equal(t, "Short description", desc2Str, "Short description should be unchanged")

	// Check third issue (empty description)
	desc3, exists := resultSlice[2]["description"]
	require.True(t, exists, "Empty description should still exist as a field")
	desc3Str, ok := desc3.(string)
	require.True(t, ok)
	assert.Equal(t, "", desc3Str, "Empty description should remain empty")
}

func TestTruncator_TruncateListResponse_MergeRequests(t *testing.T) {
	truncator := NewTextTruncator(300)

	longDesc := strings.Repeat("b", 400)
	mrs := []*gl.MergeRequest{
		{
			BasicMergeRequest: gl.BasicMergeRequest{
				ID:          1,
				IID:         10,
				Title:       "Test MR",
				Description: longDesc,
			},
		},
	}

	result, err := truncator.TruncateListResponse(mrs, MergeRequestFields)
	require.NoError(t, err)

	resultSlice, ok := result.([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, resultSlice, 1)

	desc, exists := resultSlice[0]["description"]
	require.True(t, exists)
	descStr, ok := desc.(string)
	require.True(t, ok)
	assert.Equal(t, 303, utf8.RuneCountInString(descStr))
	assert.True(t, strings.HasSuffix(descStr, "..."))
}

func TestTruncator_TruncateListResponse_Projects(t *testing.T) {
	truncator := NewTextTruncator(300)

	longDesc := strings.Repeat("c", 350)
	projects := []*gl.Project{
		{
			ID:          1,
			Name:        "Test Project",
			Description: longDesc,
		},
	}

	result, err := truncator.TruncateListResponse(projects, ProjectFields)
	require.NoError(t, err)

	resultSlice, ok := result.([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, resultSlice, 1)

	desc, exists := resultSlice[0]["description"]
	require.True(t, exists)
	descStr, ok := desc.(string)
	require.True(t, ok)
	assert.Equal(t, 303, utf8.RuneCountInString(descStr))
}

func TestTruncator_TruncateListResponse_Milestones(t *testing.T) {
	truncator := NewTextTruncator(300)

	longDesc := strings.Repeat("d", 450)
	milestones := []*gl.Milestone{
		{
			ID:          1,
			Title:       "Sprint 1",
			Description: longDesc,
		},
	}

	result, err := truncator.TruncateListResponse(milestones, MilestoneFields)
	require.NoError(t, err)

	resultSlice, ok := result.([]map[string]interface{})
	require.True(t, ok)

	desc, exists := resultSlice[0]["description"]
	require.True(t, exists)
	descStr, ok := desc.(string)
	require.True(t, ok)
	assert.Equal(t, 303, utf8.RuneCountInString(descStr))
}

func TestTruncator_TruncateListResponse_Commits(t *testing.T) {
	truncator := NewTextTruncator(300)

	longMsg := strings.Repeat("e", 500) + "\n\n" + strings.Repeat("f", 200)
	commits := []*gl.Commit{
		{
			ID:      "abc123",
			Message: longMsg,
		},
	}

	result, err := truncator.TruncateListResponse(commits, CommitFields)
	require.NoError(t, err)

	resultSlice, ok := result.([]map[string]interface{})
	require.True(t, ok)

	msg, exists := resultSlice[0]["message"]
	require.True(t, exists)
	msgStr, ok := msg.(string)
	require.True(t, ok)
	assert.Equal(t, 303, utf8.RuneCountInString(msgStr))
	assert.True(t, strings.HasSuffix(msgStr, "..."))
}

func TestTruncator_TruncateListResponse_Users(t *testing.T) {
	truncator := NewTextTruncator(300)

	longBio := strings.Repeat("f", 400)
	users := []*gl.User{
		{
			ID:       1,
			Name:     "Test User",
			Username: "testuser",
			Bio:      longBio,
		},
	}

	result, err := truncator.TruncateListResponse(users, UserFields)
	require.NoError(t, err)

	resultSlice, ok := result.([]map[string]interface{})
	require.True(t, ok)

	bio, exists := resultSlice[0]["bio"]
	require.True(t, exists)
	bioStr, ok := bio.(string)
	require.True(t, ok)
	assert.Equal(t, 303, utf8.RuneCountInString(bioStr))
}

func TestTruncator_TruncateListResponse_NonSlice(t *testing.T) {
	truncator := NewTextTruncator(300)

	// Test with a single object (not a slice)
	longDesc := strings.Repeat("a", 500)
	singleIssue := &gl.Issue{
		ID:          1,
		Title:       "Test",
		Description: longDesc,
	}

	result, err := truncator.TruncateListResponse(singleIssue, IssueFields)
	require.NoError(t, err)

	// Should return as-is since it's not a slice
	assert.Equal(t, singleIssue, result, "Non-slice should be returned unchanged")
}

func TestTruncator_TruncateListResponse_EmptySlice(t *testing.T) {
	truncator := NewTextTruncator(300)

	var emptyIssues []*gl.Issue

	result, err := truncator.TruncateListResponse(emptyIssues, IssueFields)
	require.NoError(t, err)

	resultSlice, ok := result.([]map[string]interface{})
	require.True(t, ok)
	assert.Empty(t, resultSlice, "Empty slice should remain empty")
}

func TestTruncator_TruncateListResponse_NoFields(t *testing.T) {
	truncator := NewTextTruncator(300)

	longDesc := strings.Repeat("a", 500)
	issues := []*gl.Issue{
		{
			ID:          1,
			Title:       "Test",
			Description: longDesc,
		},
	}

	// Truncate with no field names
	result, err := truncator.TruncateListResponse(issues, []string{})
	require.NoError(t, err)

	resultSlice, ok := result.([]map[string]interface{})
	require.True(t, ok)

	// Description should NOT be truncated since we didn't specify any fields
	desc, exists := resultSlice[0]["description"]
	require.True(t, exists)
	descStr, ok := desc.(string)
	require.True(t, ok)
	assert.Equal(t, 500, utf8.RuneCountInString(descStr), "Description should not be truncated when no fields specified")
}

func TestTruncator_TruncateListResponse_MultipleFields(t *testing.T) {
	truncator := NewTextTruncator(300)

	// Test with Note fields
	longBody := strings.Repeat("x", 400)
	notes := []*gl.Note{
		{
			ID:   1,
			Body: longBody,
		},
	}

	result, err := truncator.TruncateListResponse(notes, NoteFields)
	require.NoError(t, err)

	resultSlice, ok := result.([]map[string]interface{})
	require.True(t, ok)

	// Body should be truncated
	body, hasBody := resultSlice[0]["body"]
	assert.True(t, hasBody, "Body field should be present")

	if bodyStr, ok := body.(string); ok {
		assert.Equal(t, 303, utf8.RuneCountInString(bodyStr), "Body should be truncated")
		assert.True(t, strings.HasSuffix(bodyStr, "..."), "Should end with '...'")
	}
}
