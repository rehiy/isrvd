#!/usr/bin/env python3
"""
前端样式 Review 脚本 — 基于 AGENTS.md 6.x 规范
用法：cd webview && python3 scripts/review-style.py
"""

import re
import sys
from pathlib import Path
from collections import defaultdict

VIEWS_DIR = Path(__file__).parent.parent / "src" / "views"
SKIP_DIRS = {"widget"}

RED    = "\033[31m"
YELLOW = "\033[33m"
GREEN  = "\033[32m"
CYAN   = "\033[36m"
BOLD   = "\033[1m"
RESET  = "\033[0m"

issues: list[tuple[str, int, str, str]] = []

def report(filepath: str, lineno: int, level: str, msg: str):
    issues.append((filepath, lineno, level, msg))

# ─── 工具函数 ────────

def get_template(content: str) -> str:
    m = re.search(r'<template>(.*?)</template>', content, re.DOTALL)
    return m.group(1) if m else ""

def find_line(lines: list[str], text: str) -> int:
    for i, line in enumerate(lines, 1):
        if text in line:
            return i
    return 1

def has_toolbar(tmpl: str) -> bool:
    return 'bg-slate-50 border-b border-slate-200 rounded-t-2xl' in tmpl

def has_card_list(tmpl: str) -> bool:
    """判断是否是有卡片列表的页面（移动端 space-y-3 + rounded-xl 卡片）"""
    pos = tmpl.find('md:hidden space-y-3')
    if pos == -1:
        return False
    return 'rounded-xl border border-slate-200 bg-white p-4' in tmpl[pos:pos+400]

# ─── 检查函数 ────────

def check_toolbar_layout(filepath, lines, tmpl, tmpl_line0):
    """6.6 toolbar 必须有桌面/移动双布局，且有标题结构"""
    if not has_toolbar(tmpl):
        return
    # 排除面包屑导航页（没有 h1 标题）
    toolbar_start = tmpl.find('bg-slate-50 border-b border-slate-200 rounded-t-2xl')
    toolbar_snippet = tmpl[toolbar_start:toolbar_start + 300]
    if '<h1' not in toolbar_snippet and 'font-semibold text-slate-800' not in toolbar_snippet:
        return

    if 'hidden md:flex' not in tmpl:
        report(filepath, 0, "ERROR", "缺少桌面端 toolbar 布局 (hidden md:flex)")
    if 'flex md:hidden' not in tmpl and 'block md:hidden' not in tmpl:
        report(filepath, 0, "ERROR", "缺少移动端 toolbar 布局 (flex md:hidden)")

    # 移动端 toolbar 标题应为 text-lg，不得用 text-base
    mobile_start = tmpl.find('flex md:hidden')
    if mobile_start == -1:
        mobile_start = tmpl.find('block md:hidden')
    if mobile_start != -1:
        next_desktop = tmpl.find('hidden md:', mobile_start + 10)
        mobile_snippet = tmpl[mobile_start: next_desktop if next_desktop != -1 else mobile_start + 2000]
        for i, line in enumerate(mobile_snippet.splitlines()):
            if 'font-semibold' in line and 'text-base' in line and 'text-lg' not in line:
                lineno = tmpl_line0 + tmpl[:mobile_start].count('\n') + i + 1
                report(filepath, lineno, "WARN",
                       f"移动端 toolbar 标题用了 text-base，应为 text-lg: {line.strip()[:80]}")


