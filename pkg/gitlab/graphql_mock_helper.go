package gitlab

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	gl "gitlab.com/gitlab-org/api/client-go"
)

// GraphQLMockMatcher defines a mock GraphQL request/response matcher
type GraphQLMockMatcher struct {
	Query    string
	Response map[string]any
}

// DataResponse creates a successful GraphQL response
func DataResponse(data map[string]any) map[string]any {
	return map[string]any{
		"data": data,
	}
}

// ErrorResponse creates a GraphQL error response
func ErrorResponse(message string) map[string]any {
	return map[string]any{
		"errors": []map[string]any{
			{"message": message},
		},
	}
}

// NewMockGraphQLHTTPClient creates a mock HTTP client for GraphQL testing
// This follows the pattern from github-mcp-server's githubv4mock
func NewMockGraphQLHTTPClient(matchers ...GraphQLMockMatcher) *http.Client {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/graphql", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Extract query from request
		var gqlReq struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(body, &gqlReq); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		// Try to find a matching matcher
		var matchedMatcher *GraphQLMockMatcher
		for _, matcher := range matchers {
			// Simple matching: check if the query contains the matcher's query string
			// This is more flexible than exact matching
			if strings.Contains(gqlReq.Query, matcher.Query) {
				matchedMatcher = &matcher
				break
			}
		}

		if matchedMatcher == nil {
			// No matcher found - return 404
			http.Error(w, fmt.Sprintf("no matcher found for query: %s", gqlReq.Query), http.StatusNotFound)
			return
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		jsonBytes, _ := json.Marshal(matchedMatcher.Response)
		w.Write(jsonBytes)
	})

	// Create HTTP client with mock transport
	return &http.Client{
		Transport: &roundTripper{
			mux: mux,
		},
	}
}

// roundTripper implements http.RoundTripper for mocking
type roundTripper struct {
	mux *http.ServeMux
}

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
		// Capture the response
		w := &responseRecorder{
			header: http.Header{},
		}

		rt.mux.ServeHTTP(w, req)

	return &http.Response{
		StatusCode: w.status,
		Header:     w.header,
		Body:       io.NopCloser(strings.NewReader(w.body)),
	}, nil
}

// responseRecorder captures HTTP response
type responseRecorder struct {
	status int
	header http.Header
	body   string
}

func (r *responseRecorder) Header() http.Header {
	return r.header
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body += string(b)
	return len(b), nil
}

// MockGraphQLClient creates a GitLab client with mocked HTTP for GraphQL testing
func MockGraphQLClient(httpClient *http.Client, token string) (*gl.Client, error) {
	// Use WithHTTPClient option to inject our mock HTTP client
	client, err := gl.NewClient(
		token,
		gl.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create mock GitLab client: %w", err)
	}

	return client, nil
}
