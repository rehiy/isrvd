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
class NetworkCreateModal extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: any

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    formData = { name: '', driver: 'bridge', subnet: '' }

    // ─── 方法 ───
    show() {
        this.formData = { name: '', driver: 'bridge', subnet: '' }
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.formData.name.trim()) return
        this.modalLoading = true
        try {
            await api.createNetwork(this.formData)
            this.actions.showNotification('success', '网络创建成功')
            this.isOpen = false
            this.$emit('success')
        } catch (e) {}
        this.modalLoading = false
    }
}

export default toNative(NetworkCreateModal)
</script>

<template>
  <BaseModal
    v-model="isOpen"
    title="创建网络"
    :loading="modalLoading"
    show-footer
    @confirm="handleConfirm"
  >
    <template #confirm-text>确认创建</template>
    <form @submit.prevent="handleConfirm" class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">网络名称</label>
        <input type="text" v-model="formData.name" placeholder="例如: my-network" required class="input" />
      </div>
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">驱动类型</label>
        <select v-model="formData.driver" class="input">
          <option value="bridge">bridge (桥接)</option>
          <option value="host">host (主机)</option>
          <option value="overlay">overlay (覆盖)</option>
          <option value="macvlan">macvlan</option>
        </select>
      </div>
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">子网 CIDR（可选）</label>
        <input type="text" v-model="formData.subnet" placeholder="例如: 172.20.0.0/16" class="input" />
      </div>
    </form>
  </BaseModal>
</template>
