<script setup>
import { inject, ref } from 'vue'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const actions = inject(APP_ACTIONS_KEY)

const loading = ref(false)
const file = ref(null)
const modalRef = ref(null)

const show = (fileData) => {
  file.value = fileData
  loading.value = false
  modalRef.value.show()
}

const handleConfirm = async () => {
  if (!file.value) return

  loading.value = true

  try {
    await api.deleteFile(file.value.path)
    actions.loadFiles()
    modalRef.value.hide()
  } catch (error) {
  } finally {
    loading.value = false
  }
}

defineExpose({ show })
</script>

<template>
  <BaseModal ref="modalRef" id="deleteModal" title="确认删除" header-class="text-bg-danger" :loading="loading" @confirm="handleConfirm">
    <p>确定要删除 <strong>{{ file?.name }}</strong> 吗？</p>
    <p class="text-danger mb-0">此操作不可恢复！</p>
    <template #confirm-text>
      {{ loading ? '删除中...' : '删除' }}
    </template>
  </BaseModal>
</template>
