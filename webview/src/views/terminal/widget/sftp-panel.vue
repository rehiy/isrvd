<script lang="ts">
import { Component, Prop, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { SFTPFileInfo, SFTPListResult } from '@/service/types'

import { formatFileSize, formatUnixTime, getFileIcon, joinPath, downloadBlob } from '@/helper/utils'
import { type UploadTask, collectFileSystemEntries } from '@/helper/ssh'

@Component
class SftpPanel extends Vue {
    @Prop({ required: true }) readonly hostId!: string
    @Prop({ default: 280 }) readonly height!: number

    portal = usePortal()

    @Ref readonly uploadInputRef!: HTMLInputElement

    // ─── 状态 ───
    sftpPath = '/'
    sftpFiles: SFTPFileInfo[] = []
    sftpLoading = false
    sftpError = ''
    renamingFile: SFTPFileInfo | null = null
    renameNewName = ''
    mkdirMode = false
    mkdirName = ''
    pathEditMode = false
    pathEditValue = ''

    // ─── 上传进度 ───
    uploadTasks: UploadTask[] = []
    get sftpUploading() { return this.uploadTasks.some(t => !t.done && !t.error) }

    // ─── 拖拽状态 ───
    dragOver = false
    dragCounter = 0   // 用计数器避免子元素 dragenter/dragleave 抖动

    // ─── 下载状态 ───
    downloadingFile = ''

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

    // ─── 下载（带进度提示）───
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

    // ─── 重命名 ───
    startRename(file: SFTPFileInfo) {
        this.renamingFile = file
        this.renameNewName = file.name
        this.mkdirMode = false
    }

    cancelRename() {
        this.renamingFile = null
        this.renameNewName = ''
    }

    async confirmRename() {
        if (!this.renamingFile || !this.renameNewName.trim()) return
        const oldPath = joinPath(this.sftpPath, this.renamingFile.name)
        const newPath = joinPath(this.sftpPath, this.renameNewName.trim())
        try {
            await api.sftpRename(this.hostId, { oldPath, newPath })
            this.portal.showNotification('success', '重命名成功')
            this.cancelRename()
            this.sftpLoad()
        } catch (e: unknown) {
            this.portal.showNotification('error', (e instanceof Error ? e.message : '') || '重命名失败')
        }
    }

    // ─── 新建目录 ───
    startMkdir() {
        this.mkdirMode = true
        this.mkdirName = ''
        this.renamingFile = null
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

    // ─── 上传核心：带进度的单文件上传 ───
    async uploadOneFile(file: File, destDir: string): Promise<void> {
        const task: UploadTask = { name: file.name, percent: 0, done: false, error: '' }
        this.uploadTasks.push(task)
        const form = new FormData()
        form.append('file', file)
        try {
            await api.sftpUpload(this.hostId, destDir, form, (percent) => {
                task.percent = percent
            })
            task.percent = 100
            task.done = true
        } catch (e: unknown) {
            task.error = (e instanceof Error ? e.message : '') || '上传失败'
            task.done = true
            throw e
        }
    }

    // ─── 上传入口（input 选择）───
    triggerUpload() {
        this.uploadInputRef?.click()
    }

    async handleUpload(e: Event) {
        const input = e.target as HTMLInputElement
        const files = input.files
        if (!files || files.length === 0) return
        this.uploadTasks = []
        let failCount = 0
        for (const file of Array.from(files)) {
            try {
                await this.uploadOneFile(file, this.sftpPath)
            } catch {
                failCount++
            }
        }
        input.value = ''
        const total = files.length
        if (failCount === 0) {
            this.portal.showNotification('success', `上传成功（${total} 个文件）`)
        } else {
            this.portal.showNotification('error', `${total - failCount} 个成功，${failCount} 个失败`)
        }
        // 延迟清除进度条，让用户看到 100%
        setTimeout(() => { this.uploadTasks = [] }, 1500)
        this.sftpLoad()
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

        // 必须在同步代码中一次性取出所有 entry，
        // DataTransfer 在首次 await 后会被浏览器清空
        const entries: FileSystemEntry[] = []
        for (let i = 0; i < items.length; i++) {
            const entry = items[i].webkitGetAsEntry()
            if (entry) entries.push(entry)
        }

        const tasks = await collectFileSystemEntries(
            entries,
            this.sftpPath,
            async (path) => { try { await api.sftpMkdir(this.hostId, { path }) } catch { /* 已存在则忽略 */ } },
        )

        if (tasks.length === 0) return

        this.uploadTasks = []
        let failCount = 0
        for (const { file, destDir } of tasks) {
            try {
                await this.uploadOneFile(file, destDir)
            } catch {
                failCount++
            }
        }
        const total = tasks.length
        if (failCount === 0) {
            this.portal.showNotification('success', `上传成功（${total} 个文件）`)
        } else {
            this.portal.showNotification('error', `${total - failCount} 个成功，${failCount} 个失败`)
        }
        setTimeout(() => { this.uploadTasks = [] }, 1500)
        this.sftpLoad()
    }

    // ─── 格式化（复用 utils）───
    formatSize = formatFileSize
    formatTime = formatUnixTime
    getFileIcon = getFileIcon

    // ─── 生命周期 ───
    mounted() {
        this.sftpLoad()
    }
}

export default toNative(SftpPanel)
</script>

<template>
  <div class="border-t border-slate-200 flex flex-col" :style="{ height: height + 'px', minHeight: '120px' }">
    <!-- 工具栏 -->
    <div class="flex items-center gap-2 px-3 py-2 bg-slate-50 border-b border-slate-200 flex-shrink-0">
      <!-- 路径输入框（编辑模式） -->
      <input
        v-if="pathEditMode"
        v-model="pathEditValue"
        class="input text-xs py-0.5 font-mono"
        autofocus
        @keyup.enter="confirmPathEdit()"
        @keyup.esc="cancelPathEdit()"
      />
      <!-- 面包屑（普通模式） -->
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
      <!-- 操作按钮 -->
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
        <button class="btn-icon btn-icon-slate" title="新建目录" @click="startMkdir()">
          <i class="fas fa-folder-plus text-xs"></i>
        </button>
        <button class="btn-icon btn-icon-teal" title="上传文件" :disabled="sftpUploading" @click="triggerUpload()">
          <i class="fas fa-upload text-xs" :class="{ 'animate-pulse': sftpUploading }"></i>
        </button>
        <input ref="uploadInputRef" type="file" multiple class="hidden" @change="handleUpload" />
      </div>
    </div>

    <!-- 上传进度条区域 -->
    <div v-if="uploadTasks.length > 0" class="flex-shrink-0 border-b border-slate-200 bg-slate-50 px-3 py-2 space-y-1.5 max-h-28 overflow-y-auto">
      <div v-for="task in uploadTasks" :key="task.name" class="flex items-center gap-2 text-xs">
        <i
          class="fas w-3 flex-shrink-0"
          :class="task.error ? 'fa-circle-exclamation text-red-400' : task.done ? 'fa-circle-check text-teal-500' : 'fa-arrow-up-from-bracket text-slate-400'"
        ></i>
        <span class="truncate flex-1 text-slate-600 min-w-0">{{ task.name }}</span>
        <span v-if="task.error" class="text-red-400 flex-shrink-0">失败</span>
        <template v-else>
          <div class="w-20 h-1.5 bg-slate-200 rounded-full flex-shrink-0 overflow-hidden">
            <div
              class="h-full rounded-full transition-all duration-200"
              :class="task.done ? 'bg-teal-500' : 'bg-teal-400'"
              :style="{ width: task.percent + '%' }"
            ></div>
          </div>
          <span class="text-slate-400 w-7 text-right flex-shrink-0">{{ task.percent }}%</span>
        </template>
      </div>
    </div>

    <!-- 文件列表（拖拽区域） -->
    <div
      class="flex-1 overflow-y-auto relative"
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
      <!-- 空目录（且不在新建目录模式） -->
      <div v-else-if="sftpFiles.length === 0 && !mkdirMode" class="flex items-center justify-center h-full text-slate-400 text-sm gap-2">
        <i class="fas fa-folder-open"></i>空目录
      </div>
      <!-- 文件表格 -->
      <table v-else class="w-full text-xs">
        <tbody>
          <!-- 新建目录输入行（插在列表顶部） -->
          <tr v-if="mkdirMode" class="border-b border-teal-100 bg-teal-50">
            <td class="px-3 py-1.5" colspan="4">
              <div class="flex items-center gap-2">
                <div class="w-5 h-5 rounded bg-amber-400 flex items-center justify-center flex-shrink-0">
                  <i class="fas fa-folder text-white text-xs"></i>
                </div>
                <input
                  v-model="mkdirName"
                  class="input text-xs py-0.5"
                  placeholder="输入目录名称..."
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
            <!-- 图标 + 名称 -->
            <td class="px-3 py-2 w-full">
              <div class="flex items-center gap-2 min-w-0">
                <div class="relative w-5 h-5 flex-shrink-0">
                  <div :class="['w-5 h-5 rounded flex items-center justify-center', file.isDir ? 'bg-amber-400' : 'bg-slate-400']">
                    <i :class="getFileIcon(file)" class="text-white text-xs"></i>
                  </div>
                  <i v-if="file.isLink" class="fas fa-link absolute -bottom-0.5 -right-0.5 text-white text-[8px]" style="text-shadow: 0 0 2px rgba(0,0,0,0.6)"></i>
                </div>
                <!-- 重命名输入 -->
                <template v-if="renamingFile?.name === file.name">
                  <input
                    v-model="renameNewName"
                    class="input text-xs py-0.5 min-w-0"
                    autofocus
                    @keyup.enter="confirmRename()"
                    @keyup.esc="cancelRename()"
                  />
                  <button class="btn-icon btn-icon-teal !w-6 !h-6 flex-shrink-0" @click="confirmRename()">
                    <i class="fas fa-check text-xs"></i>
                  </button>
                  <button class="btn-icon btn-icon-slate !w-6 !h-6 flex-shrink-0" @click="cancelRename()">
                    <i class="fas fa-xmark text-xs"></i>
                  </button>
                </template>
                <template v-else>
                  <span
                    class="truncate"
                    :class="file.isDir ? 'text-slate-700 font-medium cursor-pointer hover:text-teal-600' : 'text-slate-600'"
                    @click="file.isDir && sftpEnter(file)"
                  >{{ file.name }}<span v-if="file.isLink && file.linkTarget" class="ml-1 text-slate-400 text-xs font-normal">{{ file.linkTarget }}</span></span>
                </template>
              </div>
            </td>
            <!-- 大小 -->
            <td class="px-2 py-2 text-slate-400 whitespace-nowrap hidden sm:table-cell">
              {{ file.isDir ? '—' : formatSize(file.size) }}
            </td>
            <!-- 修改时间 -->
            <td class="px-2 py-2 text-slate-400 whitespace-nowrap hidden md:table-cell">
              {{ formatTime(file.modTime) }}
            </td>
            <!-- 操作 -->
            <td class="px-2 py-2 whitespace-nowrap">
              <div class="flex items-center gap-1 justify-end">
                <template v-if="file.isDir">
                  <button class="btn-icon btn-icon-slate !w-6 !h-6" title="进入目录" @click="sftpEnter(file)">
                    <i class="fas fa-folder-open text-xs"></i>
                  </button>
                </template>
                <template v-else>
                  <button
                    class="btn-icon !w-6 !h-6"
                    :class="downloadingFile === file.name ? 'btn-icon-teal' : 'btn-icon-slate'"
                    title="下载"
                    :disabled="!!downloadingFile"
                    @click="sftpDownload(file)"
                  >
                    <i
                      class="fas text-xs"
                      :class="downloadingFile === file.name ? 'fa-spinner animate-spin' : 'fa-download'"
                    ></i>
                  </button>
                </template>
                <button class="btn-icon btn-icon-slate !w-6 !h-6" title="重命名" @click="startRename(file)">
                  <i class="fas fa-spell-check text-xs"></i>
                </button>
                <button class="btn-icon btn-icon-red !w-6 !h-6" title="删除" @click="sftpDelete(file)">
                  <i class="fas fa-trash text-xs"></i>
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
