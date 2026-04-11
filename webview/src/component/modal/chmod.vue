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
const isOpen = ref(false)

const show = async (file) => {
  formData.path = file.path
  formData.mode = file.modeO
  isOpen.value = true
}

const handleConfirm = async () => {
  if (!formData.mode.trim()) return
  await api.chmod(formData.path, formData.mode)
  actions.loadFiles()
  isOpen.value = false
}

defineExpose({ show })
</script>

<template>
  <BaseModal ref="modalRef" v-model="isOpen" title="修改权限" :loading="state.loading" :confirm-disabled="!formData.mode.trim()" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div>
        <label for="fileMode" class="block text-sm font-medium text-slate-700 mb-2">
          权限 (八进制)
        </label>
        <div class="relative">
          <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
            <i class="fas fa-key text-slate-400"></i>
          </div>
          <input 
            type="text" 
            id="fileMode" 
            v-model="formData.mode" 
            :disabled="state.loading" 
            required 
            placeholder="755"
            class="input pl-11"
          >
        </div>
        <div class="mt-3 p-4 bg-slate-50 rounded-xl border border-slate-200">
          <p class="text-sm font-medium text-slate-700 mb-2">常用权限:</p>
          <div class="flex flex-wrap gap-2">
            <span class="badge-primary">755 - rwxr-xr-x</span>
            <span class="badge-primary">644 - rw-r--r--</span>
            <span class="badge-warning">777 - rwxrwxrwx</span>
          </div>
        </div>
      </div>
    </form>
    <template #confirm-text>
      <i class="fas fa-key mr-2" v-if="!state.loading"></i>
      {{ state.loading ? '修改中...' : '确认修改' }}
    </template>
  </BaseModal>
</template>
