<script lang="ts">
import { Component, Prop, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    emits: ['success'],
    components: { BaseModal }
})
class UploadModal extends Vue {
    @Prop({ required: true }) readonly currentPath!: string

    portal = usePortal()
    @Ref readonly fileInput!: HTMLInputElement

    isOpen = false
    uploadFiles: File[] = []
    uploadProgress = 0       // 已完成数量
    currentFileProgress = 0  // 当前文件上传百分比 0~100
    isUploading = false      // 是否正在上传

    // 拖拽状态
    dragOver = false
    dragCounter = 0

    // 文件大小限制从服务器配置获取
    // 总大小限制为单文件限制的 5 倍（防止一次上传过多内容）
    get maxFileSize() { return this.portal.maxUploadSize || 104857600 }
    get maxTotalSize() { return this.maxFileSize * 5 }

    get hasFiles() { return this.uploadFiles.length > 0 }
    get totalSize() { return this.uploadFiles.reduce((sum, f) => sum + f.size, 0) }

    // 文件状态：done | active | pending
    fileStatus(index: number) {
        if (!this.isUploading) return 'pending'
        return index < this.uploadProgress ? 'done' : index === this.uploadProgress ? 'active' : 'pending'
    }

    show() {
        this.uploadFiles = []
        this.uploadProgress = this.currentFileProgress = 0
        this.isUploading = false
        this.dragOver = false
        this.dragCounter = 0
        this.isOpen = true
    }

    // ─── 拖拽事件处理 ───
    onDragEnter(e: DragEvent) {
        e.preventDefault()
        this.dragCounter++
        this.dragOver = true
    }

    onDragOver(e: DragEvent) {
        e.preventDefault()
    }

    onDragLeave(e: DragEvent) {
        e.preventDefault()
        this.dragCounter = Math.max(0, this.dragCounter - 1)
        this.dragOver = this.dragCounter > 0
    }

    onDrop(e: DragEvent) {
        e.preventDefault()
        this.dragOver = false
        this.dragCounter = 0
        
        const files = e.dataTransfer?.files
        if (!files || files.length === 0) return
        
        this.processFiles(Array.from(files))
    }

    // ─── 点击选择文件 ───
    triggerUpload() {
        this.fileInput?.click()
    }

    // ─── 处理文件选择/拖拽 ───
    handleFileChange(event: Event) {
        const target = event.target as HTMLInputElement
        if (!target.files) return
        this.processFiles(Array.from(target.files))
        if (this.fileInput) this.fileInput.value = ''
    }

    // ─── 检查文件大小 ───
    checkFileSize(file: File): boolean {
        if (file.size > this.maxFileSize) {
            const maxSizeMB = (this.maxFileSize / 1024 / 1024).toFixed(0)
            this.portal.showNotification('error', `文件 "${file.name}" 大小超出限制（${maxSizeMB}MB）`)
            return false
        }
        return true
    }

    // ─── 处理文件列表 ───
    processFiles(files: File[]) {
        const validFiles = files.filter(file => this.checkFileSize(file))
        if (validFiles.length === 0) return
        
        // 检查总大小
        const totalSize = validFiles.reduce((sum, f) => sum + f.size, 0)
        if (totalSize > this.maxTotalSize) {
            const maxTotalSizeMB = (this.maxTotalSize / 1024 / 1024).toFixed(0)
            this.portal.showNotification('error', `文件总大小超出限制（${maxTotalSizeMB}MB）`)
            return
        }
        
        this.uploadFiles = validFiles
        this.uploadProgress = 0
        this.currentFileProgress = 0
    }

    // ─── 删除单个文件 ───
    removeFile(index: number) {
        if (this.isUploading) return
        this.uploadFiles.splice(index, 1)
        this.uploadProgress = 0
        this.currentFileProgress = 0
    }

    // ─── 确认上传 ───
    async handleConfirm() {
        if (!this.hasFiles) return
        this.isUploading = true
        this.uploadProgress = 0
        this.currentFileProgress = 0
        
        for (const file of this.uploadFiles) {
            const formData = new FormData()
            formData.append('file', file)
            formData.append('path', this.currentPath)
            try {
                await api.filerUpload(formData, {
                    onUploadProgress: (e) => {
                        this.currentFileProgress = e.total ? Math.round((e.loaded / e.total) * 100) : 0
                    }
                })
            } catch (e) {
                console.error('Upload failed:', e)
                this.portal.showNotification('error', `文件 "${file.name}" 上传失败`)
            }
            this.uploadProgress++
            this.currentFileProgress = 0
        }
        
        this.isUploading = false
        this.uploadFiles = []
        this.uploadProgress = 0
        this.isOpen = false
        this.$emit('success')
    }
}

export default toNative(UploadModal)
</script>

