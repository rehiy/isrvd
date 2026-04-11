<script setup>
import { onMounted, ref } from 'vue'

import { formatFileSize, formatTime } from '@/helper/utils.js'
import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'
import { inject } from 'vue'

import BaseModal from '@/component/modal.vue'

const actions = inject(APP_ACTIONS_KEY)

// 自己管理数据
const images = ref([])
const loading = ref(false)
const showAllImages = ref(false)

// 模态框状态
const modalOpen = ref(false)
const modalTitle = ref('')
const modalLoading = ref(false)
const formData = ref({})

// 加载镜像列表
const loadImages = async () => {
  loading.value = true
  try {
const res = await api.listImages(showAllImages.value)
    images.value = res.payload || []
  } catch (e) {
    actions.showNotification('error', '加载镜像列表失败')
  }
  loading.value = false
}

// 拉取镜像弹窗
const pullImageModal = () => {
  formData.value = { image: '', tag: '' }
  modalTitle.value = '拉取镜像'
  modalOpen.value = true
}

const handlePullImage = async () => {
  if (!formData.value.image.trim()) return
  modalLoading.value = true
  try {
    await api.pullImage(formData.value.image, formData.value.tag)
    actions.showNotification('success', '镜像拉取成功')
    modalOpen.value = false
    loadImages()
  } catch (e) {
    // error handled by interceptor
  }
  modalLoading.value = false
}

// 删除镜像
const handleImageAction = async (image, action) => {
  if (!confirm(`确定要删除镜像 "${image.repoTags[0] || image.id}" 吗？`)) return
  try {
    await api.imageAction(image.id, action)
    actions.showNotification('success', '镜像删除成功')
    loadImages()
  } catch (e) {
    // error handled
  }
}

onMounted(() => {
  loadImages()
})
</script>

<template>
  <div class="p-4 space-y-4">
    <!-- 页面标题 -->
    <div class="flex items-center justify-between mb-4">
      <div class="flex items-center gap-3">
        <div class="w-9 h-9 rounded-lg bg-blue-500 flex items-center justify-center">
          <i class="fas fa-compact-disc text-white"></i>
        </div>
        <div>
          <h1 class="text-lg font-semibold text-slate-800">镜像管理</h1>
          <p class="text-xs text-slate-500">管理 Docker 镜像</p>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <label class="flex items-center gap-1.5 text-xs text-slate-600 cursor-pointer select-none">
          <input type="checkbox" v-model="showAllImages" @change="loadImages()" class="rounded border-slate-300">
          显示全部
        </label>
<div class="flex items-center gap-2 ml-2">
          <button @click="loadImages()" class="px-3 py-1.5 rounded-lg bg-slate-100 hover:bg-slate-200 text-slate-600 text-xs font-medium flex items-center gap-1.5 transition-colors">
            <i class="fas fa-rotate"></i>刷新
          </button>
          <button @click="pullImageModal()" class="px-3 py-1.5 rounded-lg bg-blue-500 hover:bg-blue-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
            <i class="fas fa-download"></i>拉取
          </button>
        </div>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="flex flex-col items-center justify-center py-20">
      <div class="w-12 h-12 spinner mb-3"></div>
      <p class="text-slate-500">加载中...</p>
    </div>

    <!-- Image Table -->
    <div v-else-if="images.length > 0" class="overflow-x-auto rounded-xl border border-slate-200">
      <table class="w-full">
        <thead class="bg-slate-50">
          <tr>
            <th class="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase">镜像</th>
            <th class="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase">ID</th>
            <th class="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase w-32">大小</th>
            <th class="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase w-36">创建时间</th>
            <th class="text-right px-4 py-3 text-xs font-semibold text-slate-500 uppercase w-24">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-100">
          <tr v-for="img in images" :key="img.id" class="hover:bg-slate-50/50 transition-colors">
            <td class="px-4 py-3">
              <div class="flex items-center gap-2">
                <div class="w-8 h-8 rounded-lg bg-blue-400 flex items-center justify-center">
                  <i class="fas fa-compact-disc text-white text-sm"></i>
                </div>
                <span class="font-medium text-slate-800">{{ img.repoTags[0] || '<none>' }}</span>
              </div>
            </td>
            <td class="px-4 py-3"><code class="text-xs text-slate-500 font-mono">{{ img.shortId }}</code></td>
            <td class="px-4 py-3 text-sm text-slate-600">{{ formatFileSize(img.size) }}</td>
            <td class="px-4 py-3 text-sm text-slate-500 whitespace-nowrap">{{ formatTime(new Date(img.created * 1000).toISOString()) }}</td>
            <td class="px-4 py-3">
              <div class="flex items-center justify-end">
                <button @click="handleImageAction(img, 'remove')" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
                  <i class="fas fa-trash text-xs"></i>
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-else class="flex flex-col items-center justify-center py-20">
      <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
        <i class="fas fa-compact-disc text-4xl text-slate-300"></i>
      </div>
      <p class="text-slate-600 font-medium mb-1">暂无镜像</p>
      <p class="text-sm text-slate-400">点击「拉取镜像」从 Registry 获取镜像</p>
    </div>

    <!-- 拉取镜像模态框 -->
    <BaseModal 
      v-model="modalOpen" 
      :title="modalTitle" 
      :loading="modalLoading"
      :show-footer="modalTitle === '拉取镜像'"
      @confirm="handlePullImage"
    >
      <template v-if="modalTitle === '拉取镜像'">
        <form @submit.prevent="handlePullImage" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">镜像名称</label>
            <input type="text" v-model="formData.image" placeholder="例如: nginx, redis:alpine, ubuntu:22.04" required class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">Tag（可选）</label>
            <input type="text" v-model="formData.tag" placeholder="默认 latest" class="input" />
          </div>
        </form>
      </template>
    </BaseModal>
  </div>
</template>
