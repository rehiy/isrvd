<script lang="ts">
import type { ChartOptions } from 'chart.js'
import { markRaw } from 'vue'
import { Component, Prop, Ref, Vue, toNative } from 'vue-facing-decorator'

import { useLocaleStore } from '@/stores/locale'

import type { SystemGoRuntimeStat, SystemStat } from '@/service/types'

import Chart, { makeLineDataset, makeMonitorChartOptions, makeMonitorScale } from '@/helper/chart'
import { POLL_INTERVAL } from '@/helper/format'
import { appendMonitorPoint, formatMonitorBytes } from '@/helper/monitor'

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
    localeStore = useLocaleStore()
    now = Date.now()
    private timeTimer: ReturnType<typeof setInterval> | null = null
    formatMonitorBytes = formatMonitorBytes

    get lastGCDate() {
        const ts = this.current?.lastGC
        if (typeof ts !== 'number' || !Number.isFinite(ts) || ts <= 0) {
            return null
        }
        const date = new Date(ts * 1000)
        return Number.isFinite(date.getTime()) ? date : null
    }

    get lastGCTime() {
        return this.lastGCDate?.toLocaleString(this.localeStore.meta.tag, {
            year: 'numeric', month: '2-digit', day: '2-digit',
            hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false
        }) ?? (this.current?.lastGC === 0 ? this.$t('从未') : '—')
    }

    get lastGCRelativeTime() {
        const date = this.lastGCDate
        if (!date) {
            return this.lastGCTime
        }
        const seconds = Math.floor((this.now - date.getTime()) / 1000)
        if (seconds < 0) {
            return this.lastGCTime
        }
        const formatter = new Intl.RelativeTimeFormat(this.localeStore.meta.tag, { numeric: 'auto' })
        const units: [Intl.RelativeTimeFormatUnit, number][] = [
            ['year', 31536000], ['month', 2592000], ['day', 86400],
            ['hour', 3600], ['minute', 60], ['second', 1]
        ]
        const [unit, duration] = units.find(([, duration]) => seconds >= duration) ?? ['second', 1]
        return formatter.format(-Math.floor(seconds / duration), unit)
    }

    goChartOptions(): ChartOptions<'line'> {
        return makeMonitorChartOptions(
            { y: makeMonitorScale('bytes') },
            ctx => `${ctx.dataset.label || ''}: ${formatMonitorBytes(ctx.parsed.y ?? 0)}`
        )
    }

    pushData(payload: SystemStat, ts: number) {
        this.current = payload.go
        this.now = Date.now()

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
        if (this.memChart) {
            this.memChart.destroy()
            this.memChart = null
        }
        if (this.goroutineChart) {
            this.goroutineChart.destroy()
            this.goroutineChart = null
        }
        if (this.stackChart) {
            this.stackChart.destroy()
            this.stackChart = null
        }
        if (this.sysChart) {
            this.sysChart.destroy()
            this.sysChart = null
        }

        if (this.memCanvasRef) {
            this.memChart = markRaw(new Chart(this.memCanvasRef, {
                type: 'line' as const,
                data: {
                    labels: [...this.memHistory.labels],
                    datasets: [
                        makeLineDataset(this.memHistory.alloc, '#3b82f6', this.$t('已分配')),
                        makeLineDataset(this.memHistory.heapAlloc, '#10b981', this.$t('堆已分配')),
                        makeLineDataset(this.memHistory.heapInuse, '#f59e0b', this.$t('堆使用中')),
                        makeLineDataset(this.memHistory.heapIdle, '#a8a29e', this.$t('堆空闲')),
                        makeLineDataset(this.memHistory.heapReleased, '#f43f5e', this.$t('堆已释放')),
                        makeLineDataset(this.memHistory.heapSys, '#8b5cf6', this.$t('堆已申请'))
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
                        makeLineDataset(this.goroutineHistory.goroutine, '#8b5cf6', 'Goroutine'),
                        makeLineDataset(this.goroutineHistory.heapObjects, '#22c55e', this.$t('堆对象'), { yAxisID: 'y1' }),
                        makeLineDataset(this.goroutineHistory.gc, '#ef4444', this.$t('GC 次数'), { yAxisID: 'y2' })
                    ]
                },
                options: makeMonitorChartOptions(
                    {
                        y: makeMonitorScale('count', { position: 'left' }),
                        y1: makeMonitorScale('count', { display: false, position: 'right', grid: { display: false } }),
                        y2: makeMonitorScale('count', { display: false, position: 'right', grid: { display: false } })
                    },
                    ctx => `${ctx.dataset.label || ''}: ${ctx.parsed.y}`
                )
            }))
        }

        if (this.stackCanvasRef) {
            this.stackChart = markRaw(new Chart(this.stackCanvasRef, {
                type: 'line' as const,
                data: {
                    labels: [...this.stackHistory.labels],
                    datasets: [
                        makeLineDataset(this.stackHistory.stackInuse, '#f59e0b', this.$t('栈已使用')),
                        makeLineDataset(this.stackHistory.stackSys, '#8b5cf6', this.$t('栈已申请'))
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
                        makeLineDataset(this.sysHistory.totalAlloc, '#3b82f6', this.$t('累计分配')),
                        makeLineDataset(this.sysHistory.sys, '#10b981', this.$t('系统申请'))
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
            if (!this.current || !this.memCanvasRef?.isConnected || !this.goroutineCanvasRef?.isConnected || !this.stackCanvasRef?.isConnected || !this.sysCanvasRef?.isConnected) {
                return
            }
            if (!this.memChart || !this.goroutineChart || !this.stackChart || !this.sysChart || this.memChart.canvas !== this.memCanvasRef || this.goroutineChart.canvas !== this.goroutineCanvasRef || this.stackChart.canvas !== this.stackCanvasRef || this.sysChart.canvas !== this.sysCanvasRef) {
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

    mounted() {
        this.timeTimer = setInterval(() => {
            this.now = Date.now()
        }, POLL_INTERVAL)
    }

    unmounted() {
        if (this.timeTimer !== null) {
            clearInterval(this.timeTimer)
            this.timeTimer = null
        }
        this.clearData()
    }
}

export default toNative(SystemGo)
</script>

<template>
  <div class="space-y-3">
    <p v-if="!current" class="text-right text-xs text-slate-400">{{ $t('加载中...') }}</p>
    <div v-if="current" class="grid grid-cols-1 gap-4">
      <!-- 系统内存折线图 -->
      <div class="metric-card min-w-0">
        <div class="absolute inset-x-0 top-0 h-0.5 bg-gradient-to-r from-sky-400 to-transparent"></div>
        <div class="monitor-card-header">
          <div class="monitor-card-heading">
            <div class="monitor-card-icon">
              <i class="fas fa-memory text-sm"></i>
            </div>
            <span class="min-w-0 flex-1 monitor-card-title">Go - {{ $t('系统内存') }}</span>
          </div>
          <div class="monitor-card-metrics">
            <span class="inline-flex flex-wrap items-baseline justify-end gap-1.5" :title="$t('Go 运行时占用估算：系统申请减去堆已释放，不等于进程 RSS')">{{ $t('实际占用') }} <span class="monitor-card-value text-slate-700">{{ formatMonitorBytes(Math.max(0, current.sys - current.heapReleased)) }}</span></span>
            <div class="monitor-metrics-row">
              <span class="monitor-series"><span class="monitor-series-dot bg-emerald-500"></span>{{ $t('系统申请') }} <span class="tabular-nums">{{ formatMonitorBytes(current.sys) }}</span></span>
              <span class="monitor-series"><span class="monitor-series-dot bg-blue-500"></span>{{ $t('累计分配') }} <span class="tabular-nums">{{ formatMonitorBytes(current.totalAlloc) }}</span></span>
            </div>
          </div>
        </div>
        <div class="monitor-chart-box">
          <canvas ref="sysCanvasRef" class="w-full h-full"></canvas>
        </div>
      </div>

      <!-- 堆内存折线图 -->
      <div class="metric-card min-w-0">
        <div class="absolute inset-x-0 top-0 h-0.5 bg-gradient-to-r from-emerald-400 to-transparent"></div>
        <div class="monitor-card-header">
          <div class="monitor-card-heading">
            <div class="monitor-card-icon">
              <i class="fas fa-cubes text-sm"></i>
            </div>
            <span class="min-w-0 flex-1 monitor-card-title">Go - {{ $t('堆内存') }}</span>
          </div>
          <div class="monitor-card-metrics">
            <span class="inline-flex flex-wrap items-baseline justify-end gap-1.5" :title="$t('使用中的堆跨度，包含跨度内未使用的空间，不等于堆对象分配量')">{{ $t('实际占用') }} <span class="monitor-card-value text-slate-700">{{ formatMonitorBytes(current.heapInuse) }}</span></span>
            <div class="monitor-metrics-row">
              <span class="monitor-series"><span class="monitor-series-dot bg-violet-500"></span>{{ $t('堆已申请') }} <span class="tabular-nums">{{ formatMonitorBytes(current.heapSys) }}</span></span>
              <span class="monitor-series"><span class="monitor-series-dot bg-blue-500"></span>{{ $t('已分配') }} <span class="tabular-nums">{{ formatMonitorBytes(current.alloc) }}</span></span>
              <span class="monitor-series"><span class="monitor-series-dot bg-emerald-500"></span>{{ $t('堆已分配') }} <span class="tabular-nums">{{ formatMonitorBytes(current.heapAlloc) }}</span></span>
              <span class="monitor-series"><span class="monitor-series-dot bg-amber-500"></span>{{ $t('堆使用中') }} <span class="tabular-nums">{{ formatMonitorBytes(current.heapInuse) }}</span></span>
              <span class="monitor-series"><span class="monitor-series-dot bg-stone-400"></span>{{ $t('堆空闲') }} <span class="tabular-nums">{{ formatMonitorBytes(current.heapIdle) }}</span></span>
              <span class="monitor-series"><span class="monitor-series-dot bg-rose-500"></span>{{ $t('堆已释放') }} <span class="tabular-nums">{{ formatMonitorBytes(current.heapReleased) }}</span></span>
            </div>
          </div>
        </div>
        <div class="monitor-chart-box">
          <canvas ref="memCanvasRef" class="w-full h-full"></canvas>
        </div>
      </div>

      <!-- 栈内存折线图 -->
      <div class="metric-card min-w-0">
        <div class="absolute inset-x-0 top-0 h-0.5 bg-gradient-to-r from-amber-400 to-transparent"></div>
        <div class="monitor-card-header">
          <div class="monitor-card-heading">
            <div class="monitor-card-icon">
              <i class="fas fa-layer-group text-sm"></i>
            </div>
            <span class="min-w-0 flex-1 monitor-card-title">Go - {{ $t('栈内存') }}</span>
          </div>
          <div class="monitor-card-metrics">
            <span class="inline-flex flex-wrap items-baseline justify-end gap-1.5" :title="$t('使用中的栈跨度，不包含未使用的栈申请空间')">{{ $t('实际占用') }} <span class="monitor-card-value text-slate-700">{{ formatMonitorBytes(current.stackInuse) }}</span></span>
            <div class="monitor-metrics-row">
              <span class="monitor-series"><span class="monitor-series-dot bg-violet-500"></span>{{ $t('栈已申请') }} <span class="tabular-nums">{{ formatMonitorBytes(current.stackSys) }}</span></span>
              <span class="monitor-series"><span class="monitor-series-dot bg-amber-500"></span>{{ $t('栈已使用') }} <span class="tabular-nums">{{ formatMonitorBytes(current.stackInuse) }}</span></span>
            </div>
          </div>
        </div>
        <div class="monitor-chart-box">
          <canvas ref="stackCanvasRef" class="w-full h-full"></canvas>
        </div>
      </div>

      <!-- Goroutine & GC & 堆对象折线图 -->
      <div class="metric-card min-w-0">
        <div class="absolute inset-x-0 top-0 h-0.5 bg-gradient-to-r from-violet-400 to-transparent"></div>
        <div class="monitor-card-header">
          <div class="monitor-card-heading">
            <div class="monitor-card-icon">
              <i class="fas fa-hashtag text-sm"></i>
            </div>
            <span class="min-w-0 flex-1 monitor-card-title">Go - {{ $t('计数器') }}</span>
          </div>
          <div class="monitor-card-metrics">
            <span class="monitor-series" :title="lastGCTime">{{ $t('最后 GC 时间') }} <span class="monitor-card-value text-slate-700">{{ lastGCRelativeTime }}</span></span>
            <div class="monitor-metrics-row">
              <span class="monitor-series"><span class="monitor-series-dot bg-violet-500"></span>Goroutine <span class="tabular-nums">{{ current.numGoroutine }}</span></span>
              <span class="monitor-series"><span class="monitor-series-dot bg-green-500"></span>{{ $t('堆对象') }} <span class="tabular-nums">{{ current.heapObjects }}</span></span>
              <span class="monitor-series"><span class="monitor-series-dot bg-red-500"></span>{{ $t('GC 次数') }} <span class="tabular-nums">{{ current.numGC }}</span></span>
            </div>
          </div>
        </div>
        <div class="monitor-chart-box">
          <canvas ref="goroutineCanvasRef" class="w-full h-full"></canvas>
        </div>
      </div>
    </div>
  </div>
</template>
