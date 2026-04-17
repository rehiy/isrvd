<script lang="ts">
import { Component, Inject, Ref, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions, AppState } from '@/store/state'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal }
})
class UploadModal extends Vue {
    @Inject({ from: APP_STATE_KEY }) readonly state!: AppState
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── Refs ───
    @Ref readonly fileInput!: HTMLInputElement

    // ─── 数据属性 ───
    isOpen = false
    uploadFile: File | null = null

    // ─── 计算属性 ───
    get hasFile() { return this.uploadFile !== null }

    // ─── 方法 ───
    show() {
        this.uploadFile = null
        this.isOpen = true
    }

    handleFileChange(event: Event) {
        const target = event.target as HTMLInputElement
        this.uploadFile = target.files?.[0] || null
    }

    async handleConfirm() {
        if (!this.uploadFile) return
        const formDataToSend = new FormData()
        formDataToSend.append('file', this.uploadFile)
        formDataToSend.append('path', this.state.currentPath)
        await api.upload(formDataToSend)
        this.actions.loadFiles()
        this.uploadFile = null
        if (this.fileInput) this.fileInput.value = ''
        this.isOpen = false
    }
}

export default toNative(UploadModal)
</script>

<template>
  <BaseModal ref="modalRef" v-model="isOpen" title="上传文件" :loading="state.loading" :confirm-disabled="!hasFile" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div>
        <label for="uploadFile" class="block text-sm font-medium text-slate-700 mb-2">
          选择文件
        </label>
        <div class="relative">
          <input 
            type="file" 
            id="uploadFile" 
            ref="fileInput" 
            @change="handleFileChange" 
            :disabled="state.loading" 
            required
            class="input file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:text-sm file:font-medium file:bg-primary-50 file:text-primary-700 hover:file:bg-primary-100"
          >
        </div>
        <div v-if="uploadFile" class="mt-3 p-3 bg-primary-50 rounded-lg border border-primary-200">
          <div class="flex items-center">
            <div class="w-10 h-10 rounded-lg bg-primary-100 flex items-center justify-center mr-3">
              <i class="fas fa-file text-primary-600"></i>
            </div>
            <div>
              <p class="text-sm font-medium text-slate-700">{{ uploadFile.name }}</p>
              <p class="text-xs text-slate-500">{{ (uploadFile.size / 1024).toFixed(2) }} KB</p>
            </div>
          </div>
        </div>
      </div>
    </form>
    <template #confirm-text>
      <i class="fas fa-cloud-arrow-up mr-2" v-if="!state.loading"></i>
      {{ state.loading ? '上传中...' : '开始上传' }}
    </template>
  </BaseModal>
</template>
