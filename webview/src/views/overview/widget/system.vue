<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { SystemStat, MonitorHostRecord, SystemVersionInfo } from '@/service/types'

@Component
class SystemOverview extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    systemInfo: SystemStat | null = null
    versionInfo: SystemVersionInfo | null = null
    loading = false

    // ─── 计算属性 ───
    get statCards() {
        if (!this.systemInfo?.system) return []
        const s = this.systemInfo.system
        const cards = [
            { label: '主机名', value: s.hostName, icon: 'fa-server', bgColor: 'bg-blue-500' },
            { label: '操作系统', value: `${s.platform} / ${s.kernelArch}`, icon: 'fa-desktop', bgColor: 'bg-indigo-500' },
            { label: 'CPU 核心', value: `${s.cpuCore} 物理 / ${s.cpuCoreLogic} 逻辑`, icon: 'fa-microchip', bgColor: 'bg-emerald-500' },
        ]
        // 添加内存信息（放在 CPU 后面）
        if (s.memoryTotal > 0) {
            const usedGB = (s.memoryUsed / 1024 / 1024 / 1024).toFixed(1)
            const totalGB = (s.memoryTotal / 1024 / 1024 / 1024).toFixed(1)
            cards.push({ label: '内存使用',  value: `${usedGB}GB / ${totalGB}GB`, icon: 'fa-memory', bgColor: 'bg-rose-500'})
        }
        cards.push({ label: `系统时间（${this.systemInfo.timezone}）`, value: this.systemInfo.time, icon: 'fa-clock', bgColor: 'bg-cyan-500' })
        cards.push({ label: '运行时间', value: this.fmtUptime(s.uptime), icon: 'fa-clock', bgColor: 'bg-amber-500' })
        // 有权限且已获取到版本信息时显示
        if (this.versionInfo?.current) {
            cards.push({ label: '程序版本', value: this.versionInfo.current, icon: 'fa-code-branch', bgColor: 'bg-purple-500' })
        }
        return cards
    }

    // ─── 方法 ───
    fmtUptime(seconds: number) {
        if (!seconds) return '0s'
        const d = Math.floor(seconds / 86400)
        const h = Math.floor((seconds % 86400) / 3600)
        const m = Math.floor((seconds % 3600) / 60)
        const parts: string[] = []
        if (d) parts.push(`${d}d`)
        if (h) parts.push(`${h}h`)
        if (m) parts.push(`${m}m`)
        if (!parts.length) parts.push(`${seconds % 60}s`)
        return parts.join(' ')
    }

    async load() {
        this.loading = true
        try {
            await this.loadSystemInfo()
        } finally {
            this.loading = false
        }
        this.loadVersionInfo()
    }

    async loadSystemInfo() {
        if (!this.portal.hasPerm('GET /api/overview/monitor')) return
        const res = await api.overviewMonitor({ since: 0 })
        const record = res.payload as MonitorHostRecord | null
        if (record?.data) {
            this.systemInfo = record.data
        }
    }

    async loadVersionInfo() {
        if (!this.portal.hasPerm('GET /api/overview/version')) return
        try {
            const res = await api.overviewVersion()
            this.versionInfo = res.payload ?? null
        } catch {
            // 静默处理
        }
    }

    // ─── 生命周期 ───
    mounted() {
        this.load()
    }
}

export default toNative(SystemOverview)
</script>

<template>
  <div>
    <!-- 系统统计卡片 -->
    <div v-if="loading" class="flex items-center justify-center py-6">
      <div class="w-6 h-6 spinner mr-2"></div>
      <span class="text-slate-400 text-sm">加载中...</span>
    </div>
    <div v-else-if="statCards.length" class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
      <div v-for="card in statCards" :key="card.label" class="stat-card items-center">
        <div :class="['page-icon', card.bgColor]">
          <i :class="['fas', card.icon, 'text-white']"></i>
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-xs text-slate-400 mb-0.5">{{ card.label }}</p>
          <p class="font-bold text-slate-800 truncate">{{ card.value }}</p>
        </div>
      </div>
    </div>
    <div v-else class="flex items-center gap-3 py-4 px-4 rounded-xl bg-slate-50">
      <i class="fas fa-triangle-exclamation text-2xl text-slate-300"></i>
      <p class="text-sm text-slate-500">获取系统信息失败</p>
    </div>
  </div>
</template>
