<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { ApisixInfo } from '@/service/types'

import { usePortal } from '@/stores'

@Component
class ApisixOverview extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    info: ApisixInfo | null = null
    loading = false

    readonly statCards = [
        { key: 'routes',        label: '路由总数',   icon: 'fa-route',         bgColor: 'bg-orange-500' },
        { key: 'consumers',     label: '消费者总数', icon: 'fa-user-tag',      bgColor: 'bg-amber-500' },
        { key: 'upstreams',     label: '上游总数',   icon: 'fa-server',        bgColor: 'bg-emerald-500' },
        { key: 'pluginConfigs', label: '插件配置',   icon: 'fa-puzzle-piece', bgColor: 'bg-rose-500' },
        { key: 'ssl',           label: 'SSL 证书',   icon: 'fa-lock',          bgColor: 'bg-cyan-500' },
        { key: 'whitelist',     label: '白名单授权', icon: 'fa-shield-halved', bgColor: 'bg-emerald-500' },
    ]

    // ─── 方法 ───
    async load() {
        this.loading = true
        try {
            const requests: Promise<unknown>[] = []
            const keys: string[] = []

            if (this.portal.hasPerm('GET /api/apisix/routes')) {
                requests.push(api.apisixRouteList())
                keys.push('routes')
            }
            if (this.portal.hasPerm('GET /api/apisix/consumers')) {
                requests.push(api.apisixConsumerList())
                keys.push('consumers')
            }
            if (this.portal.hasPerm('GET /api/apisix/upstreams')) {
                requests.push(api.apisixUpstreamList())
                keys.push('upstreams')
            }
            if (this.portal.hasPerm('GET /api/apisix/plugin-configs')) {
                requests.push(api.apisixPluginConfigList())
                keys.push('pluginConfigs')
            }
            if (this.portal.hasPerm('GET /api/apisix/ssls')) {
                requests.push(api.apisixSSLList())
                keys.push('ssl')
            }
            if (this.portal.hasPerm('GET /api/apisix/whitelist')) {
                requests.push(api.apisixWhitelist())
                keys.push('whitelist')
            }

            const results = await Promise.all(requests)
            const info: Record<string, number> = {}

            keys.forEach((key, index) => {
                const res = results[index] as { payload?: unknown[] }
                info[key] = (res.payload || []).length
            })

            this.info = info as ApisixInfo
        } catch {
            this.portal.showNotification('error', '获取 APISIX 信息失败')
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

export default toNative(ApisixOverview)
</script>

<template>
  <div>
    <div v-if="loading" class="flex items-center justify-center py-10">
      <div class="w-8 h-8 spinner mr-2"></div>
      <span class="text-slate-400 text-sm">加载中...</span>
    </div>

    <div v-else-if="info" class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3">
      <div
        v-for="card in statCards"
        :key="card.key"
        class="rounded-xl border border-slate-200 bg-white p-4 hover:shadow-md transition-shadow"
      >
        <div class="flex flex-col items-center gap-2 text-center">
          <div :class="['page-icon', card.bgColor]">
            <i :class="['fas', card.icon, 'text-white']"></i>
          </div>
          <p class="text-2xl font-bold text-slate-800">{{ info[card.key] ?? 0 }}</p>
          <p class="text-xs text-slate-500 leading-tight">{{ card.label }}</p>
        </div>
      </div>
    </div>

    <div v-else class="flex items-center gap-3 py-6 px-4 rounded-xl bg-slate-50">
      <i class="fas fa-route text-2xl text-slate-300"></i>
      <div>
        <p class="text-sm font-medium text-slate-600">无法获取 APISIX 信息</p>
        <p class="text-xs text-slate-400">请确认 APISIX 服务是否正常运行</p>
      </div>
    </div>
  </div>
</template>
