<script setup>
import { ref } from 'vue'

import api from '@/service/api.js'
import BaseModal from '@/component/modal.vue'

const emit = defineEmits(['success'])

const isOpen = ref(false)
const loading = ref(false)
const service = ref(null)
const replicas = ref(1)

const show = (svc) => {
  service.value = svc
  replicas.value = svc.replicas ?? 1
  isOpen.value = true
}

const handleConfirm = async () => {
  if (!service.value) return
  loading.value = true
  try {
    await api.swarmServiceAction(service.value.id, 'scale', replicas.value)
    isOpen.value = false
    emit('success')
  } catch (e) {}
  loading.value = false
}

defineExpose({ show })
</script>

<template>
  <BaseModal v-model="isOpen" title="服务扩缩容" :loading="loading" show-footer @confirm="handleConfirm">
    <template #confirm-text>确认扩缩容</template>
    <div v-if="service" class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">服务</label>
        <div class="px-3 py-2 bg-slate-50 rounded-lg text-sm text-slate-600">{{ service.name }}</div>
      </div>
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">目标副本数</label>
        <input type="number" v-model.number="replicas" min="0" max="100" class="input" />
        <p class="mt-1 text-xs text-slate-400">当前运行中副本：{{ service.runningTasks }} / {{ service.replicas }}</p>
      </div>
    </div>
  </BaseModal>
</template>
