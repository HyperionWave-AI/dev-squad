#!/usr/bin/env node

/**
 * Post-install script to automatically configure Claude Code with the MCP server
 * Runs after: npm install -g @hyperion/coordinator-mcp
 */

const fs = require('fs');
const path = require('path');
const os = require('os');

console.log('Configuring Claude Code with Hyperion Coordinator MCP...');

// Determine Claude Code config directory based on platform
const platform = os.platform();
let configDir;

if (platform === 'darwin') {
  // macOS
  configDir = path.join(os.homedir(), 'Library', 'Application Support', 'Claude');
} else if (platform === 'win32') {
  // Windows
  configDir = path.join(process.env.APPDATA || '', 'Claude');
} else {
  // Linux
  configDir = path.join(os.homedir(), '.config', 'Claude');
}

const configFile = path.join(configDir, 'claude_desktop_config.json');

// Get the binary path (will be in global npm bin directory or local bin)
const { execSync } = require('child_process');
let binaryPath;

try {
  // Try to get npm global bin directory
  const npmBin = execSync('npm bin -g', { encoding: 'utf-8' }).trim();
  const binaryName = platform === 'win32'
    ? 'hyperion-coordinator-mcp.exe'
    : 'hyperion-coordinator-mcp';
  binaryPath = path.join(npmBin, binaryName);
} catch (error) {
  // Fallback: use the bin directory in the package itself
  const binaryName = platform === 'win32'
    ? 'hyperion-coordinator-mcp.exe'
    : 'hyperion-coordinator-mcp';
  binaryPath = path.join(__dirname, '..', 'bin', binaryName);
  console.log('⚠️  Using local binary path:', binaryPath);
}

// Ensure config directory exists
if (!fs.existsSync(configDir)) {
  console.log('Creating Claude config directory:', configDir);
  fs.mkdirSync(configDir, { recursive: true });
}

// Read existing config or create new one
let config = { mcpServers: {} };

if (fs.existsSync(configFile)) {
  try {
    const existingConfig = fs.readFileSync(configFile, 'utf-8');
    config = JSON.parse(existingConfig);

    if (!config.mcpServers) {
      config.mcpServers = {};
    }
  } catch (error) {
    console.warn('⚠️  Warning: Failed to parse existing config, creating new one');
  }
}

// Add or update hyperion-coordinator server
config.mcpServers['hyperion-coordinator'] = {
  command: binaryPath,
  env: {
    MONGODB_URI: process.env.MONGODB_URI || 'mongodb+srv://dev:fvOKzv9enD8CSVwD@devdb.yqf8f8r.mongodb.net/?retryWrites=true&w=majority&appName=devDB',
    MONGODB_DATABASE: process.env.MONGODB_DATABASE || 'coordinator_db'
  }
};

// Write updated config
try {
  fs.writeFileSync(configFile, JSON.stringify(config, null, 2));
  console.log('✅ Successfully configured Claude Code');
  console.log('   Config file:', configFile);
  console.log('   Binary path:', binaryPath);
  console.log('');
  console.log('🎉 Installation complete!');
  console.log('');
  console.log('Next steps:');
  console.log('1. Restart Claude Code to load the MCP server');
  console.log('2. Verify it appears in Claude Code\'s MCP servers list');
  console.log('3. Test with: mcp__hyperion-coordinator__coordinator_list_human_tasks({})');
  console.log('');
  console.log('For more information, see: https://github.com/yourorg/hyperion-coordinator-mcp');
} catch (error) {
  console.error('❌ Failed to write config file:', error.message);
  console.error('');
  console.error('Manual configuration required:');
  console.error('Add this to', configFile + ':');
  console.error('');
  console.error(JSON.stringify({
    mcpServers: {
      'hyperion-coordinator': {
        command: binaryPath
      }
    }
  }, null, 2));
  process.exit(1);
}
