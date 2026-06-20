<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'

@Component
class ServiceLogs extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    serviceName = ''
    logsContent: string[] = []
    logsLoading = false
    logsTail = '200'

    get serviceId() {
        return this.$route.params.id as string
    }

    // ─── 方法 ───
    activeTab() {
        return this.$route.name
    }

    switchTab(name: string) {
        this.$router.push({ name, params: { id: this.serviceId } })
    }

    async loadLogs() {
        this.logsLoading = true
        try {
            const res = await api.swarmServiceLogs(this.serviceId, this.logsTail)
            this.logsContent = res.payload?.logs || []
        } catch {
            this.logsContent = []
        }
        this.logsLoading = false
    }

    async loadServiceName() {
        try {
            const res = await api.swarmServiceInspect(this.serviceId)
            this.serviceName = res.payload?.name || ''
        } catch { /* 忽略，名称仅用于展示 */ }
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadServiceName()
        this.loadLogs()
    }
}

export default toNative(ServiceLogs)
</script>

<template>
  <div class="card">
    <!-- Toolbar -->
    <div class="card-toolbar">
      <!-- 桌面端 -->
      <div class="hidden md:flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-emerald-500">
            <i class="fas fa-cubes text-white"></i>
          </div>
          <div>
            <h1 class="text-lg font-semibold text-slate-800">{{ serviceName || '服务日志' }}</h1>
            <p class="text-xs text-slate-600 font-mono truncate max-w-xs">{{ serviceId }}</p>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <div class="tab-group">
            <button v-if="portal.hasPerm('GET /api/swarm/service/:id')" :class="['tab-btn', activeTab() === 'swarm-service' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="switchTab('swarm-service')">
              <i class="fas fa-circle-info"></i><span>详情</span>
            </button>
            <button v-if="portal.hasPerm('GET /api/swarm/service/:id/logs')" :class="['tab-btn', activeTab() === 'swarm-service-logs' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="switchTab('swarm-service-logs')">
              <i class="fas fa-file-lines"></i><span>日志</span>
            </button>
          </div>
          <select v-model="logsTail" class="w-28 select-sm" @change="loadLogs()">
            <option value="50">显示 50 行</option>
            <option value="100">显示 100 行</option>
            <option value="200">显示 200 行</option>
            <option value="500">显示 500 行</option>
            <option value="1000">显示 1000 行</option>
          </select>
          <button class="btn btn-secondary" @click="loadLogs()">
            <i class="fas fa-rotate"></i>刷新
          </button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="block md:hidden">
        <div class="flex items-center justify-between mb-3">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="page-icon bg-emerald-500">
              <i class="fas fa-cubes text-white"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">{{ serviceName || '服务日志' }}</h1>
              <p class="text-xs text-slate-600 font-mono truncate">{{ serviceId.slice(0, 12) }}</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <select v-model="logsTail" class="select-sm" @change="loadLogs()">
              <option value="50">50 行</option>
              <option value="100">100 行</option>
              <option value="200">200 行</option>
              <option value="500">500 行</option>
              <option value="1000">1000 行</option>
            </select>
            <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadLogs()">
              <i class="fas fa-rotate text-sm"></i>
            </button>
          </div>
        </div>
        <div class="tab-group">
          <button v-if="portal.hasPerm('GET /api/swarm/service/:id')" :class="['tab-btn', activeTab() === 'swarm-service' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="switchTab('swarm-service')">
            <i class="fas fa-circle-info"></i><span>详情</span>
          </button>
          <button v-if="portal.hasPerm('GET /api/swarm/service/:id/logs')" :class="['tab-btn', activeTab() === 'swarm-service-logs' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="switchTab('swarm-service-logs')">
            <i class="fas fa-file-lines"></i><span>日志</span>
          </button>
        </div>
      </div>
    </div>

    <!-- 内容区域 -->
    <div class="card-body space-y-3">
      <div v-if="logsLoading" class="empty-state">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
      <pre v-else-if="logsContent.length > 0" class="bg-slate-900 text-slate-100 rounded-xl p-3 md:p-4 text-xs font-mono whitespace-pre-wrap break-all">{{ logsContent.join('') }}</pre>
      <div v-else class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-file-lines text-2xl text-slate-300"></i>
        </div>
        <p class="text-slate-500 text-sm">暂无日志</p>
      </div>
    </div>
  </div>
</template>
