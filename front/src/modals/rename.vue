<script setup>
import { inject, reactive, ref } from 'vue'

import api from '@/services/api.js'
import { APP_ACTIONS_KEY } from '@/stores/state.js'

import BaseModal from '@/components/modal-base.vue'

const actions = inject(APP_ACTIONS_KEY)

const formData = reactive({
  name: '',
  loading: false,
  file: null
})

const modalRef = ref(null)

const show = (file) => {
  formData.file = file
  formData.name = file.name
  formData.loading = false
  modalRef.value.show()
}

const handleConfirm = async () => {
  if (!formData.name.trim() || !formData.file) return

  formData.loading = true

  try {
    await api.renameFile(formData.file.path, formData.name)
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
  <BaseModal ref="modalRef" id="renameModal" title="重命名" :loading="formData.loading" :confirm-disabled="!formData.name.trim()" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div class="mb-3">
        <label for="newName" class="form-label">新名称</label>
        <input type="text" class="form-control" id="newName" v-model="formData.name" :disabled="formData.loading" required>
      </div>
    </form>
    <template #confirm-text>
      {{ formData.loading ? '重命名中...' : '重命名' }}
    </template>
  </BaseModal>
</template>
