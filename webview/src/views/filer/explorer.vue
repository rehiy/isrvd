<script lang="ts">
import { Component, Inject, Ref, Vue, toNative } from 'vue-facing-decorator'

import { APP_ACTIONS_KEY, APP_STATE_KEY } from '@/store/state'
import type { AppActions, AppState } from '@/store/state'

import api from '@/service/api'
import type { FilerFileInfo } from '@/service/types'

import { downloadFile, formatFileSize, formatTime, getFileIcon, isEditableFile } from '@/helper/utils'

import ChmodModal from '@/views/filer/widget/chmod-modal.vue'
import CreateModal from '@/views/filer/widget/create-modal.vue'
import DeleteModal from '@/views/filer/widget/delete-modal.vue'
import MkdirModal from '@/views/filer/widget/mkdir-modal.vue'
import ModifyModal from '@/views/filer/widget/modify-modal.vue'
import RenameModal from '@/views/filer/widget/rename-modal.vue'
import UnzipModal from '@/views/filer/widget/unzip-modal.vue'
import UploadModal from '@/views/filer/widget/upload-modal.vue'
import ZipModal from '@/views/filer/widget/zip-modal.vue'

@Component({
    components: {
        ChmodModal, CreateModal, DeleteModal, MkdirModal,
        ModifyModal, RenameModal, UnzipModal, UploadModal, ZipModal
    }
})
class FileExplorer extends Vue {
    @Inject({ from: APP_STATE_KEY }) readonly state!: AppState
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── Refs ───
    @Ref readonly modifyModalRef!: InstanceType<typeof ModifyModal>
    @Ref readonly renameModalRef!: InstanceType<typeof RenameModal>
    @Ref readonly chmodModalRef!: InstanceType<typeof ChmodModal>
    @Ref readonly zipModalRef!: InstanceType<typeof ZipModal>
    @Ref readonly deleteModalRef!: InstanceType<typeof DeleteModal>
    @Ref readonly unzipModalRef!: InstanceType<typeof UnzipModal>
    @Ref readonly mkdirModalRef!: InstanceType<typeof MkdirModal>
    @Ref readonly createModalRef!: InstanceType<typeof CreateModal>
    @Ref readonly uploadModal!: InstanceType<typeof UploadModal>

    // ─── 数据属性 ───
    files: FilerFileInfo[] = []
    formatFileSize = formatFileSize
    formatTime = formatTime
    getFileIcon = getFileIcon
    isEditableFile = isEditableFile

    // ─── 计算属性 ───
    get paths() {
        if (!this.state.currentPath || this.state.currentPath === '/') return []
        return this.state.currentPath.split('/').filter((part: string) => part)
    }

    // ─── 方法 ───
    navigateTo(path: string) {
        this.actions.loadFiles(path)
    }

    async download(file: FilerFileInfo) {
        const response = await api.download(file.path)
        downloadFile(file.name, response)
    }

    refreshFiles() {
        this.actions.loadFiles()
    }

    async loadFiles(path: string = this.state.currentPath) {
        const res = await api.list(path)
        this.files = res.payload?.files || []
        this.state.currentPath = res.payload?.path ?? '/'
    }

    // ─── 生命周期 ───
    mounted() {
        this.actions.loadFiles = (path?: string) => this.loadFiles(path)
        this.loadFiles('/')
    }
}

export default toNative(FileExplorer)
</script>

