// SPDX-License-Identifier: MPL-2.0

const fs = require('fs');
const path = require('path');

const BIN_DIR = path.join(__dirname, 'bin');
const BINARY_PATH = path.join(BIN_DIR, 'shaper');

try {
  if (fs.existsSync(BINARY_PATH)) {
    fs.unlinkSync(BINARY_PATH);
  }
  if (fs.existsSync(BIN_DIR)) {
    fs.rmdirSync(BIN_DIR);
  }
} catch (error) {
  console.error('Error during uninstall:', error);
  process.exit(1);
}