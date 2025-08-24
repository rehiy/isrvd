<script setup>
import { inject, reactive, ref } from 'vue'

import api from '@/service/api.js'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const state = inject(APP_STATE_KEY)
const actions = inject(APP_ACTIONS_KEY)

const formData = reactive({
  name: '',
  file: null
})

const modalRef = ref(null)

const show = (file) => {
  formData.file = file
  formData.name = file.name
  modalRef.value.show()
}

const handleConfirm = async () => {
  if (!formData.name.trim() || !formData.file) return
  await api.renameFile(formData.file.path, formData.name)
  actions.loadFiles()
  modalRef.value.hide()
}

defineExpose({ show })
</script>

<template>
  <BaseModal ref="modalRef" id="renameModal" title="重命名" :loading="state.loading" :confirm-disabled="!formData.name.trim()" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div class="mb-3">
        <label for="newPath" class="form-label">新名称</label>
        <input type="text" class="form-control" id="newPath" v-model="formData.name" :disabled="state.loading" required>
      </div>
    </form>
    <template #confirm-text>
      {{ state.loading ? '重命名中...' : '重命名' }}
    </template>
  </BaseModal>
</template>
