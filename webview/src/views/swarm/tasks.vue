<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { SwarmTask, SwarmServiceInfo } from '@/service/types'

import { formatTime } from '@/helper/utils'

import PageSearch from '@/component/page-search.vue'

@Component({
    components: { PageSearch }
})
class Tasks extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    tasks: SwarmTask[] = []
    services: SwarmServiceInfo[] = []
    tasksLoading = false
    selectedServiceId = ''
    searchText = ''
    formatTime = formatTime

    // ─── 计算属性 ───
    get filteredTasks() {
        let list = this.tasks
        if (this.selectedServiceId) {
            list = list.filter((t: SwarmTask) => t.serviceID === this.selectedServiceId)
        }
        // 按更新时间降序排列（最新在前）
        list = [...list].sort((a: SwarmTask, b: SwarmTask) =>
            new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime()
        )
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return list
        return list.filter((t: SwarmTask) =>
            (t.id || '').toLowerCase().includes(keyword) ||
            (t.serviceName || '').toLowerCase().includes(keyword) ||
            (t.serviceID || '').toLowerCase().includes(keyword) ||
            (t.nodeName || '').toLowerCase().includes(keyword) ||
            (t.nodeID || '').toLowerCase().includes(keyword) ||
            (t.state || '').toLowerCase().includes(keyword) ||
            (t.message || '').toLowerCase().includes(keyword) ||
            (t.err || '').toLowerCase().includes(keyword)
        )
    }

    // ─── 方法 ───
    taskStateClass(state: string) {
        if (state === 'running' || state === 'complete') return 'text-emerald-600 font-medium'
        if (state === 'failed' || state === 'rejected') return 'text-red-500 font-medium'
        if (state === 'pending' || state === 'preparing' || state === 'starting') return 'text-amber-600 font-medium'
        return 'text-slate-500'
    }

    taskIconClass(state: string) {
        if (state === 'running') return 'bg-cyan-400'
        if (state === 'failed' || state === 'rejected') return 'bg-red-400'
        if (state === 'complete') return 'bg-blue-400'
        if (state === 'shutdown') return 'bg-slate-400'
        return 'bg-amber-400'
    }

    async loadServices() {
        try {
            const res = await api.swarmServiceList()
            this.services = res.payload || []
        } finally {
        }
    }

    async loadTasks() {
        this.tasksLoading = true
        try {
            const res = await api.swarmTaskList()
            this.tasks = res.payload || []
        } finally {
            this.tasksLoading = false
        }
    }

    goServiceDetail(serviceId: string) {
        this.$router.push({ name: 'swarm-service', params: { id: serviceId } })
    }

    goNodeDetail(nodeId: string) {
        this.$router.push({ name: 'swarm-node', params: { id: nodeId } })
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadServices()
        this.loadTasks()
    }
}

export default toNative(Tasks)
</script>

