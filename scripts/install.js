#!/usr/bin/env node
/**
 * GitLab MCP Server Installer
 * A Node.js installer that safely updates MCP configuration files
 * without losing existing settings.
 */

const fs = require('fs');
const path = require('path');
const os = require('os');
const readline = require('readline');

// UI Helper Functions
function printHeader() {
  console.log('');
  console.log('╔═══════════════════════════════════════════════════════════════╗');
  console.log('║                                                               ║');
  console.log('║          GitLab MCP Server - Configuration Installer          ║');
  console.log('║                                                               ║');
  console.log('╚═══════════════════════════════════════════════════════════════╝');
  console.log('');
}

function printSeparator() {
  console.log('─────────────────────────────────────────────────────────────────');
}

function printSection(title) {
  console.log('');
  printSeparator();
  console.log(`  ${title}`);
  printSeparator();
  console.log('');
}

function printSuccess(message) {
  console.log(`  ✓ ${message}`);
}

function printError(message) {
  console.log(`  ✗ ${message}`);
}

function printInfo(message) {
  console.log(`  ℹ ${message}`);
}

function printStep(stepNum, total, message) {
  console.log(`\n[${stepNum}/${total}] ${message}`);
}

function getPlatformPaths() {
  const home = os.homedir();
  const system = os.platform();

  const paths = {
    vscode_user_settings: null,
    vscode_workspace: path.join('.vscode', 'mcp.json'),
    claude_desktop: null,
    claude_code: null,
    cursor: null,
  };

  if (system === 'win32') {
    const appdata = process.env.APPDATA || '';
    const userprofile = process.env.USERPROFILE || '';
    paths.vscode_user_settings = path.join(appdata, 'Code', 'User', 'settings.json');
    paths.claude_desktop = path.join(appdata, 'Claude', 'claude_desktop_config.json');
    paths.claude_code = path.join(userprofile, '.claude.json');
    paths.cursor = path.join(appdata, 'Cursor', 'mcp.json');
  } else if (system === 'darwin') {
    paths.vscode_user_settings = path.join(
      home,
      'Library',
      'Application Support',
      'Code',
      'User',
      'settings.json'
    );
    paths.claude_desktop = path.join(
      home,
      'Library',
      'Application Support',
      'Claude',
      'claude_desktop_config.json'
    );
    paths.claude_code = path.join(home, '.claude.json');
    paths.cursor = path.join(home, '.cursor', 'mcp.json');
  } else {
    // Linux and others
    paths.vscode_user_settings = path.join(
      home,
      '.config',
      'Code',
      'User',
      'settings.json'
    );
    paths.claude_desktop = path.join(home, '.config', 'Claude', 'claude_desktop_config.json');
    paths.claude_code = path.join(home, '.claude.json');
    paths.cursor = path.join(home, '.cursor', 'mcp.json');
  }

  return paths;
}

function createReadlineInterface() {
  return readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });
}

function question(rl, query) {
  return new Promise((resolve) => {
    rl.question(query, resolve);
  });
}

// ANSI escape codes
const ESC = '\u001b';
const CLEAR_LINE = '\r\u001b[K';
const HIDE_CURSOR = '\u001b[?25l';
const SHOW_CURSOR = '\u001b[?25h';

function getPassword(rl, prompt) {
  return new Promise((resolve) => {
    process.stdout.write(prompt);
    process.stdin.setRawMode(true);
    process.stdin.resume();
    process.stdin.setEncoding('utf8');

    let password = '';
    let buffer = '';

    const onData = (data) => {
      buffer += data.toString();

      // Handle escape sequences (arrows, etc.)
      if (buffer.startsWith(ESC)) {
        // Wait for complete sequence
        if (buffer.length < 3) {
          return;
        }
        // Ignore escape sequences for password input
        buffer = '';
        return;
      }

      // Process single character
      const char = buffer[0];
      buffer = buffer.slice(1);

      switch (char) {
        case '\n':
        case '\r':
        case '\u0004': // Ctrl+D
          process.stdin.setRawMode(false);
          process.stdin.pause();
          process.stdin.removeListener('data', onData);
          process.stdout.write('\n');
          resolve(password);
          break;
        case '\u0003': // Ctrl+C
          process.stdin.setRawMode(false);
          process.stdin.pause();
          process.stdin.removeListener('data', onData);
          process.stdout.write('\n');
          process.exit(1);
          break;
        case '\u007f': // Backspace
        case '\b': // Backspace (alternative)
          if (password.length > 0) {
            password = password.slice(0, -1);
            process.stdout.write('\b \b');
          }
          break;
        default:
          if (char && char.charCodeAt(0) >= 32) {
            // Only printable characters
            password += char;
            process.stdout.write('●');
          }
          break;
      }
    };

    process.stdin.on('data', onData);
  });
}

