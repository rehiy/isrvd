<script setup>
import { inject, reactive, ref } from 'vue'

import { Codemirror } from 'vue-codemirror'
import { css } from '@codemirror/lang-css'
import { go } from '@codemirror/lang-go'
import { html } from '@codemirror/lang-html'
import { javascript } from '@codemirror/lang-javascript'
import { json } from '@codemirror/lang-json'
import { markdown } from '@codemirror/lang-markdown'
import { python } from '@codemirror/lang-python'
import { sql } from '@codemirror/lang-sql'
import { xml } from '@codemirror/lang-xml'
import { yaml } from '@codemirror/lang-yaml'

import api from '@/service/api.js'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const state = inject(APP_STATE_KEY)
const actions = inject(APP_ACTIONS_KEY)

const extensions = [
  css(), go(), html(), javascript(), json(), markdown(), python(), sql(), xml(), yaml()
]

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

defineExpose({ show })
</script>

<template>
  <BaseModal ref="modalRef" id="editModal" :title="'编辑文件: ' + formData.filename" size="modal-xl" :loading="state.loading" @confirm="handleConfirm">
    <Codemirror v-model="formData.content" :style="{ height: '60vh' }" :extensions="extensions" :disabled="state.loading" />
    <template #confirm-text>
      {{ state.loading ? '保存中...' : '保存' }}
    </template>
  </BaseModal>
</template>
