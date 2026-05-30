<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { MonitorHostRecord } from '@/service/types'

import { POLL_INTERVAL } from '@/helper/utils'

import SystemCpuMem from './monitor/cpu_mem.vue'
import SystemGpu from './monitor/gpu.vue'
import SystemDisk from './monitor/disk.vue'
import SystemNetwork from './monitor/network.vue'
import SystemGo from './monitor/go.vue'

@Component({
    components: { SystemCpuMem, SystemGpu, SystemDisk, SystemNetwork, SystemGo }
})
class MonitorPage extends Vue {
    portal = usePortal()

    loading = true
    ready = false

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
        this.goRef?.pushData(data)
    }

    private async fetchLatest(): Promise<boolean> {
        const res = await api.overviewMonitor({ type: 'host', since: 0 })
        if (this.destroyed) return false
        const rec = res.payload as MonitorHostRecord | null
        if (rec) {
            this.dispatchData(rec)
            return true
        }
        return false
    }

    async loadHistory() {
        try {
            const res = await api.overviewMonitor({ type: 'host', since: 3600 })
            if (res.payload && res.payload.length > 0) {
                for (const rec of res.payload) {
                    this.dispatchData(rec)
                }
            }
        } catch { /* ignore */ }
    }

    async loadData() {
        this.loading = true
        try {
            const ok = await this.fetchLatest()
            if (ok) this.ready = true
        } catch { /* ignore */ } finally {
            this.loading = false
        }
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
            this.startPoll()
        }
    }

    startPoll() {
        this.pollTimer = setInterval(() => this.poll(), POLL_INTERVAL)
    }

    stopPoll() {
        if (this.pollTimer) { clearInterval(this.pollTimer); this.pollTimer = null }
    }

    async load() {
        this.stopPoll()
        await this.loadHistory()
        await this.loadData()
        this.startPoll()
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
      <div class="flex items-center gap-3">
        <div class="page-icon bg-blue-600">
          <i class="fas fa-desktop text-white text-sm"></i>
        </div>
        <div>
          <h1 class="text-lg font-semibold text-slate-800">系统监控</h1>
          <p class="text-xs text-slate-500">实时系统资源监控</p>
        </div>
      </div>
      <div class="flex items-center gap-2 flex-shrink-0">
        <!-- 这里可以添加刷新等功能按钮 -->
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
