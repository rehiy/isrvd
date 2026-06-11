<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { FilerFileInfo } from '@/service/types'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    emits: ['success'],
    components: { BaseModal }
})
class DeleteModal extends Vue {
    // ─── 数据属性 ───
    isOpen = false
    loading = false
    files: FilerFileInfo[] = []

    // ─── 计算属性 ───
    get hasDir() {
        return this.files.some(file => file.isDir)
    }

    get targetText() {
        return this.files.length > 1 ? `选中的 ${this.files.length} 项` : this.files[0]?.name || ''
    }

    get previewText() {
        return this.files.map(file => file.name).join('、')
    }

    // ─── 方法 ───
    show(file: FilerFileInfo | FilerFileInfo[]) {
        this.files = Array.isArray(file) ? file : [file]
        this.isOpen = true
    }

    async handleConfirm() {
        if (this.files.length === 0) return
        this.loading = true
        try {
            for (let index = 0; index < this.files.length; index += 5) {
                const batch = this.files.slice(index, index + 5)
                await Promise.all(batch.map(file => api.filerDelete(file.path)))
            }
            this.$emit('success')
            this.isOpen = false
        } finally {
            this.loading = false
        }
    }
}

export default toNative(DeleteModal)
</script>

<template>
  <BaseModal ref="modalRef" v-model="isOpen" title="确认删除" :loading="loading" confirm-class="btn-danger" :confirm-disabled="files.length === 0" @confirm="handleConfirm">
    <div class="text-center py-6">
      <div class="empty-state-icon bg-red-100 mx-auto">
        <i class="fas fa-trash text-3xl text-red-500"></i>
      </div>
      <p class="text-lg text-slate-700 mb-2">
        确定要删除 <strong class="text-slate-900">{{ targetText }}</strong> 吗？
      </p>
      <p class="text-sm text-red-600 flex items-center justify-center">
        <i class="fas fa-exclamation-triangle mr-2"></i>
        <span v-if="hasDir">包含目录，目录及其内容将被删除；</span>
        此操作不可恢复！
      </p>
    </div>

    <p v-if="files.length > 1" class="detail-value-mono mb-4">
      {{ previewText }}
    </p>

    <template #confirm-text>
      {{ loading ? '删除中...' : '确认删除' }}
    </template>
  </BaseModal>
</template>
