<script lang="ts">
import type { ChartOptions } from 'chart.js'
import { Chart, registerables } from 'chart.js'
import { markRaw, nextTick } from 'vue'
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { SystemStat, SystemNetInterface, SystemDiskIO, SystemGPU } from '@/service/types'
import { hexToRgba, POLL_INTERVAL } from '@/helper/utils'

Chart.register(...registerables)

interface TimeSeriesHistory {
    labels: string[]
    recv: number[]
    sent: number[]
    [key: string]: string[] | number[]
}

interface DiskIOSeriesHistory {
    labels: string[]
    read: number[]
    write: number[]
    [key: string]: string[] | number[]
}

interface IOSnapshot {
    recv?: number
    sent?: number
    read?: number
    write?: number
}

interface ChartCallbackContext {
    parsed: { y: number | null }
    dataset: { label?: string }
}

@Component
class SystemOverview extends Vue {
    // ─── Refs ───
    @Ref readonly netContainerRef!: HTMLDivElement
    @Ref readonly diskIOContainerRef!: HTMLDivElement
    @Ref readonly cpuCanvasRef!: HTMLCanvasElement
    @Ref readonly memCanvasRef!: HTMLCanvasElement

    // ─── 数据属性 ───
    stat: SystemStat | null = null
    loading = false

    // ─── 私有属性（非响应式） ───
    private readonly NET_POINTS = 45
    private readonly MAX_STAT_POINTS = 45
    netHistory: Record<string, TimeSeriesHistory> = markRaw({})
    private lastNetSnapshot: Record<string, IOSnapshot> = markRaw({})
    private pollTimer: ReturnType<typeof setInterval> | null = null
    private netCharts: Record<string, Chart<'line'>> = markRaw({})
    diskIOHistory: Record<string, DiskIOSeriesHistory> = markRaw({})
    private lastDiskIOSnapshot: Record<string, IOSnapshot> = markRaw({})
    private diskIOCharts: Record<string, Chart<'line'>> = markRaw({})
    private cpuHistory = markRaw({ labels: [] as string[], data: [] as number[] })
    private memHistory = markRaw({ labels: [] as string[], data: [] as number[] })
    private cpuChart: Chart<'line'> | null = null
    private memChart: Chart<'line'> | null = null
    // GPU
    @Ref readonly gpuContainerRef!: HTMLDivElement
    private gpuHistories: Record<number, { labels: string[]; data: number[] }> = markRaw({})
    private gpuCharts: Record<number, Chart<'line'>> = markRaw({})

    // ─── 计算属性 ───
    get cpuVal() {
        return this.stat ? this.cpuPercent(this.stat.system.CpuPercent) : 0
    }

    get memVal() {
        return this.stat ? this.memPercent(this.stat.system.MemoryUsed, this.stat.system.MemoryTotal) : 0
    }

    // ─── 工具方法 ───
    fmtSize(bytes: number, rates = false) {
        if (!bytes || bytes < 0) return rates ? '0 B/s' : '0 B'
        const units = rates ? ['B/s', 'KB/s', 'MB/s', 'GB/s'] : ['B', 'KB', 'MB', 'GB', 'TB']
        let i = 0, v = bytes
        while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
        return `${v.toFixed(1)} ${units[i]}`
    }

    fmtBytes(b: number) { return this.fmtSize(b, false) }
    fmtRate(b: number) { return this.fmtSize(b, true) }

    timeLabel() {
        const now = new Date()
        return [now.getHours(), now.getMinutes(), now.getSeconds()]
            .map(n => n.toString().padStart(2, '0')).join(':')
    }

    fmtUptime(seconds: number) {
        if (!seconds) return '0s'
        const d = Math.floor(seconds / 86400)
        const h = Math.floor((seconds % 86400) / 3600)
        const m = Math.floor((seconds % 3600) / 60)
        const parts: string[] = []
        if (d) parts.push(`${d}d`)
        if (h) parts.push(`${h}h`)
        if (m) parts.push(`${m}m`)
        if (!parts.length) parts.push(`${seconds % 60}s`)
        return parts.join(' ')
    }

    cpuPercent(arr: number[]): number {
        if (!arr || !arr.length) return 0
        return +((arr.reduce((a, b) => a + b, 0) / arr.length).toFixed(1))
    }

    memPercent(used: number, total: number): number {
        if (!total) return 0
        return +((used / total) * 100).toFixed(1)
    }

    physicalInterfaces(list: SystemNetInterface[]) {
        if (!list) return []
        const virtualPrefixes = ['lo', 'docker', 'veth', 'br-', 'overlay', 'flannel', 'cni', 'tunl', 'dummy', 'virbr']
        return list.filter(ni => !virtualPrefixes.some(p => ni.Name.startsWith(p)))
    }

