#!/usr/bin/env python3
"""
GitLab MCP Server Installer
A Python installer that safely updates MCP configuration files
without losing existing settings.
"""

import json
import os
import platform
import sys
from pathlib import Path
from getpass import getpass


def get_platform_paths():
    """Get platform-specific paths to MCP configuration files."""
    home = Path.home()
    system = platform.system()

    paths = {
        "vscode_user_settings": None,
        "vscode_workspace": Path(".vscode/mcp.json"),
        "claude_desktop": None,
        "claude_code": None,
        "cursor": None,
    }

    if system == "Windows":
        appdata = Path(os.environ.get("APPDATA", ""))
        userprofile = Path(os.environ.get("USERPROFILE", ""))
        paths["vscode_user_settings"] = appdata / "Code" / "User" / "settings.json"
        paths["claude_desktop"] = appdata / "Claude" / "claude_desktop_config.json"
        paths["claude_code"] = userprofile / ".claude.json"
        paths["cursor"] = appdata / "Cursor" / "mcp.json"
    elif system == "Darwin":  # macOS
        paths["vscode_user_settings"] = (
            home / "Library" / "Application Support" / "Code" / "User" / "settings.json"
        )
        paths["claude_desktop"] = (
            home / "Library" / "Application Support" / "Claude" / "claude_desktop_config.json"
        )
        paths["claude_code"] = home / ".claude.json"
        paths["cursor"] = home / ".cursor" / "mcp.json"
    else:  # Linux and others
        paths["vscode_user_settings"] = (
            home / ".config" / "Code" / "User" / "settings.json"
        )
        paths["claude_desktop"] = (
            home / ".config" / "Claude" / "claude_desktop_config.json"
        )
        paths["claude_code"] = home / ".claude.json"
        paths["cursor"] = home / ".cursor" / "mcp.json"

    return paths


def prompt_user():
    """Collect configuration from user via interactive prompts."""
    config = {}

    # Prompt for mode
    while True:
        mode_input = input("Select mode [local/docker] (default: local): ").strip()
        if mode_input == "" or mode_input == "local":
            config["mode"] = "local"
            break
        elif mode_input == "docker":
            config["mode"] = "docker"
            break
        else:
            print(f"Invalid mode: {mode_input}. Must be 'local' or 'docker'")

    # Prompt for GitLab host
    host_input = input(
        "GitLab host URL (default: https://gitlab.com, press Enter to use default): "
    ).strip()
    config["gitlab_host"] = host_input if host_input else "https://gitlab.com"

    # Prompt for token (hidden input)
    while True:
        token = getpass("GitLab access token: ")
        token = token.strip()
        if token:
            config["token"] = token
            break
        print("Error: token cannot be empty")

    # Prompt for read-only mode
    readonly_input = input("Enable read-only mode? (y/n, default: n): ").strip().lower()
    config["readonly"] = readonly_input in ("y", "yes")

    return config


def prompt_multiple_servers():
    """Ask if user wants to configure multiple GitLab servers."""
    while True:
        choice = input("Configure multiple GitLab servers? (y/n, default: n): ").strip().lower()
        if choice == "" or choice == "n":
            return False
        elif choice == "y":
            return True
        print("Please enter 'y' or 'n'")


def collect_servers():
    """Collect multiple GitLab server configurations."""
    servers = []
    server_num = 0

    while True:
        server_num += 1
        print(f"\n=== Configuring Server {server_num} ===")

        # Server name
        while True:
            name = input("Server name (e.g., 'work', 'personal', 'gitlab'): ").strip()
            if name:
                # Check if name already exists
                if any(s["name"] == name for s in servers):
                    print(f"Error: Server '{name}' already configured. Please use a different name.")
                else:
                    server_name = name
                    break
            print("Error: Server name cannot be empty")

        # Collect the rest of the configuration
        # We can reuse most of prompt_user logic but need to adapt
        config = {}

        # Prompt for mode (use same for all servers for simplicity)
        if server_num == 1:
            while True:
                mode_input = input("Select mode [local/docker] (default: local): ").strip()
                if mode_input == "" or mode_input == "local":
                    config["mode"] = "local"
                    global_mode = "local"
                    break
                elif mode_input == "docker":
                    config["mode"] = "docker"
                    global_mode = "docker"
                    break
                else:
                    print(f"Invalid mode: {mode_input}. Must be 'local' or 'docker'")
        else:
            config["mode"] = global_mode

        # Prompt for GitLab host
        host_input = input(
            f"GitLab host URL for '{server_name}' (default: https://gitlab.com, press Enter to use default): "
        ).strip()
        config["gitlab_host"] = host_input if host_input else "https://gitlab.com"

        # Prompt for token (hidden input)
        while True:
            token = getpass(f"GitLab access token for '{server_name}': ")
            token = token.strip()
            if token:
                config["token"] = token
                break
            print("Error: token cannot be empty")

        # Prompt for read-only mode
        readonly_input = input("Enable read-only mode? (y/n, default: n): ").strip().lower()
        config["readonly"] = readonly_input in ("y", "yes")

        # Add to servers list with the name
        config["name"] = server_name
        servers.append(config)

        # Ask if user wants to add more servers
        print()
        while True:
            more = input("Add another server? (y/n, default: n): ").strip().lower()
            if more == "" or more == "n":
                return servers
            elif more == "y":
                break
            print("Please enter 'y' or 'n'")


