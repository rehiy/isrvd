<template>
  <div>
    <div v-if="state.loading" class="text-center p-4">
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
            <th style="width: 220px;">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="file in state.files" :key="file.name">
            <td>
              <i :class="getFileIcon(file)" class="file-icon me-2"></i>
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
              <!-- 目录操作 -->
              <template v-if="file.isDir">
                <button class="btn btn-outline-primary btn-sm me-1" @click="navigateTo(file.path)" title="进入目录">
                  <i class="fas fa-folder-open"></i>
                </button>
                <button class="btn btn-outline-secondary btn-sm me-1" @click="showZipModal(file)" title="打包目录">
                  <i class="fas fa-file-archive"></i>
                </button>
              </template>
              <!-- 文件操作 -->
              <template v-else>
                <button class="btn btn-outline-success btn-sm me-1" @click="downloadFile(file)" title="下载">
                  <i class="fas fa-download"></i>
                </button>
                <button v-if="isEditableFile(file)" class="btn btn-outline-info btn-sm me-1" @click="editFile(file)" title="编辑">
                  <i class="fas fa-edit"></i>
                </button>
                <button v-if="file.name.endsWith('.zip')" class="btn btn-outline-warning btn-sm me-1" @click="showUnzipModal(file)" title="解压">
                  <i class="fas fa-expand-arrows-alt"></i>
                </button>
              </template>
              <!-- 通用操作 -->
              <button class="btn btn-outline-dark btn-sm me-1" @click="showRenameModal(file)" title="重命名">
                <i class="fas fa-pen"></i>
              </button>
              <button class="btn btn-outline-secondary btn-sm me-1" @click="showChmodModal(file)" title="权限">
                <i class="fas fa-key"></i>
              </button>
              <button class="btn btn-outline-danger btn-sm" @click="showDeleteModal(file)" title="删除">
                <i class="fas fa-trash"></i>
              </button>
            </td>
          </tr>
        </tbody>
      </table>

      <div v-if="state.files.length === 0" class="text-center text-muted py-4">
        <i class="fas fa-folder-open fa-3x mb-3"></i>
        <p>此目录为空</p>
      </div>
    </div>

    <!-- 模态框组件 -->
    <EditModal ref="editModalRef" />
    <RenameModal ref="renameModalRef" />
    <ChmodModal ref="chmodModalRef" />
    <ZipModal ref="zipModalRef" />
    <DeleteModal ref="deleteModalRef" />
    <UnzipModal ref="unzipModalRef" />
  </div>
</template>

<script>
import { defineComponent, inject, ref } from 'vue'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '../helpers/state.js'
import { isEditableFile, getFileIcon, formatFileSize, formatTime } from '../helpers/utils.js'
import EditModal from './modals/edit.vue'
import RenameModal from './modals/rename.vue'
import ChmodModal from './modals/chmod.vue'
import ZipModal from './modals/zip.vue'
import DeleteModal from './modals/delete.vue'
import UnzipModal from './modals/unzip.vue'

export default defineComponent({
  name: 'FileIndex',
  components: {
    EditModal,
    RenameModal,
    ChmodModal,
    ZipModal,
    DeleteModal,
    UnzipModal
  },
  setup() {
    const state = inject(APP_STATE_KEY)
    const actions = inject(APP_ACTIONS_KEY)

    const editModalRef = ref(null)
    const renameModalRef = ref(null)
    const chmodModalRef = ref(null)
    const zipModalRef = ref(null)
    const deleteModalRef = ref(null)
    const unzipModalRef = ref(null)

    const navigateTo = (path) => {
      actions.loadFiles(path)
    }

    const downloadFile = (file) => {
      const url = `/api/download?file=${encodeURIComponent(file.path)}&token=${state.token}`
      window.open(url, '_blank')
    }

    const editFile = (file) => {
      if (editModalRef.value) {
        editModalRef.value.show(file)
      }
    }
    const showRenameModal = (file) => {
      if (renameModalRef.value) {
        renameModalRef.value.show(file)
      }
    }
    const showChmodModal = (file) => {
      if (chmodModalRef.value) {
        chmodModalRef.value.show(file)
      }
    }
    const showZipModal = (file) => {
      if (zipModalRef.value) {
        zipModalRef.value.show(file)
      }
    }
    const showDeleteModal = (file) => {
      if (deleteModalRef.value) {
        deleteModalRef.value.show(file)
      }
    }
    const showUnzipModal = (file) => {
      if (unzipModalRef.value) {
        unzipModalRef.value.show(file)
      }
    }

    return {
      state,
      editModalRef,
      renameModalRef,
      chmodModalRef,
      zipModalRef,
      deleteModalRef,
      unzipModalRef,
      isEditableFile,
      getFileIcon,
      formatFileSize,
      formatTime,
      navigateTo,
      downloadFile,
      editFile,
      showRenameModal,
      showChmodModal,
      showZipModal,
      showDeleteModal,
      showUnzipModal
    }
  }
})
</script>

<style scoped>
.file-icon {
  width: 16px;
}
</style>
