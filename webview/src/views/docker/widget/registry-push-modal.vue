<script setup>
import { computed, ref } from 'vue'
import { inject } from 'vue'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const actions = inject(APP_ACTIONS_KEY)

const emit = defineEmits(['success'])

const isOpen = ref(false)
const modalLoading = ref(false)
const registries = ref([])
const localImages = ref([])
const pushForm = ref({ image: '', registryUrl: '', namespace: '' })

const imageTagOptions = computed(() => {
  const tags = []
  for (const img of localImages.value) {
    for (const tag of img.repoTags) {
      if (tag !== '<none>:<none>') {
        tags.push(tag)
      }
    }
  }
  return tags
})

const pushTargetPreview = computed(() => {
  const registry = pushForm.value.registryUrl || 'registry'
  const ns = pushForm.value.namespace ? pushForm.value.namespace + '/' : ''
  let imageName = pushForm.value.image || 'image:tag'
  const lastSlash = imageName.lastIndexOf('/')
  if (lastSlash >= 0) {
    imageName = imageName.substring(lastSlash + 1)
  }
  return registry + '/' + ns + imageName
})

const loadLocalImages = async () => {
  try {
    const res = await api.listImages(false)
    localImages.value = (res.payload || []).filter(img => img.repoTags && img.repoTags.length > 0)
  } catch (e) {}
}

const show = (allRegistries, registry = null) => {
  registries.value = allRegistries
  pushForm.value = {
    image: '',
    registryUrl: registry ? registry.url : '',
    namespace: ''
  }
  loadLocalImages()
  isOpen.value = true
}

const handleConfirm = async () => {
  if (!pushForm.value.image.trim() || !pushForm.value.registryUrl.trim()) return
  modalLoading.value = true
  try {
    await api.pushImage(pushForm.value.image, pushForm.value.registryUrl, pushForm.value.namespace.trim() || undefined)
    actions.showNotification('success', '镜像推送成功')
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
    title="推送镜像到仓库"
    :loading="modalLoading"
    show-footer
    @confirm="handleConfirm"
  >
    <template #confirm-text>开始推送</template>
    <form @submit.prevent="handleConfirm" class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">本地镜像</label>
        <select v-model="pushForm.image" class="input" required>
          <option value="" disabled>请选择镜像</option>
          <option v-for="tag in imageTagOptions" :key="tag" :value="tag">{{ tag }}</option>
        </select>
      </div>
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">目标仓库地址</label>
        <select v-model="pushForm.registryUrl" class="input" required>
          <option value="" disabled>请选择仓库</option>
          <option v-for="reg in registries" :key="reg.url" :value="reg.url">{{ reg.name }} ({{ reg.url }}){{ reg.description ? ' - ' + reg.description : '' }}</option>
        </select>
      </div>
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">命名空间 <span class="text-slate-400 font-normal">(可选)</span></label>
        <input type="text" v-model="pushForm.namespace" placeholder="例如: myteam" class="input" />
        <p class="mt-1 text-xs text-slate-400">镜像将被推送为: {{ pushTargetPreview }}</p>
      </div>
    </form>
  </BaseModal>
</template>
