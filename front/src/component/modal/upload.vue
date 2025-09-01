<script setup>
import { inject, reactive, ref, computed } from 'vue'

import api from '@/service/api.js'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const state = inject(APP_STATE_KEY)
const actions = inject(APP_ACTIONS_KEY)

const formData = reactive({
  uploadFile: null
})

const fileInput = ref(null)
const modalRef = ref(null)

const show = () => {
  formData.uploadFile = null
  modalRef.value.show()
}

const handleFileChange = (event) => {
  formData.uploadFile = event.target.files[0] || null
}

const handleConfirm = async () => {
  if (!formData.uploadFile) return

  const formDataToSend = new FormData()
  formDataToSend.append('file', formData.uploadFile)
  formDataToSend.append('path', state.currentPath)

  await api.upload(formDataToSend)
  actions.loadFiles()

  formData.uploadFile = null
  if (fileInput.value) {
    fileInput.value.value = ''
  }
  modalRef.value.hide()
}

const hasFile = computed(() => {
  return formData.uploadFile !== null
})

defineExpose({ show })
</script>

<template>
  <BaseModal ref="modalRef" id="uploadModal" title="上传文件" :loading="state.loading" :confirm-disabled="!hasFile" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div class="mb-3">
        <label for="uploadFile" class="form-label">选择文件</label>
        <input type="file" class="form-control" id="uploadFile" ref="fileInput" @change="handleFileChange" :disabled="state.loading" required>
      </div>
    </form>
    <template #confirm-text>
      {{ state.loading ? '上传中...' : '上传' }}
    </template>
  </BaseModal>
</template>
