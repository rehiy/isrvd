<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'

import BaseModal from '@/component/modal.vue'

import { usePortal } from '@/stores'

@Component({
    expose: ['show'],
    components: { BaseModal }
})
class CreateModal extends Vue {
    portal = usePortal()
    // ─── 数据属性 ───
    isOpen = false
    formData = { name: '', content: '' }

    // ─── 方法 ───
    show() {
        this.formData = { name: '', content: '' }
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.formData.name.trim()) return
        await api.filerCreate(this.portal.currentPath + '/' + this.formData.name, this.formData.content)
        this.portal.loadFiles()
        this.isOpen = false
    }
}

export default toNative(CreateModal)
</script>

<template>
  <BaseModal ref="modalRef" v-model="isOpen" title="新建文件" :loading="portal.filerLoading" :confirm-disabled="!formData.name.trim()" @confirm="handleConfirm">
    <form class="space-y-5" @submit.prevent="handleConfirm">
      <div>
        <label for="fileName" class="block text-sm font-medium text-slate-700 mb-2">
          文件名称
        </label>
        <div class="relative">
          <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
            <i class="fas fa-file text-slate-400"></i>
          </div>
          <input
            id="fileName"
            v-model="formData.name"
            type="text"
            :disabled="portal.filerLoading"
            required
            class="input pl-11"
            placeholder="请输入文件名称"
          >
        </div>
      </div>
      <div>
        <label for="fileContent" class="block text-sm font-medium text-slate-700 mb-2">
          文件内容
        </label>
        <textarea
          id="fileContent"
          v-model="formData.content"
          rows="10"
          :disabled="portal.filerLoading"
          class="input font-mono text-sm"
          placeholder="请输入文件内容..."
        ></textarea>
      </div>
    </form>

    <template #confirm-text>
      {{ portal.filerLoading ? '新建中...' : '新建文件' }}
    </template>
  </BaseModal>
</template>
