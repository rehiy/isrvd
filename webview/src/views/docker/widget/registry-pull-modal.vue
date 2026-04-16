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
const registries = ref([])
const pullForm = ref({ image: '', registryUrl: '', namespace: '' })

const show = (allRegistries, registry = null) => {
  registries.value = allRegistries
  pullForm.value = {
    image: '',
    registryUrl: registry ? registry.url : '',
    namespace: ''
  }
  isOpen.value = true
}

const handleConfirm = async () => {
  if (!pullForm.value.image.trim() || !pullForm.value.registryUrl.trim()) return
  modalLoading.value = true
  try {
    await api.pullFromRegistry(pullForm.value.image, pullForm.value.registryUrl, pullForm.value.namespace.trim() || undefined)
    actions.showNotification('success', '镜像拉取成功')
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
    title="从仓库拉取镜像"
    :loading="modalLoading"
    show-footer
    @confirm="handleConfirm"
  >
    <template #confirm-text>开始拉取</template>
    <form @submit.prevent="handleConfirm" class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">源仓库地址</label>
        <select v-model="pullForm.registryUrl" class="input" required>
          <option value="" disabled>请选择仓库</option>
          <option v-for="reg in registries" :key="reg.url" :value="reg.url">{{ reg.name }} ({{ reg.url }}){{ reg.description ? ' - ' + reg.description : '' }}</option>
        </select>
      </div>
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">命名空间 <span class="text-slate-400 font-normal">(可选)</span></label>
        <input type="text" v-model="pullForm.namespace" placeholder="例如: myteam" class="input" />
      </div>
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">镜像名称</label>
        <input type="text" v-model="pullForm.image" placeholder="输入镜像名称，如 myapp:latest" class="input" required />
        <p class="mt-1 text-xs text-slate-400">
          将拉取: {{ pullForm.registryUrl || 'registry' }}/{{ pullForm.namespace ? pullForm.namespace + '/' : '' }}{{ pullForm.image || 'image:tag' }}
        </p>
      </div>
    </form>
  </BaseModal>
</template>
