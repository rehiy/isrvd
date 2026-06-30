<script lang="ts">
import type { ChartOptions } from 'chart.js'
import { markRaw } from 'vue'
import { Component, Prop, Ref, Vue, toNative } from 'vue-facing-decorator'

import type { SystemStat, SystemDiskIO, SystemDiskPartition } from '@/service/types'

import Chart from '@/helper/chart'
import { hexToRgba } from '@/helper/format'
import { appendMonitorPoint } from '@/helper/monitor'

interface DiskIOSeriesHistory {
    ts: number[]
    labels: string[]
    read: number[]
    write: number[]
    pct: number[]
}

interface ChartCallbackContext {
    parsed: { y: number | null }
    dataset: { label?: string; yAxisID?: string }
}

const NATURAL_COLLATOR = new Intl.Collator(undefined, { numeric: true, sensitivity: 'base' })

@Component
class SystemDisk extends Vue {
    @Prop({ type: Number, default: 300 }) readonly rangeSeconds!: number

    @Ref readonly diskIOContainerRef!: HTMLDivElement

    private diskIOCharts: Record<string, Chart<'line'>> = {}
    diskIOHistory: Record<string, DiskIOSeriesHistory> = {}
    private lastDiskIO: Record<string, { read: number; write: number; time: number }> = {}
    current: Pick<SystemStat['system'], 'diskTotal' | 'diskUsed' | 'diskPartition'> | null = null
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

    usageLineColor(pct: number): string {
        if (pct >= 90) return '#ef4444'
        if (pct >= 70) return '#f59e0b'
        return '#10b981'
    }

    devShortName(device: string): string { return device.split('/').pop() || device }

    sortedDiskPartitions(list: SystemDiskPartition[] = []) {
        return [...list].sort((a, b) => {
            if (a.mountpoint === '/') return -1
            if (b.mountpoint === '/') return 1
            return NATURAL_COLLATOR.compare(a.mountpoint, b.mountpoint)
                || NATURAL_COLLATOR.compare(a.device, b.device)
        })
    }

    sortedDiskIO(list: SystemDiskIO[] = []) {
        return [...list].sort((a, b) => NATURAL_COLLATOR.compare(a.name, b.name))
    }

    diskIOByDevice(device: string): SystemDiskIO | null {
        if (!this.currentDiskIO?.length) return null
        const devName = device.split('/').pop()
        return this.currentDiskIO.find(d => d.name === devName)
            || this.currentDiskIO.find(d => devName?.startsWith(d.name)) || null
    }

    diskIOHistoryKey(device: string): string {
        return this.diskIOByDevice(device)?.name || this.devShortName(device)
    }

    currentDiskRate(name: string, dir: string): number {
        const h = this.diskIOHistory[name]
        if (!h) return 0
        const arr = h[dir as keyof DiskIOSeriesHistory] as number[]
        return arr?.length ? arr[arr.length - 1] : 0
    }

    currentUsagePct(dp: SystemDiskPartition): number {
        const h = this.diskIOHistory[this.diskIOHistoryKey(dp.device)]
        if (!h?.pct?.length) return this.memPercent(dp.used, dp.total)
        return h.pct[h.pct.length - 1]
    }

