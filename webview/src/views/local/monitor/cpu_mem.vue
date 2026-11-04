<script lang="ts">
import type { ChartOptions } from 'chart.js'
import { markRaw } from 'vue'
import { Component, Prop, Ref, Vue, toNative } from 'vue-facing-decorator'

import type { SystemStat } from '@/service/types'

import Chart, { makeLineDataset, makeMonitorChartOptions, makeMonitorScale } from '@/helper/chart'
import { appendMonitorPoint, formatMonitorBytes, usagePercent, semanticColor } from '@/helper/monitor'

@Component
class SystemCpuMem extends Vue {
    @Prop({ type: Number, default: 300 }) readonly rangeSeconds!: number

    @Ref readonly cpuCanvasRef!: HTMLCanvasElement
    @Ref readonly memCanvasRef!: HTMLCanvasElement

    private cpuChart: Chart<'line'> | null = null
    private memChart: Chart<'line'> | null = null

    private cpuHistory: { ts: number[]; labels: string[]; data: number[] } = { ts: [], labels: [], data: [] }
    private memHistory: { ts: number[]; labels: string[]; data: number[] } = { ts: [], labels: [], data: [] }

    current: Pick<SystemStat['system'], 'cpuPercent' | 'cpuModel' | 'memoryUsed' | 'memoryTotal'> | null = null
    formatMonitorBytes = formatMonitorBytes

    get cpuVal() {
        return this.current ? this.avgCpuPercent(this.current.cpuPercent) : 0
    }

    get memVal() {
        return this.current ? this.memPercent(this.current.memoryUsed, this.current.memoryTotal) : 0
    }

    get cpuModel() {
        return this.current?.cpuModel?.[0] || ''
    }

    get memUsed() { return this.current?.memoryUsed ?? 0 }
    get memTotal() { return this.current?.memoryTotal ?? 0 }

    avgCpuPercent(arr: number[]): number {
        if (!arr || !arr.length) {
            return 0
        }
        return parseFloat(((arr.reduce((a, b) => a + b, 0) / arr.length)).toFixed(1))
    }

    memPercent = usagePercent

    semanticColor = semanticColor

    bgChartOptions(): ChartOptions<'line'> {
        return makeMonitorChartOptions(
            { y: makeMonitorScale('percent') },
            ctx => (ctx.parsed.y ?? 0).toFixed(1) + '%'
        )
    }

    initCharts() {
        if (this.cpuChart) {
            this.cpuChart.destroy()
            this.cpuChart = null
        }
        if (this.memChart) {
            this.memChart.destroy()
            this.memChart = null
        }

        if (this.cpuCanvasRef) {
            const chart = new Chart(this.cpuCanvasRef, {
                type: 'line' as const,
                data: {
                    labels: [...this.cpuHistory.labels],
                    datasets: [makeLineDataset(this.cpuHistory.data, '#3b82f6', '', { opacity: 0.6, fillOpacity: 0.08 })]
                },
                options: this.bgChartOptions()
            })
            this.cpuChart = markRaw(chart)
        }

        if (this.memCanvasRef) {
            const chart = new Chart(this.memCanvasRef, {
                type: 'line' as const,
                data: {
                    labels: [...this.memHistory.labels],
                    datasets: [makeLineDataset(this.memHistory.data, '#6366f1', '', { opacity: 0.6, fillOpacity: 0.08 })]
                },
                options: this.bgChartOptions()
            })
            this.memChart = markRaw(chart)
        }
    }

    flushCharts() {
        this.$nextTick(() => {
            if (!this.current || !this.cpuCanvasRef?.isConnected || !this.memCanvasRef?.isConnected) {
                return
            }
            if (!this.cpuChart || !this.memChart || this.cpuChart.canvas !== this.cpuCanvasRef || this.memChart.canvas !== this.memCanvasRef) {
                this.initCharts()
                return
            }
            this.cpuChart.data.labels = [...this.cpuHistory.labels]
            this.cpuChart.data.datasets[0].data = [...this.cpuHistory.data]
            this.cpuChart.update('none')

            this.memChart.data.labels = [...this.memHistory.labels]
            this.memChart.data.datasets[0].data = [...this.memHistory.data]
            this.memChart.update('none')
        })
    }

    clearData() {
        this.current = null
        this.cpuHistory = { ts: [], labels: [], data: [] }
        this.memHistory = { ts: [], labels: [], data: [] }
        this.cpuChart?.destroy()
        this.memChart?.destroy()
        this.cpuChart = null
        this.memChart = null
    }

    pushData(payload: SystemStat, ts: number) {
        const s = payload.system
        this.current = { cpuPercent: s.cpuPercent, cpuModel: s.cpuModel, memoryUsed: s.memoryUsed, memoryTotal: s.memoryTotal }

        appendMonitorPoint(
            this.cpuHistory,
            ts,
            this.rangeSeconds,
            () => this.cpuHistory.data.push(this.avgCpuPercent(s.cpuPercent)),
            count => this.cpuHistory.data.splice(0, count)
        )
        appendMonitorPoint(
            this.memHistory,
            ts,
            this.rangeSeconds,
            () => this.memHistory.data.push(this.memPercent(s.memoryUsed, s.memoryTotal)),
            count => this.memHistory.data.splice(0, count)
        )
    }

    unmounted() {
        this.clearData()
    }
}

export default toNative(SystemCpuMem)
</script>

<template>
  <div v-if="current" class="grid grid-cols-1 gap-4 md:grid-cols-2">
    <div class="metric-card min-w-0">
      <div class="absolute top-0 left-0 right-0 h-0.5 bg-gradient-to-r from-blue-400 to-transparent"></div>
      <div class="monitor-card-header">
        <div class="monitor-card-heading">
          <div class="monitor-card-icon">
            <i class="fas fa-microchip text-sm"></i>
          </div>
          <div class="min-w-0 flex-1">
            <p class="monitor-card-title">{{ $t('CPU 使用率') }}</p>
            <p v-if="cpuModel" class="truncate text-xs text-slate-400" :title="cpuModel">{{ cpuModel }}</p>
            <p v-else class="text-xs text-slate-300">—</p>
          </div>
        </div>
        <span :class="['monitor-card-value ml-auto whitespace-nowrap text-right', semanticColor(cpuVal)]">{{ cpuVal }}%</span>
      </div>
      <div class="monitor-chart-box">
        <canvas ref="cpuCanvasRef" class="w-full h-full"></canvas>
      </div>
    </div>

    <div class="metric-card min-w-0">
      <div class="absolute top-0 left-0 right-0 h-0.5 bg-gradient-to-r from-indigo-400 to-transparent"></div>
      <div class="monitor-card-header">
        <div class="monitor-card-heading">
          <div class="monitor-card-icon">
            <i class="fas fa-memory text-sm"></i>
          </div>
          <div class="min-w-0 flex-1">
            <p class="monitor-card-title">{{ $t('内存使用') }}</p>
            <p class="break-words text-xs text-slate-400">{{ formatMonitorBytes(memUsed) }} / {{ formatMonitorBytes(memTotal) }}</p>
          </div>
        </div>
        <span :class="['monitor-card-value ml-auto whitespace-nowrap text-right', semanticColor(memVal)]">{{ memVal }}%</span>
      </div>
      <div class="monitor-chart-box">
        <canvas ref="memCanvasRef" class="w-full h-full"></canvas>
      </div>
    </div>
  </div>
</template>
