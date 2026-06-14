/**
 * 本地文件管理适配器（基于 /api/filer/* 接口）
 */
import api from '@/service/api'
import { usePortal } from '@/stores'
import type { ExplorerAdapter, FileInfo, ListResult } from '@/component/explorer/types'

export function createFilerAdapter(): ExplorerAdapter {
    const portal = usePortal()

    const can: ExplorerAdapter['can'] = {
        list: portal.hasPerm('GET /api/filer/files'),
        download: portal.hasPerm('GET /api/filer/download'),
        upload: portal.hasPerm('POST /api/filer/upload'),
        remove: portal.hasPerm('DELETE /api/filer/file'),
        rename: portal.hasPerm('POST /api/filer/rename'),
        mkdir: portal.hasPerm('POST /api/filer/dir'),
        readFile: portal.hasPerm('GET /api/filer/file'),
        writeFile: portal.hasPerm('PUT /api/filer/file'),
        chmod: portal.hasPerm('PUT /api/filer/chmod'),
        createFile: portal.hasPerm('POST /api/filer/file'),
        zip: portal.hasPerm('POST /api/filer/zip'),
        unzip: portal.hasPerm('POST /api/filer/unzip'),
        preview: portal.hasPerm('GET /api/filer/download'), // 复用 download 权限
    }

    return {
        can,

        async list(path: string): Promise<ListResult> {
            const res = await api.filerList(path)
            const payload = res.payload!
            return {
                path: payload.path,
                files: (payload.files || []).map((f): FileInfo => ({
                    name: f.name,
                    path: f.path,
                    size: f.size,
                    mode: f.mode,
                    modeOctal: f.modeO,
                    modTime: f.modTime,
                    isDir: f.isDir,
                })),
            }
        },

        async download(path: string): Promise<Blob> {
            const res = await api.filerDownload(path)
            return res as unknown as Blob
        },

        async upload(destDir: string, file: File, _relativePath: string, onProgress?: (p: number) => void, signal?: AbortSignal): Promise<void> {
            const formData = new FormData()
            formData.append('file', file)
            formData.append('path', destDir)
            await api.filerUpload(formData, {
                signal,
                onUploadProgress: (e) => {
                    if (onProgress) onProgress(e.total ? Math.round((e.loaded / e.total) * 100) : 0)
                },
            })
        },

        async remove(path: string): Promise<void> {
            await api.filerDelete(path)
        },

        async rename(oldPath: string, newPath: string): Promise<void> {
            await api.filerRename(oldPath, newPath)
        },

        async mkdir(path: string): Promise<void> {
            await api.filerMkdir(path, { silentError: true })
        },

        async readFile(path: string): Promise<string> {
            const res = await api.filerRead(path)
            return res.payload?.content ?? ''
        },

        async writeFile(path: string, content: string): Promise<void> {
            await api.filerModify(path, content)
        },

        async chmod(path: string, mode: string): Promise<void> {
            await api.filerChmod(path, mode)
        },

        async dirSize(path: string): Promise<number | null> {
            const res = await api.filerDirSize(path)
            return res.payload?.size ?? null
        },

        async createFile(path: string, content = ''): Promise<void> {
            await api.filerCreate(path, content)
        },

        async zip(path: string): Promise<void> {
            await api.filerZip(path)
        },

        async unzip(path: string, targetDir?: string): Promise<void> {
            await api.filerUnzip(path, targetDir)
        },

        previewUrl(path: string, token: string, inline = true): string {
            return api.filerDownloadURL(path, token, inline)
        },
    }
}
