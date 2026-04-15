<script setup>
import { inject, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

const route = useRoute()
const router = useRouter()
const actions = inject(APP_ACTIONS_KEY)

const serviceId = route.params.id
const serviceName = ref('')
const logsContent = ref([])
const logsLoading = ref(false)
const logsTail = ref('200')

const activeTab = () => route.name
const switchTab = (name) => router.push({ name, params: { id: serviceId } })

const loadLogs = async () => {
  logsLoading.value = true
  try {
    const res = await api.swarmServiceLogs(serviceId, logsTail.value)
    logsContent.value = res.payload?.logs || []
  } catch (e) {
    logsContent.value = []
    actions.showNotification('error', '获取日志失败')
  }
  logsLoading.value = false
}

const loadServiceName = async () => {
  try {
    const res = await api.swarmInspectService(serviceId)
    serviceName.value = res.payload?.name || ''
  } catch (e) { /* 忽略，名称仅用于展示 */ }
}

onMounted(() => {
  loadServiceName()
  loadLogs()
})
</script>

<template>
  <div>
    <div class="card mb-4">
      <!-- Toolbar -->
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <button @click="router.back()" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-100 flex items-center justify-center text-slate-600 transition-colors" title="返回服务列表">
              <i class="fas fa-arrow-left text-sm"></i>
            </button>
            <div class="w-9 h-9 rounded-lg bg-emerald-500 flex items-center justify-center">
              <i class="fas fa-cubes text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">{{ serviceName || '服务详情' }}</h1>
              <p class="text-xs text-slate-500 font-mono truncate max-w-xs">{{ serviceId }}</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <div class="flex gap-1 bg-slate-100 p-1 rounded-lg">
              <button @click="switchTab('swarm-service-info')" :class="['px-3 py-2 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab() === 'swarm-service-info' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
                <i class="fas fa-circle-info"></i><span>详情</span>
              </button>
              <button @click="switchTab('swarm-service-logs')" :class="['px-3 py-2 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab() === 'swarm-service-logs' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
                <i class="fas fa-file-lines"></i><span>日志</span>
              </button>
              <button @click="switchTab('swarm-service-tasks')" :class="['px-3 py-2 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab() === 'swarm-service-tasks' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
                <i class="fas fa-list-check"></i><span>任务</span>
              </button>
            </div>
            <button @click="loadLogs()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-rotate"></i>刷新
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="block md:hidden">
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-3">
              <button @click="router.back()" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-100 flex items-center justify-center text-slate-600 transition-colors">
                <i class="fas fa-arrow-left text-sm"></i>
              </button>
              <div class="w-9 h-9 rounded-lg bg-emerald-500 flex items-center justify-center">
                <i class="fas fa-cubes text-white"></i>
              </div>
              <div class="min-w-0">
                <h1 class="text-lg font-semibold text-slate-800 truncate">{{ serviceName || '服务详情' }}</h1>
                <p class="text-xs text-slate-500 font-mono truncate">{{ serviceId.slice(0, 12) }}</p>
              </div>
            </div>
            <button @click="loadLogs()" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 flex items-center justify-center text-slate-600 transition-colors">
              <i class="fas fa-rotate text-sm"></i>
            </button>
          </div>
          <div class="flex justify-center gap-1 bg-slate-100 p-1 rounded-lg">
            <button @click="switchTab('swarm-service-info')" :class="['px-3 py-2 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab() === 'swarm-service-info' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
              <i class="fas fa-circle-info"></i><span class="hidden sm:inline">详情</span>
            </button>
            <button @click="switchTab('swarm-service-logs')" :class="['px-3 py-2 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab() === 'swarm-service-logs' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
              <i class="fas fa-file-lines"></i><span class="hidden sm:inline">日志</span>
            </button>
            <button @click="switchTab('swarm-service-tasks')" :class="['px-3 py-2 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab() === 'swarm-service-tasks' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
              <i class="fas fa-list-check"></i><span class="hidden sm:inline">任务</span>
            </button>
          </div>
        </div>
      </div>

      <!-- 内容 -->
      <div class="p-4 md:p-6 space-y-3">
        <div class="flex items-center gap-2">
          <label class="text-xs text-slate-500 flex-shrink-0">最近行数</label>
          <select v-model="logsTail" @change="loadLogs()" class="w-24 px-2 py-1 bg-white border border-slate-200 rounded-lg text-xs text-slate-700">
            <option value="50">50</option>
            <option value="100">100</option>
            <option value="200">200</option>
            <option value="500">500</option>
            <option value="1000">1000</option>
          </select>
        </div>
        <div v-if="logsLoading" class="flex flex-col items-center justify-center py-20">
          <div class="w-12 h-12 spinner mb-3"></div>
          <p class="text-slate-500">加载中...</p>
        </div>
        <pre v-else-if="logsContent.length > 0" class="bg-slate-900 text-slate-100 rounded-xl p-3 md:p-4 text-xs font-mono overflow-auto max-h-[600px] whitespace-pre-wrap break-all">{{ logsContent.join('') }}</pre>
        <div v-else class="flex flex-col items-center justify-center py-16">
          <div class="w-16 h-16 rounded-full bg-slate-100 flex items-center justify-center mb-3">
            <i class="fas fa-file-lines text-2xl text-slate-300"></i>
          </div>
          <p class="text-slate-500 text-sm">暂无日志</p>
        </div>
      </div>
    </div>
  </div>
</template>
