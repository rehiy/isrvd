#!/usr/bin/env python3
"""
校验前端 API 方法和路径是否存在于 OpenAPI 或中控节点网关权限路由中。
参数名及编码表达式按占位路径段比较，Compose/指标网关按类型联合分别展开。
递归核对 webview/src 的 Vue/TS 静态权限引用；动态拼接权限另行人工审阅。

用法（仓库根目录）：python3 webview/scripts/check-api-contract.py
用法（webview 目录）：python3 scripts/check-api-contract.py
"""

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Iterator

from locale_utils import compile_pattern

ROOT = Path(__file__).resolve().parents[2]
REQUEST_PATTERN = compile_pattern(r'''(?<![A-Za-z0-9_])(?:http|httpBlob)\.(get|post|put|patch|delete)\s*(?:<[^\(\n]*>)?\s*\(\s*(['"`])([^'"`]+)\2''')
PERMISSION_PATTERN = compile_pattern(r'''(?<![A-Za-z0-9_])hasPerm\s*\(\s*(['"])(GET|POST|PUT|PATCH|DELETE)\s+/api/([^'"\r\n]+)\1''')
NODE_ROUTE_PATTERN = compile_pattern(r'\{\s*Route[A-Za-z0-9_]+\s*,\s*"(GET|POST|PUT|PATCH|DELETE)"\s*,\s*"([^"]+)"\s*,')
COPILOT_ROUTE_PATTERN = compile_pattern(r'\{\s*Method:\s*"(GET|POST|PUT|PATCH|DELETE)",\s*Path:\s*"([^"]+)",\s*Handler:\s*app\.copilotAGUI(?![A-Za-z0-9_])')


def read(name: str) -> str:
    return (ROOT / name).read_bytes().decode('utf-8')


def union_values(source: str, type_name: str) -> list[str]:
    definition = re.search(r'^export type ' + re.escape(type_name) + r' = ([^\n]+)$', source, re.MULTILINE)
    if not definition:
        raise ValueError(f'找不到路径参数类型 {type_name}')
    values = re.findall(r"'([^']+)'", definition[1])
    if not values:
        raise ValueError(f'无法提取路径参数 {type_name}')
    return values


def canonical(path: str) -> str:
    path = '/' + path.split('?', 1)[0].lstrip('/')
    path = re.sub(r'\$\{[^}]+\}', '{param}', path)
    path = re.sub(r'\{[^}]+\}', '{param}', path)
    return re.sub(r':[A-Za-z0-9_-]+', '{param}', path)


def expand(path: str, compose_targets: list[str], gateway_names: list[str]) -> list[str]:
    if path.startswith('compose/${target}/'):
        return [path.replace('${target}', target, 1) for target in compose_targets]
    if path.startswith('${name}/metrics'):
        return [path.replace('${name}', name, 1) for name in gateway_names]
    return [path]


def walk_sources(directory: Path) -> Iterator[Path]:
    for path in sorted(directory.iterdir()):
        if path.is_symlink():
            continue
        if path.is_dir():
            yield from walk_sources(path)
        elif path.is_file() and path.suffix in ('.vue', '.ts'):
            yield path


def main() -> int:
    argparse.ArgumentParser(description=__doc__).parse_args()
    api = read('webview/src/service/api.ts')
    spec = json.loads(read('public/openapi/data.json'))
    compose_targets = union_values(read('webview/src/service/types/compose.ts'), 'ComposeDeployTarget')
    gateway_names = union_values(read('webview/src/service/types/metrics.ts'), 'MetricsGatewayName')
    documented = set()
    for path, operations in spec['paths'].items():
        for method in operations:
            if method in ('get', 'post', 'put', 'patch', 'delete'):
                documented.add(method + ' ' + canonical(path))
    for match in NODE_ROUTE_PATTERN.finditer(read('server/service/node/permission.go')):
        path = re.sub(r'^/api', '', match[2], count=1)
        documented.add(match[1].lower() + ' ' + canonical(path))
    for match in COPILOT_ROUTE_PATTERN.finditer(read('server/app/ctrl_copilot.go')):
        documented.add(match[1].lower() + ' ' + canonical(match[2]))

    calls = list(REQUEST_PATTERN.finditer(api))
    if not calls:
        raise ValueError('未识别任何前端 API 调用，请检查扫描规则')
    checked = 0
    errors = 0
    for match in calls:
        for variant in expand(match[3], compose_targets, gateway_names):
            checked += 1
            route = match[1] + ' ' + canonical(variant)
            if route not in documented:
                line = api.count('\n', 0, match.start()) + 1
                print(f'webview/src/service/api.ts:{line} 缺少 API 路由定义: {route}', file=sys.stderr)
                errors += 1
    print(f'已核验 {len(calls)} 处 API 调用、{checked} 个路径变体')

    permissions_checked = 0
    for path in walk_sources(ROOT / 'webview/src'):
        source = path.read_bytes().decode('utf-8')
        for match in PERMISSION_PATTERN.finditer(source):
            permissions_checked += 1
            route = match[2].lower() + ' ' + canonical(match[3])
            if route not in documented:
                line = source.count('\n', 0, match.start()) + 1
                print(f'{path.relative_to(ROOT).as_posix()}:{line} 权限引用缺少 API 路由定义: {route}', file=sys.stderr)
                errors += 1
    print(f'已核验 {permissions_checked} 处静态页面权限引用')
    if not permissions_checked:
        raise ValueError('未发现任何静态权限引用，请检查扫描规则')
    if errors:
        print(f'存在 {errors} 处接口路径/方法不一致', file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (OSError, ValueError, KeyError) as error:
        print(f'检查失败：{error}', file=sys.stderr)
        sys.exit(1)
