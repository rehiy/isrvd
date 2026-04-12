<script setup>
import { computed, inject, ref } from 'vue'

import { downloadFile, formatFileSize, formatTime, getFileIcon, isEditableFile } from '@/helper/utils.js'
import api from '@/service/api.js'
import { APP_ACTIONS_KEY, APP_STATE_KEY } from '@/store/state.js'

import ChmodModal from '@/component/file-manager/chmod.vue'
import CreateModal from '@/component/file-manager/create.vue'
import DeleteModal from '@/component/file-manager/delete.vue'
import MkdirModal from '@/component/file-manager/mkdir.vue'
import ModifyModal from '@/component/file-manager/modify.vue'
import RenameModal from '@/component/file-manager/rename.vue'
import UnzipModal from '@/component/file-manager/unzip.vue'
import UploadModal from '@/component/file-manager/upload.vue'
import ZipModal from '@/component/file-manager/zip.vue'

const state = inject(APP_STATE_KEY)
const actions = inject(APP_ACTIONS_KEY)

const modifyModalRef = ref(null)
const renameModalRef = ref(null)
const chmodModalRef = ref(null)
const zipModalRef = ref(null)
const deleteModalRef = ref(null)
const unzipModalRef = ref(null)
const mkdirModalRef = ref(null)
const createModalRef = ref(null)
const uploadModal = ref(null)

const navigateTo = (path) => {
  actions.loadFiles(path)
}

const files = ref([])

const download = async (file) => {
  const response = await api.download(file.path)
  downloadFile(file.name, response)
}

const refreshFiles = () => actions.loadFiles()

const paths = computed(() => {
  if (!state.currentPath || state.currentPath === '/') return []
  return state.currentPath.split('/').filter(part => part)
})

actions.loadFiles = async (path = state.currentPath) => {
  const data = await api.list(path)
  files.value = data.payload.files || []
  state.currentPath = data.payload.path
}

actions.loadFiles('/')
</script>

<template>
  <div>
    <!-- Toolbar Bar -->
    <div class="card mb-4">
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-6 py-3">
        <div class="flex items-center justify-between">
          <nav aria-label="breadcrumb" class="flex-1">
            <ol class="flex items-center space-x-2 text-sm">
              <li>
                <a 
                  class="flex items-center px-3 py-1.5 rounded-lg text-slate-600 hover:bg-white hover:text-primary-600 transition-all"
                  href="#" 
                  @click="navigateTo('/')"
                >
                  <i class="fas fa-home text-base"></i>
                </a>
              </li>
              
              <template v-for="(part, index) in paths" :key="index">
                <li class="text-slate-300">
                  <i class="fas fa-chevron-right text-xs"></i>
                </li>
                <li v-if="index < paths.length - 1">
                  <a 
                    class="px-3 py-1.5 rounded-lg text-slate-600 hover:bg-white hover:text-primary-600 transition-all"
                    href="#" 
                    @click="navigateTo('/' + paths.slice(0, index + 1).join('/'))"
                  >
                    {{ part }}
                  </a>
                </li>
                <li v-else class="px-3 py-1.5 text-primary-600 font-semibold">
                  {{ part }}
                </li>
              </template>
            </ol>
          </nav>

          <div class="flex items-center gap-2">
            <button 
              @click="refreshFiles()"
              class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors"
            >
              <i class="fas fa-rotate"></i>刷新
            </button>

            <button 
              class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors"
              @click="mkdirModalRef.show"
            >
              <i class="fas fa-folder"></i>新建目录
            </button>

            <button 
              class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors"
              @click="createModalRef.show"
            >
              <i class="fas fa-file"></i>新建文件
            </button>

            <button 
              class="px-3 py-1.5 rounded-lg bg-primary-500 hover:bg-primary-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors"
              @click="uploadModal.show"
            >
              <i class="fas fa-upload"></i>上传文件
            </button>
          </div>
        </div>
      </div>

      <!-- Loading State -->
      <div v-if="state.loading" class="flex flex-col items-center justify-center py-32">
        <div class="w-16 h-16 spinner mb-4"></div>
        <p class="text-slate-500 font-medium">加载中...</p>
      </div>

      <!-- File List -->
      <div v-else class="overflow-x-auto">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-50 border-b border-slate-200">
              <th class="w-1/3 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">名称</th>
              <th class="w-32 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">大小</th>
              <th class="w-32 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">权限</th>
              <th class="w-32 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">修改时间</th>
              <th class="w-40 px-4 py-3 text-center text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-slate-100">
            <tr v-for="file in files" :key="file.name" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3">
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
                      class="font-medium text-slate-800 hover:text-primary-600 transition-colors"
                    >
                      {{ file.name }}
                    </a>
                    <span v-else class="font-medium text-slate-800">{{ file.name }}</span>
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
                <div class="flex justify-center items-center gap-0.5">
                  <!-- Directory Actions -->
                  <template v-if="file.isDir">
                    <button 
                      class="btn-icon text-primary-600 hover:bg-primary-50"
                      @click="navigateTo(file.path)" 
                      title="进入目录"
                    >
                      <i class="fas fa-folder-open"></i>
                    </button>
                    <button 
                      class="btn-icon text-slate-600 hover:bg-slate-100"
                      @click="zipModalRef.show(file)" 
                      title="打包目录"
                    >
                      <i class="fas fa-file-archive"></i>
                    </button>
                  </template>

                  <!-- File Actions -->
                  <template v-else>
                    <button 
                      class="btn-icon text-emerald-600 hover:bg-emerald-50"
                      @click="download(file)" 
                      title="下载"
                    >
                      <i class="fas fa-download"></i>
                    </button>
                    <button 
                      v-if="isEditableFile(file)" 
                      class="btn-icon text-cyan-600 hover:bg-cyan-50"
                      @click="modifyModalRef.show(file)" 
                      title="编辑"
                    >
                      <i class="fas fa-edit"></i>
                    </button>
                    <button 
                      v-if="file.name.endsWith('.zip')" 
                      class="btn-icon text-amber-600 hover:bg-amber-50"
                      @click="unzipModalRef.show(file)" 
                      title="解压"
                    >
                      <i class="fas fa-expand-arrows-alt"></i>
                    </button>
                  </template>

                  <!-- Common Actions -->
                  <button 
                    class="btn-icon text-slate-600 hover:bg-slate-100"
                    @click="renameModalRef.show(file)" 
                    title="重命名"
                  >
                    <i class="fas fa-pen"></i>
                  </button>
                  <button 
                    class="btn-icon text-slate-600 hover:bg-slate-100"
                    @click="chmodModalRef.show(file)" 
                    title="权限"
                  >
                    <i class="fas fa-key"></i>
                  </button>
                  <button 
                    class="btn-icon text-red-600 hover:bg-red-50"
                    @click="deleteModalRef.show(file)" 
                    title="删除"
                  >
                    <i class="fas fa-trash"></i>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>

        <!-- Empty State -->
        <div v-if="files.length === 0" class="flex flex-col items-center justify-center py-32">
          <div class="w-24 h-24 rounded-full bg-slate-100 flex items-center justify-center mb-6">
            <i class="fas fa-folder-open text-5xl text-slate-300"></i>
          </div>
          <p class="text-slate-600 font-medium text-lg mb-2">此目录为空</p>
          <p class="text-sm text-slate-400">上传文件或创建新目录开始使用</p>
        </div>
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
