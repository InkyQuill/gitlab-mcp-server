#!/usr/bin/env node
/**
 * GitLab MCP Server Installer
 * A Node.js installer that safely updates MCP configuration files
 * without losing existing settings.
 *
 * Features:
 * - Inquirer.js-style prompts (no external dependencies)
 * - Config persistence for reuse
 * - Auto-detection of installed environments
 * - Multi-server support
 * - Read existing IDE configurations
 * - Manage servers (add, edit, delete)
 * - Sync with IDE configurations
 */

const fs = require('fs');
const path = require('path');
const os = require('os');
const readline = require('readline');

// =====================
// UI Helper Functions
// =====================

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

// =====================
// Platform Paths
// =====================

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

// =====================
// Low-level Input Functions
// =====================

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
const CURSOR_UP = (n) => `\u001b[${n}A`;
const CURSOR_DOWN = (n) => `\u001b[${n}B`;
const ERASE_LINES = (n) => `\u001b[${n}A\u001b[0J`;

// Screen manager for proper rendering
class ScreenManager {
  constructor() {
    this.height = 0;
    this.extraLinesUnderPrompt = 0;
  }

  render(content, bottomContent = '') {
    const lines = content.split('\n');
    const bottomLines = bottomContent ? bottomContent.split('\n') : [];
    const totalHeight = lines.length + bottomLines.length;

    // Move cursor up and erase previous content
    if (this.height > 0) {
      process.stdout.write(ERASE_LINES(this.height));
    }

    // Write new content
    process.stdout.write(content);
    if (bottomContent) {
      process.stdout.write('\n' + bottomContent);
    }

    this.height = totalHeight;
  }

  done(clearContent = false) {
    if (clearContent && this.height > 0) {
      process.stdout.write(ERASE_LINES(this.height));
    } else if (this.height > 0) {
      process.stdout.write('\n');
    }
    process.stdout.write(SHOW_CURSOR);
    this.height = 0;
  }
}

function formatPassword(value, showChars = 3) {
  if (value.length === 0) {
    return '';
  }
  if (value.length <= showChars * 2) {
    // If password is short, show all as dots
    return '●'.repeat(value.length);
  }
  // Show first N chars, then dots, then last N chars
  const start = value.substring(0, showChars);
  const end = value.substring(value.length - showChars);
  const dots = '●'.repeat(Math.max(0, value.length - showChars * 2));
  return `${start}${dots}${end}`;
}

function getPassword(rl, prompt) {
  return new Promise((resolve) => {
    process.stdout.write(prompt);
    process.stdin.setRawMode(true);
    process.stdin.resume();
    process.stdin.setEncoding('utf8');

    let password = '';
    let buffer = '';
    let lastDisplayLength = 0;

    const updateDisplay = () => {
      // Clear previous display
      for (let i = 0; i < lastDisplayLength; i++) {
        process.stdout.write('\b \b');
      }
      // Write new display
      const display = formatPassword(password);
      process.stdout.write(display);
      lastDisplayLength = display.length;
    };

    const onData = (data) => {
      const input = data.toString();
      buffer += input;

      // Handle escape sequences (arrows, etc.)
      if (buffer.startsWith(ESC)) {
        if (buffer.length < 3) {
          return;
        }
        buffer = '';
        return;
      }

      // Process all characters in buffer (handles paste)
      while (buffer.length > 0 && !buffer.startsWith(ESC)) {
        const char = buffer[0];
        buffer = buffer.slice(1);

        switch (char) {
          case '\n':
          case '\r':
            // Enter - finish input
            process.stdin.setRawMode(false);
            process.stdin.pause();
            process.stdin.removeListener('data', onData);
            // Clear display and move to new line
            for (let i = 0; i < lastDisplayLength; i++) {
              process.stdout.write('\b \b');
            }
            process.stdout.write('\n');
            resolve(password);
            return;

          case '\u0004': // Ctrl+D
            process.stdin.setRawMode(false);
            process.stdin.pause();
            process.stdin.removeListener('data', onData);
            for (let i = 0; i < lastDisplayLength; i++) {
              process.stdout.write('\b \b');
            }
            process.stdout.write('\n');
            resolve(password);
            return;

          case '\u0003': // Ctrl+C
            process.stdin.setRawMode(false);
            process.stdin.pause();
            process.stdin.removeListener('data', onData);
            for (let i = 0; i < lastDisplayLength; i++) {
              process.stdout.write('\b \b');
            }
            process.stdout.write('\n');
            process.exit(1);
            return;

          case '\u007f': // Backspace
          case '\b': // Backspace (alternative)
            if (password.length > 0) {
              password = password.slice(0, -1);
              updateDisplay();
            }
            break;

          default:
            // Add printable characters
            if (char && char.charCodeAt(0) >= 32 && char !== '\n' && char !== '\r') {
              password += char;
              updateDisplay();
            }
            break;
        }
      }
    };

    process.stdin.on('data', onData);
  });
}

