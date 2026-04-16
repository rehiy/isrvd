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
const isOpen = ref(false)

const show = () => {
  formData.name = ''
  isOpen.value = true
}

const handleConfirm = async () => {
  if (!formData.name.trim()) return
  await api.mkdir(state.currentPath + '/' + formData.name)
  actions.loadFiles()
  isOpen.value = false
}

defineExpose({ show })
</script>

<template>
  <BaseModal ref="modalRef" v-model="isOpen" title="新建目录" :loading="state.loading" :confirm-disabled="!formData.name.trim()" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div>
        <label for="dirName" class="block text-sm font-medium text-slate-700 mb-2">
          目录名称
        </label>
        <div class="relative">
          <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
            <i class="fas fa-folder text-slate-400"></i>
          </div>
          <input 
            type="text" 
            id="dirName" 
            v-model="formData.name" 
            :disabled="state.loading" 
            required
            class="input pl-11"
            placeholder="请输入目录名称"
          >
        </div>
      </div>
    </form>
    <template #confirm-text>
      <i class="fas fa-folder mr-2" v-if="!state.loading"></i>
      {{ state.loading ? '创建中...' : '创建目录' }}
    </template>
  </BaseModal>
</template>
