<script setup>
import { inject, ref } from 'vue'

import api from '@/services/api.js'
import { APP_ACTIONS_KEY } from '@/stores/state.js'
import { isEditableFile, getFileIcon, formatFileSize, formatTime } from '@/utils/utils.js'

import ModifyModal from '@/modals/modify.vue'
import RenameModal from '@/modals/rename.vue'
import ChmodModal from '@/modals/chmod.vue'
import DeleteModal from '@/modals/delete.vue'
import ZipModal from '@/modals/zip.vue'
import UnzipModal from '@/modals/unzip.vue'

const actions = inject(APP_ACTIONS_KEY)

const modifyModalRef = ref(null)
const renameModalRef = ref(null)
const chmodModalRef = ref(null)
const zipModalRef = ref(null)
const deleteModalRef = ref(null)
const unzipModalRef = ref(null)

const navigateTo = (path) => {
  actions.loadFiles(path)
}

const loading = ref(false)

const files = ref([])
const currentPath = ref('')

actions.loadFiles = async (path) => {
  loading.value = true
  try {
    const data = await api.getFiles(path)
    files.value = data.payload.files || []
    currentPath.value = data.payload.path
  } catch (error) {
  } finally {
    loading.value = false
  }
}

actions.loadFiles('/');

const downloadFile = async (file) => {
  try {
    const response = await api.downloadFile(file.path)
    const url = window.URL.createObjectURL(new Blob([response.data]))
    const link = document.createElement('a')
    link.href = url
    link.download = file.name
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
  } catch (error) {
    console.error('Download failed:', error)
  }
}
</script>

<template>
  <div>
    <div v-if="loading" class="text-center p-4">
      <i class="fas fa-spinner fa-spin fa-2x text-primary"></i>
      <p class="mt-3 text-muted">加载中...</p>
    </div>

    <div v-else class="table-responsive">
      <table class="table table-hover">
        <thead class="table-light">
          <tr>
            <th>名称</th>
            <th>大小</th>
            <th>权限</th>
            <th>修改时间</th>
            <th class="actions-column">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="file in files" :key="file.name">
            <td>
              <i :class="getFileIcon(file)" class="me-2"></i>
              <a v-if="file.isDir" href="#" @click="navigateTo(file.path)" class="text-decoration-none">
                {{ file.name }}
              </a>
              <span v-else>{{ file.name }}</span>
            </td>
            <td>
              <span v-if="!file.isDir" class="text-muted small">
                {{ formatFileSize(file.size) }}
              </span>
              <span v-else class="text-muted">-</span>
            </td>
            <td><code class="small">{{ file.mode }}</code></td>
            <td class="text-muted small text-nowrap">{{ formatTime(file.modTime) }}</td>
            <td class="text-end">
              <!-- 目录操作 -->
              <template v-if="file.isDir">
                <button class="btn btn-outline-primary btn-sm me-1" @click="navigateTo(file.path)" title="进入目录">
                  <i class="fas fa-folder-open"></i>
                </button>
                <button class="btn btn-outline-secondary btn-sm me-1" @click="zipModalRef.show(file)" title="打包目录">
                  <i class="fas fa-file-archive"></i>
                </button>
              </template>
              <!-- 文件操作 -->
              <template v-else>
                <button class="btn btn-outline-success btn-sm me-1" @click="downloadFile(file)" title="下载">
                  <i class="fas fa-download"></i>
                </button>
                <button v-if="isEditableFile(file)" class="btn btn-outline-info btn-sm me-1" @click="modifyModalRef.show(file)" title="编辑">
                  <i class="fas fa-edit"></i>
                </button>
                <button v-if="file.name.endsWith('.zip')" class="btn btn-outline-warning btn-sm me-1" @click="unzipModalRef.show(file)" title="解压">
                  <i class="fas fa-expand-arrows-alt"></i>
                </button>
              </template>
              <!-- 通用操作 -->
              <button class="btn btn-outline-dark btn-sm me-1" @click="renameModalRef.show(file)" title="重命名">
                <i class="fas fa-pen"></i>
              </button>
              <button class="btn btn-outline-secondary btn-sm me-1" @click="chmodModalRef.show(file)" title="权限">
                <i class="fas fa-key"></i>
              </button>
              <button class="btn btn-outline-danger btn-sm" @click="deleteModalRef.show(file)" title="删除">
                <i class="fas fa-trash"></i>
              </button>
            </td>
          </tr>
        </tbody>
      </table>

      <div v-if="files.length === 0" class="text-center text-muted py-4">
        <i class="fas fa-folder-open fa-3x mb-3"></i>
        <p>此目录为空</p>
      </div>
    </div>

    <!-- 模态框组件 -->
    <ModifyModal ref="modifyModalRef" />
    <RenameModal ref="renameModalRef" />
    <ChmodModal ref="chmodModalRef" />
    <DeleteModal ref="deleteModalRef" />
    <ZipModal ref="zipModalRef" />
    <UnzipModal ref="unzipModalRef" />
  </div>
</template>

<style scoped>
.actions-column {
  width: 220px;
  text-align: center;
}
</style>
