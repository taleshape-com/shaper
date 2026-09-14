// SPDX-License-Identifier: MPL-2.0

const path = require('path');
const fs = require('fs');

const PLATFORMS = {
  'darwin-arm64': '@taleshape/shaper-darwin-arm64',
  'darwin-x64': '@taleshape/shaper-darwin-x64',
  'linux-arm64': '@taleshape/shaper-linux-arm64',
  'linux-x64': '@taleshape/shaper-linux-x64',
};

function getBinaryPath() {
  const platformKey = `${process.platform}-${process.arch}`;
  const packageName = PLATFORMS[platformKey];

  if (!packageName) {
    throw new Error(
      `Unsupported platform or architecture: ${process.platform}-${process.arch}. ` +
      `Supported platforms are: ${Object.keys(PLATFORMS).join(', ')}.`
    );
  }

  let packageJsonPath;
  try {
    packageJsonPath = require.resolve(`${packageName}/package.json`);
  } catch (err) {
    throw new Error(
      `Failed to locate binary package "${packageName}" for platform ${platformKey}.\n` +
      `This usually happens if optional dependencies were skipped during installation.\n` +
      `Try reinstalling with optional dependencies enabled.\n` +
      `Original error: ${err.message}`
    );
  }

  const binaryName = process.platform === 'win32' ? 'shaper.exe' : 'shaper';
  const binaryPath = path.join(path.dirname(packageJsonPath), 'bin', binaryName);

  if (!fs.existsSync(binaryPath)) {
    throw new Error(`Binary not found at expected path: ${binaryPath}`);
  }

  if (process.platform !== 'win32') {
    try {
      const stat = fs.statSync(binaryPath);
      if ((stat.mode & 0o111) === 0) {
        fs.chmodSync(binaryPath, stat.mode | 0o755);
      }
    } catch {
      // Ignore if chmod fails (e.g. read-only filesystem)
    }
  }

  return binaryPath;
}

module.exports = {
  getBinaryPath,
  PLATFORMS,
};
