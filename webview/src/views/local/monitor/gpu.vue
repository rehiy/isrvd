<script lang="ts">
import type { ChartOptions } from 'chart.js'
import { markRaw } from 'vue'
import { Component, Prop, Ref, Vue, toNative } from 'vue-facing-decorator'

import type { SystemStat, SystemGPU } from '@/service/types'

import Chart, { findMonitorCanvas, makeLineDataset, makeMonitorChartOptions, makeMonitorScale, syncMonitorCharts } from '@/helper/chart'
import { appendMonitorPoint, usagePercent, semanticColor } from '@/helper/monitor'

interface GpuHistory {
    ts: number[]
    labels: string[]
    util: number[]
    vram: number[]
    power: (number | null)[]
}

@Component
class SystemGpu extends Vue {
    @Prop({ type: Number, default: 300 }) readonly rangeSeconds!: number

    @Ref readonly gpuContainerRef!: HTMLDivElement

    private gpuCharts: Record<string, Chart<'line'>> = {}
    gpuHistories: Record<string, GpuHistory> = {}
    currentGpus: SystemGPU[] = []

    memPercent = usagePercent

    semanticColor = semanticColor

    gpuTempColor(temp: number): string {
        if (temp < 0) {
            return 'text-slate-400'
        }
        if (temp >= 80) {
            return 'text-red-500'
        }
        if (temp >= 60) {
            return 'text-amber-500'
        }
        return 'text-emerald-500'
    }

    gpuKey(gpu: SystemGPU): string {
        return gpu.deviceKey || `gpu-${gpu.index}`
    }

    gpuChartOptions(): ChartOptions<'line'> {
        return makeMonitorChartOptions(
            {
                y: makeMonitorScale('percent', { position: 'left' }),
                y1: makeMonitorScale('power', {
                    position: 'right', grid: { drawOnChartArea: false }, ticks: { color: '#f59e0b' }
                })
            },
            ctx => {
                const value = (ctx.parsed.y ?? 0).toFixed(1)
                const unit = ctx.dataset.yAxisID === 'y1' ? ' W' : '%'
                return (ctx.dataset.label ?? '') + ': ' + value + unit
            }
        )
    }

    initGpuChart(key: string) {
        const canvas = findMonitorCanvas(this.gpuContainerRef, 'data-gpu', key)
        if (!canvas) {
            return
        }

        this.gpuCharts[key]?.destroy()

        const h = this.gpuHistories[key] || { ts: [], labels: [], util: [], vram: [], power: [] }
        this.gpuHistories[key] = h

        const chart = new Chart(canvas, {
            type: 'line' as const,
            data: {
                labels: [...h.labels],
                datasets: [
                    makeLineDataset(h.util, '#10b981', this.$t('使用率')),
                    makeLineDataset(h.vram, '#8b5cf6', this.$t('显存')),
                    makeLineDataset(h.power, '#f59e0b', this.$t('功耗'), { yAxisID: 'y1', fill: false })
                ]
            },
            options: this.gpuChartOptions()
        })
        this.gpuCharts[key] = markRaw(chart)
    }

    updateGpuChart(key: string) {
        const chart = this.gpuCharts[key]
        const h = this.gpuHistories[key]
        if (!chart || !h) {
            return
        }

        chart.data.labels = [...h.labels]
        chart.data.datasets[0].data = [...h.util]
        chart.data.datasets[1].data = [...h.vram]
        chart.data.datasets[2].data = [...h.power]
        chart.update('none')
    }

    flushCharts() {
        this.$nextTick(() => {
            syncMonitorCharts(
                this.gpuCharts, this.gpuContainerRef, 'data-gpu', this.currentGpus.map(gpu => this.gpuKey(gpu)),
                key => this.initGpuChart(key), key => this.updateGpuChart(key)
            )
        })
    }

    clearData() {
        this.currentGpus = []
        this.gpuHistories = {}
        Object.values(this.gpuCharts).forEach(c => c.destroy())
        this.gpuCharts = {}
    }

