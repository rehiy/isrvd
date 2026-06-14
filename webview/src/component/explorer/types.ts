/**
 * 统一文件管理抽象层
 *
 * 使用方式：
 *   - 本地文件管理：import { createFilerAdapter } from './adapters/filer'
 *   - SSH 文件管理：import { createSftpAdapter } from './adapters/sftp'
 *
 * 两种适配器都实现 ExplorerAdapter 接口，ExplorerPanel 只依赖该接口。
 */

// ─── 统一文件信息类型 ───────────────────────────────────────────────────────────

export interface FileInfo {
    name: string
    path: string         // 完整绝对路径
    size: number
    mode: string         // 字符串权限，如 "-rw-r--r--" 或 "drwxr-xr-x"
    modeOctal: string    // 八进制权限，如 "644"、"755"
    modTime: string | number  // ISO 字符串（filer）或 Unix 时间戳（sftp）
    isDir: boolean
    isLink?: boolean
    linkTarget?: string
}

// ─── 目录列表结果 ────────────────────────────────────────────────────────────────

export interface ListResult {
    path: string
    files: FileInfo[]
}

// ─── 上传节点（用于支持目录递归上传） ──────────────────────────────────────────

export interface UploadNode {
    name: string
    destDir: string
    file?: File
    children?: UploadNode[]
    // 运行时状态（由 ExplorerUpload 管理）
    status?: 'pending' | 'uploading' | 'done' | 'error' | 'cancelled'
    progress?: number
    error?: string
}

// ─── 核心操作接口 ────────────────────────────────────────────────────────────────

export interface ExplorerAdapter {
    /**
     * 列出目录内容
     * @param path 目标目录路径
     */
    list(path: string): Promise<ListResult>

    /**
     * 下载文件，返回 Blob
     * @param path 文件绝对路径
     * @param onProgress 进度回调 0~100
     */
    download(path: string, onProgress?: (percent: number) => void): Promise<Blob>

    /**
     * 上传单个文件
     * @param destDir  目标目录
     * @param file     File 对象
     * @param relativePath  相对子路径（支持目录上传时保持层级）
     * @param onProgress 进度回调 0~100
     */
    upload(destDir: string, file: File, relativePath: string, onProgress?: (percent: number) => void, signal?: AbortSignal): Promise<void>

    /**
     * 删除文件或目录
     * @param path 绝对路径
     * @param recursive 是否递归删除（目录时生效）
     */
    remove(path: string, recursive?: boolean): Promise<void>

    /**
     * 重命名 / 移动
     * @param oldPath 原路径
     * @param newPath 新路径
     */
    rename(oldPath: string, newPath: string): Promise<void>

    /**
     * 新建目录
     * @param path 目录绝对路径
     */
    mkdir(path: string): Promise<void>

    /**
     * 读取文件内容（文本）
     * @param path 文件绝对路径
     */
    readFile(path: string): Promise<string>

    /**
     * 写入文件内容（文本）
     * @param path    文件绝对路径
     * @param content 文件内容
     */
    writeFile(path: string, content: string): Promise<void>

    /**
     * 修改文件/目录权限
     * @param path 绝对路径
     * @param mode 八进制权限字符串，如 "755"
     */
    chmod(path: string, mode: string): Promise<void>

    /**
     * 计算目录大小（可选，不支持时返回 null）
     * @param path 目录绝对路径
     */
    dirSize?(path: string): Promise<number | null>

    /**
     * 新建文件（可选）
     * @param path    文件绝对路径
     * @param content 初始内容，默认空
     */
    createFile?(path: string, content?: string): Promise<void>

    /**
     * 压缩文件或目录为 zip（可选）
     * @param path 目标路径
     */
    zip?(path: string): Promise<void>

    /**
     * 解压 zip 文件（可选）
     * @param path      zip 文件路径
     * @param targetDir 解压目标目录名（相对），留空则解压到当前目录
     */
    unzip?(path: string, targetDir?: string): Promise<void>

    /**
     * 获取文件在线预览/下载 URL（可选）
     * 返回 URL 字符串可直接嵌入 <img>/<video>/<audio>/<object>
     * @param path   文件绝对路径
     * @param token  认证 token
     * @param inline 是否内联展示（非强制下载）
     */
    previewUrl?(path: string, token: string, inline?: boolean): string

    // ─── 权限检查（可选，不实现时默认返回 true） ────────────────────────────────

    /** 是否有权限执行某操作 */
    can?: {
        list?: boolean
        download?: boolean
        upload?: boolean
        remove?: boolean
        rename?: boolean
        mkdir?: boolean
        readFile?: boolean
        writeFile?: boolean
        chmod?: boolean
        createFile?: boolean
        zip?: boolean
        unzip?: boolean
        preview?: boolean
    }
}
