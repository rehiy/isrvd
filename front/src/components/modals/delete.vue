<template>
  <BaseModal ref="modalRef" id="deleteModal" title="确认删除" header-class="text-bg-danger" :loading="loading" @confirm="handleConfirm">
    <p>确定要删除 <strong>{{ file?.name }}</strong> 吗？</p>
    <p class="text-danger mb-0">此操作不可恢复！</p>
    <template #confirm-text>
      {{ loading ? '删除中...' : '删除' }}
    </template>
  </BaseModal>
</template>

<script setup>
import { inject, ref } from 'vue'
import axios from 'axios'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/stores/state.js'
import BaseModal from '@/components/base/base-modal.vue'

const state = inject(APP_STATE_KEY)
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
    await axios.delete('/api/delete', {
      params: { file: file.value.path }
    })

    actions.showSuccess(file.value.isDir ? '目录删除成功' : '文件删除成功')
    actions.loadFiles()
    modalRef.value.hide()
  } catch (error) {
    actions.showError(error.response?.data?.error || '删除失败')
  } finally {
    loading.value = false
  }
}

defineExpose({ show })
</script>
