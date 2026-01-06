package gitlab

import (
	"encoding/json"
	"unicode/utf8"
)

const (
	// MaxFieldLength is the maximum length for text fields in list operations
	MaxFieldLength = 300
	// TruncationSuffix is added to truncated strings
	TruncationSuffix = "..."
)

// TextTruncator handles text field truncation for list operation results
type TextTruncator struct {
	maxLength int
	suffix    string
}

// NewTextTruncator creates a new truncator with specified max length
func NewTextTruncator(maxLength int) *TextTruncator {
	return &TextTruncator{
		maxLength: maxLength,
		suffix:    TruncationSuffix,
	}
}

// truncateString truncates a string to maxLength + suffix if needed
// Uses UTF-8 rune counting for proper unicode support
func (t *TextTruncator) truncateString(s string) string {
	if s == "" {
		return s
	}

	// Check if truncation is needed
	if utf8.RuneCountInString(s) <= t.maxLength {
		return s
	}

	// Truncate and add suffix
	runes := []rune(s)
	if len(runes) > t.maxLength {
		return string(runes[:t.maxLength]) + t.suffix
	}
	return s
}

// TruncateListResponse truncates text fields in list operation results
// This is a generic function that works with any slice type by using JSON marshal/unmarshal
func (t *TextTruncator) TruncateListResponse(data interface{}, fieldNames []string) (interface{}, error) {
	// Convert to JSON and back to map for manipulation
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var slice []map[string]interface{}
	if err := json.Unmarshal(jsonData, &slice); err != nil {
		// If not a slice, return as-is
		return data, nil
	}

	// Truncate specified fields in each item
	for _, item := range slice {
		for _, fieldName := range fieldNames {
			if val, ok := item[fieldName].(string); ok {
				item[fieldName] = t.truncateString(val)
			}
		}
	}

	return slice, nil
}

// Predefined field sets for common GitLab types
// Note: These are JSON field names (snake_case), not Go struct field names
var (
	// IssueFields returns fields to truncate in Issue objects
	IssueFields = []string{"description"}

	// MergeRequestFields returns fields to truncate in MergeRequest objects
	MergeRequestFields = []string{"description"}

	// ProjectFields returns fields to truncate in Project objects
	ProjectFields = []string{"description"}

	// MilestoneFields returns fields to truncate in Milestone objects
	MilestoneFields = []string{"description"}

	// CommitFields returns fields to truncate in Commit objects
	CommitFields = []string{"message"}

	// UserFields returns fields to truncate in User objects
	UserFields = []string{"bio"}

	// BlobFields returns fields to truncate in Blob objects
	BlobFields = []string{"data"}

	// NoteFields returns fields to truncate in Note objects
	NoteFields = []string{"body"}
)