def check_toolbar_icon(filepath, lines, tmpl, tmpl_line0):
    """6.6 toolbar 图标应为 w-9 h-9 rounded-lg"""
    if not has_toolbar(tmpl):
        return
    toolbar_start = tmpl.find('bg-slate-50 border-b border-slate-200 rounded-t-2xl')
    toolbar_snippet = tmpl[toolbar_start:toolbar_start + 300]
    if '<h1' not in toolbar_snippet and 'font-semibold text-slate-800' not in toolbar_snippet:
        return

    # 找 toolbar 区域（到第一个 v-if= 为止）
    toolbar_end = len(tmpl)
    for marker in ['v-if="loading"', 'v-if="tasksLoading"', 'v-if="servicesLoading"',
                   'v-if="nodesLoading"', 'v-if="membersLoading"', 'v-else-if=']:
        pos = tmpl.find(marker, toolbar_start + 100)
        if pos != -1 and pos < toolbar_end:
            toolbar_end = pos
    toolbar_section = tmpl[toolbar_start:toolbar_end]
    base_lineno = tmpl_line0 + tmpl[:toolbar_start].count('\n')

    for i, line in enumerate(toolbar_section.splitlines()):
        stripped = line.strip()
        if not (stripped.startswith('<div') and 'rounded-lg' in stripped
                and 'flex items-center justify-center' in stripped):
            continue
        w_m = re.search(r'\bw-(\d+)\b', stripped)
        h_m = re.search(r'\bh-(\d+)\b', stripped)
        if not w_m or not h_m:
            continue
        w, h = w_m.group(1), h_m.group(1)
        if w != h:
            report(filepath, base_lineno + i + 1, "WARN",
                   f"toolbar 图标宽高不一致 w-{w}/h-{h}: {stripped[:60]}")
        elif w != '9':
            report(filepath, base_lineno + i + 1, "WARN",
                   f"toolbar 图标应为 w-9 h-9，当前为 w-{w} h-{h}: {stripped[:60]}")
        if 'rounded-xl' in stripped:
            report(filepath, base_lineno + i + 1, "WARN",
                   f"toolbar 图标应用 rounded-lg 而非 rounded-xl: {stripped[:60]}")


def check_empty_state(filepath, lines, tmpl, tmpl_line0):
    """6.6 列表页空状态必须有主标题+副标题两行"""
    if not has_toolbar(tmpl):
        return
    tmpl_lines = tmpl.splitlines()
    i = 0
    while i < len(tmpl_lines):
        line = tmpl_lines[i]
        if 'py-20' not in line or 'spinner' in line:
            i += 1
            continue
        depth, block_lines, j = 0, [], i
        while j < len(tmpl_lines):
            bl = tmpl_lines[j]
            block_lines.append(bl)
            depth += bl.count('<div') - bl.count('</div')
            if j > i and depth <= 0:
                break
            j += 1
        block = '\n'.join(block_lines)
        if 'spinner' in block or '未找到' in block or '详情' in block:
            i = j + 1
            continue
        p_count = len(re.findall(r'<p\b', block))
        if p_count < 2:
            report(filepath, tmpl_line0 + i, "WARN",
                   f"空状态只有 {p_count} 行文字，应有主标题+副标题两行")
        i = j + 1


def check_mobile_card_container(filepath, lines, tmpl, tmpl_line0):
    """6.7 移动端卡片容器必须有 p-4"""
    for m in re.finditer(r'class="([^"]*md:hidden[^"]*space-y-3[^"]*)"', tmpl):
        cls = m.group(1)
        after = tmpl[m.end():m.end() + 400]
        if 'rounded-xl border border-slate-200 bg-white p-4' not in after:
            continue
        if 'p-4' not in cls.split():
            lineno = tmpl_line0 + tmpl[:m.start()].count('\n')
            report(filepath, lineno, "ERROR",
                   f"移动端卡片容器缺少 p-4: class=\"{cls[:80]}\"")


def check_mobile_card_hover(filepath, lines, tmpl, tmpl_line0):
    """6.7 移动端卡片应有 transition-all hover:shadow-sm"""
    if not has_card_list(tmpl):
        return
    mobile_start = tmpl.find('md:hidden space-y-3')
    base_lineno = tmpl_line0 + tmpl[:mobile_start].count('\n')
    for i, line in enumerate(tmpl[mobile_start:].splitlines()):
        stripped = line.strip()
        if 'rounded-xl border border-slate-200 bg-white p-4' in stripped:
            if 'hover:shadow-sm' not in stripped:
                report(filepath, base_lineno + i + 1, "WARN",
                       f"移动端卡片缺少 hover:shadow-sm: {stripped[:80]}")
            if 'transition-all' not in stripped:
                report(filepath, base_lineno + i + 1, "WARN",
                       f"移动端卡片缺少 transition-all: {stripped[:80]}")


