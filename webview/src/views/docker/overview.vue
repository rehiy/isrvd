<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { DockerInfo } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

@Component
class DockerOverview extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    info: DockerInfo | null = null
    loading = false

    readonly statCards: { key: keyof DockerInfo; label: string; icon: string; bgColor: string }[] = [
        { key: 'containersRunning', label: '运行中容器', icon: 'fa-play',          bgColor: 'bg-emerald-500' },
        { key: 'containersStopped', label: '已停止容器', icon: 'fa-stop',          bgColor: 'bg-amber-500' },
        { key: 'containersPaused',  label: '已暂停容器', icon: 'fa-pause',         bgColor: 'bg-slate-400' },
        { key: 'imagesTotal',       label: '镜像总数',   icon: 'fa-compact-disc',  bgColor: 'bg-blue-500' },
        { key: 'volumesTotal',      label: '数据卷总数', icon: 'fa-database',      bgColor: 'bg-indigo-500' },
        { key: 'networksTotal',     label: '网络总数',   icon: 'fa-network-wired', bgColor: 'bg-purple-500' },
    ]

    // ─── 生命周期 ───
    mounted() {
        this.load()
    }

    // ─── 方法 ───
    async load() {
        this.loading = true
        try {
            const res = await api.dockerInfo()
            this.info = res.payload ?? null
        } catch (e) {
            this.actions.showNotification('error', '加载 Docker 信息失败')
            this.info = null
        }
        this.loading = false
    }
}

export default toNative(DockerOverview)
</script>

<template>
  <div>
    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-10">
      <div class="w-8 h-8 spinner mr-2"></div>
      <span class="text-slate-400 text-sm">加载中...</span>
    </div>

    <!-- Stats Grid -->
    <div v-else-if="info" class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3">
      <div
        v-for="card in statCards"
        :key="card.key"
        class="rounded-xl border border-slate-200 bg-white p-4 hover:shadow-md transition-shadow"
      >
        <div class="flex flex-col items-center gap-2 text-center">
          <div :class="['w-10 h-10 rounded-xl flex items-center justify-center', card.bgColor]">
            <i :class="['fas', card.icon, 'text-white']"></i>
          </div>
          <p class="text-2xl font-bold text-slate-800">{{ info[card.key] || 0 }}</p>
          <p class="text-xs text-slate-500 leading-tight">{{ card.label }}</p>
        </div>
      </div>
    </div>

    <!-- Empty/Error State -->
    <div v-else class="flex items-center gap-3 py-6 px-4 rounded-xl bg-slate-50">
      <i class="fab fa-docker text-2xl text-slate-300"></i>
      <div>
        <p class="text-sm font-medium text-slate-600">无法获取 Docker 信息</p>
        <p class="text-xs text-slate-400">请确认 Docker 服务是否正常运行</p>
      </div>
    </div>
  </div>
</template>
