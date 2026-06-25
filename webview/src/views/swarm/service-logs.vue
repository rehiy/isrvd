<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import { absUrl } from '@/service/client'

const MAX_LOG_LENGTH = 300000

type StreamState = 'snapshot' | 'connecting' | 'streaming'

@Component
class ServiceLogs extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    serviceName = ''
    logContent = ''
    logLoading = false
    logTail = '100'
    streamState: StreamState = 'snapshot'
    private source: EventSource | null = null

    get serviceId() {
        return this.$route.params.id as string
    }

    get streamActive() {
        return this.streamState !== 'snapshot'
    }

    // ─── 方法 ───
    activeTab() {
        return this.$route.name
    }

    switchTab(name: string) {
        this.stopStream()
        this.$router.push({ name, params: { id: this.serviceId } })
    }

    async loadLogs() {
        this.stopStream()
        this.logLoading = true
        this.logContent = ''
        try {
            const res = await api.swarmServiceLogs(this.serviceId, this.logTail)
            this.logContent = (res.payload?.logs || []).join('')
        } catch {
            this.logContent = '加载日志失败'
        }
        this.logLoading = false
    }

    startStream() {
        if (this.streamActive) return
        this.logLoading = false
        this.logContent = ''
        this.streamState = 'connecting'

        // 清理旧 source，但不重置 streamState（保持 connecting 使 UI 正确显示 spinner）
        this.source?.close()
        this.source = null

        const params = new URLSearchParams({ token: this.portal.token ?? '', tail: this.logTail })
        const url = absUrl(`swarm/service/${encodeURIComponent(this.serviceId)}/logs/stream?${params.toString()}`)
        this.source = new EventSource(url)
        this.source.onopen = () => {
            this.streamState = 'streaming'
        }
        this.source.onmessage = event => this.appendLog(event.data)
        this.source.addEventListener('error', event => {
            const msg = (event as MessageEvent).data ?? ''
            this.stopStream()
            this.portal.showNotification('error', msg || '实时日志连接失败')
        })
        this.source.onerror = () => {
            if (this.source?.readyState === EventSource.CLOSED) {
                this.stopStream()
            }
        }
    }

    stopStream() {
        this.source?.close()
        this.source = null
        this.streamState = 'snapshot'
    }

    handleTailChange() {
        if (this.streamActive) {
            this.stopStream()
            this.startStream()
            return
        }
        this.loadLogs()
    }

    appendLog(data: string) {
        this.logContent += data + '\n'
        if (this.logContent.length > MAX_LOG_LENGTH) {
            this.logContent = this.logContent.slice(-MAX_LOG_LENGTH)
        }
        // 仅在用户已滚到底部附近（100px 内）时才自动追尾，避免打断手动上翻查看历史
        const nearBottom = document.body.scrollHeight - window.scrollY - window.innerHeight < 100
        if (nearBottom) {
            this.$nextTick(() => requestAnimationFrame(() => {
                window.scrollTo({ top: document.body.scrollHeight })
            }))
        }
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

    unmounted() {
        this.stopStream()
    }
}

export default toNative(ServiceLogs)
</script>

<template>
  <div class="page">
    <!-- Toolbar -->
    <div class="page-toolbar">
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
          <select v-model="logTail" class="w-28 select-sm" @change="handleTailChange">
            <option value="50">显示 50 行</option>
            <option value="100">显示 100 行</option>
            <option value="200">显示 200 行</option>
            <option value="500">显示 500 行</option>
            <option value="1000">显示 1000 行</option>
          </select>
          <button class="btn btn-secondary" :disabled="streamActive" @click="loadLogs()">
            <i class="fas fa-rotate"></i>刷新
          </button>
          <button v-if="!streamActive && portal.hasPerm('GET /api/swarm/service/:id/logs/stream')" class="btn btn-emerald" @click="startStream">
            <i class="fas fa-play"></i>实时
          </button>
          <button v-else-if="streamActive" class="btn btn-secondary" @click="stopStream">
            <i :class="streamState === 'connecting' ? 'fas fa-spinner fa-spin' : 'fas fa-stop'"></i>停止
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
            <select v-model="logTail" class="select-sm" @change="handleTailChange">
              <option value="50">50 行</option>
              <option value="100">100 行</option>
              <option value="200">200 行</option>
              <option value="500">500 行</option>
              <option value="1000">1000 行</option>
            </select>
            <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" :disabled="streamActive" @click="loadLogs()">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button v-if="!streamActive && portal.hasPerm('GET /api/swarm/service/:id/logs/stream')" class="btn btn-emerald w-9 h-9 !px-0" title="实时" @click="startStream">
              <i class="fas fa-play text-sm"></i>
            </button>
            <button v-else-if="streamActive" class="btn btn-secondary w-9 h-9 !px-0" title="停止" @click="stopStream">
              <i :class="streamState === 'connecting' ? 'fas fa-spinner fa-spin text-sm' : 'fas fa-stop text-sm'"></i>
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
      <div v-if="logLoading" class="empty-state">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
      <pre v-else-if="logContent || streamActive" class="bg-slate-900 text-slate-100 rounded-xl p-3 md:p-4 text-xs font-mono whitespace-pre-wrap break-all">{{ logContent || '等待日志输出...' }}</pre>
      <div v-else class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-file-lines text-2xl text-slate-300"></i>
        </div>
        <p class="text-slate-500 text-sm">暂无日志</p>
      </div>
    </div>
  </div>
</template>
