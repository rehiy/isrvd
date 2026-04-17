<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { ContainerInfo } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

@Component
class ContainerLogs extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    container: ContainerInfo | null = null
    logLoading = false
    logContent = ''
    logTail = '100'

    get containerId() {
        return this.$route.params.id as string
    }

    // ─── 方法 ───
    goBack() {
        this.$router.push('/docker/containers')
    }

    switchTab(name: string) {
        this.$router.push({ name, params: { id: this.containerId } })
    }

    activeTab() {
        return this.$route.name
    }

    async loadContainer() {
        try {
            const res = await api.listContainers(true)
            const list = res.payload || []
            this.container = list.find((c: ContainerInfo) => c.id === this.containerId) || null
            if (!this.container) {
                this.actions.showNotification('error', '容器不存在')
                this.$router.push('/docker/containers')
            }
        } catch (e) {
            this.actions.showNotification('error', '加载容器信息失败')
            this.$router.push('/docker/containers')
        }
    }

    async loadLogs() {
        if (!this.container) return
        this.logLoading = true
        this.logContent = ''
        try {
            const data = await api.containerLogs(this.containerId, this.logTail)
            this.logContent = (data.payload.logs || []).join('\n')
        } catch (e) {
            this.logContent = '加载日志失败'
        }
        this.logLoading = false
    }

    refreshLogs() {
        this.loadLogs()
    }

    // ─── 生命周期 ───
    async mounted() {
        await this.loadContainer()
        await this.loadLogs()
    }
}

export default toNative(ContainerLogs)
</script>

<template>
  <div>
    <!-- 顶部导航栏 -->
    <div class="card mb-4">
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3">
        <!-- 桌面端布局 -->
        <div class="hidden md:flex md:items-center justify-between">
          <div class="flex items-center gap-3">
            <button @click="goBack" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-100 flex items-center justify-center text-slate-600 transition-colors" title="返回容器列表">
              <i class="fas fa-arrow-left text-sm"></i>
            </button>
            <template v-if="container">
              <div :class="['w-9 h-9 rounded-lg flex items-center justify-center', container.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
                <i class="fas fa-cube text-white"></i>
              </div>
              <div class="min-w-0">
                <h1 class="text-lg font-semibold text-slate-800 truncate">{{ container.name || container.id }}</h1>
                <p class="text-xs text-slate-500 font-mono truncate">{{ container.image }}</p>
              </div>
            </template>
            <template v-else>
              <div class="w-9 h-9 rounded-lg bg-slate-300 flex items-center justify-center animate-pulse">
                <i class="fas fa-cube text-white"></i>
              </div>
              <div><h1 class="text-lg font-semibold text-slate-800">加载中...</h1></div>
            </template>
          </div>
          <div v-if="container" class="flex gap-1 bg-slate-100 p-1 rounded-lg">
            <button @click="switchTab('docker-container-stats')" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab() === 'docker-container-stats' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
              <i class="fas fa-chart-line"></i><span>监控</span>
            </button>
            <button @click="switchTab('docker-container-logs')" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab() === 'docker-container-logs' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
              <i class="fas fa-file-lines"></i><span>日志</span>
            </button>
            <button @click="switchTab('docker-container-terminal')" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab() === 'docker-container-terminal' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
              <i class="fas fa-terminal"></i><span>终端</span>
            </button>
          </div>
        </div>
        <!-- 移动端布局 -->
        <div class="block md:hidden">
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-3">
              <button @click="goBack" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-100 flex items-center justify-center text-slate-600 transition-colors">
                <i class="fas fa-arrow-left text-sm"></i>
              </button>
              <template v-if="container">
                <div :class="['w-9 h-9 rounded-lg flex items-center justify-center', container.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
                  <i class="fas fa-cube text-white"></i>
                </div>
                <div class="min-w-0">
                  <h1 class="text-lg font-semibold text-slate-800 truncate">{{ container.name || container.id }}</h1>
                  <p class="text-xs text-slate-500 font-mono truncate">{{ container.image }}</p>
                </div>
              </template>
              <template v-else>
                <div class="w-9 h-9 rounded-lg bg-slate-300 flex items-center justify-center animate-pulse">
                  <i class="fas fa-cube text-white"></i>
                </div>
                <div><h1 class="text-lg font-semibold text-slate-800">加载中...</h1></div>
              </template>
            </div>
          </div>
          <div v-if="container" class="flex justify-center gap-1 bg-slate-100 p-1 rounded-lg">
            <button @click="switchTab('docker-container-stats')" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab() === 'docker-container-stats' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
              <i class="fas fa-chart-line"></i><span class="hidden sm:inline">监控</span>
            </button>
            <button @click="switchTab('docker-container-logs')" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab() === 'docker-container-logs' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
              <i class="fas fa-file-lines"></i><span class="hidden sm:inline">日志</span>
            </button>
            <button @click="switchTab('docker-container-terminal')" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab() === 'docker-container-terminal' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
              <i class="fas fa-terminal"></i><span class="hidden sm:inline">终端</span>
            </button>
          </div>
        </div>
      </div>
      <!-- 内容区域 -->
      <div class="p-4 md:p-6">
        <div class="flex flex-col md:flex-row md:items-center justify-between gap-3 mb-4">
          <div class="flex items-center gap-3">
            <label class="text-sm text-slate-600">显示行数</label>
            <select v-model="logTail" class="w-28 px-3 py-1.5 bg-white border border-slate-200 rounded-lg text-xs text-slate-700 hover:border-slate-300 focus:outline-none focus:border-slate-400">
              <option value="50">50 行</option>
              <option value="100">100 行</option>
              <option value="200">200 行</option>
              <option value="500">500 行</option>
              <option value="1000">1000 行</option>
            </select>
          </div>
          <button @click="refreshLogs" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
            <i class="fas fa-rotate"></i>刷新
          </button>
        </div>
        <div v-if="logLoading" class="flex flex-col items-center justify-center py-12 gap-3 text-slate-400 text-sm">
          <div class="w-8 h-8 spinner"></div>
          <span>加载日志中...</span>
        </div>
        <pre v-else class="bg-slate-900 text-green-400 p-3 md:p-4 rounded-xl overflow-auto max-h-[70vh] text-xs md:text-sm font-mono whitespace-pre-wrap">{{ logContent || '暂无日志' }}</pre>
      </div>
    </div>
  </div>
</template>
