#!/usr/bin/env node

/**
 * This file is used to run the Hyperterse binary.
 */

import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';

const binPath = path.join(import.meta.dirname, 'bin', process.platform === 'win32' ? 'hyperterse.exe' : 'hyperterse');

if (!fs.existsSync(binPath)) {
  console.error('Hyperterse binary not found');
  process.exit(1);
}

execSync(binPath, { stdio: 'inherit' });
