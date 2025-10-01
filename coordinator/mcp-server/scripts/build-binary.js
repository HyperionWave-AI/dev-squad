#!/usr/bin/env node

/**
 * Pre-install script to build the Go binary for the current platform
 * Runs automatically when user does: npm install -g @hyperion/coordinator-mcp
 */

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');
const os = require('os');

const platform = os.platform();
const arch = os.arch();

console.log(`Building Hyperion Coordinator MCP for ${platform}-${arch}...`);

// Check if Go is installed
try {
  execSync('go version', { stdio: 'pipe' });
} catch (error) {
  console.error('❌ Error: Go is not installed.');
  console.error('Please install Go 1.25+ from https://go.dev/dl/');
  process.exit(1);
}

// Create bin directory
const binDir = path.join(__dirname, '..', 'bin');
if (!fs.existsSync(binDir)) {
  fs.mkdirSync(binDir, { recursive: true });
}

// Determine binary name based on platform
const binaryName = platform === 'win32'
  ? 'hyperion-coordinator-mcp.exe'
  : 'hyperion-coordinator-mcp';

const binaryPath = path.join(binDir, binaryName);

try {
  // Build the Go binary
  console.log('Running: go build -o ' + binaryPath);

  execSync(`go build -o "${binaryPath}" main.go`, {
    cwd: path.join(__dirname, '..'),
    stdio: 'inherit'
  });

  // Make executable on Unix systems
  if (platform !== 'win32') {
    fs.chmodSync(binaryPath, 0o755);
  }

  console.log('✅ Binary built successfully: ' + binaryPath);
} catch (error) {
  console.error('❌ Failed to build binary:', error.message);
  process.exit(1);
}