function selectMenu(options, prompt, defaultIndex = 0, multiSelect = false) {
  return new Promise((resolve, reject) => {
    if (options.length === 0) {
      resolve(multiSelect ? [] : defaultIndex);
      return;
    }

    let selectedIndex = defaultIndex >= 0 && defaultIndex < options.length ? defaultIndex : 0;
    let selectedIndices = multiSelect ? new Set() : null;
    let buffer = '';
    const screen = new ScreenManager();
    process.stdout.write(HIDE_CURSOR);

    function render() {
      let content = prompt + '\n';
      if (multiSelect) {
        content += '  (Space to select/deselect, Enter to confirm)\n';
      }

      options.forEach((option, index) => {
        const isSelected = multiSelect ? selectedIndices.has(index) : index === selectedIndex;
        const isHighlighted = index === selectedIndex;

        let prefix = '    ';
        if (isHighlighted) {
          prefix = `  ${ESC}[1m>${ESC}[0m `;
        }

        const checkbox = multiSelect ? (isSelected ? '[✓]' : '[ ]') : '';
        const style = isHighlighted ? ESC + '[1m' : '';
        const reset = isHighlighted ? ESC + '[0m' : '';

        content += `${prefix}${checkbox} ${style}${option}${reset}\n`;
      });

      screen.render(content);
    }

    process.stdin.setRawMode(true);
    process.stdin.resume();
    process.stdin.setEncoding('utf8');

    render();

    const onData = (data) => {
      buffer += data.toString();

      // Handle escape sequences
      if (buffer.startsWith(ESC)) {
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
          }
        }
        if (buffer.length > 10) {
          buffer = '';
        } else {
          return;
        }
      }

      // Process single character
      if (buffer.length > 0 && !buffer.startsWith(ESC)) {
        const char = buffer[0];
        buffer = '';

        switch (char) {
          case ' ':
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
            process.stdin.setRawMode(false);
            process.stdin.pause();
            process.stdin.removeListener('data', onData);
            screen.done();
            process.stdout.write(SHOW_CURSOR);
            if (multiSelect) {
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
            screen.done();
            process.stdout.write(SHOW_CURSOR);
            process.exit(1);
            break;
          case '\u0004': // Ctrl+D
            process.stdin.setRawMode(false);
            process.stdin.pause();
            process.stdin.removeListener('data', onData);
            screen.done();
            process.stdout.write(SHOW_CURSOR);
            process.exit(1);
            break;
        }
      }
    };

    process.stdin.on('data', onData);
  });
}

// =====================
// Inquirer-style Prompts
// =====================

/**
 * Confirm prompt (y/n) - improved version
 * @param {Object} options
 * @param {string} options.message - Prompt message
 * @param {boolean} options.default - Default value (default: false)
 * @returns {Promise<boolean>}
 */
async function confirm({ message, default: defaultVal = false }) {
  const rl = createReadlineInterface();
  const defaultHint = defaultVal ? 'Y/n' : 'y/N';
  const prompt = `  ${message} (${defaultHint}): `;

  return new Promise((resolve) => {
    const onLine = (line) => {
      const answer = line.trim().toLowerCase();
      
      if (answer === '' || answer === '\n' || answer === '\r') {
        rl.close();
        resolve(defaultVal);
        return;
      }

      if (/^(y|yes)$/i.test(answer)) {
        rl.close();
        resolve(true);
      } else if (/^(n|no)$/i.test(answer)) {
        rl.close();
        resolve(false);
      } else {
        // Invalid input, ask again
        process.stdout.write(CLEAR_LINE);
        rl.question(prompt, onLine);
      }
    };

    rl.question(prompt, onLine);
  });
}

/**
 * Input prompt (text)
 * @param {Object} options
 * @param {string} options.message - Prompt message
 * @param {string} options.default - Default value
 * @param {Function} options.validate - Validation function
 * @returns {Promise<string>}
 */
async function input({ message, default: defaultVal = '', validate = null }) {
  const rl = createReadlineInterface();

  while (true) {
    const prompt = defaultVal
      ? `  ${message} (default: ${defaultVal}): `
      : `  ${message}: `;
    const answer = await question(rl, prompt);
    const value = answer.trim() || defaultVal;

    if (validate) {
      const result = await validate(value);
      if (result === true) {
        rl.close();
        return value;
      }
      printError(result);
    } else {
      rl.close();
      return value;
    }
  }
}

/**
 * Password prompt (hidden input)
 * @param {Object} options
 * @param {string} options.message - Prompt message
 * @param {string} options.mask - Mask character (default: '●')
 * @param {Function} options.validate - Validation function
 * @returns {Promise<string>}
 */
async function password({ message, mask = '●', validate = null }) {
  const rl = createReadlineInterface();

  while (true) {
    const value = await getPassword(rl, `  ${message}: `);

    if (validate) {
      const result = await validate(value);
      if (result === true) {
        rl.close();
        return value.trim();
      }
      printError(result);
    } else if (value.trim()) {
      rl.close();
      return value.trim();
    } else {
      printError('Value cannot be empty. Please try again.');
    }
  }
}

/**
 * Select prompt (single choice)
 * @param {Object} options
 * @param {string} options.message - Prompt message
 * @param {Array} options.choices - Array of {name, value, description, disabled}
 * @param {number} options.default - Default index or value
 * @returns {Promise<any>}
 */
async function select({ message, choices = [], default: defaultVal = 0 }) {
  const choiceNames = choices.map(c => {
    let name = c.name;
    if (c.description) name += `\n    ${c.description}`;
    if (c.disabled) name += ` [${c.disabled}]`;
    return name;
  });

  let defaultIndex = defaultVal;
  if (typeof defaultVal !== 'number') {
    const foundIndex = choices.findIndex(c => c.value === defaultVal);
    defaultIndex = foundIndex >= 0 ? foundIndex : 0;
  }

  const index = await selectMenu(choiceNames, `  ${message}`, defaultIndex);

  return choices[index].value;
}

