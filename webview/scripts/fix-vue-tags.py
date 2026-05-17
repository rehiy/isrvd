#!/usr/bin/env python3
"""
批量修复 Vue 模板标签：将多行属性标签合并为一行。

将这种格式：
  <button
    type="submit"
    :disabled="loading"
    class="btn btn-primary w-full"
  >
合并为：
  <button type="submit" :disabled="loading" class="btn btn-primary w-full">

规则：
1. 只处理开始标签（<tag ... >）
2. 合并后长度 < 180 字符才合并
3. 跳过自闭合标签（<tag ... />）
"""

import re
import os
import sys


def fix_vue_file(filepath):
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()

    lines = content.split('\n')
    result = []
    i = 0
    changed = False

    while i < len(lines):
        line = lines[i]

        # 检查是否是标签开始（<tagname 开头）
        m = re.match(r'^(\s*)<([\w-]+)', line)
        if not m:
            result.append(line)
            i += 1
            continue

        # 如果这一行已经有 > 且不是自闭合，跳过
        if '>' in line and not line.strip().endswith('/>'):
            result.append(line)
            i += 1
            continue

        # 如果这一行是自闭合标签
        if line.strip().endswith('/>'):
            result.append(line)
            i += 1
            continue

        # 收集多行标签
        indent = m.group(1)
        tag_name = m.group(2)
        parts = [line.strip()]
        j = i + 1

        while j < len(lines):
            current = lines[j]
            parts.append(current.strip())
            if '>' in current:
                break
            j += 1

        # 合并所有部分
        merged = ' '.join(p for p in parts if p)
        merged = re.sub(r'\s+', ' ', merged)
        # 去除 > 前的多余空格：将 " >" 替换为 ">"
        merged = re.sub(r'\s+>', '>', merged)

        if len(merged) < 180:
            # 保持原始缩进
            result.append(indent + merged.strip())
            changed = True
        else:
            result.extend(lines[i:j+1])

        i = j + 1

    if changed:
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write('\n'.join(result))
        return True
    return False


def main():
    vue_dir = '/home/rehiy-public/isrvd/webview/src'
    fixed_files = []

    for root, dirs, files in os.walk(vue_dir):
        dirs[:] = [d for d in dirs if d not in ['node_modules', 'dist', '.git']]

        for file in files:
            if file.endswith('.vue'):
                filepath = os.path.join(root, file)
                try:
                    if fix_vue_file(filepath):
                        relpath = os.path.relpath(filepath, vue_dir)
                        fixed_files.append(relpath)
                except Exception as e:
                    print(f"Error processing {filepath}: {e}", file=sys.stderr)

    if fixed_files:
        print(f"Fixed {len(fixed_files)} files:")
        for f in fixed_files:
            print(f"  - {f}")
    else:
        print("No files needed fixing.")


if __name__ == '__main__':
    main()
