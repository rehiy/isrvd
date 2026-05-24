<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { FilerFileInfo } from '@/service/types'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal }
})
class DeleteModal extends Vue {
    portal = usePortal()
    // ─── 数据属性 ───
    isOpen = false
    formData = { path: '', name: '' }

    // ─── 方法 ───
    show(file: FilerFileInfo) {
        this.formData.path = file.path
        this.formData.name = file.name
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.formData.path) return
        await api.filerDelete(this.formData.path)
        this.portal.loadFiles()
        this.isOpen = false
    }
}

export default toNative(DeleteModal)
</script>

<template>
  <BaseModal ref="modalRef" v-model="isOpen" title="确认删除" :loading="portal.filerLoading" @confirm="handleConfirm">
    <div class="text-center py-6">
      <div class="empty-state-icon bg-red-100 mx-auto">
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
      {{ portal.filerLoading ? '删除中...' : '确认删除' }}
    </template>
  </BaseModal>
</template>