def prompt_environments():
    """Ask user which development environments to configure."""
    environments = ["VS Code", "Claude Desktop", "Claude Code", "Cursor"]

    print("\nSelect development environments to configure (comma-separated, or 'all'):")
    for i, env in enumerate(environments, 1):
        print(f"  {i}. {env}")

    choice = input("Your choice (default: all): ").strip()

    if not choice or choice.lower() == "all":
        return environments

    selected = []
    for part in choice.split(","):
        part = part.strip()
        if not part:
            continue

        # Try to parse as number
        try:
            idx = int(part)
            if 1 <= idx <= len(environments):
                selected.append(environments[idx - 1])
        except ValueError:
            # Try to match by name
            found = False
            for env in environments:
                if part.lower() == env.lower():
                    selected.append(env)
                    found = True
                    break
            if not found:
                print(f"Warning: Unknown environment '{part}', skipping")

    return selected if selected else environments


def get_binary_config(mode, project_root):
    """Get binary configuration based on mode."""
    config = {"mode": mode, "env": {}}

    if mode == "docker":
        config["docker_image"] = "gitlab-mcp-server:latest"
        config["command"] = "docker"
        config["args"] = [
            "run",
            "-i",
            "--rm",
            "-e",
            "GITLAB_TOKEN",
            "-e",
            "GITLAB_HOST",
        ]
        config["args"].append(config["docker_image"])
    else:  # local
        binary_path = Path(project_root) / "bin" / "gitlab-mcp-server"

        if not binary_path.exists():
            print(f"Error: Binary not found at {binary_path}")
            print("Please run 'make build' first")
            sys.exit(1)

        abs_path = binary_path.resolve()
        config["command"] = str(abs_path)
        config["args"] = ["stdio"]

    return config


def create_server_config(binary_config, server_config_data):
    """Create server configuration for MCP."""
    server_config = {
        "command": binary_config["command"],
        "args": binary_config["args"],
        "env": {},
    }

    # Copy env from binary config
    if "env" in binary_config:
        server_config["env"].update(binary_config["env"])

    # Add GitLab-specific environment variables
    server_config["env"]["GITLAB_TOKEN"] = server_config_data["token"]
    if server_config_data.get("gitlab_host"):
        server_config["env"]["GITLAB_HOST"] = server_config_data["gitlab_host"]

    if server_config_data.get("readonly"):
        server_config["env"]["GITLAB_READ_ONLY"] = "true"

    return server_config


def update_json_file(path, update_func, description):
    """Safely update a JSON file while preserving all existing data."""
    path = Path(path)

    # Read existing file or create new structure
    if path.exists():
        try:
            with open(path, "r", encoding="utf-8") as f:
                data = json.load(f)
        except json.JSONDecodeError as e:
            print(f"  Warning: Failed to parse existing {description}: {e}")
            data = {}
    else:
        data = {}

    # Create backup if file exists
    if path.exists():
        backup_path = path.with_suffix(".json.bak")
        try:
            backup_path.write_bytes(path.read_bytes())
        except Exception as e:
            print(f"  Warning: Failed to create backup: {e}")

    # Apply updates
    update_func(data)

    # Create parent directories if needed
    path.parent.mkdir(parents=True, exist_ok=True)

    # Write updated config
    try:
        with open(path, "w", encoding="utf-8") as f:
            json.dump(data, f, indent=2)
            f.write("\n")  # Add trailing newline
        return True
    except Exception as e:
        # Try to restore backup
        backup_path = path.with_suffix(".json.bak")
        if backup_path.exists():
            try:
                path.write_bytes(backup_path.read_bytes())
            except Exception:
                pass
        print(f"  Error: Failed to write {description}: {e}")
        return False


def update_vscode_workspace(path, server_name, server_config):
    """Update VS Code workspace .vscode/mcp.json."""
    def update(data):
        if "servers" not in data:
            data["servers"] = {}
        data["servers"][server_name] = server_config

    return update_json_file(path, update, "VS Code workspace config")


def update_vscode_user(path, server_name, server_config):
    """Update VS Code user settings.json."""
    def update(data):
        if "mcp" not in data:
            data["mcp"] = {}
        if "servers" not in data["mcp"]:
            data["mcp"]["servers"] = {}
        data["mcp"]["servers"][server_name] = server_config

    return update_json_file(path, update, "VS Code user settings")


