const fs = require('fs');
const path = require('path');
const os = require('os');
const https = require('https');
const { execSync } = require('child_process');
const axios = require('axios');
const tar = require('tar');
const extract = require('extract-zip');

const BIN_DIR = path.join(__dirname, 'bin');
const VERSION = require('./package.json').version;

// Map of platform names to GitHub release asset names
const PLATFORM_MAP = {
  'linux-x64': 'shaper_Linux_x86_64.tar.gz',
  'linux-arm64': 'shaper_Linux_arm64.tar.gz',
  'darwin-x64': 'shaper_Darwin_x86_64.tar.gz',
  'darwin-arm64': 'shaper_Darwin_arm64.tar.gz',
  // 'win32-x64': 'shaper_Windows_x86_64.zip',
  // 'win32-arm64': 'shaper_Windows_arm64.zip'
};

async function getLatestRelease() {
  try {
    const response = await axios.get(
      'https://api.github.com/repos/taleshape-com/shaper/releases/latest',
      {
        headers: {
          'Accept': 'application/vnd.github.v3+json',
          'User-Agent': 'shaper-installer'
        }
      }
    );
    return response.data;
  } catch (error) {
    console.error('Error fetching latest release:', error.message);
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

async function extractArchive(archivePath, isZip) {
  const extractPath = path.join(BIN_DIR, 'temp');
  fs.mkdirSync(extractPath, { recursive: true });

  try {
    if (isZip) {
      await extract(archivePath, { dir: extractPath });
    } else {
      await tar.x({
        file: archivePath,
        cwd: extractPath
      });
    }

    // Move the binary to the bin directory
    const binaryPath = path.join(extractPath, 'shaper');
    const targetPath = path.join(BIN_DIR, 'shaper');
    fs.renameSync(binaryPath, targetPath);
    fs.chmodSync(targetPath, '755');
  } finally {
    // Cleanup
    fs.rmSync(extractPath, { recursive: true, force: true });
    fs.unlinkSync(archivePath);
  }
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
    const release = await getLatestRelease();
    const asset = release.assets.find(a => a.name === assetName);

    if (!asset) {
      console.error(`Asset not found for platform ${platform}`);
      process.exit(1);
    }

    console.log(`Downloading shaper ${release.tag_name} for ${platform}...`);
    const archivePath = path.join(os.tmpdir(), assetName);
    await downloadFile(asset.browser_download_url, archivePath);

    console.log('Extracting binary...');
    await extractArchive(archivePath, assetName.endsWith('.zip'));

    console.log('Installation complete!');
  } catch (error) {
    console.error('Installation failed:', error);
    process.exit(1);
  }
}

main(); 