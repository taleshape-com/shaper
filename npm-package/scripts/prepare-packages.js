// SPDX-License-Identifier: MPL-2.0

const fs = require('fs');
const path = require('path');

const TARGETS = [
  {
    packageName: '@taleshape/shaper-darwin-arm64',
    dirName: 'shaper-darwin-arm64',
    binarySource: 'shaper-darwin-arm64',
  },
  {
    packageName: '@taleshape/shaper-darwin-x64',
    dirName: 'shaper-darwin-x64',
    binarySource: 'shaper-darwin-amd64',
  },
  {
    packageName: '@taleshape/shaper-linux-arm64',
    dirName: 'shaper-linux-arm64',
    binarySource: 'shaper-linux-arm64',
  },
  {
    packageName: '@taleshape/shaper-linux-x64',
    dirName: 'shaper-linux-x64',
    binarySource: 'shaper-linux-amd64',
  },
];

function main() {
  const version = process.argv[2];
  const binDir = process.argv[3];

  if (!version) {
    console.error('Error: Version argument required.');
    console.error('Usage: node prepare-packages.js <version> [bin-dir]');
    process.exit(1);
  }

  const npmPackageDir = path.resolve(__dirname, '..');
  const mainPkgPath = path.join(npmPackageDir, 'package.json');

  console.log(`Preparing npm packages for version ${version}...`);

  // Update main package.json
  const mainPkg = JSON.parse(fs.readFileSync(mainPkgPath, 'utf8'));
  mainPkg.version = version;
  if (!mainPkg.optionalDependencies) {
    mainPkg.optionalDependencies = {};
  }

  for (const target of TARGETS) {
    mainPkg.optionalDependencies[target.packageName] = version;
  }

  fs.writeFileSync(mainPkgPath, JSON.stringify(mainPkg, null, 2) + '\n');
  console.log(`Updated main package.json version to ${version}`);

  // Prepare each platform package
  for (const target of TARGETS) {
    const targetDir = path.join(npmPackageDir, 'packages', target.dirName);
    const targetPkgPath = path.join(targetDir, 'package.json');

    if (!fs.existsSync(targetPkgPath)) {
      console.error(`Error: Package file missing at ${targetPkgPath}`);
      process.exit(1);
    }

    const targetPkg = JSON.parse(fs.readFileSync(targetPkgPath, 'utf8'));
    targetPkg.version = version;
    fs.writeFileSync(targetPkgPath, JSON.stringify(targetPkg, null, 2) + '\n');

    // Copy License and README
    const licensePath = path.join(npmPackageDir, 'LICENSE');
    const readmePath = path.join(npmPackageDir, 'README.md');
    if (fs.existsSync(licensePath)) {
      fs.copyFileSync(licensePath, path.join(targetDir, 'LICENSE'));
    }
    if (fs.existsSync(readmePath)) {
      fs.copyFileSync(readmePath, path.join(targetDir, 'README.md'));
    }

    // Copy binary if binDir is provided
    if (binDir) {
      const srcBinary = path.resolve(binDir, target.binarySource);
      const targetBinDir = path.join(targetDir, 'bin');
      if (!fs.existsSync(targetBinDir)) {
        fs.mkdirSync(targetBinDir, { recursive: true });
      }

      const destBinary = path.join(targetBinDir, 'shaper');
      if (fs.existsSync(srcBinary)) {
        fs.copyFileSync(srcBinary, destBinary);
        fs.chmodSync(destBinary, 0o755);
        console.log(`Copied ${srcBinary} -> ${destBinary}`);
      } else {
        console.warn(`Warning: Binary source ${srcBinary} not found!`);
      }
    }

    console.log(`Prepared platform package ${target.packageName}`);
  }

  console.log('Preparation complete!');
}

main();
