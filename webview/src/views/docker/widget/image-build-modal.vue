<script setup>
import { ref } from 'vue'
import { inject } from 'vue'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const actions = inject(APP_ACTIONS_KEY)

const emit = defineEmits(['success'])

const isOpen = ref(false)
const modalLoading = ref(false)
const buildTag = ref('')
const buildDockerfile = ref('FROM alpine:latest\nCMD ["echo", "Hello World"]')

const show = () => {
  buildTag.value = ''
  buildDockerfile.value = 'FROM alpine:latest\nCMD ["echo", "Hello World"]'
  isOpen.value = true
}

const handleConfirm = async () => {
  if (!buildDockerfile.value.trim()) return
  modalLoading.value = true
  try {
    await api.imageBuild(buildDockerfile.value, buildTag.value)
    actions.showNotification('success', '镜像构建成功')
    isOpen.value = false
    emit('success')
  } catch (e) {}
  modalLoading.value = false
}

defineExpose({ show })
</script>

<template>
  <BaseModal
    v-model="isOpen"
    title="构建镜像"
    :loading="modalLoading"
    show-footer
    @confirm="handleConfirm"
  >
    <template #confirm-text>开始构建</template>
    <div class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">镜像标签</label>
        <input type="text" v-model="buildTag" placeholder="例如: myapp:v1, custom-image:latest" class="input" />
        <p class="mt-1 text-xs text-slate-400">留空则使用 custom:latest</p>
      </div>
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">Dockerfile</label>
        <textarea
          v-model="buildDockerfile"
          rows="14"
          class="input font-mono text-sm"
          placeholder="FROM alpine:latest&#10;RUN echo hello"
          spellcheck="false"
        ></textarea>
      </div>
    </div>
  </BaseModal>
</template>
