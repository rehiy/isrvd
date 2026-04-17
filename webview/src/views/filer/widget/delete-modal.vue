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
class DeleteModal extends Vue {
    @Inject({ from: APP_STATE_KEY }) readonly state!: AppState
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    isOpen = false
    formData = { path: '', name: '' }

    // ─── 方法 ───
    show(file: FileInfo) {
        this.formData.path = file.path
        this.formData.name = file.name
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.formData.path) return
        await api.delete(this.formData.path)
        this.actions.loadFiles()
        this.isOpen = false
    }
}

export default toNative(DeleteModal)
</script>

<template>
  <BaseModal ref="modalRef" v-model="isOpen" title="确认删除" :loading="state.loading" @confirm="handleConfirm">
    <div class="text-center py-6">
      <div class="w-20 h-20 rounded-full bg-red-100 flex items-center justify-center mx-auto mb-4">
        <i class="fas fa-trash text-3xl text-red-500"></i>
      </div>
      <p class="text-lg text-slate-700 mb-2">
        确定要删除 <strong class="text-slate-900">{{ formData.name }}</strong> 吗？
      </p>
      <p class="text-sm text-red-600 flex items-center justify-center">
        <i class="fas fa-exclamation-triangle mr-2"></i>
        此操作不可恢复！
      </p>
    </div>
    <template #confirm-text>
      <i class="fas fa-trash mr-2" v-if="!state.loading"></i>
      {{ state.loading ? '删除中...' : '确认删除' }}
    </template>
  </BaseModal>
</template>
