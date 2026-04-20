package gitlab

import (
	"fmt"
	"testing"

	gl "gitlab.com/gitlab-org/api/client-go"
)

func BenchmarkTruncateText_Small(b *testing.B) {
	text := "short text"
	truncator := NewTextTruncator(300)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = truncator.truncateString(text)
	}
}

func BenchmarkTruncateText_Large(b *testing.B) {
	text := string(make([]byte, 10000))
	truncator := NewTextTruncator(300)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = truncator.truncateString(text)
	}
}

func BenchmarkTruncateText_NoTruncationNeeded(b *testing.B) {
	text := "short text"
	truncator := NewTextTruncator(300)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = truncator.truncateString(text)
	}
}

func BenchmarkTruncateListResponse_Issues(b *testing.B) {
	issues := []map[string]interface{}{
		{"id": 1, "title": "Issue 1", "description": string(make([]byte, 1000))},
		{"id": 2, "title": "Issue 2", "description": string(make([]byte, 500))},
		{"id": 3, "title": "Issue 3", "description": "short desc"},
	}
	truncator := NewTextTruncator(300)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = truncator.TruncateListResponse(issues, IssueFields)
	}
}

func BenchmarkFilterFields_Small(b *testing.B) {
	data := map[string]interface{}{"id": 1, "name": "test", "description": "desc"}
	filter := NewFieldFilter("issue")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = filter.FilterResponse([]interface{}{data})
	}
}

func BenchmarkFilterFields_Large(b *testing.B) {
	data := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		data[i] = map[string]interface{}{
			"id":          i,
			"title":       fmt.Sprintf("Issue %d", i),
			"description": fmt.Sprintf("Description %d", i),
			"web_url":     fmt.Sprintf("https://example.com/%d", i),
			"_links":      map[string]string{"self": fmt.Sprintf("https://api.example.com/%d", i)},
		}
	}
	filter := NewFieldFilter("issue")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = filter.FilterResponse(data)
	}
}

func BenchmarkOptimizeListResponse_FullPipeline(b *testing.B) {
	issues := []map[string]interface{}{
		{"id": 1, "title": "Issue 1", "description": string(make([]byte, 1000)), "web_url": "https://example.com/1"},
		{"id": 2, "title": "Issue 2", "description": string(make([]byte, 500)), "web_url": "https://example.com/2"},
	}
	optimizer := NewResponseOptimizer("issue")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = optimizer.OptimizeListResponse(issues, nil)
	}
}

func BenchmarkExtractPagination(b *testing.B) {
	// Create a mock response using the actual gl.Response type
	resp := &gl.Response{
		TotalItems:   100,
		TotalPages:   5,
		CurrentPage:  1,
		ItemsPerPage: 20,
	}
	// This benchmark cannot directly test ExtractPagination since it requires
	// a properly initialized http.Response. Instead, we test the metadata creation.
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = &PaginationMetadata{
			TotalItems:   resp.TotalItems,
			TotalPages:   resp.TotalPages,
			CurrentPage:  resp.CurrentPage,
			ItemsPerPage: resp.ItemsPerPage,
		}
	}
}
