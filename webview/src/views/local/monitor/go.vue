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
    <div class="card-header">
      <div class="card-icon bg-sky-500">
        <i class="fas fa-code text-white text-xs"></i>
      </div>
      <span class="text-sm font-semibold text-slate-700">Go 运行态</span>
      <span class="ml-auto text-xs text-slate-400 font-mono">{{ current.version }}</span>
    </div>
    <div class="grid grid-cols-1 xs:grid-cols-2 md:grid-cols-4 divide-x divide-y divide-slate-100">
      <div class="px-4 py-3" title="逻辑 CPU 核心数">
        <p class="text-xs text-slate-400 mb-1">CPU 核心数</p>
        <p class="text-sm font-bold text-slate-800">{{ current.numCPU }}</p>
      </div>
      <div class="px-4 py-3" title="当前活跃的 Goroutine 数量">
        <p class="text-xs text-slate-400 mb-1">Goroutine 数</p>
        <p class="text-sm font-bold text-slate-800">{{ current.numGoroutine }}</p>
      </div>
      <div class="px-4 py-3" title="当前活跃对象占用的堆内存（Go 真实使用量）">
        <p class="text-xs text-slate-400 mb-1">已分配 (alloc)</p>
        <p class="text-sm font-bold text-slate-800">{{ fmtSize(current.alloc) }}</p>
      </div>
      <div class="px-4 py-3" title="GC 后仍存活的堆对象占用量">
        <p class="text-xs text-slate-400 mb-1">堆已分配</p>
        <p class="text-sm font-bold text-slate-800">{{ fmtSize(current.heapAlloc) }}</p>
      </div>
      <div class="px-4 py-3" title="堆中正在使用的 span 占用量（不含已归还 OS 的部分）">
        <p class="text-xs text-slate-400 mb-1">堆使用中</p>
        <p class="text-sm font-bold text-slate-800">{{ fmtSize(current.heapInuse) }}</p>
      </div>
      <div class="px-4 py-3" title="堆向 OS 申请的虚拟内存总量">
        <p class="text-xs text-slate-400 mb-1">堆已申请</p>
        <p class="text-sm font-bold text-slate-800">{{ fmtSize(current.heapSys) }}</p>
      </div>
      <div class="px-4 py-3" title="Goroutine 栈当前实际占用量">
        <p class="text-xs text-slate-400 mb-1">栈使用中</p>
        <p class="text-sm font-bold text-slate-800">{{ fmtSize(current.stackInuse) }}</p>
      </div>
      <div class="px-4 py-3" title="栈向 OS 申请的虚拟内存总量">
        <p class="text-xs text-slate-400 mb-1">栈已申请</p>
        <p class="text-sm font-bold text-slate-800">{{ fmtSize(current.stackSys) }}</p>
      </div>
      <div class="px-4 py-3" title="进程从 OS 申请的全部虚拟内存（堆+栈+元数据等）">
        <p class="text-xs text-slate-400 mb-1">系统申请</p>
        <p class="text-sm font-bold text-slate-800">{{ fmtSize(current.sys) }}</p>
      </div>
      <div class="px-4 py-3" title="进程启动以来累计分配的堆内存总量（只增不减）">
        <p class="text-xs text-slate-400 mb-1">累计分配</p>
        <p class="text-sm font-bold text-slate-800">{{ fmtSize(current.totalAlloc) }}</p>
      </div>
      <div class="px-4 py-3" title="GC 执行总次数">
        <p class="text-xs text-slate-400 mb-1">GC 次数</p>
        <p class="text-sm font-bold text-slate-800">{{ current.numGC }}</p>
      </div>
      <div class="px-4 py-3" title="最近一次 GC 完成的时间">
        <p class="text-xs text-slate-400 mb-1">最后 GC</p>
        <p class="text-sm font-bold text-slate-800">{{ fmtGCTime(current.lastGC) }}</p>
      </div>
    </div>
  </div>
</template>