<template>
  <BaseModal ref="modalRef" v-model="isOpen" title="上传文件" :loading="isUploading" :confirm-disabled="!hasFiles || isUploading" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div
        class="relative"
        @dragenter="onDragEnter"
        @dragover="onDragOver"
        @dragleave="onDragLeave"
        @drop="onDrop"
      >
        <!-- 拖拽/点击上传区域 -->
        <div
          class="border-2 border-dashed rounded-lg p-8 text-center cursor-pointer transition-colors"
          :class="dragOver ? 'border-primary-400 bg-primary-50' : 'border-slate-300 hover:border-primary-400 hover:bg-slate-50'"
          @click="triggerUpload"
        >
          <input
            ref="fileInput"
            type="file"
            multiple
            required
            class="hidden"
            @change="handleFileChange"
          >
          <div class="flex flex-col items-center gap-3">
            <div class="w-12 h-12 rounded-full flex items-center justify-center bg-primary-50">
              <i class="fas fa-cloud-arrow-up text-2xl text-primary-500"></i>
            </div>
            <div>
              <p class="text-sm font-medium text-slate-700">
                拖拽文件到此处，或 <span class="text-primary-600">点击选择</span>
              </p>
              <p class="text-xs text-slate-400 mt-1">单个文件最大 {{ (portal.maxUploadSize / 1024 / 1024).toFixed(0) }}MB</p>
            </div>
          </div>
        </div>

        <!-- 拖拽遮罩 -->
        <div
          v-if="dragOver"
          class="absolute inset-0 z-10 flex flex-col items-center justify-center gap-2 bg-primary-50/90 border-2 border-dashed border-primary-400 rounded-lg pointer-events-none"
        >
          <i class="fas fa-cloud-arrow-up text-3xl text-primary-500"></i>
          <span class="text-sm text-primary-600 font-medium">松手上传文件</span>
        </div>
      </div>

      <!-- 文件列表 -->
      <div v-if="hasFiles" class="mt-4 space-y-2">
        <div class="flex items-center justify-between">
          <p class="text-xs font-medium text-slate-600">已选择 {{ uploadFiles.length }} 个文件</p>
          <p class="text-xs text-slate-400">{{ (totalSize / 1024).toFixed(2) }} KB</p>
        </div>
        <div class="max-h-60 overflow-y-auto space-y-2">
          <div
            v-for="(file, index) in uploadFiles" :key="index"
            class="p-3 rounded-lg border"
            :class="{
              'bg-green-50 border-green-200': fileStatus(index) === 'done',
              'bg-primary-50 border-primary-200': fileStatus(index) === 'active',
              'bg-slate-50 border-slate-200': fileStatus(index) === 'pending',
            }"
          >
            <div class="flex items-center">
              <div
                class="row-icon mr-3"
                :class="{
                  'bg-green-100': fileStatus(index) === 'done',
                  'bg-primary-100': fileStatus(index) === 'active',
                  'bg-slate-100': fileStatus(index) === 'pending',
                }"
              >
                <i
                  class="fas text-sm"
                  :class="{
                    'fa-check text-green-600': fileStatus(index) === 'done',
                    'fa-spinner fa-spin text-primary-600': fileStatus(index) === 'active',
                    'fa-file text-slate-400': fileStatus(index) === 'pending',
                  }"
                ></i>
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center justify-between gap-2">
                  <p class="text-sm font-medium text-slate-700 truncate">{{ file.name }}</p>
                  <div class="flex items-center gap-2 flex-shrink-0">
                    <span
                      v-if="fileStatus(index) !== 'pending'" class="text-xs"
                      :class="fileStatus(index) === 'done' ? 'text-green-600' : 'text-primary-600'"
                    >{{ fileStatus(index) === 'done' ? 100 : currentFileProgress }}%</span>
                    <!-- 删除按钮 -->
                    <button
                      v-if="!isUploading"
                      type="button"
                      class="text-slate-400 hover:text-red-500 transition-colors"
                      title="删除此文件"
                      @click="removeFile(index)"
                    >
                      <i class="fas fa-times text-xs"></i>
                    </button>
                  </div>
                </div>
                <p class="text-xs text-slate-500 mt-0.5">{{ (file.size / 1024).toFixed(2) }} KB</p>
                <div v-if="fileStatus(index) !== 'pending' && fileStatus(index) !== 'done'" class="mt-1.5 h-1 rounded-full bg-slate-200 overflow-hidden">
                  <div
                    class="h-full rounded-full transition-all duration-200 bg-primary-500"
                    :style="{ width: currentFileProgress + '%' }"
                  ></div>
                </div>
                <div v-if="fileStatus(index) === 'done'" class="mt-1.5 h-1 rounded-full bg-green-200 overflow-hidden">
                  <div class="h-full rounded-full bg-green-500" style="width: 100%"></div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </form>

    <template #confirm-text>
      {{ isUploading ? `上传中 ${uploadProgress}/${uploadFiles.length}...` : '开始上传' }}
    </template>
  </BaseModal>
</template>
