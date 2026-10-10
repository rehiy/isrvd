#!/usr/bin/env node
/**
 * 把译文并回语言文件（保留英文注释、按英文词典顺序排列）。
 *
 * 输入文件为 JSONL，每行形如：{"key":"中文原文","ja":"译文","en":"English"}
 *   - key 与 ja 必填；en 可选，缺失时从英文词典补
 *   - 空译文的行会跳过，方便先导出清单再逐行填
 *
 * 用法：
 *   node scripts/apply-translations.mjs --lang ja --file tmp/ja-ui.jsonl --target webview
 *   node scripts/apply-translations.mjs --lang ja --file tmp/ja-api.jsonl --target server
 *
 * --target webview 写 webview/src/locales/messages/<lang>.ts，按英文词典顺序输出并保留英文注释；
 * --target server  写 server/i18n/lang_<lang>.go，追加到既有的 map 字面量里（已存在的 key 会被替换）。
 * 写入后会用 gofmt 格式化 Go 文件。写完请跑 scripts/check-locales.mjs 复核。
 */

import { execFileSync } from 'node:child_process'
import { readFileSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')

const arg = (name) => {
    const i = process.argv.indexOf(`--${name}`)
    return i < 0 ? null : process.argv[i + 1]
}
const lang = arg('lang')
const file = arg('file')
const target = arg('target')

if (!lang || !file || !target) {
    console.error('用法：node scripts/apply-translations.mjs --lang ja --file <jsonl> --target webview|server')
    process.exit(1)
}

const TS_ENTRY = /^\s*('(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*")\s*:\s*('(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*")\s*,?\s*$/
const GO_ENTRY = /^\s*("(?:[^"\\]|\\.)*")\s*:\s*("(?:[^"\\]|\\.)*"),\s*$/
const unquote = (raw) => (raw[0] === '"' ? JSON.parse(raw) : raw.slice(1, -1).replace(/\\(.)/g, '$1'))

/** 读取英文词典，用于补 en 注释与确定顺序 */
const readEntries = (path, re) => {
    const map = new Map()
    for (const line of readFileSync(path, 'utf8').split('\n')) {
        const m = line.match(re)
        if (m) map.set(unquote(m[1]), unquote(m[2]))
    }
    return map
}

const incoming = readFileSync(file, 'utf8')
    .split('\n')
    .filter(l => l.trim())
    .map(l => JSON.parse(l))
    .filter(e => e.key && e.ja)

if (incoming.length === 0) {
    console.error('没有可导入的译文（key 与译文都必填，且译文不能为空）')
    process.exit(1)
}

const quote = (v) => `'${v.replace(/\\/g, '\\\\').replace(/'/g, "\\'")}'`

if (target === 'webview') {
    const enPath = join(root, 'webview/src/locales/messages/en.ts')
    const en = readEntries(enPath, TS_ENTRY)
    const path = join(root, `webview/src/locales/messages/${lang}.ts`)
    const current = readEntries(path, TS_ENTRY)

    let unknown = 0
    for (const { key, ja } of incoming) {
        if (!en.has(key)) {
            console.error(`  ! 英文词典里没有「${key}」，已跳过`)
            unknown++
            continue
        }
        current.set(key, ja)
    }

    // 按英文词典顺序输出，保留英文原文作为注释
    const lines = []
    for (const [key, enValue] of en) {
        if (!current.has(key)) continue
        lines.push(`    // ${enValue}`)
        lines.push(`    ${quote(key)}: ${quote(current.get(key))},`)
    }
    const meta = readFileSync(path, 'utf8').match(/const language: LanguageModule = \{[\s\S]*$/)
    const head = readFileSync(path, 'utf8').slice(0, readFileSync(path, 'utf8').indexOf('const messages:'))
    const out = `${head}const messages: Record<string, string> = {\n${lines.join('\n')}\n}\n\n${meta ? meta[0] : ''}`
    writeFileSync(path, out, 'utf8')
    console.log(`已写入 ${path.replace(root + '/', '')}，共 ${current.size} 条${unknown ? `，跳过 ${unknown} 条` : ''}`)
} else {
    const en = new Map([
        ...readEntries(join(root, 'server/i18n/lang_en.go'), GO_ENTRY),
        ...readEntries(join(root, 'server/i18n/lang_en_service.go'), GO_ENTRY),
    ])
    const path = join(root, `server/i18n/lang_${lang}.go`)
    const source = readFileSync(path, 'utf8')
    const current = readEntries(path, GO_ENTRY)

    let unknown = 0
    for (const { key, ja } of incoming) {
        if (!en.has(key)) {
            console.error(`  ! 英文词典里没有「${key}」，已跳过`)
            unknown++
            continue
        }
        current.set(key, ja)
    }

    const body = [...current].map(([k, v]) => `\t\t${JSON.stringify(k)}: ${JSON.stringify(v)},`).join('\n')
    // 兼容两种写法：Register(langXX, map[string]string{...}) 与 Register(Language{...}, map[string]string{...})
    const out = source.replace(
        /(Register\((?:[\w.]+|Language\{[\s\S]*?\})\s*,\s*map\[string\]string\{\s*\n)([\s\S]*)(\n\s*\}\))/,
        (_, prefix, _old, suffix) => `${prefix}${body}${suffix}`
    )
    let formatted = out
    try {
        formatted = execFileSync('gofmt', [], { input: out, encoding: 'utf8' })
    } catch (e) {
        console.error('gofmt 失败，已按原样写入：' + e.message)
    }
    writeFileSync(path, formatted, 'utf8')
    console.log(`已写入 ${path.replace(root + '/', '')}，共 ${current.size} 条${unknown ? `，跳过 ${unknown} 条` : ''}`)
}
