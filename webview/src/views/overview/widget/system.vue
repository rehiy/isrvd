<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { MonitorHostRecord } from '@/service/types'
import { POLL_INTERVAL } from '@/helper/utils'

import SystemCpuMem from './system_cpu_mem.vue'
import SystemDisk from './system_disk.vue'
import SystemGo from './system_go.vue'
import SystemGpu from './system_gpu.vue'
import SystemInfo from './system_info.vue'
import SystemNetwork from './system_network.vue'

@Component({
    components: { SystemInfo, SystemCpuMem, SystemGpu, SystemDisk, SystemNetwork, SystemGo }
})
class SystemOverview extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    loading = true
    ready   = false

    private pollTimer: ReturnType<typeof setInterval> | null = null
    private destroyed = false

    // ─── Refs ───
    @Ref readonly infoRef!: InstanceType<typeof SystemInfo>
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
        this.infoRef?.pushData(data)
        this.cpuMemRef?.pushData(data, ts)
        this.gpuRef?.pushData(data, ts)
        this.diskRef?.pushData(data, ts)
        this.networkRef?.pushData(data, ts)
        this.goRef?.pushData(data)
    }

    /** 获取最新一条实时数据并分发，返回是否成功 */
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
        document.addEventListener('visibilitychange', this.handleVisibilityChange)
        this.load()
    }

    unmounted() {
        this.destroyed = true
        this.stopPoll()
        document.removeEventListener('visibilitychange', this.handleVisibilityChange)
    }
}

export default toNative(SystemOverview)
</script>

<template>
  <div>
    <div v-if="loading" class="flex items-center justify-center py-10">
      <div class="w-8 h-8 spinner mr-2"></div>
      <span class="text-slate-400 text-sm">加载中...</span>
    </div>

    <div v-show="!loading && ready" class="space-y-5">
      <SystemInfo ref="infoRef" />
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
</template>
