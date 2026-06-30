<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { DockerInfo } from '@/service/types'

@Component
class DockerOverview extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    info: DockerInfo | null = null
    loading = false

    readonly statCards: { key: keyof DockerInfo; label: string; icon: string; bgColor: string }[] = [
        { key: 'containersRunning', label: '运行中容器', icon: 'fa-play',          bgColor: 'bg-emerald-500' },
        { key: 'containersStopped', label: '已停止容器', icon: 'fa-stop',          bgColor: 'bg-amber-500' },
        { key: 'containersPaused',  label: '已暂停容器', icon: 'fa-pause',         bgColor: 'bg-slate-400' },
        { key: 'imagesTotal',       label: '镜像总数',   icon: 'fa-compact-disc',  bgColor: 'bg-blue-500' },
        { key: 'networksTotal',     label: '网络总数',   icon: 'fa-network-wired', bgColor: 'bg-purple-500' },
        { key: 'volumesTotal',      label: '数据卷总数', icon: 'fa-database',      bgColor: 'bg-indigo-500' },
        { key: 'registryMirrors',   label: '镜像源',     icon: 'fa-gauge-high',    bgColor: 'bg-cyan-500' },
        { key: 'indexServerAddress', label: '默认仓库',  icon: 'fa-box-open',      bgColor: 'bg-slate-500' },
    ]

    // ─── 方法 ───
    async load() {
        this.loading = true
        try {
            const requests: Promise<unknown>[] = []
            const keys: string[] = []

            if (this.portal.hasPerm('GET /api/docker/info')) {
                requests.push(api.dockerInfo())
                keys.push('info')
            }

            const results = await Promise.all(requests)
            const info: Record<string, number> = {}

            keys.forEach((key, index) => {
                const res = results[index] as { payload?: DockerInfo }
                if (key === 'info' && res.payload) {
                    Object.assign(info, res.payload)
                }
            })

            this.info = (Object.keys(info).length ? info : null) as unknown as DockerInfo
        } catch {
            this.info = null
        } finally {
            this.loading = false
        }
    }

    cardValue(key: keyof DockerInfo): string | number {
        if (!this.info) return 0
        if (key === 'registryMirrors') return this.info.registryMirrors?.length ?? 0
        if (key === 'indexServerAddress') return this.info.indexServerAddress || '-'
        return this.info[key] as number
    }

    // ─── 生命周期 ───
    mounted() {
        this.load()
    }
}

export default toNative(DockerOverview)
</script>

<template>
  <div>
    <!-- Loading -->
    <div v-if="loading" class="overview-loading">
      <div class="spinner-md"></div>
      <span class="text-slate-400 text-sm">加载中...</span>
    </div>

    <!-- Stats Grid -->
    <div v-else-if="info" class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
      <div v-for="card in statCards" :key="card.key" class="stat-card items-center">
        <div :class="['page-icon', card.bgColor]">
          <i :class="['fas', card.icon, 'text-white']"></i>
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-xs text-slate-400 mb-0.5">{{ card.label }}</p>
          <p class="font-semibold text-slate-800 truncate">{{ cardValue(card.key) }}</p>
        </div>
      </div>
    </div>

    <!-- Empty/Error State -->
    <div v-else class="overview-unavailable">
      <i class="fab fa-docker text-2xl text-slate-300"></i>
      <div>
        <p class="text-sm font-medium text-slate-600">无法获取 Docker 信息</p>
        <p class="text-xs text-slate-400">请确认 Docker 服务是否正常运行</p>
      </div>
    </div>
  </div>
</template>
