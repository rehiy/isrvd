#!/usr/bin/env python3
"""
后端 Go 繁体词典生成与生成物检查。

用法（仓库根目录）：python3 scripts/gen-zh-hant.py [--check]
--check 只检查后端生成物，不读取或写入前端词典。
依赖：python3 -m pip install -r scripts/requirements.txt，并安装 Go（需要 gofmt）。
"""

import argparse
import json
import subprocess
import sys
from pathlib import Path
from typing import Callable

from locale_utils import LANG_FILES, make_converter, python_command, read_items

ROOT = Path(__file__).resolve().parent.parent


def build_backend(root: Path, convert: Callable[[str], str]) -> tuple[Path, str, int]:
    items = []
    for name in LANG_FILES['en']:
        items.extend(read_items(root / 'server/i18n' / name))
    if not any('key' in item for item in items):
        raise ValueError('未能解析出词条，请检查英文词典的格式')
    seen = set()
    lines = []
    for item in items:
        if 'section' in item:
            lines.append(f"\n\t\t{item['section']}")
        elif item['key'] not in seen:
            seen.add(item['key'])
            key = json.dumps(item['key'], ensure_ascii=False, separators=(',', ':'))
            value = json.dumps(convert(item['key']), ensure_ascii=False, separators=(',', ':'))
            lines.append(f'\t\t{key}: {value},')
    source = """package i18n

// 繁體中文（臺灣）：中文原文 → 繁體译文。
//
// 本文件由 scripts/gen-zh-hant.py 从英文词典的 key 生成（术语表 + 字形转换），
// 术语及上下文规则维护在 scripts/locale_utils.py 的 TERMS / RULES；请勿手工编辑，
// 需要调整用词时改后端规则后通过 scripts/gen-zh-hant.py 重新生成后端词典。

func init() {
\tRegister(Language{Code: "zh-Hant", Tags: []string{"zh-hant", "zh-tw", "zh-hk", "zh-mo"}}, map[string]string{
""" + '\n'.join(lines) + """
\t})
}
"""
    try:
        source = subprocess.run(['gofmt'], input=source, encoding='utf-8', capture_output=True, check=True).stdout
    except (OSError, subprocess.CalledProcessError) as error:
        raise ValueError('未找到 gofmt（或生成的 Go 源码有语法错误），请确认已安装 Go') from error
    return root / 'server/i18n/lang_zh_hant_messages.go', source, len(seen)


def process_backend(root: Path, path: Path, content: str, check: bool) -> bool:
    if check:
        try:
            current = path.read_bytes().decode('utf-8')
        except OSError:
            current = ''
        if current != content:
            print(f'过期：{path.relative_to(root).as_posix()}', file=sys.stderr)
            return True
    else:
        path.write_bytes(content.encode('utf-8'))
        print(f'已生成 {path.relative_to(root).as_posix()}')
    return False


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--check', action='store_true', help='只读校验后端繁体词典')
    args = parser.parse_args()
    try:
        convert = make_converter()
    except ImportError:
        command = python_command('-m', 'pip', 'install', '-r', Path(__file__).resolve().with_name('requirements.txt'))
        print(f'未找到 Python OpenCC，请在虚拟环境中执行：{command}', file=sys.stderr)
        return 1
    path, content, count = build_backend(ROOT, convert)
    if process_backend(ROOT, path, content, args.check):
        print(f'请执行 {python_command(Path(__file__).resolve())} 重新生成', file=sys.stderr)
        return 1
    print('后端繁體词条是最新的' if args.check else f'后端 {count} 条')
    return 0


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (OSError, ValueError) as error:
        print(f'生成失败：{error}', file=sys.stderr)
        sys.exit(1)
