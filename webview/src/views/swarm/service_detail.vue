<script setup>
import { inject, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

const route = useRoute()
const router = useRouter()
const actions = inject(APP_ACTIONS_KEY)

const serviceId = route.params.id
const serviceData = ref(null)
const loading = ref(false)

// 当前激活的 Tab（根据路由名称判断）
const activeTab = () => route.name

const loadDetail = async () => {
  loading.value = true
  try {
    const res = await api.swarmInspectService(serviceId)
    serviceData.value = res.payload
  } catch (e) {
    actions.showNotification('error', '获取服务详情失败')
  }
  loading.value = false
}

const switchTab = (name) => {
  router.push({ name, params: { id: serviceId } })
}

onMounted(() => loadDetail())
</script>

<template>
  <div>
    <div class="card mb-4">
      <!-- Toolbar -->
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-6 py-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <button @click="router.back()" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-100 flex items-center justify-center text-slate-600 transition-colors" title="返回服务列表">
              <i class="fas fa-arrow-left text-sm"></i>
            </button>
            <div class="w-9 h-9 rounded-lg bg-emerald-500 flex items-center justify-center">
              <i class="fas fa-cubes text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">{{ serviceData?.name || '服务详情' }}</h1>
              <p class="text-xs text-slate-500 font-mono truncate max-w-xs">{{ serviceId }}</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <!-- Tab 切换 -->
            <div class="flex items-center gap-1 bg-slate-100 p-1 rounded-lg">
              <button
                @click="switchTab('swarm-service-detail-info')"
                :class="['px-4 py-2 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab() === 'swarm-service-detail-info' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']"
              ><i class="fas fa-circle-info"></i>详情</button>
              <button
                @click="switchTab('swarm-service-detail-logs')"
                :class="['px-4 py-2 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab() === 'swarm-service-detail-logs' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']"
              ><i class="fas fa-file-lines"></i>日志</button>
              <button
                @click="switchTab('swarm-service-detail-tasks')"
                :class="['px-4 py-2 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab() === 'swarm-service-detail-tasks' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']"
              ><i class="fas fa-list-check"></i>任务</button>
            </div>
            <button @click="loadDetail()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
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

      <!-- 未找到 -->
      <div v-else-if="!serviceData" class="flex flex-col items-center justify-center py-20">
        <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
          <i class="fas fa-cubes text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium">未找到服务详情</p>
      </div>

      <!-- 子路由内容 -->
      <router-view v-else :service-data="serviceData" />
    </div>
  </div>
</template>
