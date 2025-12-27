# Project Configuration Feature

The GitLab MCP server now supports automatic project detection and local project configuration, making it easier to work with multiple GitLab instances and projects without repeatedly specifying the project ID.

## Overview

The server can now:
- **Auto-detect** your GitLab project from Git remote configuration
- **Save** project settings locally in a `.gmcprc` file
- **Automatically use** the saved project ID in all tools

## How It Works

### 1. Local Configuration File (`.gmcprc`)

When you set a project, the server creates a `.gmcprc` file in your current directory:

```json
{
  "projectId": "owner/repo",
  "gitlabHost": "https://gitlab.com",
  "lastUpdated": "2025-12-27T17:30:00Z"
}
```

This file:
- Is searched for in the current directory and all parent directories (like `.git`)
- Can be committed to your repository or added to `.gitignore`
- Contains only the project ID and optionally the GitLab host URL

### 2. Git Remote Auto-Detection

The server can automatically detect your GitLab project from:
- **HTTPS URLs**: `https://gitlab.com/owner/repo.git`
- **SSH URLs**: `git@gitlab.com:owner/repo.git`
- **Self-hosted instances**: `https://git.example.com/owner/repo.git`

It searches for:
1. `.git` directory in current or parent folders
2. GitLab remotes in `.git/config`
3. Extracts project ID and host from the remote URL

## Available Tools

### `setCurrentProject`

Manually sets the current project for this directory.

**Parameters:**
- `projectId` (required): The GitLab project ID (e.g., `"owner/repo"` or numeric ID)
- `gitlabHost` (optional): GitLab host URL (e.g., `"https://gitlab.example.com"`)

**Example:**
```json
{
  "name": "setCurrentProject",
  "arguments": {
    "projectId": "mygroup/myproject",
    "gitlabHost": "https://gitlab.example.com"
  }
}
```

**Result:**
```json
{
  "success": true,
  "configPath": "/path/to/.gmcprc",
  "projectId": "mygroup/myproject",
  "gitlabHost": "https://gitlab.example.com"
}
```

### `getCurrentProject`

Retrieves the current project configuration from `.gmcprc`.

**Parameters:** None

**Example:**
```json
{
  "name": "getCurrentProject",
  "arguments": {}
}
```

**Result:**
```json
{
  "found": true,
  "configPath": "/path/to/.gmcprc",
  "projectId": "mygroup/myproject",
  "gitlabHost": "https://gitlab.example.com"
}
```

### `detectProject`

Auto-detects the GitLab project from Git remote and verifies it exists.

**Parameters:** None

**Example:**
```json
{
  "name": "detectProject",
  "arguments": {}
}
```

**Result:**
```json
{
  "success": true,
  "projectId": "mygroup/myproject",
  "gitlabHost": "https://gitlab.com",
  "projectName": "My Project",
  "projectPath": "mygroup/myproject",
  "message": "Project detected successfully. Use 'setCurrentProject' with projectId='mygroup/myproject' to save it."
}
```

### `autoDetectAndSetProject`

Convenience command that combines detection and setting in one step.

**Parameters:** None

**Example:**
```json
{
  "name": "autoDetectAndSetProject",
  "arguments": {}
}
```

**Result:**
```json
{
  "success": true,
  "configPath": "/path/to/.gmcprc",
  "projectId": "mygroup/myproject",
  "gitlabHost": "https://gitlab.com",
  "projectName": "My Project",
  "projectPath": "mygroup/myproject",
  "message": "Project detected and configured successfully!"
}
```

## Use Cases

### 1. Quick Setup for a New Project

```bash
# In your project directory
autoDetectAndSetProject
```

Now all tools will automatically use this project!

### 2. Working with Multiple GitLab Instances

```bash
# Company GitLab
cd ~/work/company-project
setCurrentProject projectId="company/team-project" gitlabHost="https://gitlab.company.com"

# Public GitLab
cd ~/oss/open-source-project
autoDetectAndSetProject
```

### 3. Forked Projects

For forks or projects without Git remotes:

```bash
setCurrentProject projectId="original-owner/repo"
```

## Fallback Behavior

When using tools that require a `projectId` parameter:

1. **Explicit parameter** takes precedence
2. **`.gmcprc` file** is checked second
3. **Git remote detection** is attempted last
4. **Error** if none of the above work

## Migration from Manual Project ID

**Before:**
```json
{
  "name": "getProject",
  "arguments": {
    "projectId": "mygroup/myproject"
  }
}
```

**After** (with `.gmcprc`):
```json
{
  "name": "getProject",
  "arguments": {}
}
```

## Security Considerations

- `.gmcprc` stores project IDs (not tokens)
- Tokens are stored in your MCP configuration file only
- `.gmcprc` is safe to commit to version control
- `.gmcprc` is already in `.gitignore` by default

## Troubleshooting

### "No .gmcprc file found"

Run `autoDetectAndSetProject` or `setCurrentProject` first.

### "Failed to detect project"

Ensure:
- You're in a Git repository
- The repository has a GitLab remote
- The remote URL is in a supported format

### "Project not found or access denied"

Check:
- Your GitLab token has access to the project
- The project ID is correct
- The `gitlabHost` matches your GitLab instance

## Examples

### Example 1: First-Time Setup

```bash
# Clone a repository
git clone https://gitlab.com/mygroup/myproject.git
cd myproject

# Auto-detect and save project
autoDetectAndSetProject

# Now use any tool without specifying projectId
listIssues
```

### Example 2: Self-Hosted GitLab

```bash
cd ~/work/internal-project
setCurrentProject projectId="company/internal-project" gitlabHost="https://gitlab.company.com"
```

### Example 3: Multiple Projects

```bash
# Project 1
cd ~/project1
autoDetectAndSetProject

# Project 2 (different GitLab instance)
cd ~/project2
setCurrentProject projectId="team/project2" gitlabHost="https://gitlab.company.com"

# Each directory now remembers its own project
```

## Future Enhancements

- [ ] Make `projectId` optional in all existing tools
- [ ] Support for multiple project configurations in one directory
- [ ] Project aliases for frequently used projects
- [ ] Integration with GitLab forking workflows
