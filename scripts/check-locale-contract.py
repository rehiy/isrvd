#!/usr/bin/env python3
"""
核对前后端独立词典工具的基础语义契约，不比较本端业务术语和语言例外。

只加载两个本地辅助模块，使用内存数据核对空白、JSON 解码、译文协议及报告。
不读取业务词典，不调用生成器、OpenCC 或 gofmt，不修改文件。
用法（仓库根目录）：python3 scripts/check-locale-contract.py
"""

import argparse
from contextlib import redirect_stderr, redirect_stdout
import importlib.util
import io
import json
from pathlib import Path
import re
import shlex
import sys

sys.dont_write_bytecode = True
ROOT = Path(__file__).resolve().parent.parent


def require(condition: bool, message: str) -> None:
    if not condition:
        raise ValueError(message)


def load_module(name: str, path: Path):
    spec = importlib.util.spec_from_file_location(name, path)
    if spec is None or spec.loader is None:
        raise ValueError(f'无法加载辅助模块：{path}')
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def check_basic(module) -> None:
    whitespace = ('\t\n\v\f\r \u00a0\u1680' + ''.join(chr(code) for code in range(0x2000, 0x200b))
                  + '\u2028\u2029\u202f\u205f\u3000\ufeff')
    require(set(module.JS_WHITESPACE) == set(whitespace), 'ECMAScript 空白集合不符合契约')
    pattern = module.compile_pattern(r'^\s$', re.MULTILINE)
    require(pattern.flags & re.MULTILINE, '正则 flags 未保留')
    require(all(pattern.fullmatch(char) for char in whitespace), '合法空白未匹配')
    require(not any(pattern.fullmatch(char) for char in ('\x85', '\x1c', '\x1d', '\x1e', '\x1f')), '额外空白被错误匹配')
    for value in ('中文', 'Quote " and \\ slash', '\t\n', '𠮷'):
        require(module.unquote(json.dumps(value, ensure_ascii=False)) == value, 'JSON 字符串解码不一致')
    if hasattr(module, 'quote_ts'):
        for value in ('中文', "Quote ' and \\ slash", '\t\n\v\0', '𠮷', '\u2028\u2029'):
            require(module.unquote(module.quote_ts(value)) == value, '前端 TS 字符串往返不一致')
        for raw, value in ((r"'\u4e2d'", '中'), (r"'\x41'", 'A'), (r"'\u{1f600}'", '😀'), (r"'\ud83d\ude00'", '😀')):
            require(module.unquote(raw) == value, '前端 Unicode 字符串解码不一致')
    args = ('space dir/tool.py', "quote'argument", 'semi;colon', '--worklist', 'ja')
    require(shlex.split(module.python_command(*args)) == [sys.executable, *args], '恢复命令参数未安全引用')


def check_translation(module) -> None:
    cases = [
        ({'key': '原文', 'translation': 'New'}, ('原文', 'New')),
        ({'key': '原文', 'ja': 'Legacy'}, ('原文', 'Legacy')),
        ({'key': '原文', 'translation': 'New', 'ja': 'Legacy'}, ('原文', 'New')),
        ({'key': '原文', 'translation': '', 'ja': 'Legacy'}, None),
        ({'key': '原文', 'translation': '\ufeff', 'ja': 'Legacy'}, None),
        ({'key': '', 'translation': 'New'}, None),
        ({'key': '\ufeff', 'translation': 'New'}, None),
        ({'key': '原文'}, None),
        ({'translation': 'New'}, None),
        ({'key': '原文', 'translation': '  New  '}, ('原文', '  New  ')),
    ]
    for entry, expected in cases:
        require(module.normalize_translation(entry, 7) == expected, f'译文字段选择或空值规则不一致：{entry!r}')
    invalid = [
        [], {'key': 0, 'translation': 'New'}, {'key': '原文', 'translation': None, 'ja': 'Legacy'},
        {'key': '原文', 'translation': False}, {'key': '原文', 'ja': []},
        *({'key': '原文', 'translation': 'line' + char + 'break'} for char in ('\n', '\r', '\u2028', '\u2029')),
    ]
    for entry in invalid:
        try:
            module.normalize_translation(entry, 7)
        except ValueError as error:
            require('第 7 行' in str(error), '译文错误缺少行号')
        else:
            raise ValueError(f'非法译文未被拒绝：{entry!r}')


