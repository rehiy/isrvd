<script lang="ts">
import { Component, Prop, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { SFTPFileInfo, SFTPListResult } from '@/service/types'

import { buildUploadTree } from '@/helper/ssh'
import { formatFileSize, formatUnixTime, getFileIcon, joinPath, downloadBlob, isEditableFile, isPreviewableFile } from '@/helper/utils'

import SftpChmodModal from './sftp-chmod-modal.vue'
import SftpModifyModal from './sftp-modify-modal.vue'
import SftpRenameModal from './sftp-rename-modal.vue'
import UploadWidget from './sftp-upload.vue'

@Component({ components: { UploadWidget, SftpChmodModal, SftpRenameModal, SftpModifyModal } })
class SftpPanel extends Vue {
    @Prop({ required: true }) readonly hostId!: string
    @Prop({ default: 280 }) readonly height!: number | string

    portal = usePortal()

    @Ref readonly uploadWidgetRef!: InstanceType<typeof UploadWidget>
    @Ref readonly chmodModalRef!: InstanceType<typeof SftpChmodModal>
    @Ref readonly renameModalRef!: InstanceType<typeof SftpRenameModal>
    @Ref readonly modifyModalRef!: InstanceType<typeof SftpModifyModal>

    // ─── 状态 ───
    sftpPath = '/'
    sftpFiles: SFTPFileInfo[] = []
    sftpLoading = false
    sftpError = ''
    mkdirMode = false
    mkdirName = ''
    pathEditMode = false
    pathEditValue = ''
    // 记录已计算大小的文件夹路径
    calculatedDirs = new Set<string>()
    // 记录正在计算大小的文件夹路径
    calculatingDirs = new Set<string>()

    // ─── 拖拽状态 ───
    dragOver = false
    dragCounter = 0

    // ─── 下载状态 ───
    downloadingFile = ''

    // ─── 工具方法 ───
    isEditableFile = isEditableFile
    isPreviewableFile = isPreviewableFile

    // ─── 计算属性 ───
    get sftpPathParts() {
        const parts = this.sftpPath.replace(/\/+$/, '').split('/').filter(Boolean)
        const result = [{ label: '/', path: '/' }]
        let cur = ''
        for (const p of parts) {
            cur += '/' + p
            result.push({ label: p, path: cur })
        }
        return result
    }

    // ─── 目录加载 ───
    async sftpLoad(dirPath?: string) {
        if (dirPath !== undefined) this.sftpPath = dirPath
        this.sftpLoading = true
        this.sftpError = ''
        try {
            const res = await api.sftpList(this.hostId, this.sftpPath)
            const result = res.payload as SFTPListResult
            this.sftpPath = result.path
            const files = result.files || []
            this.sftpFiles = [
                ...files.filter((f: SFTPFileInfo) => f.isDir).sort((a: SFTPFileInfo, b: SFTPFileInfo) => a.name.localeCompare(b.name)),
                ...files.filter((f: SFTPFileInfo) => !f.isDir).sort((a: SFTPFileInfo, b: SFTPFileInfo) => a.name.localeCompare(b.name)),
            ]
        } catch (e: unknown) {
            this.sftpError = (e instanceof Error ? e.message : '') || '加载失败'
        } finally {
            this.sftpLoading = false
        }
    }

    // ─── 导航 ───
    sftpEnter(file: SFTPFileInfo) {
        if (!file.isDir) return
        this.sftpLoad(joinPath(this.sftpPath, file.name))
    }

    sftpGoUp() {
        if (this.sftpPath === '/') return
        const parent = this.sftpPath.replace(/\/+$/, '').split('/').slice(0, -1).join('/') || '/'
        this.sftpLoad(parent)
    }

    // ─── 文件夹大小计算 ───
    sftpCalcPath(file: SFTPFileInfo): string {
        return joinPath(this.sftpPath, file.name)
    }

    async sftpCalcDirSize(file: SFTPFileInfo) {
        const filePath = this.sftpCalcPath(file)
        if (!file.isDir || this.calculatingDirs.has(filePath)) return
        this.calculatingDirs.add(filePath)
        try {
            const result = await api.sftpDirSize(this.hostId, filePath)
            file.size = result.payload?.size || 0
            this.calculatedDirs.add(filePath)
        } catch (error: unknown) {
            console.error('计算目录大小失败:', error)
        } finally {
            this.calculatingDirs.delete(filePath)
        }
    }

    // ─── 路径编辑 ───
    startPathEdit() {
        this.pathEditMode = true
        this.pathEditValue = this.sftpPath
    }

    cancelPathEdit() {
        this.pathEditMode = false
        this.pathEditValue = ''
    }

    confirmPathEdit() {
        const p = this.pathEditValue.trim()
        if (!p) return
        this.pathEditMode = false
        this.pathEditValue = ''
        this.sftpLoad(p)
    }

    // ─── 下载 ───
    async sftpDownload(file: SFTPFileInfo) {
        const filePath = joinPath(this.sftpPath, file.name)
        this.downloadingFile = file.name
        try {
            const blob = await api.sftpDownload(this.hostId, filePath)
            downloadBlob(blob, file.name)
        } catch {
            // 错误已由 axios 拦截器统一弹出通知
        } finally {
            this.downloadingFile = ''
        }
    }

    // ─── 删除 ───
    async sftpDelete(file: SFTPFileInfo) {
        const filePath = joinPath(this.sftpPath, file.name)
        const isDir = file.isDir
        this.portal.showConfirm({
            title: '删除确认',
            message: isDir
                ? `确定要删除目录 <strong class="text-slate-900">${file.name}</strong> 及其所有内容吗？`
                : `确定要删除 <strong class="text-slate-900">${file.name}</strong> 吗？`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                try {
                    await api.sftpRemove(this.hostId, filePath, isDir)
                    this.portal.showNotification('success', '删除成功')
                    this.sftpLoad()
                } catch (e: unknown) {
                    this.portal.showNotification('error', (e instanceof Error ? e.message : '') || '删除失败')
                }
            }
        })
    }

    // ─── 新建目录 ───
    startMkdir() {
        this.mkdirMode = true
        this.mkdirName = ''
    }

    cancelMkdir() {
        this.mkdirMode = false
        this.mkdirName = ''
    }

    async confirmMkdir() {
        if (!this.mkdirName.trim()) return
        const dirPath = joinPath(this.sftpPath, this.mkdirName.trim())
        try {
            await api.sftpMkdir(this.hostId, { path: dirPath })
            this.portal.showNotification('success', '创建成功')
            this.cancelMkdir()
            this.sftpLoad()
        } catch (e: unknown) {
            this.portal.showNotification('error', (e instanceof Error ? e.message : '') || '创建失败')
        }
    }

    // ─── 上传入口（input 选择）───
    triggerUpload() {
        this.dragOver = true
        this.dragCounter = 1
        
        // 3秒后自动关闭拖拽提示
        setTimeout(() => {
            if (this.dragOver) {
                this.dragOver = false
                this.dragCounter = 0
            }
        }, 3000)
    }

    // ─── 拖拽上传 ───
    onDragEnter(e: DragEvent) {
        e.preventDefault()
        this.dragCounter++
        this.dragOver = true
    }

    onDragOver(e: DragEvent) {
        e.preventDefault()
    }

    onDragLeave(e: DragEvent) {
        e.preventDefault()
        this.dragCounter--
        if (this.dragCounter <= 0) {
            this.dragCounter = 0
            this.dragOver = false
        }
    }

    async onDrop(e: DragEvent) {
        e.preventDefault()
        this.dragOver = false
        this.dragCounter = 0
        const items = e.dataTransfer?.items
        if (!items || items.length === 0) return

        const entries: FileSystemEntry[] = []
        for (let i = 0; i < items.length; i++) {
            const entry = items[i].webkitGetAsEntry()
            if (entry) entries.push(entry)
        }
        if (entries.length === 0) return

        const nodes = await buildUploadTree(entries, this.sftpPath)
        if (nodes.length === 0) return
        this.uploadWidgetRef?.upload(nodes)
    }

    // ─── 格式化（复用 utils）───
    formatSize = formatFileSize
    formatTime = formatUnixTime
    getFileIcon = getFileIcon

    // ─── 权限修改 ───
    startChmod(file: SFTPFileInfo) {
        this.chmodModalRef?.show(this.hostId, file, this.sftpPath)
    }

    // ─── 重命名 ───
    startRename(file: SFTPFileInfo) {
        this.renameModalRef?.show(this.hostId, file, this.sftpPath)
    }

    // ─── 编辑文件 ───
    startModify(file: SFTPFileInfo) {
        this.modifyModalRef?.show(this.hostId, file, this.sftpPath)
    }

    // ─── 生命周期 ───
    mounted() {
        this.sftpLoad()
    }
}

