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

<script setup>
import { inject, reactive, ref } from 'vue'
import axios from 'axios'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/stores/state.js'
import BaseModal from '@/components/base/base-modal.vue'

const state = inject(APP_STATE_KEY)
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
    await axios.post('/api/rename', {
      oldPath: formData.file.path,
      newName: formData.name
    })

    actions.showSuccess('重命名成功')
    actions.loadFiles()
    modalRef.value.hide()
  } catch (error) {
    actions.showError(error.response?.data?.error || '重命名失败')
  } finally {
    formData.loading = false
  }
}

defineExpose({ show })
</script>