function selectMenu(options, prompt, defaultIndex = 0, multiSelect = false) {
  return new Promise((resolve) => {
    if (options.length === 0) {
      resolve(multiSelect ? [] : defaultIndex);
      return;
    }

    let selectedIndex = defaultIndex >= 0 && defaultIndex < options.length ? defaultIndex : 0;
    let selectedIndices = multiSelect ? new Set() : null;
    let buffer = '';

    process.stdout.write(prompt);
    process.stdout.write('\n');
    if (multiSelect) {
      process.stdout.write('  (Space to select/deselect, Enter to confirm)\n');
    }

    function render() {
      // Move cursor up to redraw menu
      const linesToMove = options.length + (multiSelect ? 2 : 1);
      process.stdout.write(ESC + '[A'.repeat(linesToMove));
      process.stdout.write(CLEAR_LINE);

      options.forEach((option, index) => {
        process.stdout.write(CLEAR_LINE);
        const isSelected = multiSelect ? selectedIndices.has(index) : index === selectedIndex;
        const isHighlighted = index === selectedIndex;

        let prefix = '    ';
        if (isHighlighted) {
          prefix = multiSelect
            ? `  ${ESC}[1m>${ESC}[0m `
            : `  ${ESC}[1m>${ESC}[0m `;
        } else {
          prefix = multiSelect ? '    ' : '    ';
        }

        const checkbox = multiSelect ? (isSelected ? '[✓]' : '[ ]') : '';
        const style = isHighlighted ? ESC + '[1m' : '';
        const reset = isHighlighted ? ESC + '[0m' : '';

        process.stdout.write(`${prefix}${checkbox} ${style}${option}${reset}\n`);
      });
    }

    process.stdin.setRawMode(true);
    process.stdin.resume();
    process.stdin.setEncoding('utf8');

    render();

    const onData = (data) => {
      buffer += data.toString();

      // Handle escape sequences
      if (buffer.startsWith(ESC)) {
        // Check if we have a complete sequence [A or [B
        if (buffer.length >= 3 && buffer[1] === '[') {
          const code = buffer[2];
          if (code === 'A') {
            // Up arrow
            selectedIndex = (selectedIndex - 1 + options.length) % options.length;
            render();
            buffer = '';
            return;
          } else if (code === 'B') {
            // Down arrow
            selectedIndex = (selectedIndex + 1) % options.length;
            render();
            buffer = '';
            return;
          } else if (code >= '0' && code <= '9') {
            // Multi-character sequence, wait for more
            if (buffer.length < 4) return;
          }
          // Unknown or complete escape sequence, ignore
          buffer = '';
          return;
        } else if (buffer.length > 10) {
          // Too long, probably not an escape sequence
          buffer = '';
          return;
        } else {
          // Wait for more characters
          return;
        }
      }

      // Process single character (not part of escape sequence)
      if (buffer.length > 0 && !buffer.startsWith(ESC)) {
        const char = buffer[0];
        buffer = '';

        switch (char) {
          case ' ':
            // Space - toggle selection in multi-select mode
            if (multiSelect) {
              if (selectedIndices.has(selectedIndex)) {
                selectedIndices.delete(selectedIndex);
              } else {
                selectedIndices.add(selectedIndex);
              }
              render();
            }
            break;
          case '\n':
          case '\r':
            // Enter - confirm selection
            process.stdin.setRawMode(false);
            process.stdin.pause();
            process.stdin.removeListener('data', onData);
            process.stdout.write('\n');
            if (multiSelect) {
              // If nothing selected, select all
              if (selectedIndices.size === 0) {
                resolve(Array.from({ length: options.length }, (_, i) => i));
              } else {
                resolve(Array.from(selectedIndices).sort((a, b) => a - b));
              }
            } else {
              resolve(selectedIndex);
            }
            return;
          case '\u0003': // Ctrl+C
            process.stdin.setRawMode(false);
            process.stdin.pause();
            process.stdin.removeListener('data', onData);
            process.stdout.write('\n');
            process.exit(1);
            break;
          case '\u0004': // Ctrl+D
            process.stdin.setRawMode(false);
            process.stdin.pause();
            process.stdin.removeListener('data', onData);
            process.stdout.write('\n');
            process.exit(1);
            break;
          default:
            // Ignore other characters
            break;
        }
      }
    };

    process.stdin.on('data', onData);
  });
}

