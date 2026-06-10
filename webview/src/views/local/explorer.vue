<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { FilerFileInfo } from '@/service/types'

import { downloadFile, formatFileSize, formatTime, getFileIcon, isEditableFile, isPreviewableFile } from '@/helper/utils'

import PageSearch from '@/component/page-search.vue'

import ChmodModal from './explorer/chmod-modal.vue'
import CreateModal from './explorer/create-modal.vue'
import DeleteModal from './explorer/delete-modal.vue'
import MkdirModal from './explorer/mkdir-modal.vue'
import ModifyModal from './explorer/modify-modal.vue'
import PreviewModal from './explorer/preview-modal.vue'
import RenameModal from './explorer/rename-modal.vue'
import UnzipModal from './explorer/unzip-modal.vue'
import UploadModal from './explorer/upload-modal.vue'
import ZipModal from './explorer/zip-modal.vue'

@Component({
    components: {
        PageSearch, ChmodModal, CreateModal, DeleteModal, MkdirModal,
        ModifyModal, PreviewModal, RenameModal, UnzipModal, UploadModal, ZipModal
    }
})
class FileExplorer extends Vue {
    portal = usePortal()

    // ─── Refs ───
    @Ref readonly modifyModalRef!: InstanceType<typeof ModifyModal>
    @Ref readonly previewModalRef!: InstanceType<typeof PreviewModal>
    @Ref readonly renameModalRef!: InstanceType<typeof RenameModal>
    @Ref readonly chmodModalRef!: InstanceType<typeof ChmodModal>
    @Ref readonly zipModalRef!: InstanceType<typeof ZipModal>
    @Ref readonly deleteModalRef!: InstanceType<typeof DeleteModal>
    @Ref readonly unzipModalRef!: InstanceType<typeof UnzipModal>
    @Ref readonly mkdirModalRef!: InstanceType<typeof MkdirModal>
    @Ref readonly createModalRef!: InstanceType<typeof CreateModal>
    @Ref readonly uploadModal!: InstanceType<typeof UploadModal>

    // ─── 数据属性 ───
    formatFileSize = formatFileSize
    formatTime = formatTime
    getFileIcon = getFileIcon
    isEditableFile = isEditableFile
    isPreviewableFile = isPreviewableFile
    searchText = ''
    // 记录已计算大小的文件夹路径
    calculatedDirs = new Set<string>()
    // 记录正在计算大小的文件夹路径
    calculatingDirs = new Set<string>()

    // ─── 计算属性 ───
    get files() {
        return this.portal.files
    }

    get filteredFiles() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.files
        return this.files.filter((file: FilerFileInfo) =>
            file.name.toLowerCase().includes(keyword) ||
            file.path.toLowerCase().includes(keyword) ||
            file.mode.toLowerCase().includes(keyword)
        )
    }

    get paths() {
        if (!this.portal.currentPath || this.portal.currentPath === '/') return []
        return this.portal.currentPath.split('/').filter((part: string) => part)
    }

    // ─── 方法 ───
    navigateTo(path: string) {
        this.portal.loadFiles(path)
    }

    async download(file: FilerFileInfo) {
        const response = await api.filerDownload(file.path)
        downloadFile(file.name, response)
    }

    refreshFiles() {
        this.portal.loadFiles()
    }

    async calcDirSize(file: FilerFileInfo) {
        if (!file.isDir || this.calculatingDirs.has(file.path)) return
        this.calculatingDirs.add(file.path)
        try {
            const result = await api.filerDirSize(file.path)
            file.size = result.payload?.size || 0
            this.calculatedDirs.add(file.path)
        } catch (error: unknown) {
            console.error('计算目录大小失败:', error)
        } finally {
            this.calculatingDirs.delete(file.path)
        }
    }

    // ─── 生命周期 ───
    mounted() {
        this.portal.loadFiles('/')
    }
}

export default toNative(FileExplorer)
</script>

