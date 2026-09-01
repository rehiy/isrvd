<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import BaseModal from '@/component/modal.vue'

import type { FileInfo, ExplorerAdapter } from '../types'

@Component({
    expose: ['show'],
    emits: ['success'],
    components: { BaseModal },
})
class ZipModal extends Vue {
    isOpen = false
    loading = false
    adapter: ExplorerAdapter | null = null
    file: FileInfo | null = null

    show(adapter: ExplorerAdapter, file: FileInfo) {
        this.adapter = adapter
        this.file = file
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.file || !this.adapter) return
        this.loading = true
        try {
            await this.adapter.zip?.(this.file.path)
            this.$emit('success')
            this.isOpen = false
        } finally {
            this.loading = false
        }
    }
}

export default toNative(ZipModal)
</script>

<template>
  <BaseModal
    v-model="isOpen"
    title="压缩确认"
    :loading="loading"
    :confirm-disabled="!file"
    @confirm="handleConfirm"
  >
    <div v-if="file" class="text-center py-6">
      <div class="empty-state-icon bg-amber-400 mx-auto shadow-lg shadow-amber-500/30">
        <i class="fas fa-file-archive text-3xl text-white"></i>
      </div>
      <p class="text-lg text-slate-700 mb-2">
        确定要压缩 <strong class="text-slate-900">{{ file.name }}</strong> 吗？
      </p>
      <p class="text-sm text-slate-500">压缩后的文件将保存在当前目录</p>
    </div>
    <template #confirm-text>{{ loading ? '压缩中...' : '开始压缩' }}</template>
  </BaseModal>
</template>
