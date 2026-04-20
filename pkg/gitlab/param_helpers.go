package gitlab

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	gl "gitlab.com/gitlab-org/api/client-go"
)

// ParseLabelString parses a comma-separated label string and returns LabelOptions
// Input: "feature,bug,high-priority"
// Output: LabelOptions ready for GitLab API
func ParseLabelString(labels string) (*gl.LabelOptions, error) {
	if labels == "" {
		return nil, nil
	}

	labelSlice := strings.Split(labels, ",")
	// Trim whitespace from each label
	for i := range labelSlice {
		labelSlice[i] = strings.TrimSpace(labelSlice[i])
	}

	// Filter out empty strings
	filtered := make([]string, 0, len(labelSlice))
	for _, label := range labelSlice {
		if label != "" {
			filtered = append(filtered, label)
		}
	}

	if len(filtered) == 0 {
		return nil, nil
	}

	labelOpts := gl.LabelOptions(filtered)
	return &labelOpts, nil
}

// ParseAssigneeIDsString parses a comma-separated string of user IDs
// Input: "1,2,3"
// Output: []int{1, 2, 3} wrapped as pointer
func ParseAssigneeIDsString(assigneeIdsStr string) (*[]int64, error) {
	if assigneeIdsStr == "" {
		return nil, nil
	}

	assigneeIdsList := strings.Split(assigneeIdsStr, ",")
	assigneeIds := make([]int64, 0, len(assigneeIdsList))

	for _, idStr := range assigneeIdsList {
		idStr = strings.TrimSpace(idStr)
		if idStr == "" {
			continue
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid assignee ID %q: %w", idStr, err)
		}
		assigneeIds = append(assigneeIds, id)
	}

	if len(assigneeIds) == 0 {
		return nil, nil
	}

	return &assigneeIds, nil
}

// ValidateAndConvertMilestoneID validates a float64 milestone ID and converts to int
// Returns error if the value is not a valid integer
func ValidateAndConvertMilestoneID(milestoneIDFloat float64) (int, error) {
	if milestoneIDFloat == 0 {
		return 0, nil
	}

	milestoneID := int(milestoneIDFloat)
	if float64(milestoneID) != milestoneIDFloat {
		return 0, fmt.Errorf("milestoneId %v is not a valid integer", milestoneIDFloat)
	}

	return milestoneID, nil
}

// ParseDueDate parses a due date string in YYYY-MM-DD format
// Returns gl.ISOTime ready for GitLab API
func ParseDueDate(dueDateStr string) (*gl.ISOTime, error) {
	if dueDateStr == "" {
		return nil, nil
	}

	dueDate, err := time.Parse("2006-01-02", dueDateStr)
	if err != nil {
		return nil, fmt.Errorf("dueDate must be in YYYY-MM-DD format, got %q: %w", dueDateStr, err)
	}

	isoTime := gl.ISOTime(dueDate)
	return &isoTime, nil
}

// MustParseFloatToInt converts float64 to int with precision check
// Panics if conversion loses precision (useful for constants)
func MustParseFloatToInt(f float64, paramName string) int {
	i := int(f)
	if float64(i) != f {
		panic(fmt.Sprintf("parameter '%s' %v is not a valid integer", paramName, f))
	}
	return i
}

// RequiredFloatToIntParam converts a required float64 parameter to int
// Returns an error if the conversion loses precision
func RequiredFloatToIntParam(f float64, paramName string) (int, error) {
	i := int(f)
	if float64(i) != f {
		return 0, fmt.Errorf("parameter '%s' %v is not a valid integer", paramName, f)
	}
	return i, nil
}

// OptionalFloatToIntParam converts an optional float64 parameter to int
// Returns 0 and nil error if the value is 0
func OptionalFloatToIntParam(f float64, paramName string) (int, error) {
	if f == 0 {
		return 0, nil
	}
	i := int(f)
	if float64(i) != f {
		return 0, fmt.Errorf("parameter '%s' %v is not a valid integer", paramName, f)
	}
	return i, nil
}

// ApplyLabelsWithString sets labels on CreateIssueOptions or CreateMergeRequestOptions
// This is a helper to reduce duplication in tool handlers
func ApplyLabelsWithString(opts interface{}, labels string) error {
	if labels == "" {
		return nil
	}

	labelOpts, err := ParseLabelString(labels)
	if err != nil {
		return err
	}

	if labelOpts == nil {
		return nil
	}

	// Use type assertion to set the Labels field
	switch o := opts.(type) {
	case *gl.CreateIssueOptions:
		o.Labels = labelOpts
	case *gl.UpdateIssueOptions:
		o.Labels = labelOpts
	case *gl.CreateMergeRequestOptions:
		o.Labels = labelOpts
	case *gl.UpdateMergeRequestOptions:
		o.Labels = labelOpts
	default:
		return fmt.Errorf("unsupported options type for ApplyLabelsWithString")
	}

	return nil
}

// ApplyAssigneeIDsWithString sets assignee IDs on CreateIssueOptions or CreateMergeRequestOptions
func ApplyAssigneeIDsWithString(opts interface{}, assigneeIdsStr string) error {
	assigneeIds, err := ParseAssigneeIDsString(assigneeIdsStr)
	if err != nil {
		return err
	}

	if assigneeIds == nil {
		return nil
	}

	// Use type assertion to set the AssigneeIDs field
	switch o := opts.(type) {
	case *gl.CreateIssueOptions:
		o.AssigneeIDs = assigneeIds
	case *gl.UpdateIssueOptions:
		o.AssigneeIDs = assigneeIds
	case *gl.CreateMergeRequestOptions:
		o.AssigneeIDs = assigneeIds
	case *gl.UpdateMergeRequestOptions:
		o.AssigneeIDs = assigneeIds
	default:
		return fmt.Errorf("unsupported options type for ApplyAssigneeIDsWithString")
	}

	return nil
}