<template>
  <div>
    <!-- Toolbar Bar -->
    <div class="card">
      <div class="card-toolbar">
        <div class="flex items-center justify-between gap-3">
          <nav aria-label="breadcrumb" class="flex-1 min-w-0">
            <ol class="flex items-center space-x-2 text-sm overflow-x-auto">
              <li class="flex-shrink-0">
                <button type="button" class="breadcrumb-btn" @click="navigateTo('/')">
                  <i class="fas fa-home text-base"></i>
                </button>
              </li>

              <template v-for="(part, index) in paths" :key="index">
                <li class="text-slate-300 flex-shrink-0">
                  <i class="fas fa-chevron-right text-xs"></i>
                </li>
                <li v-if="Number(index) < paths.length - 1" class="flex-shrink-0">
                  <button type="button" class="breadcrumb-btn" @click="navigateTo('/' + paths.slice(0, Number(index) + 1).join('/'))">
                    {{ part }}
                  </button>
                </li>
                <li v-else class="px-3 py-1.5 text-primary-600 font-semibold flex-shrink-0">
                  {{ part }}
                </li>
              </template>
            </ol>
          </nav>

          <div class="hidden md:flex items-center gap-2 flex-shrink-0">
            <PageSearch v-model="searchText" search-key="filer-explorer" placeholder="搜索文件名、路径或权限..." focus-color="primary" type-to-search />
            <button v-if="portal.hasPerm('GET /api/filer/files')" class="btn btn-secondary" @click="refreshFiles()">
              <i class="fas fa-rotate"></i><span>刷新</span>
            </button>
            <button v-if="portal.hasPerm('POST /api/filer/dir')" class="btn btn-secondary" @click="mkdirModalRef.show()">
              <i class="fas fa-folder"></i><span>新建目录</span>
            </button>
            <button v-if="portal.hasPerm('POST /api/filer/file')" class="btn btn-secondary" @click="createModalRef.show()">
              <i class="fas fa-file"></i><span>新建文件</span>
            </button>
            <button v-if="portal.hasPerm('POST /api/filer/upload')" class="btn btn-primary" @click="uploadModal.show()">
              <i class="fas fa-upload"></i><span>上传文件</span>
            </button>
          </div>
          <!-- 移动端图标按钮 -->
          <div class="flex md:hidden items-center gap-1.5 flex-shrink-0">
            <button v-if="portal.hasPerm('GET /api/filer/files')" class="btn btn-secondary w-9 h-9 !px-0" title="刷新" @click="refreshFiles()">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button v-if="portal.hasPerm('POST /api/filer/dir')" class="btn btn-secondary w-9 h-9 !px-0" title="新建目录" @click="mkdirModalRef.show()">
              <i class="fas fa-folder text-sm"></i>
            </button>
            <button v-if="portal.hasPerm('POST /api/filer/file')" class="btn btn-secondary w-9 h-9 !px-0" title="新建文件" @click="createModalRef.show()">
              <i class="fas fa-file text-sm"></i>
            </button>
            <button v-if="portal.hasPerm('POST /api/filer/upload')" class="btn btn-primary w-9 h-9 !px-0" title="上传文件" @click="uploadModal.show()">
              <i class="fas fa-upload text-sm"></i>
            </button>
          </div>
        </div>
      </div>
      <div class="mobile-search">
        <PageSearch v-model="searchText" search-key="filer-explorer" placeholder="搜索文件名、路径或权限..." width-class="w-full" focus-color="primary" />
      </div>

      <!-- Loading State -->
      <div v-if="portal.filerLoading" class="card-body">
        <div class="empty-state">
          <div class="w-12 h-12 spinner mb-3"></div>
          <p class="text-slate-500">加载中...</p>
        </div>
      </div>

      <!-- Empty State -->
      <div v-else-if="filteredFiles.length === 0" class="card-body">
        <div class="empty-state">
          <div class="empty-state-icon">
            <i class="fas fa-folder-open text-4xl text-slate-300"></i>
          </div>
          <p class="text-slate-600 font-medium mb-1">{{ files.length === 0 ? '此目录为空' : '未找到匹配文件' }}</p>
          <p class="text-sm text-slate-400">{{ files.length === 0 ? '上传文件或创建新目录开始使用' : '尝试更换关键词或清空搜索条件' }}</p>
        </div>
      </div>

      <!-- 文件列表 -->
      <template v-else>
        <!-- 桌面端表格视图 -->
        <div class="card-table hidden md:block">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="th">名称</th>
                <th class="w-32 th">大小</th>
                <th class="w-32 th">权限</th>
                <th class="w-32 th">修改时间</th>
                <th class="w-40 th-right">操作</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="file in filteredFiles" :key="file.path" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3 max-w-[280px]">
                  <div class="flex items-center gap-2 min-w-0">
                    <div :class="['row-icon', file.isDir ? 'bg-amber-400' : 'bg-blue-400']">
                      <i :class="getFileIcon(file)" class="text-white text-sm"></i>
                    </div>
                    <div class="min-w-0">
                      <button v-if="file.isDir" type="button" class="font-medium text-slate-800 hover:text-primary-600 transition-colors truncate block text-left" @click="navigateTo(file.path)">
                        {{ file.name }}
                      </button>
                      <span v-else class="font-medium text-slate-800 truncate block">{{ file.name }}</span>
                    </div>
                  </div>
                </td>
                
                <td class="px-4 py-3">
                  <span v-if="!file.isDir" class="text-sm text-slate-600">
                    {{ formatFileSize(file.size) }}
                  </span>
                  <button 
                    v-else 
                    type="button" 
                    class="group text-sm text-slate-400 hover:text-primary-600 transition-colors w-20 text-left"
                    @click="calcDirSize(file)"
                  >
                    <template v-if="calculatedDirs.has(file.path)">{{ formatFileSize(file.size) }}</template>
                    <template v-else-if="calculatingDirs.has(file.path)">计算中...</template>
                    <template v-else>
                      <span class="group-hover:hidden">--</span>
                      <span class="group-hover:inline hidden">计算大小</span>
                    </template>
                  </button>
                </td>
                
                <td class="px-4 py-3">
                  <code class="px-2 py-1 bg-slate-100 rounded-lg text-xs text-slate-700 font-mono">
                    {{ file.mode }}
                  </code>
                </td>
                
                <td class="px-4 py-3">
                  <span class="text-sm text-slate-600 whitespace-nowrap">{{ formatTime(file.modTime) }}</span>
                </td>
                
                <td class="px-4 py-3">
                  <div class="flex justify-end items-center gap-1">
                    <!-- Directory Actions -->
                    <template v-if="file.isDir">
                      <button v-if="portal.hasPerm('GET /api/filer/files')" class="btn-icon btn-icon-slate" title="进入目录" @click="navigateTo(file.path)">
                        <i class="fas fa-folder-open text-xs"></i>
                      </button>
                      <button v-if="portal.hasPerm('POST /api/filer/zip')" class="btn-icon btn-icon-amber" title="压缩" @click="zipModalRef.show(file)">
                        <i class="fas fa-file-zipper text-xs"></i>
                      </button>
                      <button v-if="portal.hasPerm('POST /api/filer/rename')" class="btn-icon btn-icon-blue" title="重命名" @click="renameModalRef.show(file)">
                        <i class="fas fa-spell-check text-xs"></i>
                      </button>
                      <button v-if="portal.hasPerm('PUT /api/filer/chmod')" class="btn-icon btn-icon-slate" title="权限" @click="chmodModalRef.show(file)">
                        <i class="fas fa-key text-xs"></i>
                      </button>
                      <button v-if="portal.hasPerm('DELETE /api/filer/file')" class="btn-icon btn-icon-red" title="删除" @click="deleteModalRef.show(file)">
                        <i class="fas fa-trash text-xs"></i>
                      </button>
                    </template>
                    
                    <!-- File Actions -->
                    <template v-else>
                      <button v-if="portal.hasPerm('GET /api/filer/download')" class="btn-icon btn-icon-slate" title="下载" @click="download(file)">
                        <i class="fas fa-download text-xs"></i>
                      </button>
                      <button v-if="isPreviewableFile(file.name) && portal.hasPerm('GET /api/filer/download')" class="btn-icon btn-icon-slate" title="预览" @click="previewModalRef.show(file)">
                        <i class="fas fa-eye text-xs"></i>
                      </button>
                      <button v-if="file.name.endsWith('.zip') && portal.hasPerm('POST /api/filer/unzip')" class="btn-icon btn-icon-amber" title="解压" @click="unzipModalRef.show(file)">
                        <i class="fas fa-file-zipper text-xs"></i>
                      </button>
                      <button 
                        v-if="isEditableFile(file.name) && portal.hasPerm('GET /api/filer/file') && portal.hasPerm('PUT /api/filer/file')"
                        class="btn-icon btn-icon-blue"
                        title="编辑" 
                        @click="modifyModalRef.show(file)"
                      >
                        <i class="fas fa-file-pen text-xs"></i>
                      </button>
                      <button v-if="portal.hasPerm('POST /api/filer/rename')" class="btn-icon btn-icon-blue" title="重命名" @click="renameModalRef.show(file)">
                        <i class="fas fa-spell-check text-xs"></i>
                      </button>
                      <button v-if="portal.hasPerm('PUT /api/filer/chmod')" class="btn-icon btn-icon-slate" title="权限" @click="chmodModalRef.show(file)">
                        <i class="fas fa-key text-xs"></i>
                      </button>
                      <button v-if="portal.hasPerm('DELETE /api/filer/file')" class="btn-icon btn-icon-red" title="删除" @click="deleteModalRef.show(file)">
                        <i class="fas fa-trash text-xs"></i>
                      </button>
                    </template>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 移动端卡片视图 -->
        <div class="card-body md:hidden space-y-3">
          <div v-for="file in filteredFiles" :key="file.path" class="card-interactive">
            <!-- 顶部：文件信息和图标 -->
            <div class="card-info-row">
              <div :class="['list-icon', file.isDir ? 'bg-amber-400' : 'bg-blue-400']">
                <i :class="getFileIcon(file)" class="text-white text-base"></i>
              </div>
              <div class="min-w-0">
                <button 
                  v-if="file.isDir" 
                  type="button"
                  class="font-medium text-slate-800 hover:text-primary-600 transition-colors text-sm truncate block text-left" 
                  @click="navigateTo(file.path)"
                >
                  {{ file.name }}
                </button>
                <span v-else class="font-medium text-slate-800 text-sm truncate block">{{ file.name }}</span>
                <span v-if="!file.isDir" class="text-xs text-slate-400 truncate block mt-0.5">{{ formatFileSize(file.size) }}</span>
                <button 
                  v-else 
                  type="button" 
                  class="group text-xs text-slate-400 hover:text-primary-600 transition-colors truncate block mt-0.5 text-left"
                  @click="calcDirSize(file)"
                >
                  <template v-if="calculatedDirs.has(file.path)">{{ formatFileSize(file.size) }}</template>
                  <template v-else-if="calculatingDirs.has(file.path)">计算中...</template>
                  <template v-else>
                    <span class="group-hover:hidden">--</span>
                    <span class="group-hover:inline hidden">计算大小</span>
                  </template>
                </button>
              </div>
            </div>

            <!-- 时间 -->
            <div class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">修改时间</span>
              <span class="text-xs text-slate-500">{{ formatTime(file.modTime) }}</span>
            </div>

            <!-- 权限 -->
            <div class="flex items-start gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">权限</span>
              <code class="text-xs bg-slate-100 px-2 py-0.5 rounded-lg font-mono text-slate-700">{{ file.mode }}</code>
            </div>

            <!-- 底部：操作按钮 -->
            <div class="card-actions">
              <!-- Directory Actions -->
              <template v-if="file.isDir">
                <button v-if="portal.hasPerm('GET /api/filer/files')" class="btn-icon btn-icon-slate" title="进入目录" @click="navigateTo(file.path)">
                  <i class="fas fa-folder-open text-xs"></i><span class="text-xs ml-1">进入</span>
                </button>
                <button v-if="portal.hasPerm('POST /api/filer/zip')" class="btn-icon btn-icon-amber" title="压缩" @click="zipModalRef.show(file)">
                  <i class="fas fa-file-zipper text-xs"></i><span class="text-xs ml-1">压缩</span>
                </button>
                <button v-if="portal.hasPerm('POST /api/filer/rename')" class="btn-icon btn-icon-blue" title="重命名" @click="renameModalRef.show(file)">
                  <i class="fas fa-spell-check text-xs"></i><span class="text-xs ml-1">重命名</span>
                </button>
                <button v-if="portal.hasPerm('PUT /api/filer/chmod')" class="btn-icon btn-icon-slate" title="权限" @click="chmodModalRef.show(file)">
                  <i class="fas fa-key text-xs"></i><span class="text-xs ml-1">权限</span>
                </button>
                <button v-if="portal.hasPerm('DELETE /api/filer/file')" class="btn-icon btn-icon-red" title="删除" @click="deleteModalRef.show(file)">
                  <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
                </button>
              </template>

              <!-- File Actions -->
              <template v-else>
                <button v-if="portal.hasPerm('GET /api/filer/download')" class="btn-icon btn-icon-slate" title="下载" @click="download(file)">
                  <i class="fas fa-download text-xs"></i><span class="text-xs ml-1">下载</span>
                </button>
                <button v-if="isPreviewableFile(file.name) && portal.hasPerm('GET /api/filer/download')" class="btn-icon btn-icon-slate" title="预览" @click="previewModalRef.show(file)">
                  <i class="fas fa-eye text-xs"></i><span class="text-xs ml-1">预览</span>
                </button>
                <button v-if="file.name.endsWith('.zip') && portal.hasPerm('POST /api/filer/unzip')" class="btn-icon btn-icon-amber" title="解压" @click="unzipModalRef.show(file)">
                  <i class="fas fa-file-zipper text-xs"></i><span class="text-xs ml-1">解压</span>
                </button>
                <button 
                  v-if="isEditableFile(file.name) && portal.hasPerm('GET /api/filer/file') && portal.hasPerm('PUT /api/filer/file')"
                  class="btn-icon btn-icon-blue"
                  title="编辑" 
                  @click="modifyModalRef.show(file)"
                >
                  <i class="fas fa-file-pen text-xs"></i><span class="text-xs ml-1">编辑</span>
                </button>
                <button v-if="portal.hasPerm('POST /api/filer/rename')" class="btn-icon btn-icon-blue" title="重命名" @click="renameModalRef.show(file)">
                  <i class="fas fa-spell-check text-xs"></i><span class="text-xs ml-1">重命名</span>
                </button>
                <button v-if="portal.hasPerm('PUT /api/filer/chmod')" class="btn-icon btn-icon-slate" title="权限" @click="chmodModalRef.show(file)">
                  <i class="fas fa-key text-xs"></i><span class="text-xs ml-1">权限</span>
                </button>
                <button v-if="portal.hasPerm('DELETE /api/filer/file')" class="btn-icon btn-icon-red" title="删除" @click="deleteModalRef.show(file)">
                  <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
                </button>
              </template>
            </div>
          </div>
        </div>
      </template>
    </div>

    <!-- Modals -->
    <ModifyModal ref="modifyModalRef" />
    <PreviewModal ref="previewModalRef" />
    <RenameModal ref="renameModalRef" />
    <ChmodModal ref="chmodModalRef" />
    <DeleteModal ref="deleteModalRef" />
    <ZipModal ref="zipModalRef" />
    <UnzipModal ref="unzipModalRef" />
    <MkdirModal ref="mkdirModalRef" />
    <CreateModal ref="createModalRef" />
    <UploadModal ref="uploadModal" />
  </div>
</template>
