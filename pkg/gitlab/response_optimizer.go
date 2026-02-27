package gitlab

import (
	"encoding/json"
	"fmt"
	"strings"

	gl "gitlab.com/gitlab-org/api/client-go"
)

// PaginationMetadata contains pagination information for list responses
type PaginationMetadata struct {
	TotalItems   int `json:"total_items"`
	TotalPages   int `json:"total_pages"`
	CurrentPage  int `json:"current_page"`
	ItemsPerPage int `json:"items_per_page"`
}

// PaginatedResponse wraps list results with pagination metadata
type PaginatedResponse struct {
	Items      interface{}         `json:"items"`
	Pagination *PaginationMetadata `json:"pagination,omitempty"`
}

// ExtractPagination extracts pagination metadata from GitLab API response
func ExtractPagination(resp *gl.Response) *PaginationMetadata {
	if resp == nil {
		return nil
	}

	// GitLab Response struct has direct pagination fields
	totalItems := resp.TotalItems
	totalPages := resp.TotalPages
	currentPage := resp.CurrentPage
	itemsPerPage := resp.ItemsPerPage

	// Only return pagination if we have meaningful data
	if totalItems == 0 && totalPages == 0 && currentPage == 0 && itemsPerPage == 0 {
		return nil
	}

	return &PaginationMetadata{
		TotalItems:   totalItems,
		TotalPages:   totalPages,
		CurrentPage:  currentPage,
		ItemsPerPage: itemsPerPage,
	}
}

// FieldFilter removes unnecessary fields from API responses
type FieldFilter struct {
	excludedFields []string
}

// NewFieldFilter creates a field filter for the specified entity type
func NewFieldFilter(entityType string) *FieldFilter {
	var fields []string

	switch strings.ToLower(entityType) {
	case "issue":
		fields = []string{
			"_links",                // API URLs
			"web_url",               // Web URL
			"author.web_url",        // Author profile URL
			"author.avatar_url",     // Author avatar URL
			"assignees.web_url",     // Assignee profile URLs
			"assignees.avatar_url",  // Assignee avatar URLs
			"service_desk_reply_to", // Customer service field
			"moved_to_id",           // For moved issues
			"epic_issue_id",         // Technical ID
			"issue_link_id",         // Technical ID
			"user_notes_count",      // Note count
			"label_details",         // Detailed label info
		}
	case "merge_request":
		fields = []string{
			"_links",
			"web_url",
			"author.web_url",
			"author.avatar_url",
			"assignees.web_url",
			"assignees.avatar_url",
			"reviewers.web_url",
			"reviewers.avatar_url",
			"label_details",
			"diff_refs",     // Technical diff references
			"references",    // Duplicates iid
		}
	case "project":
		fields = []string{
			"web_url",
			"avatar_url",
			"forked_from_project", // Fork source info
			"star_count",          // Metrics
			"forks_count",         // Metrics
		}
	case "user":
		fields = []string{
			"web_url",
			"avatar_url",
			"website_url", // Personal website
			"linkedin",    // Social links
			"twitter",     // Social links
			"skype",       // Social links
			"location",    // Location info
		}
	}

	return &FieldFilter{
		excludedFields: fields,
	}
}

// FilterResponse removes excluded fields from the response data
func (f *FieldFilter) FilterResponse(data interface{}) (interface{}, error) {
	// Convert to JSON and back to map for manipulation
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	var slice []map[string]interface{}
	if err := json.Unmarshal(jsonData, &slice); err != nil {
		// If not a slice, return as-is
		return data, nil
	}

	// Remove excluded fields from each item
	for _, item := range slice {
		f.removeFields(item)
	}

	return slice, nil
}

// removeFields recursively removes excluded fields from a map
func (f *FieldFilter) removeFields(item map[string]interface{}) {
	for _, fieldPath := range f.excludedFields {
		parts := strings.Split(fieldPath, ".")

		switch len(parts) {
		case 1:
			// Top-level field (e.g., "_links")
			delete(item, parts[0])
		case 2:
			// Handle both nested objects and slices
			fieldName := parts[0]
			nestedFieldName := parts[1]

			// Try as nested object first (e.g., "author.avatar_url")
			if nested, ok := item[fieldName].(map[string]interface{}); ok {
				delete(nested, nestedFieldName)
			}

			// Try as slice (e.g., "assignees.avatar_url")
			if sliceField, ok := item[fieldName].([]interface{}); ok {
				for _, sliceItem := range sliceField {
					if nestedMap, ok := sliceItem.(map[string]interface{}); ok {
						delete(nestedMap, nestedFieldName)
					}
				}
			}
		}
	}
}

// ResponseOptimizer combines text truncation and field filtering
type ResponseOptimizer struct {
	truncator  *TextTruncator
	filter     *FieldFilter
	truncFields []string
}

// NewResponseOptimizer creates an optimizer for the specified entity type
func NewResponseOptimizer(entityType string) *ResponseOptimizer {
	return &ResponseOptimizer{
		truncator:  NewTextTruncator(MaxFieldLength),
		filter:     NewFieldFilter(entityType),
		truncFields: getTruncationFields(entityType),
	}
}

// getTruncationFields returns the appropriate truncation field set for an entity type
func getTruncationFields(entityType string) []string {
	switch strings.ToLower(entityType) {
	case "issue":
		return IssueFields
	case "merge_request":
		return MergeRequestFields
	case "project":
		return ProjectFields
	case "user":
		return UserFields
	default:
		return []string{}
	}
}

// OptimizeListResponse applies both truncation and field filtering to list responses
func (o *ResponseOptimizer) OptimizeListResponse(data interface{}, resp *gl.Response) (*PaginatedResponse, error) {
	// Step 1: Truncate long text fields
	truncated, err := o.truncator.TruncateListResponse(data, o.truncFields)
	if err != nil {
		return nil, fmt.Errorf("failed to truncate response: %w", err)
	}

	// Step 2: Filter out unnecessary fields
	filtered, err := o.filter.FilterResponse(truncated)
	if err != nil {
		return nil, fmt.Errorf("failed to filter response: %w", err)
	}

	// Step 3: Extract pagination metadata
	pagination := ExtractPagination(resp)

	// Step 4: Wrap in paginated response
	return &PaginatedResponse{
		Items:      filtered,
		Pagination: pagination,
	}, nil
}
