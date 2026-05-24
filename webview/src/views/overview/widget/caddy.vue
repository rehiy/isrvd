<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { CaddyInfo } from '@/service/types'

import { usePortal } from '@/stores'

@Component
class CaddyOverview extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    info: CaddyInfo | null = null
    loading = false

    readonly statCards = [
        { key: 'servers', label: 'Server 数',  icon: 'fa-server',         bgColor: 'bg-indigo-500' },
        { key: 'routes',  label: '路由总数',   icon: 'fa-route',          bgColor: 'bg-orange-500' },
        { key: 'certs',   label: 'TLS 证书',   icon: 'fa-lock',           bgColor: 'bg-cyan-500' },
        { key: 'hasTls',  label: 'TLS 启用',   icon: 'fa-certificate',    bgColor: 'bg-emerald-500' },
    ]

    // ─── 方法 ───
    async load() {
        this.loading = true
        try {
            if (!this.portal.hasPerm('GET /api/caddy/info')) {
                this.info = null
                return
            }

            this.info = (await api.caddyInfo()).payload || null
        } catch {
            this.portal.showNotification('error', '获取 Caddy 信息失败')
            this.info = null
        } finally {
            this.loading = false
        }
    }

    cardValue(key: string): string | number {
        if (!this.info) return 0
        if (key === 'hasTls') return this.info.hasTls ? 'Y' : 'N'
        const v = (this.info as unknown as Record<string, unknown>)[key]
        return typeof v === 'number' ? v : 0
    }

    // ─── 生命周期 ───
    mounted() {
        this.load()
    }
}

export default toNative(CaddyOverview)
</script>

<template>
  <div>
    <div v-if="loading" class="flex items-center justify-center py-10">
      <div class="w-8 h-8 spinner mr-2"></div>
      <span class="text-slate-400 text-sm">加载中...</span>
    </div>

    <div v-else-if="info && info.available" class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
      <div v-for="card in statCards" :key="card.key" class="rounded-xl border border-slate-200 bg-white p-4 hover:shadow-md transition-shadow">
        <div class="flex flex-col items-center gap-2 text-center">
          <div :class="['page-icon', card.bgColor]">
            <i :class="['fas', card.icon, 'text-white']"></i>
          </div>
          <p class="text-2xl font-bold text-slate-800">{{ cardValue(card.key) }}</p>
          <p class="text-xs text-slate-500 leading-tight">{{ card.label }}</p>
        </div>
      </div>
    </div>

    <div v-else class="flex items-center gap-3 py-6 px-4 rounded-xl bg-slate-50">
      <i class="fas fa-shield text-2xl text-slate-300"></i>
      <div>
        <p class="text-sm font-medium text-slate-600">无法获取 Caddy 信息</p>
        <p class="text-xs text-slate-400">请确认 Caddy admin api 是否正常运行</p>
      </div>
    </div>
  </div>
</template>
