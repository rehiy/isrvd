<script setup>
import { inject, reactive, ref } from 'vue'

import api from '@/service/api.js'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const state = inject(APP_STATE_KEY)
const actions = inject(APP_ACTIONS_KEY)

const formData = reactive({
  path: '',
  mode: ''
})

const modalRef = ref(null)

const show = async (file) => {
  const data = await api.getFilePermissions(file.path)
  formData.path = file.path
  formData.mode = data.payload.mode
  modalRef.value.show()
}

const handleConfirm = async () => {
  if (!formData.mode.trim()) return
  await api.setFilePermissions(formData.path, formData.mode)
  actions.loadFiles()
  modalRef.value.hide()
}

defineExpose({ show })
</script>

<template>
  <BaseModal ref="modalRef" id="chmodModal" title="修改权限" :loading="state.loading" :confirm-disabled="!formData.mode.trim()" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div class="mb-3">
        <label for="fileMode" class="form-label">权限 (八进制)</label>
        <input type="text" class="form-control" id="fileMode" v-model="formData.mode" :disabled="state.loading" required placeholder="755">
        <div class="form-text">
          常用权限: 755 (rwxr-xr-x), 644 (rw-r--r--), 777 (rwxrwxrwx)
        </div>
      </div>
    </form>
    <template #confirm-text>
      {{ state.loading ? '修改中...' : '修改' }}
    </template>
  </BaseModal>
</template>
