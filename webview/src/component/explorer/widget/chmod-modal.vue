<script lang="ts">
import type { FileInfo, ExplorerAdapter } from '../types'
import { Component, Vue, toNative } from 'vue-facing-decorator'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    emits: ['success'],
    components: { BaseModal },
})
class ChmodModal extends Vue {
    isOpen = false
    loading = false
    adapter: ExplorerAdapter | null = null
    formData = { path: '', mode: '' }

    show(adapter: ExplorerAdapter, file: FileInfo) {
        this.adapter = adapter
        this.formData.path = file.path
        this.formData.mode = file.modeOctal
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.formData.mode.trim() || !this.adapter) return
        if (!/^[0-7]{3,4}$/.test(this.formData.mode)) return
        this.loading = true
        try {
            await this.adapter.chmod(this.formData.path, this.formData.mode)
            this.$emit('success')
            this.isOpen = false
        } finally {
            this.loading = false
        }
    }
}

export default toNative(ChmodModal)
</script>

<template>
  <BaseModal v-model="isOpen" title="修改权限" :loading="loading" :confirm-disabled="!formData.mode.trim()" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div>
        <label for="fmFileMode" class="form-label">权限 (八进制)</label>
        <div class="relative">
          <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
            <i class="fas fa-key text-slate-400"></i>
          </div>
          <input
            id="fmFileMode"
            v-model="formData.mode"
            type="text"
            :disabled="loading"
            required
            placeholder="请输入文件权限"
            class="input pl-11"
          >
        </div>
        <p class="text-xs text-slate-400 mt-1">三位八进制数，例如：755、644</p>
        <div class="mt-3 p-4 bg-slate-50 rounded-xl border border-slate-200">
          <p class="text-sm font-medium text-slate-700 mb-2">常用权限:</p>
          <div class="flex flex-wrap gap-2">
            <span class="badge-primary cursor-pointer" @click="formData.mode = '755'">755 - rwxr-xr-x</span>
            <span class="badge-primary cursor-pointer" @click="formData.mode = '644'">644 - rw-r--r--</span>
            <span class="badge-warning cursor-pointer" @click="formData.mode = '777'">777 - rwxrwxrwx</span>
          </div>
        </div>
      </div>
    </form>
    <template #confirm-text>{{ loading ? '修改中...' : '确认修改' }}</template>
  </BaseModal>
</template>
