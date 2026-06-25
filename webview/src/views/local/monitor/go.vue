<script lang="ts">
import type { ChartOptions } from 'chart.js'
import { markRaw } from 'vue'
import { Component, Prop, Ref, Vue, toNative } from 'vue-facing-decorator'

import type { SystemStat, SystemGoRuntimeStat } from '@/service/types'

import Chart from '@/helper/chart'
import { appendMonitorPoint } from '@/helper/monitor'
import { hexToRgba } from '@/helper/utils'

@Component
class SystemGo extends Vue {
    @Prop({ type: Number, default: 300 }) readonly rangeSeconds!: number

    @Ref readonly memCanvasRef!: HTMLCanvasElement
    @Ref readonly goroutineCanvasRef!: HTMLCanvasElement
    @Ref readonly sysCanvasRef!: HTMLCanvasElement
    @Ref readonly stackCanvasRef!: HTMLCanvasElement

    private memChart: Chart<'line'> | null = null
    private goroutineChart: Chart<'line'> | null = null
    private stackChart: Chart<'line'> | null = null
    private sysChart: Chart<'line'> | null = null

    private memHistory: { ts: number[]; labels: string[]; alloc: number[]; heapAlloc: number[]; heapInuse: number[]; heapIdle: number[]; heapReleased: number[]; heapSys: number[] } = { ts: [], labels: [], alloc: [], heapAlloc: [], heapInuse: [], heapIdle: [], heapReleased: [], heapSys: [] }
    private goroutineHistory: { ts: number[]; labels: string[]; goroutine: number[]; gc: number[]; heapObjects: number[] } = { ts: [], labels: [], goroutine: [], gc: [], heapObjects: [] }
    private stackHistory: { ts: number[]; labels: string[]; stackInuse: number[]; stackSys: number[] } = { ts: [], labels: [], stackInuse: [], stackSys: [] }
    private sysHistory: { ts: number[]; labels: string[]; totalAlloc: number[]; sys: number[] } = { ts: [], labels: [], totalAlloc: [], sys: [] }

    current: SystemGoRuntimeStat | null = null
    lastGCTime: string = '从未'

    fmtSize(bytes: number) {
        if (!bytes || bytes < 0) return '0 B'
        const units = ['B', 'KB', 'MB', 'GB', 'TB']
        let i = 0, v = bytes
        while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
        return `${v.toFixed(1)} ${units[i]}`
    }

    fmtGCTime(ts: number) {
        if (!ts) return '从未'
        this.lastGCTime = new Date(ts * 1000).toLocaleString('zh-CN')
        return this.lastGCTime
    }

    goChartOptions(): ChartOptions<'line'> {
        const fmtSize = (v: number) => this.fmtSize(v)
        return {
            responsive: true, maintainAspectRatio: false, animation: false,
            interaction: { intersect: false, mode: 'index' as const },
            plugins: {
                legend: { display: true, position: 'bottom' as const, labels: { boxWidth: 8, padding: 8, font: { size: 10 }, color: '#64748b' } },
                tooltip: {
                    backgroundColor: 'rgba(15,23,42,0.9)', titleFont: { size: 10 }, bodyFont: { size: 10 }, padding: 8, cornerRadius: 6,
                    callbacks: {
                        label: (ctx: { parsed: { y: number | null }; dataset: { label?: string } }) => {
                            const value = ctx.parsed.y ?? 0
                            const label = ctx.dataset.label || ''
                            if (label.includes('GC') || label === 'Goroutine' || label === '堆对象') {
                                return `${label}: ${value}`
                            }
                            return `${label}: ${fmtSize(value)}`
                        }
                    }
                }
            },
            scales: {
                x: { display: false },
                y: {
                    display: true, beginAtZero: true, grid: { color: 'rgba(148,163,184,0.08)' }, border: { display: false },
                    ticks: { font: { size: 9 }, color: '#94a3b8', maxTicksLimit: 4, padding: 4 }
                }
            },
            elements: { point: { radius: 0, hoverRadius: 3 }, line: { tension: 0.4, borderWidth: 1.5 } }
        }
    }

    makeDataset(data: number[], color: string, label: string) {
        return { label, data: [...data], borderColor: color, backgroundColor: hexToRgba(color, 0.1), fill: true }
    }