<template>
  <div class="card">
    <!-- Toolbar -->
    <div class="card-toolbar">
      <!-- 桌面端 -->
      <div class="hidden md:flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-cyan-500">
            <i class="fas fa-list-check text-white"></i>
          </div>
          <div>
            <h1 class="text-lg font-semibold text-slate-800 truncate">任务列表</h1>
            <p class="text-xs text-slate-500">查看 Swarm 集群任务状态</p>
          </div>
        </div>
        <div class="flex items-center gap-2 flex-shrink-0">
          <PageSearch v-model="searchText" search-key="swarm-tasks" placeholder="请输入搜索关键词..." focus-color="cyan" type-to-search />
          <select v-model="selectedServiceId" class="select-sm min-w-[160px]">
            <option value="">全部服务</option>
            <option v-for="s in services" :key="s.id" :value="s.id">{{ s.name }}</option>
          </select>
          <button class="btn btn-secondary" @click="loadTasks()">
            <i class="fas fa-rotate"></i>刷新
          </button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="block md:hidden">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="page-icon bg-cyan-500">
              <i class="fas fa-list-check text-white"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">任务列表</h1>
              <p class="text-xs text-slate-500 truncate">查看 Swarm 集群任务状态</p>
            </div>
          </div>
          <div class="flex items-center gap-1.5 flex-shrink-0">
            <select v-model="selectedServiceId" class="w-28 select-sm">
              <option value="">全部服务</option>
              <option v-for="s in services" :key="s.id" :value="s.id">{{ s.name }}</option>
            </select>
            <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadTasks()">
              <i class="fas fa-rotate text-sm"></i>
            </button>
          </div>
        </div>
      </div>
    </div>
    <div class="mobile-search">
      <PageSearch v-model="searchText" search-key="swarm-tasks" placeholder="请输入搜索关键词..." width-class="w-full" focus-color="emerald" />
    </div>

    <!-- 内容 -->
    <div v-if="tasksLoading" class="card-body">
      <div class="empty-state">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
    </div>
    <template v-else-if="filteredTasks.length > 0">
      <!-- 桌面端表格视图 -->
      <div class="card-table hidden md:block">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-50 border-b border-slate-200">
              <th class="th">任务 ID</th>
              <th class="th">服务</th>
              <th class="w-16 th">Slot</th>
              <th class="w-28 th">状态</th>
              <th class="th">消息</th>
              <th class="w-36 th">节点</th>
              <th class="w-52 th">更新时间</th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-slate-100">
            <tr v-for="t in filteredTasks" :key="t.id" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3"><code class="text-xs text-slate-600 font-mono">{{ t.id.slice(0, 12) }}</code></td>
              <td class="px-4 py-3">
                <button class="text-xs text-emerald-600 hover:text-emerald-700 hover:underline" @click="goServiceDetail(t.serviceID)">
                  {{ t.serviceName || t.serviceID?.slice(0, 12) }}
                </button>
              </td>
              <td class="px-4 py-3 text-sm text-slate-600">{{ t.slot || '-' }}</td>
              <td class="px-4 py-3">
                <span :class="taskStateClass(t.state)" class="text-sm capitalize">{{ t.state }}</span>
              </td>
              <td class="px-4 py-3 text-xs text-slate-500">
                <span v-if="t.err" class="text-red-500">{{ t.err }}</span>
                <span v-else>{{ t.message || '-' }}</span>
              </td>
              <td class="px-4 py-3 whitespace-nowrap">
                <button v-if="t.nodeID" class="text-xs text-blue-600 hover:text-blue-700 hover:underline" @click="goNodeDetail(t.nodeID)">
                  {{ t.nodeName || t.nodeID.slice(0, 12) }}
                </button>
                <span v-else class="text-xs text-slate-400">-</span>
              </td>
              <td class="px-4 py-3 text-sm text-slate-600">{{ formatTime(t.updatedAt) }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 移动端卡片视图 -->
      <div class="card-body md:hidden space-y-3">
        <div v-for="t in filteredTasks" :key="t.id" class="card-interactive">
          <!-- 顶部：图标 + ID -->
          <div class="card-info-row">
            <div :class="['list-icon', taskIconClass(t.state)]">
              <i class="fas fa-list-check text-white text-base"></i>
            </div>
            <div class="min-w-0">
              <code class="font-mono text-xs text-slate-500">{{ t.id.slice(0, 12) }}</code>
              <span :class="taskStateClass(t.state)" class="text-xs capitalize block mt-0.5">{{ t.state }}</span>
            </div>
          </div>
          <!-- 服务 + Slot（关联：Slot 是服务副本编号） -->
          <div class="flex items-center gap-2 mb-3">
            <span class="text-xs text-slate-400 flex-shrink-0">服务</span>
            <button class="text-xs text-emerald-600 hover:text-emerald-700 hover:underline truncate" @click="goServiceDetail(t.serviceID)">
              {{ t.serviceName || t.serviceID?.slice(0, 12) }}
            </button>
            <span class="text-xs text-slate-300">|</span>
            <span class="text-xs text-slate-400 flex-shrink-0">Slot</span>
            <span class="text-xs text-slate-500">{{ t.slot || '-' }}</span>
          </div>
          <!-- 节点（独立） -->
          <div class="flex items-center gap-2 mb-3">
            <span class="text-xs text-slate-400 flex-shrink-0">节点</span>
            <button v-if="t.nodeID" class="text-xs text-blue-600 hover:text-blue-700 hover:underline" @click="goNodeDetail(t.nodeID)">
              {{ t.nodeName || t.nodeID.slice(0, 12) }}
            </button>
            <span v-else class="text-xs text-slate-400">-</span>
          </div>
          <!-- 消息（与状态关联，紧跟） -->
          <div v-if="t.err || t.message" class="flex items-start gap-2 mb-3">
            <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">消息</span>
            <span class="text-xs break-words" :class="t.err ? 'text-red-500' : 'text-slate-500'">{{ t.err || t.message }}</span>
          </div>
          <!-- 更新时间（最后） -->
          <div class="flex items-center gap-2 mb-3">
            <span class="text-xs text-slate-400 flex-shrink-0">更新</span>
            <span class="text-xs text-slate-500">{{ formatTime(t.updatedAt) }}</span>
          </div>
        </div>
      </div>
    </template>
    <div v-else class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-list-check text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">{{ tasks.length === 0 ? '暂无任务' : '未找到匹配任务' }}</p>
        <p class="text-sm text-slate-400">{{ tasks.length === 0 ? '当前没有运行中的任务' : '尝试更换关键词或清空搜索条件' }}</p>
      </div>
    </div>
  </div>
</template>
