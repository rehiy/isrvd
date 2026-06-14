<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import BaseModal from '@/component/modal.vue'

import type { FileInfo, ExplorerAdapter } from '../types'

@Component({
    expose: ['show'],
    emits: ['success'],
    components: { BaseModal },
})
class UnzipModal extends Vue {
    isOpen = false
    loading = false
    adapter: ExplorerAdapter | null = null
    file: FileInfo | null = null
    targetDir = ''

    show(adapter: ExplorerAdapter, file: FileInfo) {
        this.adapter = adapter
        this.file = file
        this.targetDir = ''
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.file || !this.adapter) return
        this.loading = true
        try {
            await this.adapter.unzip!(this.file.path, this.targetDir.trim() || undefined)
            this.$emit('success')
            this.isOpen = false
        } finally {
            this.loading = false
        }
    }
}

export default toNative(UnzipModal)
</script>

<template>
  <BaseModal
    v-model="isOpen"
    title="解压确认"
    :loading="loading"
    :confirm-disabled="!file"
    @confirm="handleConfirm"
  >
    <div v-if="file" class="space-y-4">
      <div class="text-center py-6">
        <div class="empty-state-icon bg-amber-400 mx-auto shadow-lg shadow-amber-500/30">
          <i class="fas fa-expand-arrows-alt text-3xl text-white"></i>
        </div>
        <p class="text-lg text-slate-700 mb-2">
          确定要解压 <strong class="text-slate-900">{{ file.name }}</strong> 吗？
        </p>
        <p class="text-sm text-slate-500">目标目录留空时，文件将解压到当前目录</p>
      </div>
      <form class="max-w-3xl space-y-4" @submit.prevent="handleConfirm">
        <div>
          <label for="fmUnzipTarget" class="form-label">目标目录</label>
          <div class="relative">
            <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
              <i class="fas fa-folder text-slate-400"></i>
            </div>
            <input
              id="fmUnzipTarget"
              v-model="targetDir"
              type="text"
              :disabled="loading"
              class="input pl-11"
              placeholder="请输入目录名，如：output"
            >
          </div>
          <p class="text-xs text-slate-400 mt-1">只能输入目录名，不允许包含 / 等路径分隔符</p>
        </div>
      </form>
    </div>
    <template #confirm-text>{{ loading ? '解压中...' : '开始解压' }}</template>
  </BaseModal>
</template>
