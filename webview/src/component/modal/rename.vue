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
const isOpen = ref(false)

const show = (file) => {
  formData.file = file
  formData.name = file.name
  isOpen.value = true
}

const handleConfirm = async () => {
  if (!formData.name.trim() || !formData.file) return
  await api.rename(formData.file.path, formData.name)
  actions.loadFiles()
  isOpen.value = false
}

defineExpose({ show })
</script>

<template>
  <BaseModal ref="modalRef" v-model="isOpen" title="重命名" :loading="state.loading" :confirm-disabled="!formData.name.trim()" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div>
        <label for="target" class="block text-sm font-medium text-slate-700 mb-2">
          新名称
        </label>
        <div class="relative">
          <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
            <i class="fas fa-pen text-slate-400"></i>
          </div>
          <input 
            type="text" 
            id="target" 
            v-model="formData.name" 
            :disabled="state.loading" 
            required
            class="input pl-11"
            placeholder="请输入新名称"
          >
        </div>
      </div>
    </form>
    <template #confirm-text>
      <i class="fas fa-check mr-2" v-if="!state.loading"></i>
      {{ state.loading ? '重命名中...' : '确认重命名' }}
    </template>
  </BaseModal>
</template>
