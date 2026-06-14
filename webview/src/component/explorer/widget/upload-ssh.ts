import { joinPath } from '@/helper/utils'

// ==================== SFTP ====================

// 上传文件节点（叶子）
export interface UploadFileNode {
    type: 'file'
    name: string
    size: number
    destDir: string      // 服务器目标目录
    file: File
    percent: number      // 0-100
    done: boolean
    error: string
    cancelled: boolean   // 是否已被用户取消
    controller?: AbortController // 当前上传请求控制器
}

// 上传目录节点（中间节点）
export interface UploadDirNode {
    type: 'dir'
    name: string
    destDir: string      // 服务器目标目录
    children: UploadNode[]
    expanded: boolean
    cancelled: boolean   // 是否已被用户取消
    // 聚合进度（只读计算，由组件自行计算）
}

export type UploadNode = UploadFileNode | UploadDirNode

// 需要过滤掉的系统/隐藏文件名（精确匹配）
const UPLOAD_IGNORE_NAMES = new Set([
    '.DS_Store', '.DS_Store?',
    'Thumbs.db', 'desktop.ini',
    '.Spotlight-V100', '.Trashes', '.fseventsd',
])

// 判断是否应该忽略该条目
export const shouldIgnoreUploadEntry = (name: string): boolean => {
    if (UPLOAD_IGNORE_NAMES.has(name)) return true
    // 以 ._ 开头的 macOS 资源分叉文件
    if (name.startsWith('._')) return true
    return false
}

const isUploadNode = (node: UploadNode | null): node is UploadNode => node !== null

/** 递归收集 FileSystemEntry，构建上传树（不创建服务器目录） */
export const buildUploadTree = async (
    entries: FileSystemEntry[],
    baseDir: string,
): Promise<UploadNode[]> => {
    const collect = async (entry: FileSystemEntry, dir: string): Promise<UploadNode | null> => {
        if (shouldIgnoreUploadEntry(entry.name)) return null
        if (entry.isFile) {
            const file = await new Promise<File>((resolve, reject) =>
                (entry as FileSystemFileEntry).file(resolve, reject)
            )
            return { type: 'file', name: entry.name, size: file.size, destDir: dir, file, percent: 0, done: false, error: '', cancelled: false }
        } else {
            const newDir = joinPath(dir, entry.name)
            const reader = (entry as FileSystemDirectoryEntry).createReader()
            const readAll = (): Promise<FileSystemEntry[]> =>
                new Promise((resolve, reject) => {
                    const all: FileSystemEntry[] = []
                    const readBatch = () => {
                        reader.readEntries((batch) => {
                            if (batch.length === 0) { resolve(all); return }
                            all.push(...batch)
                            readBatch()
                        }, reject)
                    }
                    readBatch()
                })
            const children = await readAll()
            const childNodes = (await Promise.all(children.map(c => collect(c, newDir)))).filter(isUploadNode)
            return { type: 'dir', name: entry.name, destDir: newDir, children: childNodes, expanded: true, cancelled: false }
        }
    }
    const nodes = await Promise.all(entries.map(e => collect(e, baseDir)))
    return nodes.filter(isUploadNode)
}

/** 从上传树中按深度优先顺序收集所有文件节点（用于顺序上传） */
export const flattenUploadTree = (nodes: UploadNode[]): UploadFileNode[] => {
    const result: UploadFileNode[] = []
    const walk = (ns: UploadNode[]) => {
        for (const n of ns) {
            if (n.type === 'file') result.push(n)
            else walk(n.children)
        }
    }
    walk(nodes)
    return result
}
