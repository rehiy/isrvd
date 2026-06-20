<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import { absUrl } from '@/service/client'
import type { DockerContainerInfo } from '@/service/types'

const MAX_LOG_LENGTH = 300000

type StreamState = 'snapshot' | 'connecting' | 'streaming'

@Component
class ContainerLogs extends Vue {
    portal = usePortal()

    @Ref readonly logRef!: HTMLPreElement

    // ─── 数据属性 ───
    container: DockerContainerInfo | null = null
    logLoading = false
    logContent = ''
    logTail = '100'
    streamState: StreamState = 'snapshot'
    private source: EventSource | null = null

    get containerId() {
        return this.$route.params.id as string
    }

    get streamActive() {
        return this.streamState !== 'snapshot'
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
        } catch {}
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
        this.streamState = 'connecting'

        // 清理旧 source，但不重置 streamState（保持 connecting 使 UI 正确显示 spinner）
        this.source?.close()
        this.source = null

        const params = new URLSearchParams({ token: this.portal.token ?? '', tail: this.logTail })
        const url = absUrl(`docker/container/${encodeURIComponent(this.containerId)}/logs/stream?${params.toString()}`)
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
        this.$nextTick(() => this.scrollToBottom())
    }

    scrollToBottom() {
        if (!this.logRef) return
        this.logRef.scrollTop = this.logRef.scrollHeight
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
  <div class="h-[calc(100vh-100px)]">
    <div class="h-full card flex flex-col overflow-hidden">
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
            <select v-model="logTail" class="w-28 select-sm" @change="handleTailChange">
              <option value="50">显示 50 行</option>
              <option value="100">显示 100 行</option>
              <option value="200">显示 200 行</option>
              <option value="500">显示 500 行</option>
              <option value="1000">显示 1000 行</option>
            </select>
            <button class="btn btn-secondary" :disabled="streamActive" @click="loadLogs">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button v-if="!streamActive && portal.hasPerm('GET /api/docker/container/:id/logs/stream')" class="btn btn-emerald" @click="startStream">
              <i class="fas fa-play"></i>实时
            </button>
            <button v-else-if="streamActive" class="btn btn-secondary" @click="stopStream">
              <i :class="streamState === 'connecting' ? 'fas fa-spinner fa-spin' : 'fas fa-stop'"></i>停止
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
              <option value="50">50 行</option>
              <option value="100">100 行</option>
              <option value="200">200 行</option>
              <option value="500">500 行</option>
              <option value="1000">1000 行</option>
            </select>
            <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" :disabled="streamActive" @click="loadLogs">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button v-if="!streamActive && portal.hasPerm('GET /api/docker/container/:id/logs/stream')" class="btn btn-emerald w-9 h-9 !px-0" title="实时" @click="startStream">
              <i class="fas fa-play text-sm"></i>
            </button>
            <button v-else-if="streamActive" class="btn btn-secondary w-9 h-9 !px-0" title="停止" @click="stopStream">
              <i :class="streamState === 'connecting' ? 'fas fa-spinner fa-spin text-sm' : 'fas fa-stop text-sm'"></i>
            </button>
          </div>
        </div>
      </div>

      <!-- 内容区域 -->
      <div class="flex-1 flex flex-col overflow-hidden p-3 md:p-4">
        <div v-if="logLoading" class="empty-state flex-1">
          <div class="w-12 h-12 spinner mb-3"></div>
          <p class="text-slate-500">加载中...</p>
        </div>
        <pre v-else-if="logContent || streamActive" ref="logRef" class="flex-1 bg-slate-900 text-slate-100 rounded-xl p-3 md:p-4 text-xs font-mono overflow-auto whitespace-pre-wrap break-all">{{ logContent || '等待日志输出...' }}</pre>
        <div v-else class="empty-state flex-1">
          <div class="empty-state-icon">
            <i class="fas fa-file-lines text-2xl text-slate-300"></i>
          </div>
          <p class="text-slate-500 text-sm">暂无日志</p>
        </div>
      </div>
    </div>
  </div>
</template>
