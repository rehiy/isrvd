<script lang="ts">
import type { Chart as LineChart, Point } from 'chart.js'
import { markRaw } from 'vue'
import { Component, Prop, Ref, Vue, Watch, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { MonitorHostRecord } from '@/service/types'

import Chart, { makeLineDataset, makeMonitorChartOptions, makeMonitorScale } from '@/helper/chart'
import { POLL_INTERVAL, hexToRgba } from '@/helper/format'
import { formatMonitorBytes, hostCpuUsage, monitorCapacity, monitorTimeLabel, usagePercent } from '@/helper/monitor'

@Component({ emits: ['sample'] })
class SystemOverview extends Vue {
    portal = usePortal()

    @Prop({ default: '' }) readonly nodeName!: string

    readonly ranges = [
        { label: '5分钟', seconds: 300 },
        { label: '1小时', seconds: 3600 },
        { label: '6小时', seconds: 21600 }
    ]
    readonly cpuColor = '#0284c7'
    readonly memoryColor = '#7c3aed'
    range = 300
    records: MonitorHostRecord[] = []
    latest: MonitorHostRecord | null = null
    version = ''
    sampleTime = monitorTimeLabel
    historyState: 'loading' | 'ready' | 'failed' = 'loading'
    sampleState: 'loading' | 'ready' | 'failed' = 'loading'
    private chart: LineChart<'line', Point[]> | null = null
    private timer: ReturnType<typeof setInterval> | null = null
    private readonly lifetime = new AbortController()
    private historyRequest: AbortController | null = null
    private sampling = false

    @Ref readonly canvas!: HTMLCanvasElement

    get system() {
        return this.latest?.data.system
    }

    get uptime() {
        const seconds = this.system?.uptime
        if (seconds === undefined) {
            return '—'
        }
        const days = Math.floor(seconds / 86400)
        const hours = Math.floor(seconds / 3600) % 24
        const minutes = Math.floor(seconds / 60) % 60
        return days ? `${days}d ${hours}h` : `${hours}h ${minutes}m`
    }

    get networkRates() {
        const latest = this.latest
        if (!latest) {
            return null
        }
        let index = this.records.length - 1
        while (index >= 0 && this.records[index].ts >= latest.ts) {
            index--
        }
        const baseline = this.records[index]
        if (!baseline) {
            return null
        }
        const period = latest.ts - baseline.ts
        const previous = new Map(baseline.data.system.netInterface?.map(item => [item.name, item]))
        let received = 0
        let sent = 0
        let matched = false
        for (const item of latest.data.system.netInterface ?? []) {
            const old = previous.get(item.name)
            if (old && item.bytesRecv >= old.bytesRecv && item.bytesSent >= old.bytesSent) {
                matched = true
                received += item.bytesRecv - old.bytesRecv
                sent += item.bytesSent - old.bytesSent
            }
        }
        return matched ? { received: received / period, sent: sent / period } : null
    }

    get statCards() {
        const system = this.system
        const memory = monitorCapacity(system?.memoryUsed, system?.memoryTotal)
        const disk = monitorCapacity(system?.diskUsed, system?.diskTotal)
        const cards = [
            {
                label: 'CPU 使用率', icon: 'fa-microchip', color: 'sky', percent: hostCpuUsage(system),
                detail: system ? this.$t('{0} 物理 / {1} 逻辑', system.cpuCore, system.cpuCoreLogic) : '—'
            },
            { label: '内存使用', icon: 'fa-memory', color: 'violet', ...memory, detail: memory.detail || '—' },
            { label: '磁盘使用率', icon: 'fa-hard-drive', color: 'amber', ...disk, detail: disk.detail || this.$t('暂无磁盘数据') }
        ].map(card => ({
            ...card,
            value: card.percent?.toFixed(1) ?? '—',
            unit: card.percent === null ? '' : '%'
        }))
        const network = this.networkRates
        const [download, unit] = network ? formatMonitorBytes(network.received, true).split(' ') : ['—', '']
        return [...cards, {
            label: '网络速率', icon: 'fa-network-wired', color: 'teal', percent: null,
            value: network ? '↓ ' + download : download, unit,
            detail: network ? '↑ ' + formatMonitorBytes(network.sent, true) : this.$t('等待更多采样')
        }]
    }

    load() {
        return Promise.all([this.loadHistory(), this.sample(), this.loadVersion()])
    }

    async loadHistory() {
        this.historyRequest?.abort()
        const request = new AbortController()
        this.historyRequest = request
        this.historyState = 'loading'
        let records: MonitorHostRecord[]
        try {
            const { payload } = await api.overviewMonitor({ since: this.range }, { signal: request.signal })
            records = payload ?? []
        } catch {
            if (!request.signal.aborted) {
                this.historyState = 'failed'
            }
            return
        }
        if (!request.signal.aborted) {
            this.historyState = 'ready'
            this.storeRecords([...records, ...this.records])
        }
    }

    async sample() {
        if (this.sampling) {
            return
        }
        const { signal } = this.lifetime
        this.sampling = true
        let reading: MonitorHostRecord | null | undefined
        try {
            const { payload } = await api.overviewMonitor({ since: 0 }, { signal })
            reading = payload?.data?.system ? payload : null
        } catch {
            reading = null
        } finally {
            this.sampling = false
        }
        if (signal.aborted) {
            return
        }
        this.sampleState = reading ? 'ready' : 'failed'
        this.$emit('sample', reading)
        if (reading) {
            this.latest = reading
            this.storeRecords([...this.records, reading])
        }
    }

    async loadVersion() {
        if (this.portal.hasPerm('GET /api/overview/version')) {
            const { payload } = await api.overviewVersion({ params: { current: 1 }, signal: this.lifetime.signal })
            this.version = payload?.current ?? ''
        }
    }

    storeRecords(records: MonitorHostRecord[]) {
        const distinct = new Map(records.filter(record => record.data?.system).map(record => [record.ts, record]))
        const sorted = [...distinct.values()].sort((a, b) => a.ts - b.ts)
        const cutoff = (sorted[sorted.length - 1]?.ts ?? 0) - this.range
        this.records = sorted.filter(record => record.ts >= cutoff)
        this.drawChart()
    }

    async switchRange(seconds: number) {
        if (this.range !== seconds) {
            this.range = seconds
            this.storeRecords(this.records)
            await this.loadHistory()
        }
    }

    @Watch('portal.locale')
    drawChart() {
        const chart = this.chart
        if (!chart) {
            return
        }
        chart.data.datasets[0].label = this.$t('CPU 使用率')
        chart.data.datasets[0].data = this.records.flatMap(({ ts, data }) => {
            const usage = hostCpuUsage(data.system)
            return usage === null ? [] : [{ x: ts, y: usage }]
        })
        chart.data.datasets[1].label = this.$t('内存使用')
        chart.data.datasets[1].data = this.records.flatMap(({ ts, data }) => {
            const { memoryUsed, memoryTotal } = data.system
            return memoryTotal ? [{ x: ts, y: usagePercent(memoryUsed, memoryTotal) }] : []
        })
        const timeAxis = chart.options.scales?.x
        const windowEnd = this.records[this.records.length - 1]?.ts
        if (timeAxis && windowEnd !== undefined) {
            timeAxis.min = windowEnd - this.range
            timeAxis.max = windowEnd
        }
        chart.update('none')
    }

    handleVisibility() {
        if (!document.hidden) {
            this.load()
        }
    }

    mounted() {
        const options = makeMonitorChartOptions({
            x: {
                type: 'linear',
                border: { display: false },
                grid: { display: false },
                ticks: { maxTicksLimit: 6, color: '#75849b', callback: value => monitorTimeLabel(Number(value)) }
            },
            y: makeMonitorScale('percent', { max: undefined, suggestedMax: 20, ticks: { stepSize: undefined, maxTicksLimit: 5 } })
        }, context => `${context.dataset.label}  ${context.parsed.y?.toFixed(1) ?? '—'}%`)
        options.interaction = { intersect: false, mode: 'nearest', axis: 'x' }
        const cpuFill = ({ chart }: { chart: LineChart }) => {
            const area = chart.chartArea
            if (!area) {
                return hexToRgba(this.cpuColor, 0.12)
            }
            const gradient = chart.ctx.createLinearGradient(0, area.top, 0, area.bottom)
            gradient.addColorStop(0, hexToRgba(this.cpuColor, 0.24))
            gradient.addColorStop(1, hexToRgba(this.cpuColor, 0.015))
            return gradient
        }
        const callbacks = options.plugins?.tooltip?.callbacks
        if (callbacks) {
            callbacks.title = items => items.length ? monitorTimeLabel(items[0].parsed.x ?? 0) : ''
        }
        this.chart = markRaw(new Chart(this.canvas, {
            type: 'line',
            data: {
                datasets: [
                    makeLineDataset<Point>([], this.cpuColor, '', { backgroundColor: cpuFill, tension: 0.22 }),
                    makeLineDataset<Point>([], this.memoryColor, '', { fill: false, tension: 0.22 })
                ]
            },
            options
        }))
        this.load()
        document.addEventListener('visibilitychange', this.handleVisibility)
        this.timer = setInterval(() => {
            if (!document.hidden) {
                this.sample()
            }
        }, POLL_INTERVAL)
    }

    unmounted() {
        this.lifetime.abort()
        this.historyRequest?.abort()
        this.chart?.destroy()
        if (this.timer) {
            clearInterval(this.timer)
        }
        document.removeEventListener('visibilitychange', this.handleVisibility)
    }
}

export default toNative(SystemOverview)
</script>

<template>
  <section class="space-y-4">
    <div class="card resource-panel">
      <div class="resource-section-header">
        <div class="resource-section-heading max-w-full">
          <div class="row-icon resource-section-icon">
            <i class="fas fa-server"></i>
          </div>
          <div class="min-w-0">
            <div class="flex flex-wrap items-baseline gap-x-2 gap-y-1">
              <h2 class="resource-section-title">{{ $t('当前主机') }}</h2>
              <span class="truncate text-sm text-slate-600" :title="nodeName || system?.hostName">{{ nodeName || system?.hostName || '—' }}</span>
            </div>
            <p v-if="system" class="text-xs resource-host-muted truncate mt-0.5">
              <template v-if="nodeName">{{ system.hostName }} <span class="mx-1">·</span></template>
              {{ system.platform }} <span class="mx-1">·</span> {{ system.kernelArch }}
            </p>
          </div>
        </div>
        <div class="resource-host-status text-xs tabular-nums">
          <span v-if="sampleState === 'loading'">{{ $t('加载中...') }}</span>
          <span v-else-if="sampleState === 'failed'" class="text-amber-700">{{ $t('采样失败') }}</span>
          <template v-else-if="latest">
            <span class="resource-live-dot size-1.5 shrink-0 rounded-full"></span>
            <span>{{ $t('实时采集') }}</span>
            <time class="resource-host-muted font-mono">{{ sampleTime(latest.ts) }}</time>
          </template>
        </div>
      </div>

      <div class="mt-4 grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
        <div v-for="card in statCards" :key="card.label" :class="['stat-card min-w-0', 'resource-metric-' + card.color]">
          <div class="page-icon resource-metric-icon">
            <i :class="['fas', card.icon]"></i>
          </div>
          <div class="min-w-0 flex-1">
            <p class="text-xs font-medium resource-host-muted">{{ $t(card.label) }}</p>
            <p class="resource-metric-value leading-tight font-semibold tracking-tight tabular-nums mt-1">
              {{ card.value }}<span v-if="card.unit" class="resource-metric-unit">{{ card.unit }}</span>
            </p>
            <p class="text-xs resource-host-muted mt-1 truncate tabular-nums" :title="card.detail">{{ card.detail }}</p>
            <div v-if="card.percent !== null" class="resource-meter mt-2.5" aria-hidden="true">
              <span :style="{ width: Math.min(100, Math.max(0, card.percent)) + '%' }"></span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="grid gap-4 xl:grid-cols-[minmax(0,2fr)_minmax(18rem,1fr)]">
      <div class="card resource-panel">
        <div class="resource-section-header">
          <div class="resource-section-heading">
            <span class="row-icon resource-section-icon"><i class="fas fa-wave-square"></i></span>
            <h2 class="resource-section-title">{{ $t('资源趋势') }}</h2>
          </div>
          <div class="tab-group">
            <button
              v-for="item in ranges" :key="item.seconds" type="button"
              :class="['tab-btn', range === item.seconds ? 'tab-btn-active text-primary-600' : 'tab-btn-inactive']"
              :aria-pressed="range === item.seconds"
              @click="switchRange(item.seconds)"
            >
              {{ $t(item.label) }}
            </button>
          </div>
        </div>
        <div class="flex gap-4 text-xs font-medium text-slate-600 mt-3">
          <span class="inline-flex items-center gap-1.5"><span class="size-2 rounded-full" :style="{ backgroundColor: cpuColor }"></span>CPU</span>
          <span class="inline-flex items-center gap-1.5"><span class="size-2 rounded-full" :style="{ backgroundColor: memoryColor }"></span>{{ $t('内存') }}</span>
        </div>
        <div class="relative h-40 sm:h-[10.75rem] mt-3">
          <canvas ref="canvas" role="img" :aria-label="$t('CPU 和内存使用率趋势')"></canvas>
          <div v-if="records.length < 2" class="absolute inset-0 grid place-items-center bg-white text-sm text-slate-500 text-center">
            {{ $t(historyState === 'loading' ? '加载中...' : historyState === 'failed' ? '获取监控数据失败' : '等待更多采样') }}
          </div>
        </div>
        <p v-if="historyState === 'failed' && records.length >= 2" class="mt-2 text-xs text-amber-700">{{ $t('获取监控数据失败') }}</p>
      </div>

      <div class="card resource-panel flex flex-col">
        <div class="resource-section-heading">
          <span class="row-icon resource-section-icon"><i class="fas fa-circle-info"></i></span>
          <h2 class="resource-section-title">{{ $t('主机详情') }}</h2>
        </div>
        <dl class="mt-2 divide-y divide-slate-100 text-sm flex-1">
          <div class="flex flex-wrap items-baseline justify-between gap-x-2 py-2.5">
            <dt class="text-slate-500">{{ $t('运行时间') }}</dt><dd class="font-medium text-slate-800 tabular-nums">{{ uptime }}</dd>
          </div>
          <div class="flex flex-wrap items-baseline justify-between gap-x-2 py-2.5">
            <dt class="text-slate-500">{{ $t('系统时间') }}</dt>
            <dd class="text-right font-medium text-slate-800 tabular-nums">
              <span class="block">{{ latest?.data.time || '—' }}</span>
              <span v-if="latest?.data.timezone" class="block text-xs font-normal text-slate-500">{{ latest.data.timezone }}</span>
            </dd>
          </div>
          <div v-if="version" class="flex flex-wrap items-baseline justify-between gap-x-2 py-2.5">
            <dt class="text-slate-500">{{ $t('程序版本') }}</dt><dd class="font-medium text-slate-800 break-all">{{ version }}</dd>
          </div>
          <div v-if="latest?.data.gpu?.length" class="flex flex-wrap items-baseline justify-between gap-x-2 py-2.5">
            <dt class="text-slate-500">GPU</dt><dd class="font-medium text-slate-800">{{ latest.data.gpu.length }}</dd>
          </div>
        </dl>
        <router-link to="/local/monitor" class="btn btn-secondary resource-detail-link mt-3">
          <i class="fas fa-chart-line"></i>{{ $t('系统监控') }}<i class="fas fa-arrow-right text-xs ml-auto"></i>
        </router-link>
      </div>
    </div>
  </section>
</template>
