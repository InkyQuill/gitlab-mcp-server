# Quick Start Guide

Get up and running with the GitLab MCP Server in minutes.

## Prerequisites

Before starting, ensure you have:

1. A GitLab account with access to at least one project
2. A GitLab access token (see [Token Management](TOKEN_MANAGEMENT.md) for details)
3. One of the supported development environments:
   - VS Code
   - Claude Desktop
   - Claude Code
   - Cursor

## Step 1: Install the Server

Use the automated installer:

```bash
git clone <repository-url>
cd gitlab-mcp-server
make setup
make install-mcp
```

Or use the setup script:

```bash
./setup.sh  # Linux/macOS
setup.bat   # Windows
```

The installer will guide you through:
- Building the server binary
- Configuring your GitLab token
- Setting up your development environment

## Step 2: Verify Installation

After installation, verify the server is available in your development environment:

1. **VS Code**: Check the MCP servers list in the agent mode
2. **Claude Desktop**: Restart the application and check available tools
3. **Claude Code**: Run `claude mcp list` to see configured servers
4. **Cursor**: Check the MCP server status

## Step 3: Configure Your First Project

### Option A: Auto-Detection (Recommended)

If you're in a Git repository with a GitLab remote:

```bash
cd your-project
# The server will auto-detect from Git remotes
```

Use the `autoDetectAndSetProject` tool to automatically configure the project.

### Option B: Manual Configuration

Set the project manually:

```bash
# Create .gmcprc file
echo '{"projectId":"your-group/your-project"}' > .gmcprc
```

Or use the `setCurrentProject` tool with your project ID.

## Step 4: Try Your First Tool

### List Projects

Try listing your projects:

```
> listProjects
```

This will return a list of projects you have access to.

### Get Project Details

Get details for a specific project:

```
> getProject {"projectId": "your-group/your-project"}
```

### List Issues

List issues in your project:

```
> listIssues {"projectId": "your-group/your-project"}
```

## Common Workflows

### Creating an Issue

```
> createIssue {
  "projectId": "your-group/your-project",
  "title": "New feature request",
  "description": "Description of the feature",
  "labels": "enhancement,help-wanted"
}
```

### Creating a Merge Request

```
> createMergeRequest {
  "projectId": "your-group/your-project",
  "sourceBranch": "feature-branch",
  "targetBranch": "main",
  "title": "Add new feature",
  "description": "Implements the requested feature"
}
```

### Searching for Projects

```
> searchProjects {
  "search": "my-project"
}
```

## Next Steps

Now that you're up and running:

1. **Explore Available Tools**: See [Tools Reference](TOOLS.md) for all available tools
2. **Configure Multiple Servers**: See [Multi-Server Setup](MULTI_SERVER_SETUP.md) if you use multiple GitLab instances
3. **Customize Tool Descriptions**: See [Internationalization](I18N.md) for customizing tool descriptions
4. **Enable Dynamic Toolsets**: See [Dynamic Tool Discovery](DYNAMIC_TOOLS.md) for on-demand tool loading

## Troubleshooting

### Server Not Appearing

- Verify installation completed successfully
- Check configuration file location and syntax
- Restart your development environment

### Authentication Errors

- Verify your token is valid
- Check token has required scopes
- See [Token Management](TOKEN_MANAGEMENT.md) for details

### Project Not Found

- Verify project ID is correct (format: `group/project` or numeric ID)
- Check you have access to the project
- Ensure `GITLAB_HOST` matches your GitLab instance

## Getting Help

- Check the [Documentation](../README.md) for detailed guides
- Review [Tools Reference](TOOLS.md) for tool-specific help
- Open an issue on GitHub for bugs or questions

