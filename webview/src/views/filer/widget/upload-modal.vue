<script lang="ts">
import { Component, Inject, Ref, Vue, toNative } from 'vue-facing-decorator'

import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions, AppState } from '@/store/state'

import api from '@/service/api'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal }
})
class UploadModal extends Vue {
    @Inject({ from: APP_STATE_KEY }) readonly state!: AppState
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    @Ref readonly fileInput!: HTMLInputElement

    isOpen = false
    uploadFiles: File[] = []
    uploadProgress = 0       // 已完成数量
    currentFileProgress = 0  // 当前文件上传百分比 0~100

    get hasFiles() { return this.uploadFiles.length > 0 }
    get totalSize() { return this.uploadFiles.reduce((sum, f) => sum + f.size, 0) }

    // 文件状态：done | active | pending
    fileStatus(index: number) {
        if (!this.state.loading) return 'pending'
        if (index < this.uploadProgress) return 'done'
        if (index === this.uploadProgress) return 'active'
        return 'pending'
    }

    show() {
        this.uploadFiles = []
        this.uploadProgress = 0
        this.currentFileProgress = 0
        this.isOpen = true
    }

    handleFileChange(event: Event) {
        const target = event.target as HTMLInputElement
        this.uploadFiles = target.files ? Array.from(target.files) : []
        this.uploadProgress = 0
        this.currentFileProgress = 0
    }

    async handleConfirm() {
        if (!this.hasFiles) return
        this.uploadProgress = 0
        this.currentFileProgress = 0
        for (let i = 0; i < this.uploadFiles.length; i++) {
            const formData = new FormData()
            formData.append('file', this.uploadFiles[i])
            formData.append('path', this.state.currentPath)
            await api.upload(formData, {
                onUploadProgress: (e) => {
                    this.currentFileProgress = e.total ? Math.round((e.loaded / e.total) * 100) : 0
                }
            })
            this.uploadProgress++
            this.currentFileProgress = 0
        }
        this.actions.loadFiles()
        this.uploadFiles = []
        this.uploadProgress = 0
        if (this.fileInput) this.fileInput.value = ''
        this.isOpen = false
    }
}

export default toNative(UploadModal)
</script>

<template>
  <BaseModal ref="modalRef" v-model="isOpen" title="上传文件" :loading="state.loading" :confirm-disabled="!hasFiles" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div>
        <label for="uploadFile" class="block text-sm font-medium text-slate-700 mb-2">选择文件</label>
        <input
          type="file" id="uploadFile" ref="fileInput" multiple required
          @change="handleFileChange" :disabled="state.loading"
          class="input file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:text-sm file:font-medium file:bg-primary-50 file:text-primary-700 hover:file:bg-primary-100"
        >
        <div v-if="hasFiles" class="mt-3 space-y-2">
          <div
            v-for="(file, index) in uploadFiles" :key="index"
            class="p-3 rounded-lg border"
            :class="{
              'bg-green-50 border-green-200': fileStatus(index) === 'done',
              'bg-primary-50 border-primary-200': fileStatus(index) === 'active' || !state.loading,
              'bg-slate-50 border-slate-200': fileStatus(index) === 'pending' && state.loading,
            }"
          >
            <div class="flex items-center">
              <div class="w-8 h-8 rounded-lg flex items-center justify-center mr-3 flex-shrink-0"
                :class="{
                  'bg-green-100': fileStatus(index) === 'done',
                  'bg-primary-100': fileStatus(index) === 'active' || !state.loading,
                  'bg-slate-100': fileStatus(index) === 'pending' && state.loading,
                }"
              >
                <i class="fas text-sm"
                  :class="{
                    'fa-check text-green-600': fileStatus(index) === 'done',
                    'fa-spinner fa-spin text-primary-600': fileStatus(index) === 'active',
                    'fa-file text-primary-600': !state.loading,
                    'fa-file text-slate-400': fileStatus(index) === 'pending' && state.loading,
                  }"
                ></i>
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center justify-between gap-2">
                  <p class="text-sm font-medium text-slate-700 truncate">{{ file.name }}</p>
                  <span v-if="fileStatus(index) !== 'pending'" class="text-xs flex-shrink-0"
                    :class="fileStatus(index) === 'done' ? 'text-green-600' : 'text-primary-600'"
                  >{{ fileStatus(index) === 'done' ? 100 : currentFileProgress }}%</span>
                </div>
                <p class="text-xs text-slate-500 mt-0.5">{{ (file.size / 1024).toFixed(2) }} KB</p>
                <div v-if="fileStatus(index) !== 'pending'" class="mt-1.5 h-1 rounded-full bg-slate-200 overflow-hidden">
                  <div
                    class="h-full rounded-full transition-all duration-200"
                    :class="fileStatus(index) === 'done' ? 'bg-green-500' : 'bg-primary-500'"
                    :style="{ width: (fileStatus(index) === 'done' ? 100 : currentFileProgress) + '%' }"
                  ></div>
                </div>
              </div>
            </div>
          </div>
          <p v-if="uploadFiles.length > 1" class="text-xs text-slate-400 text-right">
            共 {{ uploadFiles.length }} 个文件，合计 {{ (totalSize / 1024).toFixed(2) }} KB
            <span v-if="state.loading">（{{ uploadProgress }}/{{ uploadFiles.length }}）</span>
          </p>
        </div>
      </div>
    </form>
    <template #confirm-text>
      {{ state.loading ? `上传中 ${uploadProgress}/${uploadFiles.length}...` : '开始上传' }}
    </template>
  </BaseModal>
</template>