/**
 * Checkbox prompt (multiple choice)
 * @param {Object} options
 * @param {string} options.message - Prompt message
 * @param {Array} options.choices - Array of {name, value, checked, disabled}
 * @returns {Promise<Array>}
 */
async function checkbox({ message, choices = [] }) {
  const choiceNames = choices.map(c => c.name);

  // Find pre-checked indices
  const defaultIndices = choices
    .map((c, i) => c.checked ? i : -1)
    .filter(i => i >= 0);

  const selectedIndices = await selectMenu(
    choiceNames,
    `  ${message}`,
    defaultIndices[0] || 0,
    true
  );

  return selectedIndices.map(i => choices[i].value);
}

// =====================
// Config Persistence
// =====================

const INSTALLER_CONFIG_FILE = '.gitlab-mcp-installer.json';

function saveInstallerConfig(config) {
  try {
    const configToSave = JSON.parse(JSON.stringify(config));
    // Save tokens if they exist (user requested this)
    fs.writeFileSync(
      INSTALLER_CONFIG_FILE,
      JSON.stringify(configToSave, null, 2)
    );
  } catch (e) {
    printError(`Failed to save config: ${e.message}`);
  }
}

function loadInstallerConfig() {
  try {
    if (fs.existsSync(INSTALLER_CONFIG_FILE)) {
      const content = fs.readFileSync(INSTALLER_CONFIG_FILE, 'utf8');
      return JSON.parse(content);
    }
  } catch (e) {
    printError(`Failed to load config: ${e.message}`);
  }
  return null;
}

// =====================
// Environment Detection
// =====================

async function detectEnvironments() {
  const paths = getPlatformPaths();
  const detected = {
    vscode: false,
    claude_desktop: false,
    claude_code: false,
    cursor: false
  };

  // Detect VS Code (check binary or config)
  if (paths.vscode_user_settings) {
    const settingsDir = path.dirname(paths.vscode_user_settings);
    detected.vscode = fs.existsSync(settingsDir) ||
                     fs.existsSync(paths.vscode_user_settings);
  }

  // Detect Claude Desktop
  if (paths.claude_desktop) {
    const configDir = path.dirname(paths.claude_desktop);
    detected.claude_desktop = fs.existsSync(configDir) ||
                             fs.existsSync(paths.claude_desktop);
  }

  // Detect Claude Code (check .claude.json)
  if (paths.claude_code) {
    detected.claude_code = fs.existsSync(paths.claude_code);
  }

  // Detect Cursor (check .cursor/mcp.json or binary)
  if (paths.cursor) {
    const cursorDir = path.dirname(paths.cursor);
    detected.cursor = fs.existsSync(cursorDir) ||
                      fs.existsSync(paths.cursor);
  }

  return detected;
}

// =====================
// Read IDE Configurations
// =====================

function readIdeConfigs() {
  const paths = getPlatformPaths();
  const configs = {
    vscode: null,
    claude_desktop: null,
    claude_code: null,
    cursor: null
  };

  // Read VS Code config
  if (paths.vscode_user_settings && fs.existsSync(paths.vscode_user_settings)) {
    try {
      const content = fs.readFileSync(paths.vscode_user_settings, 'utf8');
      const data = JSON.parse(content);
      configs.vscode = data.mcp?.servers || {};
    } catch (e) {
      // Ignore parse errors
    }
  }

  // Try workspace config
  const workspaceConfig = path.join(process.cwd(), paths.vscode_workspace);
  if (fs.existsSync(workspaceConfig)) {
    try {
      const content = fs.readFileSync(workspaceConfig, 'utf8');
      const data = JSON.parse(content);
      if (data.servers) {
        configs.vscode = { ...configs.vscode, ...data.servers };
      }
    } catch (e) {
      // Ignore parse errors
    }
  }

  // Read Claude Desktop config
  if (paths.claude_desktop && fs.existsSync(paths.claude_desktop)) {
    try {
      const content = fs.readFileSync(paths.claude_desktop, 'utf8');
      const data = JSON.parse(content);
      configs.claude_desktop = data.mcpServers || {};
    } catch (e) {
      // Ignore parse errors
    }
  }

  // Read Claude Code config
  if (paths.claude_code && fs.existsSync(paths.claude_code)) {
    try {
      const content = fs.readFileSync(paths.claude_code, 'utf8');
      const data = JSON.parse(content);
      configs.claude_code = data.mcpServers || {};
    } catch (e) {
      // Ignore parse errors
    }
  }

  // Read Cursor config
  if (paths.cursor && fs.existsSync(paths.cursor)) {
    try {
      const content = fs.readFileSync(paths.cursor, 'utf8');
      const data = JSON.parse(content);
      configs.cursor = data.mcpServers || {};
    } catch (e) {
      // Ignore parse errors
    }
  }

  return configs;
}

// =====================
// Detect GitLab MCP Servers
// =====================

