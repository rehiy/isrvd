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
class ZipModal extends Vue {
    // ─── 数据属性 ───
    isOpen = false
    loading = false
    formData = { file: null as FilerFileInfo | null }

    // ─── 方法 ───
    show(file: FilerFileInfo) {
        this.formData.file = file
        this.isOpen = true
    }

    async handleConfirm() {
        this.loading = true
        try {
            await api.filerZip(this.formData.file?.path ?? '')
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
  <BaseModal ref="modalRef" v-model="isOpen" title="压缩确认" :loading="loading" :confirm-disabled="!formData.file" @confirm="handleConfirm">
    <div v-if="formData.file" class="text-center py-6">
      <div class="empty-state-icon bg-amber-400 mx-auto shadow-lg shadow-amber-500/30">
        <i class="fas fa-file-archive text-3xl text-white"></i>
      </div>
      <p class="text-lg text-slate-700 mb-2">
        确定要压缩 <strong class="text-slate-900">{{ formData.file.name }}</strong> 吗？
      </p>
      <p class="text-sm text-slate-500">压缩后的文件将保存在当前目录</p>
    </div>

    <template #confirm-text>
      {{ loading ? '压缩中...' : '开始压缩' }}
    </template>
  </BaseModal>
</template>
