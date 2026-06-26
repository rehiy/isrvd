<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { DockerImageInfo } from '@/service/types'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class ImageTagModal extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    tagImage: DockerImageInfo | null = null
    tagRepoTag = ''

    // ─── 方法 ───
    show(image: DockerImageInfo) {
        this.tagImage = image
        this.tagRepoTag = ''
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.tagRepoTag.trim() || !this.tagImage) return
        this.modalLoading = true
        try {
            await api.dockerImageTag(this.tagImage.id, this.tagRepoTag.trim())
            this.portal.showNotification('success', '镜像标签添加成功')
            this.isOpen = false
            this.$emit('success')
        } catch {}
        this.modalLoading = false
    }
}

export default toNative(ImageTagModal)
</script>

<template>
  <BaseModal v-model="isOpen" title="新建镜像标签" :loading="modalLoading" confirm-class="btn-blue" show-footer @confirm="handleConfirm">
    <div v-if="tagImage" class="space-y-4">
      <div>
        <label class="form-label">当前镜像</label>
        <div class="detail-value text-slate-500">{{ tagImage.repoTags[0] || tagImage.shortId }}</div>
      </div>
      <div v-if="tagImage.repoTags.length > 1">
        <label class="form-label">已有标签</label>
        <div class="flex flex-wrap gap-1.5">
          <span v-for="tag in tagImage.repoTags" :key="tag" class="badge-sm bg-blue-50 text-blue-700">
            {{ tag }}
          </span>
        </div>
      </div>
      <div>
        <label class="form-label">新标签</label>
        <input v-model="tagRepoTag" type="text" class="input" placeholder="请输入镜像标签" />
        <p class="mt-1 text-xs text-slate-400">格式: 仓库路径:标签，如 myapp:v1.0</p>
      </div>
    </div>

    <template #confirm-text>确认新建</template>
  </BaseModal>
</template>
