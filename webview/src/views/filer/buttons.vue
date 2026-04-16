<script lang="ts">
import { Component, Inject, Ref, Vue, toNative } from 'vue-facing-decorator'

import { APP_ACTIONS_KEY } from '@/store/state'

import CreateModal from '@/views/filer/widget/create-modal.vue'
import MkdirModal from '@/views/filer/widget/mkdir-modal.vue'
import UploadModal from '@/views/filer/widget/upload-modal.vue'

@Component({
    components: { CreateModal, MkdirModal, UploadModal }
})
class FilerButtons extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: any

    // ─── Refs ───
    @Ref readonly mkdirModalRef!: InstanceType<typeof MkdirModal>
    @Ref readonly createModalRef!: InstanceType<typeof CreateModal>
    @Ref readonly uploadModal!: InstanceType<typeof UploadModal>

    // ─── 方法 ───
    refreshFiles() {
        this.actions.loadFiles()
    }
}

export default toNative(FilerButtons)
</script>

<template>
  <div class="flex items-center justify-between gap-3 mb-6">
    <div class="flex items-center gap-3">
      <button 
        class="btn-success"
        @click="mkdirModalRef.show()"
      >
        <i class="fas fa-folder mr-2"></i>
        新建目录
      </button>

      <button 
        class="btn-primary"
        @click="createModalRef.show()"
      >
        <i class="fas fa-file-circle-plus mr-2"></i>
        新建文件
      </button>

      <button 
        class="btn-cyan"
        @click="uploadModal.show()"
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