export default toNative(SftpPanel)
</script>

<template>
  <div class="border-t border-slate-200 flex flex-col h-full">
    <!-- 工具栏（不收缩，不在滚动范围内） -->
    <div class="flex items-center gap-2 px-3 py-2 bg-slate-50 border-b border-slate-200 flex-shrink-0">
      <input
        v-if="pathEditMode"
        v-model="pathEditValue"
        class="input text-xs py-0.5 font-mono"
        autofocus
        @keyup.enter="confirmPathEdit()"
        @keyup.esc="cancelPathEdit()"
      />
      <div v-else class="flex items-center gap-1 text-xs text-slate-600 min-w-0 flex-1 overflow-x-auto">
        <template v-for="(part, i) in sftpPathParts" :key="part.path">
          <span v-if="i > 0" class="text-slate-300 flex-shrink-0">/</span>
          <button
            class="flex-shrink-0 max-w-32 truncate transition-colors hover:text-teal-600"
            :class="i === sftpPathParts.length - 1 ? 'text-slate-800 font-medium' : 'text-slate-500'"
            @click="sftpLoad(part.path)"
          >
            <template v-if="i === 0"><i class="fas fa-home text-xs"></i></template>
            <span v-else>{{ part.label }}</span>
          </button>
        </template>
      </div>
      <div class="flex items-center gap-1 flex-shrink-0">
        <button
          class="btn-icon flex-shrink-0"
          :class="pathEditMode ? 'btn-icon-teal' : 'btn-icon-slate'"
          :title="pathEditMode ? '确认跳转' : '输入路径跳转'"
          @click="pathEditMode ? confirmPathEdit() : startPathEdit()"
        >
          <i class="fas text-xs" :class="pathEditMode ? 'fa-arrow-right' : 'fa-location-arrow'"></i>
        </button>
        <button class="btn-icon btn-icon-slate" title="返回上级" :disabled="sftpPath === '/'" @click="sftpGoUp()">
          <i class="fas fa-level-up-alt text-xs"></i>
        </button>
        <button class="btn-icon btn-icon-slate" title="刷新" :disabled="sftpLoading" @click="sftpLoad()">
          <i class="fas fa-rotate text-xs" :class="{ 'animate-spin': sftpLoading }"></i>
        </button>
        <button v-if="portal.hasPerm('POST /api/ssh/sftp/:id/mkdir')" class="btn-icon btn-icon-slate" title="新建目录" @click="startMkdir()">
          <i class="fas fa-folder-plus text-xs"></i>
        </button>
        <button v-if="portal.hasPerm('POST /api/ssh/sftp/:id/write')" class="btn-icon btn-icon-teal" title="上传文件" @click="triggerUpload()">
          <i class="fas fa-upload text-xs"></i>
        </button>
      </div>
    </div>

    <!-- 上传进度 Widget -->
    <UploadWidget
      ref="uploadWidgetRef"
      :host-id="hostId"
      :sftp-path="sftpPath"
      @done="sftpLoad()"
    />

    <!-- 文件列表（拖拽区域） -->
    <div
      class="flex-1 min-h-20 overflow-y-auto relative"
      @dragenter="onDragEnter"
      @dragover="onDragOver"
      @dragleave="onDragLeave"
      @drop="onDrop"
    >
      <!-- 拖拽遮罩 -->
      <div
        v-if="dragOver"
        class="absolute inset-0 z-10 flex flex-col items-center justify-center gap-2 bg-teal-50/90 border-2 border-dashed border-teal-400 rounded pointer-events-none"
      >
        <i class="fas fa-cloud-arrow-up text-2xl text-teal-500"></i>
        <span class="text-sm text-teal-600 font-medium">松手上传到当前目录</span>
        <span class="text-xs text-teal-400">支持文件夹</span>
      </div>

      <!-- 加载中 -->
      <div v-if="sftpLoading" class="flex items-center justify-center h-full text-slate-400 text-sm gap-2">
        <div class="w-4 h-4 spinner"></div>加载中...
      </div>
      <!-- 错误 -->
      <div v-else-if="sftpError" class="flex items-center justify-center h-full text-red-400 text-sm gap-2">
        <i class="fas fa-circle-exclamation"></i>{{ sftpError }}
      </div>
      <!-- 空目录 -->
      <div v-else-if="sftpFiles.length === 0 && !mkdirMode" class="flex items-center justify-center h-full text-slate-400 text-sm gap-2">
        <i class="fas fa-folder-open"></i>空目录
      </div>
      <!-- 文件表格 -->
      <table v-else class="w-full text-xs">
        <tbody>
          <tr v-if="mkdirMode" class="border-b border-teal-100 bg-teal-50">
            <td class="px-3 py-1.5" colspan="5">
              <div class="flex items-center gap-2">
                <div class="w-5 h-5 rounded bg-amber-400 flex items-center justify-center flex-shrink-0">
                  <i class="fas fa-folder text-white text-xs"></i>
                </div>
                <input
                  v-model="mkdirName"
                  class="input text-xs py-0.5"
                  placeholder="请输入目录名称"
                  autofocus
                  @keyup.enter="confirmMkdir()"
                  @keyup.esc="cancelMkdir()"
                />
                <button class="btn-icon btn-icon-teal !w-6 !h-6 flex-shrink-0" title="确认" @click="confirmMkdir()">
                  <i class="fas fa-check text-xs"></i>
                </button>
                <button class="btn-icon btn-icon-slate !w-6 !h-6 flex-shrink-0" title="取消" @click="cancelMkdir()">
                  <i class="fas fa-xmark text-xs"></i>
                </button>
              </div>
            </td>
          </tr>
          <tr
            v-for="file in sftpFiles"
            :key="file.name"
            class="border-b border-slate-100 hover:bg-slate-50 transition-colors group"
          >
            <td class="px-3 py-2 w-full">
              <div class="flex items-center gap-2 min-w-0">
                <div class="relative w-5 h-5 flex-shrink-0">
                  <div :class="['w-5 h-5 rounded flex items-center justify-center', file.isDir ? 'bg-amber-400' : 'bg-slate-400']">
                    <i :class="getFileIcon(file)" class="text-white text-xs"></i>
                  </div>
                  <i v-if="file.isLink" class="fas fa-link absolute -bottom-0.5 -right-0.5 text-white text-[8px]" style="text-shadow: 0 0 2px rgba(0,0,0,0.6)"></i>
                </div>
                <span
                  class="truncate"
                  :class="file.isDir ? 'text-slate-700 font-medium cursor-pointer hover:text-teal-600' : 'text-slate-600'"
                  @click="file.isDir && sftpEnter(file)"
                >{{ file.name }}<span v-if="file.isLink && file.linkTarget" class="ml-1 text-slate-400 text-xs font-normal">{{ file.linkTarget }}</span></span>
              </div>
            </td>
            <td class="px-2 py-2 text-slate-400 whitespace-nowrap hidden sm:table-cell">
              <span v-if="!file.isDir">{{ formatSize(file.size) }}</span>
              <button 
                v-else 
                type="button" 
                class="group text-slate-400 hover:text-teal-600 transition-colors w-20 text-left"
                @click="sftpCalcDirSize(file)"
              >
                <template v-if="calculatedDirs.has(sftpCalcPath(file))">{{ formatSize(file.size) }}</template>
                <template v-else-if="calculatingDirs.has(sftpCalcPath(file))">计算中...</template>
                <template v-else>
                  <span class="group-hover:hidden">--</span>
                  <span class="group-hover:inline hidden">计算大小</span>
                </template>
              </button>
            </td>
            <td class="px-2 py-2 text-slate-400 whitespace-nowrap hidden md:table-cell">
              {{ formatTime(file.modTime) }}
            </td>
            <td class="px-2 py-2 text-slate-400 whitespace-nowrap hidden lg:table-cell font-mono text-xs">
              {{ file.mode }}
            </td>
            <td class="px-2 py-2 whitespace-nowrap">
              <div class="flex items-center gap-1 justify-end">
                <template v-if="file.isDir">
                  <button class="btn-icon btn-icon-slate !w-6 !h-6" title="进入目录" @click="sftpEnter(file)">
                    <i class="fas fa-folder-open text-xs"></i>
                  </button>
                </template>
                <template v-else>
                  <button
                    v-if="portal.hasPerm('GET /api/ssh/sftp/:id/download')"
                    class="btn-icon !w-6 !h-6"
                    :class="downloadingFile === file.name ? 'btn-icon-teal' : 'btn-icon-slate'"
                    title="下载"
                    :disabled="!!downloadingFile"
                    @click="sftpDownload(file)"
                  >
                    <i class="fas text-xs" :class="downloadingFile === file.name ? 'fa-spinner animate-spin' : 'fa-download'"></i>
                  </button>
                </template>
                <button v-if="portal.hasPerm('POST /api/ssh/sftp/:id/rename')" class="btn-icon btn-icon-blue !w-6 !h-6" title="重命名" @click="startRename(file)">
                  <i class="fas fa-spell-check text-xs"></i>
                </button>
                <button v-if="!file.isDir && isEditableFile(file.name) && portal.hasPerm('GET /api/ssh/sftp/:id/read') && portal.hasPerm('POST /api/ssh/sftp/:id/write')" class="btn-icon btn-icon-blue !w-6 !h-6" title="编辑" @click="startModify(file)">
                  <i class="fas fa-file-pen text-xs"></i>
                </button>
                <button v-if="portal.hasPerm('POST /api/ssh/sftp/:id/chmod')" class="btn-icon btn-icon-slate !w-6 !h-6" title="修改权限" @click="startChmod(file)">
                  <i class="fas fa-unlock text-xs"></i>
                </button>
                <button v-if="portal.hasPerm('DELETE /api/ssh/sftp/:id/rm')" class="btn-icon btn-icon-red !w-6 !h-6" title="删除" @click="sftpDelete(file)">
                  <i class="fas fa-trash text-xs"></i>
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 模态组件 -->
    <SftpChmodModal ref="chmodModalRef" @success="sftpLoad()" />
    <SftpRenameModal ref="renameModalRef" @success="sftpLoad()" />
    <SftpModifyModal ref="modifyModalRef" @success="sftpLoad()" />
  </div>
</template>
