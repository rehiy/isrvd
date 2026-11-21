<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { DockerContainerInfo } from '@/service/types'

import { LogStream } from '@/helper/log'

import LogControls from '@/component/log-controls.vue'

import ContainerTabs from '@/views/docker/widget/container-tabs.vue'

@Component({
    components: { ContainerTabs, LogControls }
})
class ContainerLogs extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    container: DockerContainerInfo | null = null
    logs = new LogStream()

    get containerId() {
        return this.$route.params.id as string
    }

    // ─── 方法 ───
    async loadContainer() {
        const res = await api.dockerContainerList(true, { id: [this.containerId] })
        this.container = (res.payload || []).find((c: DockerContainerInfo) => c.id === this.containerId) ?? null
        if (!this.container) {
            this.portal.showNotification('error', this.$t('容器不存在'))
            this.$router.push('/docker/containers')
            return
        }
        this.loadLogs()
    }

    async loadLogs() {
        if (!this.container) {
            return
        }
        await this.logs.load(async signal => (await api.dockerContainerLogs(this.containerId, this.logs.tail, signal)).payload?.logs || [])
    }

    startStream() {
        if (!this.container) {
            return
        }
        this.logs.start(
            `docker/container/${encodeURIComponent(this.containerId)}/logs/stream`,
            this.portal.token ?? '',
            message => this.portal.showNotification('error', message)
        )
    }

    handleTailChange(tail: string) {
        this.logs.tail = tail
        this.logs.reloadTail(this.loadLogs, this.startStream)
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
        <div class="title-group">
          <div :class="['page-icon', container?.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
            <i class="fas fa-file-lines text-white text-sm"></i>
          </div>
          <div class="min-w-0">
            <h1 class="title-text">{{ $t('容器日志') }}</h1>
            <p class="text-xs text-slate-500 font-mono truncate max-w-xs">{{ container ? `${container.name || container.id} · ${container.image}` : $t('加载中...') }}</p>
          </div>
        </div>
        <div class="action-group">
          <ContainerTabs :container-id="containerId" />
          <LogControls :logs="logs" :can-stream="portal.hasPerm('GET /api/docker/container/:id/logs/stream')" @tail-change="handleTailChange" @refresh="loadLogs" @start="startStream" @stop="logs.stop()" />
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
              <h1 class="title-text">{{ $t('容器日志') }}</h1>
              <p class="text-xs text-slate-500 font-mono truncate">{{ container ? `${container.name || container.id} · ${container.image}` : $t('加载中...') }}</p>
            </div>
          </div>
          <div class="action-group-sm">
            <LogControls :logs="logs" :can-stream="portal.hasPerm('GET /api/docker/container/:id/logs/stream')" mobile @tail-change="handleTailChange" @refresh="loadLogs" @start="startStream" @stop="logs.stop()" />
          </div>
        </div>
        <ContainerTabs :container-id="containerId" />
      </div>
    </div>

    <!-- 内容区域 -->
    <div class="p-4">
      <div v-if="logs.loading" class="empty-state">
        <div class="spinner-lg"></div>
        <p class="text-slate-500">{{ $t('加载中...') }}</p>
      </div>
      <pre v-else-if="logs.content || logs.active" class="min-h-[18rem] bg-white text-xs font-mono leading-relaxed text-slate-700 whitespace-pre-wrap break-all">{{ logs.content || $t('等待日志输出...') }}</pre>
      <div v-else class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-file-lines text-2xl text-slate-300"></i>
        </div>
        <p class="text-slate-500 text-sm">{{ $t('暂无日志') }}</p>
      </div>
    </div>
  </div>
</template>
