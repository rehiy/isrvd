<script setup>
import { inject, reactive, ref } from 'vue'

import api from '@/services/api.js'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/stores/state.js'

import BaseModal from '@/components/modal-base.vue'

const state = inject(APP_STATE_KEY)
const actions = inject(APP_ACTIONS_KEY)

const formData = reactive({
  file: null,
  loading: false
})

const modalRef = ref(null)

const show = (file) => {
  formData.file = file
  formData.loading = false
  modalRef.value.show()
}

const handleConfirm = async () => {
  if (!formData.file) return

  formData.loading = true

  try {
    await api.unzipFile(formData.file.name, state.currentPath)
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
  <BaseModal ref="modalRef" id="unzipModal" title="解压确认" header-class="bg-warning text-dark" :loading="formData.loading" :confirm-disabled="!formData.file" @confirm="handleConfirm">
    <div v-if="formData.file" class="text-center">s
      <div class="mb-3">
        <i class="fas fa-file-archive text-warning display-1"></i>
      </div>
      <p class="mb-3">
        确定要解压 <strong>{{ formData.file.name }}</strong> 到当前目录吗？
      </p>
    </div>
    <template #confirm-text>
      {{ formData.loading ? '解压中...' : '解压' }}
    </template>
  </BaseModal>
</template>