function selectYesNo(prompt, defaultValue = false) {
  return new Promise((resolve) => {
    const options = ['Yes', 'No'];
    const defaultIndex = defaultValue ? 0 : 1;

    selectMenu(options, prompt, defaultIndex).then((index) => {
      resolve(index === 0);
    });
  });
}

async function promptUser(rl) {
  const config = {};

  printInfo('Configure single GitLab server');
  console.log('');

  // Prompt for mode with menu
  const modeIndex = await selectMenu(
    ['Local binary', 'Docker'],
    '  Deployment mode:',
    0
  );
  config.mode = modeIndex === 0 ? 'local' : 'docker';
  printInfo(`Using ${modeIndex === 0 ? 'local binary' : 'Docker'} deployment`);

  // Prompt for GitLab host
  const hostInput = await question(
    rl,
    '  GitLab host URL (default: https://gitlab.com, press Enter to use default): '
  );
  config.gitlab_host = hostInput.trim() || 'https://gitlab.com';
  printInfo(`GitLab host: ${config.gitlab_host}`);

  // Prompt for token (hidden input with visual feedback)
  while (true) {
    const token = await getPassword(rl, '  GitLab access token: ');
    const trimmedToken = token.trim();
    if (trimmedToken) {
      config.token = trimmedToken;
      printSuccess('Token received');
      break;
    }
    printError('Token cannot be empty. Please try again.');
  }

  // Prompt for read-only mode with menu
  config.readonly = await selectYesNo('  Enable read-only mode?', false);
  if (config.readonly) {
    printInfo('Read-only mode enabled');
  }

  return config;
}

async function promptMultipleServers(rl) {
  console.log('This installer can configure one or multiple GitLab servers.');
  console.log('Use multiple servers if you work with different GitLab instances');
  console.log('(e.g., work GitLab, personal GitLab, self-hosted GitLab).');
  console.log('');
  return await selectYesNo('Configure multiple GitLab servers?', false);
}

