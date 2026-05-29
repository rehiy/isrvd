<script lang="ts">
import { Component, Prop, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { SFTPFileInfo, SFTPListResult } from '@/service/types'

import { formatFileSize, formatUnixTime, getFileIcon } from '@/helper/utils'

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
    sftpUploading = false
    renamingFile: SFTPFileInfo | null = null
    renameNewName = ''
    mkdirMode = false
    mkdirName = ''
    pathEditMode = false
    pathEditValue = ''

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
            // 后端返回实际路径（初始加载时同步 home 目录）
            this.sftpPath = result.path
            const files = result.files || []
            // 目录在前，文件在后，各自按名称排序
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
        const newPath = this.sftpPath.replace(/\/+$/, '') + '/' + file.name
        this.sftpLoad(newPath)
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

    // ─── 下载 ───
    sftpDownload(file: SFTPFileInfo) {
        const filePath = this.sftpPath.replace(/\/+$/, '') + '/' + file.name
        const url = api.sftpDownloadURL(this.hostId, filePath, this.portal.token || '')
        const a = document.createElement('a')
        a.href = url
        a.download = file.name
        a.click()
    }

    // ─── 删除 ───
    async sftpDelete(file: SFTPFileInfo) {
        const filePath = this.sftpPath.replace(/\/+$/, '') + '/' + file.name
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
        const base = this.sftpPath.replace(/\/+$/, '')
        const oldPath = base + '/' + this.renamingFile.name
        const newPath = base + '/' + this.renameNewName.trim()
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
        const dirPath = this.sftpPath.replace(/\/+$/, '') + '/' + this.mkdirName.trim()
        try {
            await api.sftpMkdir(this.hostId, { path: dirPath })
            this.portal.showNotification('success', '创建成功')
            this.cancelMkdir()
            this.sftpLoad()
        } catch (e: unknown) {
            this.portal.showNotification('error', (e instanceof Error ? e.message : '') || '创建失败')
        }
    }

    // ─── 上传 ───
    triggerUpload() {
        this.uploadInputRef?.click()
    }

    async handleUpload(e: Event) {
        const input = e.target as HTMLInputElement
        const files = input.files
        if (!files || files.length === 0) return
        this.sftpUploading = true
        try {
            for (const file of Array.from(files)) {
                const form = new FormData()
                form.append('file', file)
                await api.sftpUpload(this.hostId, this.sftpPath, form)
            }
            this.portal.showNotification('success', `上传成功（${files.length} 个文件）`)
            this.sftpLoad()
        } catch (e: unknown) {
            this.portal.showNotification('error', (e instanceof Error ? e.message : '') || '上传失败')
        } finally {
            this.sftpUploading = false
            input.value = ''
        }
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

    <!-- 文件列表 -->
    <div class="flex-1 overflow-y-auto">
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
                  >{{ file.name }}<span v-if="file.isLink" class="ml-1 text-slate-400 text-xs font-normal">{{ file.linkTarget }}</span></span>
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
                  <button class="btn-icon btn-icon-slate !w-6 !h-6" title="下载" @click="sftpDownload(file)">
                    <i class="fas fa-download text-xs"></i>
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
