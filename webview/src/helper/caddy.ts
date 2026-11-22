import type { CaddyCert } from '@/service/types'

export function caddyCertSummary(cert: CaddyCert, t: (key: string) => string): string {
    if (cert.subject) {
        return cert.subject
    }
    if (cert.source === 'file') {
        return cert.certificate || '-'
    }
    return cert.certificate?.split('\n')[0].trim() || t('(空)')
}

export function caddyCertExpireLabel(notAfter: string | undefined, t: (key: string, ...args: unknown[]) => string): string {
    if (!notAfter) {
        return '-'
    }
    const days = Math.floor((new Date(notAfter).getTime() - Date.now()) / 86400000)
    if (days < 0) {
        return t('已过期 {0} 天', -days)
    }
    if (days === 0) {
        return t('今日过期')
    }
    return t('{0} 天后到期', days)
}

export function caddyConfigReloadConfirm(t: (key: string) => string, onConfirm: () => Promise<void>) {
    const title = t('放弃修改并重新加载')
    return {
        title,
        message: t('当前修改尚未保存，重新加载将丢弃这些修改。'),
        icon: 'fa-rotate',
        iconColor: 'amber',
        confirmText: title,
        danger: true,
        onConfirm,
    }
}
