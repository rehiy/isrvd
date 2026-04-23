<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import api from '@/service/api'
import type { ApisixConsumer } from '@/service/types'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class ConsumerEditModal extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    isEditMode = false
    formData = { username: '', desc: '' }

    // ─── 方法 ───
    show(consumer: ApisixConsumer | null = null) {
        if (consumer) {
            this.isEditMode = true
            this.formData = { username: consumer.username, desc: consumer.desc || '' }
        } else {
            this.isEditMode = false
            this.formData = { username: '', desc: '' }
        }
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.formData.username) {
            this.actions.showNotification('error', '用户名不能为空')
            return
        }
        this.modalLoading = true
        try {
            if (this.isEditMode) {
                await api.apisixUpdateConsumer(this.formData.username, { desc: this.formData.desc })
            } else {
                await api.apisixCreateConsumer(this.formData)
            }
            this.isOpen = false
            this.$emit('success')
        } catch (e: unknown) {
            this.actions.showNotification('error', (e instanceof Error ? e.message : '') || '操作失败')
        }
        this.modalLoading = false
    }
}

export default toNative(ConsumerEditModal)
</script>

<template>
  <BaseModal
    v-model="isOpen"
    :title="isEditMode ? '编辑用户' : '创建用户'"
    :loading="modalLoading"
    show-footer
    @confirm="handleConfirm"
  >
    <template #confirm-text>{{ isEditMode ? '保存' : '创建' }}</template>
    <form @submit.prevent="handleConfirm" class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">用户名 <span class="text-red-500">*</span></label>
        <input
          v-model="formData.username"
          type="text"
          :disabled="isEditMode"
          class="input"
          :class="{ 'disabled:bg-slate-50 disabled:text-slate-500': isEditMode }"
          placeholder="输入用户名"
        />
      </div>
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">描述</label>
        <input v-model="formData.desc" type="text" class="input" placeholder="用户描述" />
      </div>
    </form>
  </BaseModal>
</template>
