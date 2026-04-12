<script setup>
import { inject, ref } from 'vue'

import { APP_ACTIONS_KEY } from '@/store/state.js'

import MkdirModal from '@/component/file-manager/mkdir.vue'
import CreateModal from '@/component/file-manager/create.vue'
import UploadModal from '@/component/file-manager/upload.vue'

const actions = inject(APP_ACTIONS_KEY)

const mkdirModalRef = ref(null)
const createModalRef = ref(null)
const uploadModal = ref(null)

const refreshFiles = () => actions.loadFiles()
</script>

<template>
  <div class="flex items-center justify-between gap-3 mb-6">
    <div class="flex items-center gap-3">
      <button 
        class="btn-success"
        @click="mkdirModalRef.show"
      >
        <i class="fas fa-folder mr-2"></i>
        新建目录
      </button>

      <button 
        class="btn-primary"
        @click="createModalRef.show"
      >
        <i class="fas fa-file-circle-plus mr-2"></i>
        新建文件
      </button>

      <button 
        class="btn bg-cyan-500 text-white hover:bg-cyan-600 focus:ring-cyan-500"
        style="box-shadow: 0 4px 14px 0 rgba(6, 182, 212, 0.39)"
        @click="uploadModal.show"
      >
        <i class="fas fa-cloud-arrow-up mr-2"></i>
        上传文件
      </button>
    </div>

    <button 
      class="btn-secondary"
      @click="refreshFiles"
    >
      <i class="fas fa-rotate mr-2"></i>
      刷新
    </button>

    <!-- 模态框组件 -->
    <MkdirModal ref="mkdirModalRef" />
    <CreateModal ref="createModalRef" />
    <UploadModal ref="uploadModal" />
  </div>
</template>
