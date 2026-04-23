<script lang="ts">
import { markRaw } from 'vue'
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'
import type { ChartOptions } from 'chart.js'

import type { SystemStat, SystemNetInterface } from '@/service/types'

import Chart from '@/helper/chart'
import { hexToRgba } from '@/helper/utils'

interface TimeSeriesHistory {
    labels: string[]
    recv: number[]
    sent: number[]
}

interface ChartCallbackContext {
    parsed: { y: number | null }
    dataset: { label?: string }
}

const MAX_HISTORY = 60

@Component
class SystemNetwork extends Vue {
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
        return list.filter(ni => !virtualPrefixes.some(p => ni.Name.startsWith(p)))
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
        const h = this.netHistory[name] || { labels: [], recv: [], sent: [] }
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

    pushData(payload: SystemStat) {
        const ifaces = payload.system?.NetInterface || []
        this.currentIfaces = this.physicalInterfaces(ifaces)
        if (!ifaces.length) return

        const now = new Date()
        const label = `${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}:${now.getSeconds().toString().padStart(2, '0')}`
        const nowTime = Date.now()

        this.physicalInterfaces(ifaces).forEach(ni => {
            const name = ni.Name
            const last = this.lastNetIO[name]

            let recvRate = 0
            let sentRate = 0

            if (last && last.time > 0) {
                const dt = (nowTime - last.time) / 1000
                if (dt > 0) {
                    recvRate = Math.max(0, (ni.BytesRecv - last.recv) / dt)
                    sentRate = Math.max(0, (ni.BytesSent - last.sent) / dt)
                }
            }

            this.lastNetIO[name] = { recv: ni.BytesRecv, sent: ni.BytesSent, time: nowTime }

            if (!this.netHistory[name]) {
                this.netHistory[name] = { labels: [], recv: [], sent: [] }
            }
            const h = this.netHistory[name]

            h.labels.push(label)
            h.recv.push(recvRate)
            h.sent.push(sentRate)

            if (h.labels.length > MAX_HISTORY) {
                h.labels.shift()
                h.recv.shift()
                h.sent.shift()
            }

            if (!this.netCharts[name]) {
                this.$nextTick(() => this.initNetChart(name))
            } else {
                this.updateNetChart(name)
            }
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
    <div class="px-4 py-3 border-b border-slate-100 flex items-center gap-2">
      <div class="w-6 h-6 rounded-md bg-cyan-500 flex items-center justify-center">
        <i class="fas fa-network-wired text-white text-xs"></i>
      </div>
      <span class="text-sm font-semibold text-slate-700">网络接口</span>
    </div>
    <div ref="netContainerRef" class="divide-y divide-slate-50">
      <div v-for="ni in currentIfaces" :key="ni.Name" class="px-4 py-3">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between mb-2 gap-1">
          <p class="text-xs font-semibold text-slate-700">{{ ni.Name }}</p>
          <div class="flex items-center gap-4 text-xs">
            <span class="flex items-center gap-1">
              <i class="fas fa-arrow-down text-emerald-500"></i>
              <span class="font-mono text-slate-600 w-20 text-right">{{ fmtRate(currentRate(ni.Name, 'recv')) }}</span>
            </span>
            <span class="flex items-center gap-1">
              <i class="fas fa-arrow-up text-blue-500"></i>
              <span class="font-mono text-slate-600 w-20 text-right">{{ fmtRate(currentRate(ni.Name, 'sent')) }}</span>
            </span>
          </div>
        </div>
        <div class="relative h-20 bg-slate-50 rounded-lg overflow-hidden">
          <canvas :data-iface="ni.Name" class="w-full h-full"></canvas>
          <div v-if="!netHistory[ni.Name]?.labels?.length" class="absolute inset-0 flex items-center justify-center">
            <span class="text-xs text-slate-300">等待数据...</span>
          </div>
        </div>
        <div class="flex flex-col sm:flex-row gap-2 sm:gap-4 mt-1.5 text-xs text-slate-400">
          <span>累计收: {{ fmtBytes(ni.BytesRecv) }}</span>
          <span>累计发: {{ fmtBytes(ni.BytesSent) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>
