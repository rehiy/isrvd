<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { MonitorHostRecord } from '@/service/types'

import { POLL_INTERVAL } from '@/helper/utils'

import SystemCpuMem from './monitor/cpu_mem.vue'
import SystemDisk from './monitor/disk.vue'
import SystemGo from './monitor/go.vue'
import SystemGpu from './monitor/gpu.vue'
import SystemNetwork from './monitor/network.vue'

@Component({
    components: { SystemCpuMem, SystemGpu, SystemDisk, SystemNetwork, SystemGo }
})
class MonitorPage extends Vue {
    portal = usePortal()

    loading = true
    ready = false

    // ─── 时间区间选择 ───
    timeRanges = [
        { label: '5分钟', value: 300 },
        { label: '1小时', value: 3600 },
        { label: '6小时', value: 21600 },
        { label: '12小时', value: 43200 },
        { label: '24小时', value: 86400 }
    ]
    selectedRange = 300  // 默认5分钟

    private pollTimer: ReturnType<typeof setInterval> | null = null
    private destroyed = false
    private polling = false
    private lastDispatchedTs = 0
    private dataVersion = 0

    // ─── Refs ───
    @Ref readonly cpuMemRef!: InstanceType<typeof SystemCpuMem>
    @Ref readonly gpuRef!: InstanceType<typeof SystemGpu>
    @Ref readonly diskRef!: InstanceType<typeof SystemDisk>
    @Ref readonly networkRef!: InstanceType<typeof SystemNetwork>
    @Ref readonly goRef!: InstanceType<typeof SystemGo>

    // ─── 方法 ───
    private dispatchData(rec: MonitorHostRecord): boolean {
        const { ts, data } = rec
        if (!data || ts <= this.lastDispatchedTs) return false
        this.lastDispatchedTs = ts
        if (data.version) {
            this.portal.currentVersion = data.version
        }
        // 只更新数据，不更新图表
        this.cpuMemRef?.pushData(data, ts)
        this.gpuRef?.pushData(data, ts)
        this.diskRef?.pushData(data, ts)
        this.networkRef?.pushData(data, ts)
        this.goRef?.pushData(data, ts)
        return true
    }

    private updateCharts() {
        // 显式调用各组件的图表更新方法
        this.cpuMemRef?.flushCharts()
        this.gpuRef?.flushCharts()
        this.diskRef?.flushCharts()
        this.networkRef?.flushCharts()
        this.goRef?.flushCharts()
    }

    private async fetchRealtime(version = this.dataVersion): Promise<boolean> {
        const res = await api.overviewMonitor({ type: 'host', since: 0 })
        if (this.destroyed || version !== this.dataVersion) {
            return false
        }

        return !!res.payload && !Array.isArray(res.payload) && this.dispatchData(res.payload as MonitorHostRecord)
    }

    async loadHistory(version = this.dataVersion): Promise<boolean> {
        try {
            const res = await api.overviewMonitor({ type: 'host', since: this.selectedRange })
            if (this.destroyed || version !== this.dataVersion) return false
            let ok = false
            if (res.payload && Array.isArray(res.payload) && res.payload.length > 0) {
                const sorted = [...res.payload].sort((a, b) => a.ts - b.ts)
                for (const rec of sorted) {
                    // 批量加载时只更新数据，最后统一更新图表
                    ok = this.dispatchData(rec) || ok
                }
                // 数据加载完成后，显式更新所有图表
                if (ok) this.updateCharts()
            }
            return ok
        } catch { return false }
    }

    clearAllData() {
        this.dataVersion++
        this.lastDispatchedTs = 0
        this.cpuMemRef?.clearData()
        this.gpuRef?.clearData()
        this.diskRef?.clearData()
        this.networkRef?.clearData()
        this.goRef?.clearData()
    }

    async switchTimeRange(range: number) {
        if (this.selectedRange === range) return
        this.selectedRange = range
        this.loading = true
        this.ready = false
        this.clearAllData()
        this.stopPoll()
        const version = this.dataVersion
        const historyOk = await this.loadHistory(version)
        const realtimeOk = await this.loadData(version)
        this.ready = historyOk || realtimeOk
        this.loading = false
        // 只有5分钟模式才启动轮询
        if (this.selectedRange === 300) {
            this.startPoll()
        }
    }

    async loadData(version = this.dataVersion) {
        try {
            const ok = await this.fetchRealtime(version)
            if (ok) this.ready = true
            return ok
        } catch { /* ignore */ }
        return false
    }