class MemoryJSONL:
    def __init__(self, content: str):
        self.content = content

    def read_bytes(self) -> bytes:
        return self.content.encode('utf-8')


def check_reader(module) -> None:
    entries = [
        {'key': '原文', 'translation': 'New'},
        {'key': '原文', 'ja': 'Legacy'},
        {'key': '跳过', 'translation': '\ufeff', 'ja': 'No fallback'},
    ]
    content = '\ufeff\r\n \r\n' + '\r\n'.join(json.dumps(entry, ensure_ascii=False) for entry in entries) + '\r\n'
    require(module.read_translations(MemoryJSONL(content)) == [('原文', 'New'), ('原文', 'Legacy')], 'JSONL 空行、顺序或规范化不符合契约')
    for entry in ({'key': '原文', 'translation': 0}, {'key': '原文', 'translation': 'line\nbreak'}):
        try:
            module.read_translations(MemoryJSONL('\ufeff\n\n' + json.dumps(entry, ensure_ascii=False)))
        except ValueError as error:
            require('第 3 行' in str(error), 'JSONL 错误行号不符合契约')
        else:
            raise ValueError('JSONL 读取器未校验译文')
    try:
        module.read_translations(MemoryJSONL('not json'))
    except json.JSONDecodeError:
        pass
    else:
        raise ValueError('JSONL 读取器未拒绝非法 JSON')


def report(module, strict: bool = False, worklist: str = None):
    en = {'缺失': 'Missing', '空值': 'Empty', '占位 {0}': 'Value {0}'}
    languages = {'ja': {'空值': '\ufeff', '占位 {0}': 'Value {1}', '过期': 'Stale'}}
    stdout, stderr = io.StringIO(), io.StringIO()
    with redirect_stdout(stdout), redirect_stderr(stderr):
        status = module.check_locales(
            en, ['ja'], languages.__getitem__, re.compile(r'\{([0-9]+)\}'),
            lambda code: f'export {code}', strict=strict, worklist=worklist,
        )
    return status, stdout.getvalue(), stderr.getvalue()


def main() -> int:
    argparse.ArgumentParser(description=__doc__).parse_args()
    modules = [
        load_module('backend_locale_contract', ROOT / 'scripts/locale_utils.py'),
        load_module('frontend_locale_contract', ROOT / 'webview/scripts/locale_utils.py'),
    ]
    for module in modules:
        check_basic(module)
        check_translation(module)
        check_reader(module)
        for strict, worklist in ((False, None), (True, None), (False, 'ja'), (False, 'invalid')):
            status, stdout, stderr = report(module, strict, worklist)
            if worklist == 'ja':
                require(status == 0 and json.loads(stdout) == {'key': '缺失', 'en': 'Missing'} and not stderr, 'JSONL 清单不符合契约')
            elif worklist:
                require(status == 1 and not stdout and '没有语言 invalid' in stderr, '无效语言的退出行为不符合契约')
            else:
                require(status == int(strict) and not stderr, 'strict 退出行为不符合契约')
                for text in ('缺失 1 条', '多余 1 条', '空译文 1 条', '占位符与原文不一致 1 条'):
                    require(text in stdout, f'检查遗漏：{text}')
    print('前后端词典基础语义契约一致（业务规则独立维护）')
    return 0


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (OSError, ValueError, ImportError, AttributeError, TypeError) as error:
        print(f'词典基础语义契约检查失败：{error}', file=sys.stderr)
        sys.exit(1)