def check_mobile_card_top(filepath, lines, tmpl, tmpl_line0):
    """
    6.7 移动端卡片顶部：
    - 图标 flex-shrink-0，w-10 h-10 rounded-lg
    - 文字容器 min-w-0
    - 主名称 truncate block（不得用额外 flex 包裹）
    - 副信息 truncate block mt-0.5
    """
    if not has_card_list(tmpl):
        return
    mobile_start = tmpl.find('md:hidden space-y-3')
    mobile_section = tmpl[mobile_start:]
    base_lineno = tmpl_line0 + tmpl[:mobile_start].count('\n')
    mobile_lines = mobile_section.splitlines()

    for i, line in enumerate(mobile_lines):
        stripped = line.strip()
        # 找卡片顶部容器（mb-3 且后面有 w-10 h-10 图标）
        if not ('mb-3' in stripped and 'flex' in stripped and stripped.startswith('<div')):
            continue
        if 'gap-3' not in stripped and 'justify-between' not in stripped:
            continue
        snippet = '\n'.join(mobile_lines[i:i+15])
        if 'w-10 h-10' not in snippet:
            continue

        lineno = base_lineno + i + 1

        # 图标 flex-shrink-0
        if 'flex-shrink-0' not in snippet:
            report(filepath, lineno, "WARN", "移动端卡片顶部图标缺少 flex-shrink-0")

        # 文字容器 min-w-0
        text_snippet = '\n'.join(mobile_lines[i+2:i+10])
        if 'min-w-0' not in text_snippet:
            report(filepath, lineno, "WARN", "移动端卡片顶部文字容器缺少 min-w-0")

        # 主名称 truncate block（不得用 flex 包裹）
        for nl in mobile_lines[i:i+10]:
            if 'font-medium' in nl and 'text-slate-800' in nl:
                if 'truncate' not in nl:
                    report(filepath, lineno, "WARN",
                           f"移动端卡片主名称缺少 truncate: {nl.strip()[:80]}")
                # 检查是否被 flex 包裹（主名称行本身不应有 flex）
                if 'flex items-center' in nl and 'gap-2' in nl:
                    report(filepath, lineno, "WARN",
                           f"移动端卡片主名称不得用额外 flex 包裹: {nl.strip()[:80]}")
                break

        # 副信息 mt-0.5
        for nl in mobile_lines[i:i+10]:
            if 'text-xs text-slate-400' in nl and 'truncate block' in nl and 'mt-0.5' not in nl and 'font-semibold' not in nl:
                report(filepath, lineno, "WARN",
                       f"移动端卡片副信息缺少 mt-0.5: {nl.strip()[:80]}")
                break


