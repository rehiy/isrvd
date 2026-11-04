<script lang="ts">
import type { ChartOptions } from 'chart.js'
import { markRaw } from 'vue'
import { Component, Prop, Ref, Vue, toNative } from 'vue-facing-decorator'

import type { SystemDiskIO, SystemDiskPartition, SystemStat } from '@/service/types'

import Chart, { findMonitorCanvas, makeLineDataset, makeMonitorChartOptions, makeMonitorScale, syncMonitorCharts } from '@/helper/chart'
import { appendMonitorPoint, formatMonitorBytes, usagePercent } from '@/helper/monitor'

interface DiskIOSeriesHistory {
    ts: number[]
    labels: string[]
    read: number[]
    write: number[]
    pct: number[]
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
    formatMonitorBytes = formatMonitorBytes

    memPercent = usagePercent

    usageLineColor(pct: number): string {
        if (pct >= 90) {
            return '#ef4444'
        }
        if (pct >= 70) {
            return '#f59e0b'
        }
        return '#10b981'
    }

    sortedDiskPartitions(list: SystemDiskPartition[] = []) {
        return [...list].sort((a, b) => {
            if (a.mountpoint === '/') {
                return -1
            }
            if (b.mountpoint === '/') {
                return 1
            }
            return NATURAL_COLLATOR.compare(a.mountpoint, b.mountpoint)
                || NATURAL_COLLATOR.compare(a.device, b.device)
        })
    }

    sortedDiskIO(list: SystemDiskIO[] = []) {
        return [...list].sort((a, b) => NATURAL_COLLATOR.compare(a.name, b.name))
    }

    diskIOByDevice(device: string): SystemDiskIO | null {
        if (!this.currentDiskIO?.length) {
            return null
        }
        const devName = device.split('/').pop()
        return this.currentDiskIO.find(d => d.name === devName)
            || this.currentDiskIO.find(d => devName?.startsWith(d.name)) || null
    }

    diskIOHistoryKey(dp: SystemDiskPartition): string {
        return JSON.stringify([dp.device, dp.mountpoint])
    }

    currentDiskRate(name: string, dir: string): number {
        const h = this.diskIOHistory[name]
        if (!h) {
            return 0
        }
        const arr = h[dir as keyof DiskIOSeriesHistory] as number[]
        return arr?.length ? arr[arr.length - 1] : 0
    }

    currentUsagePct(dp: SystemDiskPartition): number {
        return this.memPercent(dp.used, dp.total)
    }

    diskChartOptions(pctColor: string): ChartOptions<'line'> {
        return makeMonitorChartOptions(
            {
                y: makeMonitorScale('rate', { position: 'left' }),
                y1: makeMonitorScale('percent', {
                    min: 0, position: 'right', grid: { drawOnChartArea: false },
                    ticks: { color: pctColor }
                })
            },
            ctx => {
                if (ctx.dataset.yAxisID === 'y1') {
                    return this.$t('使用率: ') + (ctx.parsed.y ?? 0).toFixed(1) + '%'
                }
                return (ctx.dataset.label ?? '') + ': ' + formatMonitorBytes(ctx.parsed.y ?? 0, true)
            }
        )
    }

    initDiskChart(name: string) {
        const canvas = findMonitorCanvas(this.diskIOContainerRef, 'data-disk', name)
        if (!canvas) {
            return
        }
        this.diskIOCharts[name]?.destroy()
        const h = this.diskIOHistory[name] || { ts: [], labels: [], read: [], write: [], pct: [] }
        this.diskIOHistory[name] = h
        const pct = h.pct[h.pct.length - 1] ?? 0
        const pctColor = this.usageLineColor(pct)
        const chart = new Chart(canvas, {
            type: 'line' as const,
            data: {
                labels: [...h.labels],
                datasets: [
                    makeLineDataset(h.read, '#f59e0b', this.$t('读取'), { yAxisID: 'y' }),
                    makeLineDataset(h.write, '#8b5cf6', this.$t('写入'), { yAxisID: 'y' }),
                    makeLineDataset(h.pct, pctColor, this.$t('使用率'), { yAxisID: 'y1', fill: false }),
                ]
            },
            options: this.diskChartOptions(pctColor)
        })
        this.diskIOCharts[name] = markRaw(chart)
    }

    updateDiskChart(name: string) {
        const chart = this.diskIOCharts[name]
        const h = this.diskIOHistory[name]
        if (!chart || !h) {
            return
        }
        chart.data.labels = [...h.labels]
        chart.data.datasets[0].data = [...h.read]
        chart.data.datasets[1].data = [...h.write]
        chart.data.datasets[2].data = [...h.pct]
        const pctColor = this.usageLineColor(h.pct[h.pct.length - 1] ?? 0)
        chart.data.datasets[2].borderColor = pctColor
        if (chart.options.scales?.y1?.ticks) {
            chart.options.scales.y1.ticks.color = pctColor
        }
        chart.update('none')
    }

