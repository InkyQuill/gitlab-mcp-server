package gitlab

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type requestedServerContextKey struct{}

// WithRequestedServer returns a context carrying the requested GitLab server name.
func WithRequestedServer(ctx context.Context, name string) context.Context {
	name = strings.TrimSpace(name)
	if name == "" {
		return ctx
	}
	return context.WithValue(ctx, requestedServerContextKey{}, name)
}

// RequestedServerFromContext returns the GitLab server name carried by ctx.
func RequestedServerFromContext(ctx context.Context) (string, bool) {
	name, ok := ctx.Value(requestedServerContextKey{}).(string)
	if !ok || name == "" {
		return "", false
	}
	return name, true
}

// RequestedServerFromRequest returns the requested GitLab server name from a tool request.
func RequestedServerFromRequest(request mcp.CallToolRequest) (string, bool, error) {
	raw, ok := request.GetArguments()["server"]
	if !ok || raw == nil {
		return "", false, nil
	}
	name, ok := raw.(string)
	if !ok {
		return "", true, fmt.Errorf("parameter 'server' must be a string, got %T", raw)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false, nil
	}
	return name, true, nil
}

// WithServerSelection adds an optional server parameter and injects it into handler contexts.
func WithServerSelection(tool mcp.Tool, handler server.ToolHandlerFunc) (mcp.Tool, server.ToolHandlerFunc) {
	mcp.WithString("server",
		mcp.Description("Configured GitLab server name. Overrides .gmcprc/default routing for this call."),
	)(&tool)

	if handler == nil {
		return tool, nil
	}

	return tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		serverName, _, err := RequestedServerFromRequest(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if serverName != "" {
			ctx = WithRequestedServer(ctx, serverName)
		}
		return handler(ctx, request)
	}
}