def check_mobile_card_rows(filepath, lines, tmpl, tmpl_line0):
    """
    6.8 移动端卡片属性行：
    - 行间距统一 mb-3（不得用 mb-2）
    - badge/code(py-0.5/py-1) 行：items-start，标签 mt-0.5/mt-1
    - 纯文本值：text-slate-500（不得用 text-slate-600）
    - badge 形状：rounded 或 rounded-lg（不得用 rounded-full）
    """
    if not has_card_list(tmpl):
        return
    mobile_start = tmpl.find('md:hidden space-y-3')
    mobile_end = len(tmpl)
    for marker in ['hidden md:block']:
        pos = tmpl.find(marker, mobile_start)
        if pos != -1 and pos < mobile_end:
            mobile_end = pos

    mobile_section = tmpl[mobile_start:mobile_end]
    base_lineno = tmpl_line0 + tmpl[:mobile_start].count('\n')
    mobile_lines = mobile_section.splitlines()

    for i, line in enumerate(mobile_lines):
        stripped = line.strip()
        if not (stripped.startswith('<div') and 'flex' in stripped and 'gap-2' in stripped):
            continue
        # 跳过操作按钮区、顶部区域、卡片容器
        if any(x in stripped for x in ['pt-2', 'border-t', 'gap-3', 'justify-between',
                                        'rounded-xl', 'space-y-3', 'justify-center']):
            continue
        if 'min-w-0' in stripped and 'flex-1' in stripped:
            continue
        indent = len(line) - len(line.lstrip())
        if indent < 12:
            continue

        lineno = base_lineno + i + 1

        # mb-3 检查
        if 'mb-3' not in stripped:
            report(filepath, lineno, "WARN",
                   f"移动端属性行缺少 mb-3: {stripped[:80]}")
            continue

        # 检查下一行值的类型
        next_lines = mobile_lines[i+1:i+4]
        for nl in next_lines:
            nls = nl.strip()
            if not nls or nls.startswith('<!--'):
                continue
            has_py05 = 'py-0.5' in nls
            has_py1 = bool(re.search(r'\bpy-1\b', nls))
            has_flex_wrap = 'flex-wrap' in nls
            needs_start = has_py05 or has_py1 or has_flex_wrap

            if needs_start and 'items-center' in stripped and 'items-start' not in stripped:
                report(filepath, lineno, "WARN",
                       f"属性行值为 badge/code，容器应用 items-start: {stripped[:80]}")

            # badge rounded-full 检查
            if (has_py05 or has_py1) and 'rounded-full' in nls and 'w-' not in nls:
                report(filepath, lineno + 1, "WARN",
                       f"属性行 badge 应用 rounded 或 rounded-lg，不得用 rounded-full: {nls[:80]}")

            # 纯文本值颜色检查
            if (not has_py05 and not has_py1 and not has_flex_wrap
                    and nls.startswith('<span') and 'text-slate-600' in nls
                    and 'font-medium' not in nls and 'px-' not in nls):
                report(filepath, lineno + 1, "WARN",
                       f"移动端属性行纯文本值应用 text-slate-500 而非 text-slate-600: {nls[:80]}")
            break


def check_table_first_col(filepath, lines, tmpl, tmpl_line0):
    """6.9 表格第一列布局"""
    desktop_start = tmpl.find('hidden md:block overflow-x-auto')
    if desktop_start == -1:
        return
    desktop_end = tmpl.find('md:hidden space-y-3', desktop_start)
    if desktop_end == -1:
        desktop_end = len(tmpl)
    desktop_section = tmpl[desktop_start:desktop_end]
    base_lineno = tmpl_line0 + tmpl[:desktop_start].count('\n')

    for i, line in enumerate(desktop_section.splitlines()):
        stripped = line.strip()
        if not (stripped.startswith('<td') and 'px-4 py-3' in stripped and 'max-w-[280px]' in stripped):
            continue
        snippet = '\n'.join(desktop_section.splitlines()[i:i+12])
        if 'w-8 h-8' not in snippet:
            continue  # 不是有图标的第一列
        lineno = base_lineno + i + 1
        if 'min-w-0' not in snippet:
            report(filepath, lineno, "WARN", "表格第一列缺少 min-w-0")
        if 'flex-shrink-0' not in snippet:
            report(filepath, lineno, "WARN", "表格第一列图标缺少 flex-shrink-0")
        if 'truncate block' not in snippet:
            report(filepath, lineno, "WARN", "表格第一列主名称缺少 truncate block")


