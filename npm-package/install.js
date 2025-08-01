// SPDX-License-Identifier: MPL-2.0

const fs = require('fs');
const path = require('path');
const os = require('os');
const crypto = require('crypto');
const axios = require('axios');

const BIN_DIR = path.join(__dirname, 'bin');
const VERSION = require('./package.json').version;

// Map of platform names to GitHub release asset names
const PLATFORM_MAP = {
  'linux-x64': 'shaper-linux-amd64',
  'linux-arm64': 'shaper-linux-arm64',
  'darwin-x64': 'shaper-darwin-amd64',
  'darwin-arm64': 'shaper-darwin-arm64',
  // 'win32-x64': 'shaper_Windows_x86_64.zip',
  // 'win32-arm64': 'shaper_Windows_arm64.zip'
};

async function getRelease(version) {
  try {
    const response = await axios.get(
      `https://api.github.com/repos/taleshape-com/shaper/releases/tags/v${version}`,
      {
        headers: {
          'Accept': 'application/vnd.github.v3+json',
          'User-Agent': 'shaper-installer'
        }
      }
    );
    return response.data;
  } catch (error) {
    if (error.response?.status === 404) {
      throw new Error(`Version v${version} not found. Please check if this version exists in the releases.`);
    }
    throw new Error(`Error fetching release: ${error.message}`);
  }
}

async function downloadFile(url, dest) {
  const response = await axios({
    method: 'GET',
    url: url,
    responseType: 'stream',
    maxContentLength: 100 * 1024 * 1024, // 100MB max
    validateStatus: status => status === 200
  });

  const writer = fs.createWriteStream(dest);
  response.data.pipe(writer);

  return new Promise((resolve, reject) => {
    writer.on('finish', resolve);
    writer.on('error', reject);
    response.data.on('error', reject);
  });
}

function calculateSHA256(filePath) {
  return new Promise((resolve, reject) => {
    const hash = crypto.createHash('sha256');
    const stream = fs.createReadStream(filePath);

    stream.on('error', err => reject(err));
    stream.on('data', chunk => hash.update(chunk));
    stream.on('end', () => resolve(hash.digest('hex')));
  });
}

async function verifyChecksum(filePath, expectedChecksum) {
  const actualChecksum = await calculateSHA256(filePath);
  if (actualChecksum !== expectedChecksum) {
    throw new Error(`Checksum verification failed for ${path.basename(filePath)}\nExpected: ${expectedChecksum}\nActual:   ${actualChecksum}`);
  }
}

function loadChecksums() {
  const checksumsPath = path.join(__dirname, 'bin', 'SHA256SUMS');
  if (!fs.existsSync(checksumsPath)) {
    throw new Error('SHA256SUMS file not found in package');
  }

  const checksumsContent = fs.readFileSync(checksumsPath, 'utf8');
  const checksums = {};
  checksumsContent.split('\n').forEach(line => {
    if (line.trim()) {
      const [checksum, filename] = line.split(/\s+/);
      checksums[filename] = checksum;
    }
  });

  return checksums;
}

async function main() {
  try {
    // Create bin directory if it doesn't exist
    if (!fs.existsSync(BIN_DIR)) {
      fs.mkdirSync(BIN_DIR, { recursive: true });
    }

    const platform = `${process.platform}-${process.arch}`;
    const assetName = PLATFORM_MAP[platform];

    if (!assetName) {
      throw new Error(`Unsupported platform: ${platform}`);
    }

    // Load checksums from the package
    console.log('Loading checksums...');
    const checksums = loadChecksums();
    const expectedChecksum = checksums[assetName];

    if (!expectedChecksum) {
      throw new Error(`No checksum found for ${assetName} in SHA256SUMS`);
    }

    const release = await getRelease(VERSION);
    const asset = release.assets.find(a => a.name === assetName);

    if (!asset) {
      throw new Error(`Asset not found for platform ${platform} in version v${VERSION}`);
    }

    console.log(`Downloading shaper v${VERSION} for ${platform}...`);
    const binaryPath = path.join(BIN_DIR, 'shaper');
    await downloadFile(asset.browser_download_url, binaryPath);

    // Verify checksum
    console.log('Verifying checksum...');
    await verifyChecksum(binaryPath, expectedChecksum);
    console.log('Checksum verified successfully');

    // Set executable permissions
    fs.chmodSync(binaryPath, '755');

    console.log('Installation complete!');
  } catch (error) {
    console.error('Installation failed:', error.message);
    // Clean up binary if verification failed
    const binaryPath = path.join(BIN_DIR, 'shaper');
    if (fs.existsSync(binaryPath)) fs.unlinkSync(binaryPath);
    process.exit(1);
  }
}

main();