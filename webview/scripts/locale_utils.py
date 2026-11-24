#!/usr/bin/env python3
"""前端词典工具的解析、导入协议、完整性检查及繁体转换规则。

基础语义由根目录 check-locale-contract.py 核对；本端术语和语言例外独立维护。
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
    ('非活跃文件缓存', '非活躍檔案快取'),
], key=lambda item: -len(item[0]))
RULES = [
    (re.compile(r'获取(?=[\u4e00-\u9fff])(?!列)'), '取得'),
    (re.compile(r'^直接运行$'), '直接執行'),
    (re.compile(r'^Docker 运行$'), '使用 Docker 執行'),
    (re.compile(r'逻辑核(?!心)'), '邏輯核心'),
    (re.compile(r'(?<=按占用)字节(?=排序)'), '位元組'),
]


def compile_pattern(pattern: str, flags: int = 0) -> re.Pattern:
    whitespace = '[' + re.escape(JS_WHITESPACE) + ']'
    return re.compile(pattern.replace(r'\s', whitespace), flags)


def python_command(*args: object) -> str:
    return shlex.join([sys.executable, *(str(arg) for arg in args)])


def script_command(script: str, *args: object) -> str:
    return python_command(Path(__file__).resolve().parent / script, *args)


def unquote(raw: str) -> str:
    if len(raw) < 2 or raw[0] not in ("'", '"') or raw[-1] != raw[0]:
        raise ValueError('无法识别词典字符串字面量')
    text = raw[1:-1]
    escapes = {'b': '\b', 'f': '\f', 'n': '\n', 'r': '\r', 't': '\t', 'v': '\v', '0': '\0'}
    result = []
    index = 0
    while index < len(text):
        char = text[index]
        if char != '\\':
            result.append(char)
            index += 1
            continue
        index += 1
        if index >= len(text):
            raise ValueError('词典字符串包含不完整转义')
        char = text[index]
        if char in ('x', 'u'):
            if char == 'u' and text[index + 1:index + 2] == '{':
                end = text.find('}', index + 2)
                digits = text[index + 2:end] if end >= 0 else ''
            else:
                size = 2 if char == 'x' else 4
                end = index + size
                digits = text[index + 1:end + 1]
                if len(digits) != size:
                    raise ValueError('词典字符串包含不完整 Unicode 转义')
            if not re.fullmatch(r'[0-9a-fA-F]+', digits) or int(digits, 16) > 0x10ffff:
                raise ValueError('词典字符串包含无效 Unicode 转义')
            result.append(chr(int(digits, 16)))
            index = end + 1
            continue
        if char in '123456789' or (char == '0' and text[index + 1:index + 2].isdigit()):
            raise ValueError('词典字符串不支持旧式数字转义')
        result.append(escapes.get(char, char))
        index += 1
    return ''.join(result).encode('utf-16-le', 'surrogatepass').decode('utf-16-le', 'surrogatepass')


def quote_ts(value: str) -> str:
    text = json.dumps(value, ensure_ascii=False)[1:-1]
    text = text.replace('\\"', '"').replace("'", "\\'").replace('\u2028', '\\u2028').replace('\u2029', '\\u2029')
    return "'" + text + "'"


def read_entries(path: Path, pattern: re.Pattern) -> dict[str, str]:
    entries = {}
    for line in path.read_bytes().decode('utf-8').split('\n'):
        match = pattern.match(line)
        if match:
            entries[unquote(match[1])] = unquote(match[2])
    return entries


def language_metadata(source: str) -> str:
    tokens = re.compile(r'''//[^\n]*|/\*.*?\*/|'(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*"|`(?:[^`\\]|\\.)*`''', re.DOTALL)

    def mask(match: re.Match) -> str:
        text = match[0]
        blank = ''.join('\n' if char == '\n' else ' ' for char in text)
        return blank if text.startswith('/') else '0' + blank[1:]

    masked = tokens.sub(mask, source)
    declaration = re.search(r'^\s*const\s+language\s*:\s*LanguageModule\s*=\s*\{', masked, re.MULTILINE)
    if declaration is None:
        raise ValueError('未找到语言元信息定义，未写入目标词典')
    opening = masked.find('{', declaration.start())
    stack = []
    fields = set()
    index = opening
    end = None
    pairs = {')': '(', ']': '[', '}': '{'}
    while index < len(masked):
        char = masked[index]
        if char in '([{':
            stack.append(char)
        elif char in ')]}':
            if not stack or stack.pop() != pairs[char]:
                raise ValueError('语言元信息括号不完整，未写入目标词典')
            if not stack:
                end = index + 1
                break
        if stack == ['{'] and char in '{,':
            field = re.match(r'\s*([A-Za-z_$][\w$]*)\s*(?::|(?=[,}]))', masked[index + 1:])
            if field:
                fields.add(field[1])
        index += 1
    required = {'code', 'label', 'short', 'htmlLang', 'tags', 'messages'}
    if end is None or not required.issubset(fields):
        raise ValueError('语言元信息缺少闭合括号或必需字段，未写入目标词典')
    tail = masked[end:]
    if not re.fullmatch(r'\s*;?\s*export\s+default\s+language\s*;?\s*', tail):
        raise ValueError('未找到真实默认导出，未写入目标词典')
    start = masked.find('const', declaration.start())
    return source[start:]


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
