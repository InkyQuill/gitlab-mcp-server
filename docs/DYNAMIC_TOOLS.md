# Dynamic Tool Discovery

Dynamic toolset discovery allows the MCP host (like VS Code or Claude) to list available toolsets and enable them selectively in response to user needs. This prevents overwhelming the language model with too many tools initially and improves performance.

## How It Works

When dynamic tool discovery is enabled:
- The server starts with **only 2 tools** available: `list_available_toolsets` and `enable_toolset`
- You can query which toolsets are available and their descriptions
- Toolsets are loaded on-demand when you enable them
- Once enabled, all tools from that toolset become available

## Enabling Dynamic Tool Discovery

Enable it via:

**Command Line:**
```bash
./gitlab-mcp-server stdio --dynamic-toolsets
```

**Environment Variable:**
```bash
export GITLAB_DYNAMIC_TOOLSETS=true
./gitlab-mcp-server stdio
```

**Docker:**
```bash
docker run -i --rm \
  -e GITLAB_TOKEN=<your-token> \
  -e GITLAB_DYNAMIC_TOOLSETS=true \
  gitlab-mcp-server:latest
```

**MCP Configuration:**
```json
{
  "mcpServers": {
    "gitlab-go-mcp": {
      "command": "/path/to/gitlab-mcp-server",
      "args": ["stdio"],
      "env": {
        "GITLAB_TOKEN": "<YOUR_TOKEN>",
        "GITLAB_DYNAMIC_TOOLSETS": "true"
      }
    }
  }
}
```

## Available Discovery Tools

### list_available_toolsets

Lists all available GitLab MCP toolsets that can be enabled.

**Parameters:** None

**Example Output:**
```
Available Toolsets (8):
- token_management: Tools for managing GitLab tokens and authentication. [6 tools] (enabled)
- project_config: Tools for managing GitLab project configuration and auto-detection. [4 tools] (disabled)
- projects: Tools for interacting with GitLab projects, repositories, branches, commits, tags. [6 tools] (disabled)
- issues: Tools for CRUD operations on GitLab issues, comments, labels. [10 tools] (disabled)
- merge_requests: Tools for CRUD operations on GitLab merge requests, comments, approvals, diffs. [7 tools] (disabled)
- security: Tools for accessing GitLab security scan results (SAST, DAST, etc.). [6 tools] (disabled)
- users: Tools for looking up GitLab user information. [12 tools] (disabled)
- search: Tools for utilizing GitLab's scoped search capabilities. [14 tools] (disabled)
```

### enable_toolset

Enables a specific GitLab MCP toolset, making its tools available.

**Parameters:**
- `toolset` (required, string): Name of the toolset to enable (e.g., 'projects', 'issues', 'merge_requests')

**Example:**
```
> enable_toolset {"toolset": "projects"}
```

**Response:**
```
Toolset 'projects' enabled successfully. 6 tools are now available.
```

## Example Workflow

1. **Start server with dynamic toolsets:**
   ```bash
   ./gitlab-mcp-server stdio --dynamic-toolsets
   ```

2. **List available toolsets:**
   ```
   > list_available_toolsets
   ```

3. **Enable a toolset as needed:**
   ```
   > enable_toolset {"toolset": "projects"}
   ```

4. **Use the newly available tools:**
   ```
   > listProjects
   > getProject {"projectId": "mygroup/myproject"}
   ```

5. **Enable additional toolsets as needed:**
   ```
   > enable_toolset {"toolset": "issues"}
   > listIssues {"projectId": "mygroup/myproject"}
   ```

## Default Enabled Toolsets

Even in dynamic mode, the following toolsets are enabled by default:
- `token_management` - Essential for server operation
- `project_config` - Essential for project configuration

These cannot be disabled as they're required for basic server functionality.

## Benefits

### Reduced Initial Context

Instead of loading all 60+ tools at startup, only 2 discovery tools are loaded initially. This:
- Reduces token usage
- Improves response times
- Allows the LLM to focus on relevant tools

### On-Demand Loading

Toolsets are loaded only when needed:
- User asks about issues → enable issues toolset
- User wants to search → enable search toolset
- User needs CI/CD info → enable pipeline toolset (when implemented)

### Better Tool Selection

With fewer tools available, the LLM can:
- Make better decisions about which tools to use
- Avoid confusion from too many similar tools
- Focus on the task at hand

## Comparison with Static Mode

### Static Mode (Default)

```bash
./gitlab-mcp-server stdio
```

- All available toolsets are loaded at startup
- All tools are immediately available
- Higher initial context size
- Simpler for basic use cases

### Dynamic Mode

```bash
./gitlab-mcp-server stdio --dynamic-toolsets
```

- Only discovery tools loaded at startup
- Toolsets enabled on-demand
- Lower initial context size
- Better for complex workflows with many tools

## Use Cases

### When to Use Dynamic Mode

- Working with many toolsets and want to reduce context size
- Building AI applications that need selective tool loading
- Testing specific toolsets in isolation
- Optimizing token usage in production

### When to Use Static Mode

- Simple workflows with few tools
- Always need access to all tools
- Prefer simplicity over optimization
- Development and testing

## Troubleshooting

### Toolset Not Found

**Problem:** `enable_toolset` returns "toolset not found"

**Solution:** Check available toolsets with `list_available_toolsets`. Ensure you're using the correct toolset name (case-sensitive).

### Tools Not Appearing

**Problem:** Enabled toolset but tools don't appear

**Solution:**
- Verify toolset was enabled successfully
- Check server logs for errors
- Restart the MCP connection
- Verify toolset name is correct

### Performance Issues

**Problem:** Dynamic mode is slower than expected

**Solution:** Consider using static mode if you frequently use multiple toolsets. Dynamic mode adds overhead for toolset loading.

## Related Documentation

- [Tools Reference](TOOLS.md) - Complete list of available tools
- [Installation Guide](INSTALLATION.md) - Installation and setup
- [Configuration](README.md#configuration) - Server configuration options