def check_table_icon_size(filepath, lines, tmpl, tmpl_line0):
    """6.9 桌面端表格图标应为 w-8 h-8，移动端卡片图标应为 w-10 h-10"""
    desktop_start = tmpl.find('hidden md:block overflow-x-auto')
    if desktop_start != -1:
        desktop_end = tmpl.find('md:hidden space-y-3', desktop_start)
        if desktop_end == -1:
            desktop_end = len(tmpl)
        desktop_section = tmpl[desktop_start:desktop_end]
        base_lineno = tmpl_line0 + tmpl[:desktop_start].count('\n')
        for i, line in enumerate(desktop_section.splitlines()):
            stripped = line.strip()
            if not (stripped.startswith('<div') and 'rounded-lg' in stripped
                    and 'flex items-center justify-center' in stripped and 'flex-shrink-0' in stripped):
                continue
            w_m = re.search(r'\bw-(\d+)\b', stripped)
            h_m = re.search(r'\bh-(\d+)\b', stripped)
            if w_m and h_m and (w_m.group(1) != '8' or h_m.group(1) != '8'):
                report(filepath, base_lineno + i + 1, "WARN",
                       f"桌面端表格图标应为 w-8 h-8，当前为 w-{w_m.group(1)} h-{h_m.group(1)}")

    mobile_start = tmpl.find('md:hidden space-y-3')
    if mobile_start != -1 and has_card_list(tmpl):
        mobile_section = tmpl[mobile_start:]
        base_lineno = tmpl_line0 + tmpl[:mobile_start].count('\n')
        for i, line in enumerate(mobile_section.splitlines()):
            stripped = line.strip()
            if not (stripped.startswith('<div') and 'rounded-lg' in stripped
                    and 'flex items-center justify-center' in stripped and 'flex-shrink-0' in stripped):
                continue
            w_m = re.search(r'\bw-(\d+)\b', stripped)
            h_m = re.search(r'\bh-(\d+)\b', stripped)
            if w_m and h_m and (w_m.group(1) != '10' or h_m.group(1) != '10'):
                report(filepath, base_lineno + i + 1, "WARN",
                       f"移动端卡片图标应为 w-10 h-10，当前为 w-{w_m.group(1)} h-{h_m.group(1)}")


def check_table_text_colors(filepath, lines, tmpl, tmpl_line0):
    """6.9 桌面端表格时间列/普通列应用 text-slate-600，不得用 text-slate-500"""
    desktop_start = tmpl.find('hidden md:block overflow-x-auto')
    if desktop_start == -1:
        return
    desktop_end = tmpl.find('md:hidden space-y-3', desktop_start)
    if desktop_end == -1:
        desktop_end = len(tmpl)
    desktop_section = tmpl[desktop_start:desktop_end]
    base_lineno = tmpl_line0 + tmpl[:desktop_start].count('\n')

    for i, line in enumerate(desktop_section.splitlines()):
        stripped = line.strip()
        if not stripped.startswith('<td'):
            continue
        # 时间列（whitespace-nowrap）
        if 'whitespace-nowrap' in stripped and 'text-slate-500' in stripped:
            report(filepath, base_lineno + i + 1, "WARN",
                   f"桌面端时间列应用 text-slate-600 而非 text-slate-500: {stripped[:80]}")
        # 普通数据列（text-sm text-slate-500）
        if 'text-sm text-slate-500' in stripped:
            report(filepath, base_lineno + i + 1, "WARN",
                   f"桌面端数据列应用 text-slate-600 而非 text-slate-500: {stripped[:80]}")


def check_desktop_action_gap(filepath, lines, tmpl, tmpl_line0):
    """6.9 桌面端操作按钮列 gap 必须是 gap-1"""
    for i, line in enumerate(lines, 1):
        if 'flex justify-end items-center' in line:
            gap_m = re.search(r'gap-([\d.]+)', line)
            if gap_m and gap_m.group(1) != '1':
                report(filepath, i, "WARN",
                       f"桌面端操作按钮列 gap 应为 gap-1，当前为 gap-{gap_m.group(1)}")


def check_mobile_action_gap(filepath, lines, tmpl, tmpl_line0):
    """6.10 移动端操作按钮区 gap 必须是 gap-1.5"""
    for i, line in enumerate(lines, 1):
        if 'flex flex-wrap' in line and 'pt-2' in line and 'border-t' in line:
            if 'gap-1.5' not in line:
                gap_m = re.search(r'gap-[\d.]+', line)
                gap_val = gap_m.group(0) if gap_m else '无 gap'
                report(filepath, i, "WARN",
                       f"移动端操作按钮区 gap 应为 gap-1.5，当前为 {gap_val}")


