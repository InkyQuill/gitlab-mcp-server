package gitlab

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestedServerContext(t *testing.T) {
	ctx := WithRequestedServer(context.Background(), "work")
	server, ok := RequestedServerFromContext(ctx)
	require.True(t, ok)
	assert.Equal(t, "work", server)
}

func TestRequestedServerFromRequest(t *testing.T) {
	tests := []struct {
		name        string
		request     mcp.CallToolRequest
		wantServer  string
		wantPresent bool
		wantErr     string
	}{
		{
			name: "missing",
		},
		{
			name: "nil",
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]any{"server": nil},
				},
			},
		},
		{
			name: "normal string",
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]any{"server": "personal"},
				},
			},
			wantServer:  "personal",
			wantPresent: true,
		},
		{
			name: "trimmed string",
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]any{"server": " work "},
				},
			},
			wantServer:  "work",
			wantPresent: true,
		},
		{
			name: "whitespace string",
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]any{"server": " \t\n"},
				},
			},
		},
		{
			name: "non-string",
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]any{"server": 123},
				},
			},
			wantPresent: true,
			wantErr:     "parameter 'server' must be a string, got int",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, present, err := RequestedServerFromRequest(tt.request)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantServer, server)
			assert.Equal(t, tt.wantPresent, present)
		})
	}
}

func TestWithServerSelectionInjectsContext(t *testing.T) {
	tool := mcp.NewTool("sample")
	var gotServer string

	_, handler := WithServerSelection(tool, func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		gotServer, _ = RequestedServerFromContext(ctx)
		return mcp.NewToolResultText("ok"), nil
	})

	result, err := handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{"server": "personal"},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "personal", gotServer)
}

func TestWithServerSelectionRejectsMalformedServer(t *testing.T) {
	tool := mcp.NewTool("sample")
	called := false

	_, handler := WithServerSelection(tool, func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		called = true
		return mcp.NewToolResultText("ok"), nil
	})

	result, err := handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{"server": 123},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	require.Len(t, result.Content, 1)
	content, ok := result.Content[0].(mcp.TextContent)
	require.True(t, ok)
	assert.Equal(t, "parameter 'server' must be a string, got int", content.Text)
	assert.False(t, called)
}

func TestWithServerSelectionPreservesNilHandler(t *testing.T) {
	tool := mcp.NewTool("sample")
	_, handler := WithServerSelection(tool, nil)
	assert.Nil(t, handler)
}

func TestWithServerSelectionAddsSchemaParameter(t *testing.T) {
	tool := mcp.NewTool("sample")
	wrapped, _ := WithServerSelection(tool, nil)

	props := wrapped.InputSchema.Properties
	require.Contains(t, props, "server")
	assert.NotContains(t, wrapped.InputSchema.Required, "server")
}
