# GitHub Detection

This document explains how GitLab MCP Server detects and prevents usage with GitHub repositories.

## Overview

GitLab MCP Server is designed specifically for GitLab. It does not support GitHub repositories. When a GitHub repository is detected, the server returns a clear error message directing users to use the appropriate GitHub MCP server instead.

## Why GitHub Detection?

1. **API Compatibility**: GitLab and GitHub have different APIs
2. **Tool Specificity**: GitLab MCP tools are optimized for GitLab-specific features
3. **User Experience**: Clear errors prevent confusion when tools don't work as expected
4. **Alternative Solution**: Users should use [github-mcp-server](https://github.com/github/github-mcp-server) for GitHub repositories

## Detection Mechanism

### When Detection Happens

GitHub detection occurs in two places:

1. **Auto-Detection** (`detectProject` tool)
   - Analyzes `.git/config` remote URLs
   - Returns error if GitHub remote is found

2. **Manual Project Set** (`setCurrentProject` tool)
   - Validates provided `gitlabHost` parameter
   - Returns error if GitHub URL is detected

### GitHub URL Patterns

The following patterns are recognized as GitHub:

| Pattern | Example |
|---------|---------|
| `github.com` | `https://github.com/user/repo.git` |
| `git@github.com:` | `git@github.com:user/repo.git` |
| `https://github.com/` | `https://github.com/user/repo` |
| `http://github.com/` | `http://github.com/user/repo` |
| `ssh://git@github.com/` | `ssh://git@github.com/user/repo.git` |

All checks are **case-insensitive** (e.g., `GITHUB.COM`, `GitHub.Com`).

## Error Messages

### Auto-Detection Error

When running `detectProject` in a GitHub repository:

```
Error: GitHub repository detected: git@github.com:user/repo.git.
This MCP server only supports GitLab repositories. Please use github-mcp-server instead.
```

### Manual Set Error

When running `setCurrentProject` with a GitHub URL:

```
Error: Failed to detect project: GitHub repository detected: https://github.com/user/repo.
This MCP server only supports GitLab repositories. Please use github-mcp-server instead.
```

## Code Implementation

### Detection Function

Located in `pkg/gitlab/project_config.go`:

```go
func isGitHubURL(url string) bool {
    lowerURL := strings.ToLower(url)
    githubPatterns := []string{
        "github.com",
        "git@github.com:",
        "https://github.com/",
        "http://github.com/",
        "ssh://git@github.com/",
    }
    for _, pattern := range githubPatterns {
        if strings.Contains(lowerURL, pattern) {
            return true
        }
    }
    return false
}
```

### Integration Points

1. **`parseGitLabURL`**: Returns error if `isGitHubURL(url)` is true
2. **`parseGitRemotes`**: Propagates GitHub detection error
3. **`DetectProjectFromGit`**: Returns GitHub detection error to caller

## Usage Examples

### Detecting GitLab Repository (Success)

```bash
cd ~/projects/my-gitlab-project
# .git/config has: remote = "https://gitlab.com/user/repo.git"

> detectProject
```

Response:
```json
{
  "success": true,
  "projectId": "user%2Frepo",
  "gitlabHost": "https://gitlab.com",
  "message": "Auto-detected GitLab project: user/repo"
}
```

### Detecting GitHub Repository (Error)

```bash
cd ~/projects/my-github-project
# .git/config has: remote = "git@github.com:user/repo.git"

> detectProject
```

Error:
```
GitHub repository detected: git@github.com:user/repo.git.
This MCP server only supports GitLab repositories. Please use github-mcp-server instead.
```

## Using Both GitLab and GitHub

If you have both GitHub and GitLab repositories, you can use both MCP servers simultaneously:

1. **Install GitLab MCP Server:**
   ```bash
   # Follow installation instructions for this project
   node scripts/install.js
   ```

2. **Install GitHub MCP Server:**
   - GitHub MCP Server is a separate project maintained by GitHub
   - Follow the [official installation instructions](https://github.com/github/github-mcp-server)
   - Installation methods may differ from this project

3. **Configure each project appropriately:**
   ```bash
   # GitLab project
   cd ~/projects/gitlab-project
   echo '{"projectId":"123"}' > .gmcprc

   # GitHub project
   cd ~/projects/github-project
   # GitHub MCP Server will handle GitHub repositories
   ```

4. **Use the right tools for each project**
   - GitLab MCP Server tools for GitLab repositories
   - GitHub MCP Server tools for GitHub repositories

## Troubleshooting

### False Positive Detection

**Problem:** You're using a self-hosted GitLab instance with "github" in the URL (unlikely but possible).

**Solution:** Contact your GitLab administrator to use a different hostname, or set the project manually using `setCurrentProject` with the correct `gitlabHost` parameter.

### Can't Detect Project

**Problem:** `detectProject` fails but you're sure it's a GitLab repository.

**Possible causes:**
1. No `.git` directory (not a git repository)
2. No GitLab remotes configured
3. Using a custom GitLab hostname

**Solution:** Use `setCurrentProject` to manually configure:
```
> setCurrentProject {
  "projectId": "123",
  "gitlabHost": "https://gitlab.example.com"
}
```

### Mixed Repositories

**Problem:** Project has both GitHub and GitLab remotes.

**Solution:** The tool will detect the first GitHub remote and error. Configure manually using `setCurrentProject` to specify which remote to use.

## Future Enhancements

Potential improvements (not currently implemented):

1. **Whitelist mode**: Allow specific GitHub URLs via configuration
2. **Auto-migration**: Suggest using github-mcp-server automatically
3. **Hybrid mode**: Support both GitLab and GitHub in a single server (complex, not recommended)

## Related Documentation

- [Project Configuration](PROJECT_CONFIG.md) - How `.gmcprc` files work
- [Multi-Server Setup](MULTI_SERVER_SETUP.md) - Using multiple GitLab instances
- [Token Management](TOKEN_MANAGEMENT.md) - Managing access tokens

## External Resources

- [GitLab Documentation](https://docs.gitlab.com/)
- [GitHub MCP Server](https://github.com/github/github-mcp-server)
- [Model Context Protocol](https://modelcontextprotocol.io/)
