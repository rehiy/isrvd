<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import BaseModal from '@/component/modal.vue'

import type { ExplorerAdapter } from '../types'

@Component({
    expose: ['show'],
    emits: ['success'],
    components: { BaseModal },
})
class CreateModal extends Vue {
    isOpen = false
    loading = false
    adapter: ExplorerAdapter | null = null
    currentPath = '/'
    formData = { name: '', content: '' }

    show(adapter: ExplorerAdapter, currentPath: string) {
        this.adapter = adapter
        this.currentPath = currentPath
        this.formData = { name: '', content: '' }
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.formData.name.trim() || !this.adapter) return
        this.loading = true
        try {
            const path = this.currentPath.replace(/\/+$/, '') + '/' + this.formData.name.trim()
            await this.adapter.createFile!(path, this.formData.content)
            this.$emit('success')
            this.isOpen = false
        } finally {
            this.loading = false
        }
    }
}

export default toNative(CreateModal)
</script>

<template>
  <BaseModal
    v-model="isOpen"
    title="新建文件"
    :loading="loading"
    :confirm-disabled="!formData.name.trim()"
    @confirm="handleConfirm"
  >
    <form class="space-y-5" @submit.prevent="handleConfirm">
      <div>
        <label for="fmCreateName" class="form-label">文件名称</label>
        <div class="relative">
          <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
            <i class="fas fa-file text-slate-400"></i>
          </div>
          <input
            id="fmCreateName"
            v-model="formData.name"
            type="text"
            :disabled="loading"
            required
            autofocus
            class="input pl-11"
            placeholder="请输入文件名称"
          >
        </div>
      </div>
      <div>
        <label for="fmCreateContent" class="form-label">文件内容</label>
        <textarea
          id="fmCreateContent"
          v-model="formData.content"
          rows="10"
          :disabled="loading"
          class="input font-mono text-sm"
          placeholder="请输入文件内容..."
        ></textarea>
      </div>
    </form>
    <template #confirm-text>{{ loading ? '新建中...' : '新建文件' }}</template>
  </BaseModal>
</template>
