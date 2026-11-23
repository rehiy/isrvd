#!/usr/bin/env python3
"""
前端繁體中文（臺灣）词条生成器。

以前端英文词典的 key（简体中文原文）为全集，经本地术语表、上下文规则和 OpenCC
字形转换生成前端 zh-Hant.ts，不读取或修改后端词典，不需要 Go 或 gofmt。
需要调整用词时改 webview/scripts/locale_utils.py 的 TERMS / RULES，然后重新执行；
生成物不要手工编辑。

依赖（仓库根目录）：python3 -m pip install -r webview/scripts/requirements.txt（建议在虚拟环境中执行）
用法（仓库根目录）：python3 webview/scripts/gen-zh-hant.py [--check]
    --check 只校验前端生成物是否是最新，不写文件。
"""

import argparse
import sys
from pathlib import Path

from locale_utils import JS_WHITESPACE, compile_pattern, make_converter, python_command, quote_ts, script_command, unquote

ROOT = Path(__file__).resolve().parent.parent
TS_ENTRY = compile_pattern(r'''^\s*('(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*")\s*:\s*('(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*")\s*,?\s*$''')
SECTION = compile_pattern(r'^\s*// ─── ')


def read_items(path: Path) -> list[dict[str, str]]:
    items = []
    for line in path.read_bytes().decode('utf-8').split('\n'):
        match = TS_ENTRY.match(line)
        if match:
            items.append({'key': unquote(match[1]), 'en': unquote(match[2])})
        elif SECTION.match(line):
            items.append({'section': line.strip(JS_WHITESPACE)})
    return items


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--check', action='store_true', help='校验前端生成物，不写文件')
    args = parser.parse_args()
    try:
        convert = make_converter()
    except ImportError:
        requirements = Path(__file__).resolve().with_name('requirements.txt')
        command = python_command('-m', 'pip', 'install', '-r', requirements)
        print(f'未找到 Python OpenCC，请在虚拟环境中执行：{command}', file=sys.stderr)
        return 1

    frontend = read_items(ROOT / 'src/locales/messages/en.ts')
    count_frontend = sum('key' in item for item in frontend)
    if not count_frontend:
        print('未能解析出词条，请检查英文词典的格式', file=sys.stderr)
        return 1
    frontend_body = '\n'.join(
        f"\n    {item['section']}" if 'section' in item else
        f"    {quote_ts(item['key'])}: {quote_ts(convert(item['key']))},"
        for item in frontend
    )
    frontend_ts = """import type { LanguageModule } from '../types'

/**
 * 繁體中文（臺灣）譯文：以簡體中文原文為 key
 *
 * 由 webview/scripts/gen-zh-hant.py 從英文詞典的 key 生成（術語表 + 字形轉換），請勿手工編輯；
 * 需要調整用詞時改 webview/scripts/locale_utils.py 的 TERMS / RULES 後重新執行前端生成器。
 */

const messages: Record<string, string> = {
""" + frontend_body + """
}

const language: LanguageModule = {
    code: 'zh-Hant',
    label: '繁體中文',
    short: '繁',
    htmlLang: 'zh-TW',
    tags: ['zh-hant', 'zh-tw', 'zh-hk', 'zh-mo'],
    messages
}

export default language
"""
    path = ROOT / 'src/locales/messages/zh-Hant.ts'
    stale = False
    if args.check:
        try:
            current = path.read_bytes().decode('utf-8')
        except OSError:
            current = ''
        if current != frontend_ts:
            print(f'过期：{path.relative_to(ROOT).as_posix()}', file=sys.stderr)
            stale = True
    else:
        path.write_bytes(frontend_ts.encode('utf-8'))
        print(f'已生成 {path.relative_to(ROOT).as_posix()}')
    if args.check:
        if stale:
            command = script_command('gen-zh-hant.py')
            print(f'请执行 {command} 重新生成', file=sys.stderr)
            return 1
        print('前端繁體词条是最新的')
    else:
        print(f'前端 {count_frontend} 条')
    return 0


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (OSError, ValueError) as error:
        print(f'生成失败：{error}', file=sys.stderr)
        sys.exit(1)
