<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { DockerRegistryInfo, DockerRegistryUpsertRequest } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class RegistryEditModal extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    // 编辑时保存原始 URL，用于后端定位
    originalUrl = ''
    formData: DockerRegistryUpsertRequest = { name: '', url: '', username: '', password: '', description: '' }

    // ─── 计算属性 ───
    get isEdit() {
        return this.originalUrl !== ''
    }

    get title() {
        return this.isEdit ? '编辑镜像仓库' : '添加镜像仓库'
    }

    // ─── 方法 ───
    show(registry: DockerRegistryInfo | null = null) {
        if (registry) {
            this.originalUrl = registry.url
            this.formData = {
                name: registry.name,
                url: registry.url,
                username: registry.username,
                password: '',
                description: registry.description
            }
        } else {
            this.originalUrl = ''
            this.formData = { name: '', url: '', username: '', password: '', description: '' }
        }
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.formData.name?.trim() || !this.formData.url?.trim()) return
        this.modalLoading = true
        try {
            if (this.isEdit) {
                await api.updateRegistry(this.originalUrl, this.formData)
                this.actions.showNotification('success', '仓库更新成功')
            } else {
                await api.createRegistry(this.formData)
                this.actions.showNotification('success', '仓库添加成功')
            }
            this.isOpen = false
            this.$emit('success')
        } catch (e) {}
        this.modalLoading = false
    }
}

export default toNative(RegistryEditModal)
</script>

<template>
  <BaseModal
    v-model="isOpen"
    :title="title"
    :loading="modalLoading"
    show-footer
    @confirm="handleConfirm"
  >
    <form @submit.prevent="handleConfirm" class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">名称 <span class="text-red-500">*</span></label>
        <input type="text" v-model="formData.name" placeholder="例如: 内部仓库" required class="input" />
      </div>
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">仓库地址 <span class="text-red-500">*</span></label>
        <input type="text" v-model="formData.url" placeholder="例如: registry.example.com" required class="input" />
      </div>
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">描述 <span class="text-slate-400 font-normal">(可选)</span></label>
        <input type="text" v-model="formData.description" placeholder="仓库简介" class="input" />
      </div>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">用户名 <span class="text-slate-400 font-normal">(可选)</span></label>
          <input type="text" v-model="formData.username" placeholder="用于推送/拉取认证" class="input" autocomplete="off" />
        </div>
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">密码 <span class="text-slate-400 font-normal">(可选)</span></label>
          <input type="password" v-model="formData.password" :placeholder="isEdit ? '留空则保持不变' : '仓库密码'" class="input" autocomplete="new-password" />
        </div>
      </div>
    </form>
  </BaseModal>
</template>
