<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { RegistryInfo } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class RegistryPullModal extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    registries: RegistryInfo[] = []
    pullForm = { image: '', registryUrl: '', namespace: '' }

    // ─── 方法 ───
    show(allRegistries: RegistryInfo[], registry: RegistryInfo | null = null) {
        this.registries = allRegistries
        this.pullForm = {
            image: '',
            registryUrl: registry ? registry.url : '',
            namespace: ''
        }
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.pullForm.image.trim() || !this.pullForm.registryUrl.trim()) return
        this.modalLoading = true
        try {
            await api.pullFromRegistry(this.pullForm.image, this.pullForm.registryUrl, this.pullForm.namespace.trim() || undefined)
            this.actions.showNotification('success', '镜像拉取成功')
            this.isOpen = false
            this.$emit('success')
        } catch (e) {}
        this.modalLoading = false
    }
}

export default toNative(RegistryPullModal)
</script>

<template>
  <BaseModal
    v-model="isOpen"
    title="从仓库拉取镜像"
    :loading="modalLoading"
    show-footer
    @confirm="handleConfirm"
  >
    <template #confirm-text>开始拉取</template>
    <form @submit.prevent="handleConfirm" class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">源仓库地址</label>
        <select v-model="pullForm.registryUrl" class="input" required>
          <option value="" disabled>请选择仓库</option>
          <option v-for="reg in registries" :key="reg.url" :value="reg.url">{{ reg.name }} ({{ reg.url }}){{ reg.description ? ' - ' + reg.description : '' }}</option>
        </select>
      </div>
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">命名空间 <span class="text-slate-400 font-normal">(可选)</span></label>
        <input type="text" v-model="pullForm.namespace" placeholder="例如: myteam" class="input" />
      </div>
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">镜像名称</label>
        <input type="text" v-model="pullForm.image" placeholder="输入镜像名称，如 myapp:latest" class="input" required />
        <p class="mt-1 text-xs text-slate-400">
          将拉取: {{ pullForm.registryUrl || 'registry' }}/{{ pullForm.namespace ? pullForm.namespace + '/' : '' }}{{ pullForm.image || 'image:tag' }}
        </p>
      </div>
    </form>
  </BaseModal>
</template>
