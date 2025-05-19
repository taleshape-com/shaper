#!/usr/bin/env node

const path = require('path');
const { spawn } = require('child_process');

const binaryPath = path.join(__dirname, 'bin', 'shaper');

// Forward all arguments to the binary
const args = process.argv.slice(2);

// Spawn the binary with the forwarded arguments
const child = spawn(binaryPath, args, {
  stdio: 'inherit',
  shell: false
});

// Forward the exit code
child.on('exit', (code) => {
  process.exit(code);
}); 