<script setup>
import { inject, reactive, ref, computed } from 'vue'

import api from '@/services/api.js'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/stores/state.js'

import BaseModal from '@/components/modal-base.vue'

const state = inject(APP_STATE_KEY)
const actions = inject(APP_ACTIONS_KEY)

const formData = reactive({
  loading: false,
  selectedFile: null
})

const fileInput = ref(null)
const modalRef = ref(null)

const show = () => {
  formData.loading = false
  formData.selectedFile = null
  modalRef.value.show()
}

const handleFileChange = (event) => {
  formData.selectedFile = event.target.files[0] || null
}

const handleConfirm = async () => {
  if (!formData.selectedFile) return

  formData.loading = true

  const formDataToSend = new FormData()
  formDataToSend.append('file', formData.selectedFile)
  formDataToSend.append('path', state.currentPath)

  try {
    await api.uploadFile(formDataToSend)
    actions.loadFiles()
    formData.selectedFile = null
    if (fileInput.value) {
      fileInput.value.value = ''
    }
    modalRef.value.hide()
  } catch (error) {
  } finally {
    formData.loading = false
  }
}

const hasFile = computed(() => {
  return formData.selectedFile !== null
})

defineExpose({ show })
</script>

<template>
  <BaseModal ref="modalRef" id="uploadModal" title="上传文件" :loading="formData.loading" :confirm-disabled="!hasFile" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div class="mb-3">
        <label for="uploadFile" class="form-label">选择文件</label>
        <input type="file" class="form-control" id="uploadFile" ref="fileInput" @change="handleFileChange" :disabled="formData.loading" required>
      </div>
    </form>
    <template #confirm-text>
      {{ formData.loading ? '上传中...' : '上传' }}
    </template>
  </BaseModal>
</template>
