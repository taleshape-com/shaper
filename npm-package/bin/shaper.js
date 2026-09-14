#!/usr/bin/env node

// SPDX-License-Identifier: MPL-2.0

const { spawnSync } = require('child_process');
const { getBinaryPath } = require('./get-binary');

try {
  const binaryPath = getBinaryPath();
  const result = spawnSync(binaryPath, process.argv.slice(2), {
    stdio: 'inherit',
    windowsHide: true,
  });

  if (result.error) {
    console.error(`Failed to execute shaper binary: ${result.error.message}`);
    process.exit(1);
  }

  if (result.signal) {
    process.kill(process.pid, result.signal);
  } else {
    process.exit(result.status ?? 0);
  }
} catch (err) {
  console.error(`Error: ${err.message}`);
  process.exit(1);
}
