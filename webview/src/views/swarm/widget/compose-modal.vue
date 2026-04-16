<script setup>
import { ref } from 'vue'

import api from '@/service/api.js'
import { Codemirror } from 'vue-codemirror'
import { yaml } from '@codemirror/lang-yaml'

import BaseModal from '@/component/modal.vue'

const emit = defineEmits(['success'])

const isOpen = ref(false)
const loading = ref(false)
const content = ref('')
const extensions = [yaml()]

const show = () => {
  content.value = ''
  isOpen.value = true
}

const handleConfirm = async () => {
  if (!content.value.trim()) return
  loading.value = true
  try {
    const res = await api.swarmDeployComposeService(content.value)
    const created = res.payload || []
    isOpen.value = false
    emit('success', created.length)
  } catch (e) {}
  loading.value = false
}

defineExpose({ show })
</script>

<template>
  <BaseModal
    v-model="isOpen"
    title="通过 Compose 创建服务"
    :loading="loading"
    show-footer
    @confirm="handleConfirm"
  >
    <template #confirm-text>部署</template>
    <div class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">Compose 内容 <span class="text-red-500">*</span></label>
        <div class="rounded-xl overflow-hidden border border-slate-200">
          <Codemirror v-model="content" :style="{ height: '50vh' }" :extensions="extensions" :disabled="loading" />
        </div>
        <p class="mt-1 text-xs text-slate-400">粘贴 docker-compose.yml 内容，将按照服务定义逐个创建 Swarm 服务</p>
      </div>
    </div>
  </BaseModal>
</template>
