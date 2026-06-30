// ─── 文件管理相关 ───

export interface FilerFileInfo {
    name: string
    path: string
    size: number
    mode: string
    modeO: string
    modTime: string
    isDir: boolean
    isLink?: boolean
    linkTarget?: string
}

export interface FilerList {
    path: string
    files: FilerFileInfo[]
}

export interface FilerRead {
    path: string
    content: string
}
