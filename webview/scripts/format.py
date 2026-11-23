#!/usr/bin/env python3
"""
前端格式化入口：Vue import 排序、ESLint 及 TypeScript 类型检查。

默认修改 import 和 ESLint 可修复项，再检查类型；--dry-run 只检查，发现问题返回 1。
--check 仅表示类型检查，不是格式检查；多个选择参数可组合，--all 不可与选择参数混用。
仅只读任务允许并行；所有子命令均在前端目录执行，使用本地 node_modules 工具。
依赖：Node.js、Python 3.10+，先在 webview 目录执行 npm ci。

用法（webview 目录）：
    npm run format
    npm run format:check
    python3 scripts/format.py --check --eslint
用法（仓库根目录）：python3 webview/scripts/format.py --dry-run
"""

import argparse
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path
import shutil
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parent.parent


def run_command(command: list[str], capture: bool = True) -> bool:
    try:
        result = subprocess.run(command, cwd=ROOT, capture_output=capture, encoding='utf-8', timeout=300)
    except (OSError, subprocess.TimeoutExpired) as error:
        print(f'命令执行失败：{error}', file=sys.stderr)
        return False
    if capture:
        if result.stdout:
            print(result.stdout, end='')
        if result.stderr:
            print(result.stderr, end='', file=sys.stderr)
    return result.returncode == 0


def run_node_tool(package: str, entrypoint: str, *args: str) -> bool:
    node = shutil.which('node')
    tool = ROOT / 'node_modules' / package / entrypoint
    if node is None:
        print('未找到 Node.js，请先安装 Node.js', file=sys.stderr)
        return False
    if not tool.is_file():
        print(f'未找到本地 {package}，请在 {ROOT} 执行 npm ci', file=sys.stderr)
        return False
    return run_command([node, str(tool), *args])


def run_typescript_check() -> bool:
    print('执行 TypeScript 类型检查...', flush=True)
    return run_node_tool('vue-tsc', 'bin/vue-tsc.js', '--noEmit')


def run_eslint_check(dry_run: bool = False) -> bool:
    print('执行 ESLint 检查...' if dry_run else '执行 ESLint 修复...', flush=True)
    args = ['src', '--ext', '.ts,.vue']
    if not dry_run:
        args.append('--fix')
    return run_node_tool('eslint', 'bin/eslint.js', *args)


def sort_imports(dry_run: bool = False) -> bool:
    print('检查 Vue import 排序...' if dry_run else '整理 Vue import 排序...', flush=True)
    script = Path(__file__).resolve().with_name('sort-imports.py')
    if not script.is_file():
        print(f'缺少必要脚本：{script}', file=sys.stderr)
        return False
    command = [sys.executable, '-B', str(script)]
    if dry_run:
        command.append('--dry-run')
    return run_command(command, capture=False)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument('--format', action='store_true', help='执行 ESLint 修复；--dry-run 时只检查')
    parser.add_argument('--imports', action='store_true', help='整理 Vue import；--dry-run 时只检查')
    parser.add_argument('--check', action='store_true', help='仅选择 TypeScript 类型检查')
    parser.add_argument('--eslint', action='store_true', help='仅选择 ESLint 只读检查')
    parser.add_argument('--all', action='store_true', help='选择全部步骤（未选择步骤时的默认行为）')
    parser.add_argument('--dry-run', action='store_true', help='不修改文件，发现问题非零退出')
    parser.add_argument('--parallel', action='store_true', help='并行执行只读检查，写入步骤仍按顺序执行')
    args = parser.parse_args()
    selected = args.format or args.imports or args.check or args.eslint
    if args.all and selected:
        parser.error('--all 不能与 --format / --imports / --check / --eslint 混用')
    all_steps = not selected
    tasks = []
    if all_steps or args.imports:
        tasks.append((lambda: sort_imports(args.dry_run), not args.dry_run))
    if all_steps or args.format or args.eslint:
        readonly = args.dry_run or (args.eslint and not args.format and not all_steps)
        tasks.append((lambda: run_eslint_check(readonly), not readonly))
    if all_steps or args.check:
        tasks.append((run_typescript_check, False))
    start = time.monotonic()
    if args.parallel and not any(writes for _, writes in tasks):
        with ThreadPoolExecutor(max_workers=3) as executor:
            futures = [executor.submit(task) for task, _ in tasks]
            results = [future.result() for future in futures]
    else:
        results = [task() for task, _ in tasks]
    success = all(results)
    print(f"{'全部步骤通过' if success else '存在未通过的步骤'}，耗时 {time.monotonic() - start:.2f}s")
    return 0 if success else 1


if __name__ == '__main__':
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        print('用户中断执行', file=sys.stderr)
        sys.exit(130)
