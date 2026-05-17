<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import type { SystemStat } from '@/service/types'

@Component
class SystemInfo extends Vue {
    current: SystemStat | null = null

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

    pushData(payload: SystemStat) {
        this.current = payload
    }
}

export default toNative(SystemInfo)
</script>

<template>
  <div v-if="current?.system" class="grid grid-cols-1 xs:grid-cols-2 md:grid-cols-5 gap-3">
    <div class="rounded-xl border border-slate-200 bg-white p-4">
      <p class="text-xs text-slate-400 mb-1">程序版本</p>
      <div class="flex items-center gap-2">
        <p class="text-sm font-semibold text-slate-800 truncate">{{ current.version }}</p>
        <a
          v-if="current.versionCheck?.update" :href="current.versionCheck.release" target="_blank" rel="noopener noreferrer"
          class="update-link"
        >
          <i class="fas fa-arrow-up text-[10px]"></i>
          更新 {{ current.versionCheck.latest }}
        </a>
      </div>
    </div>
    <div class="rounded-xl border border-slate-200 bg-white p-4">
      <p class="text-xs text-slate-400 mb-1">主机名</p>
      <p class="text-sm font-semibold text-slate-800 truncate">{{ current.system.hostName }}</p>
    </div>
    <div class="rounded-xl border border-slate-200 bg-white p-4">
      <p class="text-xs text-slate-400 mb-1">操作系统</p>
      <p class="text-sm font-semibold text-slate-800 truncate">{{ current.system.platform }} / {{ current.system.kernelArch }}</p>
    </div>
    <div class="rounded-xl border border-slate-200 bg-white p-4">
      <p class="text-xs text-slate-400 mb-1">运行时间</p>
      <p class="text-sm font-semibold text-slate-800">{{ fmtUptime(current.system.uptime) }}</p>
    </div>
    <div class="rounded-xl border border-slate-200 bg-white p-4">
      <p class="text-xs text-slate-400 mb-1">CPU 核心</p>
      <p class="text-sm font-semibold text-slate-800">{{ current.system.cpuCore }} 物理 / {{ current.system.cpuCoreLogic }} 逻辑</p>
    </div>
  </div>
</template>
