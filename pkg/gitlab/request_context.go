package gitlab

import (
	"context"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type requestedServerContextKey struct{}

func WithRequestedServer(ctx context.Context, name string) context.Context {
	name = strings.TrimSpace(name)
	if name == "" {
		return ctx
	}
	return context.WithValue(ctx, requestedServerContextKey{}, name)
}

func RequestedServerFromContext(ctx context.Context) (string, bool) {
	name, ok := ctx.Value(requestedServerContextKey{}).(string)
	if !ok || name == "" {
		return "", false
	}
	return name, true
}

func RequestedServerFromRequest(request mcp.CallToolRequest) string {
	raw, ok := request.GetArguments()["server"]
	if !ok || raw == nil {
		return ""
	}
	name, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(name)
}

func WithServerSelection(tool mcp.Tool, handler server.ToolHandlerFunc) (mcp.Tool, server.ToolHandlerFunc) {
	mcp.WithString("server",
		mcp.Description("Configured GitLab server name. Overrides .gmcprc/default routing for this call."),
	)(&tool)

	if handler == nil {
		return tool, nil
	}

	return tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if serverName := RequestedServerFromRequest(request); serverName != "" {
			ctx = WithRequestedServer(ctx, serverName)
		}
		return handler(ctx, request)
	}
}
