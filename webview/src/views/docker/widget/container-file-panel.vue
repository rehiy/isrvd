<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'

import { ExplorerPanel } from '@/component/explorer'
import type { ExplorerAdapter, FileInfo, ListResult } from '@/component/explorer/types'

function createContainerFileAdapter(containerId: string): ExplorerAdapter {
    const portal = usePortal()
    const perm = (p: string) => portal.hasPerm(p)
    const can: ExplorerAdapter['can'] = {
        list:       perm('GET /api/docker/container/:id/file/ls'),
        download:   perm('GET /api/docker/container/:id/file/download'),
        upload:     perm('POST /api/docker/container/:id/file/upload'),
        remove:     perm('DELETE /api/docker/container/:id/file/rm'),
        rename:     perm('POST /api/docker/container/:id/file/rename'),
        mkdir:      perm('POST /api/docker/container/:id/file/mkdir'),
        readFile:   perm('GET /api/docker/container/:id/file/read'),
        writeFile:  perm('POST /api/docker/container/:id/file/write'),
        chmod:      perm('POST /api/docker/container/:id/file/chmod'),
        createFile: perm('POST /api/docker/container/:id/file/write'),
        zip:        false,
        unzip:      false,
        preview:    perm('GET /api/docker/container/:id/file/download'),
    }
    return {
        can,
        async list(path: string): Promise<ListResult> {
            const res = await api.dockerContainerFileLs(containerId, path)
            const payload = res.payload ?? { path: '', files: [] }
            return {
                path: payload.path,
                files: (payload.files || []).map((f): FileInfo => ({
                    name:      f.name,
                    path:      path.replace(/\/+$/, '') + '/' + f.name,
                    size:      f.size,
                    mode:      f.mode,
                    modeOctal: modeToOctal(f.mode),
                    modTime:   f.modTime,
                    isDir:     f.isDir,
                    isLink:    f.isLink || !!f.linkTarget,
                    linkTarget: f.linkTarget,
                })),
            }
        },
        async download(path: string, onProgress): Promise<Blob> {
            return await api.dockerContainerFileDownload(containerId, path, onProgress) as unknown as Blob
        },
        async upload(destDir, file, relativePath, onProgress, signal): Promise<void> {
            const formData = new FormData()
            formData.append('file', file)
            formData.append('path', destDir)
            formData.append('relativePath', relativePath)
            await api.dockerContainerFileUpload(containerId, destDir, formData, onProgress, { signal })
        },
        async remove(path, recursive = false): Promise<void> {
            await api.dockerContainerFileRemove(containerId, path, recursive)
        },
        async rename(oldPath, newPath): Promise<void> {
            await api.dockerContainerFileRename(containerId, oldPath, newPath)
        },
        async mkdir(path): Promise<void> {
            await api.dockerContainerFileMkdir(containerId, path, { silentError: true })
        },
        async readFile(path): Promise<string> {
            return (await api.dockerContainerFileRead(containerId, path)).payload?.content ?? ''
        },
        async writeFile(path, content): Promise<void> {
            await api.dockerContainerFileWrite(containerId, path, content)
        },
        async chmod(path, mode): Promise<void> {
            await api.dockerContainerFileChmod(containerId, path, mode)
        },
        async createFile(path, content = ''): Promise<void> {
            await api.dockerContainerFileWrite(containerId, path, content)
        },
        previewUrl(path, token): string {
            return api.dockerContainerFileDownloadURL(containerId, path, token)
        },
    }
}

/** 将 rwxr-xr-x 字符串权限转换为八进制字符串，如 "755" */
function modeToOctal(mode: string): string {
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

@Component({ components: { ExplorerPanel } })
class ContainerFilePanel extends Vue {
    @Prop({ required: true }) readonly containerId!: string

    get adapter(): ExplorerAdapter {
        return createContainerFileAdapter(this.containerId)
    }
}

export default toNative(ContainerFilePanel)
</script>

<template>
  <ExplorerPanel :adapter="adapter" :show-batch-ops="true" />
</template>
