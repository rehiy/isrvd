<script lang="ts">
import type { ChartOptions } from 'chart.js'
import { Chart, registerables } from 'chart.js'
import { nextTick } from 'vue'
import { Component, Inject, Ref, Vue, Watch, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { ContainerInfo, ContainerStatsResponse } from '@/service/types'
import { formatFileSize, POLL_INTERVAL } from '@/helper/utils'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

Chart.register(...registerables)

// ─── Chart.js 回调类型 ───
interface ChartCallbackContext {
    parsed: { y: number | null }
    dataset: { label: string }
}

@Component({
    beforeRouteLeave(this: unknown) {
        (this as InstanceType<typeof ContainerStats>).stopStatsTimer()
    }
} as Record<string, unknown>)
class ContainerStats extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── Refs ───
    @Ref readonly cpuRef!: HTMLCanvasElement
    @Ref readonly memRef!: HTMLCanvasElement
    @Ref readonly netRef!: HTMLCanvasElement
    @Ref readonly blkRef!: HTMLCanvasElement

    // ─── 数据属性 ───
    container: ContainerInfo | null = null
    statsData: ContainerStatsResponse | null = null
    statsLoading = true
    netRxRate = 0
    netTxRate = 0
    blkRRate = 0
    blkWRate = 0
    formatFileSize = formatFileSize

    // ─── 私有属性（非响应式） ───
    private statsTimer: ReturnType<typeof setInterval> | null = null
    private destroyed = false
    private readonly MAX_POINTS = 60
    private labels: string[] = []
    private cpuData: number[] = []
    private memData: number[] = []
    private netRxData: number[] = []
    private netTxData: number[] = []
    private blkRData: number[] = []
    private blkWData: number[] = []
    private prevNetRx = 0
    private prevNetTx = 0
    private prevBlkR = 0
    private prevBlkW = 0
    private prevTime = 0
    private cpuChart: Chart<'line'> | null = null
    private memChart: Chart<'line'> | null = null
    private netChart: Chart<'line'> | null = null
    private blkChart: Chart<'line'> | null = null

    get containerId() {
        return this.$route.params.id as string
    }

    // ─── 方法 ───
    goBack() {
        this.$router.push('/docker/containers')
    }

    switchTab(name: string) {
        this.$router.push({ name, params: { id: this.containerId } })
    }

    activeTab() {
        return this.$route.name
    }

    async loadContainer() {
        try {
            const res = await api.listContainers(true)
            const list = res.payload || []
            this.container = list.find((c: ContainerInfo) => c.id === this.containerId) || null
            if (!this.container) {
                this.actions.showNotification('error', '容器不存在')
                this.$router.push('/docker/containers')
            }
        } catch (e) {
            this.actions.showNotification('error', '加载容器信息失败')
            this.$router.push('/docker/containers')
        }
    }

    startStatsTimer() {
        this.stopStatsTimer()
        this.statsTimer = setInterval(() => this.loadStats(), POLL_INTERVAL)
    }

    stopStatsTimer() {
        if (this.statsTimer) {
            clearInterval(this.statsTimer)
            this.statsTimer = null
        }
    }

    handleVisibilityChange() {
        if (document.hidden) {
            this.stopStatsTimer()
        } else if (this.container?.state === 'running') {
            this.startStatsTimer()
        }
    }

    async loadStats() {
        try {
            const res = await api.containerStats(this.containerId)
            if (this.destroyed || !res.payload) return
            this.statsData = res.payload ?? null
            this.pushPoint(res.payload)
            this.renderCharts()
        } catch (e) {
            // 静默失败
        }
    }

    pushPoint(data: ContainerStatsResponse) {
        const now = new Date()
        const label = now.getHours().toString().padStart(2, '0') + ':' +
            now.getMinutes().toString().padStart(2, '0') + ':' +
            now.getSeconds().toString().padStart(2, '0')

        const nowMs = Date.now()
        const elapsed = this.prevTime > 0 ? (nowMs - this.prevTime) / 1000 : 0

        this.labels.push(label)
        this.cpuData.push(+(data.cpuPercent || 0).toFixed(2))
        this.memData.push(+(data.memoryPercent || 0).toFixed(2))

        if (elapsed > 0) {
            const rxRate = +Math.max(0, ((data.networkRx || 0) - this.prevNetRx) / elapsed).toFixed(0)
            const txRate = +Math.max(0, ((data.networkTx || 0) - this.prevNetTx) / elapsed).toFixed(0)
            const brRate = +Math.max(0, ((data.blockRead || 0) - this.prevBlkR) / elapsed).toFixed(0)
            const bwRate = +Math.max(0, ((data.blockWrite || 0) - this.prevBlkW) / elapsed).toFixed(0)
            this.netRxData.push(rxRate)
            this.netTxData.push(txRate)
            this.blkRData.push(brRate)
            this.blkWData.push(bwRate)
            this.netRxRate = rxRate
            this.netTxRate = txRate
            this.blkRRate = brRate
            this.blkWRate = bwRate
        } else {
            this.netRxData.push(0)
            this.netTxData.push(0)
            this.blkRData.push(0)
            this.blkWData.push(0)
        }

        this.prevNetRx = data.networkRx || 0
        this.prevNetTx = data.networkTx || 0
        this.prevBlkR = data.blockRead || 0
        this.prevBlkW = data.blockWrite || 0
        this.prevTime = nowMs

        if (this.labels.length > this.MAX_POINTS) {
            this.labels.shift()
            this.cpuData.shift()
            this.memData.shift()
            this.netRxData.shift()
            this.netTxData.shift()
            this.blkRData.shift()
            this.blkWData.shift()
        }
    }

    baseOptions(yOptions: Record<string, unknown> = {}, tooltipCb: ((ctx: ChartCallbackContext) => string) | null = null): ChartOptions<'line'> {
        return {
            responsive: true,
            maintainAspectRatio: false,
            animation: false,
            interaction: { intersect: false, mode: 'index' as const },
            plugins: {
                legend: { display: false },
                tooltip: {
                    backgroundColor: 'rgba(15,23,42,0.9)',
                    titleFont: { size: 10 },
                    bodyFont: { size: 10 },
                    padding: 8,
                    cornerRadius: 6,
                    callbacks: tooltipCb ? { label: tooltipCb as unknown as (this: unknown, tooltipItem: unknown) => string | void | string[] } : {}
                }
            },
            scales: {
                x: { display: false },
                y: {
                    display: true,
                    beginAtZero: true,
                    grid: { color: 'rgba(148,163,184,0.08)' },
                    border: { display: false },
                    ticks: { font: { size: 9 }, color: '#94a3b8', maxTicksLimit: 4, padding: 4 },
                    ...yOptions
                }
            },
            elements: {
                point: { radius: 0, hoverRadius: 3 },
                line: { tension: 0.4, borderWidth: 1.5 }
            }
        }
    }

    makeDataset(data: number[], color: string, label = '') {
        return {
            label,
            data: [...data],
            borderColor: color,
            backgroundColor: color.replace(')', ', 0.08)').replace('rgb', 'rgba'),
            fill: true
        }
    }

    initCharts() {
        this.destroyCharts()

        if (this.cpuRef) {
            this.cpuChart = new Chart(this.cpuRef, {
                type: 'line' as const,
                data: { labels: [...this.labels], datasets: [this.makeDataset(this.cpuData, '#3b82f6')] },
                options: this.baseOptions(
                    { max: 100, ticks: { font: { size: 9 }, color: '#94a3b8', maxTicksLimit: 4, padding: 4, callback: (v: string | number) => v + '%' } },
                    (ctx: ChartCallbackContext) => (ctx.parsed.y ?? 0).toFixed(1) + '%'
                )
            })
        }

        if (this.memRef) {
            this.memChart = new Chart(this.memRef, {
                type: 'line' as const,
                data: { labels: [...this.labels], datasets: [this.makeDataset(this.memData, '#8b5cf6')] },
                options: this.baseOptions(
                    { max: 100, ticks: { font: { size: 9 }, color: '#94a3b8', maxTicksLimit: 4, padding: 4, callback: (v: string | number) => v + '%' } },
                    (ctx: ChartCallbackContext) => (ctx.parsed.y ?? 0).toFixed(1) + '%'
                )
            })
        }

        if (this.netRef) {
            this.netChart = new Chart(this.netRef, {
                type: 'line' as const,
                data: {
                    labels: [...this.labels],
                    datasets: [
                        { ...this.makeDataset(this.netRxData, '#14b8a6'), label: '接收' },
                        { ...this.makeDataset(this.netTxData, '#0d9488'), label: '发送' }
                    ]
                },
                options: {
                    ...this.baseOptions({}, (ctx: ChartCallbackContext) => ctx.dataset.label + ': ' + formatFileSize(ctx.parsed.y ?? 0) + '/s'),
                    plugins: {
                        ...this.baseOptions().plugins,
                        legend: { display: true, position: 'bottom', labels: { boxWidth: 8, padding: 8, font: { size: 10 }, color: '#64748b' } },
                        tooltip: {
                            backgroundColor: 'rgba(15,23,42,0.9)', titleFont: { size: 10 }, bodyFont: { size: 10 }, padding: 8, cornerRadius: 6,
                            callbacks: { label: (ctx: ChartCallbackContext) => ctx.dataset.label + ': ' + formatFileSize(ctx.parsed.y ?? 0) + '/s' }
                        }
                    }
                } as ChartOptions<'line'>
            })
        }

        if (this.blkRef) {
            this.blkChart = new Chart(this.blkRef, {
                type: 'line' as const,
                data: {
                    labels: [...this.labels],
                    datasets: [
                        { ...this.makeDataset(this.blkRData, '#f59e0b'), label: '读取' },
                        { ...this.makeDataset(this.blkWData, '#d97706'), label: '写入' }
                    ]
                },
                options: {
                    ...this.baseOptions({}, (ctx: ChartCallbackContext) => ctx.dataset.label + ': ' + formatFileSize(ctx.parsed.y ?? 0) + '/s'),
                    plugins: {
                        ...this.baseOptions().plugins,
                        legend: { display: true, position: 'bottom', labels: { boxWidth: 8, padding: 8, font: { size: 10 }, color: '#64748b' } },
                        tooltip: {
                            backgroundColor: 'rgba(15,23,42,0.9)', titleFont: { size: 10 }, bodyFont: { size: 10 }, padding: 8, cornerRadius: 6,
                            callbacks: { label: (ctx: ChartCallbackContext) => ctx.dataset.label + ': ' + formatFileSize(ctx.parsed.y ?? 0) + '/s' }
                        }
                    }
                } as ChartOptions<'line'>
            })
        }
    }

    renderCharts() {
        if (this.cpuChart) { this.cpuChart.data.labels = [...this.labels]; this.cpuChart.data.datasets[0].data = [...this.cpuData]; this.cpuChart.update('none') }
        if (this.memChart) { this.memChart.data.labels = [...this.labels]; this.memChart.data.datasets[0].data = [...this.memData]; this.memChart.update('none') }
        if (this.netChart) { this.netChart.data.labels = [...this.labels]; this.netChart.data.datasets[0].data = [...this.netRxData]; this.netChart.data.datasets[1].data = [...this.netTxData]; this.netChart.update('none') }
        if (this.blkChart) { this.blkChart.data.labels = [...this.labels]; this.blkChart.data.datasets[0].data = [...this.blkRData]; this.blkChart.data.datasets[1].data = [...this.blkWData]; this.blkChart.update('none') }
    }

    destroyCharts() {
        if (this.cpuChart) { this.cpuChart.destroy(); this.cpuChart = null }
        if (this.memChart) { this.memChart.destroy(); this.memChart = null }
        if (this.netChart) { this.netChart.destroy(); this.netChart = null }
        if (this.blkChart) { this.blkChart.destroy(); this.blkChart = null }
    }

    clearHistory() {
        this.labels.length = 0; this.cpuData.length = 0; this.memData.length = 0
        this.netRxData.length = 0; this.netTxData.length = 0; this.blkRData.length = 0; this.blkWData.length = 0
        this.prevNetRx = this.prevNetTx = this.prevBlkR = this.prevBlkW = this.prevTime = 0
        this.netRxRate = this.netTxRate = this.blkRRate = this.blkWRate = 0
    }

    // ─── 侦听器 ───
    @Watch('statsData')
    async onStatsDataChange(val: ContainerStatsResponse | null, old: ContainerStatsResponse | null) {
        if (val && !old) {
            await nextTick()
            this.initCharts()
        }
    }

    // ─── 生命周期 ───
    async mounted() {
        await this.loadContainer()
        document.addEventListener('visibilitychange', this.handleVisibilityChange)

        if (this.container?.state === 'running') {
            await this.loadStats()
            this.statsLoading = false
            this.startStatsTimer()
        } else {
            this.statsLoading = false
        }
    }

    unmounted() {
        this.destroyed = true
        this.stopStatsTimer()
        this.destroyCharts()
        this.clearHistory()
        document.removeEventListener('visibilitychange', this.handleVisibilityChange)
    }
}

