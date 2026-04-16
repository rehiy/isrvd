<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import { APP_ACTIONS_KEY } from '@/store/state'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class ImageTagModal extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: any

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    tagImage: any = null
    tagRepoTag = ''

    // ─── 方法 ───
    show(image: any) {
        this.tagImage = image
        this.tagRepoTag = ''
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.tagRepoTag.trim() || !this.tagImage) return
        this.modalLoading = true
        try {
            await api.imageTag(this.tagImage.id, this.tagRepoTag.trim())
            this.actions.showNotification('success', '镜像标签添加成功')
            this.isOpen = false
            this.$emit('success')
        } catch (e) {}
        this.modalLoading = false
    }
}

export default toNative(ImageTagModal)
</script>

<template>
  <BaseModal
    v-model="isOpen"
    title="添加镜像标签"
    :loading="modalLoading"
    show-footer
    @confirm="handleConfirm"
  >
    <template #confirm-text>确认添加</template>
    <div v-if="tagImage" class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">当前镜像</label>
        <div class="px-3 py-2 bg-slate-50 rounded-lg text-sm text-slate-500">{{ tagImage.repoTags[0] || tagImage.shortId }}</div>
      </div>
      <div v-if="tagImage.repoTags.length > 1">
        <label class="block text-sm font-medium text-slate-700 mb-2">已有标签</label>
        <div class="flex flex-wrap gap-1.5">
          <span v-for="tag in tagImage.repoTags" :key="tag" class="inline-flex items-center px-2 py-1 rounded-lg text-xs font-medium bg-blue-50 text-blue-700">
            {{ tag }}
          </span>
        </div>
      </div>
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">新标签</label>
        <input type="text" v-model="tagRepoTag" placeholder="例如: myimage:v1, registry.example.com/app:latest" class="input" @keyup.enter="handleConfirm" />
        <p class="mt-1 text-xs text-slate-400">格式: 仓库路径:标签，如 myapp:v1.0</p>
      </div>
    </div>
  </BaseModal>
</template>