async function collectServers(rl) {
  const servers = [];
  let serverNum = 0;
  let globalMode = 'local';

  printSection('Multi-Server Configuration');
  printInfo('You can configure multiple GitLab instances (e.g., work, personal, client)');
  printInfo('Each server will have a unique name and can use different tokens and hosts.');
  console.log('');

  while (true) {
    serverNum++;
    printSection(`Server ${serverNum} Configuration`);

    // Server name
    let serverName;
    while (true) {
      const name = await question(rl, "  Server name (e.g., 'work', 'personal', 'gitlab'): ");
      const trimmedName = name.trim();
      if (trimmedName) {
        if (servers.some((s) => s.name === trimmedName)) {
          printError(`Server '${trimmedName}' already configured. Please use a different name.`);
        } else {
          serverName = trimmedName;
          printSuccess(`Server name: ${serverName}`);
          break;
        }
      } else {
        printError('Server name cannot be empty');
      }
    }

    const config = {};

    // Prompt for mode (use same for all servers for simplicity)
    if (serverNum === 1) {
      const modeIndex = await selectMenu(
        ['Local binary', 'Docker'],
        '  Deployment mode (applies to all servers):',
        0
      );
      config.mode = modeIndex === 0 ? 'local' : 'docker';
      globalMode = config.mode;
      printInfo(`Using ${modeIndex === 0 ? 'local binary' : 'Docker'} deployment (applies to all servers)`);
    } else {
      config.mode = globalMode;
      printInfo(`Deployment mode: ${globalMode} (same as first server)`);
    }

    // Prompt for GitLab host
    const hostInput = await question(
      rl,
      `  GitLab host URL for '${serverName}' (default: https://gitlab.com, press Enter to use default): `
    );
    config.gitlab_host = hostInput.trim() || 'https://gitlab.com';
    printInfo(`GitLab host: ${config.gitlab_host}`);

    // Prompt for token (hidden input)
    while (true) {
      const token = await getPassword(rl, `  GitLab access token for '${serverName}': `);
      const trimmedToken = token.trim();
      if (trimmedToken) {
        config.token = trimmedToken;
        printSuccess('Token received');
        break;
      }
      printError('Token cannot be empty. Please try again.');
    }

    // Prompt for read-only mode
    config.readonly = await selectYesNo('  Enable read-only mode?', false);
    if (config.readonly) {
      printInfo('Read-only mode enabled');
    }

    // Add to servers list with the name
    config.name = serverName;
    servers.push(config);

    printSuccess(`Server '${serverName}' configured successfully`);
    console.log('');

    // Ask if user wants to add more servers
    while (true) {
      const more = await question(rl, 'Add another server? (y/n, default: n): ');
      const trimmed = more.trim().toLowerCase();
      if (trimmed === '' || trimmed === 'n') {
        return servers;
      } else if (trimmed === 'y') {
        break;
      }
      printError("Please enter 'y' or 'n'");
    }
  }
}

async function promptEnvironments(rl) {
  const environments = ['VS Code', 'Claude Desktop', 'Claude Code', 'Cursor'];

  printSection('Development Environment Selection');
  printInfo('Select which development environments to configure:');
  console.log('');

  const selectedIndices = await selectMenu(environments, '  Select environments (Space to toggle, Enter to confirm):', 0, true);

  if (selectedIndices.length === 0 || selectedIndices.length === environments.length) {
    printInfo('Configuring all environments');
    return environments;
  } else {
    const selected = selectedIndices.map((idx) => environments[idx]);
    printInfo(`Selected: ${selected.join(', ')}`);
    return selected;
  }
}

function getBinaryConfig(mode, projectRoot) {
  const config = { mode, env: {} };

  if (mode === 'docker') {
    config.docker_image = 'gitlab-mcp-server:latest';
    config.command = 'docker';
    config.args = ['run', '-i', '--rm', '-e', 'GITLAB_TOKEN', '-e', 'GITLAB_HOST'];
    config.args.push(config.docker_image);
  } else {
    // local
    const binaryPath = path.join(projectRoot, 'bin', 'gitlab-mcp-server');
    const isWindows = os.platform() === 'win32';
    const fullBinaryPath = isWindows ? `${binaryPath}.exe` : binaryPath;

    if (!fs.existsSync(fullBinaryPath)) {
      printError(`Binary not found at ${fullBinaryPath}`);
      console.log('');
      printInfo('Please run \'make build\' first to build the binary');
      process.exit(1);
    }

    const absPath = path.resolve(fullBinaryPath);
    config.command = absPath;
    config.args = ['stdio'];
  }

  return config;
}

function createServerConfig(binaryConfig, serverConfigData) {
  const serverConfig = {
    command: binaryConfig.command,
    args: binaryConfig.args,
    env: {},
  };

  // Copy env from binary config
  if (binaryConfig.env) {
    Object.assign(serverConfig.env, binaryConfig.env);
  }

  // Add GitLab-specific environment variables
  serverConfig.env.GITLAB_TOKEN = serverConfigData.token;
  if (serverConfigData.gitlab_host) {
    serverConfig.env.GITLAB_HOST = serverConfigData.gitlab_host;
  }

  if (serverConfigData.readonly) {
    serverConfig.env.GITLAB_READ_ONLY = 'true';
  }

  return serverConfig;
}

