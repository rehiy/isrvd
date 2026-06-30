export const TEXT_EXTENSIONS: string[] = [
    'txt', 'text', 'md', 'markdown', 'rst', 'adoc', 'log', 'csv', 'tsv',
    'json', 'jsonc', 'json5', 'xml', 'toml', 'yaml', 'yml', 'ini', 'conf', 'cfg', 'cnf',
    'env', 'envrc', 'properties', 'editorconfig', 'gitignore', 'gitattributes', 'dockerignore',
    'npmrc', 'yarnrc', 'pnpmrc', 'nvmrc', 'node-version', 'python-version',
    'ruby-version', 'tool-versions', 'bashrc', 'bash_profile', 'zshrc', 'zprofile',
    'profile', 'vimrc', 'gvimrc', 'curlrc', 'wgetrc', 'inputrc',
    'js', 'jsx', 'ts', 'tsx', 'mjs', 'cjs', 'vue', 'svelte', 'astro',
    'css', 'scss', 'sass', 'less', 'html', 'htm', 'xhtml', 'svg',
    'go', 'py', 'pyw', 'rb', 'php', 'java', 'kt', 'kts', 'scala', 'groovy',
    'c', 'h', 'cc', 'cpp', 'cxx', 'hpp', 'hh', 'cs', 'rs', 'swift', 'm', 'mm',
    'sh', 'bash', 'zsh', 'fish', 'ps1', 'bat', 'cmd', 'sql', 'lua', 'pl', 'pm',
    'r', 'dart', 'ex', 'exs', 'erl', 'hrl', 'clj', 'cljs', 'fs', 'fsx', 'vb',
    'tf', 'tfvars', 'hcl', 'dockerfile', 'compose', 'lock', 'mod', 'sum',
    'makefile', 'rakefile', 'gemfile', 'podfile', 'procfile',
    'tmpl', 'tpl', 'template', 'mustache', 'hbs'
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

const editableExtension = (filename: string): string => {
    const parts = filename.toLowerCase().split('.').filter(Boolean)
    if (parts.length === 0) return ''
    if (parts.length > 1 && parts[parts.length - 1] === 'bak') return parts[parts.length - 2]
    return parts[parts.length - 1]
}

export const isEditableFile = (filename: string): boolean => {
    if (!filename) return false
    return TEXT_EXTENSIONS.includes(editableExtension(filename))
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
    if (file.isDir) return 'fas fa-folder text-warning'
    const ext = file.name.split('.').pop()?.toLowerCase() ?? ''
    return FILE_ICON_MAP[ext] || 'fas fa-file text-secondary'
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
