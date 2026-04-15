<script setup>
import { inject, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'

import { formatTime } from '@/helper/utils.js'
import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

const route = useRoute()
const actions = inject(APP_ACTIONS_KEY)

const serviceId = route.params.id
const tasks = ref([])
const tasksLoading = ref(false)

const taskStateClass = (state) => {
  if (state === 'running') return 'bg-emerald-100 text-emerald-700'
  if (state === 'failed' || state === 'rejected') return 'bg-red-100 text-red-700'
  if (state === 'complete') return 'bg-blue-100 text-blue-700'
  if (state === 'shutdown') return 'bg-slate-100 text-slate-500'
  return 'bg-amber-100 text-amber-700'
}

const loadTasks = async () => {
  tasksLoading.value = true
  try {
    const res = await api.swarmListTasks(serviceId)
    tasks.value = res.payload || []
  } catch (e) {
    actions.showNotification('error', '获取任务列表失败')
  }
  tasksLoading.value = false
}

onMounted(() => loadTasks())
</script>

<template>
  <div>
    <div v-if="tasksLoading" class="flex flex-col items-center justify-center py-20">
      <div class="w-12 h-12 spinner mb-3"></div>
      <p class="text-slate-500">加载中...</p>
    </div>
    <div v-else-if="tasks.length > 0" class="overflow-x-auto">
      <table class="w-full border-collapse">
        <thead>
          <tr class="bg-slate-50 border-b border-slate-200">
            <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">任务 ID</th>
            <th class="w-16 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">Slot</th>
            <th class="w-28 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">状态</th>
            <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">节点</th>
            <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">消息</th>
            <th class="w-40 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">更新时间</th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-slate-100">
          <tr v-for="t in tasks" :key="t.id" class="hover:bg-slate-50 transition-colors">
            <td class="px-4 py-3 font-mono text-xs text-slate-500">{{ t.id.slice(0, 12) }}</td>
            <td class="px-4 py-3 text-sm text-slate-600">{{ t.slot || '-' }}</td>
            <td class="px-4 py-3">
              <span :class="taskStateClass(t.state)" class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize">{{ t.state }}</span>
            </td>
            <td class="px-4 py-3 font-mono text-xs text-slate-500">{{ t.nodeID ? t.nodeID.slice(0, 12) : '-' }}</td>
            <td class="px-4 py-3 text-xs text-slate-500">
              <span v-if="t.err" class="text-red-500">{{ t.err }}</span>
              <span v-else>{{ t.message || '-' }}</span>
            </td>
            <td class="px-4 py-3 text-xs text-slate-400">{{ formatTime(t.updatedAt) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-else class="flex flex-col items-center justify-center py-20">
      <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
        <i class="fas fa-list-check text-4xl text-slate-300"></i>
      </div>
      <p class="text-slate-600 font-medium">暂无任务</p>
    </div>
  </div>
</template>
