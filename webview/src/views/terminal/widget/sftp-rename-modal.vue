<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { SFTPFileInfo } from '@/service/types'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal }
})
class SftpRenameModal extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    isOpen = false
    loading = false
    hostId = ''
    formData = { oldPath: '', newName: '', file: null as SFTPFileInfo | null }

    // ─── 方法 ───
    show(hostId: string, file: SFTPFileInfo, basePath: string) {
        this.hostId = hostId
        this.formData.file = file
        this.formData.oldPath = basePath === '/' ? '/' + file.name : basePath + '/' + file.name
        this.formData.newName = file.name
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.formData.newName.trim() || !this.formData.file) return
        
        this.loading = true
        try {
            await api.sftpRename(this.hostId, { 
                oldPath: this.formData.oldPath, 
                newPath: this.formData.newName 
            })
            this.portal.showNotification('success', '重命名成功')
            this.isOpen = false
            // 触发父组件刷新
            this.$emit('success')
        } catch (e: unknown) {
            this.portal.showNotification('error', '重命名失败: ' + (e instanceof Error ? e.message : ''))
        } finally {
            this.loading = false
        }
    }
}

export default toNative(SftpRenameModal)
</script>

<template>
  <BaseModal ref="modalRef" v-model="isOpen" title="重命名" :loading="loading" :confirm-disabled="!formData.newName.trim()" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div>
        <label for="newName" class="form-label">
          新名称
        </label>
        <div class="relative">
          <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
            <i class="fas fa-spell-check text-slate-400"></i>
          </div>
          <input id="newName" v-model="formData.newName" type="text" :disabled="loading" required placeholder="请输入新名称" class="input pl-11">
        </div>
      </div>
    </form>

    <template #confirm-text>
      {{ loading ? '重命名中...' : '确认重命名' }}
    </template>
  </BaseModal>
</template>
