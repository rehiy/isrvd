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
class VolumeCreateModal extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    formData = { name: '', driver: 'local' }

    // ─── 方法 ───
    show() {
        this.formData = { name: '', driver: 'local' }
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.formData.name.trim()) return
        this.modalLoading = true
        try {
            await api.createVolume(this.formData)
            this.actions.showNotification('success', '数据卷创建成功')
            this.isOpen = false
            this.$emit('success')
        } catch (e) {}
        this.modalLoading = false
    }
}

export default toNative(VolumeCreateModal)
</script>

<template>
  <BaseModal
    v-model="isOpen"
    title="创建数据卷"
    :loading="modalLoading"
    show-footer
    @confirm="handleConfirm"
  >
    <template #confirm-text>确认创建</template>
    <form @submit.prevent="handleConfirm" class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">卷名称</label>
        <input type="text" v-model="formData.name" placeholder="例如: my-data" required class="input" />
      </div>
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">驱动类型</label>
        <select v-model="formData.driver" class="input">
          <option value="local">local (本地)</option>
        </select>
      </div>
    </form>
  </BaseModal>
</template>