    pushData(payload: SystemStat, ts: number) {
        this.current = payload.go
        this.fmtGCTime(payload.go.lastGC)

        // 堆内存数据（包含空闲和释放）
        appendMonitorPoint(
            this.memHistory,
            ts,
            this.rangeSeconds,
            () => {
                this.memHistory.alloc.push(payload.go.alloc)
                this.memHistory.heapAlloc.push(payload.go.heapAlloc)
                this.memHistory.heapInuse.push(payload.go.heapInuse)
                this.memHistory.heapIdle.push(payload.go.heapIdle)
                this.memHistory.heapReleased.push(payload.go.heapReleased)
                this.memHistory.heapSys.push(payload.go.heapSys)
            },
            count => {
                this.memHistory.alloc.splice(0, count)
                this.memHistory.heapAlloc.splice(0, count)
                this.memHistory.heapInuse.splice(0, count)
                this.memHistory.heapIdle.splice(0, count)
                this.memHistory.heapReleased.splice(0, count)
                this.memHistory.heapSys.splice(0, count)
            }
        )

        // Goroutine、GC 和堆对象数据
        appendMonitorPoint(
            this.goroutineHistory,
            ts,
            this.rangeSeconds,
            () => {
                this.goroutineHistory.goroutine.push(payload.go.numGoroutine)
                this.goroutineHistory.gc.push(payload.go.numGC)
                this.goroutineHistory.heapObjects.push(payload.go.heapObjects)
            },
            count => {
                this.goroutineHistory.goroutine.splice(0, count)
                this.goroutineHistory.gc.splice(0, count)
                this.goroutineHistory.heapObjects.splice(0, count)
            }
        )

        // 栈内存数据
        appendMonitorPoint(
            this.stackHistory,
            ts,
            this.rangeSeconds,
            () => {
                this.stackHistory.stackInuse.push(payload.go.stackInuse)
                this.stackHistory.stackSys.push(payload.go.stackSys)
            },
            count => {
                this.stackHistory.stackInuse.splice(0, count)
                this.stackHistory.stackSys.splice(0, count)
            }
        )

        // 系统内存数据
        appendMonitorPoint(
            this.sysHistory,
            ts,
            this.rangeSeconds,
            () => {
                this.sysHistory.totalAlloc.push(payload.go.totalAlloc)
                this.sysHistory.sys.push(payload.go.sys)
            },
            count => {
                this.sysHistory.totalAlloc.splice(0, count)
                this.sysHistory.sys.splice(0, count)
            }
        )
    }

