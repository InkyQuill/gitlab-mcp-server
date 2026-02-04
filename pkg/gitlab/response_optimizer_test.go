package gitlab

import (
	"fmt"
	"net/http"
	"testing"

	gl "gitlab.com/gitlab-org/api/client-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractPagination(t *testing.T) {
	tests := []struct {
		name           string
		headers        map[string]string
		expected       *PaginationMetadata
		description    string
	}{
		{
			name: "Full pagination headers",
			headers: map[string]string{
				"X-Total":       "150",
				"X-Total-Pages": "8",
				"X-Page":        "1",
				"X-Per-Page":    "20",
			},
			expected: &PaginationMetadata{
				TotalItems:   150,
				TotalPages:   8,
				CurrentPage:  1,
				ItemsPerPage: 20,
			},
			description: "Should extract all pagination fields correctly",
		},
		{
			name:     "Nil response",
			headers:  nil,
			expected: nil,
			description: "Should return nil for nil response",
		},
		{
			name: "Empty headers",
			headers: map[string]string{
				"X-Total":       "",
				"X-Total-Pages": "",
				"X-Page":        "",
				"X-Per-Page":    "",
			},
			expected: nil,
			description: "Should return nil when all headers are empty",
		},
		{
			name: "Partial headers",
			headers: map[string]string{
				"X-Total": "100",
			},
			expected: &PaginationMetadata{
				TotalItems: 100,
			},
			description: "Should handle partial headers gracefully",
		},
		{
			name: "First page",
			headers: map[string]string{
				"X-Total":       "50",
				"X-Total-Pages": "3",
				"X-Page":        "1",
				"X-Per-Page":    "20",
			},
			expected: &PaginationMetadata{
				TotalItems:   50,
				TotalPages:   3,
				CurrentPage:  1,
				ItemsPerPage: 20,
			},
			description: "Should correctly identify first page",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a response with the test data
			var resp *gl.Response
			if tt.headers != nil {
				// Parse headers to get values
				totalItems := 0
				totalPages := 0
				currentPage := 0
				itemsPerPage := 0

				if v, ok := tt.headers["X-Total"]; ok && v != "" {
					fmt.Sscanf(v, "%d", &totalItems)
				}
				if v, ok := tt.headers["X-Total-Pages"]; ok && v != "" {
					fmt.Sscanf(v, "%d", &totalPages)
				}
				if v, ok := tt.headers["X-Page"]; ok && v != "" {
					fmt.Sscanf(v, "%d", &currentPage)
				}
				if v, ok := tt.headers["X-Per-Page"]; ok && v != "" {
					fmt.Sscanf(v, "%d", &itemsPerPage)
				}

				resp = &gl.Response{
					Response:    &http.Response{StatusCode: 200},
					TotalItems:  totalItems,
					TotalPages:  totalPages,
					CurrentPage: currentPage,
					ItemsPerPage: itemsPerPage,
				}
			}

			result := ExtractPagination(resp)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestFieldFilter_Issues(t *testing.T) {
	filter := NewFieldFilter("issue")

	// Mock issue data with all fields
	issues := []map[string]interface{}{
		{
			"id":          1405,
			"iid":         147,
			"title":       "Test Issue",
			"description": "Test description",
			"_links": map[string]interface{}{
				"self": "http://example.com/api/v4/projects/1/issues/1",
			},
			"web_url": "http://example.com/projects/1/issues/1",
			"author": map[string]interface{}{
				"id":         2,
				"name":       "Test User",
				"username":   "testuser",
				"state":      "active",
				"web_url":    "http://example.com/testuser",
				"avatar_url": "http://example.com/avatar.png",
			},
			"assignees": []interface{}{
				map[string]interface{}{
					"id":         3,
					"name":       "Assignee",
					"username":   "assignee",
					"state":      "active",
					"web_url":    "http://example.com/assignee",
					"avatar_url": "http://example.com/assignee.png",
				},
			},
			"service_desk_reply_to": "customer@example.com",
			"label_details":         []interface{}{},
			"user_notes_count":      5,
		},
	}

	result, err := filter.FilterResponse(issues)
	require.NoError(t, err)

	resultSlice, ok := result.([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, resultSlice, 1)

	issue := resultSlice[0]

	// Verify fields are removed
	assert.NotContains(t, issue, "_links", "Should remove _links field")
	assert.NotContains(t, issue, "web_url", "Should remove web_url field")
	assert.NotContains(t, issue, "service_desk_reply_to", "Should remove service_desk_reply_to")
	assert.NotContains(t, issue, "label_details", "Should remove label_details")
	assert.NotContains(t, issue, "user_notes_count", "Should remove user_notes_count")

	// Verify nested fields are removed from author
	author, ok := issue["author"].(map[string]interface{})
	require.True(t, ok)
	assert.NotContains(t, author, "web_url", "Should remove author.web_url")
	assert.NotContains(t, author, "avatar_url", "Should remove author.avatar_url")
	assert.Contains(t, author, "id", "Should keep author.id")
	assert.Contains(t, author, "name", "Should keep author.name")
	assert.Contains(t, author, "username", "Should keep author.username")
	assert.Contains(t, author, "state", "Should keep author.state")

	// Verify nested fields are removed from assignees
	assignees, ok := issue["assignees"].([]interface{})
	require.True(t, ok)
	require.Len(t, assignees, 1)
	assignee := assignees[0].(map[string]interface{})
	assert.NotContains(t, assignee, "web_url", "Should remove assignees.web_url")
	assert.NotContains(t, assignee, "avatar_url", "Should remove assignees.avatar_url")

	// Verify important fields are kept
	assert.Contains(t, issue, "id", "Should keep id field")
	assert.Contains(t, issue, "iid", "Should keep iid field")
	assert.Contains(t, issue, "title", "Should keep title field")
	assert.Contains(t, issue, "description", "Should keep description field")
}

func TestFieldFilter_MergeRequests(t *testing.T) {
	filter := NewFieldFilter("merge_request")

	mrs := []map[string]interface{}{
		{
			"id":          1,
			"iid":         10,
			"title":       "Test MR",
			"description": "Test MR description",
			"_links":      map[string]interface{}{},
			"web_url":     "http://example.com/mr/10",
			"author": map[string]interface{}{
				"id":         2,
				"name":       "Author",
				"username":   "author",
				"web_url":    "http://example.com/author",
				"avatar_url": "http://example.com/avatar.png",
			},
			"assignees": []interface{}{
				map[string]interface{}{
					"id":         3,
					"name":       "Assignee",
					"web_url":    "http://example.com/assignee",
					"avatar_url": "http://example.com/assignee.png",
				},
			},
			"reviewers": []interface{}{
				map[string]interface{}{
					"id":         4,
					"name":       "Reviewer",
					"web_url":    "http://example.com/reviewer",
					"avatar_url": "http://example.com/reviewer.png",
				},
			},
			"label_details": []interface{}{},
			"diff_refs":     map[string]interface{}{},
			"references":    map[string]interface{}{},
		},
	}

	result, err := filter.FilterResponse(mrs)
	require.NoError(t, err)

	resultSlice, ok := result.([]map[string]interface{})
	require.True(t, ok)
	mr := resultSlice[0]

	// Verify removed fields
	assert.NotContains(t, mr, "_links")
	assert.NotContains(t, mr, "web_url")
	assert.NotContains(t, mr, "label_details")
	assert.NotContains(t, mr, "diff_refs")
	assert.NotContains(t, mr, "references")

	// Verify author nested fields
	author := mr["author"].(map[string]interface{})
	assert.NotContains(t, author, "web_url")
	assert.NotContains(t, author, "avatar_url")

	// Verify assignees nested fields
	assignees := mr["assignees"].([]interface{})
	assignee := assignees[0].(map[string]interface{})
	assert.NotContains(t, assignee, "web_url")
	assert.NotContains(t, assignee, "avatar_url")

	// Verify reviewers nested fields
	reviewers := mr["reviewers"].([]interface{})
	reviewer := reviewers[0].(map[string]interface{})
	assert.NotContains(t, reviewer, "web_url")
	assert.NotContains(t, reviewer, "avatar_url")

	// Verify kept fields
	assert.Contains(t, mr, "id")
	assert.Contains(t, mr, "iid")
	assert.Contains(t, mr, "title")
}

func TestFieldFilter_Projects(t *testing.T) {
	filter := NewFieldFilter("project")

	projects := []map[string]interface{}{
		{
			"id":                  1,
			"name":                "Test Project",
			"description":         "Test description",
			"web_url":             "http://example.com/project",
			"avatar_url":          "http://example.com/avatar.png",
			"star_count":          10,
			"forks_count":         5,
			"forked_from_project": map[string]interface{}{},
			"last_activity_at":    "2024-01-01T00:00:00Z",
		},
	}

	result, err := filter.FilterResponse(projects)
	require.NoError(t, err)

	resultSlice, ok := result.([]map[string]interface{})
	require.True(t, ok)
	project := resultSlice[0]

	// Verify removed fields
	assert.NotContains(t, project, "web_url")
	assert.NotContains(t, project, "avatar_url")
	assert.NotContains(t, project, "star_count")
	assert.NotContains(t, project, "forks_count")
	assert.NotContains(t, project, "forked_from_project")

	// Verify kept fields
	assert.Contains(t, project, "id")
	assert.Contains(t, project, "name")
	assert.Contains(t, project, "description")
	assert.Contains(t, project, "last_activity_at")
}

func TestFieldFilter_Users(t *testing.T) {
	filter := NewFieldFilter("user")

	users := []map[string]interface{}{
		{
			"id":         1,
			"name":       "Test User",
			"username":   "testuser",
			"email":      "test@example.com",
			"state":      "active",
			"web_url":    "http://example.com/testuser",
			"avatar_url": "http://example.com/avatar.png",
			"website_url": "http://example.com",
			"linkedin":    "linkedin",
			"twitter":     "twitter",
			"skype":       "skype",
			"location":    "Location",
		},
	}

	result, err := filter.FilterResponse(users)
	require.NoError(t, err)

	resultSlice, ok := result.([]map[string]interface{})
	require.True(t, ok)
	user := resultSlice[0]

	// Verify removed fields
	assert.NotContains(t, user, "web_url")
	assert.NotContains(t, user, "avatar_url")
	assert.NotContains(t, user, "website_url")
	assert.NotContains(t, user, "linkedin")
	assert.NotContains(t, user, "twitter")
	assert.NotContains(t, user, "skype")
	assert.NotContains(t, user, "location")

	// Verify kept fields
	assert.Contains(t, user, "id")
	assert.Contains(t, user, "name")
	assert.Contains(t, user, "username")
	assert.Contains(t, user, "email")
	assert.Contains(t, user, "state")
}

func TestResponseOptimizer_Integration(t *testing.T) {
	optimizer := NewResponseOptimizer("issue")

	// Create mock data
	issues := []map[string]interface{}{
		{
			"id":          1,
			"iid":         1,
			"title":       "Test Issue",
			"description": string(make([]byte, 500)), // Long description
			"_links":      map[string]interface{}{},
			"web_url":     "http://example.com",
		},
	}

	// Create mock response with pagination data
	resp := &gl.Response{
		Response:    &http.Response{StatusCode: 200},
		TotalItems:  100,
		TotalPages:  5,
		CurrentPage: 1,
		ItemsPerPage: 20,
	}

	// Optimize response
	result, err := optimizer.OptimizeListResponse(issues, resp)
	require.NoError(t, err)

	// Verify structure
	assert.NotNil(t, result)
	assert.NotNil(t, result.Items)
	assert.NotNil(t, result.Pagination)

	// Verify pagination
	assert.Equal(t, 100, result.Pagination.TotalItems)
	assert.Equal(t, 5, result.Pagination.TotalPages)
	assert.Equal(t, 1, result.Pagination.CurrentPage)
	assert.Equal(t, 20, result.Pagination.ItemsPerPage)

	// Verify items are filtered and truncated
	items, ok := result.Items.([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, items, 1)

	issue := items[0]
	assert.NotContains(t, issue, "_links", "Should remove _links")
	assert.NotContains(t, issue, "web_url", "Should remove web_url")

	// Verify description is truncated
	description, ok := issue["description"].(string)
	require.True(t, ok)
	assert.LessOrEqual(t, len(description), 303, "Description should be truncated to max length + suffix")
}

func TestFieldFilter_NilAssignees(t *testing.T) {
	// Test that nil assignees slice is handled gracefully
	filter := NewFieldFilter("issue")

	issues := []map[string]interface{}{
		{
			"id":       1,
			"title":    "Test Issue",
			"assignees": nil, // nil slice
		},
	}

	result, err := filter.FilterResponse(issues)
	require.NoError(t, err)

	resultSlice, ok := result.([]map[string]interface{})
	require.True(t, ok)
	issue := resultSlice[0]

	assert.Contains(t, issue, "assignees")
	assert.Nil(t, issue["assignees"], "Nil assignees should remain nil")
}

func TestFieldFilter_EmptySlice(t *testing.T) {
	filter := NewFieldFilter("issue")

	issues := []map[string]interface{}{
		{
			"id":       1,
			"title":    "Test Issue",
			"assignees": []interface{}{}, // empty slice
		},
	}

	result, err := filter.FilterResponse(issues)
	require.NoError(t, err)

	resultSlice, ok := result.([]map[string]interface{})
	require.True(t, ok)
	issue := resultSlice[0]

	assert.Contains(t, issue, "assignees")
	assignees, ok := issue["assignees"].([]interface{})
	require.True(t, ok)
	assert.Empty(t, assignees, "Empty assignees should remain empty")
}

func TestExtractPagination_EmptyResponse(t *testing.T) {
	result := ExtractPagination(&gl.Response{})
	assert.Nil(t, result, "Should return nil for response with no headers")
}