    pushData(payload: SystemStat, ts: number) {
        this.currentGpus = payload.gpu || []
        const active = new Set(this.currentGpus.map(gpu => this.gpuKey(gpu)))
        for (const key of Object.keys(this.gpuHistories)) {
            if (!active.has(key)) {
                delete this.gpuHistories[key]
            }
        }
        if (!payload.gpu?.length) {
            return
        }

        payload.gpu.forEach(gpu => {
            const key = this.gpuKey(gpu)

            if (!this.gpuHistories[key]) {
                this.gpuHistories[key] = { ts: [], labels: [], util: [], vram: [], power: [] }
            }
            const h = this.gpuHistories[key]

            appendMonitorPoint(
                h,
                ts,
                this.rangeSeconds,
                () => {
                    h.util.push(gpu.utilization)
                    h.vram.push(this.memPercent(gpu.memoryUsed, gpu.memoryTotal))
                    h.power.push(gpu.powerUsage >= 0 ? gpu.powerUsage : null)
                },
                count => {
                    h.util.splice(0, count)
                    h.vram.splice(0, count)
                    h.power.splice(0, count)
                }
            )
        })
    }

    unmounted() {
        this.clearData()
    }
}

export default toNative(SystemGpu)
</script>

<template>
  <div v-if="currentGpus?.length" ref="gpuContainerRef" class="grid grid-cols-1 gap-4 md:grid-cols-2">
    <div v-for="gpu in currentGpus" :key="gpuKey(gpu)" class="metric-card min-w-0">
      <div class="absolute inset-x-0 top-0 h-0.5 bg-gradient-to-r from-emerald-400 to-transparent"></div>
      <div class="monitor-card-header">
        <div class="monitor-card-heading">
          <div class="monitor-card-icon">
            <i class="fas fa-microchip text-sm"></i>
          </div>
          <div class="min-w-0 flex-1">
            <p class="monitor-card-title">{{ $t('显卡') }}<template v-if="currentGpus.length > 1"> {{ gpu.index }}</template></p>
            <p class="truncate text-xs text-slate-400" :title="gpu.name">{{ gpu.name }}</p>
          </div>
        </div>
        <div class="monitor-card-metrics">
          <span class="monitor-headline">{{ $t('使用率') }}
            <span :class="['monitor-card-value', semanticColor(gpu.utilization)]">{{ gpu.utilization.toFixed(1) }}%</span>
          </span>
        </div>
      </div>
      <div class="monitor-chart-box">
        <canvas :data-gpu="gpuKey(gpu)" class="w-full h-full"></canvas>
        <div v-if="!gpuHistories[gpuKey(gpu)]?.labels?.length" class="absolute inset-0 flex items-center justify-center">
          <span class="text-xs text-slate-300">{{ $t('等待数据...') }}</span>
        </div>
      </div>
      <div class="monitor-card-legend">
        <span class="monitor-series"><span class="monitor-series-dot bg-emerald-500"></span>{{ $t('使用率') }}</span>
        <span v-if="gpu.memoryTotal > 0" class="monitor-series">
          <span class="monitor-series-dot bg-violet-500"></span>{{ $t('显存') }}
          <span class="text-slate-600 tabular-nums">{{ memPercent(gpu.memoryUsed, gpu.memoryTotal) }}%</span>
        </span>
        <span v-if="gpu.powerUsage >= 0" class="monitor-series">
          <span class="monitor-series-dot bg-amber-500"></span>{{ $t('功耗') }}
          <span class="text-slate-600 tabular-nums">{{ gpu.powerUsage.toFixed(1) }} W</span>
        </span>
        <span v-if="gpu.temperature >= 0" :class="['inline-flex shrink-0 items-center gap-1 whitespace-nowrap', gpuTempColor(gpu.temperature)]">
          <i class="fas fa-temperature-half"></i>{{ gpu.temperature }}°C
        </span>
        <span v-if="gpu.fanSpeed >= 0" class="inline-flex shrink-0 items-center gap-1 whitespace-nowrap">
          <i class="fas fa-fan"></i>{{ gpu.fanSpeed }}%
        </span>
      </div>
    </div>
  </div>
</template>