function updateJsonFile(filePath, updateFunc, description) {
  let data = {};

  // Read existing file or create new structure
  if (fs.existsSync(filePath)) {
    try {
      const content = fs.readFileSync(filePath, 'utf8');
      data = JSON.parse(content);
    } catch (e) {
      // Silently handle parse errors - will create new config
      data = {};
    }
  }

  // Create backup if file exists
  if (fs.existsSync(filePath)) {
    const backupPath = filePath + '.bak';
    try {
      fs.copyFileSync(filePath, backupPath);
    } catch (e) {
      // Continue even if backup fails
    }
  }

  // Apply updates
  updateFunc(data);

  // Create parent directories if needed
  const dir = path.dirname(filePath);
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
  }

  // Write updated config
  try {
    const content = JSON.stringify(data, null, 2) + '\n';
    fs.writeFileSync(filePath, content, 'utf8');
    return true;
  } catch (e) {
    // Try to restore backup
    const backupPath = filePath + '.bak';
    if (fs.existsSync(backupPath)) {
      try {
        fs.copyFileSync(backupPath, filePath);
      } catch {
        // Ignore restore errors
      }
    }
    printError(`Failed to write ${description}: ${e.message}`);
    return false;
  }
}

function updateVscodeWorkspace(filePath, serverName, serverConfig) {
  return updateJsonFile(
    filePath,
    (data) => {
      if (!data.servers) {
        data.servers = {};
      }
      data.servers[serverName] = serverConfig;
    },
    'VS Code workspace config'
  );
}

function updateVscodeUser(filePath, serverName, serverConfig) {
  return updateJsonFile(
    filePath,
    (data) => {
      if (!data.mcp) {
        data.mcp = {};
      }
      if (!data.mcp.servers) {
        data.mcp.servers = {};
      }
      data.mcp.servers[serverName] = serverConfig;
    },
    'VS Code user settings'
  );
}

function updateClaudeConfig(filePath, serverName, serverConfig, isClaudeCode = false) {
  return updateJsonFile(
    filePath,
    (data) => {
      if (!data.mcpServers) {
        data.mcpServers = {};
      }

      // Add type field for Claude Code
      const config = { ...serverConfig };
      if (isClaudeCode) {
        config.type = 'stdio';
      }

      data.mcpServers[serverName] = config;
    },
    isClaudeCode ? 'Claude Code config' : 'Claude/Cursor config'
  );
}

function updateVscode(paths, serverName, serverConfig) {
  // Try workspace config first
  if (paths.vscode_workspace) {
    try {
      if (updateVscodeWorkspace(paths.vscode_workspace, serverName, serverConfig)) {
        return true;
      }
    } catch (e) {
      // Ignore and fall back
    }
  }

  // Fall back to user settings
  if (paths.vscode_user_settings) {
    return updateVscodeUser(paths.vscode_user_settings, serverName, serverConfig);
  }

  return false;
}

function updateEnvironment(env, paths, serverName, serverConfig) {
  let success = false;
  let error = null;

  try {
    if (env === 'VS Code') {
      success = updateVscode(paths, serverName, serverConfig);
    } else if (env === 'Claude Desktop') {
      if (paths.claude_desktop) {
        success = updateClaudeConfig(paths.claude_desktop, serverName, serverConfig, false);
      } else {
        error = 'Path not available';
      }
    } else if (env === 'Claude Code') {
      if (paths.claude_code) {
        success = updateClaudeConfig(paths.claude_code, serverName, serverConfig, true);
      } else {
        error = 'Path not available';
      }
    } else if (env === 'Cursor') {
      if (paths.cursor) {
        success = updateClaudeConfig(paths.cursor, serverName, serverConfig, false);
      } else {
        error = 'Path not available';
      }
    }
  } catch (e) {
    error = e.message;
  }

  if (success) {
    printSuccess(`${env} configured successfully`);
    return true;
  } else {
    printError(`Failed to configure ${env}${error ? `: ${error}` : ''}`);
    return false;
  }
}

function getProjectRoot() {
  let currentDir = process.cwd();
  const root = path.parse(currentDir).root;

  while (currentDir !== root) {
    const goModPath = path.join(currentDir, 'go.mod');
    if (fs.existsSync(goModPath)) {
      return currentDir;
    }
    currentDir = path.dirname(currentDir);
  }

  return process.cwd();
}

