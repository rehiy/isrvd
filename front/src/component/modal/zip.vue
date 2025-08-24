<script setup>
import { inject, reactive, ref } from 'vue'

import api from '@/service/api.js'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const state = inject(APP_STATE_KEY)
const actions = inject(APP_ACTIONS_KEY)

const formData = reactive({
  path: '',
  name: '',
  zipName: ''
})

const modalRef = ref(null)

const show = (file) => {
  formData.path = file.path
  formData.name = file.name
  formData.zipName = file.name + '.zip'
  modalRef.value.show()
}

const handleConfirm = async () => {
  if (!formData.zipName.trim()) return
  await api.zipFiles(formData.path, formData.zipName)
  actions.loadFiles()
  modalRef.value.hide()
}

defineExpose({ show })
</script>

<template>
  <BaseModal ref="modalRef" id="zipModal" title="压缩目录" :loading="state.loading" :confirm-disabled="!formData.zipName.trim()" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div class="mb-3">
        <label for="zipName" class="form-label">压缩包名称</label>
        <input type="text" class="form-control" id="zipName" v-model="formData.zipName" :disabled="state.loading" required>
      </div>
    </form>
    <template #confirm-text>
      {{ state.loading ? '压缩中...' : '压缩' }}
    </template>
  </BaseModal>
</template>
