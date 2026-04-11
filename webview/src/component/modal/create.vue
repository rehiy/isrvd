<script setup>
import { inject, reactive, ref } from 'vue'

import api from '@/service/api.js'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const state = inject(APP_STATE_KEY)
const actions = inject(APP_ACTIONS_KEY)

const formData = reactive({
  name: '',
  content: ''
})

const modalRef = ref(null)
const isOpen = ref(false)

const show = () => {
  formData.name = ''
  formData.content = ''
  isOpen.value = true
}

const handleConfirm = async () => {
  if (!formData.name.trim()) return
  await api.create(state.currentPath + '/' + formData.name, formData.content)
  actions.loadFiles()
  isOpen.value = false
}

defineExpose({ show })
</script>

<template>
  <BaseModal ref="modalRef" v-model="isOpen" title="新建文件" size="lg" :loading="state.loading" :confirm-disabled="!formData.name.trim()" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm" class="space-y-5">
      <div>
        <label for="fileName" class="block text-sm font-medium text-slate-700 mb-2">
          文件名称
        </label>
        <div class="relative">
          <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
            <i class="fas fa-file text-slate-400"></i>
          </div>
          <input 
            type="text" 
            id="fileName" 
            v-model="formData.name" 
            :disabled="state.loading" 
            required
            class="input pl-11"
            placeholder="请输入文件名称"
          >
        </div>
      </div>
      <div>
        <label for="fileContent" class="block text-sm font-medium text-slate-700 mb-2">
          文件内容
        </label>
        <textarea 
          id="fileContent" 
          rows="10" 
          v-model="formData.content" 
          :disabled="state.loading"
          class="input font-mono text-sm"
          placeholder="请输入文件内容..."
        ></textarea>
      </div>
    </form>
    <template #confirm-text>
      <i class="fas fa-file-circle-plus mr-2" v-if="!state.loading"></i>
      {{ state.loading ? '创建中...' : '创建文件' }}
    </template>
  </BaseModal>
</template>
