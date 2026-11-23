#!/usr/bin/env python3
"""
前端多语言词条检查，以前端英文词典的 key 为全集。

仅检查 locales/messages/*.ts，不读取或调用后端词典工具。
检查缺词、多余词条、空译文、占位符不一致及可能未翻译的词条。
--worklist 导出待翻译清单（JSONL），--strict 有缺口时以非零退出。

用法（仓库根目录）：
    python3 webview/scripts/check-locales.py
    python3 webview/scripts/check-locales.py --worklist ja
    python3 webview/scripts/check-locales.py --strict
"""

import argparse
import re
import sys
from pathlib import Path

from locale_utils import check_locales, compile_pattern, read_entries, script_command

ROOT = Path(__file__).resolve().parent.parent
ENTRY = compile_pattern(r'''^\s*('(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*")\s*:\s*('(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*")\s*,?\s*$''')
PLACEHOLDER = re.compile(r'\{([0-9]+)\}')


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--strict', action='store_true', help='有缺口时以非零退出')
    parser.add_argument('--worklist', metavar='LANG', help='导出指定语言的待翻译清单')
    args = parser.parse_args()

    directory = ROOT / 'src/locales/messages'
    files = sorted(path.name for path in directory.iterdir())
    if 'en.ts' not in files:
        print('缺少 locales/messages/en.ts，英文词典是完整集', file=sys.stderr)
        return 1

    def load_lang(code: str) -> dict[str, str]:
        return read_entries(directory / f'{code}.ts', ENTRY)

    codes = [name[:-3] for name in files if name.endswith('.ts')]
    en = load_lang('en')
    if not en:
        raise ValueError('前端英文词典未解析到词条，请检查文件格式')
    return check_locales(
        en, codes, load_lang, PLACEHOLDER,
        lambda code: script_command('check-locales.py', '--worklist', code),
        strict=args.strict, worklist=args.worklist,
    )


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (OSError, ValueError) as error:
        print(f'检查失败：{error}', file=sys.stderr)
        sys.exit(1)
