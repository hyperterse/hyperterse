#!/usr/bin/env node

/**
 * Post-install script to download the correct binary for the current platform
 */

const fs = require('fs');
const path = require('path');
const https = require('https');
const http = require('http');
const { execSync } = require('child_process');

// Version is read from package.json (npm_package_version) or can be overridden via environment variable
const VERSION = process.env.npm_package_version || '0.0.0';
const BASE_URL = `https://github.com/hyperterse/hyperterse/releases/download/v${VERSION}`;

// Detect platform and architecture
const platform = process.platform;
const arch = process.arch;

// Map Node.js arch to Go arch
const archMap = {
  'x64': 'amd64',
  'arm64': 'arm64',
  'arm': 'arm',
  'ia32': 'amd64', // 32-bit x86 -> amd64
};

// Map Node.js platform to Go OS
const platformMap = {
  'darwin': 'darwin',
  'linux': 'linux',
  'win32': 'windows',
};

const goArch = archMap[arch] || arch;
const goOS = platformMap[platform] || platform;

// Determine binary name and extension
const binaryName = platform === 'win32'
  ? `hyperterse-${goOS}-${goArch}.exe`
  : `hyperterse-${goOS}-${goArch}`;

const archiveName = platform === 'win32'
  ? `hyperterse-${goOS}-${goArch}.zip`
  : `hyperterse-${goOS}-${goArch}.tar.gz`;

const downloadUrl = `${BASE_URL}/${archiveName}`;
const binDir = path.join(__dirname, 'bin');
const binPath = path.join(binDir, platform === 'win32' ? 'hyperterse.exe' : 'hyperterse');

// Create bin directory if it doesn't exist
if (!fs.existsSync(binDir)) {
  fs.mkdirSync(binDir, { recursive: true });
}

// Download function
function download(url, dest) {
  return new Promise((resolve, reject) => {
    const protocol = url.startsWith('https') ? https : http;
    const file = fs.createWriteStream(dest);

    protocol.get(url, (response) => {
      if (response.statusCode === 302 || response.statusCode === 301) {
        // Follow redirect
        return download(response.headers.location, dest).then(resolve).catch(reject);
      }

      if (response.statusCode !== 200) {
        reject(new Error(`Failed to download: ${response.statusCode} ${response.statusMessage}`));
        return;
      }

      response.pipe(file);
      file.on('finish', () => {
        file.close();
        resolve();
      });
    }).on('error', (err) => {
      fs.unlinkSync(dest);
      reject(err);
    });
  });
}

// Extract function
function extract(archivePath, destDir) {
  if (archivePath.endsWith('.zip')) {
    // Use unzip command (Windows)
    try {
      execSync(`unzip -o "${archivePath}" -d "${destDir}"`, { stdio: 'inherit' });
    } catch (err) {
      console.error('Error extracting zip file. Make sure unzip is installed.');
      throw err;
    }
  } else if (archivePath.endsWith('.tar.gz')) {
    // Use tar command (Unix)
    try {
      execSync(`tar -xzf "${archivePath}" -C "${destDir}"`, { stdio: 'inherit' });
    } catch (err) {
      console.error('Error extracting tar.gz file. Make sure tar is installed.');
      throw err;
    }
  }
}

// Main installation logic
async function install() {
  // Check if binary already exists
  if (fs.existsSync(binPath)) {
    console.log('Binary already exists, skipping download.');
    return;
  }

  console.log(`Downloading hyperterse ${VERSION} for ${platform}-${arch}...`);
  console.log(`URL: ${downloadUrl}`);

  const tempArchive = path.join(__dirname, archiveName);

  try {
    // Download archive
    await download(downloadUrl, tempArchive);

    // Extract archive
    console.log('Extracting archive...');
    extract(tempArchive, binDir);

    // Find the extracted binary
    const extractedBinary = path.join(binDir, binaryName);
    if (fs.existsSync(extractedBinary)) {
      // Move to final location
      fs.renameSync(extractedBinary, binPath);
    } else {
      // Try finding any hyperterse binary in the directory
      const files = fs.readdirSync(binDir);
      const hyperterseFile = files.find(f => f.startsWith('hyperterse'));
      if (hyperterseFile) {
        fs.renameSync(path.join(binDir, hyperterseFile), binPath);
      } else {
        throw new Error('Could not find extracted binary');
      }
    }

    // Make binary executable (Unix)
    if (platform !== 'win32') {
      fs.chmodSync(binPath, '755');
    }

    // Clean up archive
    fs.unlinkSync(tempArchive);

    console.log('✓ Installation complete!');
  } catch (error) {
    console.error('Error installing binary:', error.message);
    console.error('You may need to manually download the binary from:', downloadUrl);
    process.exit(1);
  }
}

// Run installation
install().catch((err) => {
  console.error('Installation failed:', err);
  process.exit(1);
});
