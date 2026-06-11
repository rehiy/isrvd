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
class RenameModal extends Vue {
    // ─── 数据属性 ───
    isOpen = false
    loading = false
    formData = { name: '', file: null as FilerFileInfo | null }

    // ─── 方法 ───
    show(file: FilerFileInfo) {
        this.formData.file = file
        this.formData.name = file.name
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.formData.name.trim() || !this.formData.file) return
        this.loading = true
        try {
            await api.filerRename(this.formData.file.path, this.formData.name)
            this.$emit('success')
            this.isOpen = false
        } finally {
            this.loading = false
        }
    }
}

export default toNative(RenameModal)
</script>

<template>
  <BaseModal ref="modalRef" v-model="isOpen" title="重命名" :loading="loading" :confirm-disabled="!formData.name.trim()" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div>
        <label for="target" class="form-label">
          新名称
        </label>
        <div class="relative">
          <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
            <i class="fas fa-spell-check text-slate-400"></i>
          </div>
          <input id="target" v-model="formData.name" type="text" :disabled="loading" required class="input pl-11" placeholder="请输入新名称">
        </div>
      </div>
    </form>

    <template #confirm-text>
      {{ loading ? '重命名中...' : '确认重命名' }}
    </template>
  </BaseModal>
</template>
