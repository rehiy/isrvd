<script setup>
import { inject, onMounted, ref } from 'vue'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

const actions = inject(APP_ACTIONS_KEY)

const tasks = ref([])
const tasksLoading = ref(false)
const filterServiceID = ref('')
const services = ref([])

const taskStateClass = (state) => {
  if (state === 'running')  return 'bg-emerald-100 text-emerald-700'
  if (state === 'failed')   return 'bg-red-100 text-red-700'
  if (state === 'complete') return 'bg-blue-100 text-blue-700'
  if (state === 'shutdown') return 'bg-slate-100 text-slate-500'
  return 'bg-amber-100 text-amber-700'
}

const loadTasks = async () => {
  tasksLoading.value = true
  try {
    const res = await api.swarmListTasks(filterServiceID.value)
    tasks.value = res.payload || []
  } catch (e) {
    actions.showNotification('error', '获取任务列表失败')
  }
  tasksLoading.value = false
}

const loadServices = async () => {
  try {
    const res = await api.swarmListServices()
    services.value = res.payload || []
  } catch (e) {}
}

onMounted(async () => {
  await loadServices()
  loadTasks()
})
</script>

<template>
  <div>
    <div class="card mb-4">
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-6 py-3 flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="w-9 h-9 rounded-lg bg-amber-500 flex items-center justify-center">
            <i class="fas fa-tasks text-white"></i>
          </div>
          <div>
            <h1 class="text-lg font-semibold text-slate-800">任务列表</h1>
            <p class="text-xs text-slate-500">查看 Swarm 集群任务</p>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <select v-model="filterServiceID" @change="loadTasks" class="w-32 md:w-44 px-3 py-1.5 bg-white border border-slate-200 rounded-lg text-xs text-slate-700 hover:border-slate-300 focus:outline-none">
            <option value="">全部服务</option>
            <option v-for="s in services" :key="s.id" :value="s.id">{{ s.name }}</option>
          </select>
          <button @click="loadTasks" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
            <i class="fas fa-rotate"></i>刷新
          </button>
        </div>
      </div>

      <div v-if="tasksLoading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
      <div v-else-if="tasks.length > 0">
        <!-- 桌面端表格视图 -->
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">服务名</th>
                <th class="w-16 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">Slot</th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">镜像</th>
                <th class="w-28 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">状态</th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">消息</th>
                <th class="w-40 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">更新时间</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="t in tasks" :key="t.id" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3 font-medium text-slate-800">{{ t.serviceName || t.serviceID }}</td>
                <td class="px-4 py-3 text-sm text-slate-500">{{ t.slot || '-' }}</td>
                <td class="px-4 py-3"><code class="text-xs bg-slate-100 px-2 py-1 rounded">{{ t.image }}</code></td>
                <td class="px-4 py-3">
                  <span :class="taskStateClass(t.state)" class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize">{{ t.state }}</span>
                </td>
                <td class="px-4 py-3 text-xs text-slate-500 max-w-xs truncate" :title="t.err || t.message">{{ t.err || t.message || '-' }}</td>
                <td class="px-4 py-3 text-xs text-slate-500 whitespace-nowrap">{{ t.updatedAt }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 移动端卡片视图 -->
        <div class="md:hidden space-y-3">
          <div v-for="t in tasks" :key="t.id" class="rounded-xl border border-slate-200 bg-white p-4 transition-all hover:shadow-sm">
            <!-- 顶部：服务名和状态 -->
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-amber-400 flex items-center justify-center">
                  <i class="fas fa-tasks text-white text-base"></i>
                </div>
                <div class="min-w-0">
                  <span class="font-medium text-slate-800 text-sm">{{ t.serviceName || t.serviceID }}</span>
                  <div class="flex items-center gap-2 mt-1">
                    <span class="text-xs text-slate-500">Slot: {{ t.slot || '-' }}</span>
                    <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize" :class="taskStateClass(t.state)">{{ t.state }}</span>
                  </div>
                </div>
              </div>
            </div>
            
            <!-- 镜像信息 -->
            <div class="mb-3">
              <p class="text-xs text-slate-500 mb-1">镜像</p>
              <code class="text-xs bg-slate-100 px-2 py-1 rounded break-all">{{ t.image }}</code>
            </div>
            
            <!-- 消息和时间 -->
            <div class="grid grid-cols-2 gap-3">
              <div v-if="t.err || t.message" class="col-span-2">
                <p class="text-xs text-slate-500 mb-1">消息</p>
                <span class="text-xs break-words" :class="t.err ? 'text-red-500' : 'text-slate-500'">{{ t.err || t.message }}</span>
              </div>
              <div>
                <p class="text-xs text-slate-500 mb-1">更新时间</p>
                <span class="text-xs text-slate-500">{{ t.updatedAt }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div v-else class="flex flex-col items-center justify-center py-20">
        <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
          <i class="fas fa-tasks text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">暂无任务</p>
      </div>
    </div>
  </div>
</template>
