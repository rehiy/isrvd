<script setup>
import { inject, reactive, ref } from 'vue'

import api from '@/service/api.js'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const state = inject(APP_STATE_KEY)
const actions = inject(APP_ACTIONS_KEY)

const formData = reactive({
  name: '',
  content: ''
})

const modalRef = ref(null)

const show = () => {
  formData.name = ''
  formData.content = ''
  modalRef.value.show()
}

const handleConfirm = async () => {
  if (!formData.name.trim()) return
  await api.create(state.currentPath + '/' + formData.name, formData.content)
  actions.loadFiles()
  modalRef.value.hide()
}

defineExpose({ show })
</script>

<template>
  <BaseModal ref="modalRef" id="newFileModal" title="新建文件" size="modal-lg" :loading="state.loading" :confirm-disabled="!formData.name.trim()" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div class="mb-3">
        <label for="fileName" class="form-label">文件名称</label>
        <input type="text" class="form-control" id="fileName" v-model="formData.name" :disabled="state.loading" required>
      </div>
      <div class="mb-3">
        <label for="fileContent" class="form-label">文件内容</label>
        <textarea class="form-control" id="fileContent" rows="10" v-model="formData.content" :disabled="state.loading"></textarea>
      </div>
    </form>
    <template #confirm-text>
      {{ state.loading ? '创建中...' : '创建' }}
    </template>
  </BaseModal>
</template>
