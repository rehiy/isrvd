<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'

import BaseModal from '@/component/modal.vue'

import { usePortal } from '@/stores'

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class NetworkCreateModal extends Vue {
    portal = usePortal()

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
        if (!this.formData.name?.trim()) return
        this.modalLoading = true
        try {
            await api.dockerNetworkCreate(this.formData)
            this.portal.showNotification('success', '网络创建成功')
            this.isOpen = false
            this.$emit('success')
        } catch {}
        this.modalLoading = false
    }}

export default toNative(NetworkCreateModal)
</script>

<template>
  <BaseModal v-model="isOpen" title="新建网络" :loading="modalLoading" confirm-class="btn-purple" show-footer @confirm="handleConfirm">
    <form class="space-y-4" @submit.prevent="handleConfirm">
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">网络名称</label>
        <input v-model="formData.name" type="text" placeholder="例如: my-network" required class="input" />
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
        <input v-model="formData.subnet" type="text" placeholder="例如: 172.20.0.0/16" class="input" />
      </div>
    </form>

    <template #confirm-text>确认新建</template>
  </BaseModal>
</template>
