#!/usr/bin/env python3
"""
后端 Go 多语言词典检查，以后端英文词典的 key 为全集。

检查 locale_utils.py 的 LANG_FILES 映射所列 server/i18n 文件，不扫描业务 Go 源码。
新增后端词典时需同步本端 LANG_FILES，以纳入检查范围。
--worklist LANG 导出缺失词条 JSONL，--strict 有问题时以非零退出。

用法（仓库根目录）：
    python3 scripts/check-go-locales.py --strict
    python3 scripts/check-go-locales.py --worklist ja
"""

import argparse
import re
import sys
from pathlib import Path

from locale_utils import LANG_FILES, check_locales, load_lang, python_command

ROOT = Path(__file__).resolve().parent.parent
VERB = re.compile(r'%[sdvwqT%]')


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--strict', action='store_true', help='有问题时以非零退出')
    parser.add_argument('--worklist', metavar='LANG', help='导出后端语言的待翻译清单')
    args = parser.parse_args()
    codes = [code for code in LANG_FILES if code != 'en']
    en = load_lang(ROOT, 'en')
    if not en:
        raise ValueError('后端英文词典未解析到词条，请检查文件格式')
    return check_locales(
        en, codes, lambda code: load_lang(ROOT, code), VERB,
        lambda code: python_command(Path(__file__).resolve(), '--worklist', code),
        strict=args.strict, worklist=args.worklist,
    )


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (OSError, ValueError) as error:
        print(f'检查失败：{error}', file=sys.stderr)
        sys.exit(1)
