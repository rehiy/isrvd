<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'

import { ExplorerPanel } from '@/component/explorer'
import type { ExplorerAdapter, FileInfo, ListResult } from '@/component/explorer/types'

function createFilerAdapter(): ExplorerAdapter {
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
        preview: portal.hasPerm('GET /api/filer/download'),
    }
    return {
        can,
        async list(path: string): Promise<ListResult> {
            const res = await api.filerList(path)
            const payload = res.payload ?? { path: '', files: [] }
            return {
                path: payload.path,
                files: (payload.files || []).map((f): FileInfo => ({
                    name: f.name, path: f.path, size: f.size,
                    mode: f.mode, modeOctal: f.modeO, modTime: f.modTime, isDir: f.isDir,
                    isLink: f.isLink, linkTarget: f.linkTarget,
                })),
            }
        },
        async download(path: string): Promise<Blob> {
            return await api.filerDownload(path) as unknown as Blob
        },
        async upload(destDir, file, _rel, onProgress, signal): Promise<void> {
            const formData = new FormData()
            formData.append('file', file)
            formData.append('path', destDir)
            await api.filerUpload(formData, {
                signal,
                onUploadProgress: (e) => onProgress?.(e.total ? Math.round((e.loaded / e.total) * 100) : 0),
            })
        },
        async remove(path): Promise<void> { await api.filerDelete(path) },
        async rename(oldPath, newPath): Promise<void> { await api.filerRename(oldPath, newPath) },
        async mkdir(path): Promise<void> { await api.filerMkdir(path, { silentError: true }) },
        async readFile(path): Promise<string> { return (await api.filerRead(path)).payload?.content ?? '' },
        async writeFile(path, content): Promise<void> { await api.filerModify(path, content) },
        async chmod(path, mode): Promise<void> { await api.filerChmod(path, mode) },
        async dirSize(path): Promise<number | null> { return (await api.filerDirSize(path)).payload?.size ?? null },
        async createFile(path, content = ''): Promise<void> { await api.filerCreate(path, content) },
        async zip(path): Promise<void> { await api.filerZip(path) },
        async unzip(path, targetDir): Promise<void> { await api.filerUnzip(path, targetDir) },
        previewUrl(path, token, inline = true): string { return api.filerDownloadURL(path, token, inline) },
    }
}

@Component({ components: { ExplorerPanel } })
class FileExplorer extends Vue {
    adapter = createFilerAdapter()
}

export default toNative(FileExplorer)
</script>

<template>
  <ExplorerPanel :adapter="adapter" :sticky-toolbar="true" :show-search="true" :show-batch-ops="true" :show-mobile-view="true" />
</template>
