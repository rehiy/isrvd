<script setup>
import { inject, reactive, ref, computed } from 'vue'

import api from '@/service/api.js'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const state = inject(APP_STATE_KEY)
const actions = inject(APP_ACTIONS_KEY)

const formData = reactive({
  uploadFile: null
})

const fileInput = ref(null)
const modalRef = ref(null)
const isOpen = ref(false)

const show = () => {
  formData.uploadFile = null
  isOpen.value = true
}

const handleFileChange = (event) => {
  formData.uploadFile = event.target.files[0] || null
}

const handleConfirm = async () => {
  if (!formData.uploadFile) return

  const formDataToSend = new FormData()
  formDataToSend.append('file', formData.uploadFile)
  formDataToSend.append('path', state.currentPath)

  await api.upload(formDataToSend)
  actions.loadFiles()

  formData.uploadFile = null
  if (fileInput.value) {
    fileInput.value.value = ''
  }
  isOpen.value = false
}

const hasFile = computed(() => {
  return formData.uploadFile !== null
})

defineExpose({ show })
</script>

<template>
  <BaseModal ref="modalRef" v-model="isOpen" title="上传文件" :loading="state.loading" :confirm-disabled="!hasFile" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div>
        <label for="uploadFile" class="block text-sm font-medium text-slate-700 mb-2">
          选择文件
        </label>
        <div class="relative">
          <input 
            type="file" 
            id="uploadFile" 
            ref="fileInput" 
            @change="handleFileChange" 
            :disabled="state.loading" 
            required
            class="input file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:text-sm file:font-medium file:bg-primary-50 file:text-primary-700 hover:file:bg-primary-100"
          >
        </div>
        <div v-if="formData.uploadFile" class="mt-3 p-3 bg-primary-50 rounded-lg border border-primary-200">
          <div class="flex items-center">
            <div class="w-10 h-10 rounded-lg bg-primary-100 flex items-center justify-center mr-3">
              <i class="fas fa-file text-primary-600"></i>
            </div>
            <div>
              <p class="text-sm font-medium text-slate-700">{{ formData.uploadFile.name }}</p>
              <p class="text-xs text-slate-500">{{ (formData.uploadFile.size / 1024).toFixed(2) }} KB</p>
            </div>
          </div>
        </div>
      </div>
    </form>
    <template #confirm-text>
      <i class="fas fa-cloud-arrow-up mr-2" v-if="!state.loading"></i>
      {{ state.loading ? '上传中...' : '开始上传' }}
    </template>
  </BaseModal>
</template>