async function main() {
  printHeader();

  const rl = createReadlineInterface();

  // Handle Ctrl+C
  rl.on('SIGINT', () => {
    console.log('\n\nInstallation cancelled by user.');
    rl.close();
    process.exit(1);
  });

  try {
    // Get project root
    const projectRoot = getProjectRoot();

    // Check if user wants multi-server configuration
    let multiServer;
    try {
      multiServer = await promptMultipleServers(rl);
    } catch (e) {
      console.log('\n\nInstallation cancelled.');
      rl.close();
      process.exit(1);
    }

    // Collect server configurations
    let servers = [];
    try {
      if (multiServer) {
        servers = await collectServers(rl);
      } else {
        // Single server configuration - use default name "gitlab"
        printSection('Single Server Configuration');
        const config = await promptUser(rl);
        config.name = 'gitlab';
        servers = [config];
        printSuccess('Server configuration completed');
      }
    } catch (e) {
      console.log('\n\nInstallation cancelled.');
      rl.close();
      process.exit(1);
    }

    // Prompt for environments
    let environments;
    try {
      environments = await promptEnvironments(rl);
    } catch (e) {
      console.log('\n\nInstallation cancelled.');
      rl.close();
      process.exit(1);
    }

    // Get configuration paths
    const paths = getPlatformPaths();

    // Process each server
    printSection('Applying Configuration');
    printInfo(`Configuring ${servers.length} server(s) for ${environments.length} environment(s)...`);
    console.log('');

    for (const serverConfigData of servers) {
      const serverName = serverConfigData.name;
      console.log(`Configuring server: ${serverName}`);

      // Get binary configuration
      let binaryConfig;
      try {
        binaryConfig = getBinaryConfig(serverConfigData.mode, projectRoot);
      } catch (e) {
        printError(e.message);
        rl.close();
        process.exit(1);
      }

      // Add read-only environment variable if needed
      if (serverConfigData.readonly) {
        if (binaryConfig.mode === 'docker') {
          // Insert before the image name (last arg)
          binaryConfig.args.splice(-1, 0, '-e', 'GITLAB_READ_ONLY');
        }
        binaryConfig.env.GITLAB_READ_ONLY = 'true';
      }

      // Create server configuration
      const serverConfig = createServerConfig(binaryConfig, serverConfigData);

      // Update configurations for selected environments
      for (const env of environments) {
        updateEnvironment(env, paths, serverName, serverConfig);
      }
      console.log('');
    }

    // Summary
    printSection('Installation Complete');
    console.log('Successfully configured:');
    console.log('');
    console.log(`  Servers (${servers.length}):`);
    for (const server of servers) {
      const modeStr = server.mode === 'docker' ? 'Docker' : 'Local binary';
      const readonlyStr = server.readonly ? ' [read-only]' : '';
      console.log(`    • ${server.name}: ${server.gitlab_host} (${modeStr})${readonlyStr}`);
    }
    console.log('');
    console.log(`  Environments (${environments.length}):`);
    environments.forEach((env) => {
      console.log(`    • ${env}`);
    });

    console.log('');
    printSection('Next Steps');
    console.log('1. Restart your development environment(s) to load the new configuration');
    console.log('2. The MCP server(s) will be available with the name(s) you chose');
    if (servers[0].mode === 'local') {
      console.log('3. Ensure the binary exists at the specified path');
    } else {
      console.log('3. Build the Docker image: docker build -t gitlab-mcp-server:latest .');
    }
    console.log('');
    printSection('Usage');
    if (servers.length === 1) {
      console.log(`Use MCP tools with server name: ${servers[0].name}`);
    } else {
      console.log('For multiple servers:');
      console.log('  • Use .gmcprc files to specify which server to use for each project');
      console.log('  • Run the \'setCurrentProject\' tool to configure project-specific server');
      console.log('  • See docs/MULTI_SERVER_SETUP.md for detailed instructions');
    }
    console.log('');
    printSeparator();
    console.log('');
  } finally {
    rl.close();
  }
}

if (require.main === module) {
  main().catch((e) => {
    console.error('Unexpected error:', e);
    process.exit(1);
  });
}


