// ─── 文件管理相关 ───

export interface FilerFileInfo {
    name: string
    path: string
    size: number
    mode: string
    modTime: string
    isDir: boolean
}

export interface FilerListResponse {
    path: string
    files: FilerFileInfo[]
}

export interface FilerReadResponse {
    path: string
    content: string
}
