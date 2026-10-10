import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

import { isLocale, localeMeta, LOCALES, matchLocale, SOURCE, translate, type Locale } from '@/locales'

// ─── 本地存储 ───

const LOCALE_KEY = 'app-locale'

/**
 * 读取界面语言：已保存的选择优先，其次按浏览器语言，最后回落源语言（中文）
 *
 * 只有用户主动切换过才会写入存储，所以未切换时始终跟随浏览器语言。
 */
const readLocale = (): Locale => {
    const saved = localStorage.getItem(LOCALE_KEY)
    if (isLocale(saved)) return saved
    return matchLocale(navigator.languages ?? [navigator.language]) ?? SOURCE
}

/** 保存用户的选择 */
const writeLocale = (locale: Locale): void => {
    localStorage.setItem(LOCALE_KEY, locale)
}

/** 同步 <html lang> */
const applyLocale = (locale: Locale): void => {
    document.documentElement.lang = localeMeta(locale).tag
}

/**
 * Locale Store - 语言设置
 *
 * 中文是源语言，也是文案的 key；切换语言后所有经 t 输出的文案与请求语言同步生效。
 */
export const useLocaleStore = defineStore('locale', () => {
    // ─── 状态定义 ───

    const locale = ref<Locale>(readLocale())

    /** 当前语言的展示信息 */
    const meta = computed(() => localeMeta(locale.value))

    /** 全部可选语言，供切换入口渲染 */
    const locales = LOCALES

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
        if (!isLocale(next)) return
        writeLocale(next)
        locale.value = next
        applyLocale(next)
    }

    /** 启动时生效（同步 <html lang>） */
    function initLocale(): void {
        applyLocale(locale.value)
    }

    return { locale, meta, locales, t, setLocale, initLocale }
})
