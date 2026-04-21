// ─── 文件管理相关 ───

export interface FileInfo {
    name: string
    path: string
    size: number
    mode: string
    modTime: string
    isDir: boolean
}

export interface FileListResponse {
    path: string
    files: FileInfo[]
}

export interface FileReadResponse {
    path: string
    content: string
}
