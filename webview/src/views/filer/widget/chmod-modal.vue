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
class ChmodModal extends Vue {
    @Inject({ from: APP_STATE_KEY }) readonly state!: AppState
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    isOpen = false
    formData = { path: '', mode: '' }

    // ─── 方法 ───
    show(file: FileInfo) {
        this.formData.path = file.path
        this.formData.mode = file.modeO
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.formData.mode.trim()) return
        await api.chmod(this.formData.path, this.formData.mode)
        this.actions.loadFiles()
        this.isOpen = false
    }
}

export default toNative(ChmodModal)
</script>

<template>
  <BaseModal ref="modalRef" v-model="isOpen" title="修改权限" :loading="state.loading" :confirm-disabled="!formData.mode.trim()" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div>
        <label for="fileMode" class="block text-sm font-medium text-slate-700 mb-2">
          权限 (八进制)
        </label>
        <div class="relative">
          <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
            <i class="fas fa-key text-slate-400"></i>
          </div>
          <input 
            type="text" 
            id="fileMode" 
            v-model="formData.mode" 
            :disabled="state.loading" 
            required 
            placeholder="755"
            class="input pl-11"
          >
        </div>
        <div class="mt-3 p-4 bg-slate-50 rounded-xl border border-slate-200">
          <p class="text-sm font-medium text-slate-700 mb-2">常用权限:</p>
          <div class="flex flex-wrap gap-2">
            <span class="badge-primary">755 - rwxr-xr-x</span>
            <span class="badge-primary">644 - rw-r--r--</span>
            <span class="badge-warning">777 - rwxrwxrwx</span>
          </div>
        </div>
      </div>
    </form>
    <template #confirm-text>
      <i class="fas fa-key mr-2" v-if="!state.loading"></i>
      {{ state.loading ? '修改中...' : '确认修改' }}
    </template>
  </BaseModal>
</template>
