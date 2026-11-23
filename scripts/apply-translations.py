#!/usr/bin/env python3
"""
把译文并回后端 Go 语言文件，保留既有词条顺序并运行 gofmt。

输入为 JSONL，每行形如：{"key":"中文原文","translation":"译文","en":"English"}。
key 与译文必填；优先使用 translation，缺少该字段时兼容旧 ja 字段。
空白译文的行会跳过，非法类型或换行会报错；目标优先按本端 LANG_FILES 映射。
不接受源语言、英文基准或繁体生成物；全部 key 未命中时失败且不写入。
相对 --file 路径按调用者当前目录解释；需要 Go 的 gofmt，除此之外仅使用标准库。
写完请运行 scripts/check-go-locales.py 复核，不读取或修改前端词典。

用法（仓库根目录）：
    python3 scripts/apply-translations.py --lang ja --file tmp/ja-api.jsonl
"""

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path

from locale_utils import GO_ENTRY, LANG_FILES, compile_pattern, load_lang, read_entries, read_translations

ROOT = Path(__file__).resolve().parent.parent
GO_MAP = compile_pattern(r'(Register\((?:[A-Za-z0-9_.]+|Language\{.*?\})\s*,\s*map\[string\]string\{\s*\n)(.*)(\n\s*\}\))', re.DOTALL)


def json_quote(value: str) -> str:
    return json.dumps(value, ensure_ascii=False, separators=(',', ':'))


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

    directory = ROOT / 'server/i18n'
    en = load_lang(ROOT, 'en')
    if not en:
        raise ValueError('后端英文词典未解析到词条，未写入目标词典')
    names = LANG_FILES.get(args.lang, [f'lang_{args.lang}.go'])
    if len(names) != 1:
        raise ValueError('翻译导入仅支持单文件目标词典')
    path = directory / names[0]
    if path.resolve().parent != directory.resolve():
        raise ValueError('目标词典必须位于指定语言目录内')
    source = path.read_bytes().decode('utf-8')
    current = read_entries(path, GO_ENTRY)
    unknown = 0
    for key, value in incoming:
        if key not in en:
            print(f'  ! 英文词典里没有「{key}」，已跳过', file=sys.stderr)
            unknown += 1
            continue
        current[key] = value
    if unknown == len(incoming):
        raise ValueError('全部译文 key 都不在英文词典中，未写入目标词典')

    body = '\n'.join(f'\t\t{json_quote(key)}: {json_quote(value)},' for key, value in current.items())
    out, matches = GO_MAP.subn(lambda match: match[1] + body + match[3], source, count=1)
    if not matches:
        raise ValueError('未找到后端 Register 词典 map')
    try:
        out = subprocess.run(['gofmt'], input=out, encoding='utf-8', capture_output=True, check=True).stdout
    except (OSError, subprocess.CalledProcessError) as error:
        detail = error.stderr if isinstance(error, subprocess.CalledProcessError) else str(error)
        raise ValueError(f'gofmt 失败，未写入目标词典：{detail.strip()}') from error
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
