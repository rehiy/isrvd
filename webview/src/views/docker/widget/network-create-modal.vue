<script setup>
import { ref } from 'vue'
import { inject } from 'vue'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const actions = inject(APP_ACTIONS_KEY)

const emit = defineEmits(['success'])

const isOpen = ref(false)
const modalLoading = ref(false)
const formData = ref({ name: '', driver: 'bridge', subnet: '' })

const show = () => {
  formData.value = { name: '', driver: 'bridge', subnet: '' }
  isOpen.value = true
}

const handleConfirm = async () => {
  if (!formData.value.name.trim()) return
  modalLoading.value = true
  try {
    await api.createNetwork(formData.value)
    actions.showNotification('success', '网络创建成功')
    isOpen.value = false
    emit('success')
  } catch (e) {}
  modalLoading.value = false
}

defineExpose({ show })
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
