#!/usr/bin/env node
/**
 * 繁體中文（臺灣）词条生成器。
 *
 * 繁體与简体同源，不需要逐条翻译：以英文词典的 key（简体中文原文）为全集，
 * 经「术语表 + 上下文规则 + OpenCC 字形转换」得到译文。前后端共用同一份规则，
 * 所以两侧的繁體用词天然一致。
 *
 *   前端  webview/src/locales/messages/en.ts              →  messages/zh-Hant.ts
 *   后端  server/i18n/lang_en.go、lang_en_service.go      →  lang_zh_hant_messages.go
 *
 * 用法：node scripts/gen-zh-hant.mjs [--check]
 *   --check  只校验生成物是否是最新，不写文件（用于 CI）
 *
 * 术语表只登记「两岸用词不同、无法按字形推导」的词，其余与简体同形。
 * 需要调整用词时改下面的 TERMS / RULES，然后重新执行本脚本；生成物不要手工编辑。
 */

import { execFileSync } from 'node:child_process'
import { readFileSync, writeFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const check = process.argv.includes('--check')

// opencc-js 装在 webview 下（只用于生成词条，不进运行时包体）
const require = createRequire(import.meta.url)
let OpenCC
try {
    OpenCC = require(join(root, 'webview/node_modules/opencc-js/dist/umd/full.js'))
} catch {
    console.error('未找到 opencc-js，请先在 webview 目录执行 npm install')
    process.exit(1)
}

// ─── 转换规则 ───

/** 简体 → 繁體的术语表：在字形转换之前替换，长词优先 */
const TERMS = [
    ['内存', '記憶體'],
    ['信息', '資訊'],
    ['进程', '程序'],
    ['账号', '帳號'],
    ['账户', '帳戶'],
    ['阈值', '臨界值'],
    ['缺省', '預設'],
    ['默认', '預設'],
    ['计划任务', '計劃任務'],
    ['镜像仓库', '鏡像倉庫'],
].sort((a, b) => b[0].length - a[0].length)

/**
 * 依赖上下文的规则：「获取+宾语」在台湾习惯说「取得」，
 * 但「获取列表」这类固定搭配保持不变，所以排除后接「列」的情况。
 */
const RULES = [[/获取(?=[\u4e00-\u9fff])(?!列)/g, '取得']]

const toTW = OpenCC.Converter({ from: 'cn', to: 'tw' })

/** 简体原文 → 繁體译文：术语表 → 上下文规则 → 字形转换 */
const convert = (text) => {
    let out = text
    for (const [cn, tw] of TERMS) out = out.split(cn).join(tw)
    for (const [pattern, tw] of RULES) out = out.replace(pattern, tw)
    return toTW(out)
}

// ─── 读取英文词典的 key ───

/** 还原字符串字面量里的转义（单双引号都支持） */
const unquote = (raw) => (raw[0] === '"' ? JSON.parse(raw) : raw.slice(1, -1).replace(/\\(.)/g, '$1'))

/** 前端：messages/en.ts 的 'key': 'value' 词条，保留词条前的分组注释 */
const readFrontend = (file) => {
    const items = []
    const entry = /^\s*('(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*")\s*:\s*('(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*")\s*,?\s*$/
    for (const line of readFileSync(file, 'utf8').split('\n')) {
        const m = line.match(entry)
        if (m) items.push({ key: unquote(m[1]), en: unquote(m[2]) })
        else if (/^\s*\/\/ ─── /.test(line)) items.push({ section: line.trim() })
    }
    return items
}

/** 后端：lang_en*.go 的 "key": "value" 词条 */
const readBackend = (file) => {
    const items = []
    const entry = /^\s*("(?:[^"\\]|\\.)*")\s*:\s*("(?:[^"\\]|\\.)*"),\s*$/
    for (const line of readFileSync(file, 'utf8').split('\n')) {
        const m = line.match(entry)
        if (m) items.push({ key: JSON.parse(m[1]), en: JSON.parse(m[2]) })
        else if (/^\s*\/\/ ─── /.test(line)) items.push({ section: line.trim() })
    }
    return items
}

// ─── 生成 ───

const quoteTS = (value) => `'${value.replace(/\\/g, '\\\\').replace(/'/g, "\\'")}'`

const frontend = readFrontend(join(root, 'webview/src/locales/messages/en.ts'))
const backend = [
    ...readBackend(join(root, 'server/i18n/lang_en.go')),
    ...readBackend(join(root, 'server/i18n/lang_en_service.go')),
]

const countOf = (items) => items.filter((i) => i.key !== undefined).length
if (countOf(frontend) === 0 || countOf(backend) === 0) {
    console.error('未能解析出词条，请检查英文词典的格式')
    process.exit(1)
}

const frontendTS = `import type { LanguageModule } from '../types'

/**
 * 繁體中文（臺灣）譯文：以簡體中文原文為 key
 *
 * 由 scripts/gen-zh-hant.mjs 從英文詞典的 key 生成（術語表 + 字形轉換），請勿手工編輯；
 * 需要調整用詞時改生成腳本的 TERMS / RULES 後重新執行。
 */

const messages: Record<string, string> = {
${frontend
    .map((i) => (i.section ? `\n    ${i.section}` : `    ${quoteTS(i.key)}: ${quoteTS(convert(i.key))},`))
    .join('\n')}
}

const language: LanguageModule = {
    code: 'zh-Hant',
    label: '繁體中文',
    short: '繁',
    htmlLang: 'zh-TW',
    tags: ['zh-hant', 'zh-tw', 'zh-hk', 'zh-mo'],
    messages
}

export default language
`

const seen = new Set()
const backendGo = `package i18n

// 繁體中文（臺灣）：中文原文 → 繁體译文。
//
// 本文件由 scripts/gen-zh-hant.mjs 从英文词典的 key 生成（术语表 + 字形转换），
// 与前端 locales/messages/zh-Hant.ts 共用同一份规则；请勿手工编辑，
// 需要调整用词时改生成脚本的 TERMS / RULES 后重新执行。

func init() {
	Register(Language{Code: "zh-Hant", Tags: []string{"zh-hant", "zh-tw", "zh-hk", "zh-mo"}}, map[string]string{
${backend
    .filter((i) => {
        if (i.section) return true
        if (seen.has(i.key)) return false
        seen.add(i.key)
        return true
    })
    .map((i) => (i.section ? `\n\t\t${i.section}` : `\t\t${JSON.stringify(i.key)}: ${JSON.stringify(convert(i.key))},`))
    .join('\n')}
	})
}
`

/** Go 源码用 gofmt 规范化对齐，保证生成物与 gofmt 的结果一致，--check 才不会误报 */
const gofmt = (source) => {
    try {
        return execFileSync('gofmt', [], { input: source, encoding: 'utf8' })
    } catch {
        console.error('未找到 gofmt（或生成的 Go 源码有语法错误），请确认已安装 Go')
        process.exit(1)
    }
}

const outputs = [
    [join(root, 'webview/src/locales/messages/zh-Hant.ts'), frontendTS],
    [join(root, 'server/i18n/lang_zh_hant_messages.go'), gofmt(backendGo)],
]

let stale = false
for (const [file, content] of outputs) {
    if (check) {
        let current = ''
        try {
            current = readFileSync(file, 'utf8')
        } catch {
            /* 文件不存在视为过期 */
        }
        if (current !== content) {
            console.error(`过期：${file.replace(root + '/', '')}`)
            stale = true
        }
    } else {
        writeFileSync(file, content, 'utf8')
        console.log(`已生成 ${file.replace(root + '/', '')}`)
    }
}

if (check) {
    if (stale) {
        console.error('请执行 node scripts/gen-zh-hant.mjs 重新生成')
        process.exit(1)
    }
    console.log('繁體词条是最新的')
} else {
    console.log(`前端 ${countOf(frontend)} 条，后端 ${seen.size} 条`)
}