def check_action_buttons(filepath, lines, tmpl, tmpl_line0):
    """6.10 操作按钮语义色"""
    is_apisix = 'apisix' in filepath.lower()
    for i, line in enumerate(lines, 1):
        if 'btn-icon' not in line or 'cursor-not-allowed' in line:
            continue
        if 'hover:bg-' not in line:
            report(filepath, i, "WARN", f"btn-icon 缺少 hover:bg-xxx-50: {line.strip()[:80]}")
        if 'fa-trash' in line and 'text-red-600' not in line:
            report(filepath, i, "WARN", f"删除按钮应用 text-red-600: {line.strip()[:80]}")
        if 'fa-circle-info' in line and 'text-slate-600' not in line:
            report(filepath, i, "WARN", f"详情按钮应用 text-slate-600: {line.strip()[:80]}")
        if 'fa-pen' in line:
            allowed = ['text-blue-600', 'text-indigo-600', 'text-violet-600',
                       'text-cyan-600', 'text-rose-600', 'text-emerald-600']
            if not any(c in line for c in allowed):
                report(filepath, i, "WARN", f"编辑按钮颜色不规范: {line.strip()[:80]}")
            elif not is_apisix and 'text-blue-600' not in line:
                report(filepath, i, "WARN",
                       f"非 APISIX 模块编辑按钮应用 text-blue-600: {line.strip()[:80]}")


def check_desktop_badge_shape(filepath, lines, tmpl, tmpl_line0):
    """桌面端表格状态列 badge 不得用 rounded-full（应用 rounded 或 rounded-lg）"""
    desktop_start = tmpl.find('hidden md:block overflow-x-auto')
    if desktop_start == -1:
        return
    desktop_end = tmpl.find('md:hidden space-y-3', desktop_start)
    if desktop_end == -1:
        desktop_end = len(tmpl)
    desktop_section = tmpl[desktop_start:desktop_end]
    base_lineno = tmpl_line0 + tmpl[:desktop_start].count('\n')

    for i, line in enumerate(desktop_section.splitlines()):
        stripped = line.strip()
        if ('rounded-full' in stripped and ('py-0.5' in stripped or re.search(r'\bpy-1\b', stripped))
                and 'px-2' in stripped and 'w-' not in stripped):
            report(filepath, base_lineno + i + 1, "WARN",
                   f"桌面端 badge 应用 rounded 或 rounded-lg，不得用 rounded-full: {stripped[:80]}")


def check_h_tags_in_mobile(filepath, lines, tmpl, tmpl_line0):
    """移动端卡片内不应使用 h2-h6 标签"""
    mobile_start = tmpl.find('md:hidden space-y-3')
    if mobile_start == -1:
        return
    base_lineno = tmpl_line0 + tmpl[:mobile_start].count('\n')
    for i, line in enumerate(tmpl[mobile_start:].splitlines()):
        if re.search(r'<h[2-6]\b', line.strip()):
            report(filepath, base_lineno + i + 1, "WARN",
                   f"移动端卡片内不应使用 h2-h6 标签: {line.strip()[:60]}")


# 6.9.1 状态关键词
_STATE_POSITIVE = re.compile(r'\b(running|ready|active|enabled|healthy)\b', re.I)
_STATE_NEGATIVE = re.compile(r'\b(stopped|down|error|failed|unhealthy|exited)\b', re.I)
_STATE_WARNING  = re.compile(r'\b(drain|paused|warning|degraded|pending)\b', re.I)

# 状态值用 badge 背景色（bg-xxx-100/50 + text-xxx-700）的模式
_BADGE_BG = re.compile(r'bg-(?:emerald|green|red|rose|amber|yellow|orange|slate|gray)-(?:50|100)')


