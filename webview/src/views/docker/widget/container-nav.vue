<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { DockerContainerInfo } from '@/service/types'

import { usePortal } from '@/stores'

@Component({
    emits: ['loaded']
})
class ContainerNav extends Vue {
    portal = usePortal()
    @Prop({ type: String, required: true }) readonly containerId!: string

    container: DockerContainerInfo | null = null

    get activeTab() {
        return this.$route.name
    }

    switchTab(name: string) {
        this.$router.push({ name, params: { id: this.containerId } })
    }

    goBack() {
        this.$router.push('/docker/containers')
    }

    async loadContainer() {
        try {
            const res = await api.dockerContainerList(true)
            const list = res.payload || []
            this.container = list.find(c => c.id === this.containerId) ?? null
            if (!this.container) {
                this.portal.showNotification('error', '容器不存在')
                this.$router.push('/docker/containers')
                return
            }
            this.$emit('loaded', this.container)
        } catch {
            this.portal.showNotification('error', '加载容器信息失败')
            this.$router.push('/docker/containers')
        }
    }

    mounted() {
        this.loadContainer()
    }
}

export default toNative(ContainerNav)
</script>

<template>
  <div class="card-toolbar">
    <!-- 桌面端布局 -->
    <div class="hidden md:flex md:items-center justify-between">
      <div class="flex items-center gap-3">
        <template v-if="container">
          <div :class="['page-icon', container.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
            <i class="fas fa-cube text-white text-sm"></i>
          </div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">{{ container.name || container.id }}</h1>
            <p class="text-xs text-slate-600 font-mono truncate">{{ container.image }}</p>
          </div>
        </template>
        <template v-else>
          <div class="page-icon bg-slate-300 animate-pulse">
            <i class="fas fa-cube text-white text-sm"></i>
          </div>
          <div><h1 class="text-lg font-semibold text-slate-800">加载中...</h1></div>
        </template>
      </div>
      <div v-if="container" class="flex items-center gap-2">
        <div class="flex gap-1 bg-slate-100 p-1 rounded-lg">
          <button v-if="portal.hasPerm('GET /api/docker/container/:id')" :class="['tab-btn', activeTab === 'docker-container' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="switchTab('docker-container')">
            <i class="fas fa-circle-info"></i><span>详情</span>
          </button>
          <button v-if="container.state === 'running' && portal.hasPerm('GET /api/docker/container/:id/stats')" :class="['tab-btn', activeTab === 'docker-container-stats' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="switchTab('docker-container-stats')">
            <i class="fas fa-chart-line"></i><span>监控</span>
          </button>
          <button v-if="portal.hasPerm('GET /api/docker/container/:id/logs')" :class="['tab-btn', activeTab === 'docker-container-logs' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="switchTab('docker-container-logs')">
            <i class="fas fa-file-lines"></i><span>日志</span>
          </button>
          <button v-if="container.state === 'running' && portal.hasPerm('GET /api/docker/container/:id/exec')" :class="['tab-btn', activeTab === 'docker-container-exec' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="switchTab('docker-container-exec')">
            <i class="fas fa-terminal"></i><span>终端</span>
          </button>
        </div>
      </div>
    </div>
    <!-- 移动端布局 -->
    <div class="block md:hidden">
      <div class="flex items-center justify-between mb-3">
        <div class="flex items-center gap-3 min-w-0 flex-1">
          <template v-if="container">
            <div :class="['page-icon', container.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
              <i class="fas fa-cube text-white text-sm"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">{{ container.name || container.id }}</h1>
              <p class="text-xs text-slate-600 font-mono truncate">{{ container.image }}</p>
            </div>
          </template>
          <template v-else>
            <div class="page-icon bg-slate-300 animate-pulse flex-shrink-0">
              <i class="fas fa-cube text-white text-sm"></i>
            </div>
            <div><h1 class="text-lg font-semibold text-slate-800">加载中...</h1></div>
          </template>
        </div>
      </div>
      <div v-if="container" class="flex justify-center gap-1 bg-slate-100 p-1 rounded-lg">
        <button v-if="portal.hasPerm('GET /api/docker/container/:id')" :class="['tab-btn', activeTab === 'docker-container' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="switchTab('docker-container')">
          <i class="fas fa-circle-info"></i><span>详情</span>
        </button>
        <button v-if="container.state === 'running' && portal.hasPerm('GET /api/docker/container/:id/stats')" :class="['tab-btn', activeTab === 'docker-container-stats' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="switchTab('docker-container-stats')">
          <i class="fas fa-chart-line"></i><span>监控</span>
        </button>
        <button v-if="portal.hasPerm('GET /api/docker/container/:id/logs')" :class="['tab-btn', activeTab === 'docker-container-logs' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="switchTab('docker-container-logs')">
          <i class="fas fa-file-lines"></i><span>日志</span>
        </button>
        <button v-if="container.state === 'running' && portal.hasPerm('GET /api/docker/container/:id/exec')" :class="['tab-btn', activeTab === 'docker-container-exec' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="switchTab('docker-container-exec')">
          <i class="fas fa-terminal"></i><span>终端</span>
        </button>
      </div>
    </div>
  </div>
</template>
