/**
 * SSH SFTP 文件管理适配器（基于 /api/ssh/sftp/:id/* 接口）
 */
import api from '@/service/api'
import { usePortal } from '@/stores'
import type { ExplorerAdapter, FileInfo, ListResult } from '@/component/explorer/types'

/**
 * 从 mode 字符串（如 "-rw-r--r--"）提取八进制权限（如 "644"）
 */
function extractOctalMode(mode: string): string {
    if (mode.length < 10) return ''
    const parseRwx = (s: string): number => {
        let v = 0
        if (s[0] === 'r') v += 4
        if (s[1] === 'w') v += 2
        if (s[2] === 'x' || s[2] === 's' || s[2] === 't') v += 1
        return v
    }
    return `${parseRwx(mode.slice(1, 4))}${parseRwx(mode.slice(4, 7))}${parseRwx(mode.slice(7, 10))}`
}

export function createSftpAdapter(hostId: string): ExplorerAdapter {
    const portal = usePortal()

    const perm = (p: string) => portal.hasPerm(p.replace(':id', hostId))

    const can: ExplorerAdapter['can'] = {
        list: perm('GET /api/ssh/sftp/:id/ls'),
        download: perm('GET /api/ssh/sftp/:id/download'),
        upload: perm('POST /api/ssh/sftp/:id/write'),
        remove: perm('DELETE /api/ssh/sftp/:id/rm'),
        rename: perm('POST /api/ssh/sftp/:id/rename'),
        mkdir: perm('POST /api/ssh/sftp/:id/mkdir'),
        readFile: perm('GET /api/ssh/sftp/:id/read'),
        writeFile: perm('POST /api/ssh/sftp/:id/write'),
        chmod: perm('POST /api/ssh/sftp/:id/chmod'),
        preview: perm('GET /api/ssh/sftp/:id/download'), // 复用 download 权限
        createFile: perm('POST /api/ssh/sftp/:id/write'), // 通过 writeFile 写入空内容实现
        zip: false,   // SFTP 不支持服务端压缩
        unzip: false, // SFTP 不支持服务端解压
    }

    return {
        can,

        async list(path: string): Promise<ListResult> {
            const res = await api.sftpList(hostId, path)
            const payload = res.payload!
            return {
                path: payload.path,
                files: (payload.files || []).map((f): FileInfo => ({
                    name: f.name,
                    path: path.replace(/\/+$/, '') + '/' + f.name,
                    size: f.size,
                    mode: f.mode,
                    modeOctal: extractOctalMode(f.mode),
                    modTime: f.modTime,
                    isDir: f.isDir,
                    isLink: f.isLink,
                    linkTarget: f.linkTarget,
                })),
            }
        },

        async download(path: string, onProgress?: (p: number) => void): Promise<Blob> {
            return await api.sftpDownload(hostId, path, onProgress)
        },

        async upload(destDir: string, file: File, relativePath: string, onProgress?: (p: number) => void, signal?: AbortSignal): Promise<void> {
            const formData = new FormData()
            formData.append('file', file)
            formData.append('path', destDir)
            formData.append('relativePath', relativePath)
            await api.sftpUpload(hostId, destDir, formData, onProgress, { signal })
        },

        async remove(path: string, recursive = false): Promise<void> {
            await api.sftpRemove(hostId, path, recursive)
        },

        async rename(oldPath: string, newPath: string): Promise<void> {
            await api.sftpRename(hostId, { oldPath, newPath })
        },

        async mkdir(path: string): Promise<void> {
            await api.sftpMkdir(hostId, { path }, { silentError: true })
        },

        async readFile(path: string): Promise<string> {
            const res = await api.sftpRead(hostId, path)
            return res.payload?.content ?? ''
        },

        async writeFile(path: string, content: string): Promise<void> {
            await api.sftpWrite(hostId, { path, content })
        },

        async chmod(path: string, mode: string): Promise<void> {
            await api.sftpFileChmod(hostId, { path, mode })
        },

        async dirSize(path: string): Promise<number | null> {
            const res = await api.sftpDirSize(hostId, path)
            return res.payload?.size ?? null
        },

        previewUrl(path: string, token: string): string {
            return api.sftpDownloadURL(hostId, path, token)
        },

        async createFile(path: string, content = ''): Promise<void> {
            await api.sftpWrite(hostId, { path, content })
        },
    }
}
