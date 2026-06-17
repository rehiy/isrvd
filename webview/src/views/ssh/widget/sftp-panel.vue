<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import { usePortal } from '@/stores'
import { ExplorerPanel } from '@/component/explorer'
import type { ExplorerAdapter, FileInfo, ListResult } from '@/component/explorer/types'

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

function createSftpAdapter(hostId: string): ExplorerAdapter {
    const portal = usePortal()
    const perm = (p: string) => portal.hasPerm(p)
    const can: ExplorerAdapter['can'] = {
        list: perm('GET /api/sftp/:id/ls'),
        download: perm('GET /api/sftp/:id/download'),
        upload: perm('POST /api/sftp/:id/upload'),
        remove: perm('DELETE /api/sftp/:id/rm'),
        rename: perm('POST /api/sftp/:id/rename'),
        mkdir: perm('POST /api/sftp/:id/mkdir'),
        readFile: perm('GET /api/sftp/:id/read'),
        writeFile: perm('POST /api/sftp/:id/write'),
        chmod: perm('POST /api/sftp/:id/chmod'),
        preview: perm('GET /api/sftp/:id/download'),
        createFile: perm('POST /api/sftp/:id/write'),
        zip: false,
        unzip: false,
    }
    return {
        can,
        async list(path: string): Promise<ListResult> {
            const res = await api.sftpList(hostId, path)
            const payload = res.payload ?? { path: '', files: [] }
            return {
                path: payload.path,
                files: (payload.files || []).map((f): FileInfo => ({
                    name: f.name,
                    path: path.replace(/\/+$/, '') + '/' + f.name,
                    size: f.size, mode: f.mode,
                    modeOctal: extractOctalMode(f.mode),
                    modTime: f.modTime, isDir: f.isDir,
                    isLink: f.isLink, linkTarget: f.linkTarget,
                })),
            }
        },
        async download(path, onProgress): Promise<Blob> {
            return await api.sftpDownload(hostId, path, onProgress)
        },
        async upload(destDir, file, relativePath, onProgress, signal): Promise<void> {
            const formData = new FormData()
            formData.append('file', file)
            formData.append('path', destDir)
            formData.append('relativePath', relativePath)
            await api.sftpUpload(hostId, destDir, formData, onProgress, { signal })
        },
        async remove(path, recursive = false): Promise<void> { await api.sftpRemove(hostId, path, recursive) },
        async rename(oldPath, newPath): Promise<void> { await api.sftpRename(hostId, { oldPath, newPath }) },
        async mkdir(path): Promise<void> { await api.sftpMkdir(hostId, { path }, { silentError: true }) },
        async readFile(path): Promise<string> { return (await api.sftpRead(hostId, path)).payload?.content ?? '' },
        async writeFile(path, content): Promise<void> { await api.sftpWrite(hostId, { path, content }) },
        async chmod(path, mode): Promise<void> { await api.sftpFileChmod(hostId, { path, mode }) },
        async dirSize(path): Promise<number | null> { return (await api.sftpDirSize(hostId, path)).payload?.size ?? null },
        previewUrl(path, token): string { return api.sftpDownloadURL(hostId, path, token) },
        async createFile(path, content = ''): Promise<void> { await api.sftpWrite(hostId, { path, content }) },
    }
}

@Component({ components: { ExplorerPanel } })
class SftpPanel extends Vue {
    @Prop({ required: true }) readonly hostId!: string

    get adapter(): ExplorerAdapter {
        return createSftpAdapter(this.hostId)
    }
}

export default toNative(SftpPanel)
</script>

<template>
  <ExplorerPanel :adapter="adapter" :show-card="false" :show-batch-ops="true" />
</template>