def check_status_uses_text_color(filepath, lines, tmpl, tmpl_line0):
    """
    6.9.1 状态值应优先用文字颜色区分，不应用 badge 背景色。
    检测：:class 绑定中同时出现状态关键词 + badge 背景色（bg-xxx-100/50）
    仅检查 :class=（动态绑定），排除 v-if/v-show 条件判断行。
    """
    for i, line in enumerate(lines, 1):
        stripped = line.strip()
        # 跳过注释
        if stripped.startswith('//') or stripped.startswith('*') or stripped.startswith('<!--'):
            continue
        # 只检查动态 :class 绑定（排除静态 class= 和 v-if 条件行）
        if ':class=' not in stripped:
            continue
        # 跳过按钮（v-if 条件判断中含状态关键词）
        if stripped.startswith('<button') or 'btn-icon' in stripped or 'hover:bg-' in stripped:
            continue
        # 跳过图标容器（bg-xxx-400 是图标背景）
        if re.search(r'bg-\w+-400', stripped):
            continue
        # 跳过空状态图标容器
        if 'justify-center' in stripped and re.search(r'\bw-20\b', stripped):
            continue
        # 跳过图标容器（w-8/w-10 + justify-center）
        if re.search(r'\bw-(?:8|9|10)\b', stripped) and 'justify-center' in stripped:
            continue
        # 跳过任务状态 badge（getStateClass 属于枚举分类，允许用 badge）
        if 'getStateClass' in stripped or 'getState(' in stripped:
            continue

        has_state = (_STATE_POSITIVE.search(stripped) or
                     _STATE_NEGATIVE.search(stripped) or
                     _STATE_WARNING.search(stripped))
        has_badge_bg = _BADGE_BG.search(stripped)

        if has_state and has_badge_bg:
            # 排除两值枚举 badge（如 running ? bg-emerald-100 : bg-slate-100）
            # 这类场景只有两种状态，用 badge 是合理的枚举分类展示
            if re.search(r"'\s*\?\s*'bg-\w+-(?:50|100)", stripped):
                continue
            report(filepath, i, "WARN",
                   f"状态值应用文字颜色（text-xxx-600 font-medium）而非 badge 背景色: {stripped[:100]}")


def check_status_text_color_values(filepath, lines, tmpl, tmpl_line0):
    """
    6.9.1 状态文字颜色值规范：
    - 正常/运行/就绪 → text-emerald-600 font-medium
    - 异常/停止/下线 → text-red-500 font-medium（注意是 red-500 不是 red-600）
    - 警告/排空/暂停 → text-amber-600 font-medium
    检测：:class 动态绑定中用了文字颜色表示状态但颜色值不符合规范
    """
    for i, line in enumerate(lines, 1):
        stripped = line.strip()
        if stripped.startswith('//') or stripped.startswith('*') or stripped.startswith('<!--'):
            continue
        # 只检查动态 :class 绑定
        if ':class=' not in stripped:
            continue
        # 跳过按钮行（v-if 条件中含状态关键词）
        if stripped.startswith('<button') or 'btn-icon' in stripped:
            continue
        # 只检查含状态关键词的行
        if not (_STATE_POSITIVE.search(stripped) or _STATE_NEGATIVE.search(stripped) or _STATE_WARNING.search(stripped)):
            continue
        # 跳过图标容器
        if re.search(r'\bw-(?:8|9|10)\b', stripped) and 'justify-center' in stripped:
            continue
        # 跳过 badge 背景（已由上一个检查处理）
        if _BADGE_BG.search(stripped):
            continue
        # 跳过 getStateClass 等函数调用
        if 'getStateClass' in stripped or 'getState(' in stripped:
            continue

        # 检查负向状态用了 red-600 而非 red-500
        if _STATE_NEGATIVE.search(stripped):
            if 'text-red-600' in stripped and 'font-medium' in stripped:
                report(filepath, i, "WARN",
                       f"负向状态（down/error/stopped）应用 text-red-500 而非 text-red-600: {stripped[:100]}")


