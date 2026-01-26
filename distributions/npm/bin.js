#!/usr/bin/env node

/**
 * This file is used to run the Hyperterse binary.
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const binPath = path.join(__dirname, 'bin', process.platform === 'win32' ? 'hyperterse.exe' : 'hyperterse');

if (!fs.existsSync(binPath)) {
  console.error('Hyperterse binary not found');
  process.exit(1);
}

execSync(binPath, { stdio: 'inherit' });
