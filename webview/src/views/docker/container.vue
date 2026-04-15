<script setup>
import { inject, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

const route = useRoute()
const router = useRouter()
const actions = inject(APP_ACTIONS_KEY)

// 容器信息
const container = ref(null)
const loading = ref(true)

// 从路由获取容器 ID
const containerId = ref(route.params.id)

// 当前激活的 Tab（根据路由名称判断）
const activeTab = () => route.name

// ========== 加载容器信息 ==========

const loadContainer = async () => {
  loading.value = true
  try {
    const res = await api.listContainers(true)
    const list = res.payload || []
    container.value = list.find(c => c.id === containerId.value)
    if (!container.value) {
      actions.showNotification('error', '容器不存在')
      router.push('/docker/containers')
      return
    }
  } catch (e) {
    actions.showNotification('error', '加载容器信息失败')
    router.push('/docker/containers')
  }
  loading.value = false
}

const goBack = () => {
  router.push('/docker/containers')
}

const switchTab = (name) => {
  router.push({ name, params: { id: containerId.value } })
}

onMounted(async () => {
  await loadContainer()
})
</script>

<template>
  <div>
    <!-- 顶部导航栏 -->
    <div class="card mb-4">
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-6 py-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <button @click="goBack" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-100 flex items-center justify-center text-slate-600 transition-colors" title="返回容器列表">
              <i class="fas fa-arrow-left text-sm"></i>
            </button>
            <template v-if="container">
              <div :class="['w-9 h-9 rounded-lg flex items-center justify-center', container.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
                <i class="fas fa-cube text-white"></i>
              </div>
              <div>
                <h1 class="text-lg font-semibold text-slate-800">{{ container.name || container.id }}</h1>
                <p class="text-xs text-slate-500 font-mono">{{ container.image }}</p>
              </div>
            </template>
            <template v-else>
              <div class="w-9 h-9 rounded-lg bg-slate-300 flex items-center justify-center animate-pulse">
                <i class="fas fa-cube text-white"></i>
              </div>
              <div>
                <h1 class="text-lg font-semibold text-slate-800">加载中...</h1>
              </div>
            </template>
          </div>

          <!-- Tab 切换 -->
          <div v-if="container" class="flex items-center gap-1 bg-slate-100 p-1 rounded-lg">
            <button
              @click="switchTab('docker-container-stats')"
              :class="[
                'px-4 py-2 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5',
                activeTab() === 'docker-container-stats'
                  ? 'bg-white text-emerald-600 shadow-sm'
                  : 'text-slate-500 hover:text-slate-700'
              ]"
            >
              <i class="fas fa-chart-line"></i>监控
            </button>
            <button
              @click="switchTab('docker-container-logs')"
              :class="[
                'px-4 py-2 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5',
                activeTab() === 'docker-container-logs'
                  ? 'bg-white text-emerald-600 shadow-sm'
                  : 'text-slate-500 hover:text-slate-700'
              ]"
            >
              <i class="fas fa-file-alt"></i>日志
            </button>
            <button
              @click="switchTab('docker-container-terminal')"
              :class="[
                'px-4 py-2 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5',
                activeTab() === 'docker-container-terminal'
                  ? 'bg-white text-emerald-600 shadow-sm'
                  : 'text-slate-500 hover:text-slate-700'
              ]"
              :disabled="container.state !== 'running'"
            >
              <i class="fas fa-terminal"></i>终端
            </button>
          </div>
        </div>
      </div>

      <!-- 加载状态 -->
      <div v-if="loading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <!-- 子路由内容 -->
      <router-view v-else-if="container" :container-id="containerId" :container="container" />
    </div>
  </div>
</template>
