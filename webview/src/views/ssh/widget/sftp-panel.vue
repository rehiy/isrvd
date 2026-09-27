<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'

import { ExplorerPanel } from '@/component/explorer'
import { remoteFileInfo } from '@/component/explorer/helper'
import type { ExplorerAdapter, ListResult } from '@/component/explorer/types'

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
                files: (payload.files || []).map(file => remoteFileInfo(path, file)),
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
        async mkdir(path): Promise<void> { await api.sftpMkdir(hostId, { path }) },
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
  <ExplorerPanel :adapter="adapter" :show-batch-ops="true" />
</template>
