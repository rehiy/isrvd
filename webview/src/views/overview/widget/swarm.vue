<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { SwarmInfo } from '@/service/types'

import { usePortal } from '@/stores'

@Component
class SwarmOverview extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    swarmInfo: SwarmInfo | null = null
    loading = false

    readonly statCards: { key: keyof SwarmInfo; label: string; icon: string; bgColor: string }[] = [
        { key: 'nodes',    label: '总节点数', icon: 'fa-server',       bgColor: 'bg-blue-500' },
        { key: 'managers', label: 'Manager', icon: 'fa-crown',        bgColor: 'bg-indigo-500' },
        { key: 'workers',  label: 'Worker',  icon: 'fa-circle-nodes', bgColor: 'bg-slate-400' },
        { key: 'services', label: '服务总数', icon: 'fa-cubes',        bgColor: 'bg-emerald-500' },
        { key: 'tasks',    label: '任务总数', icon: 'fa-tasks',        bgColor: 'bg-amber-500' },
    ]

    // ─── 生命周期 ───
    mounted() {
        this.load()
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
            this.portal.showNotification('error', '获取 Swarm 信息失败，请确认集群已初始化')
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
    <div v-if="loading" class="flex items-center justify-center py-10">
      <div class="w-8 h-8 spinner mr-2"></div>
      <span class="text-slate-400 text-sm">加载中...</span>
    </div>
    <div v-else-if="swarmInfo" class="space-y-4">
      <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-3">
        <div
          v-for="card in statCards" :key="card.key"
          class="rounded-xl border border-slate-200 bg-white p-4 hover:shadow-md transition-shadow"
        >
          <div class="flex flex-col items-center gap-2 text-center">
            <div :class="['w-9 h-9 rounded-lg flex items-center justify-center', card.bgColor]">
              <i :class="['fas', card.icon, 'text-white']"></i>
            </div>
            <p class="text-2xl font-bold text-slate-800">{{ swarmInfo[card.key] ?? 0 }}</p>
            <p class="text-xs text-slate-500 leading-tight">{{ card.label }}</p>
          </div>
        </div>
      </div>
    </div>
    <div v-else class="flex items-center gap-3 py-6 px-4 rounded-xl bg-slate-50">
      <i class="fas fa-circle-nodes text-2xl text-slate-300"></i>
      <div>
        <p class="text-sm font-medium text-slate-600">Swarm 集群未初始化</p>
        <p class="text-xs text-slate-400">请先执行 <code class="bg-slate-200 px-1 rounded">docker swarm init</code></p>
      </div>
    </div>
  </div>
</template>
