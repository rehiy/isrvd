<script lang="ts">
import type { ChartOptions } from 'chart.js'
import { markRaw } from 'vue'
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import type { SystemStat, SystemGoRuntimeStat } from '@/service/types'

import Chart from '@/helper/chart'

const MAX_HISTORY = 60

@Component
class SystemGo extends Vue {
    @Ref readonly memCanvasRef!: HTMLCanvasElement
    @Ref readonly goroutineCanvasRef!: HTMLCanvasElement
    @Ref readonly sysCanvasRef!: HTMLCanvasElement
    @Ref readonly stackCanvasRef!: HTMLCanvasElement
    @Ref readonly heapCanvasRef!: HTMLCanvasElement

    private memChart: Chart<'line'> | null = null
    private goroutineChart: Chart<'line'> | null = null
    private stackChart: Chart<'line'> | null = null
    private sysChart: Chart<'line'> | null = null
    private heapChart: Chart<'line'> | null = null

    private memHistory: { labels: string[]; alloc: number[]; heapAlloc: number[]; heapInuse: number[]; heapSys: number[] } = { labels: [], alloc: [], heapAlloc: [], heapInuse: [], heapSys: [] }
    private goroutineHistory: { labels: string[]; goroutine: number[]; gc: number[] } = { labels: [], goroutine: [], gc: [] }
    private stackHistory: { labels: string[]; stackInuse: number[]; stackSys: number[] } = { labels: [], stackInuse: [], stackSys: [] }
    private sysHistory: { labels: string[]; totalAlloc: number[]; sys: number[] } = { labels: [], totalAlloc: [], sys: [] }
    private heapHistory: { labels: string[]; heapIdle: number[]; heapReleased: number[]; heapObjects: number[] } = { labels: [], heapIdle: [], heapReleased: [], heapObjects: [] }

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

    bgChartOptions(title?: string): ChartOptions<'line'> {
        return {
            responsive: true, maintainAspectRatio: false, animation: false,
            interaction: { intersect: false, mode: 'index' as const },
            plugins: {
                legend: { display: false },
                title: title ? { display: true, text: title, font: { size: 10 }, color: '#64748b' } : undefined,
                tooltip: {
                    backgroundColor: 'rgba(15,23,42,0.85)', titleFont: { size: 10 }, bodyFont: { size: 10 }, padding: 6, cornerRadius: 6,
                    callbacks: {
                        label: (ctx: any) => {
                            const value = ctx.parsed.y
                            const label = ctx.dataset.label || ''
                            if (label.includes('GC') || label === 'Goroutine') {
                                return `${label}: ${value}`
                            }
                            return `${label}: ${this.fmtSize(value)}`
                        }
                    }
                }
            },
            scales: {
                x: { display: false },
                y: { display: false, beginAtZero: true, grid: { display: false }, border: { display: false } }
            },
            elements: { point: { radius: 0, hoverRadius: 3 }, line: { tension: 0.4, borderWidth: 2 } }
        }
    }