    initCharts() {
        if (this.memChart) { this.memChart.destroy(); this.memChart = null }
        if (this.goroutineChart) { this.goroutineChart.destroy(); this.goroutineChart = null }
        if (this.stackChart) { this.stackChart.destroy(); this.stackChart = null }
        if (this.sysChart) { this.sysChart.destroy(); this.sysChart = null }

        if (this.memCanvasRef) {
            this.memChart = markRaw(new Chart(this.memCanvasRef, {
                type: 'line' as const,
                data: {
                    labels: [...this.memHistory.labels],
                    datasets: [
                        this.makeDataset(this.memHistory.alloc, '#3b82f6', '已分配'),
                        this.makeDataset(this.memHistory.heapAlloc, '#10b981', '堆已分配'),
                        this.makeDataset(this.memHistory.heapInuse, '#f59e0b', '堆使用中'),
                        this.makeDataset(this.memHistory.heapIdle, '#a8a29e', '堆空闲'),
                        this.makeDataset(this.memHistory.heapReleased, '#f43f5e', '堆已释放'),
                        this.makeDataset(this.memHistory.heapSys, '#8b5cf6', '堆已申请')
                    ]
                },
                options: this.goChartOptions()
            }))
        }

        if (this.goroutineCanvasRef) {
            this.goroutineChart = markRaw(new Chart(this.goroutineCanvasRef, {
                type: 'line' as const,
                data: {
                    labels: [...this.goroutineHistory.labels],
                    datasets: [
                        this.makeDataset(this.goroutineHistory.goroutine, '#8b5cf6', 'Goroutine'),
                        { ...this.makeDataset(this.goroutineHistory.heapObjects, '#22c55e', '堆对象'), yAxisID: 'y1' },
                        { ...this.makeDataset(this.goroutineHistory.gc, '#ef4444', 'GC 次数'), yAxisID: 'y2' }
                    ]
                },
                options: {
                    responsive: true, maintainAspectRatio: false, animation: false,
                    interaction: { intersect: false, mode: 'index' as const },
                    plugins: {
                        legend: { display: true, position: 'bottom' as const, labels: { boxWidth: 8, padding: 8, font: { size: 10 }, color: '#64748b' } },
                        tooltip: {
                            backgroundColor: 'rgba(15,23,42,0.9)', titleFont: { size: 10 }, bodyFont: { size: 10 }, padding: 8, cornerRadius: 6,
                            callbacks: {
                                label: (ctx: { parsed: { y: number | null }; dataset: { label?: string } }) => {
                                    const value = ctx.parsed.y
                                    const label = ctx.dataset.label || ''
                                    return `${label}: ${value}`
                                }
                            }
                        }
                    },
                    scales: {
                        x: { display: false },
                        y: {
                            type: 'linear' as const,
                            display: true, beginAtZero: true, position: 'left' as const,
                            grid: { color: 'rgba(148,163,184,0.08)' }, border: { display: false },
                            ticks: { font: { size: 9 }, color: '#94a3b8', maxTicksLimit: 4, padding: 4 }
                        },
                        y1: {
                            type: 'linear' as const,
                            display: false,
                            position: 'right' as const,
                            beginAtZero: true,
                            grid: { display: false },
                            border: { display: false }
                        },
                        y2: {
                            type: 'linear' as const,
                            display: false,
                            position: 'right' as const,
                            beginAtZero: true,
                            grid: { display: false },
                            border: { display: false }
                        }
                    },
                    elements: { point: { radius: 0, hoverRadius: 3 }, line: { tension: 0.4, borderWidth: 1.5 } }
                }
            }))
        }

        if (this.stackCanvasRef) {
            this.stackChart = markRaw(new Chart(this.stackCanvasRef, {
                type: 'line' as const,
                data: {
                    labels: [...this.stackHistory.labels],
                    datasets: [
                        this.makeDataset(this.stackHistory.stackInuse, '#f59e0b', '栈已使用'),
                        this.makeDataset(this.stackHistory.stackSys, '#8b5cf6', '栈已申请')
                    ]
                },
                options: this.goChartOptions()
            }))
        }

        if (this.sysCanvasRef) {
            this.sysChart = markRaw(new Chart(this.sysCanvasRef, {
                type: 'line' as const,
                data: {
                    labels: [...this.sysHistory.labels],
                    datasets: [
                        this.makeDataset(this.sysHistory.totalAlloc, '#3b82f6', '累计分配'),
                        this.makeDataset(this.sysHistory.sys, '#10b981', '系统申请')
                    ]
                },
                options: this.goChartOptions()
            }))
        }
    }

    /**
     * 刷新图表：等待 DOM 更新后初始化或更新图表
     * 参考 network.vue 的 flushCharts 设计
     */
    flushCharts() {
        this.$nextTick(() => {
            if (!this.memChart || !this.goroutineChart || !this.stackChart || !this.sysChart) {
                this.initCharts()
            } else {
                this.updateCharts()
            }
        })
    }

    updateCharts() {
        if (!this.memChart || !this.goroutineChart || !this.stackChart || !this.sysChart) {
            return
        }
        this.memChart.data.labels = [...this.memHistory.labels]
        this.memChart.data.datasets[0].data = [...this.memHistory.alloc]
        this.memChart.data.datasets[1].data = [...this.memHistory.heapAlloc]
        this.memChart.data.datasets[2].data = [...this.memHistory.heapInuse]
        this.memChart.data.datasets[3].data = [...this.memHistory.heapIdle]
        this.memChart.data.datasets[4].data = [...this.memHistory.heapReleased]
        this.memChart.data.datasets[5].data = [...this.memHistory.heapSys]
        this.memChart.update('none')

        this.goroutineChart.data.labels = [...this.goroutineHistory.labels]
        this.goroutineChart.data.datasets[0].data = [...this.goroutineHistory.goroutine]
        this.goroutineChart.data.datasets[1].data = [...this.goroutineHistory.heapObjects]
        this.goroutineChart.data.datasets[2].data = [...this.goroutineHistory.gc]
        this.goroutineChart.update('none')

        this.stackChart.data.labels = [...this.stackHistory.labels]
        this.stackChart.data.datasets[0].data = [...this.stackHistory.stackInuse]
        this.stackChart.data.datasets[1].data = [...this.stackHistory.stackSys]
        this.stackChart.update('none')

        this.sysChart.data.labels = [...this.sysHistory.labels]
        this.sysChart.data.datasets[0].data = [...this.sysHistory.totalAlloc]
        this.sysChart.data.datasets[1].data = [...this.sysHistory.sys]
        this.sysChart.update('none')
    }

