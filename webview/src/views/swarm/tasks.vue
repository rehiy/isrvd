<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import { formatTime } from '@/helper/utils'
import api from '@/service/api'
import type { SwarmTask, SwarmService } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

@Component
class Tasks extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    tasks: SwarmTask[] = []
    services: SwarmService[] = []
    tasksLoading = false
    selectedServiceId = ''
    formatTime = formatTime

    // ─── 计算属性 ───
    get filteredTasks() {
        if (!this.selectedServiceId) return this.tasks
        return this.tasks.filter((t: SwarmTask) => t.serviceID === this.selectedServiceId)
    }

    // ─── 方法 ───
    taskStateClass(state: string) {
        if (state === 'running') return 'bg-emerald-100 text-emerald-700'
        if (state === 'failed' || state === 'rejected') return 'bg-red-100 text-red-700'
        if (state === 'complete') return 'bg-blue-100 text-blue-700'
        if (state === 'shutdown') return 'bg-slate-100 text-slate-500'
        return 'bg-amber-100 text-amber-700'
    }

    async loadServices() {
        try {
            const res = await api.swarmListServices()
            this.services = res.payload || []
        } catch (e) {
            this.actions.showNotification('error', '获取服务列表失败')
        }
    }

    async loadTasks() {
        this.tasksLoading = true
        try {
            const res = await api.swarmListTasks()
            this.tasks = res.payload || []
        } catch (e) {
            this.actions.showNotification('error', '获取任务列表失败')
        }
        this.tasksLoading = false
    }

    goServiceDetail(serviceId: string) {
        this.$router.push({ name: 'swarm-service-info', params: { id: serviceId } })
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
  <div>
    <div class="card mb-4">
      <!-- Toolbar -->
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-emerald-500 flex items-center justify-center">
              <i class="fas fa-list-check text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">任务列表</h1>
              <p class="text-xs text-slate-500">查看 Swarm 集群任务状态</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <select v-model="selectedServiceId" class="px-3 py-1.5 bg-white border border-slate-200 rounded-lg text-xs text-slate-700 min-w-[160px]">
              <option value="">全部服务</option>
              <option v-for="s in services" :key="s.id" :value="s.id">{{ s.name }}</option>
            </select>
            <button @click="loadTasks()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-rotate"></i>刷新
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="block md:hidden">
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-3 min-w-0 flex-1">
              <div class="w-9 h-9 rounded-lg bg-emerald-500 flex items-center justify-center flex-shrink-0">
                <i class="fas fa-list-check text-white"></i>
              </div>
              <div class="min-w-0">
                <h1 class="text-lg font-semibold text-slate-800 truncate">任务列表</h1>
                <p class="text-xs text-slate-500 truncate">查看 Swarm 集群任务状态</p>
              </div>
            </div>
            <button @click="loadTasks()" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 flex items-center justify-center text-slate-600 transition-colors flex-shrink-0">
              <i class="fas fa-rotate text-sm"></i>
            </button>
          </div>
          <select v-model="selectedServiceId" class="w-full px-3 py-2 bg-white border border-slate-200 rounded-lg text-xs text-slate-700">
            <option value="">全部服务</option>
            <option v-for="s in services" :key="s.id" :value="s.id">{{ s.name }}</option>
          </select>
        </div>
      </div>

      <!-- 内容 -->
      <div v-if="tasksLoading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
      <div v-else-if="filteredTasks.length > 0">
        <!-- 桌面端表格视图 -->
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">任务 ID</th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">服务</th>
                <th class="w-16 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">Slot</th>
                <th class="w-28 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">状态</th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">节点</th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">消息</th>
                <th class="w-40 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">更新时间</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="t in filteredTasks" :key="t.id" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3 font-mono text-xs text-slate-500">{{ t.id.slice(0, 12) }}</td>
                <td class="px-4 py-3">
                  <button @click="goServiceDetail(t.serviceID)" class="text-xs text-emerald-600 hover:text-emerald-700 hover:underline">
                    {{ t.serviceName || t.serviceID?.slice(0, 12) }}
                  </button>
                </td>
                <td class="px-4 py-3 text-sm text-slate-600">{{ t.slot || '-' }}</td>
                <td class="px-4 py-3">
                  <span :class="taskStateClass(t.state)" class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize">{{ t.state }}</span>
                </td>
                <td class="px-4 py-3">
                  <button v-if="t.nodeID" @click="goNodeDetail(t.nodeID)" class="font-mono text-xs text-blue-600 hover:text-blue-700 hover:underline">
                    {{ t.nodeID.slice(0, 12) }}
                  </button>
                  <span v-else class="text-xs text-slate-400">-</span>
                </td>
                <td class="px-4 py-3 text-xs text-slate-500">
                  <span v-if="t.err" class="text-red-500">{{ t.err }}</span>
                  <span v-else>{{ t.message || '-' }}</span>
                </td>
                <td class="px-4 py-3 text-xs text-slate-400">{{ formatTime(t.updatedAt) }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 移动端卡片视图 -->
        <div class="md:hidden space-y-3 p-4">
          <div v-for="t in filteredTasks" :key="t.id" class="rounded-xl border border-slate-200 bg-white p-4 transition-all hover:shadow-sm">
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-3 min-w-0 flex-1">
                <div :class="['w-10 h-10 rounded-lg flex items-center justify-center flex-shrink-0', taskStateClass(t.state).includes('emerald') ? 'bg-emerald-400' : taskStateClass(t.state).includes('red') ? 'bg-red-400' : 'bg-slate-400']">
                  <i class="fas fa-list-check text-white text-base"></i>
                </div>
                <div class="min-w-0">
                  <code class="font-mono text-xs text-slate-500">{{ t.id.slice(0, 12) }}</code>
                  <div class="flex items-center gap-2 mt-1">
                    <span class="text-xs text-slate-500">Slot: {{ t.slot || '-' }}</span>
                    <span :class="taskStateClass(t.state)" class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize">{{ t.state }}</span>
                  </div>
                </div>
              </div>
            </div>
            <div class="grid grid-cols-2 gap-3 mb-3">
              <div>
                <p class="text-xs text-slate-500 mb-1">服务</p>
                <button @click="goServiceDetail(t.serviceID)" class="text-xs text-emerald-600 hover:text-emerald-700 hover:underline truncate">
                  {{ t.serviceName || t.serviceID?.slice(0, 12) }}
                </button>
              </div>
              <div>
                <p class="text-xs text-slate-500 mb-1">节点</p>
                <button v-if="t.nodeID" @click="goNodeDetail(t.nodeID)" class="text-xs text-blue-600 hover:text-blue-700 hover:underline font-mono">
                  {{ t.nodeID.slice(0, 12) }}
                </button>
                <span v-else class="text-xs text-slate-400">-</span>
              </div>
              <div class="col-span-2">
                <p class="text-xs text-slate-500 mb-1">更新时间</p>
                <span class="text-xs text-slate-400">{{ formatTime(t.updatedAt) }}</span>
              </div>
            </div>
            <div v-if="t.err || t.message" class="pt-2 border-t border-slate-100">
              <p class="text-xs text-slate-500 mb-1">消息</p>
              <span class="text-xs break-words" :class="t.err ? 'text-red-500' : 'text-slate-500'">{{ t.err || t.message }}</span>
            </div>
          </div>
        </div>
      </div>
      <div v-else class="flex flex-col items-center justify-center py-20">
        <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
          <i class="fas fa-list-check text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium">暂无任务</p>
      </div>
    </div>
  </div>
</template>