    async poll() {
        if (this.polling) {
            return
        }
        if (!this.portal.token) {
            this.stopPoll()
            return
        }
        const version = this.dataVersion
        this.polling = true
        try {
            const ok = await this.fetchRealtime(version)
            if (ok) {
                this.updateCharts()
            }
        } catch {
            // ignore poll error
        } finally {
            this.polling = false
        }
    }

    async handleVisibilityChange() {
        if (document.hidden) {
            this.stopPoll()
        } else {
            // 只有5分钟模式才在页面重新可见时启动轮询
            if (this.selectedRange === 300) {
                this.loading = true
                this.ready = false
                const version = this.dataVersion
                await this.loadHistory(version)
                await this.loadData(version)
                this.loading = false
                this.startPoll()
            }
        }
    }

    startPoll() {
        if (this.pollTimer) {
            return
        }
        this.pollTimer = setInterval(() => {
            this.poll()
        }, POLL_INTERVAL)
    }

    stopPoll() {
        if (this.pollTimer) { clearInterval(this.pollTimer); this.pollTimer = null }
    }

    async load() {
        this.loading = true
        this.ready = false
        this.stopPoll()
        this.clearAllData()
        const version = this.dataVersion
        const historyOk = await this.loadHistory(version)
        const realtimeOk = await this.loadData(version)
        this.ready = historyOk || realtimeOk
        this.loading = false
        // 只有5分钟模式才启动轮询
        if (this.selectedRange === 300) {
            this.startPoll()
        }
    }

    // ─── 生命周期 ───
    mounted() {
        // 组件挂载后开始监听页面可见性变化并加载数据
        document.addEventListener('visibilitychange', this.handleVisibilityChange)
        this.load()
    }

    unmounted() {
        // 组件销毁时清理定时器和事件监听
        this.destroyed = true
        this.stopPoll()
        document.removeEventListener('visibilitychange', this.handleVisibilityChange)
    }
}

export default toNative(MonitorPage)
</script>

<template>
  <div class="card">
    <div class="card-toolbar">
      <!-- 桌面端 -->
      <div class="hidden md:flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-blue-600">
            <i class="fas fa-desktop text-white text-sm"></i>
          </div>
          <div>
            <h1 class="text-lg font-semibold text-slate-800">系统监控</h1>
            <p class="text-xs text-slate-500">实时系统资源监控</p>
          </div>
        </div>
        <div class="flex items-center gap-3">
          <div class="tab-group">
            <button
              v-for="range in timeRanges"
              :key="range.value"
              type="button"
              :class="['tab-btn', selectedRange === range.value ? 'tab-btn-active text-blue-600' : 'tab-btn-inactive']"
              @click="switchTimeRange(range.value)"
            >
              {{ range.label }}
            </button>
          </div>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="flex md:hidden items-center justify-between">
        <div class="flex items-center gap-3 min-w-0 flex-1">
          <div class="page-icon bg-blue-600">
            <i class="fas fa-desktop text-white text-sm"></i>
          </div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">系统监控</h1>
            <p class="text-xs text-slate-500 truncate">实时系统资源监控</p>
          </div>
        </div>
      </div>
      <!-- 移动端 Tab -->
      <div class="tab-group md:hidden mt-3 overflow-x-auto">
        <button
          v-for="range in timeRanges"
          :key="range.value"
          type="button"
          :class="['tab-btn flex-1 justify-center whitespace-nowrap', selectedRange === range.value ? 'tab-btn-active text-blue-600' : 'tab-btn-inactive']"
          @click="switchTimeRange(range.value)"
        >
          {{ range.label }}
        </button>
      </div>
    </div>

    <div v-if="loading" class="card-body">
      <div class="empty-state">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
    </div>

    <div v-else-if="!ready" class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-triangle-exclamation text-3xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">获取系统信息失败</p>
        <p class="text-sm text-slate-400">请检查服务状态后刷新重试</p>
      </div>
    </div>

    <!-- 图表组件用 v-show 保持挂载，避免切换时销毁重建 -->
    <div v-show="!loading && ready" class="card-body space-y-4">
      <SystemCpuMem ref="cpuMemRef" :range-seconds="selectedRange" />
      <SystemGpu ref="gpuRef" :range-seconds="selectedRange" />
      <SystemDisk ref="diskRef" :range-seconds="selectedRange" />
      <SystemNetwork ref="networkRef" :range-seconds="selectedRange" />
      <SystemGo ref="goRef" :range-seconds="selectedRange" />
    </div>
  </div>
</template>
