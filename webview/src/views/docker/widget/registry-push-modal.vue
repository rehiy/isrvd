<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { RegistryInfo, ImageInfo } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class RegistryPushModal extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    registries: RegistryInfo[] = []
    localImages: ImageInfo[] = []
    pushForm = { image: '', registryUrl: '', namespace: '' }

    // ─── 计算属性 ───
    get imageTagOptions() {
        const tags: string[] = []
        for (const img of this.localImages) {
            for (const tag of img.repoTags) {
                if (tag !== '<none>:<none>') tags.push(tag)
            }
        }
        return tags
    }

    get pushTargetPreview() {
        const registry = this.pushForm.registryUrl || 'registry'
        const ns = this.pushForm.namespace ? this.pushForm.namespace + '/' : ''
        let imageName = this.pushForm.image || 'image:tag'
        const lastSlash = imageName.lastIndexOf('/')
        if (lastSlash >= 0) imageName = imageName.substring(lastSlash + 1)
        return registry + '/' + ns + imageName
    }

    // ─── 方法 ───
    async loadLocalImages() {
        try {
            const res = await api.listImages(false)
            this.localImages = (res.payload || []).filter((img: ImageInfo) => img.repoTags && img.repoTags.length > 0)
        } catch (e) {}
    }

    show(allRegistries: RegistryInfo[], registry: RegistryInfo | null = null) {
        this.registries = allRegistries
        this.pushForm = {
            image: '',
            registryUrl: registry ? registry.url : '',
            namespace: ''
        }
        this.loadLocalImages()
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.pushForm.image.trim() || !this.pushForm.registryUrl.trim()) return
        this.modalLoading = true
        try {
            await api.pushImage(this.pushForm.image, this.pushForm.registryUrl, this.pushForm.namespace.trim())
            this.actions.showNotification('success', '镜像推送成功')
            this.isOpen = false
            this.$emit('success')
        } catch (e) {}
        this.modalLoading = false
    }
}

export default toNative(RegistryPushModal)
</script>

<template>
  <BaseModal
    v-model="isOpen"
    title="推送镜像到仓库"
    :loading="modalLoading"
    show-footer
    @confirm="handleConfirm"
  >
    <template #confirm-text>开始推送</template>
    <form @submit.prevent="handleConfirm" class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">本地镜像</label>
        <select v-model="pushForm.image" class="input" required>
          <option value="" disabled>请选择镜像</option>
          <option v-for="tag in imageTagOptions" :key="tag" :value="tag">{{ tag }}</option>
        </select>
      </div>
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">目标仓库地址</label>
        <select v-model="pushForm.registryUrl" class="input" required>
          <option value="" disabled>请选择仓库</option>
          <option v-for="reg in registries" :key="reg.url" :value="reg.url">{{ reg.name }} ({{ reg.url }}){{ reg.description ? ' - ' + reg.description : '' }}</option>
        </select>
      </div>
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">命名空间 <span class="text-slate-400 font-normal">(可选)</span></label>
        <input type="text" v-model="pushForm.namespace" placeholder="例如: myteam" class="input" />
        <p class="mt-1 text-xs text-slate-400">镜像将被推送为: {{ pushTargetPreview }}</p>
      </div>
    </form>
  </BaseModal>
</template>
