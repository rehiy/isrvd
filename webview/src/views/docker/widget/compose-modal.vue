<script setup>
import { inject, ref } from 'vue'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

import { Codemirror } from 'vue-codemirror'
import { yaml } from '@codemirror/lang-yaml'

import BaseModal from '@/component/modal.vue'

const actions = inject(APP_ACTIONS_KEY)

const emit = defineEmits(['success'])

const modalRef = ref(null)
const isOpen = ref(false)
const composeLoading = ref(false)
const composeContent = ref('')
const composeExtensions = [yaml()]

const show = () => {
  composeContent.value = ''
  isOpen.value = true
}

const handleConfirm = async () => {
  if (!composeContent.value.trim()) return
  composeLoading.value = true
  try {
    const res = await api.deployCompose(composeContent.value)
    const created = res.payload || []
    actions.showNotification('success', `Compose 部署成功，已创建 ${created.length} 个容器`)
    isOpen.value = false
    emit('success')
  } catch (e) {
    actions.showNotification('error', 'Compose 部署失败')
  }
  composeLoading.value = false
}

defineExpose({ show })
</script>

<template>
  <BaseModal
    ref="modalRef"
    v-model="isOpen"
    title="通过 Compose 创建容器"
    :loading="composeLoading"
    show-footer
    @confirm="handleConfirm"
  >
    <template #confirm-text>部署</template>
    <div class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">Compose 内容 <span class="text-red-500">*</span></label>
        <div class="rounded-xl overflow-hidden border border-slate-200">
          <Codemirror v-model="composeContent" :style="{ height: '50vh' }" :extensions="composeExtensions" :disabled="composeLoading" />
        </div>
        <p class="mt-1 text-xs text-slate-400">粘贴 docker-compose.yml 内容，将按照服务定义逐个创建容器</p>
      </div>
    </div>
  </BaseModal>
</template>
