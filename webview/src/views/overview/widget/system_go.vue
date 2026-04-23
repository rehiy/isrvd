<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import type { SystemStat, SystemGoRuntimeStat } from '@/service/types'

@Component
class SystemGo extends Vue {
    current: SystemGoRuntimeStat | null = null

    fmtSize(bytes: number) {
        if (!bytes || bytes < 0) return '0 B'
        const units = ['B', 'KB', 'MB', 'GB', 'TB']
        let i = 0, v = bytes
        while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
        return `${v.toFixed(1)} ${units[i]}`
    }

    fmtGCTime(ts: number) {
        if (!ts) return '从未'
        return new Date(ts * 1000).toLocaleString('zh-CN')
    }

    pushData(payload: SystemStat) {
        this.current = payload.go
    }
}

export default toNative(SystemGo)
</script>

<template>
  <div v-if="current" class="rounded-xl border border-slate-200 bg-white overflow-hidden">
    <div class="px-4 py-3 border-b border-slate-100 flex items-center gap-2">
      <div class="w-6 h-6 rounded-md bg-sky-500 flex items-center justify-center">
        <i class="fas fa-code text-white text-xs"></i>
      </div>
      <span class="text-sm font-semibold text-slate-700">Go 运行态</span>
      <span class="ml-auto text-xs text-slate-400 font-mono">{{ current.version }}</span>
    </div>
    <div class="grid grid-cols-1 xs:grid-cols-2 md:grid-cols-4 divide-x divide-y divide-slate-100">
      <div class="px-4 py-3">
        <p class="text-xs text-slate-400 mb-1">Goroutine 数</p>
        <p class="text-sm font-bold text-slate-800">{{ current.numGoroutine }}</p>
      </div>
      <div class="px-4 py-3">
        <p class="text-xs text-slate-400 mb-1">堆已分配</p>
        <p class="text-sm font-bold text-slate-800">{{ fmtSize(current.HeapAlloc) }}</p>
      </div>
      <div class="px-4 py-3">
        <p class="text-xs text-slate-400 mb-1">堆使用中</p>
        <p class="text-sm font-bold text-slate-800">{{ fmtSize(current.HeapInuse) }}</p>
      </div>
      <div class="px-4 py-3">
        <p class="text-xs text-slate-400 mb-1">系统申请</p>
        <p class="text-sm font-bold text-slate-800">{{ fmtSize(current.Sys) }}</p>
      </div>
      <div class="px-4 py-3">
        <p class="text-xs text-slate-400 mb-1">栈使用中</p>
        <p class="text-sm font-bold text-slate-800">{{ fmtSize(current.StackInuse) }}</p>
      </div>
      <div class="px-4 py-3">
        <p class="text-xs text-slate-400 mb-1">累计分配</p>
        <p class="text-sm font-bold text-slate-800">{{ fmtSize(current.TotalAlloc) }}</p>
      </div>
      <div class="px-4 py-3">
        <p class="text-xs text-slate-400 mb-1">GC 次数</p>
        <p class="text-sm font-bold text-slate-800">{{ current.NumGC }}</p>
      </div>
      <div class="px-4 py-3">
        <p class="text-xs text-slate-400 mb-1">最后 GC</p>
        <p class="text-sm font-bold text-slate-800">{{ fmtGCTime(current.LastGC) }}</p>
      </div>
    </div>
  </div>
</template>
