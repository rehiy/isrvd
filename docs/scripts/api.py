#!/usr/bin/env python3
# =============================================================================
# isrvd API Harness — Python stdlib edition, no curl/jq dependency
# Usage: python3 scripts/api.py <command> ...
# =============================================================================

from __future__ import annotations

import argparse
import json
import mimetypes
import os
import stat
import sys
import uuid
from pathlib import Path
from typing import Any
from urllib import error, parse, request

CONFIG_DIR = Path(os.environ.get('XDG_CONFIG_HOME', Path.home() / '.config')) / 'isrvd'
CONFIG_FILE = CONFIG_DIR / 'profile.json'


def color(code: str, text: str) -> str:
    return f'\033[{code}m{text}\033[0m'


def red(text: str) -> None:
    print(color('0;31', text), file=sys.stderr)


def green(text: str) -> None:
    print(color('0;32', text), file=sys.stderr)


def yellow(text: str) -> None:
    print(color('0;33', text), file=sys.stderr)


def blue(text: str) -> None:
    print(color('0;34', text), file=sys.stderr)


def load_config() -> dict[str, str]:
    cfg = {
        'base_url': os.environ.get('ISRVD_BASE_URL', '').rstrip('/'),
        'token': os.environ.get('ISRVD_TOKEN', ''),
        'username': os.environ.get('ISRVD_USERNAME', ''),
    }
    if cfg['base_url'] and cfg['token']:
        return cfg
    if CONFIG_FILE.exists():
        try:
            saved = json.loads(CONFIG_FILE.read_text(encoding='utf-8'))
            cfg['base_url'] = cfg['base_url'] or str(saved.get('base_url') or '').rstrip('/')
            cfg['token'] = cfg['token'] or str(saved.get('token') or '')
            cfg['username'] = cfg['username'] or str(saved.get('username') or '')
        except json.JSONDecodeError:
            pass
    return cfg


