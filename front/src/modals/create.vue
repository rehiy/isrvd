<script setup>
import { inject, reactive, ref } from 'vue'

import api from '@/services/api.js'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/stores/state.js'

import BaseModal from '@/components/modal.vue'

const state = inject(APP_STATE_KEY)
const actions = inject(APP_ACTIONS_KEY)

const formData = reactive({
  name: '',
  content: '',
  loading: false
})

const modalRef = ref(null)

const show = () => {
  formData.name = ''
  formData.content = ''
  formData.loading = false
  modalRef.value.show()
}

const handleConfirm = async () => {
  if (!formData.name.trim()) return

  formData.loading = true

  try {
    await api.createFile(state.currentPath, formData.name, formData.content)
    actions.loadFiles()
    modalRef.value.hide()
  } catch (error) {
  } finally {
    formData.loading = false
  }
}

defineExpose({ show })
</script>

<template>
  <BaseModal ref="modalRef" id="newFileModal" title="新建文件" size="modal-lg" :loading="formData.loading" :confirm-disabled="!formData.name.trim()" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div class="mb-3">
        <label for="fileName" class="form-label">文件名称</label>
        <input type="text" class="form-control" id="fileName" v-model="formData.name" :disabled="formData.loading" required>
      </div>
      <div class="mb-3">
        <label for="fileContent" class="form-label">文件内容</label>
        <textarea class="form-control" id="fileContent" rows="10" v-model="formData.content" :disabled="formData.loading"></textarea>
      </div>
    </form>
    <template #confirm-text>
      {{ formData.loading ? '创建中...' : '创建' }}
    </template>
  </BaseModal>
</template>
