#!/usr/bin/env python3
"""
前端代码批量格式化脚本
功能：ESLint 格式化、整理 import、TypeScript 类型检查
"""

import subprocess
import sys
import argparse
import time
from pathlib import Path
from concurrent.futures import ThreadPoolExecutor
from typing import List, Tuple, Optional

# 颜色输出
class Color:
    RED = '\033[0;31m'
    GREEN = '\033[0;32m'
    YELLOW = '\033[1;33m'
    BLUE = '\033[0;34m'
    CYAN = '\033[0;36m'
    NC = '\033[0m'

def log_info(msg):
    print(f"{Color.BLUE}[INFO]{Color.NC} {msg}")

def log_success(msg):
    print(f"{Color.GREEN}[SUCCESS]{Color.NC} {msg}")

def log_warn(msg):
    print(f"{Color.YELLOW}[WARN]{Color.NC} {msg}")

def log_error(msg):
    print(f"{Color.RED}[ERROR]{Color.NC} {msg}")

def log_debug(msg):
    print(f"{Color.CYAN}[DEBUG]{Color.NC} {msg}")

def run_command(cmd: str, check: bool = True, capture: bool = False, timeout: int = 300) -> Tuple[bool, Optional[str], Optional[str]]:
    """运行命令，支持超时控制"""
    try:
        if capture:
            result = subprocess.run(cmd, shell=True, capture_output=True, text=True, timeout=timeout)
            return result.returncode == 0, result.stdout, result.stderr
        else:
            result = subprocess.run(cmd, shell=True, timeout=timeout)
            return result.returncode == 0, None, None
    except subprocess.TimeoutExpired:
        log_error(f"命令执行超时: {cmd}")
        if check:
            sys.exit(1)
        return False, None, "Timeout"
    except Exception as e:
        if check:
            log_error(f"命令执行失败: {e}")
            sys.exit(1)
        return False, None, str(e)

def check_npm():
    """检查 npm 是否可用"""
    log_info("检查 npm 环境...")
    success, _, _ = run_command("npm --version", check=False, capture=True)
    if not success:
        log_error("npm 未安装，请先安装 Node.js")
        sys.exit(1)
    log_success("npm 环境正常")

def run_typescript_check() -> bool:
    """执行 TypeScript 类型检查"""
    log_info("执行 TypeScript 类型检查...")
    start_time = time.time()

    success, stdout, stderr = run_command("vue-tsc --noEmit", check=False, capture=True)
    elapsed = time.time() - start_time

    if success:
        log_success(f"TypeScript 类型检查通过 (耗时: {elapsed:.2f}s)")
        return True
    else:
        log_warn(f"TypeScript 类型检查发现问题 (耗时: {elapsed:.2f}s)")
        if stderr:
            log_debug(stderr)
        return False

def run_eslint_check(dry_run: bool = False) -> bool:
    """执行 ESLint 检查或修复（同时承担格式化职责）"""
    if dry_run:
        log_info("执行 ESLint 检查...")
        cmd = "eslint src --ext .ts,.vue"
    else:
        log_info("执行 ESLint 修复...")
        cmd = "eslint src --ext .ts,.vue --fix"

    start_time = time.time()
    success, stdout, stderr = run_command(cmd, check=False, capture=True)
    elapsed = time.time() - start_time

    if success:
        if dry_run:
            log_success(f"ESLint 检查通过 (耗时: {elapsed:.2f}s)")
        else:
            log_success(f"ESLint 修复完成 (耗时: {elapsed:.2f}s)")
        return True
    else:
        if dry_run:
            log_warn(f"ESLint 检查发现问题 (耗时: {elapsed:.2f}s)")
        else:
            log_warn(f"ESLint 修复发现问题 (耗时: {elapsed:.2f}s)")
        if stderr:
            log_debug(stderr)
        return False