    semanticColor(pct: number, prefix = 'bg') {
        const p = pct
        if (p >= 90) return `${prefix}-red-500`
        if (p >= 70) return `${prefix}-amber-500`
        return `${prefix}-emerald-500`
    }

    barColor(pct: number) { return this.semanticColor(pct, 'bg') }
    textColor(pct: number) { return this.semanticColor(pct, 'text') }

    fmtGCTime(ts: number) {
        if (!ts) return '从未'
        return new Date(ts * 1000).toLocaleString('zh-CN')
    }

    currentIORate(history: Record<string, TimeSeriesHistory | DiskIOSeriesHistory>, name: string, dir: string): number {
        const h = history[name]
        if (!h || !h[dir] || !(h[dir] as number[]).length) return 0
        const arr = h[dir] as number[]
        return arr[arr.length - 1]
    }

    currentRate(name: string, dir: string): number {
        return this.currentIORate(this.netHistory, name, dir)
    }

    currentDiskRate(name: string, dir: string): number {
        return this.currentIORate(this.diskIOHistory, name, dir)
    }

    devShortName(device: string): string { return device.split('/').pop() || device }

    diskIOByDevice(device: string) {
        if (!this.stat?.diskIO) return null
        const devName = device.split('/').pop()
        return this.stat.diskIO.find(d => d.Name === devName)
            || this.stat.diskIO.find(d => devName!.startsWith(d.Name))            || null
    }

    // ─── 图表配置 ───
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

    netChartOptions(): ChartOptions<'line'> {
        const fmtRate = (v: number) => this.fmtRate(v)
        return {
            responsive: true, maintainAspectRatio: false, animation: false,
            interaction: { intersect: false, mode: 'index' as const },
            plugins: {
                legend: { display: true, position: 'bottom' as const, labels: { boxWidth: 8, padding: 8, font: { size: 10 }, color: '#64748b' } },
                tooltip: {
                    backgroundColor: 'rgba(15,23,42,0.9)', titleFont: { size: 10 }, bodyFont: { size: 10 }, padding: 8, cornerRadius: 6,
                    callbacks: { label: (ctx: ChartCallbackContext) => (ctx.dataset.label ?? '') + ': ' + fmtRate(ctx.parsed.y ?? 0) }
                }
            },
            scales: {
                x: { display: false },
                y: {
                    display: true, beginAtZero: true, grid: { color: 'rgba(148,163,184,0.08)' }, border: { display: false },
                    ticks: { font: { size: 9 }, color: '#94a3b8', maxTicksLimit: 4, padding: 4, callback: (v: string | number) => fmtRate(Number(v)) }
                }
            },
            elements: { point: { radius: 0, hoverRadius: 3 }, line: { tension: 0.4, borderWidth: 1.5 } }
        }
    }

    makeDataset(data: number[], color: string, label: string) {
        return { label, data: [...data], borderColor: color, backgroundColor: hexToRgba(color, 0.1), fill: true }
    }

    // ─── CPU/内存图表 ───
    makeStatChart(canvas: HTMLCanvasElement, history: { labels: string[]; data: number[] }, borderColor: string, bgColor: string): Chart<'line'> | null {
        if (!canvas) return null
        return markRaw(new Chart(canvas, {
            type: 'line' as const,
            data: { labels: [...history.labels], datasets: [{ data: [...history.data], borderColor, backgroundColor: bgColor, fill: true }] },
            options: this.bgChartOptions()
        }))
    }

    initStatCharts() {
        if (this.cpuChart) { this.cpuChart.destroy(); this.cpuChart = null }
        if (this.memChart) { this.memChart.destroy(); this.memChart = null }
        this.cpuChart = this.makeStatChart(this.cpuCanvasRef, this.cpuHistory, 'rgba(59,130,246,0.6)', 'rgba(59,130,246,0.08)')
        this.memChart = this.makeStatChart(this.memCanvasRef, this.memHistory, 'rgba(99,102,241,0.6)', 'rgba(99,102,241,0.08)')
    }

    pushStatPoint(cpuV: number, memV: number) {
        const label = this.timeLabel()
        this.cpuHistory.labels.push(label)
        this.cpuHistory.data.push(cpuV)
        this.memHistory.labels.push(label)
        this.memHistory.data.push(memV)
        if (this.cpuHistory.labels.length > this.MAX_STAT_POINTS) {
            this.cpuHistory.labels.shift(); this.cpuHistory.data.shift()
            this.memHistory.labels.shift(); this.memHistory.data.shift()
        }
    }

