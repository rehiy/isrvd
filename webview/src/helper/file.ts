const BINARY_EXTENSIONS = new Set([
    // 图片 / 音视频 / 字体
    'jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'ico', 'icns', 'tif', 'tiff', 'psd', 'heic', 'avif',
    'mp3', 'wav', 'ogg', 'm4a', 'flac', 'aac', 'wma', 'mid', 'midi',
    'mp4', 'webm', 'avi', 'mov', 'm4v', 'mkv', 'flv', 'wmv', 'mpg', 'mpeg',
    'ttf', 'otf', 'woff', 'woff2', 'eot',
    // 压缩包 / 镜像 / 安装包
    'zip', 'rar', '7z', 'tar', 'gz', 'tgz', 'bz2', 'tbz2', 'xz', 'txz', 'zst', 'lz', 'lz4', 'z', 'cab',
    'iso', 'img', 'dmg', 'deb', 'rpm', 'apk', 'ipa', 'jar', 'war', 'ear', 'whl', 'egg', 'gem', 'nupkg',
    // 可执行 / 库 / 编译产物
    'exe', 'dll', 'so', 'dylib', 'bin', 'o', 'a', 'obj', 'lib', 'class', 'pyc', 'pyo', 'wasm', 'node',
    // 文档 / 数据库 / 其他二进制
    'pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'odt', 'ods', 'odp', 'epub',
    'db', 'sqlite', 'sqlite3', 'mdb', 'dat', 'pack', 'idx', 'parquet', 'der', 'p12', 'pfx', 'jks', 'keystore'
])

export type PreviewFileType = 'image' | 'audio' | 'video' | 'pdf' | ''

const PREVIEW_MIME_MAP: Record<string, string> = {
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

const FILE_ICON_MAP: Record<string, string> = {
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

const BACKUP_SUFFIXES = ['bak', 'orig', 'old', 'backup', 'tmp', 'swp']

const fileExtension = (filename: string): string => {
    const idx = filename.lastIndexOf('.')
    return idx < 0 ? '' : filename.slice(idx + 1).toLowerCase()
}

// 忽略备份后缀后的真实扩展名；无扩展名返回空串
const effectiveExtension = (filename: string): string => {
    let name = filename.toLowerCase()
    while (BACKUP_SUFFIXES.includes(fileExtension(name)) && name.lastIndexOf('.') > 0) {
        name = name.slice(0, name.lastIndexOf('.'))
    }
    const idx = name.lastIndexOf('.')
    return idx <= 0 ? '' : name.slice(idx + 1)
}

/**
 * 是否允许在线编辑：采用「排除已知二进制扩展名 + 大小限制」策略，
 * 无扩展名或未知扩展名的文件默认允许，真正的二进制内容由后端读取时嗅探拒绝。
 */
export const isEditableFile = (filename: string, size?: number, maxSize?: number): boolean => {
    if (!filename) return false
    if (typeof size === 'number' && typeof maxSize === 'number' && size > maxSize) return false
    return !BINARY_EXTENSIONS.has(effectiveExtension(filename))
}

export const getPreviewType = (filename: string): PreviewFileType => {
    if (!filename) return ''
    const mime = PREVIEW_MIME_MAP[fileExtension(filename)]
    if (!mime) return ''
    if (mime === 'application/pdf') return 'pdf'
    const type = mime.split('/')[0]
    return type === 'image' || type === 'audio' || type === 'video' ? type : ''
}

export const isPreviewableFile = (filename: string): boolean => getPreviewType(filename) !== ''

export const getPreviewMimeType = (filename: string): string => {
    return PREVIEW_MIME_MAP[fileExtension(filename)] || 'application/octet-stream'
}

export const getFileIcon = (file: { isDir: boolean; name: string }): string => {
    if (file.isDir) return 'fas fa-folder text-warning'
    return FILE_ICON_MAP[fileExtension(file.name)] || 'fas fa-file text-secondary'
}

/**
 * 用 Blob 触发浏览器下载，下载完成后自动释放 ObjectURL
 */
export const downloadBlob = (blob: Blob, filename: string): void => {
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.click()
    setTimeout(() => URL.revokeObjectURL(url), 10000)
}
