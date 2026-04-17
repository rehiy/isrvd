<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class ImageBuildModal extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

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
            await api.imageBuild(this.buildDockerfile, this.buildTag)
            this.actions.showNotification('success', '镜像构建成功')
            this.isOpen = false
            this.$emit('success')
        } catch (e) {}
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
    show-footer
    @confirm="handleConfirm"
  >
    <template #confirm-text>开始构建</template>
    <div class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">镜像标签</label>
        <input type="text" v-model="buildTag" placeholder="例如: myapp:v1, custom-image:latest" class="input" />
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
  </BaseModal>
</template>
