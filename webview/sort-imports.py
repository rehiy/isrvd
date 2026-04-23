#!/usr/bin/env python3
"""
批量整理 Vue 文件中 <script> 块内的 import 语句。

分组顺序（组间空一行，组内按字母排序）：
  1. 第三方库          vue-facing-decorator、vue、vue-router 等（不以 @/ 开头）
  2. @/store/...
  3. @/router
  4. @/service/...
  5. @/helper/...
  6. @/component/...
  7. @/views/...
  8. 其余 @/ 路径（兜底）

同一模块的普通导入与 type 导入合并为相邻两行（普通在前，type 在后），
不跨组拆散。
"""

import re
import sys
from pathlib import Path

# ── 分组规则（按优先级匹配，返回 (group_index, sort_key)）──────────────────
GROUP_PATTERNS = [
    (0, re.compile(r"^'(?!@/)")),          # 第三方：不以 @/ 开头
    (1, re.compile(r"^'@/store/")),         # store
    (2, re.compile(r"^'@/router")),         # router
    (3, re.compile(r"^'@/service/")),       # service
    (4, re.compile(r"^'@/helper/")),        # helper
    (5, re.compile(r"^'@/component/")),     # component
    (6, re.compile(r"^'@/views/")),         # views
    (7, re.compile(r"^'@/")),              # 其余 @/ 兜底
]

def get_group(import_line: str) -> int:
    """从 import 语句中提取模块路径并返回分组编号。"""
    m = re.search(r"from\s+('[^']+')", import_line)
    if not m:
        return 99
    path = m.group(1)
    for idx, pattern in GROUP_PATTERNS:
        if pattern.match(path):
            return idx
    return 99

def sort_key(import_line: str) -> tuple:
    """排序键：(分组, 是否 type import, 模块路径, 导入内容)"""
    group = get_group(import_line)
    is_type = 1 if re.match(r"import\s+type\s+", import_line) else 0
    m = re.search(r"from\s+('[^']+')$", import_line.rstrip())
    path = m.group(1) if m else ''
    return (group, is_type, path, import_line)

def reorder_imports(lines: list[str]) -> list[str]:
    """
    接收 script 块内的所有行，提取并重排 import 语句，
    返回重排后的完整行列表。
    """
    import_lines: list[str] = []
    other_lines: list[str] = []
    # 收集连续 import 块（允许 import 块之间有空行）
    in_import_zone = True
    for line in lines:
        stripped = line.rstrip('\n')
        if in_import_zone:
            if re.match(r"^import\s+", stripped) or stripped == '':
                if re.match(r"^import\s+", stripped):
                    import_lines.append(stripped)
                # 空行在 import 区内跳过（重新生成）
            else:
                in_import_zone = False
                other_lines.append(line)
        else:
            other_lines.append(line)

    if not import_lines:
        return lines

    # 按分组 + 字母排序
    import_lines.sort(key=sort_key)

    # 按分组插入空行分隔
    grouped: list[str] = []
    prev_group = -1
    for imp in import_lines:
        g = get_group(imp)
        if prev_group != -1 and g != prev_group:
            grouped.append('')
        grouped.append(imp)
        prev_group = g

    # 重新组合：import 块 + 空行 + 其余代码
    result_lines = [l + '\n' for l in grouped]
    # 确保 import 块与后续代码之间有且仅有一个空行
    if other_lines:
        # 去掉 other_lines 开头多余空行
        while other_lines and other_lines[0].strip() == '':
            other_lines.pop(0)
        result_lines.append('\n')
        result_lines.extend(other_lines)

    return result_lines

def process_vue_file(path: Path, dry_run: bool = False) -> bool:
    """处理单个 Vue 文件，返回是否有修改。"""
    text = path.read_text(encoding='utf-8')
    lines = text.splitlines(keepends=True)

    # 找到 <script> 块范围
    script_start = script_end = -1
    for i, line in enumerate(lines):
        if re.match(r"<script(\s|>)", line) and script_start == -1:
            script_start = i
        elif line.strip() == '</script>' and script_start != -1:
            script_end = i
            break

    if script_start == -1:
        return False

    # script 块内容（不含 <script> 和 </script> 行本身）
    inner = lines[script_start + 1: script_end]
    new_inner = reorder_imports(inner)

    if inner == new_inner:
        return False

    new_lines = lines[:script_start + 1] + new_inner + lines[script_end:]
    new_text = ''.join(new_lines)

    if not dry_run:
        path.write_text(new_text, encoding='utf-8')
    return True

def main():
    src_dir = Path(__file__).parent / 'src'
    dry_run = '--dry-run' in sys.argv

    vue_files = sorted(src_dir.rglob('*.vue'))
    changed = []
    for f in vue_files:
        if process_vue_file(f, dry_run=dry_run):
            changed.append(f)
            print(f"{'[dry]' if dry_run else '[fix]'} {f.relative_to(src_dir.parent.parent)}")

    print(f"\n共处理 {len(vue_files)} 个文件，{'需要' if dry_run else '已'}修改 {len(changed)} 个。")

if __name__ == '__main__':
    main()