def sort_imports(dry_run: bool = False) -> bool:
    """整理 import 语句"""
    log_info("整理 import 语句...")

    script_dir = Path(__file__).parent
    sort_script = script_dir / "sort-imports.py"

    if sort_script.exists():
        cmd = f"python3 {sort_script}"
        if dry_run:
            cmd += " --dry-run"
        success, _, _ = run_command(cmd, check=False)
        if success:
            log_success("import 语句整理完成")
            return True
        else:
            log_warn("import 语句整理失败")
            return False
    else:
        log_warn("sort-imports.py 脚本不存在，跳过")
        return True

def run_parallel_tasks(tasks: List[Tuple[callable, tuple]], max_workers: int = 2) -> bool:
    """并行执行任务"""
    all_success = True

    with ThreadPoolExecutor(max_workers=max_workers) as executor:
        futures = []
        for task_func, args in tasks:
            futures.append(executor.submit(task_func, *args))

        for future in futures:
            try:
                success = future.result(timeout=600)
                if not success:
                    all_success = False
            except Exception as e:
                log_error(f"任务执行异常: {e}")
                all_success = False

    return all_success

def main():
    parser = argparse.ArgumentParser(description="前端代码格式化工具", add_help=False)
    parser.add_argument("--format", action="store_true", help="仅执行 ESLint 格式化修复")
    parser.add_argument("--imports", action="store_true", help="仅整理 import")
    parser.add_argument("--check", action="store_true", help="仅执行类型检查")
    parser.add_argument("--eslint", action="store_true", help="仅执行 ESLint 检查")
    parser.add_argument("--all", action="store_true", help="执行所有步骤（默认）")
    parser.add_argument("--dry-run", action="store_true", help="预览模式（不修改文件）")
    parser.add_argument("--parallel", action="store_true", help="启用并行执行")
    parser.add_argument("-h", "--help", action="store_true", help="显示帮助信息")

    args = parser.parse_args()

    if args.help:
        print("""用法: python format.py [选项]

选项:
  --format      仅执行 ESLint 格式化修复
  --imports     仅整理 import
  --check       仅执行 TypeScript 类型检查
  --eslint      仅执行 ESLint 检查（不修改文件）
  --all         执行所有步骤（默认）
  --dry-run     预览模式（不修改文件）
  --parallel    启用并行执行（仅对某些任务有效）
  -h, --help    显示帮助信息

示例:
  python format.py                  # 执行完整格式化流程
  python format.py --format         # 仅 ESLint 格式化
  python format.py --dry-run        # 预览问题，不修改
  python format.py --check --eslint # 执行类型和 ESLint 检查
  python format.py --all --parallel # 并行执行所有任务
""")
        sys.exit(0)

    # 确定执行模式
    if sum([args.format, args.imports, args.check, args.eslint]) == 0:
        action = "all"
    else:
        action = "custom"

    dry_run = args.dry_run
    use_parallel = args.parallel

    log_info("前端代码格式化工具")
    log_info("========================")

    check_npm()

    start_time = time.time()

    try:
        if args.format:
            run_eslint_check(dry_run)
        elif args.imports:
            sort_imports(dry_run)
        elif args.check:
            run_typescript_check()
        elif args.eslint:
            run_eslint_check(dry_run=True)
        elif action == "all":
            if dry_run:
                log_info("预览模式：检查代码质量...")
                if use_parallel:
                    tasks = [
                        (run_typescript_check, ()),
                        (run_eslint_check, (True,)),
                        (sort_imports, (True,))
                    ]
                    run_parallel_tasks(tasks)
                else:
                    run_typescript_check()
                    run_eslint_check(True)
                    sort_imports(True)
            else:
                if use_parallel:
                    tasks = [
                        (run_typescript_check, ()),
                        (sort_imports, (False,))
                    ]
                    run_parallel_tasks(tasks)
                    # ESLint fix 放最后，避免与 sort_imports 冲突
                    run_eslint_check(False)
                else:
                    sort_imports()
                    run_eslint_check(False)
                    run_typescript_check()
    except KeyboardInterrupt:
        log_warn("用户中断执行")
        sys.exit(1)
    except Exception as e:
        log_error(f"执行过程中发生错误: {e}")
        sys.exit(1)

    elapsed = time.time() - start_time
    log_success(f"完成！总耗时: {elapsed:.2f}s")

if __name__ == "__main__":
    main()

