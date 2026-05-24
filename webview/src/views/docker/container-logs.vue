<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { DockerContainerInfo } from '@/service/types'

import * as ContainerLogsStream from '@/helper/container-logs'

const MAX_LOG_LENGTH = 300000

@Component
class ContainerLogs extends Vue {
    portal = usePortal()

    @Ref readonly logRef!: HTMLPreElement

    // ─── 数据属性 ───
    container: DockerContainerInfo | null = null
    logLoading = false
    logContent = ''
    logTail = '100'
    streamActive = false
    streamConnected = false

    get containerId() {
        return this.$route.params.id as string
    }

    // ─── 方法 ───
    async loadContainer() {
        try {
            const res = await api.dockerContainerList(true)
            this.container = (res.payload || []).find((c: DockerContainerInfo) => c.id === this.containerId) ?? null
            if (!this.container) {
                this.portal.showNotification('error', '容器不存在')
                this.$router.push('/docker/containers')
                return
            }
            this.loadLogs()
        } catch {
            this.portal.showNotification('error', '加载容器信息失败')
        }
    }

    async loadLogs() {
        if (!this.container) return
        this.stopStream()
        this.logLoading = true
        this.logContent = ''
        try {
            const res = await api.dockerContainerLogs(this.containerId, this.logTail)
            this.logContent = (res.payload?.logs || []).join('')
        } catch {
            this.logContent = '加载日志失败'
        }
        this.logLoading = false
    }

    startStream() {
        if (!this.container || this.streamActive) return
        this.logLoading = false
        this.logContent = ''
        this.streamActive = true
        this.streamConnected = false
        ContainerLogsStream.create(this.portal.token ?? '', this.containerId, this.logTail, {
            onOpen: () => {
                this.streamConnected = true
            },
            onMessage: (data: string) => this.appendLog(data),
            onClose: () => {
                this.streamConnected = false
                this.streamActive = false
            },
            onError: () => {
                ContainerLogsStream.destroy()
                this.streamConnected = false
                this.streamActive = false
                this.portal.showNotification('error', '实时日志连接失败')
            }
        })
    }

    stopStream() {
        ContainerLogsStream.destroy()
        this.streamActive = false
        this.streamConnected = false
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
        this.logContent += data
        if (this.logContent.length > MAX_LOG_LENGTH) {
            this.logContent = this.logContent.slice(-MAX_LOG_LENGTH)
        }
        this.$nextTick(() => this.scrollToBottom())
    }

    scrollToBottom() {
        if (!this.logRef) return
        this.logRef.scrollTop = this.logRef.scrollHeight
    }

    getStreamStatusText() {
        if (!this.streamActive) return '快照'
        return this.streamConnected ? '实时中' : '连接中'
    }

    getStreamStatusClass() {
        if (!this.streamActive) return 'text-slate-500'
        return this.streamConnected ? 'text-emerald-600 font-medium' : 'text-amber-600 font-medium'
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadContainer()
    }

    unmounted() {
        this.stopStream()
    }
}

export default toNative(ContainerLogs)
</script>

<template>
  <div>
    <div class="card mb-4">
      <!-- Toolbar -->
      <div class="card-toolbar">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div :class="['page-icon', container?.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
              <i class="fas fa-file-lines text-white text-sm"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">容器日志</h1>
              <p class="text-xs text-slate-500 font-mono truncate max-w-xs">{{ container ? `${container.name || container.id} · ${container.image}` : '加载中...' }}</p>
            </div>
          </div>
          <div class="flex items-center gap-2 flex-shrink-0">
            <span :class="getStreamStatusClass()" class="text-xs">{{ getStreamStatusText() }}</span>
            <label class="text-xs text-slate-500">显示行数</label>
            <select v-model="logTail" class="w-28 select-sm" @change="handleTailChange">
              <option value="50">50 行</option>
              <option value="100">100 行</option>
              <option value="200">200 行</option>
              <option value="500">500 行</option>
              <option value="1000">1000 行</option>
            </select>
            <button class="btn btn-secondary" :disabled="streamActive" @click="loadLogs">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button v-if="!streamActive && portal.hasPerm('GET /api/docker/container/:id/logs/stream')" class="btn btn-emerald" @click="startStream">
              <i class="fas fa-play"></i>实时
            </button>
            <button v-else-if="streamActive" class="btn btn-secondary" @click="stopStream">
              <i class="fas fa-stop"></i>停止
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div :class="['page-icon', container?.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
              <i class="fas fa-file-lines text-white text-sm"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">容器日志</h1>
              <p class="text-xs text-slate-500 font-mono truncate">{{ container ? `${container.name || container.id} · ${container.image}` : '加载中...' }}</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <select v-model="logTail" class="select-sm" @change="handleTailChange">
              <option value="50">50</option>
              <option value="100">100</option>
              <option value="200">200</option>
              <option value="500">500</option>
              <option value="1000">1000</option>
            </select>
            <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" :disabled="streamActive" @click="loadLogs">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button v-if="!streamActive && portal.hasPerm('GET /api/docker/container/:id/logs/stream')" class="btn btn-emerald w-9 h-9 !px-0" title="实时" @click="startStream">
              <i class="fas fa-play text-sm"></i>
            </button>
            <button v-else-if="streamActive" class="btn btn-secondary w-9 h-9 !px-0" title="停止" @click="stopStream">
              <i class="fas fa-stop text-sm"></i>
            </button>
          </div>
        </div>
      </div>

      <!-- 内容区域 -->
      <div class="p-4 md:p-6">
        <div v-if="logLoading" class="empty-state">
          <div class="w-12 h-12 spinner mb-3"></div>
          <p class="text-slate-500">加载中...</p>
        </div>
        <pre v-else-if="logContent || streamActive" ref="logRef" class="bg-slate-900 text-slate-100 rounded-xl p-3 md:p-4 text-xs font-mono overflow-auto max-h-[600px] whitespace-pre-wrap break-all">{{ logContent || '等待日志输出...' }}</pre>
        <div v-else class="empty-state">
          <div class="empty-state-icon">
            <i class="fas fa-file-lines text-2xl text-slate-300"></i>
          </div>
          <p class="text-slate-500 text-sm">暂无日志</p>
        </div>
      </div>
    </div>
  </div>
</template>
