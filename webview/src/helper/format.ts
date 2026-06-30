// 全局自动刷新间隔（毫秒），所有轮询定时器统一使用此常量
export const POLL_INTERVAL = 5000

/**
 * 解析 host:port 字符串，正确处理 IPv6 字面量地址。
 *
 * 规则（遵循 RFC 3986）：
 *   [::1]:8080   -> { host: '::1',       port: '8080' }
 *   ::1          -> { host: '::1',       port: ''     }  (裸 IPv6，无端口)
 *   127.0.0.1:80 -> { host: '127.0.0.1', port: '80'  }
 *   hostname:80  -> { host: 'hostname',  port: '80'   }
 *   hostname     -> { host: 'hostname',  port: ''     }
 */
export function parseHostPort(value: string): { host: string; port: string } {
    const s = value.trim()
    if (!s) return { host: '', port: '' }

    if (s.startsWith('[')) {
        const close = s.indexOf(']')
        if (close === -1) return { host: s, port: '' }
        const host = s.slice(1, close)
        const rest = s.slice(close + 1)
        const port = rest.startsWith(':') ? rest.slice(1) : ''
        return { host, port }
    }

    const colonCount = (s.match(/:/g) ?? []).length
    if (colonCount > 1) return { host: s, port: '' }

    const idx = s.lastIndexOf(':')
    if (idx <= 0) return { host: s, port: '' }
    return { host: s.slice(0, idx), port: s.slice(idx + 1) }
}

export const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

export const formatTime = (timeString: string): string => {
    const date = new Date(timeString)
    return date.toLocaleDateString('zh-CN') + ' ' + date.toLocaleTimeString('zh-CN', { hour12: false })
}

export const formatUnixTime = (ts: number): string => {
    return new Date(ts * 1000).toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

export const hexToRgba = (hex: string, alpha: number): string => {
    const r = parseInt(hex.slice(1, 3), 16)
    const g = parseInt(hex.slice(3, 5), 16)
    const b = parseInt(hex.slice(5, 7), 16)
    return `rgba(${r},${g},${b},${alpha})`
}

/**
 * 拼接路径，自动处理末尾多余的斜杠
 * joinPath('/foo/', 'bar') -> '/foo/bar'
 */
export const joinPath = (...parts: string[]): string => {
    return parts.reduce((acc, part) => acc.replace(/\/+$/, '') + '/' + part.replace(/^\/+/, ''))
}
