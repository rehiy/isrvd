<script lang="ts">
import { markRaw } from 'vue'
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'
import type { ChartOptions } from 'chart.js'

import type { SystemStat, SystemDiskIO } from '@/service/types'

import Chart from '@/helper/chart'
import { hexToRgba } from '@/helper/utils'

interface DiskIOSeriesHistory {
    labels: string[]
    read: number[]
    write: number[]
}

interface ChartCallbackContext {
    parsed: { y: number | null }
    dataset: { label?: string }
}

const MAX_HISTORY = 60

@Component
class SystemDisk extends Vue {
    @Ref readonly diskIOContainerRef!: HTMLDivElement

    private diskIOCharts: Record<string, Chart<'line'>> = {}
    diskIOHistory: Record<string, DiskIOSeriesHistory> = {}
    private lastDiskIO: Record<string, { read: number; write: number; time: number }> = {}
    current: Pick<SystemStat['system'], 'DiskTotal' | 'DiskUsed' | 'DiskPartition'> | null = null
    private currentDiskIO: SystemDiskIO[] = []

    fmtSize(bytes: number, rates = false) {
        if (!bytes || bytes < 0) return rates ? '0 B/s' : '0 B'
        const units = rates ? ['B/s', 'KB/s', 'MB/s', 'GB/s'] : ['B', 'KB', 'MB', 'GB', 'TB']
        let i = 0, v = bytes
        while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
        return `${v.toFixed(1)} ${units[i]}`
    }

    fmtBytes(b: number) { return this.fmtSize(b, false) }
    fmtRate(b: number) { return this.fmtSize(b, true) }

    memPercent(used: number, total: number): number {
        if (!total) return 0
        return parseFloat(((used / total) * 100).toFixed(1))
    }

    semanticColor(pct: number, prefix = 'bg') {
        if (pct >= 90) return `${prefix}-red-500`
        if (pct >= 70) return `${prefix}-amber-500`
        return `${prefix}-emerald-500`
    }

    barColor(pct: number) { return this.semanticColor(pct, 'bg') }

    devShortName(device: string): string { return device.split('/').pop() || device }

    diskIOByDevice(device: string): SystemDiskIO | null {
        if (!this.currentDiskIO?.length) return null
        const devName = device.split('/').pop()
        return this.currentDiskIO.find(d => d.Name === devName)
            || this.currentDiskIO.find(d => devName!.startsWith(d.Name)) || null
    }

