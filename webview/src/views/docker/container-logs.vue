<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { DockerContainerInfo } from '@/service/types'

import { LogStream } from '@/helper/log'

@Component
class ContainerLogs extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    container: DockerContainerInfo | null = null
    logs = new LogStream()

    get containerId() {
        return this.$route.params.id as string
    }

    activeTab() {
        return this.$route.name
    }

    switchTab(name: string) {
        this.logs.stop()
        this.$router.push({ name, params: { id: this.containerId } })
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
        await this.logs.load(async () => (await api.dockerContainerLogs(this.containerId, this.logs.tail)).payload?.logs || [])
    }

    startStream() {
        if (!this.container) return
        this.logs.start(
            `docker/container/${encodeURIComponent(this.containerId)}/logs/stream`,
            this.portal.token ?? '',
            message => this.portal.showNotification('error', message)
        )
    }

    handleTailChange() {
        if (this.logs.active) {
            this.logs.stop()
            this.startStream()
            return
        }
        this.loadLogs()
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadContainer()
    }

    unmounted() {
        this.logs.stop()
    }
}

export default toNative(ContainerLogs)
</script>

<template>
  <div class="page">
    <!-- Toolbar -->
    <div class="page-toolbar">
      <!-- 桌面端 -->
      <div class="toolbar-desktop">
        <div class="flex items-center gap-3">
          <div :class="['page-icon', container?.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
            <i class="fas fa-file-lines text-white text-sm"></i>
          </div>
          <div>
            <h1 class="text-lg font-semibold text-slate-800">容器日志</h1>
            <p class="text-xs text-slate-500 font-mono truncate max-w-xs">{{ container ? `${container.name || container.id} · ${container.image}` : '加载中...' }}</p>
          </div>
        </div>
        <div class="action-group">
          <div class="tab-group">
            <button v-if="portal.hasPerm('GET /api/docker/container/:id')" :class="['tab-btn', activeTab() === 'docker-container' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="switchTab('docker-container')">
              <i class="fas fa-circle-info"></i><span>详情</span>
            </button>
            <button v-if="portal.hasPerm('GET /api/docker/container/:id/logs')" :class="['tab-btn', activeTab() === 'docker-container-logs' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="switchTab('docker-container-logs')">
              <i class="fas fa-file-lines"></i><span>日志</span>
            </button>
          </div>
          <select v-model="logs.tail" class="w-28 select-sm" @change="handleTailChange">
            <option value="50">显示 50 行</option>
            <option value="100">显示 100 行</option>
            <option value="200">显示 200 行</option>
            <option value="500">显示 500 行</option>
            <option value="1000">显示 1000 行</option>
          </select>
          <button class="btn btn-secondary" :disabled="logs.active" @click="loadLogs">
            <i class="fas fa-rotate"></i>刷新
          </button>
          <button v-if="!logs.active && portal.hasPerm('GET /api/docker/container/:id/logs/stream')" class="btn btn-emerald" @click="startStream">
            <i class="fas fa-play"></i>实时
          </button>
          <button v-else-if="logs.active" class="btn btn-secondary" @click="logs.stop()">
            <i :class="logs.state === 'connecting' ? 'fas fa-spinner fa-spin' : 'fas fa-stop'"></i>停止
          </button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="block md:hidden">
        <div class="flex items-center justify-between mb-3">
          <div class="title-group">
            <div :class="['page-icon', container?.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
              <i class="fas fa-file-lines text-white text-sm"></i>
            </div>
            <div class="min-w-0">
              <h1 class="title-text">容器日志</h1>
              <p class="text-xs text-slate-500 font-mono truncate">{{ container ? `${container.name || container.id} · ${container.image}` : '加载中...' }}</p>
            </div>
          </div>
          <div class="action-group-sm">
            <select v-model="logs.tail" class="select-sm" @change="handleTailChange">
              <option value="50">50 行</option>
              <option value="100">100 行</option>
              <option value="200">200 行</option>
              <option value="500">500 行</option>
              <option value="1000">1000 行</option>
            </select>
            <button class="btn btn-secondary btn-square" title="刷新" :disabled="logs.active" @click="loadLogs">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button v-if="!logs.active && portal.hasPerm('GET /api/docker/container/:id/logs/stream')" class="btn btn-emerald btn-square" title="实时" @click="startStream">
              <i class="fas fa-play text-sm"></i>
            </button>
            <button v-else-if="logs.active" class="btn btn-secondary btn-square" title="停止" @click="logs.stop()">
              <i :class="logs.state === 'connecting' ? 'fas fa-spinner fa-spin text-sm' : 'fas fa-stop text-sm'"></i>
            </button>
          </div>
        </div>
        <div class="tab-group">
          <button v-if="portal.hasPerm('GET /api/docker/container/:id')" :class="['tab-btn', activeTab() === 'docker-container' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="switchTab('docker-container')">
            <i class="fas fa-circle-info"></i><span>详情</span>
          </button>
          <button v-if="portal.hasPerm('GET /api/docker/container/:id/logs')" :class="['tab-btn', activeTab() === 'docker-container-logs' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="switchTab('docker-container-logs')">
            <i class="fas fa-file-lines"></i><span>日志</span>
          </button>
        </div>
      </div>
    </div>

    <!-- 内容区域 -->
    <div class="p-4">
      <div v-if="logs.loading" class="empty-state">
        <div class="spinner-lg"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
      <pre v-else-if="logs.content || logs.active" class="min-h-[18rem] bg-white text-xs font-mono leading-relaxed text-slate-700 whitespace-pre-wrap break-all">{{ logs.content || '等待日志输出...' }}</pre>
      <div v-else class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-file-lines text-2xl text-slate-300"></i>
        </div>
        <p class="text-slate-500 text-sm">暂无日志</p>
      </div>
    </div>
  </div>
</template>
