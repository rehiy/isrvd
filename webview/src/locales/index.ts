/**
 * 前端国际化：以中文为 key
 *
 * 中文（zh-Hans）是源语言：模板与代码里直接书写中文，`t('中文文案')` 在中文下原样返回；
 * 其他语言查各自词典，未登记译文时同样回落到中文，不会出现空白或 key 泄漏。
 *
 * 语言由 `messages/` 目录自动发现：新增语言只需新建 `messages/<code>.ts`，
 * 默认导出一个 LanguageModule（名称、标签、词典），不需要改动本文件或任何其他代码。
 */

import type { LanguageModule } from './types'

/** 语言代码；取值来自 messages/ 目录，运行时校验 */
export type Locale = string

/** 源语言：代码里书写文案所用的语言，不需要词典 */
export const SOURCE: Locale = 'zh-Hans'

/** 语言展示信息（切换入口里以各语言自己的名称呈现，不参与翻译） */
export interface LocaleMeta {
    code: Locale
    label: string
    short: string
    /** 写入 <html lang> 的 BCP 47 标签 */
    tag: string
}

const SOURCE_META: LocaleMeta = { code: SOURCE, label: '简体中文', short: '简', tag: 'zh-CN' }

// 自动发现 messages/ 下的全部语言文件
const modules = import.meta.glob<{ default: LanguageModule }>('./messages/*.ts', { eager: true })

/** 全部可选语言：源语言在前，其余按代码排序 */
export const LOCALES: LocaleMeta[] = [
    SOURCE_META,
    ...Object.values(modules)
        .map(m => m.default)
        .sort((a, b) => a.code.localeCompare(b.code))
        .map(m => ({ code: m.code, label: m.label, short: m.short, tag: m.htmlLang ?? m.code }))
]

/** 语言词典：code → 中文原文 → 译文 */
const dictionaries = new Map<Locale, Record<string, string>>(
    Object.values(modules).map(m => [m.default.code, m.default.messages])
)

/** 浏览器语言标签 → 语言代码，用于首次访问时按浏览器语言选择 */
const tagIndex = new Map<string, Locale>([
    ['zh', SOURCE],
    ['zh-cn', SOURCE],
    ['zh-sg', SOURCE],
    ['zh-hans', SOURCE],
    ...Object.values(modules).flatMap(m => m.default.tags.map(tag => [tag.toLowerCase(), m.default.code] as [string, Locale]))
])

/** 判断取值是否是支持的语言 */
export const isLocale = (value: unknown): value is Locale =>
    typeof value === 'string' && LOCALES.some(l => l.code === value)

/** 取语言的展示信息，未知语言回落源语言 */
export const localeMeta = (locale: Locale): LocaleMeta => LOCALES.find(l => l.code === locale) ?? SOURCE_META

/** 按浏览器语言标签选择语言：逐级去掉末尾子标签匹配，无法识别时返回 undefined */
export const matchLocale = (tags: readonly string[]): Locale | undefined => {
    for (const raw of tags) {
        let tag = raw.toLowerCase()
        while (tag) {
            const hit = tagIndex.get(tag)
            if (hit) return hit
            const i = tag.lastIndexOf('-')
            if (i < 0) break
            tag = tag.slice(0, i)
        }
    }
    return undefined
}

/** 仅取词典自有属性，避免 constructor / __proto__ 等原型键被当成译文 */
const lookup = (locale: Locale, key: string): string | undefined => {
    const dict = dictionaries.get(locale)
    return dict && Object.prototype.hasOwnProperty.call(dict, key) ? dict[key] : undefined
}

/**
 * 按语言翻译文案，未命中译文时返回原文
 *
 * 参数按 {0}、{1} 占位一次性替换：已代入的参数值里即使含有 {n} 也不会被再次替换。
 */
export const translate = (key: string, locale: Locale, ...args: unknown[]): string => {
    if (typeof key !== 'string') return key
    const text = locale === SOURCE ? key : lookup(locale, key) || key
    if (args.length === 0) return text
    return text.replace(/\{(\d+)\}/g, (match, index: string) =>
        Number(index) < args.length ? String(args[Number(index)]) : match
    )
}

// 模板内使用的全局属性由 main.ts 注入：app.config.globalProperties.$t = t
declare module 'vue' {
    interface ComponentCustomProperties {
        $t: (key: string, ...args: unknown[]) => string
    }
}
