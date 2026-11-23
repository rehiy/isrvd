#!/usr/bin/env python3
"""
把译文并回前端语言文件（保留英文注释、按英文词典顺序排列）。

输入为 JSONL，每行形如：{"key":"中文原文","translation":"译文","en":"English"}。
key 与译文必填；优先使用 translation，缺少该字段时兼容旧 ja 字段。
空白译文的行会跳过，非法类型或换行会报错；en 注释由前端英文词典提供。
仅更新既有的 <lang>.ts，不读取或修改后端词典；不接受源语言、英文基准和繁体生成物。
全部 key 未命中、英文基准为空或语言元信息不完整时失败且不写入。
相对 --file 路径按调用者当前目录解释；仅使用 Python 标准库。
写完请运行 webview/scripts/check-locales.py 复核。

用法（仓库根目录）：
    python3 webview/scripts/apply-translations.py --lang ja --file tmp/ja-ui.jsonl
"""

import argparse
import re
import sys
from pathlib import Path

from locale_utils import JS_WHITESPACE, compile_pattern, language_metadata, quote_ts as quote, read_entries, read_translations

ROOT = Path(__file__).resolve().parent.parent
TS_ENTRY = compile_pattern(r'''^\s*('(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*")\s*:\s*('(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*")\s*,?\s*$''')


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--lang', required=True, help='目标语言代码')
    parser.add_argument('--file', required=True, type=Path, help='译文 JSONL 文件')
    args = parser.parse_args()
    if not re.fullmatch(r'[A-Za-z][A-Za-z0-9]*(?:[-_][A-Za-z0-9]+)*', args.lang):
        parser.error('语言代码只能包含字母、数字、连字符和下划线')
    if args.lang in ('en', 'zh-Hans', 'zh-Hant'):
        parser.error('源语言、英文基准和繁体生成物不能通过翻译导入器修改')

    incoming = read_translations(args.file)
    if not incoming:
        print('没有可导入的译文（key 与译文都必填，且译文不能为空）', file=sys.stderr)
        return 1

    directory = ROOT / 'src/locales/messages'
    en = read_entries(directory / 'en.ts', TS_ENTRY)
    if not en:
        raise ValueError('前端英文词典未解析到词条，未写入目标词典')
    path = directory / f'{args.lang}.ts'
    if path.resolve().parent != directory.resolve():
        raise ValueError('目标词典必须位于指定语言目录内')
    source = path.read_bytes().decode('utf-8')
    current = read_entries(path, TS_ENTRY)
    unknown = 0
    for key, value in incoming:
        if key not in en:
            print(f'  ! 英文词典里没有「{key}」，已跳过', file=sys.stderr)
            unknown += 1
            continue
        current[key] = value
    if unknown == len(incoming):
        raise ValueError('全部译文 key 都不在英文词典中，未写入目标词典')

    lines = []
    for key, value in en.items():
        if key in current:
            lines.append(f'    // {value}'.rstrip(JS_WHITESPACE))
            lines.append(f'    {quote(key)}: {quote(current[key])},')
    start = source.find('const messages:')
    if start < 0:
        raise ValueError('未找到前端 messages 定义')
    meta = language_metadata(source)
    body = '\n'.join(lines)
    out = f"{source[:start]}const messages: Record<string, string> = {{\n{body}\n}}\n\n{meta}"
    path.write_bytes(out.encode('utf-8'))
    skipped = f'，跳过 {unknown} 条' if unknown else ''
    print(f'已写入 {path.relative_to(ROOT).as_posix()}，共 {len(current)} 条{skipped}')
    return 0


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (OSError, ValueError) as error:
        print(f'导入失败：{error}', file=sys.stderr)
        sys.exit(1)