    updateStatCharts() {
        for (const [chart, history] of [[this.cpuChart, this.cpuHistory], [this.memChart, this.memHistory]] as [Chart<'line'> | null, { labels: string[]; data: number[] }][]) {
            if (!chart) continue
            chart.data.labels = [...history.labels]
            chart.data.datasets[0].data = [...history.data]
            chart.update('none')
        }
    }

    destroyStatCharts() {
        this.cpuChart?.destroy(); this.cpuChart = null
        this.memChart?.destroy(); this.memChart = null
    }

    // ─── 网络图表 ───
    getNetCanvas(name: string) {
        return this.netContainerRef?.querySelector(`[data-iface="${name}"]`) ?? null
    }

    initNetChart(name: string) {
        const canvas = this.getNetCanvas(name)
        if (!canvas) return
        this.netCharts[name]?.destroy()
        const h = this.netHistory[name] || { labels: [], recv: [], sent: [] }
        this.netCharts[name] = markRaw(new Chart(canvas as HTMLCanvasElement, {
            type: 'line' as const,
            data: { labels: [...h.labels], datasets: [this.makeDataset(h.recv, '#10b981', '下行'), this.makeDataset(h.sent, '#3b82f6', '上行')] },
            options: this.netChartOptions()
        }))
    }

    initAllNetCharts() {
        if (!this.stat) return
        this.physicalInterfaces(this.stat.system.NetInterface).forEach(ni => {
            if (!this.netHistory[ni.Name]) this.netHistory[ni.Name] = { labels: [], recv: [], sent: [] }
            this.initNetChart(ni.Name)
        })
    }

    updateNetChart(name: string) {
        const chart = this.netCharts[name]
        const h = this.netHistory[name]
        if (!chart || !h) return
        chart.data.labels = [...h.labels]
        chart.data.datasets[0].data = [...h.recv]
        chart.data.datasets[1].data = [...h.sent]
        chart.update('none')
    }

    destroyNetCharts() {
        Object.values(this.netCharts).forEach(c => c.destroy())
        this.netCharts = {}
    }

    // ─── 硬盘 IO 图表 ───
    getDiskCanvas(name: string) {
        return this.diskIOContainerRef?.querySelector(`[data-disk="${name}"]`) ?? null
    }

    initDiskChart(name: string) {
        const canvas = this.getDiskCanvas(name)
        if (!canvas) return
        this.diskIOCharts[name]?.destroy()
        const h = this.diskIOHistory[name] || { labels: [], read: [], write: [] }
        this.diskIOCharts[name] = markRaw(new Chart(canvas as HTMLCanvasElement, {
            type: 'line' as const,
            data: { labels: [...h.labels], datasets: [this.makeDataset(h.read, '#f59e0b', '读取'), this.makeDataset(h.write, '#8b5cf6', '写入')] },
            options: this.netChartOptions()
        }))
    }

    initAllDiskCharts() {
        if (!this.stat?.system?.DiskPartition) return
        this.stat.system.DiskPartition.forEach(dp => {
            const devName = dp.Device.split('/').pop() || dp.Device
            if (!this.diskIOHistory[devName]) this.diskIOHistory[devName] = { labels: [], read: [], write: [] }
            this.initDiskChart(devName)
        })
    }

    updateDiskChart(name: string) {
        const chart = this.diskIOCharts[name]
        const h = this.diskIOHistory[name]
        if (!chart || !h) return
        chart.data.labels = [...h.labels]
        chart.data.datasets[0].data = [...h.read]
        chart.data.datasets[1].data = [...h.write]
        chart.update('none')
    }

    destroyDiskCharts() {
        Object.values(this.diskIOCharts).forEach(c => c.destroy())
        this.diskIOCharts = {}
    }

    updateDiskIOHistory(diskList: SystemDiskIO[], intervalSec: number) {
        const snapshot: Record<string, IOSnapshot> = {}
        diskList.forEach(d => { snapshot[d.Name] = { read: d.ReadBytes, write: d.WriteBytes } })
        if (Object.keys(this.lastDiskIOSnapshot).length > 0) {
            const label = this.timeLabel()
            diskList.forEach(d => {
                const prev = this.lastDiskIOSnapshot[d.Name]
                if (!prev) return
                if (!this.diskIOHistory[d.Name]) this.diskIOHistory[d.Name] = { labels: [], read: [], write: [] }
                const h = this.diskIOHistory[d.Name]
                h.labels.push(label)
                h.read.push(+Math.max(0, (d.ReadBytes - (prev.read ?? 0)) / intervalSec).toFixed(0))
                h.write.push(+Math.max(0, (d.WriteBytes - (prev.write ?? 0)) / intervalSec).toFixed(0))
                if (h.labels.length > this.NET_POINTS) { h.labels.shift(); h.read.shift(); h.write.shift() }
                this.updateDiskChart(d.Name)
            })
        }
        this.lastDiskIOSnapshot = snapshot
    }

