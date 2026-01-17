<script setup>
import { inject, ref, reactive } from 'vue'

import api from '@/service/api.js'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const state = inject(APP_STATE_KEY)
const actions = inject(APP_ACTIONS_KEY)

const formData = reactive({
  path: '',
  name: ''
})

const modalRef = ref(null)

const show = (file) => {
  formData.path = file.path
  formData.name = file.name
  modalRef.value.show()
}

const handleConfirm = async () => {
  if (!formData.path) return
  await api.delete(formData.path)
  actions.loadFiles()
  modalRef.value.hide()
}

defineExpose({ show })
</script>

<template>
  <BaseModal ref="modalRef" id="deleteModal" title="确认删除" header-class="text-bg-danger" :loading="state.loading" @confirm="handleConfirm">
    <p>确定要删除 <strong>{{ formData.name }}</strong> 吗？</p>
    <p class="text-danger mb-0">此操作不可恢复！</p>
    <template #confirm-text>
      {{ state.loading ? '删除中...' : '删除' }}
    </template>
  </BaseModal>
</template>
