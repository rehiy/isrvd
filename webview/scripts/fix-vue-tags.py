#!/usr/bin/env python3
"""
手动整理 Vue 模板中的多行开始标签；不属于 format.py 的自动修复链。

仅合并长度小于 180 的开始标签，跳过自闭合标签、注释和属性值跨行的标签。
保留引号内的空格和表达式，脚本与样式区不处理。
默认修改文件；--dry-run / --check 不修改，有差异返回 1，执行失败非零退出。
用法（webview 目录）：python3 scripts/fix-vue-tags.py --check
用法（仓库根目录）：python3 webview/scripts/fix-vue-tags.py --dry-run
"""

import argparse
from pathlib import Path
import re
import sys

ROOT = Path(__file__).resolve().parent.parent
TEMPLATE = re.compile(r'(^<template(?:\s[^>]*|)>[^\S\n]*\r?\n)([\s\S]*)(^</template>[^\S\n]*$)', re.MULTILINE)


def join_tag(lines: list[str]) -> str | None:
    parts = []
    for index, line in enumerate(lines):
        text = line.strip()
        quote = None
        for position, char in enumerate(text):
            if quote:
                if char == quote:
                    quote = None
            elif char in ('"', "'"):
                quote = char
            elif char == '<' and (index != 0 or position != 0):
                return None
            elif char == '>':
                if index != len(lines) - 1 or position != len(text) - 1 or text[:position].rstrip().endswith('/'):
                    return None
        if quote:
            return None
        parts.append(text)
    merged = ' '.join(part for part in parts if part)
    if not merged.endswith('>') or len(merged) >= 180:
        return None
    return merged


def format_template(template: str) -> str:
    lines = template.splitlines(keepends=True)
    output = []
    index = 0
    in_comment = False
    while index < len(lines):
        line = lines[index]
        if in_comment or '<!--' in line:
            in_comment = '-->' not in line
            output.append(line)
            index += 1
            continue
        start = re.match(r'^([ \t]*)<[A-Za-z][\w.-]*(?:\s|$)', line)
        if not start:
            output.append(line)
            index += 1
            continue
        quote = None
        end = index
        closed = False
        while end < len(lines):
            for char in lines[end]:
                if quote:
                    if char == quote:
                        quote = None
                elif char in ('"', "'"):
                    quote = char
                elif char == '>':
                    closed = True
                    break
            if closed:
                break
            end += 1
        if not closed:
            output.extend(lines[index:])
            break
        merged = join_tag(lines[index:end + 1]) if end > index else None
        if merged:
            newline = '\r\n' if lines[end].endswith('\r\n') else '\n' if lines[end].endswith('\n') else ''
            output.append(start[1] + merged + newline)
        else:
            output.extend(lines[index:end + 1])
        index = end + 1
    return ''.join(output)


def fix_vue_file(filepath: Path, dry_run: bool = False) -> bool:
    path = Path(filepath)
    source = path.read_bytes().decode('utf-8')
    match = TEMPLATE.search(source)
    if not match:
        return False
    template = format_template(match[2])
    result = source[:match.start(2)] + template + source[match.end(2):]
    if result == source:
        return False
    if not dry_run:
        path.write_bytes(result.encode('utf-8'))
    return True


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument('--dry-run', '--check', dest='dry_run', action='store_true', help='只检查，不写入；存在差异返回 1')
    args = parser.parse_args()
    directory = ROOT / 'src'
    if not directory.is_dir():
        raise ValueError(f'扫描目录不存在：{directory}')
    files = sorted(directory.rglob('*.vue'))
    if not files:
        raise ValueError('未找到 Vue 文件，未执行修复')
    changed = 0
    for path in files:
        if fix_vue_file(path, args.dry_run):
            changed += 1
            print(f"{'[dry]' if args.dry_run else '[fix]'} {path.relative_to(ROOT)}")
    print(f"共处理 {len(files)} 个文件，{'需要' if args.dry_run else '已'}修改 {changed} 个。")
    return 1 if args.dry_run and changed else 0


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (OSError, ValueError) as error:
        print(f'Vue 标签整理失败：{error}', file=sys.stderr)
        sys.exit(1)
