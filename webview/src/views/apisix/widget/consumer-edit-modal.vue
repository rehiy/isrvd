<script setup>
import { inject, ref } from 'vue'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const actions = inject(APP_ACTIONS_KEY)

const emit = defineEmits(['success'])

const isOpen = ref(false)
const modalLoading = ref(false)
const isEditMode = ref(false)
const formData = ref({ username: '', desc: '' })

const show = (consumer = null) => {
  if (consumer) {
    isEditMode.value = true
    formData.value = { username: consumer.username, desc: consumer.desc || '' }
  } else {
    isEditMode.value = false
    formData.value = { username: '', desc: '' }
  }
  isOpen.value = true
}

const handleConfirm = async () => {
  if (!formData.value.username) {
    actions.showNotification('error', '用户名不能为空')
    return
  }
  modalLoading.value = true
  try {
    if (isEditMode.value) {
      await api.apisixUpdateConsumer(formData.value.username, { desc: formData.value.desc })
    } else {
      await api.apisixCreateConsumer(formData.value)
    }
    isOpen.value = false
    emit('success')
  } catch (e) {
    actions.showNotification('error', e.message || '操作失败')
  }
  modalLoading.value = false
}

defineExpose({ show })
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