<template>
  <div>
    <!-- Toolbar Bar -->
    <div class="card mb-4">
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3">
        <div class="flex items-center justify-between gap-3">
          <nav aria-label="breadcrumb" class="flex-1 min-w-0">
            <ol class="flex items-center space-x-2 text-sm overflow-x-auto">
              <li class="flex-shrink-0">
                <a 
                  class="flex items-center px-3 py-1.5 rounded-lg text-slate-600 hover:bg-white hover:text-primary-600 transition-all"
                  href="#" 
                  @click="navigateTo('/')"
                >
                  <i class="fas fa-home text-base"></i>
                </a>
              </li>
              
              <template v-for="(part, index) in paths" :key="index">
                <li class="text-slate-300 flex-shrink-0">
                  <i class="fas fa-chevron-right text-xs"></i>
                </li>
                <li v-if="Number(index) < paths.length - 1" class="flex-shrink-0">
                  <a 
                    class="px-3 py-1.5 rounded-lg text-slate-600 hover:bg-white hover:text-primary-600 transition-all"
                    href="#" 
                    @click="navigateTo('/' + paths.slice(0, Number(index) + 1).join('/'))"
                  >
                    {{ part }}
                  </a>
                </li>
                <li v-else class="px-3 py-1.5 text-primary-600 font-semibold flex-shrink-0">
                  {{ part }}
                </li>
              </template>
            </ol>
          </nav>

          <div class="flex items-center gap-1 flex-shrink-0">
            <button 
              @click="refreshFiles()"
              class="hidden md:flex px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium items-center gap-1.5 transition-colors"
            >
              <i class="fas fa-rotate"></i><span>刷新</span>
            </button>
            <template v-if="actions.hasPerm('filer', true)">
              <button 
                class="hidden md:flex px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium items-center gap-1.5 transition-colors"
                @click="mkdirModalRef.show()"
              >
                <i class="fas fa-folder"></i><span>新建目录</span>
              </button>
              <button 
                class="hidden md:flex px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium items-center gap-1.5 transition-colors"
                @click="createModalRef.show()"
              >
                <i class="fas fa-file"></i><span>新建文件</span>
              </button>
              <button 
                class="hidden md:flex px-3 py-1.5 rounded-lg bg-primary-500 hover:bg-primary-600 text-white text-xs font-medium items-center gap-1.5 transition-colors"
                @click="uploadModal.show()"
              >
                <i class="fas fa-upload"></i><span>上传文件</span>
              </button>
            </template>
            <!-- 移动端图标按鈕 -->
            <button @click="refreshFiles()" class="md:hidden w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 flex items-center justify-center text-slate-600 transition-colors" title="刷新">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <template v-if="actions.hasPerm('filer', true)">
              <button @click="mkdirModalRef.show()" class="md:hidden w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 flex items-center justify-center text-slate-600 transition-colors" title="新建目录">
                <i class="fas fa-folder text-sm"></i>
              </button>
              <button @click="createModalRef.show()" class="md:hidden w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 flex items-center justify-center text-slate-600 transition-colors" title="新建文件">
                <i class="fas fa-file text-sm"></i>
              </button>
              <button @click="uploadModal.show()" class="md:hidden w-9 h-9 rounded-lg bg-primary-500 hover:bg-primary-600 flex items-center justify-center text-white transition-colors" title="上传文件">
                <i class="fas fa-upload text-sm"></i>
              </button>
            </template>
          </div>
        </div>
      </div>

      <!-- Loading State -->
      <div v-if="state.loading" class="flex flex-col items-center justify-center py-32">
        <div class="w-16 h-16 spinner mb-4"></div>
        <p class="text-slate-500 font-medium">加载中...</p>
      </div>

      <!-- File List -->
      <div v-else class="space-y-3">
        <!-- 桌面端表格视图 -->
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">名称</th>
                <th class="w-32 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">大小</th>
                <th class="w-32 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">权限</th>
                <th class="w-32 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">修改时间</th>
                <th class="w-40 px-4 py-3 text-right text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="file in files" :key="file.name" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3 max-w-[280px]">
                  <div class="flex items-center">
                    <div :class="[
                      'w-9 h-9 rounded-lg flex items-center justify-center mr-3',
                      file.isDir 
                        ? 'bg-amber-400' 
                        : 'bg-blue-400'
                    ]">
                      <i :class="getFileIcon(file)" class="text-white text-base"></i>
                    </div>
                    <div class="min-w-0">
                      <a 
                        v-if="file.isDir" 
                        href="#" 
                        @click="navigateTo(file.path)" 
                        class="font-medium text-slate-800 hover:text-primary-600 transition-colors truncate block"
                      >
                        {{ file.name }}
                      </a>
                      <span v-else class="font-medium text-slate-800 truncate block">{{ file.name }}</span>
                    </div>
                  </div>
                </td>
                
                <td class="px-4 py-3">
                  <span v-if="!file.isDir" class="text-sm text-slate-600">
                    {{ formatFileSize(file.size) }}
                  </span>
                  <span v-else class="text-slate-400">—</span>
                </td>
                
                <td class="px-4 py-3">
                  <code class="px-2 py-1 bg-slate-100 rounded text-xs text-slate-700 font-mono">
                    {{ file.mode }}
                  </code>
                </td>
                
                <td class="px-4 py-3">
                  <span class="text-sm text-slate-500 whitespace-nowrap">
                    {{ formatTime(file.modTime) }}
                  </span>
                </td>
                
                <td class="px-4 py-3">
                  <div class="flex justify-end items-center gap-0.5">
                    <!-- Directory Actions -->
                    <template v-if="file.isDir">
                      <button 
                        class="btn-icon text-primary-600 hover:bg-primary-50"
                        @click="navigateTo(file.path)" 
                        title="进入目录"
                      >
                        <i class="fas fa-folder-open text-xs"></i>
                      </button>
                      <template v-if="actions.hasPerm('filer', true)">
                        <button 
                          class="btn-icon text-amber-600 hover:bg-amber-50"
                          @click="zipModalRef.show(file)" 
                          title="压缩"
                        >
                          <i class="fas fa-file-zipper text-xs"></i>
                        </button>
                        <button 
                          class="btn-icon text-slate-600 hover:bg-slate-50"
                          @click="renameModalRef.show(file)" 
                          title="重命名"
                        >
                          <i class="fas fa-pen text-xs"></i>
                        </button>
                        <button 
                          class="btn-icon text-red-600 hover:bg-red-50"
                          @click="deleteModalRef.show(file)" 
                          title="删除"
                        >
                          <i class="fas fa-trash text-xs"></i>
                        </button>
                      </template>
                    </template>
                    
                    <!-- File Actions -->
                    <template v-else>
                      <button 
                        class="btn-icon text-slate-600 hover:bg-slate-50"
                        @click="download(file)" 
                        title="下载"
                      >
                        <i class="fas fa-download text-xs"></i>
                      </button>
                      <template v-if="actions.hasPerm('filer', true)">
                        <button 
                          v-if="isEditableFile(file.name)"
                          class="btn-icon text-violet-600 hover:bg-violet-50"
                          @click="modifyModalRef.show(file)" 
                          title="编辑"
                        >
                          <i class="fas fa-file-pen text-xs"></i>
                        </button>
                        <button 
                          class="btn-icon text-slate-600 hover:bg-slate-50"
                          @click="renameModalRef.show(file)" 
                          title="重命名"
                        >
                          <i class="fas fa-pen text-xs"></i>
                        </button>
                        <button 
                          class="btn-icon text-red-600 hover:bg-red-50"
                          @click="deleteModalRef.show(file)" 
                          title="删除"
                        >
                          <i class="fas fa-trash text-xs"></i>
                        </button>
                      </template>
                    </template>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 移动端卡片视图 -->
        <div class="md:hidden space-y-3 p-4">
          <div 
            v-for="file in files" 
            :key="file.name"
            class="rounded-xl border border-slate-200 bg-white p-4 transition-all hover:shadow-sm"
          >
            <!-- 顶部：文件信息和图标 -->
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-3 min-w-0 flex-1">
                <div :class="[
                  'w-10 h-10 rounded-lg flex items-center justify-center flex-shrink-0',
                  file.isDir 
                    ? 'bg-amber-400' 
                    : 'bg-blue-400'
                ]">
                  <i :class="getFileIcon(file)" class="text-white text-base"></i>
                </div>
                <div class="min-w-0">
                  <div class="flex items-center gap-2 min-w-0">
                    <a 
                      v-if="file.isDir" 
                      href="#" 
                      @click="navigateTo(file.path)" 
                      class="font-medium text-slate-800 hover:text-primary-600 transition-colors text-sm truncate"
                    >
                      {{ file.name }}
                    </a>
                    <span v-else class="font-medium text-slate-800 text-sm truncate">{{ file.name }}</span>
                  </div>
                  <div class="flex items-center gap-3 mt-1">
                    <span v-if="!file.isDir" class="text-xs text-slate-500">
                      {{ formatFileSize(file.size) }}
                    </span>
                    <span v-else class="text-xs text-slate-400">目录</span>
                    <span class="text-xs text-slate-500">{{ formatTime(file.modTime) }}</span>
                  </div>
                </div>
              </div>
            </div>
            
            <!-- 中间：权限信息 -->
            <div class="mb-3">
              <p class="text-xs text-slate-500 mb-1">权限</p>
              <code class="px-2 py-1 bg-slate-100 rounded text-xs text-slate-700 font-mono">
                {{ file.mode }}
              </code>
            </div>
            
            <!-- 底部：操作按钮 -->
            <div class="flex flex-wrap gap-1 pt-2 border-t border-slate-100">
              <!-- Directory Actions -->
              <template v-if="file.isDir">
                <button 
                  class="btn-icon text-primary-600 hover:bg-primary-50"
                  @click="navigateTo(file.path)" 
                  title="进入目录"
                >
                  <i class="fas fa-folder-open text-xs"></i>
                  <span class="text-xs ml-1 hidden xs:inline">进入</span>
                </button>
                <template v-if="actions.hasPerm('filer', true)">
                  <button 
                    class="btn-icon text-amber-600 hover:bg-amber-50"
                    @click="zipModalRef.show(file)" 
                    title="压缩"
                  >
                    <i class="fas fa-file-zipper text-xs"></i>
                    <span class="text-xs ml-1 hidden xs:inline">压缩</span>
                  </button>
                  <button 
                    class="btn-icon text-slate-600 hover:bg-slate-50"
                    @click="renameModalRef.show(file)" 
                    title="重命名"
                  >
                    <i class="fas fa-pen text-xs"></i>
                    <span class="text-xs ml-1 hidden xs:inline">重命名</span>
                  </button>
                  <button 
                    class="btn-icon text-red-600 hover:bg-red-50"
                    @click="deleteModalRef.show(file)" 
                    title="删除"
                  >
                    <i class="fas fa-trash text-xs"></i>
                    <span class="text-xs ml-1 hidden xs:inline">删除</span>
                  </button>
                </template>
              </template>

              <!-- File Actions -->
              <template v-else>
                <button 
                  class="btn-icon text-slate-600 hover:bg-slate-50"
                  @click="download(file)" 
                  title="下载"
                >
                  <i class="fas fa-download text-xs"></i>
                  <span class="text-xs ml-1 hidden xs:inline">下载</span>
                </button>
                <template v-if="actions.hasPerm('filer', true)">
                  <button 
                    v-if="isEditableFile(file.name)"
                    class="btn-icon text-violet-600 hover:bg-violet-50"
                    @click="modifyModalRef.show(file)" 
                    title="编辑"
                  >
                    <i class="fas fa-file-pen text-xs"></i>
                    <span class="text-xs ml-1 hidden xs:inline">编辑</span>
                  </button>
                  <button 
                    class="btn-icon text-slate-600 hover:bg-slate-50"
                    @click="renameModalRef.show(file)" 
                    title="重命名"
                  >
                    <i class="fas fa-pen text-xs"></i>
                    <span class="text-xs ml-1 hidden xs:inline">重命名</span>
                  </button>
                  <button 
                    class="btn-icon text-red-600 hover:bg-red-50"
                    @click="deleteModalRef.show(file)" 
                    title="删除"
                  >
                    <i class="fas fa-trash text-xs"></i>
                    <span class="text-xs ml-1 hidden xs:inline">删除</span>
                  </button>
                </template>
              </template>
            </div>
          </div>
        </div>
      </div>

      <!-- Empty State -->
      <div v-if="files.length === 0" class="flex flex-col items-center justify-center py-32">
        <div class="w-24 h-24 rounded-full bg-slate-100 flex items-center justify-center mb-6">
          <i class="fas fa-folder-open text-5xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium text-lg mb-2">此目录为空</p>
        <p class="text-sm text-slate-400">上传文件或创建新目录开始使用</p>
      </div>
    </div>

    <!-- Modals -->
    <ModifyModal ref="modifyModalRef" />
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