function detectGitlabMcpServers(config, ideType, projectRoot) {
  if (!config || typeof config !== 'object') {
    return [];
  }

  const detected = [];
  const binaryPath = path.join(projectRoot, 'bin', 'gitlab-mcp-server');
  const isWindows = os.platform() === 'win32';
  const fullBinaryPath = isWindows ? `${binaryPath}.exe` : binaryPath;
  const absBinaryPath = fs.existsSync(fullBinaryPath) ? path.resolve(fullBinaryPath) : null;

  for (const [serverName, serverConfig] of Object.entries(config)) {
    if (!serverConfig || typeof serverConfig !== 'object') {
      continue;
    }

    let isGitlabServer = false;

    // Check 1: Command contains gitlab-mcp-server or path to binary
    const command = serverConfig.command || '';
    const args = serverConfig.args || [];
    const fullCommand = [command, ...args].join(' ').toLowerCase();

    if (fullCommand.includes('gitlab-mcp-server') ||
        (absBinaryPath && fullCommand.includes(absBinaryPath.toLowerCase())) ||
        (absBinaryPath && fullCommand.includes(path.basename(absBinaryPath).toLowerCase()))) {
      isGitlabServer = true;
    }

    // Check 2: Server name contains "gitlab" (case-insensitive)
    if (!isGitlabServer && /gitlab/i.test(serverName)) {
      isGitlabServer = true;
    }

    // Check 3: Environment variables contain GITLAB_TOKEN or GITLAB_HOST
    if (!isGitlabServer && serverConfig.env) {
      const envKeys = Object.keys(serverConfig.env);
      if (envKeys.includes('GITLAB_TOKEN') || envKeys.includes('GITLAB_HOST')) {
        isGitlabServer = true;
      }
    }

    if (isGitlabServer) {
      detected.push({
        name: serverName,
        config: serverConfig,
        ideType: ideType
      });
    }
  }

  return detected;
}

function getAllGitlabServers(projectRoot) {
  const ideConfigs = readIdeConfigs();
  const allServers = [];

  for (const [ideType, config] of Object.entries(ideConfigs)) {
    if (config) {
      const servers = detectGitlabMcpServers(config, ideType, projectRoot);
      allServers.push(...servers);
    }
  }

  return allServers;
}

function detectDeploymentMode(servers, projectRoot) {
  if (servers.length === 0) {
    return null;
  }

  // Check first server's command to determine mode
  const firstServer = servers[0];
  const command = firstServer.config?.command || '';
  const args = firstServer.config?.args || [];
  const fullCommand = [command, ...args].join(' ').toLowerCase();

  if (fullCommand.includes('docker') || fullCommand.includes('docker run')) {
    return 'docker';
  }

  // Check if it's a local binary path
  const binaryPath = path.join(projectRoot, 'bin', 'gitlab-mcp-server');
  const isWindows = os.platform() === 'win32';
  const fullBinaryPath = isWindows ? `${binaryPath}.exe` : binaryPath;
  const absBinaryPath = fs.existsSync(fullBinaryPath) ? path.resolve(fullBinaryPath) : null;

  if (absBinaryPath && (fullCommand.includes(absBinaryPath.toLowerCase()) || 
      fullCommand.includes(path.basename(absBinaryPath).toLowerCase()))) {
    return 'local';
  }

  // Default to local if command is a path
  if (command && (command.startsWith('/') || command.includes('\\') || command.includes('bin'))) {
    return 'local';
  }

  return null;
}

function validateServerConfig(server, deploymentMode, projectRoot) {
  const errors = [];

  // Check if command is valid
  if (!server.config || !server.config.command) {
    errors.push('Missing command');
  } else {
    const command = server.config.command;
    const args = server.config.args || [];
    const fullCommand = [command, ...args].join(' ');

    if (deploymentMode === 'local') {
      // Check if binary path exists
      const binaryPath = path.join(projectRoot, 'bin', 'gitlab-mcp-server');
      const isWindows = os.platform() === 'win32';
      const fullBinaryPath = isWindows ? `${binaryPath}.exe` : binaryPath;
      const absBinaryPath = fs.existsSync(fullBinaryPath) ? path.resolve(fullBinaryPath) : null;

      if (absBinaryPath && !fullCommand.includes(absBinaryPath) && !fullCommand.includes(path.basename(absBinaryPath))) {
        errors.push('Command path does not match expected binary');
      }
    } else if (deploymentMode === 'docker') {
      if (!fullCommand.includes('docker') && !fullCommand.includes('gitlab-mcp-server')) {
        errors.push('Command does not appear to be Docker-based');
      }
    }
  }

  // Check environment variables
  const env = server.config?.env || {};
  if (!env.GITLAB_TOKEN && !env.GITLAB_HOST) {
    errors.push('Missing GITLAB_TOKEN or GITLAB_HOST');
  }

  return {
    valid: errors.length === 0,
    errors: errors
  };
}

function mergeServersFromIde(servers, projectRoot) {
  // Group servers by name and merge their properties
  const serverMap = new Map();

  for (const server of servers) {
    const name = server.name;
    const config = server.config;

    if (!serverMap.has(name)) {
      // Extract server info from config
      const env = config.env || {};
      const gitlabHost = env.GITLAB_HOST || '';
      const readonly = env.GITLAB_READ_ONLY === 'true';

      serverMap.set(name, {
        name: name,
        gitlab_host: gitlabHost,
        token: '', // Tokens are not stored in IDE configs
        readonly: readonly,
        ideTypes: [server.ideType],
        config: config, // Keep original config for validation
        ideType: server.ideType // Keep for reference
      });
    } else {
      // Merge IDE types
      const existing = serverMap.get(name);
      if (!existing.ideTypes.includes(server.ideType)) {
        existing.ideTypes.push(server.ideType);
      }
    }
  }

  return Array.from(serverMap.values());
}