def save_config(cfg: dict[str, str]) -> None:
    CONFIG_DIR.mkdir(parents=True, exist_ok=True)
    CONFIG_FILE.write_text(json.dumps({
        'base_url': cfg.get('base_url', '').rstrip('/'),
        'token': cfg.get('token', ''),
        'username': cfg.get('username', ''),
    }, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    CONFIG_FILE.chmod(stat.S_IRUSR | stat.S_IWUSR)


def require_auth() -> dict[str, str]:
    cfg = load_config()
    if not cfg.get('base_url') or not cfg.get('token'):
        red('✗ 未认证。请先执行:')
        red('  api.py login <base_url> <username> <password> [totpCode]')
        red('  api.py token <base_url> <token>')
        raise SystemExit(1)
    return cfg


def request_json(method: str, url: str, token: str = '', body: Any = None, headers: dict[str, str] | None = None) -> Any:
    data = None
    req_headers = dict(headers or {})
    if token:
        req_headers['Authorization'] = f'Bearer {token}'
    if body is not None:
        data = json.dumps(body, ensure_ascii=False, separators=(',', ':')).encode('utf-8')
        req_headers['Content-Type'] = 'application/json'
    req = request.Request(url, data=data, headers=req_headers, method=method)
    try:
        with request.urlopen(req) as resp:
            raw = resp.read().decode('utf-8')
    except error.HTTPError as exc:
        raw = exc.read().decode('utf-8', errors='replace')
        print_compact_json_or_raw(raw)
        red(f'✗ HTTP {exc.code}')
        raise SystemExit(1)
    except error.URLError as exc:
        red(f'✗ 请求失败: {exc.reason}')
        raise SystemExit(1)
    if not raw:
        return None
    try:
        return json.loads(raw)
    except json.JSONDecodeError:
        return raw


def request_multipart(url: str, token: str, file_field: str, file_path: str, fields: list[str]) -> Any:
    path = Path(file_path)
    boundary = f'isrvd-{uuid.uuid4().hex}'
    parts: list[bytes] = []
    for field in fields:
        key, _, value = field.partition('=')
        parts.append(f'--{boundary}\r\n'.encode())
        parts.append(f'Content-Disposition: form-data; name="{key}"\r\n\r\n{value}\r\n'.encode())
    ctype = mimetypes.guess_type(path.name)[0] or 'application/octet-stream'
    parts.append(f'--{boundary}\r\n'.encode())
    parts.append(f'Content-Disposition: form-data; name="{file_field}"; filename="{path.name}"\r\n'.encode())
    parts.append(f'Content-Type: {ctype}\r\n\r\n'.encode())
    parts.append(path.read_bytes())
    parts.append(f'\r\n--{boundary}--\r\n'.encode())
    data = b''.join(parts)
    req = request.Request(url, data=data, method='POST', headers={
        'Authorization': f'Bearer {token}',
        'Content-Type': f'multipart/form-data; boundary={boundary}',
        'Content-Length': str(len(data)),
    })
    try:
        with request.urlopen(req) as resp:
            raw = resp.read().decode('utf-8')
    except error.HTTPError as exc:
        raw = exc.read().decode('utf-8', errors='replace')
        print_compact_json_or_raw(raw)
        red(f'✗ HTTP {exc.code}')
        raise SystemExit(1)
    if not raw:
        return None
    try:
        return json.loads(raw)
    except json.JSONDecodeError:
        return raw


def payload_of(value: Any) -> Any:
    if isinstance(value, dict) and 'payload' in value:
        return value['payload']
    return value


def apply_filter(value: Any, selector: str) -> Any:
    if not selector:
        return value
    if not selector.startswith('.'):
        raise SystemExit(f'unsupported selector: {selector}')
    current = value
    for part in selector[1:].split('.'):
        if not part:
            continue
        if isinstance(current, dict):
            current = current.get(part)
        elif isinstance(current, list) and part.isdigit():
            idx = int(part)
            current = current[idx] if idx < len(current) else None
        else:
            return None
    return current


def encode_cell(value: Any) -> str:
    if value is None:
        return 'null'
    if isinstance(value, str):
        return json.dumps(value, ensure_ascii=False)
    if isinstance(value, (int, float, bool)):
        return str(value).lower() if isinstance(value, bool) else str(value)
    return json.dumps(value, ensure_ascii=False, separators=(',', ':'))


def print_result(value: Any) -> None:
    if isinstance(value, list) and value and all(isinstance(item, dict) for item in value):
        keys = list(value[0].keys())
        print(f'[{len(value)}]{{{",".join(keys)}}}:')
        for item in value:
            print('  ' + ','.join(encode_cell(item.get(key)) for key in keys))
        return
    print(json.dumps(value, ensure_ascii=False, separators=(',', ':')))


def print_compact_json_or_raw(raw: str) -> None:
    try:
        print(json.dumps(json.loads(raw), ensure_ascii=False, separators=(',', ':')))
    except json.JSONDecodeError:
        print(raw)


def api_call(method: str, path: str, body_text: str = '', selector: str = '') -> None:
    cfg = require_auth()
    body = json.loads(body_text) if body_text else None
    url = cfg['base_url'] + '/api' + path
    data = request_json(method, url, cfg['token'], body)
    print_result(apply_filter(payload_of(data), selector))


def cmd_login(args: argparse.Namespace) -> None:
    base_url = args.base_url.rstrip('/')
    body = {'username': args.username, 'password': args.password}
    if args.totp_code:
        body['totpCode'] = args.totp_code
    blue(f'→ 登录 {base_url} ...')
    data = request_json('POST', base_url + '/api/account/login', body=body)
    if not isinstance(data, dict) or data.get('success') is not True:
        red(f'✗ 登录失败: {data.get("message", "未知错误") if isinstance(data, dict) else data}')
        raise SystemExit(1)
    payload = data.get('payload') or {}
    if payload.get('twoFactorRequired'):
        red('✗ 该账号已启用 TOTP 二次验证。请使用: api.py login <base_url> <username> <password> <totpCode>')
        raise SystemExit(1)
    token = payload.get('token') or ''
    if not token:
        red('✗ 登录失败: 响应中缺少 token')
        raise SystemExit(1)
    save_config({'base_url': base_url, 'token': token, 'username': args.username})
    green(f'✓ 登录成功，已保存到 {CONFIG_FILE}')


def cmd_token(args: argparse.Namespace) -> None:
    base_url = args.base_url.rstrip('/')
    blue('→ 验证 token ...')
    data = request_json('GET', base_url + '/api/overview/bootstrap', args.token)
    if not isinstance(data, dict) or data.get('success') is not True:
        red(f'✗ token 无效: {data.get("message", "未知错误") if isinstance(data, dict) else data}')
        raise SystemExit(1)
    username = str((data.get('payload') or {}).get('auth', {}).get('username') or 'unknown')
    save_config({'base_url': base_url, 'token': args.token, 'username': username})
    green(f'✓ token 有效 (用户: {username})，已保存到 {CONFIG_FILE}')


def cmd_upload(args: argparse.Namespace) -> None:
    cfg = require_auth()
    data = request_multipart(cfg['base_url'] + '/api' + args.path, cfg['token'], args.file_field, args.file_path, args.fields)
    print_result(data)


def cmd_status(_: argparse.Namespace) -> None:
    cfg = load_config()
    if CONFIG_FILE.exists():
        green(f'✓ 配置文件: {CONFIG_FILE}')
    else:
        yellow('○ 无配置文件')
    masked = (cfg.get('token') or '')[:20] + ('...' if cfg.get('token') else '')
    print_result({'base_url': cfg.get('base_url', ''), 'username': cfg.get('username', ''), 'token': masked})


def cmd_logout(_: argparse.Namespace) -> None:
    if CONFIG_FILE.exists():
        CONFIG_FILE.unlink()
        green('✓ 已清除认证信息')
    else:
        yellow('○ 无需清除')


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description='isrvd API Harness (Python stdlib edition)')
    sub = parser.add_subparsers(dest='command', required=True)
    p = sub.add_parser('login')
    p.add_argument('base_url')
    p.add_argument('username')
    p.add_argument('password')
    p.add_argument('totp_code', nargs='?')
    p.set_defaults(func=cmd_login)
    p = sub.add_parser('token')
    p.add_argument('base_url')
    p.add_argument('token')
    p.set_defaults(func=cmd_token)
    for name, method in [('get', 'GET'), ('delete', 'DELETE')]:
        p = sub.add_parser(name)
        p.add_argument('path')
        p.add_argument('selector', nargs='?', default='')
        p.set_defaults(func=lambda args, method=method: api_call(method, args.path, selector=args.selector))
    for name, method in [('post', 'POST'), ('put', 'PUT'), ('patch', 'PATCH')]:
        p = sub.add_parser(name)
        p.add_argument('path')
        p.add_argument('body', nargs='?', default='')
        p.add_argument('selector', nargs='?', default='')
        p.set_defaults(func=lambda args, method=method: api_call(method, args.path, args.body, args.selector))
    p = sub.add_parser('upload')
    p.add_argument('path')
    p.add_argument('file_field')
    p.add_argument('file_path')
    p.add_argument('fields', nargs='*')
    p.set_defaults(func=cmd_upload)
    p = sub.add_parser('whoami')
    p.set_defaults(func=lambda args: api_call('GET', '/overview/bootstrap'))
    p = sub.add_parser('status')
    p.set_defaults(func=cmd_status)
    p = sub.add_parser('logout')
    p.set_defaults(func=cmd_logout)
    return parser


def main() -> None:
    parser = build_parser()
    args = parser.parse_args()
    args.func(args)


if __name__ == '__main__':
    main()
