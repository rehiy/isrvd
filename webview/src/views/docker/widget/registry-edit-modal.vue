<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { DockerRegistryInfo, DockerRegistryUpsert } from '@/service/types'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class RegistryEditModal extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    // 编辑时保存原始 URL，用于后端定位
    originalUrl = ''
    formData: DockerRegistryUpsert = { name: '', url: '', username: '', password: '', description: '' }

    // ─── 计算属性 ───
    get isEdit() {
        return this.originalUrl !== ''
    }

    get title() {
        return this.isEdit ? this.$t('编辑镜像仓库') : this.$t('新建镜像仓库')
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
                await api.dockerRegistryUpdate(this.originalUrl, this.formData)
                this.portal.showNotification('success', this.$t('仓库更新成功'))
            } else {
                await api.dockerRegistryCreate(this.formData)
                this.portal.showNotification('success', this.$t('仓库添加成功'))
            }
            this.isOpen = false
            this.$emit('success')
        } catch {}
        this.modalLoading = false
    }
}

export default toNative(RegistryEditModal)
</script>

<template>
  <BaseModal v-model="isOpen" :title="title" :loading="modalLoading" confirm-class="btn-purple" show-footer @confirm="handleConfirm">
    <form class="space-y-4" @submit.prevent="handleConfirm">
      <div>
        <label class="form-label">{{ $t('名称') }} <span class="text-red-500">*</span></label>
        <input v-model="formData.name" type="text" :placeholder="$t('请输入仓库名称')" required class="input" />
      </div>
      <div>
        <label class="form-label">{{ $t('仓库地址') }} <span class="text-red-500">*</span></label>
        <input v-model="formData.url" type="text" :placeholder="$t('请输入仓库地址')" required class="input" />
      </div>
      <div>
        <label class="form-label">{{ $t('描述') }} <span class="text-slate-400 font-normal">{{ $t('(可选)') }}</span></label>
        <input v-model="formData.description" type="text" :placeholder="$t('请输入仓库描述')" class="input" />
      </div>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
        <div>
          <label class="form-label">{{ $t('用户名') }} <span class="text-slate-400 font-normal">{{ $t('(可选)') }}</span></label>
          <input v-model="formData.username" type="text" :placeholder="$t('请输入用户名')" class="input" autocomplete="off" />
        </div>
        <div>
          <label class="form-label">{{ $t('密码') }} <span class="text-slate-400 font-normal">{{ $t('(可选)') }}</span></label>
          <input v-model="formData.password" type="password" :placeholder="isEdit ? $t('留空则保持不变') : $t('请输入仓库密码')" class="input" autocomplete="new-password" />
        </div>
      </div>
    </form>

    <template #confirm-text>{{ $t('确认添加') }}</template>
  </BaseModal>
</template>
