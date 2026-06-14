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

func TestWithServerSelectionAddsSchemaParameter(t *testing.T) {
	tool := mcp.NewTool("sample")
	wrapped, _ := WithServerSelection(tool, nil)

	props := wrapped.InputSchema.Properties
	require.Contains(t, props, "server")
}