    diskChartOptions(pctColor: string): ChartOptions<'line'> {
        const fmtRate = (v: number) => this.fmtRate(v)
        return {
            responsive: true, maintainAspectRatio: false, animation: false,
            interaction: { intersect: false, mode: 'index' as const },
            plugins: {
                legend: { display: true, position: 'bottom' as const, labels: { boxWidth: 8, padding: 8, font: { size: 10 }, color: '#64748b' } },
                tooltip: {
                    backgroundColor: 'rgba(15,23,42,0.9)', titleFont: { size: 10 }, bodyFont: { size: 10 }, padding: 8, cornerRadius: 6,
                    callbacks: {
                        label: (ctx: ChartCallbackContext) => {
                            if (ctx.dataset.yAxisID === 'y1') {
                                return '使用率: ' + (ctx.parsed.y ?? 0).toFixed(1) + '%'
                            }
                            return (ctx.dataset.label ?? '') + ': ' + fmtRate(ctx.parsed.y ?? 0)
                        }
                    }
                }
            },
            scales: {
                x: { display: false },
                y: {
                    display: true, beginAtZero: true, position: 'left' as const,
                    grid: { color: 'rgba(148,163,184,0.08)' }, border: { display: false },
                    ticks: { font: { size: 9 }, color: '#94a3b8', maxTicksLimit: 4, padding: 4, callback: (v: string | number) => fmtRate(Number(v)) }
                },
                y1: {
                    display: true, min: 0, max: 100, position: 'right' as const,
                    grid: { drawOnChartArea: false },
                    border: { display: false },
                    ticks: { font: { size: 9 }, color: pctColor, maxTicksLimit: 4, padding: 4, callback: (v: string | number) => Number(v) + '%' }
                }
            },
            elements: { point: { radius: 0, hoverRadius: 3 }, line: { tension: 0.4, borderWidth: 1.5 } }
        }
    }

    makeDataset(data: number[], color: string, label: string, yAxisID = 'y') {
        return { label, data: [...data], borderColor: color, backgroundColor: hexToRgba(color, 0.1), fill: yAxisID === 'y', yAxisID }
    }

    getDiskCanvas(name: string) {
        return this.diskIOContainerRef?.querySelector(`[data-disk="${name}"]`) ?? null
    }

