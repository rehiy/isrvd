<script lang="ts">
import { markRaw } from 'vue'
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'
import type { ChartOptions } from 'chart.js'

import type { SystemStat, SystemGPU } from '@/service/types'

import Chart from '@/helper/chart'
import { hexToRgba } from '@/helper/utils'

interface GpuHistory {
    labels: string[]
    util: number[]
    vram: number[]
    power: number[]
}

interface ChartCallbackContext {
    parsed: { y: number | null }
    dataset: { label?: string }
}

const MAX_HISTORY = 60

@Component
class SystemGpu extends Vue {
    @Ref readonly gpuContainerRef!: HTMLDivElement

    private gpuCharts: Record<string, Chart<'line'>> = {}
    gpuHistories: Record<string, GpuHistory> = {}
    currentGpus: SystemGPU[] = []

    memPercent(used: number, total: number): number {
        if (!total) return 0
        return parseFloat(((used / total) * 100).toFixed(1))
    }

    semanticColor(pct: number, prefix = 'bg') {
        if (pct >= 90) return `${prefix}-red-500`
        if (pct >= 70) return `${prefix}-amber-500`
        return `${prefix}-emerald-500`
    }

    textColor(pct: number) { return this.semanticColor(pct, 'text') }

    gpuTempColor(temp: number): string {
        if (temp < 0) return 'text-slate-400'
        if (temp >= 80) return 'text-red-500'
        if (temp >= 60) return 'text-amber-500'
        return 'text-emerald-500'
    }

    gpuKey(gpu: SystemGPU): string {
        return gpu.deviceKey || `gpu-${gpu.index}`
    }

    gpuChartOptions(): ChartOptions<'line'> {
        return {
            responsive: true, maintainAspectRatio: false, animation: false,
            interaction: { intersect: false, mode: 'index' as const },
            plugins: {
                legend: { display: true, position: 'bottom' as const, labels: { boxWidth: 8, padding: 8, font: { size: 10 }, color: '#64748b' } },
                tooltip: {
                    backgroundColor: 'rgba(15,23,42,0.9)', titleFont: { size: 10 }, bodyFont: { size: 10 }, padding: 8, cornerRadius: 6,
                    callbacks: {
                        label: (ctx: ChartCallbackContext) => {
                            const v = (ctx.parsed.y ?? 0).toFixed(1)
                            const label = ctx.dataset.label ?? ''
                            const unit = label === '功耗' ? 'W' : '%'
                            return label + ': ' + v + unit
                        }
                    }
                }
            },
            scales: {
                x: { display: false },
                y: {
                    display: true, beginAtZero: true, max: 100,
                    grid: { color: 'rgba(148,163,184,0.08)' }, border: { display: false },
                    ticks: { font: { size: 9 }, color: '#94a3b8', maxTicksLimit: 4, padding: 4, callback: (v: string | number) => Number(v).toFixed(0) + '%' }
                }
            },
            elements: { point: { radius: 0, hoverRadius: 3 }, line: { tension: 0.4, borderWidth: 1.5 } }
        }
    }

    makeDataset(data: number[], color: string, label: string) {
        return { label, data: [...data], borderColor: color, backgroundColor: hexToRgba(color, 0.1), fill: true }
    }

    initGpuChart(key: string) {
        const canvas = this.gpuContainerRef?.querySelector(`[data-gpu="${key}"]`) as HTMLCanvasElement | null
        if (!canvas) return

        this.gpuCharts[key]?.destroy()

        const h = this.gpuHistories[key] || { labels: [], util: [], vram: [], power: [] }
        this.gpuHistories[key] = h

        const chart = new Chart(canvas, {
            type: 'line' as const,
            data: {
                labels: [...h.labels],
                datasets: [
                    this.makeDataset(h.util, '#10b981', '使用率'),
                    this.makeDataset(h.vram, '#8b5cf6', '显存'),
                    this.makeDataset(h.power, '#f59e0b', '功耗')
                ]
            },
            options: this.gpuChartOptions()
        })
        this.gpuCharts[key] = markRaw(chart)
    }