def update_claude_config(path, server_name, server_config, is_claude_code=False):
    """Update Claude Desktop/Cursor/Claude Code configuration."""
    def update(data):
        if "mcpServers" not in data:
            data["mcpServers"] = {}

        # Add type field for Claude Code
        config = dict(server_config)
        if is_claude_code:
            config["type"] = "stdio"

        data["mcpServers"][server_name] = config

    desc = "Claude Code config" if is_claude_code else "Claude/Cursor config"
    return update_json_file(path, update, desc)


def update_vscode(paths, server_name, server_config):
    """Update VS Code configuration."""
    # Try workspace config first
    if paths["vscode_workspace"]:
        try:
            if update_vscode_workspace(paths["vscode_workspace"], server_name, server_config):
                return True
        except Exception:
            pass

    # Fall back to user settings
    if paths["vscode_user_settings"]:
        return update_vscode_user(paths["vscode_user_settings"], server_name, server_config)

    return False


def update_environment(env, paths, server_name, server_config):
    """Update configuration for a specific environment."""
    print(f"\nConfiguring {env}...")

    success = False
    error = None

    try:
        if env == "VS Code":
            success = update_vscode(paths, server_name, server_config)
        elif env == "Claude Desktop":
            if paths["claude_desktop"]:
                success = update_claude_config(
                    paths["claude_desktop"], server_name, server_config, is_claude_code=False
                )
            else:
                error = "Path not available"
        elif env == "Claude Code":
            if paths["claude_code"]:
                success = update_claude_config(
                    paths["claude_code"], server_name, server_config, is_claude_code=True
                )
            else:
                error = "Path not available"
        elif env == "Cursor":
            if paths["cursor"]:
                success = update_claude_config(
                    paths["cursor"], server_name, server_config, is_claude_code=False
                )
            else:
                error = "Path not available"
    except Exception as e:
        error = str(e)

    if success:
        print(f"  ✓ {env} configured successfully")
        return True
    else:
        print(f"  ✗ Error configuring {env}" + (f": {error}" if error else ""))
        return False


def get_project_root():
    """Find project root by looking for go.mod file."""
    cwd = Path.cwd()
    for path in [cwd] + list(cwd.parents):
        if (path / "go.mod").exists():
            return path
    return cwd


def main():
    print("=== GitLab MCP Server Installer ===")
    print()

    # Get project root
    project_root = get_project_root()

    # Check if user wants multi-server configuration
    try:
        multi_server = prompt_multiple_servers()
    except (KeyboardInterrupt, EOFError):
        print("\nInstallation cancelled")
        sys.exit(1)

    # Collect server configurations
    servers = []
    try:
        if multi_server:
            servers = collect_servers()
        else:
            # Single server configuration - use default name "gitlab"
            config = prompt_user()
            config["name"] = "gitlab"
            servers = [config]
    except (KeyboardInterrupt, EOFError):
        print("\nInstallation cancelled")
        sys.exit(1)

    # Prompt for environments
    try:
        environments = prompt_environments()
    except (KeyboardInterrupt, EOFError):
        print("\nInstallation cancelled")
        sys.exit(1)

    # Get configuration paths
    paths = get_platform_paths()

    # Process each server
    print(f"\n=== Configuring {len(servers)} server(s) ===")
    for server_config_data in servers:
        server_name = server_config_data["name"]
        print(f"\n--- Processing server: {server_name} ---")

        # Get binary configuration
        try:
            binary_config = get_binary_config(server_config_data["mode"], project_root)
        except SystemExit:
            raise
        except Exception as e:
            print(f"Error: {e}")
            sys.exit(1)

        # Add read-only environment variable if needed
        if server_config_data["readonly"]:
            if binary_config["mode"] == "docker":
                # Insert before the image name (last arg)
                binary_config["args"].insert(-1, "-e")
                binary_config["args"].insert(-1, "GITLAB_READ_ONLY")
            binary_config["env"]["GITLAB_READ_ONLY"] = "true"

        # Create server configuration
        server_config = create_server_config(binary_config, server_config_data)

        # Update configurations for selected environments
        for env in environments:
            update_environment(env, paths, server_name, server_config)

    # Summary
    print()
    print(f"=== Installation Complete ===")
    print(f"Configured {len(servers)} GitLab server(s):")
    for server in servers:
        mode_str = "Docker" if server["mode"] == "docker" else "Local binary"
        print(f"  - {server['name']}: {server['gitlab_host']} ({mode_str})")

    print(f"\nConfigured {len(environments)} development environment(s)")
    print("\nNext steps:")
    print("1. Restart your development environment(s)")
    print("2. The MCP server(s) will be available with the name(s) you chose")
    if servers[0]["mode"] == "local":
        print(f"3. Make sure the binary exists at the specified path")
    else:
        print("3. Make sure Docker image exists: docker build -t gitlab-mcp-server:latest .")
    print("\nUsage:")
    if len(servers) == 1:
        print(f"  - Use MCP tools with server: {servers[0]['name']}")
    else:
        print("  - Use .gmcprc files to specify which server to use for each project")
        print("  - Run 'setCurrentProject' tool to configure project-specific server")


if __name__ == "__main__":
    main()
