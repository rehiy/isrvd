<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'

import BaseModal from '@/component/modal.vue'

import { usePortal } from '@/stores'

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class ImageBuildModal extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    buildTag = ''
    buildDockerfile = 'FROM alpine:latest\nCMD ["echo", "Hello World"]'

    // ─── 方法 ───
    show() {
        this.buildTag = ''
        this.buildDockerfile = 'FROM alpine:latest\nCMD ["echo", "Hello World"]'
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.buildDockerfile.trim()) return
        this.modalLoading = true
        try {
            await api.dockerImageBuild(this.buildDockerfile, this.buildTag)
            this.portal.showNotification('success', '镜像构建成功')
            this.isOpen = false
            this.$emit('success')
        } catch {}
        this.modalLoading = false
    }
}

export default toNative(ImageBuildModal)
</script>

<template>
  <BaseModal
    v-model="isOpen"
    title="构建镜像"
    :loading="modalLoading"
    confirm-class="btn-blue"
    show-footer
    @confirm="handleConfirm"
  >
    <div class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">镜像标签</label>
        <input v-model="buildTag" type="text" placeholder="例如: myapp:v1, custom-image:latest" class="input" />
        <p class="mt-1 text-xs text-slate-400">留空则使用 custom:latest</p>
      </div>
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">Dockerfile</label>
        <textarea
          v-model="buildDockerfile"
          rows="14"
          class="input font-mono text-sm"
          placeholder="FROM alpine:latest&#10;RUN echo hello"
          spellcheck="false"
        ></textarea>
      </div>
    </div>

    <template #confirm-text>开始构建</template>
  </BaseModal>
</template>