// =====================
// Server Management
// =====================

async function listServers(servers) {
  if (servers.length === 0) {
    printInfo('No servers configured');
    return;
  }

  console.log('');
  console.log('Configured servers:');
  servers.forEach((server, index) => {
    const readonlyStr = server.readonly ? ' [read-only]' : '';
    console.log(`  ${index + 1}. ${server.name}: ${server.gitlab_host || 'default'}${readonlyStr}`);
  });
  console.log('');
}

async function editServer(server, deploymentMode, projectRoot) {
  printSection(`Editing Server: ${server.name}`);

  // Show validation errors if any
  if (server.config) {
    const validation = validateServerConfig(server, deploymentMode, projectRoot);
    if (!validation.valid) {
      printError(`Configuration errors: ${validation.errors.join(', ')}`);
      console.log('');
    }
  }

  const newName = await input({
    message: 'Server name',
    default: server.name,
    validate: (val) => {
      if (!val.trim()) return 'Server name cannot be empty';
      return true;
    }
  });

  const newHost = await input({
    message: 'GitLab host URL',
    default: server.gitlab_host || 'https://gitlab.com',
    validate: (val) => val.trim() ? true : 'Host cannot be empty'
  });

  const updateToken = await confirm({
    message: server.token && server.token.trim() ? 'Update access token?' : 'Set access token?',
    default: !server.token || !server.token.trim()
  });

  let newToken = server.token;
  if (updateToken) {
    newToken = await password({
      message: 'GitLab access token',
      validate: (val) => val.trim() ? true : 'Token cannot be empty'
    });
  }

  const newReadonly = await confirm({
    message: 'Enable read-only mode?',
    default: server.readonly || false
  });

  return {
    name: newName.trim(),
    gitlab_host: newHost.trim(),
    token: newToken,
    readonly: newReadonly,
    config: server.config || null,
    ideTypes: server.ideTypes || []
  };
}

async function deleteServer(serverName, servers) {
  const confirmed = await confirm({
    message: `Are you sure you want to delete server '${serverName}'?`,
    default: false
  });

  if (confirmed) {
    return servers.filter(s => s.name !== serverName);
  }
  return servers;
}

