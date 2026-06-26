<script lang="ts">
/**
 * 通用文件管理面板组件
 *
 * Props：
 *   adapter       操作适配器（必填）
 *   initialPath   初始路径，默认 /
 *   stickyToolbar 页面完整展示时吸顶（默认 false）
 *   showSearch    是否显示搜索框（默认 false）
 *   showBatchOps  是否启用多选/批量删除/移动（默认 false）
 *   showMobileView 是否显示移动端卡片视图（默认 false）
 *
 * 用法：
 *   // 本地文件管理（完整页面）
 *   <ExplorerPanel :adapter="createFilerAdapter()" :sticky-toolbar="true" :show-search="true" :show-batch-ops="true" :show-mobile-view="true" />
 *
 *   // SSH SFTP 侧边栏（嵌入式面板）
 *   <ExplorerPanel :adapter="createSftpAdapter(hostId)" :show-batch-ops="true" />
 */
import { Component, Prop, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import { downloadBlob, formatFileSize, formatTime, formatUnixTime, getFileIcon, isEditableFile, isPreviewableFile } from '@/helper/utils'

import ChmodModal from './widget/chmod-modal.vue'
import CreateModal from './widget/create-modal.vue'
import DeleteModal from './widget/delete-modal.vue'
import ModifyModal from './widget/modify-modal.vue'
import PreviewModal from './widget/preview-modal.vue'
import RenameModal from './widget/rename-modal.vue'
import UnzipModal from './widget/unzip-modal.vue'
import Upload from './widget/upload.vue'
import ZipModal from './widget/zip-modal.vue'
import MkdirRow from './widget/mkdir-row.vue'
import UploadZone from './widget/upload-zone.vue'
import type { FileInfo, ExplorerAdapter } from './types'

@Component({
    expose: ['refresh', 'navigate'],
    emits: ['path-change'],
    components: {
        Upload, ChmodModal, RenameModal, ModifyModal,
        PreviewModal, DeleteModal, CreateModal,
        ZipModal, UnzipModal, MkdirRow, UploadZone,
    },
})
class ExplorerPanel extends Vue {
    @Prop({ required: true }) adapter!: ExplorerAdapter
    @Prop({ default: '/' }) initialPath!: string
    @Prop({ default: false }) stickyToolbar!: boolean
    @Prop({ default: false }) showSearch!: boolean
    @Prop({ default: false }) showBatchOps!: boolean
    @Prop({ default: false }) showMobileView!: boolean

    portal = usePortal()

    @Ref readonly uploadRef!: InstanceType<typeof Upload>
    @Ref readonly uploadZoneRef!: InstanceType<typeof UploadZone>
    @Ref readonly chmodModalRef!: InstanceType<typeof ChmodModal>
    @Ref readonly renameModalRef!: InstanceType<typeof RenameModal>
    @Ref readonly modifyModalRef!: InstanceType<typeof ModifyModal>
    @Ref readonly previewModalRef!: InstanceType<typeof PreviewModal>
    @Ref readonly deleteModalRef!: InstanceType<typeof DeleteModal>
    @Ref readonly createModalRef!: InstanceType<typeof CreateModal>
    @Ref readonly zipModalRef!: InstanceType<typeof ZipModal>
    @Ref readonly unzipModalRef!: InstanceType<typeof UnzipModal>

    currentPath = '/'
    files: FileInfo[] = []
    loading = false
    error = ''
    searchText = ''
    selectedPaths: string[] = []
    calculatedDirs = new Set<string>()
    calculatingDirs = new Set<string>()
    downloadingFile = ''
    mkdirMode = false
    private refreshDebounceTimer: ReturnType<typeof setTimeout> | null = null

    formatFileSize = formatFileSize
    getFileIcon = getFileIcon
    isEditableFile = isEditableFile
    isPreviewableFile = isPreviewableFile

    formatModTime(modTime: string | number): string {
        return typeof modTime === 'number' ? formatUnixTime(modTime) : formatTime(modTime)
    }

    get filteredFiles(): FileInfo[] {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.files
        return this.files.filter(f =>
            f.name.toLowerCase().includes(keyword) ||
            f.path.toLowerCase().includes(keyword) ||
            f.mode.toLowerCase().includes(keyword)
        )
    }

    get pathParts(): { label: string; path: string }[] {
        if (!this.currentPath || this.currentPath === '/') return []
        return this.currentPath.split('/').filter(Boolean).map((part, index, arr) => ({
            label: part,
            path: '/' + arr.slice(0, index + 1).join('/'),
        }))
    }

    get can() { return this.adapter.can ?? {} }
    get canSelect(): boolean { return !!(this.showBatchOps && (this.can.remove || this.can.rename)) }
    get allVisibleSelected(): boolean {
        return this.filteredFiles.length > 0 && this.filteredFiles.every(f => this.selectedPaths.includes(f.path))
    }
    get selectedFiles(): FileInfo[] { return this.files.filter(f => this.selectedPaths.includes(f.path)) }
    get selectedCount(): number { return this.selectedPaths.length }

    mounted() { this.currentPath = this.initialPath || '/'; this.loadFiles() }

    async loadFiles(path?: string) {
        const target = path ?? this.currentPath
        this.loading = true; this.error = ''
        try {
            const result = await this.adapter.list(target)
            this.currentPath = result.path
            this.files = [
                ...result.files.filter(f => f.isDir).sort((a, b) => a.name.localeCompare(b.name)),
                ...result.files.filter(f => !f.isDir).sort((a, b) => a.name.localeCompare(b.name)),
            ]
            this.$emit('path-change', this.currentPath)
        } catch (e: unknown) {
            this.error = (e instanceof Error ? e.message : '') || '加载失败'
            this.files = []
        } finally { this.loading = false }
    }

    refresh() { this.clearSelection(); this.loadFiles() }

    debouncedLoadFiles() {
        if (this.refreshDebounceTimer) clearTimeout(this.refreshDebounceTimer)
        this.refreshDebounceTimer = setTimeout(() => { this.refreshDebounceTimer = null; this.loadFiles() }, 300)
    }

    navigate(path: string) { this.clearSelection(); this.loadFiles(path) }

    isSelected(file: FileInfo) { return this.selectedPaths.includes(file.path) }
    toggleFileSelection(file: FileInfo) {
        this.selectedPaths = this.isSelected(file)
            ? this.selectedPaths.filter(p => p !== file.path)
            : [...this.selectedPaths, file.path]
    }
    toggleAllVisible() {
        const paths = this.filteredFiles.map(f => f.path)
        this.selectedPaths = this.allVisibleSelected
            ? this.selectedPaths.filter(p => !paths.includes(p))
            : Array.from(new Set([...this.selectedPaths, ...paths]))
    }
    clearSelection() { this.selectedPaths = [] }

    async calcDirSize(file: FileInfo) {
        if (!file.isDir || !this.adapter.dirSize || this.calculatingDirs.has(file.path)) return
        this.calculatingDirs.add(file.path)
        try {
            const size = await this.adapter.dirSize(file.path)
            if (size !== null) { file.size = size; this.calculatedDirs.add(file.path) }
        } catch { } finally { this.calculatingDirs.delete(file.path) }
    }

    async download(file: FileInfo) {
        this.downloadingFile = file.name
        try { downloadBlob(await this.adapter.download(file.path), file.name) }
        finally { this.downloadingFile = '' }
    }

    startDelete(file: FileInfo) { this.deleteModalRef?.show(this.adapter, file) }
    startBatchDelete() { if (this.selectedCount > 0) this.deleteModalRef?.show(this.adapter, this.selectedFiles) }

    startMkdir() { this.mkdirMode = true }
    cancelMkdir() { this.mkdirMode = false }
    async confirmMkdir(name: string) {
        const dirName = name.trim()
        if (!dirName) return
        try {
            await this.adapter.mkdir(this.currentPath.replace(/\/+$/, '') + '/' + dirName)
            this.portal.showNotification('success', '创建成功')
            this.cancelMkdir(); this.loadFiles()
        } catch (e: unknown) {
            this.portal.showNotification('error', (e instanceof Error ? e.message : '') || '创建失败')
        }
    }

    startRename(file: FileInfo) { this.renameModalRef?.show(this.adapter, file) }
    startBatchMove() { if (this.selectedCount > 0) this.renameModalRef?.show(this.adapter, this.selectedFiles) }
    startChmod(file: FileInfo) { this.chmodModalRef?.show(this.adapter, file) }
    startModify(file: FileInfo) { this.modifyModalRef?.show(this.adapter, file) }
    startPreview(file: FileInfo) { this.previewModalRef?.show(this.adapter, file) }
    startCreate() { this.createModalRef?.show(this.adapter, this.currentPath) }
    startZip(file: FileInfo) { this.zipModalRef?.show(this.adapter, file) }
    startUnzip(file: FileInfo) { this.unzipModalRef?.show(this.adapter, file) }
    triggerUpload() { this.uploadRef?.triggerFileSelect() }
}

export default toNative(ExplorerPanel)
</script>

<template>
  <!-- 无渲染：提供拖拽事件绑定和 dragOver 状态 -->
  <UploadZone ref="uploadZoneRef" :enabled="!!can.upload" :upload-ref="uploadRef" />

  <div
    class="relative flex flex-col h-full"
    v-bind="uploadZoneRef?.dragAttrs"
  >
    <!-- 拖拽遮罩 -->
    <div
      v-if="uploadZoneRef?.dragOver && can.upload"
      class="absolute inset-0 z-20 flex flex-col items-center justify-center gap-2 border-2 border-dashed rounded pointer-events-none bg-primary-50/90 border-primary-400"
    >
      <i class="fas fa-cloud-arrow-up text-2xl text-primary-500"></i>
      <span class="text-sm font-medium text-primary-600">松手上传到当前目录</span>
      <span class="text-xs text-primary-400">支持文件夹</span>
    </div>

    <!-- ─── 工具栏 ──────────────────────────────────────────────────────────── -->
    <div :class="['page-toolbar', { 'page-toolbar-static': !stickyToolbar }]">
      <div class="flex items-center justify-between gap-3">
        <!-- 面包屑导航 -->
        <nav aria-label="breadcrumb" class="flex-1 min-w-0">
          <ol class="flex items-center space-x-2 text-sm overflow-x-auto">
            <li class="flex-shrink-0">
              <button type="button" class="breadcrumb-btn" @click="navigate('/')">
                <i class="fas fa-home text-base"></i>
              </button>
            </li>
            <template v-for="(part, index) in pathParts" :key="index">
              <li class="text-slate-300 flex-shrink-0"><i class="fas fa-chevron-right text-xs"></i></li>
              <li v-if="index < pathParts.length - 1" class="flex-shrink-0">
                <button type="button" class="breadcrumb-btn" @click="navigate(part.path)">{{ part.label }}</button>
              </li>
              <li v-else class="px-3 py-1.5 text-primary-600 font-semibold flex-shrink-0">{{ part.label }}</li>
            </template>
          </ol>
        </nav>

        <!-- 搜索框 -->
        <input v-if="showSearch" v-model="searchText" class="input text-sm py-1 w-48 hidden md:block" placeholder="搜索文件..." />

        <!-- 桌面端操作按钮 -->
        <div class="action-group-desktop">
          <button v-if="can.list" class="btn btn-secondary" @click="refresh()">
            <i class="fas fa-rotate"></i><span>刷新</span>
          </button>
          <template v-if="selectedCount === 0">
            <button v-if="can.mkdir" class="btn btn-secondary" @click="startMkdir()">
              <i class="fas fa-folder"></i><span>新建目录</span>
            </button>
            <button v-if="can.createFile" class="btn btn-secondary" @click="startCreate()">
              <i class="fas fa-file"></i><span>新建文件</span>
            </button>
            <button v-if="can.upload" class="btn btn-primary" @click="triggerUpload()">
              <i class="fas fa-upload"></i><span>上传文件</span>
            </button>
          </template>
          <template v-if="showBatchOps && selectedCount > 0">
            <button v-if="can.rename" class="btn btn-blue" @click="startBatchMove()">
              <i class="fas fa-folder-plus"></i><span>移动 {{ selectedCount }} 项</span>
            </button>
            <button v-if="can.remove" class="btn btn-danger" @click="startBatchDelete()">
              <i class="fas fa-trash"></i><span>删除 {{ selectedCount }} 项</span>
            </button>
          </template>
        </div>

        <!-- 移动端图标按钮 -->
        <div class="flex md:hidden items-center gap-1.5 flex-shrink-0">
          <button v-if="can.list" class="btn btn-secondary btn-square" title="刷新" @click="refresh()">
            <i class="fas fa-rotate text-sm"></i>
          </button>
          <template v-if="selectedCount === 0">
            <button v-if="can.mkdir" class="btn btn-secondary btn-square" title="新建目录" @click="startMkdir()">
              <i class="fas fa-folder text-sm"></i>
            </button>
            <button v-if="can.createFile" class="btn btn-secondary btn-square" title="新建文件" @click="startCreate()">
              <i class="fas fa-file text-sm"></i>
            </button>
            <button v-if="can.upload" class="btn btn-primary btn-square" title="上传文件" @click="triggerUpload()">
              <i class="fas fa-upload text-sm"></i>
            </button>
          </template>
          <template v-if="showBatchOps && selectedCount > 0">
            <button v-if="can.rename" class="btn btn-blue btn-square" :title="`移动 ${selectedCount} 项`" @click="startBatchMove()">
              <i class="fas fa-folder-plus text-sm"></i>
            </button>
            <button v-if="can.remove" class="btn btn-danger btn-square" :title="`删除 ${selectedCount} 项`" @click="startBatchDelete()">
              <i class="fas fa-trash text-sm"></i>
            </button>
          </template>
        </div>
      </div>

      <!-- 移动端搜索框 -->
      <div v-if="showSearch" class="mt-2 md:hidden">
        <input v-model="searchText" class="input text-sm py-1 w-full" placeholder="搜索文件..." />
      </div>
    </div>

    <!-- ─── 上传进度 ────────────────────────────────────────────────────────── -->
    <Upload
      v-if="can.upload"
      ref="uploadRef"
      :adapter="adapter"
      :current-path="currentPath"
      @done="refresh()"
      @refresh="debouncedLoadFiles()"
    />

    <!-- ─── 内容区 ──────────────────────────────────────────────────────────── -->
    <div class="flex-1 overflow-y-auto min-h-0">
      <!-- ─── 加载中 ──────────────────────────────────────────────────────────── -->
      <div v-if="loading" class="card-body">
        <div class="empty-state">
          <div class="spinner-lg"></div>
          <p class="text-slate-500">加载中...</p>
        </div>
      </div>

      <!-- ─── 错误 ────────────────────────────────────────────────────────────── -->
      <div v-else-if="error" class="card-body">
        <div class="empty-state">
          <i class="fas fa-circle-exclamation text-4xl text-red-300 mb-3"></i>
          <p class="text-slate-600 font-medium mb-1">加载失败</p>
          <p class="text-sm text-slate-400">{{ error }}</p>
          <button class="btn btn-secondary mt-4" @click="refresh()">重试</button>
        </div>
      </div>

      <!-- ─── 空目录 ──────────────────────────────────────────────────────────── -->
      <div v-else-if="filteredFiles.length === 0 && !mkdirMode" class="card-body pointer-events-none">
        <div class="empty-state">
          <div class="empty-state-icon"><i class="fas fa-folder-open text-4xl text-slate-300"></i></div>
          <p class="text-slate-600 font-medium mb-1">{{ files.length === 0 ? '此目录为空' : '未找到匹配文件' }}</p>
          <p class="text-sm text-slate-400">{{ files.length === 0 ? '上传文件或创建新目录开始使用' : '尝试更换关键词或清空搜索条件' }}</p>
        </div>
      </div>

      <!-- ─── 文件列表 ────────────────────────────────────────────────────────── -->
      <template v-else>
        <!-- 桌面端表格 -->
        <div :class="showMobileView ? 'card-table hidden md:block' : 'card-table'">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-100 border-b border-slate-200">
                <th v-if="canSelect" class="w-12 th">
                  <label class="check-label" title="选择全部可见文件">
                    <input type="checkbox" class="rounded border-slate-300 text-primary-500" :checked="allVisibleSelected" @change="toggleAllVisible()">
                  </label>
                </th>
                <th class="th">名称</th>
                <th class="w-32 th">大小</th>
                <th class="w-32 th">权限</th>
                <th class="w-32 th">修改时间</th>
                <th class="w-48 th-right">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-100">
              <MkdirRow
                v-if="mkdirMode"
                :can-select="canSelect"
                @confirm="(name: string) => confirmMkdir(name)"
                @cancel="cancelMkdir()"
              />

              <tr v-for="file in filteredFiles" :key="file.path" class="hover:bg-slate-50 transition-colors">
                <td v-if="canSelect" class="px-4 py-3 w-12">
                  <label class="check-label" :title="`选择 ${file.name}`">
                    <input type="checkbox" class="rounded border-slate-300 text-primary-500" :checked="isSelected(file)" @change="toggleFileSelection(file)">
                  </label>
                </td>
                <td class="px-4 py-3 max-w-[280px]">
                  <div class="inline-info">
                    <div :class="['row-icon', file.isDir ? 'bg-amber-400' : 'bg-blue-400']">
                      <i :class="getFileIcon(file)" class="text-white text-sm"></i>
                    </div>
                    <div class="min-w-0">
                      <button v-if="file.isDir" type="button" class="font-medium text-slate-800 hover:text-primary-600 transition-colors truncate block text-left" @click="navigate(file.path)">
                        {{ file.name }}
                      </button>
                      <span v-else class="item-title">{{ file.name }}</span>
                    </div>
                  </div>
                </td>
                <td class="px-4 py-3">
                  <span v-if="!file.isDir" class="text-sm text-slate-600">{{ formatFileSize(file.size) }}</span>
                  <button v-else-if="adapter.dirSize" type="button" class="group text-sm text-slate-400 hover:text-primary-600 transition-colors w-20 text-left" @click="calcDirSize(file)">
                    <template v-if="calculatedDirs.has(file.path)">{{ formatFileSize(file.size) }}</template>
                    <template v-else-if="calculatingDirs.has(file.path)">计算中...</template>
                    <template v-else><span class="group-hover:hidden">--</span><span class="group-hover:inline hidden">计算大小</span></template>
                  </button>
                </td>
                <td class="px-4 py-3">
                  <code class="px-2 py-1 bg-slate-100 rounded-lg text-xs text-slate-700 font-mono">{{ file.mode }}</code>
                </td>
                <td class="px-4 py-3">
                  <span class="text-sm text-slate-600 whitespace-nowrap">{{ formatModTime(file.modTime) }}</span>
                </td>
                <td class="px-4 py-3">
                  <div class="table-actions">
                    <template v-if="file.isDir">
                      <button v-if="can.list" class="btn-icon btn-icon-slate" title="进入目录" @click="navigate(file.path)">
                        <i class="fas fa-folder-open text-xs"></i>
                      </button>
                      <button v-if="can.zip" class="btn-icon btn-icon-amber" title="压缩" @click="startZip(file)">
                        <i class="fas fa-file-zipper text-xs"></i>
                      </button>
                    </template>
                    <template v-else>
                      <button v-if="can.download" class="btn-icon" :class="downloadingFile === file.name ? 'btn-icon-primary' : 'btn-icon-slate'" title="下载" :disabled="!!downloadingFile" @click="download(file)">
                        <i class="fas text-xs" :class="downloadingFile === file.name ? 'fa-spinner animate-spin' : 'fa-download'"></i>
                      </button>
                      <button v-if="can.preview && isPreviewableFile(file.name)" class="btn-icon btn-icon-slate" title="预览" @click="startPreview(file)">
                        <i class="fas fa-eye text-xs"></i>
                      </button>
                      <button v-if="can.unzip && file.name.endsWith('.zip')" class="btn-icon btn-icon-amber" title="解压" @click="startUnzip(file)">
                        <i class="fas fa-file-zipper text-xs"></i>
                      </button>
                      <button v-if="can.readFile && can.writeFile && isEditableFile(file.name)" class="btn-icon btn-icon-teal" title="编辑" @click="startModify(file)">
                        <i class="fas fa-file-pen text-xs"></i>
                      </button>
                    </template>
                    <button v-if="can.rename" class="btn-icon btn-icon-blue" title="重命名 / 移动" @click="startRename(file)">
                      <i class="fas fa-file-export text-xs"></i>
                    </button>
                    <button v-if="can.chmod" class="btn-icon btn-icon-indigo" title="权限" @click="startChmod(file)">
                      <i class="fas fa-key text-xs"></i>
                    </button>
                    <button v-if="can.remove" class="btn-icon btn-icon-red" title="删除" @click="startDelete(file)">
                      <i class="fas fa-trash text-xs"></i>
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 移动端卡片视图 -->
        <div v-if="showMobileView" class="card-body md:hidden space-y-3">
          <div v-for="file in filteredFiles" :key="file.path" class="card-interactive">
            <div class="card-info-row">
              <label v-if="canSelect" class="check-label flex-shrink-0">
                <input type="checkbox" class="rounded border-slate-300 text-primary-500" :checked="isSelected(file)" @change="toggleFileSelection(file)">
              </label>
              <div :class="['list-icon', file.isDir ? 'bg-amber-400' : 'bg-blue-400']">
                <i :class="getFileIcon(file)" class="text-white text-base"></i>
              </div>
              <div class="min-w-0">
                <button v-if="file.isDir" type="button" class="font-medium text-slate-800 hover:text-primary-600 transition-colors text-sm truncate block text-left" @click="navigate(file.path)">{{ file.name }}</button>
                <span v-else class="item-title-sm">{{ file.name }}</span>
                <span v-if="!file.isDir" class="item-subtitle">{{ formatFileSize(file.size) }}</span>
              </div>
            </div>
            <div class="card-prop-row">
              <span class="text-xs text-slate-400 flex-shrink-0">修改时间</span>
              <span class="text-xs text-slate-500">{{ formatModTime(file.modTime) }}</span>
            </div>
            <div class="card-prop-row-start">
              <span class="prop-label-start">权限</span>
              <code class="text-xs bg-slate-100 px-2 py-0.5 rounded-lg font-mono text-slate-700">{{ file.mode }}</code>
            </div>
            <div class="card-actions">
              <template v-if="file.isDir">
                <button v-if="can.list" class="btn-icon btn-icon-slate" @click="navigate(file.path)"><i class="fas fa-folder-open text-xs"></i><span class="text-xs ml-1">进入</span></button>
                <button v-if="can.zip" class="btn-icon btn-icon-amber" @click="startZip(file)"><i class="fas fa-file-zipper text-xs"></i><span class="text-xs ml-1">压缩</span></button>
              </template>
              <template v-else>
                <button v-if="can.download" class="btn-icon btn-icon-slate" @click="download(file)"><i class="fas fa-download text-xs"></i><span class="text-xs ml-1">下载</span></button>
                <button v-if="can.preview && isPreviewableFile(file.name)" class="btn-icon btn-icon-slate" @click="startPreview(file)"><i class="fas fa-eye text-xs"></i><span class="text-xs ml-1">预览</span></button>
                <button v-if="can.unzip && file.name.endsWith('.zip')" class="btn-icon btn-icon-amber" @click="startUnzip(file)"><i class="fas fa-file-zipper text-xs"></i><span class="text-xs ml-1">解压</span></button>
                <button v-if="can.readFile && can.writeFile && isEditableFile(file.name)" class="btn-icon btn-icon-teal" @click="startModify(file)"><i class="fas fa-file-pen text-xs"></i><span class="text-xs ml-1">编辑</span></button>
              </template>
              <button v-if="can.rename" class="btn-icon btn-icon-blue" @click="startRename(file)"><i class="fas fa-file-export text-xs"></i><span class="text-xs ml-1">移动</span></button>
              <button v-if="can.chmod" class="btn-icon btn-icon-indigo" @click="startChmod(file)"><i class="fas fa-key text-xs"></i><span class="text-xs ml-1">权限</span></button>
              <button v-if="can.remove" class="btn-icon btn-icon-red" @click="startDelete(file)"><i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span></button>
            </div>
          </div>
        </div>
      </template>
    </div>

    <!-- ─── 弹窗 ────────────────────────────────────────────────────────────── -->
    <ChmodModal ref="chmodModalRef" @success="refresh()" />
    <RenameModal ref="renameModalRef" @success="refresh()" />
    <ModifyModal ref="modifyModalRef" @success="refresh()" />
    <PreviewModal ref="previewModalRef" />
    <DeleteModal ref="deleteModalRef" @success="refresh()" />
    <CreateModal ref="createModalRef" @success="refresh()" />
    <ZipModal ref="zipModalRef" @success="refresh()" />
    <UnzipModal ref="unzipModalRef" @success="refresh()" />
  </div>
</template>
