<script setup>
import { inject, reactive, ref } from 'vue'

import api from '@/service/api.js'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const state = inject(APP_STATE_KEY)
const actions = inject(APP_ACTIONS_KEY)

const formData = reactive({
  file: null
})

const modalRef = ref(null)
const isOpen = ref(false)

const show = (file) => {
  formData.file = file
  isOpen.value = true
}

const handleConfirm = async () => {
  await api.zip(state.currentPath + '/' + formData.file.name)
  actions.loadFiles()
  isOpen.value = false
}

defineExpose({ show })
</script>

<template>
  <BaseModal ref="modalRef" v-model="isOpen" title="压缩确认" :loading="state.loading" :confirm-disabled="!formData.file" @confirm="handleConfirm">
    <div v-if="formData.file" class="text-center py-6">
      <div class="w-20 h-20 rounded-full bg-amber-400 flex items-center justify-center mx-auto mb-4 shadow-lg shadow-amber-500/30">
        <i class="fas fa-file-archive text-3xl text-white"></i>
      </div>
      <p class="text-lg text-slate-700 mb-2">
        确定要压缩 <strong class="text-slate-900">{{ formData.file.name }}</strong> 吗？
      </p>
      <p class="text-sm text-slate-500">压缩后的文件将保存在当前目录</p>
    </div>
    <template #confirm-text>
      <i class="fas fa-file-archive mr-2" v-if="!state.loading"></i>
      {{ state.loading ? '压缩中...' : '开始压缩' }}
    </template>
  </BaseModal>
</template>