    updateNetHistory(interfaces: SystemNetInterface[], intervalSec: number) {
        const snapshot: Record<string, IOSnapshot> = {}
        interfaces.forEach(ni => { snapshot[ni.Name] = { recv: ni.BytesRecv, sent: ni.BytesSent } })
        if (Object.keys(this.lastNetSnapshot).length > 0) {
            const label = this.timeLabel()
            interfaces.forEach(ni => {
                const prev = this.lastNetSnapshot[ni.Name]
                if (!prev) return
                if (!this.netHistory[ni.Name]) this.netHistory[ni.Name] = { labels: [], recv: [], sent: [] }
                const h = this.netHistory[ni.Name]
                h.labels.push(label)
                h.recv.push(+Math.max(0, (ni.BytesRecv - (prev.recv ?? 0)) / intervalSec).toFixed(0))
                h.sent.push(+Math.max(0, (ni.BytesSent - (prev.sent ?? 0)) / intervalSec).toFixed(0))
                if (h.labels.length > this.NET_POINTS) { h.labels.shift(); h.recv.shift(); h.sent.shift() }
                this.updateNetChart(ni.Name)
            })
        }
        this.lastNetSnapshot = snapshot
    }

    // ─── GPU 图表 ───
    gpuVendorColor(vendor: string): { border: string; bg: string; iconBg: string } {
        switch (vendor) {
            case 'nvidia': return { border: 'rgba(118,185,0,0.6)', bg: 'rgba(118,185,0,0.08)', iconBg: 'bg-[#76b900]' }
            case 'amd': return { border: 'rgba(237,28,36,0.6)', bg: 'rgba(237,28,36,0.08)', iconBg: 'bg-[#ed1c24]' }
            case 'intel': return { border: 'rgba(0,104,181,0.6)', bg: 'rgba(0,104,181,0.08)', iconBg: 'bg-[#0068b5]' }
            default: return { border: 'rgba(100,116,139,0.6)', bg: 'rgba(100,116,139,0.08)', iconBg: 'bg-slate-500' }
        }
    }

    gpuTempColor(temp: number): string {
        if (temp < 0) return 'text-slate-400'
        if (temp >= 80) return 'text-red-500'
        if (temp >= 60) return 'text-amber-500'
        return 'text-emerald-500'
    }

    getGpuCanvas(index: number): HTMLCanvasElement | null {
        return this.gpuContainerRef?.querySelector(`[data-gpu="${index}"]`) ?? null
    }

    initGpuChart(gpu: SystemGPU) {
        const canvas = this.getGpuCanvas(gpu.index)
        if (!canvas) return
        this.gpuCharts[gpu.index]?.destroy()
        const h = this.gpuHistories[gpu.index] || { labels: [], data: [] }
        const color = this.gpuVendorColor(gpu.vendor)
        this.gpuCharts[gpu.index] = markRaw(new Chart(canvas, {
            type: 'line' as const,
            data: { labels: [...h.labels], datasets: [{ data: [...h.data], borderColor: color.border, backgroundColor: color.bg, fill: true }] },
            options: this.bgChartOptions()
        }))
    }

    initAllGpuCharts() {
        if (!this.stat?.gpu?.length) return
        this.stat.gpu.forEach(gpu => {
            if (!this.gpuHistories[gpu.index]) this.gpuHistories[gpu.index] = { labels: [], data: [] }
            this.initGpuChart(gpu)
        })
    }

    updateGpuChart(index: number) {
        const chart = this.gpuCharts[index]
        const h = this.gpuHistories[index]
        if (!chart || !h) return
        chart.data.labels = [...h.labels]
        chart.data.datasets[0].data = [...h.data]
        chart.update('none')
    }

    pushGpuPoints(gpus: SystemGPU[]) {
        const label = this.timeLabel()
        gpus.forEach(gpu => {
            if (!this.gpuHistories[gpu.index]) this.gpuHistories[gpu.index] = { labels: [], data: [] }
            const h = this.gpuHistories[gpu.index]
            h.labels.push(label)
            h.data.push(gpu.utilization)
            if (h.labels.length > this.MAX_STAT_POINTS) { h.labels.shift(); h.data.shift() }
            this.updateGpuChart(gpu.index)
        })
    }

    destroyGpuCharts() {
        Object.values(this.gpuCharts).forEach(c => c.destroy())
        this.gpuCharts = {}
    }

