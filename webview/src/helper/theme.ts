export type ThemeMode = 'light' | 'dark' | 'system'

const THEME_KEY = 'app-theme'
const THEME_ORDER: ThemeMode[] = ['system', 'light', 'dark']

export const THEME_META: Record<ThemeMode, { icon: string; label: string }> = {
    light: { icon: 'fas fa-sun', label: '浅色模式' },
    dark: { icon: 'fas fa-moon', label: '深色模式' },
    system: { icon: 'fas fa-desktop', label: '跟随系统' }
}

const prefersDark = (): boolean => window.matchMedia('(prefers-color-scheme: dark)').matches

const applyTheme = (mode: ThemeMode): void => {
    const isDark = mode === 'dark' || (mode === 'system' && prefersDark())
    document.documentElement.classList.toggle('dark', isDark)
}

/** 读取已保存的主题模式，未保存则视为 system */
export const getThemeMode = (): ThemeMode => {
    const stored = localStorage.getItem(THEME_KEY)
    return stored === 'light' || stored === 'dark' ? stored : 'system'
}

/** 根据 localStorage 或系统偏好初始化主题，并在 system 模式下跟随系统切换 */
export const initTheme = (): void => {
    applyTheme(getThemeMode())
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
        if (getThemeMode() === 'system') {
            applyTheme('system')
        }
    })
}

/** 设置主题：light/dark 持久化，system 清除存储以恢复跟随系统 */
export const setThemeMode = (mode: ThemeMode): void => {
    if (mode === 'system') {
        localStorage.removeItem(THEME_KEY)
    } else {
        localStorage.setItem(THEME_KEY, mode)
    }
    applyTheme(mode)
}

/** 在 system → light → dark → system 三态间循环切换 */
export const cycleTheme = (): ThemeMode => {
    const next = THEME_ORDER[(THEME_ORDER.indexOf(getThemeMode()) + 1) % THEME_ORDER.length]
    setThemeMode(next)
    return next
}
