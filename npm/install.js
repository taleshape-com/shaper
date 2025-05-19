const fs = require('fs');
const path = require('path');
const os = require('os');
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
      console.error(`Version v${version} not found. Please check if this version exists in the releases.`);
    } else {
      console.error('Error fetching release:', error.message);
    }
    process.exit(1);
  }
}

async function downloadFile(url, dest) {
  const response = await axios({
    method: 'GET',
    url: url,
    responseType: 'stream'
  });

  const writer = fs.createWriteStream(dest);
  response.data.pipe(writer);

  return new Promise((resolve, reject) => {
    writer.on('finish', resolve);
    writer.on('error', reject);
  });
}

async function main() {
  // Create bin directory if it doesn't exist
  if (!fs.existsSync(BIN_DIR)) {
    fs.mkdirSync(BIN_DIR, { recursive: true });
  }

  const platform = `${process.platform}-${process.arch}`;
  const assetName = PLATFORM_MAP[platform];

  if (!assetName) {
    console.error(`Unsupported platform: ${platform}`);
    process.exit(1);
  }

  try {
    const release = await getRelease(VERSION);
    const asset = release.assets.find(a => a.name === assetName);

    if (!asset) {
      console.error(`Asset not found for platform ${platform} in version v${VERSION}`);
      process.exit(1);
    }

    console.log(`Downloading shaper v${VERSION} for ${platform}...`);
    const binaryPath = path.join(BIN_DIR, 'shaper');
    await downloadFile(asset.browser_download_url, binaryPath);
    fs.chmodSync(binaryPath, '755');

    console.log('Installation complete!');
  } catch (error) {
    console.error('Installation failed:', error);
    process.exit(1);
  }
}

main(); 