async function manageServers(servers, deploymentMode, projectRoot) {
  while (true) {
    printSection('Server Management');
    
    // Show current servers with validation
    if (servers.length > 0) {
      console.log('Current servers:');
      servers.forEach((server, index) => {
        const readonlyStr = server.readonly ? ' [read-only]' : '';
        const hostStr = server.gitlab_host ? ` (${server.gitlab_host})` : '';
        const tokenStr = server.token && server.token.trim() ? '' : ' [no token]';
        
        // Validate server if it has config from IDE
        let errorStr = '';
        if (server.config) {
          const validation = validateServerConfig(server, deploymentMode, projectRoot);
          if (!validation.valid) {
            errorStr = ` ✗ Invalid: ${validation.errors.join(', ')}`;
          }
        }
        
        const statusStr = errorStr || (tokenStr && ' ⚠ Missing token');
        console.log(`  ${index + 1}. ${server.name}${hostStr}${readonlyStr}${statusStr || ''}`);
        
        if (server.ideTypes && server.ideTypes.length > 0) {
          console.log(`      Found in: ${server.ideTypes.join(', ')}`);
        }
      });
      console.log('');
    } else {
      printInfo('No servers configured');
      console.log('');
    }
    
    const choices = [
      { name: '[A]dd server', value: 'add' },
      { name: '[E]dit server', value: 'edit' },
      { name: '[D]elete server', value: 'delete' },
      { name: '[C]lear all servers', value: 'clear' },
      { name: '[S]ynchronize with IDE', value: 'sync' },
      { name: '[Q]uit', value: 'quit' }
    ];

    const action = await select({
      message: 'What would you like to do?',
      choices: choices
    });

    switch (action) {
      case 'add':
        printSection('Add New Server');
        const serverName = await input({
          message: 'Server name (e.g., "work", "personal")',
          validate: (val) => {
            if (!val.trim()) return 'Server name cannot be empty';
            if (servers.some(s => s.name === val.trim())) {
              return `Server '${val}' already exists`;
            }
            return true;
          }
        });

        const gitlab_host = await input({
          message: 'GitLab host URL',
          default: 'https://gitlab.com',
          validate: (val) => val.trim() ? true : 'Host cannot be empty'
        });

        const token = await password({
          message: 'GitLab access token',
          validate: (val) => val.trim() ? true : 'Token cannot be empty'
        });

        const readonly = await confirm({
          message: 'Enable read-only mode?',
          default: false
        });

        servers.push({ 
          name: serverName.trim(), 
          gitlab_host: gitlab_host.trim(), 
          token, 
          readonly,
          config: null, // New server, no IDE config
          ideTypes: []
        });
        printSuccess(`Server '${serverName}' added`);
        console.log('');
        break;

      case 'edit':
        if (servers.length === 0) {
          printError('No servers to edit');
          console.log('');
          break;
        }

        const editChoices = servers.map((s, i) => {
          let name = `${s.name}${s.gitlab_host ? ` (${s.gitlab_host})` : ''}`;
          if (s.config) {
            const validation = validateServerConfig(s, deploymentMode, projectRoot);
            if (!validation.valid) {
              name += ` ✗ Invalid`;
            }
          }
          if (!s.token || !s.token.trim()) {
            name += ` ⚠ No token`;
          }
          return { name, value: i };
        });

        const editIndex = await select({
          message: 'Select server to edit',
          choices: editChoices
        });

        const edited = await editServer(servers[editIndex], deploymentMode, projectRoot);
        // Preserve config and ideTypes if they exist
        servers[editIndex] = {
          ...edited,
          config: servers[editIndex].config || null,
          ideTypes: servers[editIndex].ideTypes || []
        };
        printSuccess(`Server '${edited.name}' updated`);
        console.log('');
        break;

      case 'delete':
        if (servers.length === 0) {
          printError('No servers to delete');
          console.log('');
          break;
        }

        const deleteChoices = servers.map((s, i) => ({
          name: `${s.name}${s.gitlab_host ? ` (${s.gitlab_host})` : ''}`,
          value: i
        }));

        const deleteIndex = await select({
          message: 'Select server to delete',
          choices: deleteChoices
        });

        const deletedName = servers[deleteIndex].name;
        servers = await deleteServer(deletedName, servers);
        printSuccess(`Server '${deletedName}' deleted`);
        console.log('');
        break;

      case 'clear':
        if (servers.length === 0) {
          printError('No servers to clear');
          console.log('');
          break;
        }

        const confirmed = await confirm({
          message: `Are you sure you want to delete all ${servers.length} server(s)?`,
          default: false
        });

        if (confirmed) {
          servers = [];
          printSuccess('All servers cleared');
          console.log('');
        }
        break;

      case 'sync':
        if (servers.length === 0) {
          printError('No servers to synchronize');
          console.log('');
          break;
        }

        // Check for servers without tokens
        const serversWithoutTokens = servers.filter(s => !s.token || !s.token.trim());
        if (serversWithoutTokens.length > 0) {
          printError(`Cannot synchronize: ${serversWithoutTokens.length} server(s) missing tokens`);
          console.log('  Servers without tokens:');
          serversWithoutTokens.forEach(s => {
            console.log(`    • ${s.name}`);
          });
          console.log('');
          const addTokens = await confirm({
            message: 'Would you like to add tokens for these servers now?',
            default: true
          });
          
          if (addTokens) {
            for (const server of serversWithoutTokens) {
              printInfo(`Enter token for server: ${server.name}`);
              server.token = await password({
                message: `  GitLab access token for '${server.name}'`,
                validate: (val) => val.trim() ? true : 'Token cannot be empty'
              });
              printSuccess(`Token received for '${server.name}'`);
              console.log('');
            }
          } else {
            break;
          }
        }

        // Environment selection
        printSection('Select Environments to Sync');
        const detected = await detectEnvironments();

        const envChoices = [
          { name: 'VS Code', value: 'vscode', checked: detected.vscode },
          { name: 'Claude Desktop', value: 'claude_desktop', checked: detected.claude_desktop },
          { name: 'Claude Code', value: 'claude_code', checked: detected.claude_code },
          { name: 'Cursor', value: 'cursor', checked: detected.cursor }
        ];

        const selectedEnvs = await checkbox({
          message: 'Select environments to synchronize (Space to toggle, Enter to confirm):',
          choices: envChoices
        });

        if (selectedEnvs.length > 0) {
          syncToIde(servers, selectedEnvs, deploymentMode, projectRoot);
        } else {
          printError('No environments selected');
          console.log('');
        }
        break;

      case 'quit':
        return servers;
    }
  }
}

// =====================
// Server Collection Functions
// =====================

async function collectSingleServer(deploymentMode) {
  const server = {
    name: 'gitlab'
  };

  server.gitlab_host = await input({
    message: 'GitLab host URL',
    default: 'https://gitlab.com',
    validate: (val) => val.trim() ? true : 'Host cannot be empty'
  });

  server.token = await password({
    message: 'GitLab access token',
    validate: (val) => val.trim() ? true : 'Token cannot be empty'
  });

  server.readonly = await confirm({
    message: 'Enable read-only mode?',
    default: false
  });

  return [server];
}

async function collectServersNew(deploymentMode) {
  const servers = [];
  let serverNum = 0;

  printSection('Multi-Server Configuration');
  printInfo('You can configure multiple GitLab instances');
  console.log('');

  while (true) {
    serverNum++;
    printSection(`Server ${serverNum} Configuration`);

    const serverName = await input({
      message: `Server name (e.g., 'work', 'personal')`,
      validate: (val) => {
        if (!val.trim()) return 'Server name cannot be empty';
        if (servers.some(s => s.name === val.trim())) {
          return `Server '${val}' already configured`;
        }
        return true;
      }
    });

    const gitlab_host = await input({
      message: `GitLab host URL for '${serverName}'`,
      default: 'https://gitlab.com',
      validate: (val) => val.trim() ? true : 'Host cannot be empty'
    });

    const token = await password({
      message: `GitLab access token for '${serverName}'`,
      validate: (val) => val.trim() ? true : 'Token cannot be empty'
    });

    const readonly = await confirm({
      message: `Enable read-only mode for '${serverName}'?`,
      default: false
    });

    servers.push({ name: serverName.trim(), gitlab_host: gitlab_host.trim(), token, readonly });
    printSuccess(`Server '${serverName}' configured`);
    console.log('');

    const more = await confirm({
      message: 'Add another server?',
      default: false
    });

    if (!more) break;
    console.log('');
  }

  return servers;
}

