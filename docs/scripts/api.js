#!/usr/bin/env node
// =============================================================================
// isrvd API Harness — Node.js built-in edition, no curl/jq dependency
// Usage: node scripts/api.js <command> ...
// =============================================================================

const fs = require('fs');
const http = require('http');
const https = require('https');
const os = require('os');
const path = require('path');
const { URL } = require('url');
const crypto = require('crypto');

const configDir = path.join(process.env.XDG_CONFIG_HOME || path.join(os.homedir(), '.config'), 'isrvd');
const configFile = path.join(configDir, 'profile.json');

function color(code, text) { return `\x1b[${code}m${text}\x1b[0m`; }
function red(text) { console.error(color('0;31', text)); }
function green(text) { console.error(color('0;32', text)); }
function yellow(text) { console.error(color('0;33', text)); }
function blue(text) { console.error(color('0;34', text)); }

function loadConfig() {
  const cfg = {
    base_url: (process.env.ISRVD_BASE_URL || '').replace(/\/$/, ''),
    token: process.env.ISRVD_TOKEN || '',
    username: process.env.ISRVD_USERNAME || '',
  };
  if (cfg.base_url && cfg.token) return cfg;
  if (fs.existsSync(configFile)) {
    try {
      const saved = JSON.parse(fs.readFileSync(configFile, 'utf8'));
      cfg.base_url = cfg.base_url || String(saved.base_url || '').replace(/\/$/, '');
      cfg.token = cfg.token || String(saved.token || '');
      cfg.username = cfg.username || String(saved.username || '');
    } catch (_) {}
  }
  return cfg;
}

function saveConfig(cfg) {
  fs.mkdirSync(configDir, { recursive: true });
  fs.writeFileSync(configFile, JSON.stringify({
    base_url: (cfg.base_url || '').replace(/\/$/, ''),
    token: cfg.token || '',
    username: cfg.username || '',
  }, null, 2) + '\n', 'utf8');
  fs.chmodSync(configFile, 0o600);
}

function requireAuth() {
  const cfg = loadConfig();
  if (!cfg.base_url || !cfg.token) {
    red('✗ 未认证。请先执行:');
    red('  api.js login <base_url> <username> <password> [totpCode]');
    red('  api.js token <base_url> <token>');
    process.exit(1);
  }
  return cfg;
}

function httpRequest(method, urlText, token = '', body = undefined, headers = {}) {
  return new Promise((resolve, reject) => {
    const url = new URL(urlText);
    const data = body === undefined ? null : Buffer.from(JSON.stringify(body));
    const reqHeaders = { ...headers };
    if (token) reqHeaders.Authorization = `Bearer ${token}`;
    if (data) {
      reqHeaders['Content-Type'] = 'application/json';
      reqHeaders['Content-Length'] = String(data.length);
    }
    const lib = url.protocol === 'https:' ? https : http;
    const req = lib.request(url, { method, headers: reqHeaders }, res => {
      const chunks = [];
      res.on('data', chunk => chunks.push(chunk));
      res.on('end', () => {
        const raw = Buffer.concat(chunks).toString('utf8');
        if (res.statusCode < 200 || res.statusCode >= 300) {
          printCompactJsonOrRaw(raw);
          red(`✗ HTTP ${res.statusCode}`);
          process.exit(1);
        }
        if (!raw) return resolve(null);
        try { resolve(JSON.parse(raw)); } catch (_) { resolve(raw); }
      });
    });
    req.on('error', reject);
    if (data) req.write(data);
    req.end();
  });
}

