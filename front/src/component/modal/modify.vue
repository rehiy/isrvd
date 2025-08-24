<script setup>
import { inject, reactive, ref } from 'vue'

import api from '@/service/api.js'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const state = inject(APP_STATE_KEY)
const actions = inject(APP_ACTIONS_KEY)

const formData = reactive({
  filename: '',
  content: '',
  filePath: ''
})

const modalRef = ref(null)

const show = async (file) => {
  const data = await api.getFileContent(file.path)
  formData.filePath = file.path
  formData.filename = file.name
  formData.content = data.payload.content
  modalRef.value.show()
}

const handleConfirm = async () => {
  await api.saveFileContent(formData.filePath, formData.content)
  actions.loadFiles()
  modalRef.value.hide()
}

defineExpose({ show })
</script>

<template>
  <BaseModal ref="modalRef" id="editModal" :title="'编辑文件: ' + formData.filename" size="modal-xl" :loading="state.loading" @confirm="handleConfirm">
    <textarea class="form-control editor-textarea" rows="20" v-model="formData.content" :disabled="state.loading"></textarea>
    <template #confirm-text>
      {{ state.loading ? '保存中...' : '保存' }}
    </template>
  </BaseModal>
</template>

<style scoped>
.editor-textarea {
  font-family: 'Courier New', monospace;
}
</style>
