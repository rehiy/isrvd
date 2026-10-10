/**
 * 前端国际化：以中文为 key
 *
 * 中文是默认语言，也是模板与代码中的书写语言：`t('中文文案')` 在中文下直接返回原文，
 * 英文下查 `locales/en.ts` 的译文，未登记译文时同样回落到中文，不会出现空白或 key 泄漏。
 * 新增可翻译文案时：模板里保持中文原文，只在 `locales/en.ts` 补充对应英文。
 */

import enMessages from './en'

/** 支持的语言 */
export type Locale = 'zh' | 'en'

/** 语言切换入口的展示信息（各语言以自己的名称呈现，不参与翻译） */
export const LOCALE_META: Record<Locale, { label: string; short: string }> = {
    zh: { label: '简体中文', short: '中' },
    en: { label: 'English', short: 'EN' }
}

/** 仅取词典自有属性，避免 constructor / __proto__ 等原型键被当成译文 */
const lookup = (key: string): string | undefined =>
    Object.prototype.hasOwnProperty.call(enMessages, key) ? enMessages[key] : undefined

/**
 * 按语言翻译文案，未命中译文时返回原文
 *
 * 参数按 {0}、{1} 占位一次性替换：已代入的参数值里即使含有 {n} 也不会被再次替换。
 */
export const translate = (key: string, locale: Locale, ...args: unknown[]): string => {
    if (typeof key !== 'string') return key
    const text = locale === 'zh' ? key : lookup(key) || key
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