def check_enum_badge_rounded(filepath, lines, tmpl, tmpl_line0):
    """
    6.9.2 枚举 badge（驱动、类型等）形状应用 rounded 或 rounded-lg，不得用 rounded-full。
    覆盖桌面端和移动端。
    """
    for i, line in enumerate(lines, 1):
        stripped = line.strip()
        if stripped.startswith('//') or stripped.startswith('*') or stripped.startswith('<!--'):
            continue
        # 含 rounded-full 且有 px-2 py-0.5（badge 特征）且无 w-（排除圆形图标容器）
        if ('rounded-full' in stripped
                and 'px-2' in stripped
                and ('py-0.5' in stripped or re.search(r'\bpy-1\b', stripped))
                and not re.search(r'\bw-\d+\b', stripped)):
            report(filepath, i, "WARN",
                   f"badge 应用 rounded 或 rounded-lg，不得用 rounded-full: {stripped[:100]}")


# ─── 主流程 ──────────

CHECKS = [
    check_toolbar_layout,
    check_toolbar_icon,
    check_empty_state,
    check_mobile_card_container,
    check_mobile_card_hover,
    check_mobile_card_top,
    check_mobile_card_rows,
    check_table_first_col,
    check_table_icon_size,
    check_table_text_colors,
    check_desktop_action_gap,
    check_mobile_action_gap,
    check_action_buttons,
    check_desktop_badge_shape,
    check_h_tags_in_mobile,
    # 6.9.1 状态文字颜色
    check_status_uses_text_color,
    check_status_text_color_values,
    # 6.9.2 枚举 badge 形状
    check_enum_badge_rounded,
]


def review_file(vue_file: Path):
    filepath = str(vue_file)
    content = vue_file.read_text(encoding='utf-8')
    lines = content.splitlines()
    tmpl = get_template(content)
    if not tmpl:
        return
    tmpl_line0 = find_line(lines, '<template>')
    for check in CHECKS:
        try:
            check(filepath, lines, tmpl, tmpl_line0)
        except Exception as e:
            report(filepath, 0, "ERROR", f"[脚本异常] {check.__name__}: {e}")


def collect_files() -> list[Path]:
    return sorted(
        f for f in VIEWS_DIR.rglob("*.vue")
        if not any(p in SKIP_DIRS for p in f.parts)
    )


def main():
    files = collect_files()
    print(f"{BOLD}{CYAN}=== 前端样式 Review ==={RESET}")
    print(f"扫描目录：{VIEWS_DIR}")
    print(f"文件数量：{len(files)}\n")

    for f in files:
        review_file(f)

    if not issues:
        print(f"{GREEN}{BOLD}✓ 未发现问题！{RESET}")
        return 0

    by_file: dict[str, list] = defaultdict(list)
    for filepath, lineno, level, msg in issues:
        by_file[filepath].append((lineno, level, msg))

    error_count = sum(1 for _, _, l, _ in issues if l == "ERROR")
    warn_count  = sum(1 for _, _, l, _ in issues if l == "WARN")

    for filepath, file_issues in sorted(by_file.items()):
        try:
            rel = Path(filepath).relative_to(VIEWS_DIR.parent.parent)
        except ValueError:
            rel = Path(filepath)
        print(f"\n{BOLD}{rel}{RESET}")
        for lineno, level, msg in sorted(file_issues, key=lambda x: x[0]):
            line_str = f":{lineno}" if lineno > 0 else ""
            if level == "ERROR":
                print(f"  {RED}[ERROR]{RESET}{line_str}  {msg}")
            else:
                print(f"  {YELLOW}[WARN]{RESET}{line_str}   {msg}")

    print(f"\n{BOLD}{'─'*60}{RESET}")
    print(f"共 {len(by_file)} 个文件有问题：{RED}{error_count} ERROR{RESET}，{YELLOW}{warn_count} WARN{RESET}")
    return 1 if error_count > 0 else 0


if __name__ == "__main__":
    sys.exit(main())