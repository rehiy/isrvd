// 全局自动刷新间隔（毫秒），所有轮询定时器统一使用此常量
export const POLL_INTERVAL = 3000

export const TEXT_EXTENSIONS: string[] = [
    'txt', 'md', 'js', 'css', 'html', 'htm', 'json', 'xml', 'csv',
    'log', 'conf', 'ini', 'cfg', 'yaml', 'yml', 'php', 'py', 'go',
    'java', 'cpp', 'c', 'h', 'sql', 'sh', 'bat', 'env'
]

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
    'mp4': 'fas fa-file-video text-danger',
    'avi': 'fas fa-file-video text-danger',
    'mov': 'fas fa-file-video text-danger',
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
