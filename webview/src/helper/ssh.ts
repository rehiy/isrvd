import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'

import { wsUrl } from '@/service/axios'
import { joinPath } from '@/helper/utils'

let term: Terminal | null = null
let socket: WebSocket | null = null
let fitAddon: FitAddon | null = null
let resizeHandler: (() => void) | null = null

export function create(el: HTMLElement, token: string, hostId: string): void {
    if (!el) return
    destroy()

    term = new Terminal({ theme: { background: '#0f172a' }, fontSize: 15, cursorBlink: true })
    fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.open(el)
    fitAddon.fit()

    resizeHandler = () => fitAddon?.fit()
    window.addEventListener('resize', resizeHandler)

    socket = new WebSocket(wsUrl(`ssh/to/${encodeURIComponent(hostId)}?token=${token}`))

    term.onData(data => socket?.readyState === WebSocket.OPEN && socket.send(data))
    socket.onopen = () => term && term.write('[连接中...]\r\n')
    socket.onmessage = e => term && term.write(e.data)
    socket.onclose = () => term && term.write('\r\n[连接已关闭]\r\n')
    socket.onerror = (e: Event) => term && term.write(`\r\n[连接错误: ${(e as ErrorEvent).message ?? ''}]\r\n`)

    term.focus()
}

export function destroy(): void {
    resizeHandler && window.removeEventListener('resize', resizeHandler)
    fitAddon = resizeHandler = null
    term?.dispose()
    socket?.close()
    term = socket = null
}

// ==================== SFTP ====================

// 单个上传任务的进度信息
export interface UploadTask {
    name: string
    percent: number   // 0-100
    done: boolean
    error: string
}

/**
 * 递归遍历拖拽的 FileSystemEntry，收集所有文件及其目标目录。
 * 必须在同步代码中先取出所有 entry（DataTransfer 在首次 await 后失效），
 * 再调用此函数异步处理。
 *
 * @param entries   顶层 FileSystemEntry 数组（同步取出）
 * @param baseDir   上传的根目录路径
 * @param onMkdir   创建目录的回调（目录不存在时调用）
 * @returns         待上传的 { file, destDir } 列表
 */
export const collectFileSystemEntries = async (
    entries: FileSystemEntry[],
    baseDir: string,
    onMkdir: (path: string) => Promise<void>,
): Promise<Array<{ file: File; destDir: string }>> => {
    const tasks: Array<{ file: File; destDir: string }> = []

    const collect = async (entry: FileSystemEntry, dir: string): Promise<void> => {
        if (entry.isFile) {
            const file = await new Promise<File>((resolve, reject) =>
                (entry as FileSystemFileEntry).file(resolve, reject)
            )
            tasks.push({ file, destDir: dir })
        } else if (entry.isDirectory) {
            const newDir = joinPath(dir, entry.name)
            await onMkdir(newDir)
            const reader = (entry as FileSystemDirectoryEntry).createReader()
            // readEntries 每次最多返回 100 条，需循环读取直到返回空数组
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
            for (const child of children) {
                await collect(child, newDir)
            }
        }
    }

    for (const entry of entries) {
        await collect(entry, baseDir)
    }
    return tasks
}
