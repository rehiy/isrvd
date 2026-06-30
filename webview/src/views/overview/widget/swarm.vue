<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { SwarmInfo } from '@/service/types'

@Component
class SwarmOverview extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    swarmInfo: SwarmInfo | null = null
    loading = false

    readonly statCards: { key: keyof SwarmInfo; label: string; icon: string; bgColor: string }[] = [
        { key: 'clusterID', label: '集群 ID',  icon: 'fa-fingerprint', bgColor: 'bg-violet-500' },
        { key: 'createdAt', label: '创建时间', icon: 'fa-calendar',     bgColor: 'bg-cyan-500' },
        { key: 'nodes',     label: '总节点数', icon: 'fa-server',       bgColor: 'bg-blue-500' },
        { key: 'managers',  label: 'Manager', icon: 'fa-crown',        bgColor: 'bg-indigo-500' },
        { key: 'workers',   label: 'Worker',  icon: 'fa-circle-nodes', bgColor: 'bg-slate-400' },
        { key: 'services',  label: '服务总数', icon: 'fa-cubes',        bgColor: 'bg-emerald-500' },
        { key: 'tasks',     label: '任务总数', icon: 'fa-tasks',        bgColor: 'bg-amber-500' },
    ]

    // ─── 生命周期 ───
    mounted() {
        this.load()
    }

    cardValue(key: keyof SwarmInfo): string | number {
        if (!this.swarmInfo) return 0
        if (key === 'clusterID') return this.swarmInfo.clusterID ? this.swarmInfo.clusterID.slice(0, 12) : '-'
        if (key === 'createdAt') return this.swarmInfo.createdAt ? new Date(this.swarmInfo.createdAt).toLocaleDateString('zh-CN') : '-'
        return this.swarmInfo[key] as number
    }

    // ─── 方法 ───
    async load() {
        this.loading = true
        try {
            const requests: Promise<unknown>[] = []
            const keys: string[] = []

            if (this.portal.hasPerm('GET /api/swarm/info')) {
                requests.push(api.swarmInfo())
                keys.push('info')
            }

            const results = await Promise.all(requests)
            const info: Record<string, number> = {}

            keys.forEach((key, index) => {
                const res = results[index] as { payload?: SwarmInfo }
                if (key === 'info' && res.payload) {
                    Object.assign(info, res.payload)
                }
            })

            this.swarmInfo = (Object.keys(info).length ? info : null) as unknown as SwarmInfo
        } catch {
            this.swarmInfo = null
        } finally {
            this.loading = false
        }
    }
}

export default toNative(SwarmOverview)
</script>

<template>
  <div>
    <div v-if="loading" class="overview-loading">
      <div class="spinner-md"></div>
      <span class="text-slate-400 text-sm">加载中...</span>
    </div>
    <div v-else-if="swarmInfo" class="space-y-4">
      <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
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
    </div>
    <div v-else class="overview-unavailable">
      <i class="fas fa-circle-nodes text-2xl text-slate-300"></i>
      <div>
        <p class="text-sm font-medium text-slate-600">Swarm 集群未初始化</p>
        <p class="text-xs text-slate-400">请先执行 <code class="bg-slate-200 px-1 rounded">docker swarm init</code></p>
      </div>
    </div>
  </div>
</template>
