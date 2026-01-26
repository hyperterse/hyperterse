#!/usr/bin/env node

/**
 * Post-install script to download the correct binary for the current platform
 */

const fs = require('fs');
const path = require('path');
const https = require('https');
const http = require('http');

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

const downloadUrl = `${BASE_URL}/${binaryName}`;
const binDir = path.join(__dirname, 'dist');
const binPath = path.join(binDir, platform === 'win32' ? 'hyperterse.exe' : 'hyperterse');

// Create bin directory if it doesn't exist
if (!fs.existsSync(binDir)) {
  fs.mkdirSync(binDir, { recursive: true });
}

// Download function
function download(url, dest) {
  return new Promise((resolve, reject) => {
    const protocol = url.startsWith('https') ? https : http;
    const file = fs.createWriteStream(binPath);

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

// Main installation logic
async function install() {
  // Check if binary already exists
  if (fs.existsSync(binPath)) {
    console.log('Binary already exists, skipping download.');
    return;
  }

  console.log(`Downloading hyperterse ${VERSION} for ${platform}-${arch}...`);
  console.log(`URL: ${downloadUrl}`);

  try {
    // Download archive
    await download(downloadUrl);

    // Make binary executable (Unix)
    if (platform !== 'win32') {
      fs.chmodSync(binPath, '755');
    }

    console.log('✓ Installation complete!');
    process.exit(0);
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
