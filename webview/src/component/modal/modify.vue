<script setup>
import { inject, reactive, ref } from 'vue'
import { CodeEditor } from 'monaco-editor-vue3';

import api from '@/service/api.js'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const state = inject(APP_STATE_KEY)
const actions = inject(APP_ACTIONS_KEY)

const formData = reactive({
  filename: '',
  content: '',
  path: ''
})

const modalRef = ref(null)

const show = async (file) => {
  const data = await api.read(file.path)
  formData.path = file.path
  formData.filename = file.name
  formData.content = data.payload.content
  modalRef.value.show()
}

const handleConfirm = async () => {
  await api.modify(formData.path, formData.content)
  actions.loadFiles()
  modalRef.value.hide()
}

const editorOptions = {
  fontSize: 14,
  minimap: { enabled: false },
  automaticLayout: true
};

defineExpose({ show })
</script>

<template>
  <BaseModal ref="modalRef" id="editModal" :title="'编辑文件: ' + formData.filename" size="modal-xl" :loading="state.loading" @confirm="handleConfirm">
    <div class="editor-container">
      <CodeEditor language="javascript" theme="vs" v-model:value="formData.content" :options="editorOptions" :disabled="state.loading" />
    </div>
    <template #confirm-text>
      {{ state.loading ? '保存中...' : '保存' }}
    </template>
  </BaseModal>
</template>

<style scoped>
.editor-container {
  height: 60vh;
}
</style>
