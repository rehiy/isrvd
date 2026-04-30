<script lang="ts">
import { markRaw } from 'vue'
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'
import type { ChartOptions } from 'chart.js'

import type { SystemStat } from '@/service/types'

import Chart from '@/helper/chart'

interface ChartCallbackContext {
    parsed: { y: number | null }
    dataset: { label?: string }
}

const MAX_HISTORY = 60

@Component
class SystemCpuMem extends Vue {
    @Ref readonly cpuCanvasRef!: HTMLCanvasElement
    @Ref readonly memCanvasRef!: HTMLCanvasElement

    private cpuChart: Chart<'line'> | null = null
    private memChart: Chart<'line'> | null = null

    private cpuHistory: { labels: string[]; data: number[] } = { labels: [], data: [] }
    private memHistory: { labels: string[]; data: number[] } = { labels: [], data: [] }

    current: Pick<SystemStat['system'], 'CpuPercent' | 'CpuModel' | 'MemoryUsed' | 'MemoryTotal'> | null = null

    get cpuVal() {
        return this.current ? this.avgCpuPercent(this.current.CpuPercent) : 0
    }

    get memVal() {
        return this.current ? this.memPercent(this.current.MemoryUsed, this.current.MemoryTotal) : 0
    }

    get cpuModel() {
        return this.current?.CpuModel?.[0] || ''
    }

    get memUsed() { return this.current?.MemoryUsed ?? 0 }
    get memTotal() { return this.current?.MemoryTotal ?? 0 }

    avgCpuPercent(arr: number[]): number {
        if (!arr || !arr.length) return 0
        return parseFloat(((arr.reduce((a, b) => a + b, 0) / arr.length)).toFixed(1))
    }

    memPercent(used: number, total: number): number {
        if (!total) return 0
        return parseFloat(((used / total) * 100).toFixed(1))
    }

    fmtBytes(bytes: number) {
        if (!bytes || bytes < 0) return '0 B'
        const units = ['B', 'KB', 'MB', 'GB', 'TB']
        let i = 0, v = bytes
        while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
        return `${v.toFixed(1)} ${units[i]}`
    }

    semanticColor(pct: number, prefix = 'bg') {
        if (pct >= 90) return `${prefix}-red-500`
        if (pct >= 70) return `${prefix}-amber-500`
        return `${prefix}-emerald-500`
    }

    textColor(pct: number) { return this.semanticColor(pct, 'text') }

    bgChartOptions(): ChartOptions<'line'> {
        return {
            responsive: true, maintainAspectRatio: false, animation: false,
            interaction: { intersect: false, mode: 'index' as const },
            plugins: {
                legend: { display: false },
                tooltip: {
                    backgroundColor: 'rgba(15,23,42,0.85)', titleFont: { size: 10 }, bodyFont: { size: 10 }, padding: 6, cornerRadius: 6,
                    callbacks: { label: (ctx: ChartCallbackContext) => (ctx.parsed.y ?? 0).toFixed(1) + '%' }
                }
            },
            scales: {
                x: { display: false },
                y: { display: false, beginAtZero: true, max: 100, grid: { display: false }, border: { display: false } }
            },
            elements: { point: { radius: 0, hoverRadius: 3 }, line: { tension: 0.4, borderWidth: 2 } }
        }
    }

    initCharts() {
        if (this.cpuChart) { this.cpuChart.destroy(); this.cpuChart = null }
        if (this.memChart) { this.memChart.destroy(); this.memChart = null }

        if (this.cpuCanvasRef) {
            const chart = new Chart(this.cpuCanvasRef, {
                type: 'line' as const,
                data: {
                    labels: [...this.cpuHistory.labels],
                    datasets: [{ data: [...this.cpuHistory.data], borderColor: 'rgba(59,130,246,0.6)', backgroundColor: 'rgba(59,130,246,0.08)', fill: true }]
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
                    datasets: [{ data: [...this.memHistory.data], borderColor: 'rgba(99,102,241,0.6)', backgroundColor: 'rgba(99,102,241,0.08)', fill: true }]
                },
                options: this.bgChartOptions()
            })
            this.memChart = markRaw(chart)
        }
    }