    pushData(payload: SystemStat, ts: number) {
        this.current = payload.go
        this.fmtGCTime(payload.go.lastGC)

        const t = new Date(ts * 1000)
        const label = `${t.getHours().toString().padStart(2, '0')}:${t.getMinutes().toString().padStart(2, '0')}:${t.getSeconds().toString().padStart(2, '0')}`

        // 堆内存数据
        this.memHistory.labels.push(label)
        this.memHistory.alloc.push(payload.go.alloc)
        this.memHistory.heapAlloc.push(payload.go.heapAlloc)
        this.memHistory.heapInuse.push(payload.go.heapInuse)
        this.memHistory.heapSys.push(payload.go.heapSys)

        // 堆空闲/释放/对象数据
        this.heapHistory.labels.push(label)
        this.heapHistory.heapIdle.push(payload.go.heapIdle)
        this.heapHistory.heapReleased.push(payload.go.heapReleased)
        this.heapHistory.heapObjects.push(payload.go.heapObjects)

        // Goroutine 和 GC 数据（合并）
        this.goroutineHistory.labels.push(label)
        this.goroutineHistory.goroutine.push(payload.go.numGoroutine)
        this.goroutineHistory.gc.push(payload.go.numGC)

        // 栈内存数据
        this.stackHistory.labels.push(label)
        this.stackHistory.stackInuse.push(payload.go.stackInuse)
        this.stackHistory.stackSys.push(payload.go.stackSys)

        // 系统内存数据
        this.sysHistory.labels.push(label)
        this.sysHistory.totalAlloc.push(payload.go.totalAlloc)
        this.sysHistory.sys.push(payload.go.sys)

        // 保持历史数据不超过最大值
        ;[this.memHistory, this.goroutineHistory, this.stackHistory, this.sysHistory, this.heapHistory].forEach(h => {
            if (h.labels.length > MAX_HISTORY) {
                h.labels.shift()
                Object.keys(h).forEach(key => {
                    if (key !== 'labels' && Array.isArray((h as any)[key])) {
                        (h as any)[key].shift()
                    }
                })
            }
        })

        if (!this.memChart || !this.goroutineChart || !this.sysChart || !this.heapChart) {
            this.initCharts()
        } else {
            this.updateCharts()
        }
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
                        { label: '已分配', data: [...this.memHistory.alloc], borderColor: 'rgba(59,130,246,0.6)', backgroundColor: 'rgba(59,130,246,0.08)', fill: true },
                        { label: '堆已分配', data: [...this.memHistory.heapAlloc], borderColor: 'rgba(16,185,129,0.6)', backgroundColor: 'rgba(16,185,129,0.08)', fill: true },
                        { label: '堆使用中', data: [...this.memHistory.heapInuse], borderColor: 'rgba(245,158,11,0.6)', backgroundColor: 'rgba(245,158,11,0.08)', fill: true },
                        { label: '堆已申请', data: [...this.memHistory.heapSys], borderColor: 'rgba(139,92,246,0.6)', backgroundColor: 'rgba(139,92,246,0.08)', fill: true }
                    ]
                },
                options: this.bgChartOptions('堆内存')
            }))
        }

        if (this.goroutineCanvasRef) {
            this.goroutineChart = markRaw(new Chart(this.goroutineCanvasRef, {
                type: 'line' as const,
                data: {
                    labels: [...this.goroutineHistory.labels],
                    datasets: [
                        { label: 'Goroutine', data: [...this.goroutineHistory.goroutine], borderColor: 'rgba(139,92,246,0.6)', backgroundColor: 'rgba(139,92,246,0.08)', fill: true },
                        { label: 'GC 次数', data: [...this.goroutineHistory.gc], borderColor: 'rgba(239,68,68,0.6)', backgroundColor: 'rgba(239,68,68,0.08)', fill: true }
                    ]
                },
                options: this.bgChartOptions('Goroutine & GC')
            }))
        }

        if (this.stackCanvasRef) {
            this.stackChart = markRaw(new Chart(this.stackCanvasRef, {
                type: 'line' as const,
                data: {
                    labels: [...this.stackHistory.labels],
                    datasets: [
                        { label: '栈已使用', data: [...this.stackHistory.stackInuse], borderColor: 'rgba(245,158,11,0.6)', backgroundColor: 'rgba(245,158,11,0.08)', fill: true },
                        { label: '栈已申请', data: [...this.stackHistory.stackSys], borderColor: 'rgba(139,92,246,0.6)', backgroundColor: 'rgba(139,92,246,0.08)', fill: true }
                    ]
                },
                options: this.bgChartOptions('栈内存')
            }))
        }

        if (this.sysCanvasRef) {
            this.sysChart = markRaw(new Chart(this.sysCanvasRef, {
                type: 'line' as const,
                data: {
                    labels: [...this.sysHistory.labels],
                    datasets: [
                        { label: '累计分配', data: [...this.sysHistory.totalAlloc], borderColor: 'rgba(59,130,246,0.6)', backgroundColor: 'rgba(59,130,246,0.08)', fill: true },
                        { label: '系统申请', data: [...this.sysHistory.sys], borderColor: 'rgba(16,185,129,0.6)', backgroundColor: 'rgba(16,185,129,0.08)', fill: true }
                    ]
                },
                options: this.bgChartOptions('系统内存')
            }))
        }

        if (this.heapCanvasRef) {
            this.heapChart = markRaw(new Chart(this.heapCanvasRef, {
                type: 'line' as const,
                data: {
                    labels: [...this.heapHistory.labels],
                    datasets: [
                        { label: '堆空闲', data: [...this.heapHistory.heapIdle], borderColor: 'rgba(168,162,158,0.6)', backgroundColor: 'rgba(168,162,158,0.08)', fill: true },
                        { label: '堆已释放', data: [...this.heapHistory.heapReleased], borderColor: 'rgba(244,63,94,0.6)', backgroundColor: 'rgba(244,63,94,0.08)', fill: true },
                        { label: '堆对象', data: [...this.heapHistory.heapObjects], borderColor: 'rgba(34,197,94,0.6)', backgroundColor: 'rgba(34,197,94,0.08)', fill: true, yAxisID: 'y1' }
                    ]
                },
                options: {
                    ...this.bgChartOptions('堆空闲 & 释放 & 对象'),
                    scales: {
                        ...this.bgChartOptions().scales,
                        y1: {
                            display: false,
                            position: 'right' as const,
                            beginAtZero: true,
                            grid: { display: false },
                            border: { display: false }
                        }
                    }
                }
            }))
        }
    }

    updateCharts() {
        if (this.memChart) {
            this.memChart.data.labels = [...this.memHistory.labels]
            this.memChart.data.datasets[0].data = [...this.memHistory.alloc]
            this.memChart.data.datasets[1].data = [...this.memHistory.heapAlloc]
            this.memChart.data.datasets[2].data = [...this.memHistory.heapInuse]
            this.memChart.data.datasets[3].data = [...this.memHistory.heapSys]
            this.memChart.update()
        }
        if (this.goroutineChart) {
            this.goroutineChart.data.labels = [...this.goroutineHistory.labels]
            this.goroutineChart.data.datasets[0].data = [...this.goroutineHistory.goroutine]
            this.goroutineChart.data.datasets[1].data = [...this.goroutineHistory.gc]
            this.goroutineChart.update()
        }
        if (this.stackChart) {
            this.stackChart.data.labels = [...this.stackHistory.labels]
            this.stackChart.data.datasets[0].data = [...this.stackHistory.stackInuse]
            this.stackChart.data.datasets[1].data = [...this.stackHistory.stackSys]
            this.stackChart.update()
        }
        if (this.sysChart) {
            this.sysChart.data.labels = [...this.sysHistory.labels]
            this.sysChart.data.datasets[0].data = [...this.sysHistory.totalAlloc]
            this.sysChart.data.datasets[1].data = [...this.sysHistory.sys]
            this.sysChart.update()
        }
        if (this.heapChart) {
            this.heapChart.data.labels = [...this.heapHistory.labels]
            this.heapChart.data.datasets[0].data = [...this.heapHistory.heapIdle]
            this.heapChart.data.datasets[1].data = [...this.heapHistory.heapReleased]
            this.heapChart.data.datasets[2].data = [...this.heapHistory.heapObjects]
            this.heapChart.update()
        }
    }

    clearData() {
        this.current = null
        this.lastGCTime = '从未'
        this.memHistory = { labels: [], alloc: [], heapAlloc: [], heapInuse: [], heapSys: [] }
        this.goroutineHistory = { labels: [], goroutine: [], gc: [] }
        this.stackHistory = { labels: [], stackInuse: [], stackSys: [] }
        this.sysHistory = { labels: [], totalAlloc: [], sys: [] }
        this.heapHistory = { labels: [], heapIdle: [], heapReleased: [], heapObjects: [] }
        ;[this.memChart, this.goroutineChart, this.stackChart, this.sysChart, this.heapChart].forEach(chart => {
            if (chart) {
                chart.data.labels = []
                chart.data.datasets.forEach(ds => { ds.data = [] })
                chart.update()
            }
        })
    }

    mounted() {
        this.initCharts()
    }

    unmounted() {
        this.memChart?.destroy()
        this.goroutineChart?.destroy()
        this.stackChart?.destroy()
        this.sysChart?.destroy()
        this.heapChart?.destroy()
    }
}