export default toNative(ContainerStats)
</script>

<template>
  <div>
    <!-- 顶部导航栏 -->
    <div class="card mb-4">
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3">
        <!-- 桌面端布局 -->
        <div class="hidden md:flex md:items-center justify-between">
          <div class="flex items-center gap-3">
            <button @click="goBack" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-100 flex items-center justify-center text-slate-600 transition-colors" title="返回容器列表">
              <i class="fas fa-arrow-left text-sm"></i>
            </button>
            <template v-if="container">
              <div :class="['w-9 h-9 rounded-lg flex items-center justify-center', container.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
                <i class="fas fa-cube text-white"></i>
              </div>
              <div class="min-w-0">
                <h1 class="text-lg font-semibold text-slate-800 truncate">{{ container.name || container.id }}</h1>
                <p class="text-xs text-slate-500 font-mono truncate">{{ container.image }}</p>
              </div>
            </template>
            <template v-else>
              <div class="w-9 h-9 rounded-lg bg-slate-300 flex items-center justify-center animate-pulse">
                <i class="fas fa-cube text-white"></i>
              </div>
              <div><h1 class="text-lg font-semibold text-slate-800">加载中...</h1></div>
            </template>
          </div>
          <div v-if="container" class="flex gap-1 bg-slate-100 p-1 rounded-lg">
            <button @click="switchTab('docker-container-stats')" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab() === 'docker-container-stats' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
              <i class="fas fa-chart-line"></i><span>监控</span>
            </button>
            <button @click="switchTab('docker-container-logs')" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab() === 'docker-container-logs' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
              <i class="fas fa-file-lines"></i><span>日志</span>
            </button>
            <button @click="switchTab('docker-container-terminal')" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab() === 'docker-container-terminal' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
              <i class="fas fa-terminal"></i><span>终端</span>
            </button>
          </div>
        </div>
        <!-- 移动端布局 -->
        <div class="block md:hidden">
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-3">
              <button @click="goBack" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-100 flex items-center justify-center text-slate-600 transition-colors">
                <i class="fas fa-arrow-left text-sm"></i>
              </button>
              <template v-if="container">
                <div :class="['w-9 h-9 rounded-lg flex items-center justify-center', container.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
                  <i class="fas fa-cube text-white"></i>
                </div>
                <div class="min-w-0">
                  <h1 class="text-lg font-semibold text-slate-800 truncate">{{ container.name || container.id }}</h1>
                  <p class="text-xs text-slate-500 font-mono truncate">{{ container.image }}</p>
                </div>
              </template>
              <template v-else>
                <div class="w-9 h-9 rounded-lg bg-slate-300 flex items-center justify-center animate-pulse">
                  <i class="fas fa-cube text-white"></i>
                </div>
                <div><h1 class="text-lg font-semibold text-slate-800">加载中...</h1></div>
              </template>
            </div>
          </div>
          <div v-if="container" class="flex justify-center gap-1 bg-slate-100 p-1 rounded-lg">
            <button @click="switchTab('docker-container-stats')" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab() === 'docker-container-stats' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
              <i class="fas fa-chart-line"></i><span class="hidden sm:inline">监控</span>
            </button>
            <button @click="switchTab('docker-container-logs')" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab() === 'docker-container-logs' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
              <i class="fas fa-file-lines"></i><span class="hidden sm:inline">日志</span>
            </button>
            <button @click="switchTab('docker-container-terminal')" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', activeTab() === 'docker-container-terminal' ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
              <i class="fas fa-terminal"></i><span class="hidden sm:inline">终端</span>
            </button>
          </div>
        </div>
      </div>
      <!-- 内容区域 -->
      <div class="p-4 md:p-6 space-y-4">
        <!-- 加载状态 -->
        <div v-if="statsLoading && !statsData" class="flex flex-col items-center justify-center py-12 gap-3 text-slate-400 text-sm">
          <div class="w-8 h-8 spinner"></div>
          <span>正在采集数据...</span>
        </div>

        <template v-else-if="statsData">
          <!-- 核心指标：CPU 和 内存 -->
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <!-- CPU 使用率 -->
            <div class="bg-slate-50 rounded-2xl p-4 md:p-5 border border-slate-200/60 relative overflow-hidden">
              <div class="absolute top-0 left-0 right-0 h-0.5 bg-gradient-to-r from-blue-400 to-transparent"></div>
              <div class="flex items-center gap-2 mb-1">
                <div class="w-6 h-6 rounded-lg bg-gradient-to-br from-blue-500 to-blue-600 flex items-center justify-center">
                  <i class="fas fa-microchip text-white text-[9px]"></i>
                </div>
                <span class="text-sm font-semibold text-slate-700">CPU</span>
                <span class="ml-auto text-lg font-bold font-mono" :class="statsData.cpuPercent > 80 ? 'text-red-500' : statsData.cpuPercent > 60 ? 'text-amber-500' : 'text-blue-600'">
                  {{ statsData.cpuPercent }}%
                </span>
              </div>
              <div class="flex flex-wrap items-center gap-3 mb-3 text-[10px] text-slate-400">
                <span v-if="statsData.cpuCores">核心 <span class="text-slate-600 font-medium">{{ statsData.cpuCores }} 核</span></span>
                <span v-if="statsData.cpuFreq">频率 <span class="text-slate-600 font-medium">{{ statsData.cpuFreq.toFixed(0) }} MHz</span></span>
                <span v-if="statsData.cpuThrottled && statsData.cpuThrottled.throttledPeriods > 0" class="text-amber-500">
                  <i class="fas fa-bolt"></i> 节流 <span class="font-medium">{{ statsData.cpuThrottled.throttledPeriods }}</span>
                </span>
              </div>
              <div class="h-28"><canvas ref="cpuRef"></canvas></div>
            </div>

            <!-- 内存使用 -->
            <div class="bg-slate-50 rounded-2xl p-4 md:p-5 border border-slate-200/60 relative overflow-hidden">
              <div class="absolute top-0 left-0 right-0 h-0.5 bg-gradient-to-r from-purple-400 to-transparent"></div>
              <div class="flex items-center gap-2 mb-1">
                <div class="w-6 h-6 rounded-lg bg-gradient-to-br from-purple-500 to-purple-600 flex items-center justify-center">
                  <i class="fas fa-memory text-white text-[9px]"></i>
                </div>
                <span class="text-sm font-semibold text-slate-700">内存</span>
                <span class="ml-auto text-lg font-bold font-mono" :class="statsData.memoryPercent > 80 ? 'text-red-500' : statsData.memoryPercent > 60 ? 'text-amber-500' : 'text-purple-600'">
                  {{ statsData.memoryPercent }}%
                </span>
              </div>
              <div class="flex flex-wrap items-center gap-3 mb-3 text-[10px] text-slate-400">
                <span>内存 <span class="text-slate-600 font-medium">{{ formatFileSize(statsData.memoryUsage) }}</span></span>
                <span>限制 <span class="text-slate-600 font-medium">{{ formatFileSize(statsData.memoryLimit) }}</span></span>
              </div>
              <div class="h-28"><canvas ref="memRef"></canvas></div>
            </div>
          </div>

          <!-- I/O 指标卡片 -->
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <!-- 网络 I/O -->
            <div class="bg-slate-50 rounded-2xl p-4 md:p-5 border border-slate-200/60 relative overflow-hidden">
              <div class="absolute top-0 left-0 right-0 h-0.5 bg-gradient-to-r from-teal-400 to-transparent"></div>
              <div class="flex items-center gap-2 mb-1">
                <div class="w-6 h-6 rounded-lg bg-gradient-to-br from-teal-500 to-teal-600 flex items-center justify-center">
                  <i class="fas fa-network-wired text-white text-[9px]"></i>
                </div>
                <span class="text-sm font-semibold text-slate-700">网络</span>
                <span class="ml-auto text-xs font-mono text-teal-600">
                  <span class="text-teal-500">↓</span> {{ formatFileSize(netRxRate) }}/s
                  <span class="mx-1 text-slate-300">·</span>
                  <span class="text-teal-700">↑</span> {{ formatFileSize(netTxRate) }}/s
                </span>
              </div>
              <div class="flex flex-wrap items-center gap-3 mb-3 text-[10px] text-slate-400">
                <span v-if="statsData.networkDetail">网卡 <span class="text-slate-600 font-medium">{{ Object.keys(statsData.networkDetail).length }} 块</span></span>
                <span>累计收 <span class="text-slate-600 font-medium">{{ formatFileSize(statsData.networkRx) }}</span></span>
                <span>累计发 <span class="text-slate-600 font-medium">{{ formatFileSize(statsData.networkTx) }}</span></span>
              </div>
              <div class="h-28"><canvas ref="netRef"></canvas></div>
            </div>

            <!-- 硬盘 I/O -->
            <div class="bg-slate-50 rounded-2xl p-4 md:p-5 border border-slate-200/60 relative overflow-hidden">
              <div class="absolute top-0 left-0 right-0 h-0.5 bg-gradient-to-r from-amber-400 to-transparent"></div>
              <div class="flex items-center gap-2 mb-1">
                <div class="w-6 h-6 rounded-lg bg-gradient-to-br from-amber-500 to-amber-600 flex items-center justify-center">
                  <i class="fas fa-hard-drive text-white text-[9px]"></i>
                </div>
                <span class="text-sm font-semibold text-slate-700">硬盘</span>
                <span class="ml-auto text-xs font-mono text-amber-600">
                  <span class="text-amber-500">↓</span> {{ formatFileSize(blkRRate) }}/s
                  <span class="mx-1 text-slate-300">·</span>
                  <span class="text-amber-700">↑</span> {{ formatFileSize(blkWRate) }}/s
                </span>
              </div>
              <div class="flex flex-wrap items-center gap-3 mb-3 text-[10px] text-slate-400">
                <span v-if="statsData.blockDetail">设备 <span class="text-slate-600 font-medium">{{ Object.keys(statsData.blockDetail).length }} 个</span></span>
                <span>累计读 <span class="text-slate-600 font-medium">{{ formatFileSize(statsData.blockRead) }}</span></span>
                <span>累计写 <span class="text-slate-600 font-medium">{{ formatFileSize(statsData.blockWrite) }}</span></span>
              </div>
              <div class="h-28"><canvas ref="blkRef"></canvas></div>
            </div>
          </div>

          <!-- 进程信息 -->
          <div class="bg-slate-50 rounded-2xl p-5 border border-slate-200/60 relative overflow-hidden">
            <div class="absolute top-0 left-0 right-0 h-0.5 bg-gradient-to-r from-rose-400 to-transparent"></div>
            <div class="flex items-center gap-2 mb-4">
              <div class="w-7 h-7 rounded-lg bg-gradient-to-br from-rose-500 to-rose-600 flex items-center justify-center">
                <i class="fas fa-list-ol text-white text-[10px]"></i>
              </div>
              <span class="text-sm font-semibold text-slate-700">进程信息</span>
              <span class="ml-auto text-xs text-slate-400">
                <span class="font-semibold text-slate-700">{{ statsData.pids }}</span> 运行中
                <span v-if="statsData.pidsLimit > 0"> / 限制 {{ statsData.pidsLimit }}</span>
              </span>
              <div v-if="statsData.pidsLimit > 0" class="w-20 h-1.5 bg-slate-200 rounded-full overflow-hidden">
                <div
                  class="h-full rounded-full transition-all duration-500"
                  :class="statsData.pids / statsData.pidsLimit > 0.9 ? 'bg-red-500' : statsData.pids / statsData.pidsLimit > 0.7 ? 'bg-amber-500' : 'bg-emerald-500'"
                  :style="{ width: Math.min(statsData.pids / statsData.pidsLimit * 100, 100) + '%' }"
                ></div>
              </div>
            </div>
            <div v-if="statsData.processList && statsData.processList.processes && statsData.processList.processes.length > 0" class="overflow-x-auto max-h-60 overflow-y-auto">
              <table class="w-full text-xs">
                <thead class="sticky top-0 bg-slate-100">
                  <tr>
                    <th v-for="title in statsData.processList.titles" :key="title" class="px-3 py-2 text-left text-[10px] font-semibold text-slate-500 uppercase tracking-wider whitespace-nowrap">
                      {{ title }}
                    </th>
                  </tr>
                </thead>
                <tbody class="bg-white divide-y divide-slate-100">
                  <tr v-for="(proc, idx) in statsData.processList.processes" :key="idx" class="hover:bg-slate-50">
                    <td v-for="(val, vi) in proc" :key="vi" class="px-3 py-1.5 whitespace-nowrap font-mono text-slate-600">{{ val }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div v-else class="text-xs text-slate-400">暂无进程信息</div>
          </div>
        </template>

        <!-- 容器未运行提示 -->
        <div v-else-if="container && container.state !== 'running'" class="flex flex-col items-center justify-center py-12 gap-3 text-slate-400 text-sm">
          <div class="w-12 h-12 rounded-full bg-slate-100 flex items-center justify-center">
            <i class="fas fa-stop text-slate-400 text-lg"></i>
          </div>
          <span>容器未运行，无法采集监控数据</span>
        </div>
      </div>
    </div>
  </div>
</template>
