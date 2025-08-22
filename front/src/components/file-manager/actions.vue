<script setup>
import { inject, ref } from 'vue'

import { APP_ACTIONS_KEY } from '@/stores/state.js'

import MkdirModal from '@/modals/mkdir.vue'
import NewFileModal from '@/modals/new-file.vue'
import UploadModal from '@/modals/upload.vue'

const actions = inject(APP_ACTIONS_KEY)

const mkdirModalRef = ref(null)
const newFileModalRef = ref(null)
const uploadModalRef = ref(null)

const refreshFiles = () => actions.loadFiles()
</script>

<template>
  <div>
    <div class="mb-3 d-flex flex-wrap align-items-center gap-2">
      <button class="btn btn-success btn-sm" @click="mkdirModalRef.show">
        <i class="fas fa-folder me-1"></i> 新建目录
      </button>
      <button class="btn btn-primary btn-sm" @click="newFileModalRef.show">
        <i class="fas fa-file me-1"></i> 新建文件
      </button>
      <button class="btn btn-info btn-sm" @click="uploadModalRef.show">
        <i class="fas fa-upload me-1"></i> 上传文件
      </button>
      <button class="btn btn-secondary btn-sm" @click="refreshFiles">
        <i class="fas fa-sync-alt me-1"></i> 刷新
      </button>
    </div>

    <!-- 模态框组件 -->
    <MkdirModal ref="mkdirModalRef" />
    <NewFileModal ref="newFileModalRef" />
    <UploadModal ref="uploadModalRef" />
  </div>
</template>
