<script setup>
import { inject, reactive, ref } from 'vue'

import api from '@/service/api.js'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const state = inject(APP_STATE_KEY)
const actions = inject(APP_ACTIONS_KEY)

const formData = reactive({
  name: ''
})

const modalRef = ref(null)

const show = () => {
  formData.name = ''
  modalRef.value?.show()
}

const handleConfirm = async () => {
  if (!formData.name.trim()) return
  await api.createDirectory(state.currentPath, formData.name)
  actions.loadFiles()
  modalRef.value.hide()
}

defineExpose({ show })
</script>

<template>
  <BaseModal ref="modalRef" id="mkdirModal" title="新建目录" :loading="state.loading" :confirm-disabled="!formData.name.trim()" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div class="mb-3">
        <label for="dirName" class="form-label">目录名称</label>
        <input type="text" class="form-control" id="dirName" v-model="formData.name" :disabled="state.loading" required>
      </div>
    </form>
    <template #confirm-text>
      {{ state.loading ? '创建中...' : '创建' }}
    </template>
  </BaseModal>
</template>
