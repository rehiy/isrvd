<script lang="ts">
import type { FileInfo, ExplorerAdapter } from '../types'
import { Component, Vue, Watch, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import { getPreviewMimeType, getPreviewType } from '@/helper/utils'
import type { PreviewFileType } from '@/helper/utils'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal },
})
class PreviewModal extends Vue {
    portal = usePortal()

    isOpen = false
    filename = ''
    previewUrl = ''
    previewType: PreviewFileType = ''
    mimeType = ''
    loading = false
    error = ''

    @Watch('isOpen')
    onOpenChange(value: boolean) {
        if (!value) this.resetPreview()
    }

    show(adapter: ExplorerAdapter, file: FileInfo) {
        this.resetPreview()
        this.filename = file.name
        this.previewType = getPreviewType(file.name)
        this.mimeType = getPreviewMimeType(file.name)
        this.error = ''
        // PDF 不依赖 load 事件，直接不显示 loading
        this.loading = this.previewType !== 'pdf'
        this.previewUrl = adapter.previewUrl?.(file.path, this.portal.token || '', true) ?? ''
        this.isOpen = true
    }

    close() {
        this.isOpen = false
        this.resetPreview()
    }

    handleLoaded() { this.loading = false }
    handleError() { this.loading = false; this.error = '文件加载失败' }

    resetPreview() {
        this.previewUrl = ''
        this.previewType = ''
        this.mimeType = ''
        this.loading = false
        this.error = ''
    }
}

export default toNative(PreviewModal)
</script>

<template>
  <BaseModal
    v-model="isOpen"
    :title="'预览: ' + filename"
    :show-footer="false"
    max-width-class="max-w-5xl"
    :body-class="previewType === 'pdf' ? 'p-0 overflow-hidden' : 'px-6 py-6 overflow-y-auto'"
    @cancel="close"
  >
    <div v-if="loading" class="flex flex-col items-center gap-3 py-10">
      <div class="w-12 h-12 spinner"></div>
      <span class="text-sm text-slate-500">加载中...</span>
    </div>

    <div v-if="error" class="flex flex-col items-center gap-3 py-10">
      <div class="empty-state-icon bg-red-100 !mb-0">
        <i class="fas fa-circle-exclamation text-4xl text-red-400"></i>
      </div>
      <span class="text-sm text-red-500">{{ error }}</span>
    </div>

    <img
      v-if="previewUrl && previewType === 'image' && !error"
      v-show="!loading"
      :src="previewUrl"
      :alt="filename"
      class="max-h-[calc(100vh-12rem)] max-w-full object-contain rounded-lg shadow-sm select-none"
      draggable="false"
      @load="handleLoaded"
      @error="handleError"
    />

    <audio
      v-if="previewUrl && previewType === 'audio' && !error"
      v-show="!loading"
      class="w-full"
      controls
      preload="metadata"
      @loadedmetadata="handleLoaded"
      @error="handleError"
    >
      <source :src="previewUrl" :type="mimeType" />
    </audio>

    <video
      v-if="previewUrl && previewType === 'video' && !error"
      v-show="!loading"
      class="max-h-[calc(100vh-12rem)] max-w-full rounded-lg bg-slate-900 shadow-sm"
      controls
      preload="metadata"
      @loadedmetadata="handleLoaded"
      @error="handleError"
    >
      <source :src="previewUrl" :type="mimeType" />
    </video>

    <object
      v-else-if="previewUrl && previewType === 'pdf'"
      :data="previewUrl"
      type="application/pdf"
      class="w-full border-0 h-[calc(100vh-10rem)]"
    >
      <div class="flex flex-col items-center justify-center gap-3 py-20">
        <div class="empty-state-icon !mb-0">
          <i class="fas fa-file-pdf text-4xl text-slate-400"></i>
        </div>
        <span class="text-sm text-slate-500">浏览器不支持 PDF 预览</span>
        <a :href="previewUrl" target="_blank" class="text-sm text-blue-500 hover:underline">在新标签页打开</a>
      </div>
    </object>
  </BaseModal>
</template>
