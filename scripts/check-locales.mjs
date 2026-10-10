#!/usr/bin/env node
/**
 * 多语言词条检查，以英文词典的 key 为全集。
 *
 * 默认检查前端 locales/messages/*.ts，加 --go 检查后端 server/i18n/lang_*.go。
 * 检查内容：
 *   1. 各语言是否缺词（相对英文词典）
 *   2. 是否有英文词典里已不存在的多余词条（改文案后遗留）
 *   3. 译文是否为空，或占位符与原文不一致（前端 {0}，后端 %s / %d ...）
 *   4. `--worklist` 导出待翻译清单（JSONL），交给人工或机翻补全
 *
 * 用法：
 *   node scripts/check-locales.mjs                  # 报告前端各语言缺口
 *   node scripts/check-locales.mjs --go             # 报告后端各语言缺口
 *   node scripts/check-locales.mjs --worklist ja    # 导出某语言的待翻译清单到 stdout
 *   node scripts/check-locales.mjs --strict         # 有缺口时以非零退出（用于 CI 逐步收紧）
 */

import { readdirSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')

const args = process.argv.slice(2)
const worklist = args.includes('--worklist') ? args.find(a => !a.startsWith('--')) : null
const strict = args.includes('--strict')
const go = args.includes('--go')

const dir = go ? join(root, 'server/i18n') : join(root, 'webview/src/locales/messages')

/** 后端：占位符是 %s / %d / %v / %w / %q / %T */
const VERB = /%[sdvwqT%]/g

const ENTRY = /^\s*('(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*")\s*:\s*('(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*")\s*,?\s*$/
const ENTRY_GO = /^\s*("(?:[^"\\]|\\.)*")\s*:\s*("(?:[^"\\]|\\.)*"),\s*$/

/** 还原字符串字面量（单双引号都支持） */
const unquote = (raw) => (raw[0] === '"' ? JSON.parse(raw) : raw.slice(1, -1).replace(/\\(.)/g, '$1'))

/** 读取一个语言文件里的词条 */
const readFile = (file) => {
    const entries = new Map()
    const re = go ? ENTRY_GO : ENTRY
    for (const line of readFileSync(file, 'utf8').split('\n')) {
        const m = line.match(re)
        if (m) entries.set(unquote(m[1]), unquote(m[2]))
    }
    return { code: file.replace(/\.ts$/, ''), entries }
}

// 后端按语言列出组成文件：英文是完整集，可分多个文件
const LANG_FILES = {
    en: ['lang_en.go', 'lang_en_service.go'],
    ja: ['lang_ja.go'],
    'zh-Hant': ['lang_zh_hant_messages.go'],
}
// 源语言只登记标签，没有词典；不参与检查
const SOURCE_LANG = 'zh-Hans'

const files = readdirSync(dir)
if (!go && !files.includes('en.ts')) {
    console.error('缺少 locales/messages/en.ts，英文词典是完整集')
    process.exit(1)
}

/** 读出一个语言的全部词条（后端可分多个文件） */
const loadLang = (code) => {
    const names = go ? (LANG_FILES[code] ?? []) : [`${code}.ts`]
    const entries = new Map()
    for (const name of names) {
        for (const [k, v] of readFile(join(dir, name)).entries) entries.set(k, v)
    }
    return entries
}
const codes = go
    ? Object.keys(LANG_FILES).filter(code => code !== 'en')
    : files.filter(f => f.endsWith('.ts')).map(f => f.replace(/\.ts$/, ''))
const en = loadLang('en')

// ─── 导出待翻译清单 ───

if (worklist) {
    if (!codes.includes(worklist)) {
        console.error(`没有语言 ${worklist}，现有：${codes.join(', ')}`)
        process.exit(1)
    }
    const target = loadLang(worklist)
    for (const [key, enValue] of en) {
        if (target.has(key)) continue
        process.stdout.write(`${JSON.stringify({ key, en: enValue })}\n`)
    }
    process.exit(0)
}

// ─── 检查 ───

let problems = 0
const line = (text) => { console.log(text); problems++ }

for (const code of codes.sort()) {
    const entries = loadLang(code)
    if (go && (LANG_FILES[code] ?? []).every(name => !files.includes(name))) {
        line(`  ✗ 后端未找到 ${code} 的词典文件`)
        continue
    }
    const missing = [...en.keys()].filter(k => !entries.has(k))
    const stale = [...entries.keys()].filter(k => !en.has(k))
    const empty = [...entries].filter(([, v]) => v.trim() === '')
    const placeholder = go ? VERB : /\{(\d+)\}/g
    const badPlaceholder = [...entries].filter(([k, v]) => {
        const a = [...k.matchAll(placeholder)].map(m => m[1] ?? m[0]).join(',')
        const b = [...v.matchAll(placeholder)].map(m => m[1] ?? m[0]).join(',')
        return a !== b
    })
    // 译文与原文完全相同：可能是尚未翻译的占位，也可能是专有名词，列出来由人工确认
    const identity = [...entries].filter(([k, v]) => v === k && /[一-鿿]/.test(k))

    const ratio = en.size ? Math.round(((en.size - missing.length) / en.size) * 100) : 100
    console.log(`${code.padEnd(8)} ${String(entries.size).padStart(5)} 条，覆盖 ${String(ratio).padStart(3)}%，缺失 ${missing.length}`)
    if (code === 'en') continue

    if (missing.length) line(`  ✗ 缺失 ${missing.length} 条，导出清单：node scripts/check-locales.mjs --worklist ${code}`)
    if (stale.length) {
        line(`  ✗ 多余 ${stale.length} 条（英文词典里已无此原文，多为改文案后遗留）`)
        for (const k of stale.slice(0, 8)) line(`      ${k}`)
    }
    if (empty.length) {
        line(`  ✗ 空译文 ${empty.length} 条`)
        for (const [k] of empty.slice(0, 8)) line(`      ${k}`)
    }
    if (badPlaceholder.length) {
        line(`  ✗ 占位符与原文不一致 ${badPlaceholder.length} 条`)
        for (const [k, v] of badPlaceholder.slice(0, 8)) line(`      ${k} => ${v}`)
    }
    // 繁體与简体同形的词条属正常（如「退出」「路由」），只在非繁體语言里提示
    if (identity.length && code !== 'zh-Hant') {
        line(`  ! 译文与中文原文完全相同 ${identity.length} 条，需人工确认是否已翻译`)
        for (const [k] of identity.slice(0, 8)) line(`      ${k}`)
    }
}

console.log(problems ? `\n共 ${problems} 处待处理` : '\n全部语言词条完整')
if (strict && problems) process.exit(1)
