#!/usr/bin/env node
const { spawn } = require('child_process');
const path = require('path');
const bin = path.join(__dirname, process.platform === 'win32' ? 'hotline-ua-mcp.exe' : 'hotline-ua-mcp');
const child = spawn(bin, [], { stdio: 'inherit' });
child.on('exit', code => process.exit(code ?? 0));
