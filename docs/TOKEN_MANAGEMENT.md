# Token Management

This document describes how GitLab MCP Server manages access tokens and tracks their expiration.

## Overview

GitLab Personal Access Tokens (PATs) have a maximum lifetime of 1 year. The GitLab MCP Server provides automatic token validation and expiration tracking to prevent unexpected authentication failures.

## Features

### 1. Automatic Token Validation

The server validates your token on startup by calling the GitLab API:

```bash
INFO  Token validated successfully for user john_doe (ID: 12345) on server 'default'
```

If validation fails, a warning is logged but the server continues to run:

```
WARN  Token validation warning: token is invalid or expired (401)
```

### 2. Expiration Tracking

The server tracks token expiration and warns you 30 days before expiry:

```
WARN  Token will expire in 28 days. Please create a new token and update it.
```

### 3. Runtime Token Management

The server provides MCP tools for managing tokens during runtime:

#### `addToken`
Add a new GitLab token configuration to the runtime store.

**Parameters:**
- `name` (required): Token/server name (e.g., 'work', 'personal')
- `token` (required): GitLab Personal Access Token
- `gitlabHost` (optional): GitLab host URL (default: https://gitlab.com)

**Example:**
```json
{
  "name": "work",
  "token": "glpat-xxxxxxxxxxxxxxxxxxxx",
  "gitlabHost": "https://gitlab.example.com"
}
```

**Note:** Tokens added via `addToken` are only stored in runtime memory and will be lost when the server restarts. For permanent configuration, use the installer.

#### `listTokens`
List all configured tokens with their validation status and metadata.

**Example response:**
```json
{
  "tokens": [
    {
      "name": "default",
      "gitlabHost": "https://gitlab.com",
      "userId": 12345,
      "username": "john_doe",
      "createdAt": "2025-01-15T10:30:00Z",
      "lastValidated": "2025-12-27T15:45:00Z",
      "isExpired": false,
      "daysUntilExpiry": 180
    }
  ],
  "count": 1,
  "message": "Found 1 configured token(s)"
}
```

#### `updateToken`
Update an existing token or revalidate it.

**Parameters:**
- `name` (required): Token name to update
- `token` (optional): New token value (if not provided, only revalidates existing token)

**Example:**
```json
{
  "name": "work",
  "token": "glpat-newtoken123"
}
```

#### `validateToken`
Manually validate a token (or all tokens).

**Parameters:**
- `name` (optional): Token name to validate. If not provided, validates all tokens.

**Example response:**
```json
{
  "success": true,
  "tokenName": "work",
  "userId": 12345,
  "username": "john_doe",
  "message": "Token 'work' is valid"
}
```

#### `removeToken`
Remove a token from the runtime store.

**Parameters:**
- `name` (required): Token name to remove

**Note:** This only removes the token from runtime memory. It does not update your MCP configuration files.

#### `getNotifications`
Get recent notifications about token issues, validation results, and other important messages.

**Example response:**
```json
{
  "notifications": [
    {
      "level": "WARNING",
      "title": "Token Expiring Soon",
      "message": "Token 'work' will expire in 15 days. Please create a new token and update it.",
      "tokenName": "work",
      "timestamp": "2025-12-27T10:00:00Z"
    }
  ],
  "count": 1
}
```

#### `clearNotifications`
Clear all stored notifications.

### 4. 401 Error Handling

When a GitLab API call returns a 401 Unauthorized error, the server provides a user-friendly message:

```
Authentication failed (401). Your GitLab token may be expired. Please update it using the updateToken tool.
```

This happens automatically for all tools that interact with the GitLab API (projects, issues, merge requests, etc.).

## Configuration

### Environment Variables

Tokens are configured via environment variables in your MCP configuration:

**Single server:**
```json
{
  "mcpServers": {
    "gitlab": {
      "command": "/path/to/gitlab-mcp-server",
      "args": ["stdio"],
      "env": {
        "GITLAB_TOKEN": "glpat-yourtokenhere",
        "GITLAB_HOST": "https://gitlab.com"
      }
    }
  }
}
```

**Multiple servers:**
```json
{
  "mcpServers": {
    "work": {
      "command": "/path/to/gitlab-mcp-server",
      "env": {
        "GITLAB_TOKEN": "glpat-worktoken",
        "GITLAB_HOST": "https://gitlab.example.com"
      }
    },
    "personal": {
      "command": "/path/to/gitlab-mcp-server",
      "env": {
        "GITLAB_TOKEN": "glpat-personaltoken",
        "GITLAB_HOST": "https://gitlab.com"
      }
    }
  }
}
```

### Using the Installer

The installer (`node scripts/install.js`) helps you configure MCP servers:

```bash
node scripts/install.js
```

For multiple GitLab servers, select "Configure multiple GitLab servers? (y/n): y" when prompted.

## Security Best Practices

1. **Token Scope**: Only grant the permissions you need (read-only for most use cases)
2. **Token Expiration**: Set tokens to expire in 1 year (GitLab maximum)
3. **Token Storage**: Tokens are stored in your MCP configuration files (`~/.claude.json`, VS Code settings, etc.)
4. **Runtime Tokens**: Tokens added via `addToken` are only in memory and lost on restart
5. **Token Rotation**: When a token expires, use `updateToken` to update it, or re-run the installer

## Troubleshooting

### Token validation fails on startup

**Error:** `Token validation warning: token is invalid or expired (401)`

**Solution:**
1. Generate a new token in GitLab (User Settings → Access Tokens)
2. Update the token using the `updateToken` tool, or
3. Re-run the installer with the new token

### 401 errors during tool usage

**Error:** `Authentication failed (401). Your GitLab token may be expired.`

**Solution:**
1. Check token status: `listTokens` tool
2. Update the token: `updateToken` tool
3. Verify token has required permissions (api, read_repository, etc.)

### Multiple servers with same token

If you use the same token for multiple GitLab servers, you can configure them with different names but the same token value. The server will track each server independently.

## Token Metadata

The server tracks the following metadata for each token:

| Field | Description |
|-------|-------------|
| `name` | Token/server name |
| `gitlabHost` | GitLab instance URL |
| `userId` | GitLab user ID |
| `username` | GitLab username |
| `createdAt` | When token was added to the store |
| `lastValidated` | Last successful validation |
| `expiresAt` | Token expiration date (if available) |
| `isExpired` | Whether token is expired |

Note: GitLab API doesn't directly provide token expiration date, so we estimate based on creation time and GitLab's 1-year maximum.
