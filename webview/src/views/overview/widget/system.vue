<script lang="ts">
import SystemCpuMem from './system_cpu_mem.vue'
import SystemDisk from './system_disk.vue'
import SystemGo from './system_go.vue'
import SystemGpu from './system_gpu.vue'
import SystemInfo from './system_info.vue'
import SystemNetwork from './system_network.vue'
import { Component, Inject, Ref, Vue, toNative } from 'vue-facing-decorator'

import { APP_STATE_KEY } from '@/store/state'

import api from '@/service/api'
import type { SystemStat } from '@/service/types'

import { POLL_INTERVAL } from '@/helper/utils'

@Component({
    components: { SystemInfo, SystemCpuMem, SystemGpu, SystemDisk, SystemNetwork, SystemGo }
})
class SystemOverview extends Vue {
    @Inject({ from: APP_STATE_KEY }) readonly state!: { token: string | null }
    loading = false
    ready = false

    private pollTimer: ReturnType<typeof setInterval> | null = null

    @Ref readonly infoRef!: InstanceType<typeof SystemInfo>
    @Ref readonly cpuMemRef!: InstanceType<typeof SystemCpuMem>
    @Ref readonly gpuRef!: InstanceType<typeof SystemGpu>
    @Ref readonly diskRef!: InstanceType<typeof SystemDisk>
    @Ref readonly networkRef!: InstanceType<typeof SystemNetwork>
    @Ref readonly goRef!: InstanceType<typeof SystemGo>

    private dispatchData(payload: SystemStat) {
        this.infoRef?.pushData(payload)
        this.cpuMemRef?.pushData(payload)
        this.gpuRef?.pushData(payload)
        this.diskRef?.pushData(payload)
        this.networkRef?.pushData(payload)
        this.goRef?.pushData(payload)
    }

    async loadData() {
        this.loading = true
        try {
            const res = await api.systemStat()
            if (res.payload) {
                this.ready = true
                this.dispatchData(res.payload)
            }
        } catch (e) { /* ignore */ }
        this.loading = false
    }

    async poll() {
        if (!this.state.token) {
            this.stopPoll()
            return
        }
        try {
            const res = await api.systemStat()
            if (res.payload) {
                this.dispatchData(res.payload)
            }
        } catch (e) { /* ignore */ }
    }

    startPoll() {
        this.pollTimer = setInterval(() => this.poll(), POLL_INTERVAL)
    }

    stopPoll() {
        if (this.pollTimer) { clearInterval(this.pollTimer); this.pollTimer = null }
    }

    async load() {
        this.stopPoll()
        await this.loadData()
        this.startPoll()
    }

    mounted() {
        this.load()
    }

    unmounted() {
        this.stopPoll()
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

        <div v-show="!loading && !ready" class="flex items-center gap-3 py-6 px-4 rounded-xl bg-slate-50">
            <i class="fas fa-triangle-exclamation text-2xl text-slate-300"></i>
            <p class="text-sm text-slate-500">获取系统信息失败</p>
        </div>
    </div>
</template>