    currentDiskRate(name: string, dir: string): number {
        const h = this.diskIOHistory[name]
        if (!h) return 0
        const arr = h[dir as keyof DiskIOSeriesHistory] as number[]
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

    getDiskCanvas(name: string) {
        return this.diskIOContainerRef?.querySelector(`[data-disk="${name}"]`) ?? null
    }

    initDiskChart(name: string) {
        const canvas = this.getDiskCanvas(name)
        if (!canvas) return
        this.diskIOCharts[name]?.destroy()
        const h = this.diskIOHistory[name] || { labels: [], read: [], write: [] }
        this.diskIOHistory[name] = h
        const chart = new Chart(canvas as HTMLCanvasElement, {
            type: 'line' as const,
            data: { labels: [...h.labels], datasets: [this.makeDataset(h.read, '#f59e0b', '读取'), this.makeDataset(h.write, '#8b5cf6', '写入')] },
            options: this.netChartOptions()
        })
        this.diskIOCharts[name] = markRaw(chart)
    }

    updateDiskChart(name: string) {
        const chart = this.diskIOCharts[name]
        const h = this.diskIOHistory[name]
        if (!chart || !h) return
        chart.data.labels = [...h.labels]
        chart.data.datasets[0].data = [...h.read]
        chart.data.datasets[1].data = [...h.write]
        chart.update('none')
    }

    pushData(payload: SystemStat) {
        const s = payload.system
        this.current = { DiskTotal: s.DiskTotal, DiskUsed: s.DiskUsed, DiskPartition: s.DiskPartition }
        this.currentDiskIO = payload.diskIO || []

        if (!payload.diskIO?.length || !s.DiskPartition) return

        const now = new Date()
        const label = `${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}:${now.getSeconds().toString().padStart(2, '0')}`
        const nowTime = Date.now()

        payload.diskIO.forEach(dio => {
            const name = dio.Name
            const last = this.lastDiskIO[name]

            let readRate = 0
            let writeRate = 0

            if (last && last.time > 0) {
                const dt = (nowTime - last.time) / 1000
                if (dt > 0) {
                    readRate = Math.max(0, (dio.ReadBytes - last.read) / dt)
                    writeRate = Math.max(0, (dio.WriteBytes - last.write) / dt)
                }
            }

            this.lastDiskIO[name] = { read: dio.ReadBytes, write: dio.WriteBytes, time: nowTime }

            if (!this.diskIOHistory[name]) {
                this.diskIOHistory[name] = { labels: [], read: [], write: [] }
            }
            const h = this.diskIOHistory[name]

            h.labels.push(label)
            h.read.push(readRate)
            h.write.push(writeRate)

            if (h.labels.length > MAX_HISTORY) {
                h.labels.shift()
                h.read.shift()
                h.write.shift()
            }

            if (!this.diskIOCharts[name]) {
                this.$nextTick(() => this.initDiskChart(name))
            } else {
                this.updateDiskChart(name)
            }
        })
    }

    unmounted() {
        Object.values(this.diskIOCharts).forEach(c => c.destroy())
        this.diskIOCharts = {}
    }
}

export default toNative(SystemDisk)
</script>

<template>
  <div v-if="current?.DiskPartition?.length" class="rounded-xl border border-slate-200 bg-white overflow-hidden">
    <div class="px-4 py-3 border-b border-slate-100 flex items-center gap-2">
      <div class="w-6 h-6 rounded-md bg-amber-500 flex items-center justify-center">
        <i class="fas fa-hard-drive text-white text-xs"></i>
      </div>
      <span class="text-sm font-semibold text-slate-700">硬盘 I/O</span>
      <span class="ml-auto text-xs text-slate-400">
        总计 {{ fmtBytes(current.DiskTotal) }}，已用 {{ fmtBytes(current.DiskUsed) }}
      </span>
    </div>
    <div ref="diskIOContainerRef" class="divide-y divide-slate-50">
      <div v-for="dp in current.DiskPartition" :key="dp.Mountpoint" class="px-4 py-3">
        <div class="flex flex-col gap-2">
          <div class="flex items-center justify-between gap-2">
            <div class="flex items-center gap-2 min-w-0">
              <p class="text-xs font-semibold text-slate-700 truncate">{{ dp.Mountpoint }}</p>
              <p class="text-xs text-slate-400 shrink-0">{{ dp.Device }} · {{ dp.Fstype }}</p>
            </div>
            <p class="text-xs text-slate-500 font-mono shrink-0">{{ fmtBytes(dp.Used) }} / {{ fmtBytes(dp.Total) }} ({{ memPercent(dp.Used, dp.Total) }}%)</p>
          </div>
          <div class="h-1 relative bg-slate-100 rounded overflow-hidden">
            <div :class="['absolute inset-y-0 left-0 rounded', barColor(memPercent(dp.Used, dp.Total))]" :style="{ width: memPercent(dp.Used, dp.Total) + '%' }"></div>
          </div>
          <template v-if="diskIOByDevice(dp.Device)">
            <div class="flex items-center justify-between gap-1">
              <span class="text-xs text-slate-400">IO 速率</span>
              <div class="flex items-center gap-4 text-xs">
                <span class="flex items-center gap-1">
                  <i class="fas fa-arrow-down text-amber-500"></i>
                  <span class="font-mono text-slate-600 w-20 text-right">{{ fmtRate(currentDiskRate(devShortName(dp.Device), 'read')) }}</span>
                </span>
                <span class="flex items-center gap-1">
                  <i class="fas fa-arrow-up text-violet-500"></i>
                  <span class="font-mono text-slate-600 w-20 text-right">{{ fmtRate(currentDiskRate(devShortName(dp.Device), 'write')) }}</span>
                </span>
              </div>
            </div>
            <div class="relative h-16 bg-slate-50 rounded-lg overflow-hidden">
              <canvas :data-disk="devShortName(dp.Device)" class="w-full h-full"></canvas>
              <div v-if="!diskIOHistory[devShortName(dp.Device)]?.read?.length" class="absolute inset-0 flex items-center justify-center">
                <span class="text-xs text-slate-300">等待数据...</span>
              </div>
            </div>
            <div class="flex gap-4 text-xs text-slate-400">
              <span>累计读: {{ fmtBytes(diskIOByDevice(dp.Device)?.ReadBytes ?? 0) }}</span>
              <span>累计写: {{ fmtBytes(diskIOByDevice(dp.Device)?.WriteBytes ?? 0) }}</span>
            </div>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>
