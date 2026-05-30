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
        { key: 'volumesTotal',      label: '数据卷总数', icon: 'fa-database',      bgColor: 'bg-indigo-500' },
        { key: 'networksTotal',     label: '网络总数',   icon: 'fa-network-wired', bgColor: 'bg-purple-500' },
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
            this.portal.showNotification('error', '加载 Docker 信息失败')
            this.info = null
        } finally {
            this.loading = false
        }
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
    <div v-if="loading" class="flex items-center justify-center py-10">
      <div class="w-8 h-8 spinner mr-2"></div>
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
          <p class="font-semibold text-slate-800">{{ info[card.key] || 0 }}</p>
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
