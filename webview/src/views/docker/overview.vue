<script setup>
import { inject, onMounted, ref } from 'vue'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

const actions = inject(APP_ACTIONS_KEY)

const info = ref(null)
const loading = ref(false)

const loadInfo = async () => {
  loading.value = true
  try {
    const res = await api.dockerInfo()
    info.value = res.payload || {}
  } catch (e) {
    actions.showNotification('error', '加载 Docker 信息失败')
  }
  loading.value = false
}

const statCards = [
  { key: 'containersRunning', label: '运行中容器', icon: 'fa-play', color: 'emerald', bgColor: 'bg-emerald-500' },
  { key: 'containersStopped', label: '已停止容器', icon: 'fa-stop', color: 'amber', bgColor: 'bg-amber-500' },
  { key: 'containersPaused', label: '已暂停容器', icon: 'fa-pause', color: 'slate', bgColor: 'bg-slate-500' },
  { key: 'imagesTotal', label: '镜像总数', icon: 'fa-compact-disc', color: 'blue', bgColor: 'bg-blue-500' },
  { key: 'volumesTotal', label: '数据卷总数', icon: 'fa-database', color: 'indigo', bgColor: 'bg-indigo-500' },
  { key: 'networksTotal', label: '网络总数', icon: 'fa-network-wired', color: 'purple', bgColor: 'bg-purple-500' },
]

onMounted(() => {
  loadInfo()
})
</script>

<template>
  <div>
    <div class="card mb-4">
      <!-- Toolbar Bar -->
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-6 py-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-primary-500 flex items-center justify-center">
              <i class="fab fa-docker text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">Docker 概览</h1>
              <p class="text-xs text-slate-500">Docker 服务运行状态总览</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <button @click="loadInfo()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-rotate"></i>刷新
            </button>
          </div>
        </div>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <!-- Stats Grid -->
      <div v-else-if="info" class="p-6">
        <div class="grid grid-cols-2 md:grid-cols-3 gap-4">
          <div
            v-for="card in statCards"
            :key="card.key"
            class="relative overflow-hidden rounded-xl border border-slate-200 bg-white p-5 hover:shadow-md transition-shadow"
          >
            <div class="flex items-center gap-4">
              <div :class="['w-12 h-12 rounded-xl flex items-center justify-center', card.bgColor]">
                <i :class="['fas', card.icon, 'text-white text-lg']"></i>
              </div>
              <div>
                <p class="text-sm text-slate-500">{{ card.label }}</p>
                <p class="text-2xl font-bold text-slate-800 mt-1">{{ info[card.key] || 0 }}</p>
              </div>
            </div>
          </div>
        </div>

        <!-- Quick Links -->
        <div class="mt-6 pt-6 border-t border-slate-200">
          <h3 class="text-sm font-medium text-slate-600 mb-3">快速跳转</h3>
          <div class="flex flex-wrap gap-2">
            <router-link to="/docker/containers" class="px-4 py-2 rounded-lg bg-emerald-50 hover:bg-emerald-100 text-emerald-700 text-sm font-medium flex items-center gap-2 transition-colors">
              <i class="fas fa-cube"></i>容器管理
            </router-link>
            <router-link to="/docker/images" class="px-4 py-2 rounded-lg bg-blue-50 hover:bg-blue-100 text-blue-700 text-sm font-medium flex items-center gap-2 transition-colors">
              <i class="fas fa-layer-group"></i>镜像管理
            </router-link>
            <router-link to="/docker/networks" class="px-4 py-2 rounded-lg bg-purple-50 hover:bg-purple-100 text-purple-700 text-sm font-medium flex items-center gap-2 transition-colors">
              <i class="fas fa-network-wired"></i>网络管理
            </router-link>
            <router-link to="/docker/volumes" class="px-4 py-2 rounded-lg bg-indigo-50 hover:bg-indigo-100 text-indigo-700 text-sm font-medium flex items-center gap-2 transition-colors">
              <i class="fas fa-database"></i>数据卷管理
            </router-link>
          </div>
        </div>
      </div>

      <!-- Empty/Error State -->
      <div v-else class="flex flex-col items-center justify-center py-20">
        <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
          <i class="fab fa-docker text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">无法获取 Docker 信息</p>
        <p class="text-sm text-slate-400">请确认 Docker 服务是否正常运行</p>
      </div>
    </div>
  </div>
</template>
