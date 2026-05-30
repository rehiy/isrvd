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

    // ─── Refs ───
    @Ref readonly cpuMemRef!: InstanceType<typeof SystemCpuMem>
    @Ref readonly gpuRef!: InstanceType<typeof SystemGpu>
    @Ref readonly diskRef!: InstanceType<typeof SystemDisk>
    @Ref readonly networkRef!: InstanceType<typeof SystemNetwork>
    @Ref readonly goRef!: InstanceType<typeof SystemGo>

    // ─── 方法 ───
    private dispatchData(rec: MonitorHostRecord) {
        const { ts, data } = rec
        if (data.version) {
            this.portal.currentVersion = data.version
        }
        this.cpuMemRef?.pushData(data, ts)
        this.gpuRef?.pushData(data, ts)
        this.diskRef?.pushData(data, ts)
        this.networkRef?.pushData(data, ts)
        this.goRef?.pushData(data, ts)
    }

    private async fetchLatest(): Promise<boolean> {
        // 根据当前选择的时间范围获取数据
        // 如果 selectedRange > 0，则获取对应时间范围的数据（用于轮询更新）
        // 如果 selectedRange === 0，则获取实时数据（since=0）
        const since = this.selectedRange > 0 ? this.selectedRange : 0
        const res = await api.overviewMonitor({ type: 'host', since })
        if (this.destroyed) return false
        
        // 处理返回数据：since=0 返回单个对象，since>0 返回数组
        if (Array.isArray(res.payload)) {
            // 历史数据数组
            for (const rec of res.payload) {
                this.dispatchData(rec)
            }
            return res.payload.length > 0
        } else if (res.payload) {
            // 实时数据单个对象
            this.dispatchData(res.payload)
            return true
        }
        return false
    }

    async loadHistory() {
        try {
            const res = await api.overviewMonitor({ type: 'host', since: this.selectedRange })
            if (res.payload && Array.isArray(res.payload) && res.payload.length > 0) {
                for (const rec of res.payload) {
                    this.dispatchData(rec)
                }
            }
        } catch { /* ignore */ }
    }

    clearAllData() {
        this.cpuMemRef?.clearData()
        this.gpuRef?.clearData()
        this.diskRef?.clearData()
        this.networkRef?.clearData()
        this.goRef?.clearData()
    }

    async switchTimeRange(range: number) {
        if (this.selectedRange === range) return
        this.selectedRange = range
        this.clearAllData()
        this.stopPoll()
        await this.loadHistory()
        await this.loadData()
        // 只有5分钟模式才启动轮询
        if (this.selectedRange === 300) {
            this.startPoll()
        }
    }

    async loadData() {
        try {
            const ok = await this.fetchLatest()
            if (ok) this.ready = true
        } catch { /* ignore */ }
    }

    async poll() {
        if (!this.portal.token) {
            this.stopPoll()
            return
        }
        try {
            await this.fetchLatest()
        } catch { /* ignore */ }
    }

    handleVisibilityChange() {
        if (document.hidden) {
            this.stopPoll()
        } else {
            // 只有5分钟模式才在页面重新可见时启动轮询
            if (this.selectedRange === 300) {
                this.startPoll()
            }
        }
    }

    startPoll() {
        this.pollTimer = setInterval(() => this.poll(), POLL_INTERVAL)
    }

    stopPoll() {
        if (this.pollTimer) { clearInterval(this.pollTimer); this.pollTimer = null }
    }

    async load() {
        this.loading = true
        this.ready = false
        this.stopPoll()
        this.clearAllData()
        await this.loadHistory()
        await this.loadData()
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

    <div class="card-body p-4 md:p-6">
      <div v-if="loading" class="flex items-center justify-center py-10">
        <div class="w-8 h-8 spinner mr-2"></div>
        <span class="text-slate-400 text-sm">加载中...</span>
      </div>

      <div v-show="!loading && ready" class="space-y-5">
        <SystemCpuMem ref="cpuMemRef" />
        <SystemGpu ref="gpuRef" />
        <SystemDisk ref="diskRef" />
        <SystemNetwork ref="networkRef" />
        <SystemGo ref="goRef" />
      </div>

      <div v-if="!loading && !ready" class="flex items-center gap-3 py-6 px-4 rounded-xl bg-slate-50">
        <i class="fas fa-triangle-exclamation text-2xl text-slate-300"></i>
        <p class="text-sm text-slate-500">获取系统信息失败</p>
      </div>
    </div>
  </div>
</template>
