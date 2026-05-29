// 全局自动刷新间隔（毫秒），所有轮询定时器统一使用此常量
export const POLL_INTERVAL = 3000

/**
 * 解析 host:port 字符串，正确处理 IPv6 字面量地址。
 *
 * 规则（遵循 RFC 3986）：
 *   [::1]:8080   → { host: '::1',       port: '8080' }
 *   ::1          → { host: '::1',       port: ''     }  (裸 IPv6，无端口)
 *   127.0.0.1:80 → { host: '127.0.0.1', port: '80'  }
 *   hostname:80  → { host: 'hostname',  port: '80'   }
 *   hostname     → { host: 'hostname',  port: ''     }
 */
export function parseHostPort(value: string): { host: string; port: string } {
    const s = value.trim()
    if (!s) return { host: '', port: '' }

    // RFC 3986 带方括号的 IPv6：[::1] 或 [::1]:8080
    if (s.startsWith('[')) {
        const close = s.indexOf(']')
        if (close === -1) return { host: s, port: '' }
        const host = s.slice(1, close)
        const rest = s.slice(close + 1)
        const port = rest.startsWith(':') ? rest.slice(1) : ''
        return { host, port }
    }

    // 裸 IPv6（含多个冒号，不带端口）：:: / ::1 / 2001:db8::1
    const colonCount = (s.match(/:/g) ?? []).length
    if (colonCount > 1) return { host: s, port: '' }

    // IPv4 或 hostname：host 或 host:port
    const idx = s.lastIndexOf(':')
    if (idx <= 0) return { host: s, port: '' }
    return { host: s.slice(0, idx), port: s.slice(idx + 1) }
}

export const TEXT_EXTENSIONS: string[] = [
    'txt', 'md', 'js', 'css', 'html', 'htm', 'json', 'xml', 'csv',
    'log', 'conf', 'ini', 'cfg', 'yaml', 'yml', 'php', 'py', 'go',
    'java', 'cpp', 'c', 'h', 'sql', 'sh', 'bat', 'env'
]

export type PreviewFileType = 'image' | 'audio' | 'video' | 'pdf' | ''

export const PREVIEW_MIME_MAP: Record<string, string> = {
    jpg: 'image/jpeg',
    jpeg: 'image/jpeg',
    png: 'image/png',
    gif: 'image/gif',
    bmp: 'image/bmp',
    svg: 'image/svg+xml',
    webp: 'image/webp',
    ico: 'image/x-icon',
    tiff: 'image/tiff',
    tif: 'image/tiff',
    mp3: 'audio/mpeg',
    wav: 'audio/wav',
    ogg: 'audio/ogg',
    m4a: 'audio/mp4',
    flac: 'audio/flac',
    aac: 'audio/aac',
    mp4: 'video/mp4',
    webm: 'video/webm',
    mov: 'video/quicktime',
    m4v: 'video/x-m4v',
    mkv: 'video/x-matroska',
    pdf: 'application/pdf'
}

export const PREVIEW_TYPE_MAP: Record<string, PreviewFileType> = {
    jpg: 'image',
    jpeg: 'image',
    png: 'image',
    gif: 'image',
    bmp: 'image',
    svg: 'image',
    webp: 'image',
    ico: 'image',
    tiff: 'image',
    tif: 'image',
    mp3: 'audio',
    wav: 'audio',
    ogg: 'audio',
    m4a: 'audio',
    flac: 'audio',
    aac: 'audio',
    mp4: 'video',
    webm: 'video',
    mov: 'video',
    m4v: 'video',
    mkv: 'video',
    pdf: 'pdf'
}

export const FILE_ICON_MAP: Record<string, string> = {
    'txt': 'fas fa-file-alt text-secondary',
    'md': 'fab fa-markdown text-dark',
    'pdf': 'fas fa-file-pdf text-danger',
    'doc': 'fas fa-file-word text-primary',
    'docx': 'fas fa-file-word text-primary',
    'xls': 'fas fa-file-excel text-success',
    'xlsx': 'fas fa-file-excel text-success',
    'ppt': 'fas fa-file-powerpoint text-warning',
    'pptx': 'fas fa-file-powerpoint text-warning',
    'zip': 'fas fa-file-archive text-warning',
    'rar': 'fas fa-file-archive text-warning',
    '7z': 'fas fa-file-archive text-warning',
    'tar': 'fas fa-file-archive text-warning',
    'gz': 'fas fa-file-archive text-warning',
    'jpg': 'fas fa-file-image text-info',
    'jpeg': 'fas fa-file-image text-info',
    'png': 'fas fa-file-image text-info',
    'gif': 'fas fa-file-image text-info',
    'bmp': 'fas fa-file-image text-info',
    'svg': 'fas fa-file-image text-info',
    'mp3': 'fas fa-file-audio text-success',
    'wav': 'fas fa-file-audio text-success',
    'ogg': 'fas fa-file-audio text-success',
    'm4a': 'fas fa-file-audio text-success',
    'flac': 'fas fa-file-audio text-success',
    'aac': 'fas fa-file-audio text-success',
    'mp4': 'fas fa-file-video text-danger',
    'webm': 'fas fa-file-video text-danger',
    'avi': 'fas fa-file-video text-danger',
    'mov': 'fas fa-file-video text-danger',
    'm4v': 'fas fa-file-video text-danger',
    'mkv': 'fas fa-file-video text-danger',
    'js': 'fab fa-js-square text-warning',
    'html': 'fab fa-html5 text-danger',
    'css': 'fab fa-css3-alt text-primary',
    'php': 'fab fa-php text-purple',
    'py': 'fab fa-python text-info',
    'java': 'fab fa-java text-danger',
    'cpp': 'fas fa-file-code text-info',
    'c': 'fas fa-file-code text-info',
    'go': 'fas fa-file-code text-primary',
    'sql': 'fas fa-database text-secondary'
}