// =====================
// Configuration Helpers
// =====================

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

function cleanGitlabServers(config, ideType, projectRoot) {
  if (!config || typeof config !== 'object') {
    return config;
  }

  const cleaned = {};
  const gitlabServers = detectGitlabMcpServers(config, ideType, projectRoot);
  const gitlabServerNames = new Set(gitlabServers.map(s => s.name));

  for (const [serverName, serverConfig] of Object.entries(config)) {
    if (!gitlabServerNames.has(serverName)) {
      cleaned[serverName] = serverConfig;
    }
  }

  return cleaned;
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
      const workspacePath = path.join(process.cwd(), paths.vscode_workspace);
      if (updateVscodeWorkspace(workspacePath, serverName, serverConfig)) {
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

function syncToIde(servers, selectedEnvs, deploymentMode, projectRoot) {
  const paths = getPlatformPaths();
  const ideConfigs = readIdeConfigs();

  printSection('Synchronizing with IDE Configurations');
  printInfo(`Syncing ${servers.length} server(s) to ${selectedEnvs.length} environment(s)...`);
  console.log('');

  for (const env of selectedEnvs) {
    const envNameMap = {
      vscode: 'VS Code',
      claude_desktop: 'Claude Desktop',
      claude_code: 'Claude Code',
      cursor: 'Cursor'
    };
    const envName = envNameMap[env] || env;

    try {
      // Get binary config
      const binaryConfig = getBinaryConfig(deploymentMode, projectRoot);

      // Clean existing GitLab servers from config
      let existingConfig = {};
      let configPath = null;

      if (env === 'vscode') {
        configPath = paths.vscode_user_settings;
        if (configPath && fs.existsSync(configPath)) {
          try {
            const content = fs.readFileSync(configPath, 'utf8');
            const data = JSON.parse(content);
            existingConfig = data.mcp?.servers || {};
          } catch (e) {
            // Ignore
          }
        }
        existingConfig = cleanGitlabServers(existingConfig, env, projectRoot);
      } else if (env === 'claude_desktop') {
        configPath = paths.claude_desktop;
        if (configPath && fs.existsSync(configPath)) {
          try {
            const content = fs.readFileSync(configPath, 'utf8');
            const data = JSON.parse(content);
            existingConfig = data.mcpServers || {};
          } catch (e) {
            // Ignore
          }
        }
        existingConfig = cleanGitlabServers(existingConfig, env, projectRoot);
      } else if (env === 'claude_code') {
        configPath = paths.claude_code;
        if (configPath && fs.existsSync(configPath)) {
          try {
            const content = fs.readFileSync(configPath, 'utf8');
            const data = JSON.parse(content);
            existingConfig = data.mcpServers || {};
          } catch (e) {
            // Ignore
          }
        }
        existingConfig = cleanGitlabServers(existingConfig, env, projectRoot);
      } else if (env === 'cursor') {
        configPath = paths.cursor;
        if (configPath && fs.existsSync(configPath)) {
          try {
            const content = fs.readFileSync(configPath, 'utf8');
            const data = JSON.parse(content);
            existingConfig = data.mcpServers || {};
          } catch (e) {
            // Ignore
          }
        }
        existingConfig = cleanGitlabServers(existingConfig, env, projectRoot);
      }

      // Add new servers (skip invalid ones)
      for (const serverConfigData of servers) {
        const serverName = serverConfigData.name;
        
        // Skip servers without tokens
        if (!serverConfigData.token || !serverConfigData.token.trim()) {
          printError(`Skipping '${serverName}': missing token`);
          continue;
        }

        // Validate server config if it came from IDE
        if (serverConfigData.config) {
          const validation = validateServerConfig(serverConfigData, deploymentMode, projectRoot);
          if (!validation.valid) {
            printError(`Skipping '${serverName}': ${validation.errors.join(', ')}`);
            continue;
          }
        }
        
        // Add read-only environment variable if needed
        if (serverConfigData.readonly) {
          if (deploymentMode === 'docker') {
            binaryConfig.args.splice(-1, 0, '-e', 'GITLAB_READ_ONLY');
          }
          binaryConfig.env.GITLAB_READ_ONLY = 'true';
        }

        const serverConfig = createServerConfig(binaryConfig, serverConfigData);
        existingConfig[serverName] = serverConfig;
      }

      // Write updated config
      if (env === 'vscode') {
        if (configPath) {
          updateJsonFile(
            configPath,
            (data) => {
              if (!data.mcp) {
                data.mcp = {};
              }
              data.mcp.servers = existingConfig;
            },
            `${envName} config`
          );
        }
      } else {
        if (configPath) {
          updateJsonFile(
            configPath,
            (data) => {
              data.mcpServers = existingConfig;
              if (env === 'claude_code') {
                // Ensure type field for Claude Code
                for (const [name, config] of Object.entries(data.mcpServers)) {
                  if (servers.some(s => s.name === name)) {
                    config.type = 'stdio';
                  }
                }
              }
            },
            `${envName} config`
          );
        }
      }

      printSuccess(`${envName} synchronized`);
    } catch (e) {
      printError(`Failed to sync ${envName}: ${e.message}`);
    }
  }

  console.log('');
}

function updateEnvironment(env, paths, serverName, serverConfig) {
  let success = false;
  let error = null;

  try {
    if (env === 'vscode') {
      success = updateVscode(paths, serverName, serverConfig);
    } else if (env === 'claude_desktop') {
      if (paths.claude_desktop) {
        success = updateClaudeConfig(paths.claude_desktop, serverName, serverConfig, false);
      } else {
        error = 'Path not available';
      }
    } else if (env === 'claude_code') {
      if (paths.claude_code) {
        success = updateClaudeConfig(paths.claude_code, serverName, serverConfig, true);
      } else {
        error = 'Path not available';
      }
    } else if (env === 'cursor') {
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
    const envNameMap = {
      vscode: 'VS Code',
      claude_desktop: 'Claude Desktop',
      claude_code: 'Claude Code',
      cursor: 'Cursor'
    };
    printSuccess(`${envNameMap[env] || env} configured successfully`);
    return true;
  } else {
    printError(`Failed to configure ${env}${error ? `: ${error}` : ''}`);
    return false;
  }
}

// =====================
// Main Installation Flow
// =====================

async function main() {
  printHeader();

  try {
    const projectRoot = getProjectRoot();
    const paths = getPlatformPaths();

    // ========================================
    // Step 1: Detect existing configurations
    // ========================================
    printSection('Detecting Existing Configurations');
    
    const detectedEnvs = await detectEnvironments();
    const ideConfigs = readIdeConfigs();
    const existingGitlabServers = getAllGitlabServers(projectRoot);

    printInfo(`Detected environments: ${Object.entries(detectedEnvs).filter(([_, v]) => v).map(([k]) => k).join(', ') || 'none'}`);
    
    // ========================================
    // Step 2: Determine deployment mode
    // ========================================
    let deploymentMode = detectDeploymentMode(existingGitlabServers, projectRoot);
    
    if (!deploymentMode) {
      // No existing configs, ask user
      printSection('Deployment Mode');
      deploymentMode = await select({
        message: 'How will you run the MCP server?',
        choices: [
          {
            name: 'Local binary',
            value: 'local',
            description: 'Run compiled binary directly (requires make build)'
          },
          {
            name: 'Docker',
            value: 'docker',
            description: 'Run in Docker container (requires docker build)'
          }
        ],
        default: 'local'
      });
    } else {
      const modeStr = deploymentMode === 'docker' ? 'Docker' : 'Local binary';
      printInfo(`Deployment mode detected: ${modeStr}`);
    }
    console.log('');

    // ========================================
    // Step 3: Merge servers from IDE and saved config
    // ========================================
    let servers = [];
    
    // Merge servers from IDE configurations
    if (existingGitlabServers.length > 0) {
      printInfo(`Found ${existingGitlabServers.length} existing GitLab MCP server(s) in IDE configurations`);
      servers = mergeServersFromIde(existingGitlabServers, projectRoot);
    }

    // Load saved config and merge
    const savedConfig = loadInstallerConfig();
    if (savedConfig && savedConfig.servers && savedConfig.servers.length > 0) {
      // Merge saved servers (prefer saved tokens if available)
      for (const savedServer of savedConfig.servers) {
        const existingIndex = servers.findIndex(s => s.name === savedServer.name);
        if (existingIndex >= 0) {
          // Update existing server with saved data (including token if available)
          servers[existingIndex] = {
            ...servers[existingIndex],
            ...savedServer,
            // Keep original config from IDE if exists
            config: servers[existingIndex].config || null,
            ideTypes: servers[existingIndex].ideTypes || [],
            // Keep token from saved config if available
            token: savedServer.token || servers[existingIndex].token || ''
          };
        } else {
          // Add new server from saved config
          servers.push(savedServer);
        }
      }
    }

    // ========================================
    // Step 4: Server management menu
    // ========================================
    servers = await manageServers(servers, deploymentMode, projectRoot);

    // ========================================
    // Step 5: Save config
    // ========================================
    const configToSave = {
      deploymentMode,
      servers: servers,
      lastUpdated: new Date().toISOString()
    };

    saveInstallerConfig(configToSave);
    printSuccess('Configuration saved to .gitlab-mcp-installer.json');
    console.log('');

    // ========================================
    // Step 6: Summary
    // ========================================
    printSection('Configuration Summary');
    console.log(`  Deployment mode: ${deploymentMode === 'docker' ? 'Docker' : 'Local binary'}`);
    console.log(`  Servers configured: ${servers.length}`);
    if (servers.length > 0) {
      servers.forEach((server) => {
        const readonlyStr = server.readonly ? ' [read-only]' : '';
        const hostStr = server.gitlab_host ? ` (${server.gitlab_host})` : '';
        console.log(`    • ${server.name}${hostStr}${readonlyStr}`);
      });
    }
    console.log('');

    printSection('Next Steps');
    console.log('1. Use "Synchronize with IDE" from the management menu to apply configurations');
    console.log('2. Restart your development environment(s) to load the new configuration');
    if (deploymentMode === 'local') {
      console.log('3. Ensure the binary exists at the specified path');
    } else {
      console.log('3. Build the Docker image: docker build -t gitlab-mcp-server:latest .');
    }
    console.log('');
    printSeparator();
    console.log('');

  } catch (error) {
    if (error.name === 'ExitPromptError') {
      console.log('\nInstallation cancelled.');
    } else {
      console.error('\nUnexpected error:', error.message);
      console.error(error.stack);
    }
    process.exit(1);
  }
}

if (require.main === module) {
  main().catch((e) => {
    console.error('Unexpected error:', e);
    process.exit(1);
  });
}
