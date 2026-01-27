#!/usr/bin/env node

/**
 * This file is used to run the Hyperterse binary.
 * If the binary doesn't exist, it will automatically install it.
 */

import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';
import { fileURLToPath } from 'url';

import { install } from './install.mjs';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const binPath = path.join(__dirname, 'bin', process.platform === 'win32' ? 'hyperterse.exe' : 'hyperterse');

(async () => {
  if (!fs.existsSync(binPath)) {
    console.log('Hyperterse binary not found. Installing...');

    try {
      await install();
    } catch (error) {
      console.error('Error: Binary installation failed');
      process.exit(1);
    }

    // Verify binary was installed
    if (!fs.existsSync(binPath)) {
      console.error('Error: Binary installation failed');
      process.exit(1);
    }
  }

  execSync(binPath, { stdio: 'inherit' });
})();