export const isEditableFile = (filename: string): boolean => {
    if (!filename) return false
    const ext = filename.split('.').pop()?.toLowerCase() ?? ''
    return TEXT_EXTENSIONS.includes(ext)
}

export const getPreviewType = (filename: string): PreviewFileType => {
    if (!filename) return ''
    const ext = filename.split('.').pop()?.toLowerCase() ?? ''
    return PREVIEW_TYPE_MAP[ext] || ''
}

export const isPreviewableFile = (filename: string): boolean => getPreviewType(filename) !== ''

export const getPreviewMimeType = (filename: string): string => {
    const ext = filename.split('.').pop()?.toLowerCase() ?? ''
    return PREVIEW_MIME_MAP[ext] || 'application/octet-stream'
}

export const getFileIcon = (file: { isDir: boolean; name: string }): string => {
    if (file.isDir) {
        return 'fas fa-folder text-warning'
    }
    const ext = file.name.split('.').pop()?.toLowerCase() ?? ''
    return FILE_ICON_MAP[ext] || 'fas fa-file text-secondary'
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

export const downloadFile = (filename: string, data: BlobPart): void => {
    const url = window.URL.createObjectURL(new Blob([data]))
    const link = document.createElement('a')
    link.href = url
    link.download = filename
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
}

type SearchInputRef = HTMLInputElement | HTMLInputElement[] | null | undefined

const isEditableElement = (el: Element | null): boolean => {
    if (!el) return false
    if (el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement || el instanceof HTMLSelectElement) {
        return true
    }
    return el instanceof HTMLElement && el.isContentEditable
}

const resolveSearchInput = (inputRef: SearchInputRef): HTMLInputElement | null => {
    if (!inputRef) return null
    if (Array.isArray(inputRef)) {
        return inputRef.find((el: HTMLInputElement) => el && el.offsetParent !== null) || null
    }
    return inputRef
}

// 在页面空白区域直接键入时，将输入重定向到搜索框。
export const bindTypeToSearchFocus = (getInput: () => SearchInputRef): (() => void) => {
    const handleKeydown = (event: KeyboardEvent) => {
        if (event.defaultPrevented || event.isComposing) return
        if (event.ctrlKey || event.metaKey || event.altKey) return
        const isPrintable = event.key.length === 1 && event.key.trim() !== ''
        const isDeleteKey = event.key === 'Backspace' || event.key === 'Delete'
        if (!isPrintable && !isDeleteKey) return
        if (document.querySelector('.modal-card')) return
        if (isEditableElement(document.activeElement)) return

        const input = resolveSearchInput(getInput())
        if (!input || input.disabled || input.readOnly) return

        event.preventDefault()
        let nextValue = input.value
        if (isPrintable) {
            nextValue = `${input.value}${event.key}`
        } else if (event.key === 'Backspace') {
            nextValue = input.value.slice(0, -1)
        } else if (event.key === 'Delete') {
            nextValue = ''
        }

        input.focus()
        input.value = nextValue
        input.dispatchEvent(new Event('input', { bubbles: true }))
        const cursor = nextValue.length
        input.setSelectionRange(cursor, cursor)
        requestAnimationFrame(() => input.focus())
    }

    window.addEventListener('keydown', handleKeydown)
    return () => window.removeEventListener('keydown', handleKeydown)
}

/**
 * 复制文本到剪贴板
 * 优先使用 Clipboard API，降级到 execCommand
 * @returns 是否复制成功
 */
export const copyToClipboard = async (text: string): Promise<boolean> => {
    try {
        if (navigator.clipboard?.writeText) {
            await navigator.clipboard.writeText(text)
            return true
        }
        // 降级方案
        const el = document.createElement('textarea')
        el.value = text
        el.style.cssText = 'position:fixed;top:-9999px;left:-9999px;opacity:0'
        document.body.appendChild(el)
        el.select()
        const ok = document.execCommand('copy')
        document.body.removeChild(el)
        return ok
    } catch {
        return false
    }
}

