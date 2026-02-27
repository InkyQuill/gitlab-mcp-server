# Installation Guide

This guide covers installation and setup of the GitLab MCP Server.

## Prerequisites

Before installing, ensure you have:

1. **GitLab Access Token**: Create one of the following:
   - [Personal Access Token (PAT)](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html)
   - [Project Access Token](https://docs.gitlab.com/ee/user/project/settings/project_access_tokens.html)
   - [Group Access Token](https://docs.gitlab.com/ee/user/group/settings/group_access_tokens.html)

   Required scopes depend on the tools you'll use:
   - `api` - Broad access (recommended for full functionality)
   - `read_repository`, `write_repository` - Repository operations
   - `read_api`, `write_api` - API access

2. **Node.js**: Required for running the installer (usually already installed with Cursor, Claude Code, or MCP CLI)
3. **Docker (optional)**: For containerized deployment
4. **Go 1.23+ (optional)**: For building from source

## Installation Methods

### Method 1: Go Install (Recommended)

The quickest way to install the latest release:

```bash
go install github.com/InkyQuill/gitlab-mcp-server@latest
```

This installs the binary to `~/go/bin/gitlab-mcp-server` (or `%USERPROFILE%\go\bin` on Windows).

**After installation:**

```bash
# Verify installation
gitlab-mcp-server version

# Configure GitLab servers
gitlab-mcp-server config init

# Register with your IDE
gitlab-mcp-server install claude  # For Claude Desktop
# or
gitlab-mcp-server install vscode  # For VS Code
# or
gitlab-mcp-server install cursor  # For Cursor

# Set up your project
cd your-project
gitlab-mcp-server project init
```

**Ensure `~/go/bin` is in your PATH:**

```bash
# Add to ~/.bashrc, ~/.zshrc, or equivalent
export PATH="$HOME/go/bin:$PATH"
```

### Method 2: Automated Installer

The easiest way to install with automatic IDE configuration:

The easiest way to install is using the automated installer:

**Linux:**
```bash
git clone <repository-url>
cd gitlab-mcp-server
make setup
make install-mcp
```

**macOS/Linux (using setup script):**
```bash
git clone <repository-url>
cd gitlab-mcp-server
./setup.sh
```

**Windows:**
```cmd
git clone <repository-url>
cd gitlab-mcp-server
setup.bat
```

The installer will:
- Check prerequisites (Go version, dependencies)
- Build the server binary
- Prompt for configuration:
  - Deployment mode: local binary (default) or Docker
  - GitLab host URL (defaults to `https://gitlab.com`)
  - GitLab access token (entered securely, not displayed)
  - Read-only mode preference
  - Development environments to configure (VS Code, Claude Desktop, Claude Code, Cursor)
- Automatically configure your selected development environments
- Create backup copies (`.bak` files) of existing configuration files

The server will be registered as `gitlab-go-mcp` in your MCP configurations.

**Note:** The installer intelligently merges with existing configurations, preserving your other MCP server settings.

### Method 3: Building from Source

### Building from Source

#### Using Makefile

The project includes a Makefile with the following targets:

- `make setup` - Install prerequisites and dependencies
- `make build` - Build the binary to `bin/gitlab-mcp-server`
- `make clean` - Clean build artifacts
- `make test` - Run tests
- `make docker-build` - Build Docker image
- `make install-mcp` - Build installer and run MCP configuration
- `make help` - Show all available targets

**Example:**
```bash
make setup        # Install prerequisites
make build        # Build the server
make install-mcp  # Configure MCP servers
```

#### Standalone Build

To build the binary from source:

```bash
git clone <repository-url>
cd gitlab-mcp-server
go build -o bin/gitlab-mcp-server ./cmd/gitlab-mcp-server
```

The binary will be created as `bin/gitlab-mcp-server`.

### Docker Build

To build the Docker image:

```bash
docker build -t gitlab-mcp-server:latest .
```

Or using Makefile:
```bash
make docker-build
```

## Configuration

After installation, configure your development environment. See [Editor Setup](EDITOR_SETUP.md) for detailed instructions for:

- VS Code / VS Code Agent Mode
- Claude Desktop
- Claude Code
- Cursor

## Verification

To verify the installation:

1. Check that the binary exists:
   ```bash
   # If installed via go install
   gitlab-mcp-server version

   # If built from source
   ./bin/gitlab-mcp-server version
   ```

2. Check configured servers:
   ```bash
   gitlab-mcp-server config list
   ```

3. Test the server (replace `<YOUR_TOKEN>` with your actual token):
   ```bash
   GITLAB_TOKEN=<YOUR_TOKEN> gitlab-mcp-server stdio
   ```

4. In your development environment, verify the MCP server is available by checking the MCP server list.

## Troubleshooting

### Binary Not Found

**Problem:** `gitlab-mcp-server: command not found`

**Solutions:**
- Ensure `~/go/bin` is in your PATH:
  ```bash
  export PATH="$HOME/go/bin:$PATH"
  ```
- Use the full path to the binary in your MCP configuration:
  ```bash
  /home/user/go/bin/gitlab-mcp-server stdio
  ```

### Permission Denied

**Problem:** `Permission denied` when running the binary

**Solution:**
```bash
# If built from source
chmod +x bin/gitlab-mcp-server

# If installed via go install
chmod +x ~/go/bin/gitlab-mcp-server
```

### Token Validation Fails

**Problem:** Token validation fails on startup

**Solution:**
- Verify your token is valid in GitLab
- Check token has required scopes
- Ensure `GITLAB_HOST` matches your GitLab instance
- See [Token Management](TOKEN_MANAGEMENT.md) for details

### Docker Issues

**Problem:** Docker build or run fails

**Solution:**
- Ensure Docker is installed and running
- Check Docker has sufficient resources
- Verify network connectivity for pulling base images

## Next Steps

After installation:

1. **Configure your editor**: See [Editor Setup](EDITOR_SETUP.md)
2. **Configure servers**: See [Configuration Guide](CONFIGURATION.md)
3. **Set up your project**: See [Project Configuration](PROJECT_CONFIG.md)
4. **Explore CLI commands**: See [CLI Reference](CLI_REFERENCE.md)
5. **Explore available tools**: See [Tools Reference](TOOLS.md)
6. **Configure multiple servers**: See [Multi-Server Setup](MULTI_SERVER_SETUP.md)

