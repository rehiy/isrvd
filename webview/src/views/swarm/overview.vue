<script setup>
import { inject, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

const router = useRouter()
const actions = inject(APP_ACTIONS_KEY)

const swarmInfo = ref(null)
const infoLoading = ref(false)

const statCards = [
  { key: 'nodes',    label: '总节点数',  icon: 'fa-server',       color: 'bg-blue-500' },
  { key: 'managers', label: 'Manager',   icon: 'fa-crown',        color: 'bg-indigo-500' },
  { key: 'workers',  label: 'Worker',    icon: 'fa-circle-nodes', color: 'bg-slate-500' },
  { key: 'services', label: '服务总数',  icon: 'fa-cubes',        color: 'bg-emerald-500' },
  { key: 'tasks',    label: '任务总数',  icon: 'fa-tasks',        color: 'bg-amber-500' },
]

const loadInfo = async () => {
  infoLoading.value = true
  try {
    const res = await api.swarmInfo()
    swarmInfo.value = res.payload || {}
  } catch (e) {
    actions.showNotification('error', '获取 Swarm 信息失败，请确认集群已初始化')
    swarmInfo.value = null
  }
  infoLoading.value = false
}

onMounted(() => loadInfo())
</script>

<template>
  <div>
    <div class="card mb-4">
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-6 py-3 flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="w-9 h-9 rounded-lg bg-cyan-600 flex items-center justify-center">
            <i class="fas fa-tachometer-alt text-white"></i>
          </div>
          <div>
            <h1 class="text-lg font-semibold text-slate-800">Swarm 概览</h1>
            <p class="text-xs text-slate-500">集群状态与统计信息</p>
          </div>
        </div>
        <button @click="loadInfo" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
          <i class="fas fa-rotate"></i>刷新
        </button>
      </div>

      <div v-if="infoLoading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
      <div v-else-if="swarmInfo" class="p-6">
        <div class="grid grid-cols-2 md:grid-cols-5 gap-4 mb-6">
          <div
            v-for="card in statCards" :key="card.key"
            class="relative overflow-hidden rounded-xl border border-slate-200 bg-white p-5 hover:shadow-md transition-shadow"
          >
            <div class="flex items-center gap-4">
              <div :class="['w-12 h-12 rounded-xl flex items-center justify-center', card.color]">
                <i :class="['fas', card.icon, 'text-white text-lg']"></i>
              </div>
              <div>
                <p class="text-sm text-slate-500">{{ card.label }}</p>
                <p class="text-2xl font-bold text-slate-800 mt-1">{{ swarmInfo[card.key] ?? 0 }}</p>
              </div>
            </div>
          </div>
        </div>
        <div class="pt-4 border-t border-slate-200">
          <div class="grid grid-cols-2 gap-4 text-sm text-slate-600">
            <div><span class="text-xs text-slate-400">Cluster ID</span><p class="font-mono text-xs mt-0.5 truncate">{{ swarmInfo.clusterID || '-' }}</p></div>
            <div><span class="text-xs text-slate-400">创建时间</span><p class="mt-0.5">{{ swarmInfo.createdAt || '-' }}</p></div>
          </div>
        </div>
        <div class="mt-4 flex flex-wrap gap-2">
          <button @click="router.push('/swarm/nodes')" class="px-4 py-2 rounded-lg bg-blue-50 hover:bg-blue-100 text-blue-700 text-sm font-medium flex items-center gap-2 transition-colors">
            <i class="fas fa-server"></i>节点管理
          </button>
          <button @click="router.push('/swarm/services')" class="px-4 py-2 rounded-lg bg-emerald-50 hover:bg-emerald-100 text-emerald-700 text-sm font-medium flex items-center gap-2 transition-colors">
            <i class="fas fa-cubes"></i>服务管理
          </button>
          <button @click="router.push('/swarm/tasks')" class="px-4 py-2 rounded-lg bg-amber-50 hover:bg-amber-100 text-amber-700 text-sm font-medium flex items-center gap-2 transition-colors">
            <i class="fas fa-tasks"></i>任务列表
          </button>
        </div>
      </div>
      <div v-else class="flex flex-col items-center justify-center py-20">
        <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
          <i class="fas fa-circle-nodes text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">Swarm 集群未初始化</p>
        <p class="text-sm text-slate-400">请先在 Docker 主机上执行 <code class="bg-slate-100 px-1.5 rounded">docker swarm init</code></p>
      </div>
    </div>
  </div>
</template>
