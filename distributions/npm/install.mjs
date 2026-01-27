#!/usr/bin/env node

/**
 * Post-install script to download the correct binary for the current platform
 */

import fs from 'fs';
import path from 'path';
import https from 'https';
import http from 'http';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// Version is read from package.json (npm_package_version) or can be overridden via environment variable
let VERSION = process.env.npm_package_version;
if (!VERSION) {
  // Fallback: read from package.json if env var not set
  const packageJsonPath = path.join(__dirname, 'package.json');
  if (fs.existsSync(packageJsonPath)) {
    const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf8'));
    VERSION = packageJson.version || '0.0.0';
  } else {
    VERSION = '0.0.0';
  }
}
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

// Main installation function
export async function install() {
  // Check if binary already exists
  if (fs.existsSync(binPath)) {
    console.log('Binary already exists, skipping download.');
    return;
  }

  console.log(`Downloading hyperterse ${VERSION} for ${platform}-${arch}...`);
  console.log(`URL: ${downloadUrl}`);

  try {
    // Download archive
    await download(downloadUrl, binPath);

    // Make binary executable (Unix)
    if (platform !== 'win32') {
      fs.chmodSync(binPath, '755');
    }

    console.log('✓ Installation complete!');
  } catch (error) {
    console.error('Error installing binary:', error.message);
    console.error('You may need to manually download the binary from:', downloadUrl);
    throw error;
  }
}

