import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

import { LOCALE_META, translate, type Locale } from '@/locales'

// ─── 本地存储 ───

const LOCALE_KEY = 'app-locale'

/** 读取已保存的语言，未保存时回落中文 */
const readLocale = (): Locale => (localStorage.getItem(LOCALE_KEY) === 'en' ? 'en' : 'zh')

/** 中文是默认语言，不写入存储；其他语言持久化 */
const writeLocale = (locale: Locale): void => {
    if (locale === 'zh') {
        localStorage.removeItem(LOCALE_KEY)
    } else {
        localStorage.setItem(LOCALE_KEY, locale)
    }
}

const applyLocale = (locale: Locale): void => {
    document.documentElement.lang = locale === 'en' ? 'en-US' : 'zh-CN'
}

/**
 * Locale Store - 语言设置
 *
 * 中文为默认语言，也是文案的 key；切换语言后所有经 t 输出的文案与请求语言同步生效。
 */
export const useLocaleStore = defineStore('locale', () => {
    // ─── 状态定义 ───

    const locale = ref<Locale>(readLocale())

    /** 当前语言的展示信息 */
    const meta = computed(() => LOCALE_META[locale.value])

    // ─── 翻译 ───

    /**
     * 翻译文案，未登记译文时返回中文原文
     *
     * 动态片段用 `{0}`、`{1}`... 占位并依次由 args 填充
     */
    function t(key: string, ...args: unknown[]): string {
        return translate(key, locale.value, ...args)
    }

    // ─── 切换 ───

    function setLocale(next: Locale): void {
        writeLocale(next)
        locale.value = next
        applyLocale(next)
    }

    /** 在中英之间切换，返回切换后的语言 */
    function toggleLocale(): Locale {
        const next: Locale = locale.value === 'zh' ? 'en' : 'zh'
        setLocale(next)
        return next
    }

    /** 启动时按本地记录生效（同步 <html lang>） */
    function initLocale(): void {
        applyLocale(locale.value)
    }

    return { locale, meta, t, setLocale, toggleLocale, initLocale }
})
