# Self-Hosted GitLab

This guide covers connecting the GitLab MCP Server to self-managed GitLab instances.

## Overview

The GitLab MCP Server supports both GitLab.com and self-managed GitLab instances. You can connect to any GitLab instance by specifying the host URL.

## Configuration

### Using Environment Variable

Set the `GITLAB_HOST` environment variable to your GitLab instance URL:

```bash
export GITLAB_HOST="https://gitlab.example.com"
./gitlab-mcp-server stdio
```

### Using Command Line Flag

Use the `--gitlab-host` flag:

```bash
./gitlab-mcp-server stdio --gitlab-host https://gitlab.example.com
```

### Using MCP Configuration

Configure in your MCP server configuration:

```json
{
  "mcpServers": {
    "gitlab-go-mcp": {
      "command": "/path/to/gitlab-mcp-server",
      "args": ["stdio"],
      "env": {
        "GITLAB_TOKEN": "<YOUR_TOKEN>",
        "GITLAB_HOST": "https://gitlab.example.com"
      }
    }
  }
}
```

### Using Docker

```bash
docker run -i --rm \
  -e GITLAB_TOKEN=<your-token> \
  -e GITLAB_HOST="https://gitlab.example.com" \
  gitlab-mcp-server:latest
```

## Access Token

Create an access token on your self-hosted GitLab instance:

1. Go to User Settings → Access Tokens (or Project/Group Settings for project/group tokens)
2. Create a token with appropriate scopes
3. Use this token in your MCP configuration

**Note:** The token must have access to the GitLab API endpoints you plan to use.

## SSL/TLS Certificates

If your self-hosted GitLab uses a self-signed certificate, you may need to configure certificate validation:

### Skip Certificate Verification (Not Recommended for Production)

```bash
export GITLAB_SKIP_TLS_VERIFY=true
./gitlab-mcp-server stdio
```

**Warning:** Only use this in development environments. Skipping TLS verification exposes you to man-in-the-middle attacks.

### Custom CA Certificate

For production, add your CA certificate to the system trust store or configure Go to use it:

```bash
export SSL_CERT_FILE=/path/to/ca-cert.pem
./gitlab-mcp-server stdio
```

## API Version

The server uses GitLab API v4 by default. Most self-hosted instances support v4, but if you're using an older version, you may need to check compatibility.

## Multi-Server Setup

You can configure multiple GitLab instances (both GitLab.com and self-hosted):

```json
{
  "mcpServers": {
    "work": {
      "command": "/path/to/gitlab-mcp-server",
      "args": ["stdio"],
      "env": {
        "GITLAB_TOKEN": "<work-token>",
        "GITLAB_HOST": "https://gitlab.company.com"
      }
    },
    "personal": {
      "command": "/path/to/gitlab-mcp-server",
      "args": ["stdio"],
      "env": {
        "GITLAB_TOKEN": "<personal-token>",
        "GITLAB_HOST": "https://gitlab.com"
      }
    }
  }
}
```

See [Multi-Server Setup](MULTI_SERVER_SETUP.md) for detailed instructions.

## Project Configuration

When working with self-hosted instances, specify the host in your `.gmcprc` file:

```json
{
  "projectId": "group/project",
  "gitlabHost": "https://gitlab.example.com"
}
```

Or use the `setCurrentProject` tool:

```
> setCurrentProject {
  "projectId": "group/project",
  "gitlabHost": "https://gitlab.example.com"
}
```

## Troubleshooting

### Connection Errors

**Problem:** Cannot connect to self-hosted GitLab

**Solution:**
- Verify the URL is correct (include `https://` or `http://`)
- Check network connectivity
- Verify the GitLab instance is accessible
- Check firewall rules

### Authentication Errors

**Problem:** 401 Unauthorized errors

**Solution:**
- Verify token is valid for your self-hosted instance
- Check token has required scopes
- Ensure `GITLAB_HOST` matches your instance URL exactly
- Verify token hasn't expired

### SSL Certificate Errors

**Problem:** Certificate verification errors

**Solution:**
- For development: Set `GITLAB_SKIP_TLS_VERIFY=true` (not recommended for production)
- For production: Add CA certificate to system trust store
- Verify certificate is valid and not expired

### API Version Issues

**Problem:** API calls fail with version errors

**Solution:**
- Verify your GitLab instance supports API v4
- Check GitLab version compatibility
- Review GitLab API documentation for your version

### Project Not Found

**Problem:** Projects not found on self-hosted instance

**Solution:**
- Verify project ID format (may differ from GitLab.com)
- Check you have access to the project
- Ensure `GITLAB_HOST` is set correctly
- Verify project exists on the instance

## Best Practices

1. **Use HTTPS**: Always use HTTPS for self-hosted instances in production
2. **Validate Certificates**: Don't skip certificate verification in production
3. **Token Security**: Store tokens securely, never commit them to version control
4. **Network Security**: Ensure network connections are secure
5. **Token Scopes**: Grant only necessary permissions to tokens

## Related Documentation

- [Installation Guide](INSTALLATION.md) - Server installation
- [Token Management](TOKEN_MANAGEMENT.md) - Managing access tokens
- [Multi-Server Setup](MULTI_SERVER_SETUP.md) - Configuring multiple instances
- [Project Configuration](PROJECT_CONFIG.md) - Project-specific settings

