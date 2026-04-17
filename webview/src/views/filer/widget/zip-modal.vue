<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { FileInfo } from '@/service/types'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions, AppState } from '@/store/state'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal }
})
class ZipModal extends Vue {
    @Inject({ from: APP_STATE_KEY }) readonly state!: AppState
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    isOpen = false
    formData = { file: null as FileInfo | null }

    // ─── 方法 ───
    show(file: FileInfo) {
        this.formData.file = file
        this.isOpen = true
    }

    async handleConfirm() {
        await api.zip(this.state.currentPath + '/' + this.formData.file.name)
        this.actions.loadFiles()
        this.isOpen = false
    }
}

export default toNative(ZipModal)
</script>

<template>
  <BaseModal ref="modalRef" v-model="isOpen" title="压缩确认" :loading="state.loading" :confirm-disabled="!formData.file" @confirm="handleConfirm">
    <div v-if="formData.file" class="text-center py-6">
      <div class="w-20 h-20 rounded-full bg-amber-400 flex items-center justify-center mx-auto mb-4 shadow-lg shadow-amber-500/30">
        <i class="fas fa-file-archive text-3xl text-white"></i>
      </div>
      <p class="text-lg text-slate-700 mb-2">
        确定要压缩 <strong class="text-slate-900">{{ formData.file.name }}</strong> 吗？
      </p>
      <p class="text-sm text-slate-500">压缩后的文件将保存在当前目录</p>
    </div>
    <template #confirm-text>
      <i class="fas fa-file-archive mr-2" v-if="!state.loading"></i>
      {{ state.loading ? '压缩中...' : '开始压缩' }}
    </template>
  </BaseModal>
</template>