function multipartRequest(urlText, token, fileField, filePath, fields) {
  return new Promise((resolve, reject) => {
    const boundary = `isrvd-${crypto.randomUUID()}`;
    const chunks = [];
    for (const field of fields) {
      const index = field.indexOf('=');
      const key = index >= 0 ? field.slice(0, index) : field;
      const value = index >= 0 ? field.slice(index + 1) : '';
      chunks.push(Buffer.from(`--${boundary}\r\nContent-Disposition: form-data; name="${key}"\r\n\r\n${value}\r\n`));
    }
    chunks.push(Buffer.from(`--${boundary}\r\nContent-Disposition: form-data; name="${fileField}"; filename="${path.basename(filePath)}"\r\nContent-Type: application/octet-stream\r\n\r\n`));
    chunks.push(fs.readFileSync(filePath));
    chunks.push(Buffer.from(`\r\n--${boundary}--\r\n`));
    const data = Buffer.concat(chunks);
    const url = new URL(urlText);
    const lib = url.protocol === 'https:' ? https : http;
    const req = lib.request(url, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${token}`,
        'Content-Type': `multipart/form-data; boundary=${boundary}`,
        'Content-Length': String(data.length),
      },
    }, res => {
      const out = [];
      res.on('data', chunk => out.push(chunk));
      res.on('end', () => {
        const raw = Buffer.concat(out).toString('utf8');
        if (res.statusCode < 200 || res.statusCode >= 300) {
          printCompactJsonOrRaw(raw);
          red(`✗ HTTP ${res.statusCode}`);
          process.exit(1);
        }
        if (!raw) return resolve(null);
        try { resolve(JSON.parse(raw)); } catch (_) { resolve(raw); }
      });
    });
    req.on('error', reject);
    req.write(data);
    req.end();
  });
}

function payloadOf(value) {
  return value && typeof value === 'object' && !Array.isArray(value) && Object.prototype.hasOwnProperty.call(value, 'payload') ? value.payload : value;
}

function applyFilter(value, selector) {
  if (!selector) return value;
  if (!selector.startsWith('.')) throw new Error(`unsupported selector: ${selector}`);
  let current = value;
  for (const part of selector.slice(1).split('.')) {
    if (!part) continue;
    if (current && typeof current === 'object' && !Array.isArray(current)) current = current[part];
    else if (Array.isArray(current) && /^\d+$/.test(part)) current = current[Number(part)];
    else return null;
  }
  return current === undefined ? null : current;
}

function encodeCell(value) {
  if (value === null || value === undefined) return 'null';
  if (typeof value === 'string') return JSON.stringify(value);
  if (typeof value === 'number' || typeof value === 'boolean') return String(value);
  return JSON.stringify(value);
}

function printResult(value) {
  if (Array.isArray(value) && value.length > 0 && value.every(item => item && typeof item === 'object' && !Array.isArray(item))) {
    const keys = Object.keys(value[0]);
    console.log(`[${value.length}]{${keys.join(',')}}:`);
    for (const item of value) console.log('  ' + keys.map(key => encodeCell(item[key])).join(','));
    return;
  }
  console.log(JSON.stringify(value));
}

function printCompactJsonOrRaw(raw) {
  try { console.log(JSON.stringify(JSON.parse(raw))); } catch (_) { console.log(raw); }
}

async function apiCall(method, apiPath, bodyText = '', selector = '') {
  const cfg = requireAuth();
  const body = bodyText ? JSON.parse(bodyText) : undefined;
  const data = await httpRequest(method, `${cfg.base_url}/api${apiPath}`, cfg.token, body);
  printResult(applyFilter(payloadOf(data), selector));
}

function usage() {
  console.log(`isrvd API Harness (Node.js built-in edition)

  Auth:
    api.js login  <url> <user> <pass> [totp]
    api.js token  <url> <token>
    api.js logout
    api.js status
    api.js whoami

  API:
    api.js get    <path> [selector]
    api.js post   <path> [body] [selector]
    api.js put    <path> [body] [selector]
    api.js patch  <path> [body] [selector]
    api.js delete <path> [selector]
    api.js upload <path> <field> <file> [k=v...]

  Examples:
    api.js token "$ISRVD_APIURL" "$ISRVD_APITOKEN"
    api.js get /docker/containers
    api.js post /docker/container '{"image":"...","name":"..."}'
`);
}

async function main() {
  const [command, ...args] = process.argv.slice(2);
  try {
    switch (command) {
      case 'login': {
        const [base, username, password, totp] = args;
        if (!base || !username || !password) throw new Error('usage: api.js login <base_url> <username> <password> [totpCode]');
        const baseUrl = base.replace(/\/$/, '');
        const body = { username, password };
        if (totp) body.totpCode = totp;
        blue(`→ 登录 ${baseUrl} ...`);
        const data = await httpRequest('POST', `${baseUrl}/api/account/login`, '', body);
        if (!data || data.success !== true) throw new Error(`登录失败: ${data && data.message ? data.message : '未知错误'}`);
        if (data.payload && data.payload.twoFactorRequired) throw new Error('该账号已启用 TOTP 二次验证。请使用: api.js login <base_url> <username> <password> <totpCode>');
        const token = data.payload && data.payload.token;
        if (!token) throw new Error('登录失败: 响应中缺少 token');
        saveConfig({ base_url: baseUrl, token, username });
        green(`✓ 登录成功，已保存到 ${configFile}`);
        break;
      }
      case 'token': {
        const [base, token] = args;
        if (!base || !token) throw new Error('usage: api.js token <base_url> <token>');
        const baseUrl = base.replace(/\/$/, '');
        blue('→ 验证 token ...');
        const data = await httpRequest('GET', `${baseUrl}/api/account/info`, token);
        if (!data || data.success !== true) throw new Error(`token 无效: ${data && data.message ? data.message : '未知错误'}`);
        const username = (data.payload && data.payload.username) || 'unknown';
        saveConfig({ base_url: baseUrl, token, username });
        green(`✓ token 有效 (用户: ${username})，已保存到 ${configFile}`);
        break;
      }
      case 'get': await apiCall('GET', args[0], '', args[1] || ''); break;
      case 'delete': await apiCall('DELETE', args[0], '', args[1] || ''); break;
      case 'post': await apiCall('POST', args[0], args[1] || '', args[2] || ''); break;
      case 'put': await apiCall('PUT', args[0], args[1] || '', args[2] || ''); break;
      case 'patch': await apiCall('PATCH', args[0], args[1] || '', args[2] || ''); break;
      case 'upload': {
        const [apiPath, field, file, ...fields] = args;
        if (!apiPath || !field || !file) throw new Error('usage: api.js upload <path> <file_field> <file_path> [key=value ...]');
        const cfg = requireAuth();
        printResult(await multipartRequest(`${cfg.base_url}/api${apiPath}`, cfg.token, field, file, fields));
        break;
      }
      case 'whoami': await apiCall('GET', '/account/info'); break;
      case 'status': {
        const cfg = loadConfig();
        if (fs.existsSync(configFile)) green(`✓ 配置文件: ${configFile}`); else yellow('○ 无配置文件');
        printResult({ base_url: cfg.base_url, username: cfg.username, token: cfg.token ? `${cfg.token.slice(0, 20)}...` : '' });
        break;
      }
      case 'logout':
        if (fs.existsSync(configFile)) { fs.unlinkSync(configFile); green('✓ 已清除认证信息'); } else yellow('○ 无需清除');
        break;
      case 'help':
      case undefined:
        usage();
        break;
      default:
        throw new Error(`unknown command: ${command}`);
    }
  } catch (err) {
    red(`✗ ${err.message}`);
    process.exit(1);
  }
}

main();