    initDiskChart(name: string) {
        const canvas = this.getDiskCanvas(name)
        if (!canvas) return
        this.diskIOCharts[name]?.destroy()
        const h = this.diskIOHistory[name] || { ts: [], labels: [], read: [], write: [], pct: [] }
        this.diskIOHistory[name] = h
        const pct = h.pct[h.pct.length - 1] ?? 0
        const pctColor = this.usageLineColor(pct)
        const chart = new Chart(canvas as HTMLCanvasElement, {
            type: 'line' as const,
            data: {
                labels: [...h.labels],
                datasets: [
                    this.makeDataset(h.read, '#f59e0b', '读取', 'y'),
                    this.makeDataset(h.write, '#8b5cf6', '写入', 'y'),
                    this.makeDataset(h.pct, pctColor, '使用率', 'y1'),
                ]
            },
            options: this.diskChartOptions(pctColor)
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
        chart.data.datasets[2].data = [...h.pct]
        chart.update('none')
    }

    flushCharts() {
        this.$nextTick(() => {
            Object.keys(this.diskIOHistory).forEach(name => {
                if (this.diskIOCharts[name]) {
                    this.updateDiskChart(name)
                } else {
                    this.initDiskChart(name)
                }
            })
        })
    }

    clearData() {
        this.current = null
        this.currentDiskIO = []
        this.diskIOHistory = {}
        this.lastDiskIO = {}
        Object.values(this.diskIOCharts).forEach(c => c.destroy())
        this.diskIOCharts = {}
    }

    pushData(payload: SystemStat, ts: number) {
        const s = payload.system
        const partitions = this.sortedDiskPartitions(s.diskPartition)
        this.current = { diskTotal: s.diskTotal, diskUsed: s.diskUsed, diskPartition: partitions }
        const diskIO = this.sortedDiskIO(payload.diskIO)
        this.currentDiskIO = diskIO

        if (!diskIO.length || !s.diskPartition) return

        const nowTime = ts * 1000

        diskIO.forEach(dio => {
            const name = dio.name
            const last = this.lastDiskIO[name]

            let readRate = 0
            let writeRate = 0

            if (last && last.time > 0) {
                const dt = (nowTime - last.time) / 1000
                if (dt > 0) {
                    readRate = Math.max(0, (dio.readBytes - last.read) / dt)
                    writeRate = Math.max(0, (dio.writeBytes - last.write) / dt)
                }
            }

            this.lastDiskIO[name] = { read: dio.readBytes, write: dio.writeBytes, time: nowTime }

            if (!this.diskIOHistory[name]) {
                this.diskIOHistory[name] = { ts: [], labels: [], read: [], write: [], pct: [] }
            }
            const h = this.diskIOHistory[name]

            // 找到该设备对应的分区使用率
            const dp = partitions.find(p => {
                const devName = p.device.split('/').pop()
                return devName === name || devName?.startsWith(name)
            })
            const pct = dp ? this.memPercent(dp.used, dp.total) : (h.pct[h.pct.length - 1] ?? 0)

            appendMonitorPoint(
                h,
                ts,
                this.rangeSeconds,
                () => {
                    h.read.push(readRate)
                    h.write.push(writeRate)
                    h.pct.push(pct)
                },
                count => {
                    h.read.splice(0, count)
                    h.write.splice(0, count)
                    h.pct.splice(0, count)
                }
            )
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
  <div v-if="current?.diskPartition?.length" class="panel-frame">
    <div class="card-header">
      <div class="card-icon bg-amber-500">
        <i class="fas fa-hard-drive text-white text-xs"></i>
      </div>
      <span class="text-sm font-semibold text-slate-700">硬盘 I/O</span>
      <span class="ml-auto text-xs text-slate-400">
        总计 {{ fmtBytes(current.diskTotal) }}，已用 {{ fmtBytes(current.diskUsed) }}
      </span>
    </div>
    <div ref="diskIOContainerRef" class="divide-y divide-slate-100">
      <div v-for="dp in current.diskPartition" :key="dp.mountpoint" class="px-4 py-3">
        <div class="flex flex-col gap-2">
          <div class="flex items-center justify-between gap-x-3 gap-y-1 flex-wrap">
            <div class="inline-info flex-1">
              <p class="text-xs font-semibold text-slate-700 truncate">{{ dp.mountpoint }}</p>
              <p class="text-xs text-slate-400 shrink-0">{{ dp.device }} · {{ dp.fstype }}</p>
            </div>
            <div class="flex items-center justify-end gap-x-3 gap-y-1 flex-wrap flex-1 min-w-0">
              <span class="text-xs text-slate-600 font-mono shrink-0 whitespace-nowrap">{{ fmtBytes(dp.used) }} / {{ fmtBytes(dp.total) }} ({{ currentUsagePct(dp) }}%)</span>
              <template v-if="diskIOByDevice(dp.device)">
                <span class="flex items-center gap-1 text-xs shrink-0 whitespace-nowrap">
                  <i class="fas fa-arrow-down text-amber-500"></i>
                  <span class="font-mono text-slate-600">{{ fmtRate(currentDiskRate(diskIOHistoryKey(dp.device), 'read')) }}</span>
                </span>
                <span class="flex items-center gap-1 text-xs shrink-0 whitespace-nowrap">
                  <i class="fas fa-arrow-up text-violet-500"></i>
                  <span class="font-mono text-slate-600">{{ fmtRate(currentDiskRate(diskIOHistoryKey(dp.device), 'write')) }}</span>
                </span>
              </template>
            </div>
          </div>
          <template v-if="diskIOByDevice(dp.device)">
            <div class="monitor-chart-box">
              <canvas :data-disk="diskIOHistoryKey(dp.device)" class="w-full h-full"></canvas>
              <div v-if="!diskIOHistory[diskIOHistoryKey(dp.device)]?.read?.length" class="absolute inset-0 flex items-center justify-center">
                <span class="text-xs text-slate-300">等待数据...</span>
              </div>
            </div>
            <div class="flex gap-4 text-xs text-slate-400">
              <span>累计读: {{ fmtBytes(diskIOByDevice(dp.device)?.readBytes ?? 0) }}</span>
              <span>累计写: {{ fmtBytes(diskIOByDevice(dp.device)?.writeBytes ?? 0) }}</span>
            </div>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>
