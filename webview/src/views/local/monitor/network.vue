<script lang="ts">
import type { ChartOptions } from 'chart.js'
import { markRaw } from 'vue'
import { Component, Prop, Ref, Vue, toNative } from 'vue-facing-decorator'

import type { SystemStat, SystemNetInterface } from '@/service/types'

import Chart from '@/helper/chart'
import { appendMonitorPoint } from '@/helper/monitor'
import { hexToRgba } from '@/helper/utils'

interface TimeSeriesHistory {
    ts: number[]
    labels: string[]
    recv: number[]
    sent: number[]
}

interface ChartCallbackContext {
    parsed: { y: number | null }
    dataset: { label?: string }
}

const NATURAL_COLLATOR = new Intl.Collator(undefined, { numeric: true, sensitivity: 'base' })

@Component
class SystemNetwork extends Vue {
    @Prop({ type: Number, default: 300 }) readonly rangeSeconds!: number

    @Ref readonly netContainerRef!: HTMLDivElement

    private netCharts: Record<string, Chart<'line'>> = {}
    netHistory: Record<string, TimeSeriesHistory> = {}
    private lastNetIO: Record<string, { recv: number; sent: number; time: number }> = {}
    currentIfaces: SystemNetInterface[] = []

    fmtSize(bytes: number, rates = false) {
        if (!bytes || bytes < 0) return rates ? '0 B/s' : '0 B'
        const units = rates ? ['B/s', 'KB/s', 'MB/s', 'GB/s'] : ['B', 'KB', 'MB', 'GB', 'TB']
        let i = 0, v = bytes
        while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
        return `${v.toFixed(1)} ${units[i]}`
    }

    fmtBytes(b: number) { return this.fmtSize(b, false) }
    fmtRate(b: number) { return this.fmtSize(b, true) }

    physicalInterfaces(list: SystemNetInterface[]) {
        if (!list) return []
        const virtualPrefixes = ['lo', 'docker', 'veth', 'br-', 'overlay', 'flannel', 'cni', 'tunl', 'dummy', 'virbr']
        return list
            .filter(ni => !virtualPrefixes.some(p => ni.name.startsWith(p)))
            .sort((a, b) => NATURAL_COLLATOR.compare(a.name, b.name))
    }

    currentRate(name: string, dir: string): number {
        const h = this.netHistory[name]
        if (!h) return 0
        const arr = h[dir as keyof TimeSeriesHistory] as number[]
        return arr?.length ? arr[arr.length - 1] : 0
    }

    netChartOptions(): ChartOptions<'line'> {
        const fmtRate = (v: number) => this.fmtRate(v)
        return {
            responsive: true, maintainAspectRatio: false, animation: false,
            interaction: { intersect: false, mode: 'index' as const },
            plugins: {
                legend: { display: true, position: 'bottom' as const, labels: { boxWidth: 8, padding: 8, font: { size: 10 }, color: '#64748b' } },
                tooltip: {
                    backgroundColor: 'rgba(15,23,42,0.9)', titleFont: { size: 10 }, bodyFont: { size: 10 }, padding: 8, cornerRadius: 6,
                    callbacks: { label: (ctx: ChartCallbackContext) => (ctx.dataset.label ?? '') + ': ' + fmtRate(ctx.parsed.y ?? 0) }
                }
            },
            scales: {
                x: { display: false },
                y: {
                    display: true, beginAtZero: true, grid: { color: 'rgba(148,163,184,0.08)' }, border: { display: false },
                    ticks: { font: { size: 9 }, color: '#94a3b8', maxTicksLimit: 4, padding: 4, callback: (v: string | number) => fmtRate(Number(v)) }
                }
            },
            elements: { point: { radius: 0, hoverRadius: 3 }, line: { tension: 0.4, borderWidth: 1.5 } }
        }
    }

    makeDataset(data: number[], color: string, label: string) {
        return { label, data: [...data], borderColor: color, backgroundColor: hexToRgba(color, 0.1), fill: true }
    }

    initNetChart(name: string) {
        const canvas = this.netContainerRef?.querySelector(`[data-iface="${name}"]`) as HTMLCanvasElement | null
        if (!canvas) return
        this.netCharts[name]?.destroy()
        const h = this.netHistory[name] || { ts: [], labels: [], recv: [], sent: [] }
        this.netHistory[name] = h
        const chart = new Chart(canvas, {
            type: 'line' as const,
            data: { labels: [...h.labels], datasets: [this.makeDataset(h.recv, '#10b981', '下行'), this.makeDataset(h.sent, '#3b82f6', '上行')] },
            options: this.netChartOptions()
        })
        this.netCharts[name] = markRaw(chart)
    }

