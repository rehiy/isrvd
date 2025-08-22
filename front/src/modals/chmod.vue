<script setup>
import { inject, reactive, ref } from 'vue'

import api from '@/services/api.js'
import { APP_ACTIONS_KEY } from '@/stores/state.js'

import BaseModal from '@/components/modal-base.vue'

const actions = inject(APP_ACTIONS_KEY)

const formData = reactive({
  path: '',
  mode: '',
  loading: false
})

const modalRef = ref(null)

const show = async (file) => {
  formData.loading = true

  try {
    const data = await api.getFilePermissions(file.path)
    formData.path = file.path
    formData.mode = data.payload.mode
    modalRef.value.show()
  } catch (error) {
  } finally {
    formData.loading = false
  }
}

const handleConfirm = async () => {
  if (!formData.mode.trim()) return

  formData.loading = true

  try {
    await api.setFilePermissions(formData.path, formData.mode)
    actions.showSuccess('权限修改成功')
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
  <BaseModal ref="modalRef" id="chmodModal" title="修改权限" :loading="formData.loading" :confirm-disabled="!formData.mode.trim()" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div class="mb-3">
        <label for="fileMode" class="form-label">权限 (八进制)</label>
        <input type="text" class="form-control" id="fileMode" v-model="formData.mode" :disabled="formData.loading" required placeholder="755">
        <div class="form-text">
          常用权限: 755 (rwxr-xr-x), 644 (rw-r--r--), 777 (rwxrwxrwx)
        </div>
      </div>
    </form>
    <template #confirm-text>
      {{ formData.loading ? '修改中...' : '修改' }}
    </template>
  </BaseModal>
</template>