    flushCharts() {
        this.$nextTick(() => {
            const keys = (this.current?.diskPartition ?? [])
                .filter(dp => this.diskIOByDevice(dp.device))
                .map(dp => this.diskIOHistoryKey(dp))
            syncMonitorCharts(
                this.diskIOCharts, this.diskIOContainerRef, 'data-disk', keys,
                name => this.initDiskChart(name), name => this.updateDiskChart(name)
            )
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

        const activeDevices = new Set(diskIO.map(dio => dio.name))
        for (const name of Object.keys(this.lastDiskIO)) {
            if (!activeDevices.has(name)) {
                delete this.lastDiskIO[name]
            }
        }
        const activePartitions = new Set(partitions.filter(dp => this.diskIOByDevice(dp.device)).map(dp => this.diskIOHistoryKey(dp)))
        for (const key of Object.keys(this.diskIOHistory)) {
            if (!activePartitions.has(key)) {
                delete this.diskIOHistory[key]
            }
        }

        const nowTime = ts * 1000
        const rates = new Map<string, { read: number; write: number }>()
        for (const dio of diskIO) {
            const last = this.lastDiskIO[dio.name]
            let readRate = 0
            let writeRate = 0
            if (last && last.time > 0) {
                const dt = (nowTime - last.time) / 1000
                if (dt > 0) {
                    readRate = Math.max(0, (dio.readBytes - last.read) / dt)
                    writeRate = Math.max(0, (dio.writeBytes - last.write) / dt)
                }
            }
            rates.set(dio.name, { read: readRate, write: writeRate })
            this.lastDiskIO[dio.name] = { read: dio.readBytes, write: dio.writeBytes, time: nowTime }
        }

        for (const dp of partitions) {
            const dio = this.diskIOByDevice(dp.device)
            const rate = dio ? rates.get(dio.name) : null
            if (!rate) {
                continue
            }
            const key = this.diskIOHistoryKey(dp)
            const h = this.diskIOHistory[key] ??= { ts: [], labels: [], read: [], write: [], pct: [] }
            appendMonitorPoint(
                h,
                ts,
                this.rangeSeconds,
                () => {
                    h.read.push(rate.read)
                    h.write.push(rate.write)
                    h.pct.push(this.memPercent(dp.used, dp.total))
                },
                count => {
                    h.read.splice(0, count)
                    h.write.splice(0, count)
                    h.pct.splice(0, count)
                }
            )
        }
    }

    unmounted() {
        this.clearData()
    }
}

export default toNative(SystemDisk)
</script>

<template>
  <div v-if="current?.diskPartition?.length" class="space-y-3">
    <div ref="diskIOContainerRef" class="grid grid-cols-1 gap-4">
      <div v-for="dp in current.diskPartition" :key="diskIOHistoryKey(dp)" class="metric-card min-w-0">
        <div class="absolute inset-x-0 top-0 h-0.5 bg-gradient-to-r from-amber-400 to-transparent"></div>
        <div class="monitor-card-header">
          <div class="monitor-card-heading">
            <div class="monitor-card-icon">
              <i class="fas fa-hard-drive text-sm"></i>
            </div>
            <div class="min-w-0 flex-1">
              <p class="monitor-card-title" :title="`${$t('硬盘')} - ${dp.mountpoint}`">{{ $t('硬盘') }} - {{ dp.mountpoint }}</p>
              <p class="break-words text-xs text-slate-400" :title="`${dp.device} · ${dp.fstype} · ${formatMonitorBytes(dp.used)} / ${formatMonitorBytes(dp.total)}`">
                {{ dp.device }} · {{ dp.fstype }} · <span>{{ formatMonitorBytes(dp.used) }} / {{ formatMonitorBytes(dp.total) }}</span>
              </p>
            </div>
          </div>
          <div class="monitor-card-metrics">
            <span v-if="diskIOByDevice(dp.device)" class="monitor-headline">{{ $t('吞吐') }} <span class="monitor-card-value text-slate-700">{{ formatMonitorBytes(currentDiskRate(diskIOHistoryKey(dp), 'read') + currentDiskRate(diskIOHistoryKey(dp), 'write'), true) }}</span></span>
            <span v-else class="monitor-headline">{{ $t('使用率') }} <span class="monitor-card-value text-slate-700">{{ currentUsagePct(dp) }}%</span></span>
          </div>
        </div>
        <div v-if="diskIOByDevice(dp.device)" class="monitor-chart-box">
          <canvas :data-disk="diskIOHistoryKey(dp)" class="w-full h-full"></canvas>
          <div v-if="!diskIOHistory[diskIOHistoryKey(dp)]?.read?.length" class="absolute inset-0 flex items-center justify-center">
            <span class="text-xs text-slate-300">{{ $t('等待数据...') }}</span>
          </div>
        </div>
        <div v-if="diskIOByDevice(dp.device)" class="monitor-card-legend">
          <span class="monitor-series">
            <span class="monitor-series-dot" :style="{ backgroundColor: usageLineColor(currentUsagePct(dp)) }"></span>{{ $t('使用率') }} <span class="tabular-nums text-slate-600">{{ currentUsagePct(dp) }}%</span>
          </span>
          <span class="monitor-series">
            <span class="monitor-series-dot bg-amber-500"></span>{{ $t('读取') }} <span class="tabular-nums text-slate-600">{{ formatMonitorBytes(currentDiskRate(diskIOHistoryKey(dp), 'read'), true) }}</span>
            <span class="whitespace-nowrap">· {{ $t('累计读') }} <span class="tabular-nums">{{ formatMonitorBytes(diskIOByDevice(dp.device)?.readBytes ?? 0) }}</span></span>
          </span>
          <span class="monitor-series">
            <span class="monitor-series-dot bg-violet-500"></span>{{ $t('写入') }} <span class="tabular-nums text-slate-600">{{ formatMonitorBytes(currentDiskRate(diskIOHistoryKey(dp), 'write'), true) }}</span>
            <span class="whitespace-nowrap">· {{ $t('累计写') }} <span class="tabular-nums">{{ formatMonitorBytes(diskIOByDevice(dp.device)?.writeBytes ?? 0) }}</span></span>
          </span>
        </div>
      </div>
    </div>
  </div>
</template>
