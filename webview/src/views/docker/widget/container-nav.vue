<script lang="ts">
import { Component, Inject, Prop, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { DockerContainerInfo } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

@Component({
    emits: ['loaded']
})
class ContainerNav extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

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
            const res = await api.listContainers(true)
            const list = res.payload || []
            this.container = list.find(c => c.id === this.containerId) ?? null
            if (!this.container) {
                this.actions.showNotification('error', '容器不存在')
                this.$router.push('/docker/containers')
                return
            }
            this.$emit('loaded', this.container)
        } catch (e) {
            this.actions.showNotification('error', '加载容器信息失败')
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
  <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3">
    <!-- 桌面端布局 -->
    <div class="hidden md:flex md:items-center justify-between">
      <div class="flex items-center gap-3">
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
      <div v-if="container" class="flex items-center gap-2">
        <div class="flex gap-1 bg-slate-100 p-1 rounded-lg">
          <button @click="switchTab('docker-container-stats')" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab === 'docker-container-stats' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
            <i class="fas fa-chart-line"></i><span>监控</span>
          </button>
          <button @click="switchTab('docker-container-logs')" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab === 'docker-container-logs' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
            <i class="fas fa-file-lines"></i><span>日志</span>
          </button>
          <button v-if="actions.hasPerm('docker', true)" @click="switchTab('docker-container-terminal')" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab === 'docker-container-terminal' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
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
            <div :class="['w-9 h-9 rounded-lg flex items-center justify-center flex-shrink-0', container.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
              <i class="fas fa-cube text-white"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">{{ container.name || container.id }}</h1>
              <p class="text-xs text-slate-500 font-mono truncate">{{ container.image }}</p>
            </div>
          </template>
          <template v-else>
            <div class="w-9 h-9 rounded-lg bg-slate-300 flex items-center justify-center animate-pulse flex-shrink-0">
              <i class="fas fa-cube text-white"></i>
            </div>
            <div><h1 class="text-lg font-semibold text-slate-800">加载中...</h1></div>
          </template>
        </div>

      </div>
      <div v-if="container" class="flex justify-center gap-1 bg-slate-100 p-1 rounded-lg">
        <button @click="switchTab('docker-container-stats')" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab === 'docker-container-stats' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
          <i class="fas fa-chart-line"></i><span class="hidden sm:inline">监控</span>
        </button>
        <button @click="switchTab('docker-container-logs')" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab === 'docker-container-logs' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
          <i class="fas fa-file-lines"></i><span class="hidden sm:inline">日志</span>
        </button>
        <button v-if="actions.hasPerm('docker', true)" @click="switchTab('docker-container-terminal')" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab === 'docker-container-terminal' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
          <i class="fas fa-terminal"></i><span class="hidden sm:inline">终端</span>
        </button>
      </div>
    </div>
  </div>
</template>
