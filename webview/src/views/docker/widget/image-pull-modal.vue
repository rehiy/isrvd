<script setup>
import { inject, ref } from 'vue'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const actions = inject(APP_ACTIONS_KEY)

const emit = defineEmits(['success'])

const modalRef = ref(null)
const isOpen = ref(false)
const modalLoading = ref(false)

const formData = ref({ image: '', tag: '' })

// 搜索状态
const searchResults = ref([])
const searchLoading = ref(false)
const searchKeyword = ref('')

// 镜像加速器（来自 Docker daemon）
const daemonMirrors = ref([])
const indexServerAddress = ref('')

const loadDaemonInfo = async () => {
  try {
    const res = await api.dockerInfo()
    const info = res.payload || {}
    daemonMirrors.value = info.registryMirrors || []
    indexServerAddress.value = info.indexServerAddress || ''
  } catch (e) {}
}

const show = () => {
  formData.value = { image: '', tag: '' }
  searchResults.value = []
  searchKeyword.value = ''
  loadDaemonInfo()
  isOpen.value = true
}

const handleConfirm = async () => {
  if (!formData.value.image.trim()) return
  modalLoading.value = true
  try {
    await api.pullImage(formData.value.image, formData.value.tag)
    actions.showNotification('success', '镜像拉取成功')
    isOpen.value = false
    emit('success')
  } catch (e) {}
  modalLoading.value = false
}

const handleSearchImage = async () => {
  if (!searchKeyword.value.trim()) return
  searchLoading.value = true
  try {
    const res = await api.imageSearch(searchKeyword.value.trim())
    searchResults.value = res.payload || []
  } catch (e) {
    actions.showNotification('error', '搜索镜像失败')
  }
  searchLoading.value = false
}

const selectSearchResult = (item) => {
  formData.value.image = item.name
  formData.value.tag = 'latest'
  searchResults.value = []
  searchKeyword.value = ''
}

defineExpose({ show })
</script>

<template>
  <BaseModal
    ref="modalRef"
    v-model="isOpen"
    title="拉取镜像"
    :loading="modalLoading"
    show-footer
    @confirm="handleConfirm"
  >
    <form @submit.prevent="handleConfirm" class="space-y-4">
      <!-- 镜像搜索 -->
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">搜索镜像</label>
        <div class="flex gap-2">
          <input type="text" v-model="searchKeyword" placeholder="搜索 Docker Hub 镜像" class="input flex-1" @keyup.enter="handleSearchImage" />
          <button type="button" @click="handleSearchImage" :disabled="searchLoading" class="px-4 py-2 rounded-lg bg-slate-700 hover:bg-slate-800 text-white text-xs font-medium flex items-center gap-1.5 transition-colors disabled:opacity-50">
            <i :class="['fas', searchLoading ? 'fa-spinner fa-spin' : 'fa-search']"></i>
            {{ searchLoading ? '搜索中' : '搜索' }}
          </button>
        </div>
      </div>

      <!-- 搜索结果 -->
      <div v-if="searchResults.length > 0" class="border border-slate-200 rounded-xl max-h-48 overflow-y-auto">
        <div 
          v-for="item in searchResults" 
          :key="item.name" 
          @click="selectSearchResult(item)" 
          class="px-4 py-2.5 hover:bg-blue-50 cursor-pointer border-b border-slate-100 last:border-b-0 transition-colors"
        >
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <i v-if="item.isOfficial" class="fas fa-certificate text-blue-500 text-xs" title="官方镜像"></i>
              <span class="text-sm font-medium text-slate-800">{{ item.name }}</span>
            </div>
            <div class="flex items-center gap-2">
              <span v-if="item.isOfficial" class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-700">官方</span>
              <span class="text-xs text-slate-400"><i class="fas fa-star text-amber-400 mr-0.5"></i>{{ item.starCount }}</span>
            </div>
          </div>
          <p v-if="item.description" class="text-xs text-slate-500 mt-0.5 truncate">{{ item.description }}</p>
        </div>
      </div>

      <div class="border-t border-slate-200 pt-4">
        <div class="mb-4">
          <label class="block text-sm font-medium text-slate-700 mb-2">镜像名称</label>
          <input type="text" v-model="formData.image" placeholder="例如: nginx, redis:alpine, ubuntu:22.04" required class="input" />
        </div>
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">Tag（可选）</label>
          <input type="text" v-model="formData.tag" placeholder="默认 latest" class="input" />
        </div>
      </div>

      <!-- 镜像源提示 -->
      <div class="border-t border-slate-200 pt-4">
        <p class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2">当前镜像源</p>
        <div v-if="daemonMirrors.length > 0" class="flex flex-wrap gap-1.5">
          <code
            v-for="mirror in daemonMirrors"
            :key="mirror"
            class="inline-flex items-center gap-1 px-2 py-1 bg-sky-50 border border-sky-200 rounded-lg text-xs font-mono text-sky-700"
          >
            <i class="fas fa-bolt text-sky-400 text-xs"></i>{{ mirror }}
          </code>
        </div>
        <div v-else class="flex items-center gap-1.5 text-xs text-slate-400">
          <i class="fab fa-docker text-blue-400"></i>
          {{ indexServerAddress || 'https://index.docker.io/v1/' }}
          <span class="text-slate-300">（未配置加速器）</span>
        </div>
      </div>
    </form>
  </BaseModal>
</template>
