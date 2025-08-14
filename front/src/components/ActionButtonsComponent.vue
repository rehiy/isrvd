<template>
  <div>
    <div class="mb-3 d-flex flex-wrap align-items-center gap-2">
      <button class="btn btn-success btn-sm" @click="showMkdirModal">
        <i class="fas fa-folder me-1"></i>新建目录
      </button>
      <button class="btn btn-primary btn-sm" @click="showNewFileModal">
        <i class="fas fa-file me-1"></i>新建文件
      </button>
      <button class="btn btn-info btn-sm" @click="showUploadModal">
        <i class="fas fa-upload me-1"></i>上传文件
      </button>
      <button class="btn btn-secondary btn-sm" @click="refreshFiles">
        <i class="fas fa-sync-alt me-1"></i>刷新
      </button>
      <button class="btn btn-dark btn-sm ms-auto" @click="showShellModal">
        <i class="fas fa-terminal me-1"></i>终端
      </button>
    </div>

    <!-- 模态框组件 -->
    <MkdirModal ref="mkdirModalRef" />
    <NewFileModal ref="newFileModalRef" />
    <UploadModal ref="uploadModalRef" />
    <ShellModal ref="shellModalRef" />
  </div>
</template>

<script>
import { defineComponent, inject, ref } from 'vue'
import { APP_ACTIONS_KEY } from '../helpers/state.js'
import MkdirModal from './modals/MkdirModalComponent.vue'
import NewFileModal from './modals/NewFileModalComponent.vue'
import UploadModal from './modals/UploadModalComponent.vue'
import ShellModal from './modals/ShellModalComponent.vue'

export default defineComponent({
  name: 'ActionButtons',
  components: {
    MkdirModal,
    NewFileModal,
    UploadModal,
    ShellModal
  },
  setup() {
    const actions = inject(APP_ACTIONS_KEY)

    const mkdirModalRef = ref(null)
    const newFileModalRef = ref(null)
    const uploadModalRef = ref(null)
    const shellModalRef = ref(null)

    const showMkdirModal = () => mkdirModalRef.value?.show()
    const showNewFileModal = () => newFileModalRef.value?.show()
    const showUploadModal = () => uploadModalRef.value?.show()
    const showShellModal = () => shellModalRef.value?.show()
    const refreshFiles = () => actions.loadFiles()

    return {
      mkdirModalRef,
      newFileModalRef,
      uploadModalRef,
      shellModalRef,
      showMkdirModal,
      showNewFileModal,
      showUploadModal,
      showShellModal,
      refreshFiles
    }
  }
})
</script>