    clearHistory() {
        this.cpuHistory.labels.length = 0
        this.cpuHistory.data.length = 0
        this.memHistory.labels.length = 0
        this.memHistory.data.length = 0
    }

    // ─── 数据加载 & 轮询 ───
    async loadData() {
        this.loading = true
        this.destroyNetCharts()
        this.destroyStatCharts()
        this.destroyDiskCharts()
        this.destroyGpuCharts()
        this.netHistory = markRaw({})
        this.lastNetSnapshot = markRaw({})
        this.diskIOHistory = markRaw({})
        this.lastDiskIOSnapshot = markRaw({})
        this.gpuHistories = markRaw({})
        this.clearHistory()
        try {
            const res = await api.systemStat()
            const payload = res.payload as SystemStat | undefined
            this.stat = payload ?? null
            if (payload) {
                // 推入第一个数据点，图表初始化时即有数据
                this.pushStatPoint(
                    this.cpuPercent(payload.system.CpuPercent),
                    this.memPercent(payload.system.MemoryUsed, payload.system.MemoryTotal)
                )
                // 初始化网络/磁盘快照，并推入速率为 0 的初始点，图表初始化时即有数据
                const initLabel = this.timeLabel()
                this.physicalInterfaces(payload.system.NetInterface).forEach(ni => {
                    this.lastNetSnapshot[ni.Name] = { recv: ni.BytesRecv, sent: ni.BytesSent }
                    this.netHistory[ni.Name] = { labels: [initLabel], recv: [0], sent: [0] }
                })
                if (payload.diskIO?.length) {
                    payload.diskIO.forEach(d => {
                        this.lastDiskIOSnapshot[d.Name] = { read: d.ReadBytes, write: d.WriteBytes }
                        this.diskIOHistory[d.Name] = { labels: [initLabel], read: [0], write: [0] }
                    })
                }
                if (payload.gpu?.length) {
                    payload.gpu.forEach(gpu => {
                        this.gpuHistories[gpu.index] = { labels: [initLabel], data: [gpu.utilization] }
                    })
                }
            }
        } catch (e) {
            this.stat = null
        }
        this.loading = false
        await nextTick()
        this.initAllNetCharts()
        this.initAllDiskCharts()
        this.initStatCharts()
        this.initAllGpuCharts()
    }

    async pollNet() {
        try {
            const res = await api.systemStat()
            const payload = res.payload as SystemStat | undefined
            if (!payload || !this.stat) return
    this.stat.system.NetInterface = payload.system.NetInterface
            this.stat.system.CpuPercent = payload.system.CpuPercent
            this.stat.system.MemoryUsed = payload.system.MemoryUsed
            this.stat.system.MemoryTotal = payload.system.MemoryTotal
            this.pushStatPoint(
                this.cpuPercent(payload.system.CpuPercent),
                this.memPercent(payload.system.MemoryUsed, payload.system.MemoryTotal)
            )
            this.updateStatCharts()
            this.updateNetHistory(this.physicalInterfaces(payload.system.NetInterface), POLL_INTERVAL / 1000)
            if (payload.diskIO?.length) {
                this.stat.diskIO = payload.diskIO
                this.updateDiskIOHistory(payload.diskIO, POLL_INTERVAL / 1000)
            }
            if (payload.gpu?.length) {
                this.stat.gpu = payload.gpu
                this.pushGpuPoints(payload.gpu)
            }
        } catch (e) { /* ignore */ }
    }

    startPoll() {
        this.pollTimer = setInterval(() => this.pollNet(), POLL_INTERVAL)
    }

    stopPoll() {
        if (this.pollTimer) { clearInterval(this.pollTimer); this.pollTimer = null }
    }

    async load() {
        this.stopPoll()
        await this.loadData()
        this.startPoll()
    }

    // ─── 生命周期 ───
    mounted() {
        this.load()
    }

    unmounted() {
        this.stopPoll()
        this.destroyNetCharts()
        this.destroyDiskCharts()
        this.destroyStatCharts()
        this.destroyGpuCharts()
    }
}

export default toNative(SystemOverview)
</script>

