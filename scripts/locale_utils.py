#!/usr/bin/env python3
"""后端 Go 词典工具的解析、导入协议、完整性检查及繁体转换规则。

基础语义由 check-locale-contract.py 核对；本端术语和语言例外独立维护。
"""

import json
import math
import re
import shlex
import sys
from pathlib import Path
from typing import Callable

JS_WHITESPACE = (
    '\t\n\v\f\r \u00a0\u1680\u2000\u2001\u2002\u2003\u2004\u2005'
    '\u2006\u2007\u2008\u2009\u200a\u2028\u2029\u202f\u205f\u3000\ufeff'
)
JA_IDENTICAL = {
    '使用率: ', '使用率:', '操作', '匿名', '保存中...', '成功', '可用性', '(空)',
    '50 行', '100 行', '200 行', '500 行', '1000 行', 'CPU 使用率', '使用率',
}
TERMS = sorted([
    ('内存', '記憶體'), ('信息', '資訊'), ('进程', '行程'), ('账号', '帳號'),
    ('账户', '帳戶'), ('阈值', '臨界值'), ('缺省', '預設'), ('默认', '預設'),
    ('计划任务', '計劃任務'), ('镜像仓库', '鏡像倉庫'),
], key=lambda item: -len(item[0]))
RULES = [
    (re.compile(r'获取(?=[\u4e00-\u9fff])(?!列)'), '取得'),
    (re.compile(r'^直接运行$'), '直接執行'),
    (re.compile(r'^Docker 运行$'), '使用 Docker 執行'),
]


def compile_pattern(pattern: str, flags: int = 0) -> re.Pattern:
    whitespace = '[' + re.escape(JS_WHITESPACE) + ']'
    return re.compile(pattern.replace(r'\s', whitespace), flags)


def python_command(*args: object) -> str:
    return shlex.join([sys.executable, *(str(arg) for arg in args)])


GO_ENTRY = compile_pattern(r'^\s*("(?:[^"\\]|\\.)*")\s*:\s*("(?:[^"\\]|\\.)*"),\s*$')
SECTION = compile_pattern(r'^\s*// ─── ')
LANG_FILES = {
    'en': ['lang_en.go', 'lang_en_service.go'],
    'ja': ['lang_ja.go'],
    'zh-Hant': ['lang_zh_hant_messages.go'],
}


def unquote(raw: str) -> str:
    return json.loads(raw)


def read_entries(path: Path, pattern: re.Pattern) -> dict[str, str]:
    entries = {}
    for line in path.read_bytes().decode('utf-8').split('\n'):
        match = pattern.match(line)
        if match:
            entries[unquote(match[1])] = unquote(match[2])
    return entries


def load_lang(root: Path, code: str) -> dict[str, str]:
    entries = {}
    for name in LANG_FILES.get(code, []):
        entries.update(read_entries(root / 'server/i18n' / name, GO_ENTRY))
    return entries


def read_items(path: Path) -> list[dict[str, str]]:
    items = []
    for line in path.read_bytes().decode('utf-8').split('\n'):
        match = GO_ENTRY.match(line)
        if match:
            items.append({'key': unquote(match[1]), 'en': unquote(match[2])})
        elif SECTION.match(line):
            items.append({'section': line.strip(JS_WHITESPACE)})
    return items


def normalize_translation(entry: object, number: int) -> tuple[str, str] | None:
    if not isinstance(entry, dict):
        raise ValueError(f'第 {number} 行必须是 JSON 对象')
    if 'key' not in entry or not ('translation' in entry or 'ja' in entry):
        return None
    key = entry['key']
    value = entry['translation'] if 'translation' in entry else entry['ja']
    if not isinstance(key, str) or not isinstance(value, str):
        raise ValueError(f'第 {number} 行的 key 与译文必须是字符串')
    if any(char in text for text in (key, value) for char in ('\n', '\r', '\u2028', '\u2029')):
        raise ValueError(f'第 {number} 行的 key 与译文不能包含换行符')
    if not key.strip(JS_WHITESPACE) or not value.strip(JS_WHITESPACE):
        return None
    return key, value


def read_translations(path: Path) -> list[tuple[str, str]]:
    entries = []
    for number, line in enumerate(path.read_bytes().decode('utf-8').split('\n'), 1):
        if not line.strip(JS_WHITESPACE):
            continue
        entry = normalize_translation(json.loads(line), number)
        if entry is not None:
            entries.append(entry)
    return entries


def check_locales(
    en: dict[str, str],
    codes: list[str],
    load_lang: Callable[[str], dict[str, str]],
    placeholder: re.Pattern,
    worklist_command: Callable[[str], str],
    strict: bool = False,
    worklist: str = None,
) -> int:
    if worklist:
        if worklist not in codes:
            print(f"没有语言 {worklist}，现有：{', '.join(codes)}", file=sys.stderr)
            return 1
        target = load_lang(worklist)
        for key, value in en.items():
            if key not in target:
                print(json.dumps({'key': key, 'en': value}, ensure_ascii=False, separators=(',', ':')))
        return 0

    problems = 0

    def report(text: str) -> None:
        nonlocal problems
        print(text)
        problems += 1

    for code in sorted(codes):
        entries = load_lang(code)
        missing = [key for key in en if key not in entries]
        stale = [key for key in entries if key not in en]
        empty = [key for key, value in entries.items() if not value.strip(JS_WHITESPACE)]
        bad_placeholder = [(key, value) for key, value in entries.items()
                           if placeholder.findall(key) != placeholder.findall(value)]
        identity = [key for key, value in entries.items()
                    if key == value and re.search(r'[一-鿿]', key)
                    and (code != 'ja' or key not in JA_IDENTICAL)]
        ratio = math.floor((len(en) - len(missing)) / len(en) * 100 + 0.5) if en else 100
        print(f'{code:<8} {len(entries):>5} 条，覆盖 {ratio:>3}%，缺失 {len(missing)}')
        if code == 'en':
            continue
        if missing:
            report(f'  ✗ 缺失 {len(missing)} 条，导出清单：{worklist_command(code)}')
        if stale:
            report(f'  ✗ 多余 {len(stale)} 条（英文词典里已无此原文，多为改文案后遗留）')
            for key in stale[:8]:
                report(f'      {key}')
        if empty:
            report(f'  ✗ 空译文 {len(empty)} 条')
            for key in empty[:8]:
                report(f'      {key}')
        if bad_placeholder:
            report(f'  ✗ 占位符与原文不一致 {len(bad_placeholder)} 条')
            for key, value in bad_placeholder[:8]:
                report(f'      {key} => {value}')
        if identity and code != 'zh-Hant':
            report(f'  ! 译文与中文原文完全相同 {len(identity)} 条，需人工确认是否已翻译')
            for key in identity[:8]:
                report(f'      {key}')
    print(f'\n共 {problems} 处待处理' if problems else '\n全部语言词条完整')
    return 1 if strict and problems else 0


def make_converter() -> Callable[[str], str]:
    from opencc import OpenCC

    converter = OpenCC('s2tw')

    def convert(text: str) -> str:
        for cn, tw in TERMS:
            text = text.replace(cn, tw)
        for pattern, tw in RULES:
            text = pattern.sub(tw, text)
        return converter.convert(text).replace('籤發', '簽發')

    return convert