    updateCharts() {
        if (this.cpuChart) {
            this.cpuChart.data.labels = [...this.cpuHistory.labels]
            this.cpuChart.data.datasets[0].data = [...this.cpuHistory.data]
            this.cpuChart.update('none')
        }
        if (this.memChart) {
            this.memChart.data.labels = [...this.memHistory.labels]
            this.memChart.data.datasets[0].data = [...this.memHistory.data]
            this.memChart.update('none')
        }
    }

    pushData(payload: SystemStat) {
        const s = payload.system
        this.current = { CpuPercent: s.CpuPercent, CpuModel: s.CpuModel, MemoryUsed: s.MemoryUsed, MemoryTotal: s.MemoryTotal }

        const now = new Date()
        const label = `${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}:${now.getSeconds().toString().padStart(2, '0')}`

        this.cpuHistory.labels.push(label)
        this.cpuHistory.data.push(this.avgCpuPercent(s.CpuPercent))
        this.memHistory.labels.push(label)
        this.memHistory.data.push(this.memPercent(s.MemoryUsed, s.MemoryTotal))

        if (this.cpuHistory.labels.length > MAX_HISTORY) {
            this.cpuHistory.labels.shift()
            this.cpuHistory.data.shift()
        }
        if (this.memHistory.labels.length > MAX_HISTORY) {
            this.memHistory.labels.shift()
            this.memHistory.data.shift()
        }

        if (!this.cpuChart || !this.memChart) {
            this.initCharts()
        } else {
            this.updateCharts()
        }
    }

    mounted() {
        this.initCharts()
    }

    unmounted() {
        this.cpuChart?.destroy()
        this.memChart?.destroy()
    }
}

export default toNative(SystemCpuMem)
</script>

<template>
  <div v-if="current" class="grid grid-cols-1 md:grid-cols-2 gap-3">
    <div class="relative rounded-xl border border-slate-200 bg-white overflow-hidden">
      <div class="absolute inset-0 pointer-events-none">
        <canvas ref="cpuCanvasRef" class="w-full h-full"></canvas>
      </div>
      <div class="relative p-4">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <div class="w-7 h-7 rounded-lg bg-blue-500/90 flex items-center justify-center">
              <i class="fas fa-microchip text-white text-xs"></i>
            </div>
            <span class="text-sm font-semibold text-slate-700">CPU 使用率</span>
          </div>
          <span :class="['text-2xl font-bold tabular-nums', textColor(cpuVal)]">
            {{ cpuVal }}<span class="text-sm font-medium ml-0.5">%</span>
          </span>
        </div>
        <p v-if="cpuModel" class="text-xs text-slate-400 mt-3 truncate">{{ cpuModel }}</p>
        <p v-else class="mt-3 text-xs text-slate-300">—</p>
      </div>
    </div>

    <div class="relative rounded-xl border border-slate-200 bg-white overflow-hidden">
      <div class="absolute inset-0 pointer-events-none">
        <canvas ref="memCanvasRef" class="w-full h-full"></canvas>
      </div>
      <div class="relative p-4">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <div class="w-7 h-7 rounded-lg bg-indigo-500/90 flex items-center justify-center">
              <i class="fas fa-memory text-white text-xs"></i>
            </div>
            <span class="text-sm font-semibold text-slate-700">内存使用</span>
          </div>
          <span :class="['text-2xl font-bold tabular-nums', textColor(memVal)]">
            {{ memVal }}<span class="text-sm font-medium ml-0.5">%</span>
          </span>
        </div>
        <p class="text-xs text-slate-400 mt-3">{{ fmtBytes(memUsed) }} / {{ fmtBytes(memTotal) }}</p>
      </div>
    </div>
  </div>
</template>