<template>
  <div>
    <!-- 加载中 -->
    <div v-if="loading" class="flex items-center justify-center py-10">
      <div class="w-8 h-8 spinner mr-2"></div>
      <span class="text-slate-400 text-sm">加载中...</span>
    </div>

    <div v-else-if="stat" class="space-y-5">
      <!-- 主机基本信息 -->
      <div class="grid grid-cols-1 xs:grid-cols-2 md:grid-cols-4 gap-3">
        <div class="rounded-xl border border-slate-200 bg-white p-4">
          <p class="text-xs text-slate-400 mb-1">主机名</p>
          <p class="text-sm font-semibold text-slate-800 truncate">{{ stat.system.HostName }}</p>
        </div>
        <div class="rounded-xl border border-slate-200 bg-white p-4">
          <p class="text-xs text-slate-400 mb-1">操作系统</p>
          <p class="text-sm font-semibold text-slate-800 truncate">{{ stat.system.Platform }} / {{ stat.system.KernelArch }}</p>
        </div>
        <div class="rounded-xl border border-slate-200 bg-white p-4">
          <p class="text-xs text-slate-400 mb-1">运行时间</p>
          <p class="text-sm font-semibold text-slate-800">{{ fmtUptime(stat.system.Uptime) }}</p>
        </div>
        <div class="rounded-xl border border-slate-200 bg-white p-4">
          <p class="text-xs text-slate-400 mb-1">CPU 核心</p>
          <p class="text-sm font-semibold text-slate-800">{{ stat.system.CpuCore }} 物理 / {{ stat.system.CpuCoreLogic }} 逻辑</p>
        </div>
      </div>

      <!-- CPU & 内存 -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
        <!-- CPU 使用率卡片：背景折线图 + 顶部进度条 -->
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
            <p v-if="stat.system.CpuModel?.[0]" class="text-xs text-slate-400 mt-3 truncate">
              {{ stat.system.CpuModel[0] }}
            </p>
            <p v-else class="mt-3 text-xs text-slate-300">—</p>
          </div>
        </div>

        <!-- 内存使用卡片：背景折线图 + 顶部进度条 -->
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
            <p class="text-xs text-slate-400 mt-3">
              {{ fmtBytes(stat.system.MemoryUsed) }} / {{ fmtBytes(stat.system.MemoryTotal) }}
            </p>
          </div>
        </div>
      </div>

      <!-- GPU -->
      <div v-if="stat.gpu?.length" ref="gpuContainerRef" class="grid grid-cols-1 md:grid-cols-2 gap-3">
        <div
          v-for="gpu in stat.gpu"
          :key="gpu.index"
          class="relative rounded-xl border border-slate-200 bg-white overflow-hidden"
        >
          <!-- 背景使用率折线图 -->
          <div class="absolute inset-0 pointer-events-none">
            <canvas :data-gpu="gpu.index" class="w-full h-full"></canvas>
          </div>
          <!-- 前景信息 -->
          <div class="relative p-4">
            <!-- 第一行：厂商图标 + 名称 + 使用率 -->
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2 min-w-0">
                <div :class="['w-7 h-7 rounded-lg flex items-center justify-center flex-shrink-0', gpuVendorColor(gpu.vendor).iconBg]">
                  <i class="fas fa-bolt text-white text-xs"></i>
                </div>
                <span class="text-sm font-semibold text-slate-700 truncate">GPU 使用率</span>
              </div>
              <span :class="['text-2xl font-bold tabular-nums flex-shrink-0', textColor(gpu.utilization)]">
                {{ gpu.utilization.toFixed(1) }}<span class="text-sm font-medium ml-0.5">%</span>
              </span>
            </div>
            <!-- 第二行：GPU 名称 + 厂商 badge -->
            <div class="flex items-center gap-2 mt-3">
              <span class="text-xs text-slate-400 truncate">{{ gpu.name }}</span>
              <span class="text-xs text-slate-300 font-mono uppercase flex-shrink-0">{{ gpu.vendor }}</span>
            </div>
            <!-- 第三行：显存条 -->
            <div v-if="gpu.memoryTotal > 0" class="mt-3">
              <div class="flex items-center justify-between text-xs mb-1">
                <span class="text-slate-400">显存</span>
                <span class="text-slate-500 font-mono">{{ fmtBytes(gpu.memoryUsed) }} / {{ fmtBytes(gpu.memoryTotal) }}</span>
              </div>
              <div class="w-full bg-slate-100 rounded-full h-1.5">
                <div
                  :class="['h-1.5 rounded-full transition-all', barColor(memPercent(gpu.memoryUsed, gpu.memoryTotal))]"
                  :style="{ width: memPercent(gpu.memoryUsed, gpu.memoryTotal) + '%' }"
                ></div>
              </div>
            </div>
            <!-- 第四行：温度 + 功耗 + 风扇 -->
            <div class="flex items-center gap-4 mt-3 text-xs">
              <span v-if="gpu.temperature >= 0" :class="['flex items-center gap-1', gpuTempColor(gpu.temperature)]">
                <i class="fas fa-temperature-half"></i>{{ gpu.temperature }}°C
              </span>
              <span v-if="gpu.powerUsage >= 0" class="flex items-center gap-1 text-slate-500">
                <i class="fas fa-bolt"></i>{{ gpu.powerUsage.toFixed(1) }}W
              </span>
              <span v-if="gpu.fanSpeed >= 0" class="flex items-center gap-1 text-slate-500">
                <i class="fas fa-fan"></i>{{ gpu.fanSpeed }}%
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- 硬盘（容量 + IO 融合卡片） -->
      <div v-if="stat.system.DiskPartition?.length" class="rounded-xl border border-slate-200 bg-white overflow-hidden">
        <div class="px-4 py-3 border-b border-slate-100 flex items-center gap-2">
          <div class="w-6 h-6 rounded-md bg-amber-500 flex items-center justify-center">
            <i class="fas fa-hard-drive text-white text-xs"></i>
          </div>
          <span class="text-sm font-semibold text-slate-700">硬盘 I/O</span>
          <span class="ml-auto text-xs text-slate-400">
            总计 {{ fmtBytes(stat.system.DiskTotal) }}，已用 {{ fmtBytes(stat.system.DiskUsed) }}
          </span>
        </div>
        <div ref="diskIOContainerRef" class="divide-y divide-slate-50">
          <div
            v-for="dp in stat.system.DiskPartition"
            :key="dp.Mountpoint"
            class="px-4 py-3"
          >
            <!-- 第一行：挂载点 + 容量 -->
            <div class="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-3 mb-2">
              <div class="w-full sm:w-24 shrink-0">
                <p class="text-xs font-semibold text-slate-700 truncate">{{ dp.Mountpoint }}</p>
                <p class="text-xs text-slate-400">{{ dp.Device }} · {{ dp.Fstype }}</p>
              </div>
              <div class="flex-1">
                <div class="w-full bg-slate-100 rounded-full h-1.5">
                  <div
                    :class="['h-1.5 rounded-full', barColor(memPercent(dp.Used, dp.Total))]"
                    :style="{ width: memPercent(dp.Used, dp.Total) + '%' }"
                  ></div>
                </div>
              </div>
              <div class="text-right shrink-0 w-full sm:w-28">
                <p class="text-xs text-slate-600">{{ fmtBytes(dp.Used) }} / {{ fmtBytes(dp.Total) }}</p>
                <p class="text-xs text-slate-400">{{ memPercent(dp.Used, dp.Total) }}%</p>
              </div>
            </div>
            <!-- 第二行：IO 速率折线图（仅当有对应设备 IO 数据时显示） -->
            <template v-if="diskIOByDevice(dp.Device)">
              <div class="flex flex-col sm:flex-row sm:items-center justify-between mb-1.5 gap-1">
                <span class="text-xs text-slate-400">IO 速率</span>
                <div class="flex items-center gap-4 text-xs">
                  <span class="flex items-center gap-1">
                    <i class="fas fa-arrow-down text-amber-500"></i>
                    <span class="font-mono text-slate-600 w-20 text-right">{{ fmtRate(currentDiskRate(devShortName(dp.Device), 'read')) }}</span>
                  </span>
                  <span class="flex items-center gap-1">
                    <i class="fas fa-arrow-up text-violet-500"></i>
                    <span class="font-mono text-slate-600 w-20 text-right">{{ fmtRate(currentDiskRate(devShortName(dp.Device), 'write')) }}</span>
                  </span>
                </div>
              </div>
              <div class="relative h-16 bg-slate-50 rounded-lg overflow-hidden">
                <canvas :data-disk="devShortName(dp.Device)" class="w-full h-full"></canvas>
              <div v-if="!diskIOHistory[devShortName(dp.Device)]?.read?.length" class="absolute inset-0 flex items-center justify-center">
                  <span class="text-xs text-slate-300">等待数据...</span>
                </div>
              </div>
              <div class="flex flex-col sm:flex-row gap-2 sm:gap-4 mt-1.5 text-xs text-slate-400">
                <span>累计读: {{ fmtBytes(diskIOByDevice(dp.Device)?.ReadBytes ?? 0) }}</span>
                <span>累计写: {{ fmtBytes(diskIOByDevice(dp.Device)?.WriteBytes ?? 0) }}</span>
              </div>
            </template>
          </div>
        </div>
      </div>

      <!-- 网络接口 -->
      <div v-if="physicalInterfaces(stat.system.NetInterface).length" class="rounded-xl border border-slate-200 bg-white overflow-hidden">
        <div class="px-4 py-3 border-b border-slate-100 flex items-center gap-2">
          <div class="w-6 h-6 rounded-md bg-cyan-500 flex items-center justify-center">
            <i class="fas fa-network-wired text-white text-xs"></i>
          </div>
          <span class="text-sm font-semibold text-slate-700">网络接口</span>
        </div>
        <div ref="netContainerRef" class="divide-y divide-slate-50">
          <div
            v-for="ni in physicalInterfaces(stat.system.NetInterface)"
            :key="ni.Name"
            class="px-4 py-3"
          >
            <!-- 接口名 + 当前速率 -->
            <div class="flex flex-col sm:flex-row sm:items-center justify-between mb-2 gap-1">
              <p class="text-xs font-semibold text-slate-700">{{ ni.Name }}</p>
              <div class="flex items-center gap-4 text-xs">
                <span class="flex items-center gap-1">
                  <i class="fas fa-arrow-down text-emerald-500"></i>
                  <span class="font-mono text-slate-600 w-20 text-right">{{ fmtRate(currentRate(ni.Name, 'recv')) }}</span>
                </span>
                <span class="flex items-center gap-1">
                  <i class="fas fa-arrow-up text-blue-500"></i>
                  <span class="font-mono text-slate-600 w-20 text-right">{{ fmtRate(currentRate(ni.Name, 'sent')) }}</span>
                </span>
              </div>
            </div>
            <!-- 速率折线图 -->
            <div class="relative h-20 bg-slate-50 rounded-lg overflow-hidden">
              <canvas :data-iface="ni.Name" class="w-full h-full"></canvas>
              <div v-if="!netHistory[ni.Name]?.labels?.length" class="absolute inset-0 flex items-center justify-center">
                <span class="text-xs text-slate-300">等待数据...</span>
              </div>
            </div>
            <!-- 累计流量 -->
            <div class="flex flex-col sm:flex-row gap-2 sm:gap-4 mt-1.5 text-xs text-slate-400">
              <span>累计收: {{ fmtBytes(ni.BytesRecv) }}</span>
              <span>累计发: {{ fmtBytes(ni.BytesSent) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Go 运行态 -->
      <div class="rounded-xl border border-slate-200 bg-white overflow-hidden">
        <div class="px-4 py-3 border-b border-slate-100 flex items-center gap-2">
          <div class="w-6 h-6 rounded-md bg-sky-500 flex items-center justify-center">
            <i class="fas fa-code text-white text-xs"></i>
          </div>
          <span class="text-sm font-semibold text-slate-700">Go 运行态</span>
          <span class="ml-2 text-xs text-slate-400 font-mono">{{ stat.go.version }}</span>
        </div>
        <div class="grid grid-cols-1 xs:grid-cols-2 md:grid-cols-4 divide-x divide-y divide-slate-100">
          <div class="px-4 py-3">
            <p class="text-xs text-slate-400 mb-1">Goroutine 数</p>
            <p class="text-sm font-bold text-slate-800">{{ stat.go.numGoroutine }}</p>
          </div>
          <div class="px-4 py-3">
            <p class="text-xs text-slate-400 mb-1">堆已分配</p>
            <p class="text-sm font-bold text-slate-800">{{ fmtBytes(stat.go.HeapAlloc) }}</p>
          </div>
          <div class="px-4 py-3">
            <p class="text-xs text-slate-400 mb-1">堆使用中</p>
            <p class="text-sm font-bold text-slate-800">{{ fmtBytes(stat.go.HeapInuse) }}</p>
          </div>
          <div class="px-4 py-3">
            <p class="text-xs text-slate-400 mb-1">系统申请</p>
            <p class="text-sm font-bold text-slate-800">{{ fmtBytes(stat.go.Sys) }}</p>
          </div>
          <div class="px-4 py-3">
            <p class="text-xs text-slate-400 mb-1">栈使用中</p>
            <p class="text-sm font-bold text-slate-800">{{ fmtBytes(stat.go.StackInuse) }}</p>
          </div>
          <div class="px-4 py-3">
            <p class="text-xs text-slate-400 mb-1">累计分配</p>
            <p class="text-sm font-bold text-slate-800">{{ fmtBytes(stat.go.TotalAlloc) }}</p>
          </div>
          <div class="px-4 py-3">
            <p class="text-xs text-slate-400 mb-1">GC 次数</p>
            <p class="text-sm font-bold text-slate-800">{{ stat.go.NumGC }}</p>
          </div>
          <div class="px-4 py-3">
            <p class="text-xs text-slate-400 mb-1">最后 GC</p>
            <p class="text-sm font-bold text-slate-800">{{ fmtGCTime(stat.go.LastGC) }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- 加载失败 -->
    <div v-else class="flex items-center gap-3 py-6 px-4 rounded-xl bg-slate-50">
      <i class="fas fa-triangle-exclamation text-2xl text-slate-300"></i>
      <p class="text-sm text-slate-500">获取系统信息失败</p>
    </div>
  </div>
</template>