    updateGpuChart(key: string) {
        const chart = this.gpuCharts[key]
        const h = this.gpuHistories[key]
        if (!chart || !h) return

        chart.data.labels = [...h.labels]
        chart.data.datasets[0].data = [...h.util]
        chart.data.datasets[1].data = [...h.vram]
        chart.data.datasets[2].data = [...h.power]
        chart.update('none')
    }

    pushData(payload: SystemStat) {
        this.currentGpus = payload.gpu || []
        if (!payload.gpu?.length) return

        const now = new Date()
        const label = `${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}:${now.getSeconds().toString().padStart(2, '0')}`

        payload.gpu.forEach(gpu => {
            const key = this.gpuKey(gpu)

            if (!this.gpuHistories[key]) {
                this.gpuHistories[key] = { labels: [], util: [], vram: [], power: [] }
            }
            const h = this.gpuHistories[key]

            h.labels.push(label)
            h.util.push(gpu.utilization)
            h.vram.push(this.memPercent(gpu.memoryUsed, gpu.memoryTotal))
            h.power.push(gpu.powerUsage >= 0 ? gpu.powerUsage : 0)

            if (h.labels.length > MAX_HISTORY) {
                h.labels.shift()
                h.util.shift()
                h.vram.shift()
                h.power.shift()
            }

            if (!this.gpuCharts[key]) {
                this.$nextTick(() => this.initGpuChart(key))
            } else {
                this.updateGpuChart(key)
            }
        })
    }

    unmounted() {
        Object.values(this.gpuCharts).forEach(c => c.destroy())
        this.gpuCharts = {}
    }
}

export default toNative(SystemGpu)
</script>

<template>
  <div v-if="currentGpus?.length" ref="gpuContainerRef" class="space-y-3">
    <div v-for="gpu in currentGpus" :key="gpuKey(gpu)" class="rounded-xl border border-slate-200 bg-white overflow-hidden">
      <div class="px-4 py-3 border-b border-slate-100 flex items-center gap-2">
        <div class="w-6 h-6 rounded-md bg-emerald-500 flex items-center justify-center">
          <i class="fas fa-microchip text-white text-xs"></i>
        </div>
        <span class="text-sm font-semibold text-slate-700">显卡<template v-if="currentGpus.length > 1"> {{ gpu.index }}</template></span>
        <span class="ml-auto text-xs text-slate-400 font-mono">{{ gpu.name }}</span>
      </div>
      <div class="px-4 py-3">
        <div class="flex items-center justify-end mb-2 gap-4 text-xs">
          <span class="flex items-center gap-1">
            <i class="fas fa-gauge text-emerald-500"></i>
            <span :class="['font-mono w-12 text-right tabular-nums', textColor(gpu.utilization)]">{{ gpu.utilization.toFixed(1) }}%</span>
          </span>
          <span v-if="gpu.memoryTotal > 0" class="flex items-center gap-1">
            <i class="fas fa-memory text-violet-500"></i>
            <span class="font-mono text-slate-600 w-12 text-right tabular-nums">{{ memPercent(gpu.memoryUsed, gpu.memoryTotal) }}%</span>
          </span>
          <span v-if="gpu.powerUsage >= 0" class="flex items-center gap-1">
            <i class="fas fa-bolt text-amber-500"></i>
            <span class="font-mono text-slate-600 w-14 text-right tabular-nums">{{ gpu.powerUsage.toFixed(1) }}W</span>
          </span>
          <span v-if="gpu.temperature >= 0" :class="['flex items-center gap-1', gpuTempColor(gpu.temperature)]">
            <i class="fas fa-temperature-half"></i>{{ gpu.temperature }}°C
          </span>
          <span v-if="gpu.fanSpeed >= 0" class="flex items-center gap-1 text-slate-400">
            <i class="fas fa-fan"></i>{{ gpu.fanSpeed }}%
          </span>
        </div>
        <div class="relative h-20 bg-slate-50 rounded-lg overflow-hidden">
          <canvas :data-gpu="gpuKey(gpu)" class="w-full h-full"></canvas>
          <div v-if="!gpuHistories[gpuKey(gpu)]?.labels?.length" class="absolute inset-0 flex items-center justify-center">
            <span class="text-xs text-slate-300">等待数据...</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