export default toNative(SystemGo)
</script>

<template>
  <div v-if="current" class="rounded-xl border border-slate-200 bg-white overflow-hidden">
    <div class="card-header">
      <div class="card-icon bg-sky-500">
        <i class="fas fa-code text-white text-xs"></i>
      </div>
      <span class="text-sm font-semibold text-slate-700">Go 运行态</span>
      <span class="ml-auto text-xs text-slate-400 font-mono">{{ current.version }}</span>
      <span class="text-xs text-slate-400 font-mono ml-3" title="最后 GC 时间">
        <i class="fas fa-clock mr-1"></i>{{ lastGCTime }}
      </span>
    </div>

    <!-- 堆内存折线图 -->
    <div class="relative p-4 border-b border-slate-100">
      <div class="flex items-center gap-4 mb-2">
        <span class="text-xs font-medium text-slate-500">堆内存</span>
        <div class="flex items-center gap-3 text-xs text-slate-400">
          <span class="flex items-center gap-1"><span class="w-3 h-0.5 bg-blue-500 rounded-full"></span>已分配</span>
          <span class="flex items-center gap-1"><span class="w-3 h-0.5 bg-emerald-500 rounded-full"></span>堆已分配</span>
          <span class="flex items-center gap-1"><span class="w-3 h-0.5 bg-amber-500 rounded-full"></span>堆使用中</span>
          <span class="flex items-center gap-1"><span class="w-3 h-0.5 bg-purple-500 rounded-full"></span>堆已申请</span>
        </div>
      </div>
      <div class="relative h-32">
        <canvas ref="memCanvasRef" class="w-full h-full"></canvas>
      </div>
    </div>

    <!-- Goroutine & GC 折线图 -->
    <div class="relative p-4 border-b border-slate-100">
      <div class="flex items-center gap-4 mb-2">
        <span class="text-xs font-medium text-slate-500">Goroutine & GC</span>
        <div class="flex items-center gap-3 text-xs text-slate-400">
          <span class="flex items-center gap-1"><span class="w-3 h-0.5 bg-purple-500 rounded-full"></span>Goroutine</span>
          <span class="flex items-center gap-1"><span class="w-3 h-0.5 bg-red-500 rounded-full"></span>GC 次数</span>
        </div>
      </div>
      <div class="flex items-center gap-4 mb-2">
        <span class="text-lg font-bold text-slate-800 tabular-nums">{{ current.numGoroutine }}</span>
        <span class="text-lg font-bold text-slate-800 tabular-nums">{{ current.numGC }}</span>
      </div>
      <div class="relative h-32">
        <canvas ref="goroutineCanvasRef" class="w-full h-full"></canvas>
      </div>
    </div>

    <!-- 栈内存折线图 -->
    <div class="relative p-4 border-b border-slate-100">
      <div class="flex items-center gap-4 mb-2">
        <span class="text-xs font-medium text-slate-500">栈内存</span>
        <div class="flex items-center gap-3 text-xs text-slate-400">
          <span class="flex items-center gap-1"><span class="w-3 h-0.5 bg-amber-500 rounded-full"></span>栈已使用</span>
          <span class="flex items-center gap-1"><span class="w-3 h-0.5 bg-purple-500 rounded-full"></span>栈已申请</span>
        </div>
      </div>
      <div class="relative h-32">
        <canvas ref="stackCanvasRef" class="w-full h-full"></canvas>
      </div>
    </div>

    <!-- 系统内存折线图 -->
    <div class="relative p-4 border-b border-slate-100">
      <div class="flex items-center gap-4 mb-2">
        <span class="text-xs font-medium text-slate-500">系统内存</span>
        <div class="flex items-center gap-3 text-xs text-slate-400">
          <span class="flex items-center gap-1"><span class="w-3 h-0.5 bg-blue-500 rounded-full"></span>累计分配</span>
          <span class="flex items-center gap-1"><span class="w-3 h-0.5 bg-emerald-500 rounded-full"></span>系统申请</span>
        </div>
      </div>
      <div class="relative h-32">
        <canvas ref="sysCanvasRef" class="w-full h-full"></canvas>
      </div>
    </div>

    <!-- 堆空闲 & 释放 & 对象折线图 -->
    <div class="relative p-4">
      <div class="flex items-center gap-4 mb-2">
        <span class="text-xs font-medium text-slate-500">堆空闲 & 释放 & 对象</span>
        <div class="flex items-center gap-3 text-xs text-slate-400">
          <span class="flex items-center gap-1"><span class="w-3 h-0.5 bg-stone-400 rounded-full"></span>堆空闲</span>
          <span class="flex items-center gap-1"><span class="w-3 h-0.5 bg-rose-500 rounded-full"></span>堆已释放</span>
          <span class="flex items-center gap-1"><span class="w-3 h-0.5 bg-green-500 rounded-full"></span>堆对象</span>
        </div>
      </div>
      <div class="flex items-center gap-4 mb-2">
        <span class="text-lg font-bold text-slate-800 tabular-nums">{{ fmtSize(current.heapIdle) }}</span>
        <span class="text-lg font-bold text-slate-800 tabular-nums">{{ fmtSize(current.heapReleased) }}</span>
        <span class="text-lg font-bold text-slate-800 tabular-nums">{{ current.heapObjects }}</span>
      </div>
      <div class="relative h-32">
        <canvas ref="heapCanvasRef" class="w-full h-full"></canvas>
      </div>
    </div>
  </div>
</template>