    updateNetChart(name: string) {
        const chart = this.netCharts[name]
        const h = this.netHistory[name]
        if (!chart || !h) return
        chart.data.labels = [...h.labels]
        chart.data.datasets[0].data = [...h.recv]
        chart.data.datasets[1].data = [...h.sent]
        chart.update('none')
    }

    flushCharts() {
        this.$nextTick(() => {
            Object.keys(this.netHistory).forEach(name => {
                if (this.netCharts[name]) {
                    this.updateNetChart(name)
                } else {
                    this.initNetChart(name)
                }
            })
        })
    }

    clearData() {
        this.currentIfaces = []
        this.netHistory = {}
        this.lastNetIO = {}
        Object.values(this.netCharts).forEach(c => c.destroy())
        this.netCharts = {}
    }

    pushData(payload: SystemStat, ts: number) {
        const ifaces = payload.system?.netInterface || []
        const physicalIfaces = this.physicalInterfaces(ifaces)
        this.currentIfaces = physicalIfaces
        if (!ifaces.length) return

        const nowTime = ts * 1000

        physicalIfaces.forEach(ni => {
            const name = ni.name
            const last = this.lastNetIO[name]

            let recvRate = 0
            let sentRate = 0

            if (last && last.time > 0) {
                const dt = (nowTime - last.time) / 1000
                if (dt > 0) {
                    recvRate = Math.max(0, (ni.bytesRecv - last.recv) / dt)
                    sentRate = Math.max(0, (ni.bytesSent - last.sent) / dt)
                }
            }

            this.lastNetIO[name] = { recv: ni.bytesRecv, sent: ni.bytesSent, time: nowTime }

            if (!this.netHistory[name]) {
                this.netHistory[name] = { ts: [], labels: [], recv: [], sent: [] }
            }
            const h = this.netHistory[name]

            appendMonitorPoint(
                h,
                ts,
                this.rangeSeconds,
                () => {
                    h.recv.push(recvRate)
                    h.sent.push(sentRate)
                },
                count => {
                    h.recv.splice(0, count)
                    h.sent.splice(0, count)
                }
            )
        })
    }

    unmounted() {
        Object.values(this.netCharts).forEach(c => c.destroy())
        this.netCharts = {}
    }
}

export default toNative(SystemNetwork)
</script>

<template>
  <div v-if="currentIfaces.length" class="rounded-xl border border-slate-200 bg-white overflow-hidden">
    <div class="card-header">
      <div class="card-icon bg-cyan-500">
        <i class="fas fa-network-wired text-white text-xs"></i>
      </div>
      <span class="text-sm font-semibold text-slate-700">网络接口</span>
    </div>
    <div ref="netContainerRef" class="divide-y divide-slate-100">
      <div v-for="ni in currentIfaces" :key="ni.name" class="px-4 py-3">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between mb-2 gap-1">
          <p class="text-xs font-semibold text-slate-700">{{ ni.name }}</p>
          <div class="flex items-center gap-4 text-xs">
            <span class="flex items-center gap-1">
              <i class="fas fa-arrow-down text-emerald-500"></i>
              <span class="font-mono text-slate-600">{{ fmtRate(currentRate(ni.name, 'recv')) }}</span>
            </span>
            <span class="flex items-center gap-1">
              <i class="fas fa-arrow-up text-blue-500"></i>
              <span class="font-mono text-slate-600">{{ fmtRate(currentRate(ni.name, 'sent')) }}</span>
            </span>
          </div>
        </div>
        <div class="relative h-28 bg-slate-50 rounded-lg overflow-hidden">
          <canvas :data-iface="ni.name" class="w-full h-full"></canvas>
          <div v-if="!netHistory[ni.name]?.labels?.length" class="absolute inset-0 flex items-center justify-center">
            <span class="text-xs text-slate-300">等待数据...</span>
          </div>
        </div>
        <div class="flex flex-col sm:flex-row gap-2 sm:gap-4 mt-1.5 text-xs text-slate-400">
          <span>累计收: {{ fmtBytes(ni.bytesRecv) }}</span>
          <span>累计发: {{ fmtBytes(ni.bytesSent) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>
