<script lang="ts">
import type { ChartOptions } from 'chart.js'
import { markRaw } from 'vue'
import { Component, Prop, Ref, Vue, toNative } from 'vue-facing-decorator'

import type { SystemStat, SystemNetInterface } from '@/service/types'

import Chart, { findMonitorCanvas, makeLineDataset, makeMonitorChartOptions, makeMonitorScale, syncMonitorCharts } from '@/helper/chart'
import { appendMonitorPoint, formatMonitorBytes } from '@/helper/monitor'

interface TimeSeriesHistory {
    ts: number[]
    labels: string[]
    recv: number[]
    sent: number[]
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
    formatMonitorBytes = formatMonitorBytes

    physicalInterfaces(list: SystemNetInterface[]) {
        if (!list) {
            return []
        }
        const virtualPrefixes = ['lo', 'docker', 'veth', 'br-', 'overlay', 'flannel', 'cni', 'tunl', 'dummy', 'virbr']
        return list
            .filter(ni => !virtualPrefixes.some(p => ni.name.startsWith(p)))
            .sort((a, b) => NATURAL_COLLATOR.compare(a.name, b.name))
    }

    currentRate(name: string, dir: string): number {
        const h = this.netHistory[name]
        if (!h) {
            return 0
        }
        const arr = h[dir as keyof TimeSeriesHistory] as number[]
        return arr?.length ? arr[arr.length - 1] : 0
    }

    netChartOptions(): ChartOptions<'line'> {
        return makeMonitorChartOptions(
            { y: makeMonitorScale('rate') },
            ctx => (ctx.dataset.label ?? '') + ': ' + formatMonitorBytes(ctx.parsed.y ?? 0, true)
        )
    }

    initNetChart(name: string) {
        const canvas = findMonitorCanvas(this.netContainerRef, 'data-iface', name)
        if (!canvas) {
            return
        }
        this.netCharts[name]?.destroy()
        const h = this.netHistory[name] || { ts: [], labels: [], recv: [], sent: [] }
        this.netHistory[name] = h
        const chart = new Chart(canvas, {
            type: 'line' as const,
            data: { labels: [...h.labels], datasets: [makeLineDataset(h.recv, '#10b981', this.$t('下行')), makeLineDataset(h.sent, '#3b82f6', this.$t('上行'))] },
            options: this.netChartOptions()
        })
        this.netCharts[name] = markRaw(chart)
    }

    updateNetChart(name: string) {
        const chart = this.netCharts[name]
        const h = this.netHistory[name]
        if (!chart || !h) {
            return
        }
        chart.data.labels = [...h.labels]
        chart.data.datasets[0].data = [...h.recv]
        chart.data.datasets[1].data = [...h.sent]
        chart.update('none')
    }

    flushCharts() {
        this.$nextTick(() => {
            syncMonitorCharts(
                this.netCharts, this.netContainerRef, 'data-iface', this.currentIfaces.map(ni => ni.name),
                name => this.initNetChart(name), name => this.updateNetChart(name)
            )
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
        const active = new Set(physicalIfaces.map(ni => ni.name))
        for (const name of Object.keys(this.netHistory)) {
            if (!active.has(name)) {
                delete this.netHistory[name]
                delete this.lastNetIO[name]
            }
        }
        if (!ifaces.length) {
            return
        }

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
        this.clearData()
    }
}

export default toNative(SystemNetwork)
</script>

<template>
  <div v-if="currentIfaces.length" class="space-y-3">
    <div ref="netContainerRef" class="grid grid-cols-1 gap-4">
      <div v-for="ni in currentIfaces" :key="ni.name" class="metric-card min-w-0">
        <div class="absolute inset-x-0 top-0 h-0.5 bg-gradient-to-r from-cyan-400 to-transparent"></div>
        <div class="monitor-card-header">
          <div class="monitor-card-heading">
            <div class="monitor-card-icon">
              <i class="fas fa-network-wired text-sm"></i>
            </div>
            <p class="min-w-0 flex-1 monitor-card-title" :title="`${$t('网络')} - ${ni.name}`">{{ $t('网络') }} - {{ ni.name }}</p>
          </div>
          <div class="monitor-card-metrics">
            <div class="monitor-metrics-row">
              <span class="inline-flex flex-wrap items-baseline justify-end gap-1.5" :title="$t('未获取网卡带宽，无法计算使用率')" :aria-label="`${$t('使用率')}：${$t('未获取网卡带宽，无法计算使用率')}`">{{ $t('使用率') }} <span class="monitor-card-value text-slate-700">—</span></span>
            </div>
            <div class="monitor-metrics-row">
              <span class="monitor-series max-w-full">
                <span class="monitor-series-dot bg-emerald-500"></span>{{ $t('下行') }} <span class="tabular-nums text-slate-600">{{ formatMonitorBytes(currentRate(ni.name, 'recv'), true) }}</span>
                <span class="whitespace-nowrap">· {{ $t('累计收') }} <span class="tabular-nums">{{ formatMonitorBytes(ni.bytesRecv) }}</span></span>
              </span>
              <span class="monitor-series max-w-full">
                <span class="monitor-series-dot bg-blue-500"></span>{{ $t('上行') }} <span class="tabular-nums text-slate-600">{{ formatMonitorBytes(currentRate(ni.name, 'sent'), true) }}</span>
                <span class="whitespace-nowrap">· {{ $t('累计发') }} <span class="tabular-nums">{{ formatMonitorBytes(ni.bytesSent) }}</span></span>
              </span>
            </div>
          </div>
        </div>
        <div class="monitor-chart-box">
          <canvas :data-iface="ni.name" class="w-full h-full"></canvas>
          <div v-if="!netHistory[ni.name]?.labels?.length" class="absolute inset-0 flex items-center justify-center">
            <span class="text-xs text-slate-300">{{ $t('等待数据...') }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
