<script lang="ts">
import type { TooltipItem } from 'chart.js'
import { markRaw, nextTick } from 'vue'
import { Component, Ref, Vue, Watch, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { DockerContainerInfo, DockerContainerStats } from '@/service/types'

import Chart, { makeLineDataset, makeMonitorChartOptions, makeMonitorScale } from '@/helper/chart'
import { POLL_INTERVAL } from '@/helper/format'
import { containerHistoryValues, formatMonitorBytes, semanticColor } from '@/helper/monitor'

@Component({
    beforeRouteLeave(this: unknown) {
        (this as InstanceType<typeof ContainerStats>).stopStatsTimer()
    }
} as Record<string, unknown>)
class ContainerStats extends Vue {
    portal = usePortal()

    // ─── Refs ───
    @Ref readonly cpuRef!: HTMLCanvasElement
    @Ref readonly memRef!: HTMLCanvasElement
    @Ref readonly netRef!: HTMLCanvasElement
    @Ref readonly blkRef!: HTMLCanvasElement

    // ─── 数据属性 ───
    container: DockerContainerInfo | null = null
    statsData: DockerContainerStats | null = null
    statsLoading = true
    statsFailed = false
    netRxRate = 0
    netTxRate = 0
    blkRRate = 0
    blkWRate = 0
    formatMonitorBytes = formatMonitorBytes
    semanticColor = semanticColor

    // ─── 私有属性（非响应式） ───
    private statsTimer: ReturnType<typeof setInterval> | null = null
    private destroyed = false
    private generation = 0
    private statsInFlight = false
    private monitoredName = ''
    private prevContainerId = ''
    private readonly MAX_POINTS = 720
    private timestamps: number[] = []
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
        return this.container?.id || this.$route.params.id as string
    }

    @Watch('$route.fullPath')
    onRouteChanged() {
        this.loadContainer()
    }

    // ─── 方法 ───
    startStatsTimer() {
        this.stopStatsTimer()
        if (!this.destroyed && !document.hidden) {
            this.statsTimer = setInterval(() => this.loadStats(), POLL_INTERVAL)
        }
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
        } else if (this.monitoredName && !this.statsLoading) {
            this.loadStats()
            this.startStatsTimer()
        }
    }

    async loadHistory(generation = this.generation) {
        if (!this.portal.hasPerm('GET /api/overview/monitor')) {
            return
        }
        try {
            const res = await api.overviewMonitor({ type: 'container', id: this.containerId, name: this.container?.name, since: 3600 })
            if (this.destroyed || generation !== this.generation) {
                return
            }
            for (const record of [...res.payload || []].sort((a, b) => a.ts - b.ts)) {
                this.pushPoint(containerHistoryValues(record), record.ts * 1000, record.container_id || record.data.id || this.containerId)
            }
        } catch {
            /* ignore */
        }
    }

    async loadStats(generation = this.generation, recover = true) {
        if (this.destroyed || generation !== this.generation || this.statsInFlight) {
            return
        }
        this.statsInFlight = true
        try {
            if (!this.container || this.container.state !== 'running') {
                await this.recoverContainer(generation)
                return
            }
            const res = await api.dockerContainerStats(this.containerId)
            if (this.destroyed || generation !== this.generation || !res.payload) {
                return
            }
            const isFirst = !this.statsData
            this.statsData = res.payload
            this.statsFailed = false
            this.pushPoint(res.payload, res.payload.ts * 1000)
            if (isFirst) {
                await nextTick()
                if (this.destroyed || generation !== this.generation) {
                    return
                }
                this.initCharts()
            }
            this.renderCharts()
        } catch {
            if (!this.destroyed && generation === this.generation) {
                this.statsFailed = true
                if (recover) {
                    await this.recoverContainer(generation)
                }
            }
        } finally {
            if (generation === this.generation) {
                this.statsInFlight = false
            }
        }
    }

    pushPoint(data: ReturnType<typeof containerHistoryValues>, nowMs: number, instanceId = this.containerId) {
        if (!Number.isFinite(nowMs) || nowMs <= this.prevTime) {
            return
        }
        const now = new Date(nowMs)
        const label = now.getHours().toString().padStart(2, '0') + ':' +
            now.getMinutes().toString().padStart(2, '0') + ':' +
            now.getSeconds().toString().padStart(2, '0')

        const sameInstance = !!this.prevContainerId && (this.prevContainerId.startsWith(instanceId) || instanceId.startsWith(this.prevContainerId))
        const elapsed = this.prevTime > 0 && sameInstance ? (nowMs - this.prevTime) / 1000 : 0

        this.timestamps.push(nowMs)
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
            this.netRxRate = this.netTxRate = this.blkRRate = this.blkWRate = 0
        }

        this.prevContainerId = instanceId
        this.prevNetRx = data.networkRx || 0
        this.prevNetTx = data.networkTx || 0
        this.prevBlkR = data.blockRead || 0
        this.prevBlkW = data.blockWrite || 0
        this.prevTime = nowMs

        while (this.labels.length > this.MAX_POINTS || this.timestamps[0] < nowMs - 3600_000) {
            this.timestamps.shift()
            this.labels.shift()
            this.cpuData.shift()
            this.memData.shift()
            this.netRxData.shift()
            this.netTxData.shift()
            this.blkRData.shift()
            this.blkWData.shift()
        }
    }

    initCharts() {
        this.destroyCharts()

        const pctTooltipCb = (ctx: TooltipItem<'line'>) => (ctx.parsed.y ?? 0).toFixed(1) + '%'
        const ioTooltipCb = (ctx: TooltipItem<'line'>) => ctx.dataset.label + ': ' + formatMonitorBytes(ctx.parsed.y ?? 0, true)

        if (this.cpuRef) {
            this.cpuChart = markRaw(new Chart(this.cpuRef, {
                type: 'line' as const,
                data: { labels: [...this.labels], datasets: [makeLineDataset(this.cpuData, '#3b82f6', '', { opacity: 0.6, fillOpacity: 0.08 })] },
                options: makeMonitorChartOptions({ y: makeMonitorScale('percent', { max: undefined, suggestedMax: 100 }) }, pctTooltipCb)
            }))
        }

        if (this.memRef) {
            this.memChart = markRaw(new Chart(this.memRef, {
                type: 'line' as const,
                data: { labels: [...this.labels], datasets: [makeLineDataset(this.memData, '#6366f1', '', { opacity: 0.6, fillOpacity: 0.08 })] },
                options: makeMonitorChartOptions({ y: makeMonitorScale('percent') }, pctTooltipCb)
            }))
        }

        if (this.netRef) {
            this.netChart = markRaw(new Chart(this.netRef, {
                type: 'line' as const,
                data: {
                    labels: [...this.labels],
                    datasets: [
                        makeLineDataset(this.netRxData, '#10b981', this.$t('下行')),
                        makeLineDataset(this.netTxData, '#3b82f6', this.$t('上行'))
                    ]
                },
                options: makeMonitorChartOptions({ y: makeMonitorScale('rate') }, ioTooltipCb)
            }))
        }

        if (this.blkRef) {
            this.blkChart = markRaw(new Chart(this.blkRef, {
                type: 'line' as const,
                data: {
                    labels: [...this.labels],
                    datasets: [
                        makeLineDataset(this.blkRData, '#f59e0b', this.$t('读取')),
                        makeLineDataset(this.blkWData, '#8b5cf6', this.$t('写入'))
                    ]
                },
                options: makeMonitorChartOptions({ y: makeMonitorScale('rate') }, ioTooltipCb)
            }))
        }
    }

    renderCharts() {
        if (this.cpuChart) {
            this.cpuChart.data.labels = [...this.labels]
            this.cpuChart.data.datasets[0].data = [...this.cpuData]
            this.cpuChart.update('none')
        }
        if (this.memChart) {
            this.memChart.data.labels = [...this.labels]
            this.memChart.data.datasets[0].data = [...this.memData]
            this.memChart.update('none')
        }
        if (this.netChart) {
            this.netChart.data.labels = [...this.labels]
            this.netChart.data.datasets[0].data = [...this.netRxData]
            this.netChart.data.datasets[1].data = [...this.netTxData]
            this.netChart.update('none')
        }
        if (this.blkChart) {
            this.blkChart.data.labels = [...this.labels]
            this.blkChart.data.datasets[0].data = [...this.blkRData]
            this.blkChart.data.datasets[1].data = [...this.blkWData]
            this.blkChart.update('none')
        }
    }

    destroyCharts() {
        if (this.cpuChart) {
            this.cpuChart.destroy()
            this.cpuChart = null
        }
        if (this.memChart) {
            this.memChart.destroy()
            this.memChart = null
        }
        if (this.netChart) {
            this.netChart.destroy()
            this.netChart = null
        }
        if (this.blkChart) {
            this.blkChart.destroy()
            this.blkChart = null
        }
    }

    clearHistory() {
        this.timestamps.length = 0
        this.labels.length = 0; this.cpuData.length = 0; this.memData.length = 0
        this.netRxData.length = 0; this.netTxData.length = 0; this.blkRData.length = 0; this.blkWData.length = 0
        this.prevNetRx = this.prevNetTx = this.prevBlkR = this.prevBlkW = this.prevTime = 0
        this.prevContainerId = ''
        this.netRxRate = this.netTxRate = this.blkRRate = this.blkWRate = 0
    }

    async loadContainer(name = typeof this.$route.query.name === 'string' ? this.$route.query.name : '') {
        if (this.destroyed) {
            return
        }
        const generation = ++this.generation
        this.stopStatsTimer()
        this.destroyCharts()
        this.clearHistory()
        this.container = null
        this.statsData = null
        this.statsInFlight = false
        this.statsLoading = true
        this.statsFailed = false
        this.monitoredName = name
        try {
            const res = await api.dockerContainerList(true)
            if (this.destroyed || generation !== this.generation) {
                return
            }
            const id = this.$route.params.id as string
            const list = res.payload || []
            const ct = list.find(c => c.id === id || c.name === id || (id.length === 64 && id.startsWith(c.id)))
                || list.find(c => name && c.name === name)
            if (!ct) {
                if (name) {
                    this.statsFailed = true
                    this.statsLoading = false
                    this.startStatsTimer()
                } else {
                    this.portal.showNotification('error', this.$t('容器不存在'))
                    this.$router.push('/docker/containers')
                }
                return
            }
            await this.onContainerLoaded(ct, generation)
        } catch {
            if (!this.destroyed && generation === this.generation) {
                this.statsFailed = true
                this.statsLoading = false
                if (name) {
                    this.startStatsTimer()
                }
            }
        }
    }

    async recoverContainer(generation: number) {
        if (!this.monitoredName) {
            return
        }
        try {
            const res = await api.dockerContainerList(true)
            if (this.destroyed || generation !== this.generation) {
                return
            }
            const ct = (res.payload || []).find(c => c.name === this.monitoredName)
            if (ct && (ct.id !== this.container?.id || ct.state !== this.container.state)) {
                await this.loadContainer(this.monitoredName)
            }
        } catch {
            /* ignore */
        }
    }

    async onContainerLoaded(ct: DockerContainerInfo, generation = this.generation) {
        this.container = ct
        this.monitoredName = ct.name
        if (ct.state === 'running') {
            await this.loadHistory(generation)
            if (this.destroyed || generation !== this.generation) {
                return
            }
            await this.loadStats(generation, false)
        }
        if (!this.destroyed && generation === this.generation) {
            this.statsLoading = false
            if (!document.hidden && this.monitoredName) {
                this.startStatsTimer()
            }
        }
    }

    // ─── 生命周期 ───
    mounted() {
        document.addEventListener('visibilitychange', this.handleVisibilityChange)
        this.loadContainer()
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
  <div class="page">
    <div class="page-toolbar">
      <!-- 桌面端 -->
      <div class="toolbar-desktop">
        <div class="flex items-center gap-3">
          <div :class="['page-icon', container?.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
            <i class="fas fa-cube text-white text-sm"></i>
          </div>
          <div class="min-w-0">
            <h1 class="title-text">{{ container ? (container.name || container.id) : $t('加载中...') }}</h1>
            <p class="text-xs text-slate-600 font-mono truncate">{{ container?.image }}</p>
          </div>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="toolbar-mobile">
        <div class="title-group">
          <div :class="['page-icon flex-shrink-0', container?.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
            <i class="fas fa-cube text-white text-sm"></i>
          </div>
          <div class="min-w-0">
            <h1 class="title-text">{{ container ? (container.name || container.id) : $t('加载中...') }}</h1>
            <p class="text-xs text-slate-600 font-mono truncate">{{ container?.image }}</p>
          </div>
        </div>
      </div>
    </div>
    <!-- 内容区域 -->
    <div class="card-body space-y-4">
      <p v-if="statsFailed" class="text-sm text-amber-600" role="status">{{ $t('加载失败') }} · {{ $t('请检查服务状态后刷新重试') }}</p>
      <!-- 加载状态 -->
      <div v-if="statsLoading && !statsData" class="empty-state gap-3 text-slate-400 text-sm">
        <div class="w-8 h-8 spinner"></div>
        <span>{{ $t('正在采集数据...') }}</span>
      </div>

      <template v-else-if="statsData">
        <!-- 核心指标：CPU 和 内存 -->
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <!-- CPU 使用率 -->
          <div class="metric-card min-w-0">
            <div class="absolute top-0 left-0 right-0 h-0.5 bg-gradient-to-r from-blue-400 to-transparent"></div>
            <div class="monitor-card-header">
              <div class="monitor-card-heading">
                <div class="monitor-card-icon">
                  <i class="fas fa-microchip text-sm"></i>
                </div>
                <div class="min-w-0 flex-1">
                  <p class="monitor-card-title">{{ $t('CPU 使用率') }}</p>
                  <div v-if="statsData.cpuCores || statsData.cpuFreq || (statsData.cpuThrottled && statsData.cpuThrottled.throttledPeriods > 0)" class="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-400">
                    <span v-if="statsData.cpuCores">{{ $t('核心') }} <span class="tabular-nums">{{ statsData.cpuCores }} {{ $t('核') }}</span></span>
                    <span v-if="statsData.cpuFreq">{{ $t('频率') }} <span class="tabular-nums">{{ statsData.cpuFreq.toFixed(0) }} MHz</span></span>
                    <span v-if="statsData.cpuThrottled && statsData.cpuThrottled.throttledPeriods > 0" class="text-amber-500">
                      {{ $t('节流') }} <span class="tabular-nums">{{ statsData.cpuThrottled.throttledPeriods }}</span>
                    </span>
                  </div>
                  <p v-else class="text-xs text-slate-300">—</p>
                </div>
              </div>
              <span :class="['monitor-card-value ml-auto whitespace-nowrap text-right', semanticColor(statsData.cpuPercent)]">{{ statsData.cpuPercent }}%</span>
            </div>
            <div class="monitor-chart-box">
              <canvas ref="cpuRef" class="w-full h-full"></canvas>
            </div>
          </div>

          <!-- 内存使用 -->
          <div class="metric-card min-w-0">
            <div class="absolute top-0 left-0 right-0 h-0.5 bg-gradient-to-r from-indigo-400 to-transparent"></div>
            <div class="monitor-card-header">
              <div class="monitor-card-heading">
                <div class="monitor-card-icon">
                  <i class="fas fa-memory text-sm"></i>
                </div>
                <div class="min-w-0 flex-1">
                  <p class="monitor-card-title">{{ $t('内存使用') }}</p>
                  <p class="break-words text-xs text-slate-400" :title="$t('内存') + ' / ' + $t('限制')">{{ formatMonitorBytes(statsData.memoryUsage) }} / {{ formatMonitorBytes(statsData.memoryLimit) }}</p>
                </div>
              </div>
              <span :class="['monitor-card-value ml-auto whitespace-nowrap text-right', semanticColor(statsData.memoryPercent)]">{{ statsData.memoryPercent }}%</span>
            </div>
            <div class="monitor-chart-box">
              <canvas ref="memRef" class="w-full h-full"></canvas>
            </div>
          </div>
        </div>

        <!-- I/O 指标卡片 -->
        <div class="grid grid-cols-1 gap-4">
          <!-- 硬盘 I/O -->
          <div class="metric-card min-w-0">
            <div class="absolute inset-x-0 top-0 h-0.5 bg-gradient-to-r from-amber-400 to-transparent"></div>
            <div class="monitor-card-header">
              <div class="monitor-card-heading">
                <div class="monitor-card-icon">
                  <i class="fas fa-hard-drive text-sm"></i>
                </div>
                <div class="min-w-0 flex-1">
                  <p class="monitor-card-title">{{ $t('硬盘') }}</p>
                  <p v-if="statsData.blockDetail" class="mt-1 break-words text-xs text-slate-400">{{ $t('设备') }} <span class="tabular-nums">{{ Object.keys(statsData.blockDetail).length }} {{ $t('个') }}</span></p>
                </div>
              </div>
              <div class="monitor-card-metrics">
                <div class="monitor-metrics-row text-slate-600">
                  <span class="monitor-series"><span class="monitor-series-dot bg-amber-500"></span>{{ $t('读取') }} <span class="monitor-card-value text-slate-700">{{ formatMonitorBytes(blkRRate, true) }}</span></span>
                  <span class="monitor-series"><span class="monitor-series-dot bg-violet-500"></span>{{ $t('写入') }} <span class="monitor-card-value text-slate-700">{{ formatMonitorBytes(blkWRate, true) }}</span></span>
                </div>
                <div class="monitor-metrics-row">
                  <span>{{ $t('累计读') }} <span class="tabular-nums">{{ formatMonitorBytes(statsData.blockRead) }}</span></span>
                  <span>{{ $t('累计写') }} <span class="tabular-nums">{{ formatMonitorBytes(statsData.blockWrite) }}</span></span>
                </div>
              </div>
            </div>
            <div class="monitor-chart-box">
              <canvas ref="blkRef" class="w-full h-full"></canvas>
            </div>
          </div>

          <!-- 网络 I/O -->
          <div class="metric-card min-w-0">
            <div class="absolute inset-x-0 top-0 h-0.5 bg-gradient-to-r from-cyan-400 to-transparent"></div>
            <div class="monitor-card-header">
              <div class="monitor-card-heading">
                <div class="monitor-card-icon">
                  <i class="fas fa-network-wired text-sm"></i>
                </div>
                <div class="min-w-0 flex-1">
                  <p class="monitor-card-title">{{ $t('网络') }}</p>
                  <p v-if="statsData.networkDetail" class="mt-1 break-words text-xs text-slate-400">{{ $t('网卡') }} <span class="tabular-nums">{{ Object.keys(statsData.networkDetail).length }} {{ $t('块') }}</span></p>
                </div>
              </div>
              <div class="monitor-card-metrics">
                <div class="monitor-metrics-row text-slate-600">
                  <span class="monitor-series"><span class="monitor-series-dot bg-emerald-500"></span>{{ $t('下行') }} <span class="monitor-card-value text-slate-700">{{ formatMonitorBytes(netRxRate, true) }}</span></span>
                  <span class="monitor-series"><span class="monitor-series-dot bg-blue-500"></span>{{ $t('上行') }} <span class="monitor-card-value text-slate-700">{{ formatMonitorBytes(netTxRate, true) }}</span></span>
                </div>
                <div class="monitor-metrics-row">
                  <span>{{ $t('累计收') }} <span class="tabular-nums">{{ formatMonitorBytes(statsData.networkRx) }}</span></span>
                  <span>{{ $t('累计发') }} <span class="tabular-nums">{{ formatMonitorBytes(statsData.networkTx) }}</span></span>
                </div>
              </div>
            </div>
            <div class="monitor-chart-box">
              <canvas ref="netRef" class="w-full h-full"></canvas>
            </div>
          </div>
        </div>

        <!-- 进程信息 -->
        <div class="metric-card min-w-0">
          <div class="absolute inset-x-0 top-0 h-0.5 bg-gradient-to-r from-rose-400 to-transparent"></div>
          <div class="monitor-card-header">
            <div class="monitor-card-heading">
              <div class="monitor-card-icon">
                <i class="fas fa-list-ol text-sm"></i>
              </div>
              <span class="min-w-0 flex-1 monitor-card-title">{{ $t('进程信息') }}</span>
            </div>
            <div class="monitor-card-metrics">
              <span v-if="statsData.processList" class="inline-flex items-baseline gap-1.5 whitespace-nowrap">
                <span class="text-xs text-slate-400">{{ $t('进程') }}</span>
                <span class="monitor-card-value text-slate-700">{{ statsData.processList.processes?.length ?? 0 }}</span>
              </span>
              <span v-else class="text-xs text-slate-400">{{ $t('暂无进程信息') }}</span>
              <div class="monitor-metrics-row">
                <span>{{ $t('线程（含主线程）') }} <span class="tabular-nums text-slate-600">{{ statsData.pids }}</span></span>
                <span v-if="statsData.pidsLimit > 0">{{ $t('限制') }} <span class="tabular-nums text-slate-600">{{ statsData.pidsLimit }}</span></span>
                <div v-if="statsData.pidsLimit > 0" class="w-20 h-1.5 bg-slate-200 rounded-full overflow-hidden">
                  <div
                    class="h-full rounded-full transition-all duration-500"
                    :class="statsData.pids / statsData.pidsLimit > 0.9 ? 'bg-red-500' : statsData.pids / statsData.pidsLimit > 0.7 ? 'bg-amber-500' : 'bg-emerald-500'"
                    :style="{ width: Math.min(statsData.pids / statsData.pidsLimit * 100, 100) + '%' }"
                  ></div>
                </div>
              </div>
            </div>
          </div>
          <div v-if="statsData.processList && statsData.processList.processes && statsData.processList.processes.length > 0" class="overflow-x-auto">
            <table class="w-full text-xs">
              <thead class="bg-slate-100">
                <tr>
                  <th v-for="title in statsData.processList.titles" :key="title" class="th-sm whitespace-nowrap">
                    {{ title }}
                  </th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-100">
                <tr v-for="(proc, idx) in statsData.processList.processes" :key="idx" class="hover:bg-slate-50">
                  <td v-for="(val, vi) in proc" :key="vi" class="px-3 py-1.5 whitespace-nowrap font-mono text-slate-600">{{ val }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div v-else class="text-xs text-slate-400">{{ $t('暂无进程信息') }}</div>
        </div>
      </template>

      <!-- 容器未运行提示 -->
      <div v-else-if="container && container.state !== 'running'" class="flex flex-col items-center justify-center py-12 gap-3 text-slate-400 text-sm">
        <div class="empty-state-icon !mb-0">
          <i class="fas fa-stop text-slate-400 text-lg"></i>
        </div>
        <span>{{ $t('容器未运行，无法采集监控数据') }}</span>
      </div>
    </div>
  </div>
</template>