    clearData() {
        this.current = null
        this.lastGCTime = '从未'
        this.memHistory = { ts: [], labels: [], alloc: [], heapAlloc: [], heapInuse: [], heapIdle: [], heapReleased: [], heapSys: [] }
        this.goroutineHistory = { ts: [], labels: [], goroutine: [], gc: [], heapObjects: [] }
        this.stackHistory = { ts: [], labels: [], stackInuse: [], stackSys: [] }
        this.sysHistory = { ts: [], labels: [], totalAlloc: [], sys: [] }
        this.memChart?.destroy()
        this.goroutineChart?.destroy()
        this.stackChart?.destroy()
        this.sysChart?.destroy()
        this.memChart = null
        this.goroutineChart = null
        this.stackChart = null
        this.sysChart = null
    }

    unmounted() {
        this.memChart?.destroy()
        this.goroutineChart?.destroy()
        this.stackChart?.destroy()
        this.sysChart?.destroy()
    }
}

export default toNative(SystemGo)
</script>

<template>
  <div class="rounded-xl border border-slate-200 overflow-hidden">
    <div class="card-header">
      <div class="card-icon bg-sky-500">
        <i class="fas fa-code text-white text-xs"></i>
      </div>
      <span class="text-sm font-semibold text-slate-700">Go 运行态</span>
      <span v-if="current" class="ml-auto text-xs text-slate-400 font-mono">{{ current.version }}</span>
      <span v-else class="ml-auto text-xs text-slate-400">加载中...</span>
    </div>
    <div v-if="current" ref="goContainerRef" class="divide-y divide-slate-100">
      <!-- 系统内存折线图 -->
      <div class="px-4 py-3">
        <div class="flex items-center justify-between gap-x-3 gap-y-1 mb-2 flex-wrap">
          <span class="text-xs font-medium text-slate-500 shrink-0 whitespace-nowrap">系统内存</span>
          <div v-if="current" class="flex items-center justify-end gap-x-3 gap-y-1 text-xs flex-wrap flex-1 min-w-0">
            <span class="flex items-center gap-1 shrink-0 whitespace-nowrap">
              <span class="w-3 h-0.5 bg-blue-500 rounded-full"></span>
              <span class="font-mono text-slate-600">{{ fmtSize(current.totalAlloc) }}</span>
            </span>
            <span class="flex items-center gap-1 shrink-0 whitespace-nowrap">
              <span class="w-3 h-0.5 bg-emerald-500 rounded-full"></span>
              <span class="font-mono text-slate-600">{{ fmtSize(current.sys) }}</span>
            </span>
          </div>
        </div>
        <div class="relative h-28 bg-slate-50 rounded-lg overflow-hidden">
          <canvas ref="sysCanvasRef" class="w-full h-full"></canvas>
        </div>
      </div>

      <!-- 堆内存折线图 -->
      <div class="px-4 py-3">
        <div class="flex items-center justify-between gap-x-3 gap-y-1 mb-2 flex-wrap">
          <span class="text-xs font-medium text-slate-500 shrink-0 whitespace-nowrap">堆内存</span>
          <div v-if="current" class="flex items-center justify-end gap-x-3 gap-y-1 text-xs flex-wrap flex-1 min-w-0">
            <span class="flex items-center gap-1 shrink-0 whitespace-nowrap">
              <span class="w-3 h-0.5 bg-blue-500 rounded-full"></span>
              <span class="font-mono text-slate-600">{{ fmtSize(current.alloc) }}</span>
            </span>
            <span class="flex items-center gap-1 shrink-0 whitespace-nowrap">
              <span class="w-3 h-0.5 bg-emerald-500 rounded-full"></span>
              <span class="font-mono text-slate-600">{{ fmtSize(current.heapAlloc) }}</span>
            </span>
            <span class="flex items-center gap-1 shrink-0 whitespace-nowrap">
              <span class="w-3 h-0.5 bg-amber-500 rounded-full"></span>
              <span class="font-mono text-slate-600">{{ fmtSize(current.heapInuse) }}</span>
            </span>
            <span class="flex items-center gap-1 shrink-0 whitespace-nowrap">
              <span class="w-3 h-0.5 bg-gray-400 rounded-full"></span>
              <span class="font-mono text-slate-600">{{ fmtSize(current.heapIdle) }}</span>
            </span>
            <span class="flex items-center gap-1 shrink-0 whitespace-nowrap">
              <span class="w-3 h-0.5 bg-rose-500 rounded-full"></span>
              <span class="font-mono text-slate-600">{{ fmtSize(current.heapReleased) }}</span>
            </span>
            <span class="flex items-center gap-1 shrink-0 whitespace-nowrap">
              <span class="w-3 h-0.5 bg-purple-500 rounded-full"></span>
              <span class="font-mono text-slate-600">{{ fmtSize(current.heapSys) }}</span>
            </span>
          </div>
        </div>
        <div class="relative h-28 bg-slate-50 rounded-lg overflow-hidden">
          <canvas ref="memCanvasRef" class="w-full h-full"></canvas>
        </div>
      </div>

      <!-- 栈内存折线图 -->
      <div class="px-4 py-3">
        <div class="flex items-center justify-between gap-x-3 gap-y-1 mb-2 flex-wrap">
          <span class="text-xs font-medium text-slate-500 shrink-0 whitespace-nowrap">栈内存</span>
          <div v-if="current" class="flex items-center justify-end gap-x-3 gap-y-1 text-xs flex-wrap flex-1 min-w-0">
            <span class="flex items-center gap-1 shrink-0 whitespace-nowrap">
              <span class="w-3 h-0.5 bg-amber-500 rounded-full"></span>
              <span class="font-mono text-slate-600">{{ fmtSize(current.stackInuse) }}</span>
            </span>
            <span class="flex items-center gap-1 shrink-0 whitespace-nowrap">
              <span class="w-3 h-0.5 bg-purple-500 rounded-full"></span>
              <span class="font-mono text-slate-600">{{ fmtSize(current.stackSys) }}</span>
            </span>
          </div>
        </div>
        <div class="relative h-28 bg-slate-50 rounded-lg overflow-hidden">
          <canvas ref="stackCanvasRef" class="w-full h-full"></canvas>
        </div>
      </div>

      <!-- Goroutine & GC & 堆对象折线图 -->
      <div class="px-4 py-3">
        <div class="flex items-center justify-between gap-x-3 gap-y-1 mb-2 flex-wrap">
          <span class="text-xs font-medium text-slate-500 shrink-0 whitespace-nowrap">计数器</span>
          <div v-if="current" class="flex items-center justify-end gap-x-3 gap-y-1 text-xs flex-wrap flex-1 min-w-0">
            <span class="flex items-center gap-1 shrink-0 whitespace-nowrap" title="最后 GC 时间">
              <i class="fas fa-clock mr-1"></i>
              <span class="font-mono text-slate-600">{{ lastGCTime }}</span>
            </span>
            <span class="flex items-center gap-1 shrink-0 whitespace-nowrap">
              <span class="w-3 h-0.5 bg-purple-500 rounded-full"></span>
              <span class="font-mono text-slate-600">{{ current.numGoroutine }}</span>
            </span>
            <span class="flex items-center gap-1 shrink-0 whitespace-nowrap">
              <span class="w-3 h-0.5 bg-green-500 rounded-full"></span>
              <span class="font-mono text-slate-600">{{ current.heapObjects }}</span>
            </span>
            <span class="flex items-center gap-1 shrink-0 whitespace-nowrap">
              <span class="w-3 h-0.5 bg-red-500 rounded-full"></span>
              <span class="font-mono text-slate-600">{{ current.numGC }}</span>
            </span>
          </div>
        </div>
        <div class="relative h-28 bg-slate-50 rounded-lg overflow-hidden">
          <canvas ref="goroutineCanvasRef" class="w-full h-full"></canvas>
        </div>
      </div>
    </div>
  </div>
</template>
