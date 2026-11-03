<script lang="ts">
import type { ChartOptions } from 'chart.js'
import { markRaw } from 'vue'
import { Component, Ref, Vue, Watch, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import { absUrl } from '@/service/client'
import type { MetricsGatewayName, MetricsGatewayStats, MetricsSourceName, MetricsSourceStatus } from '@/service/types'

import Chart, { makeLineDataset } from '@/helper/chart'
import { copyToClipboard } from '@/helper/dom'
import { POLL_INTERVAL } from '@/helper/format'
import { formatMonitorBytes, monitorTimeLabel } from '@/helper/monitor'

import PageSearch from '@/component/page-search.vue'
import ToggleCard from '@/component/toggle-card.vue'

const SOURCES = [
    { name: 'host', label: '本机监控', desc: 'CPU、内存、磁盘、网络与进程运行时指标' },
    { name: 'containers', label: '容器监控', desc: '运行中容器的 CPU、内存、网络与磁盘指标' },
    { name: 'apisix', label: 'APISIX', desc: '请求数、状态码、耗时与响应流量' },
    { name: 'caddy', label: 'Caddy', desc: '请求数、状态码、耗时与响应流量' },
] as const

const REQUEST_SERIES = [
    { key: 's2xx', label: '2xx', color: '#10b981' },
    { key: 's3xx', label: '3xx', color: '#3b82f6' },
    { key: 's4xx', label: '4xx', color: '#f59e0b' },
    { key: 's5xx', label: '5xx', color: '#ef4444' },
] as const

@Component({ components: { PageSearch, ToggleCard } })
class MonitoringMetrics extends Vue {
    portal = usePortal()

    @Ref readonly statsRoot!: HTMLElement

    loading = false
    statuses: MetricsSourceStatus[] = []
    stats: MetricsGatewayStats[] = []
    statusFailed = false
    statsFailed = false
    statusLoaded = false
    range = 3600
    searchText: Record<MetricsGatewayName, string> = { caddy: '', apisix: '' }
    busy: Partial<Record<MetricsSourceName, boolean>> = {}
    formatMonitorBytes = formatMonitorBytes
    readonly requestSeries = REQUEST_SERIES
    hiddenRequestSeries: Record<MetricsGatewayName, boolean[]> = { caddy: [], apisix: [] }

    private charts: Chart<'line'>[] = []
    private timer = 0
    private requestId = 0
    private destroyed = false

    readonly ranges = [
        { value: 900, label: '15 分钟' },
        { value: 3600, label: '1 小时' },
        { value: 21600, label: '6 小时' },
        { value: 86400, label: '24 小时' },
    ]

    get canReadStatus() { return this.portal.hasPerm('GET /api/metrics/status') }
    get canReadStats() { return this.portal.hasPerm('GET /api/metrics/stats') }
    get canUpdate() { return this.portal.hasPerm('PUT /api/metrics/gateway/:name') }
    get statusKnown() { return this.canReadStatus && this.statusLoaded && !this.statusFailed }
    get visibleGatewayStatuses() {
        return this.statuses.filter(status => (status.name === 'caddy' || status.name === 'apisix') && status.available && (status.enabled || status.enabledUnknown))
    }
    get gateways() {
        return this.stats.filter(stats => !this.statusKnown || this.visibleGatewayStatuses.some(status => status.name === stats.name)).map(stats => ({
            name: stats.name,
            label: stats.name === 'apisix' ? 'APISIX' : 'Caddy',
            siteLabel: stats.name === 'apisix' ? '路由' : '站点',
            siteTitle: stats.name === 'apisix' ? '路由排行' : '站点排行',
            hasSeries: stats.series.length > 0,
            error: stats.error,
            summaryCards: this.summaryCards(stats),
            sites: this.filteredSites(stats),
        }))
    }

    get sources() {
        return SOURCES.map(source => {
            const status = this.statuses.find(s => s.name === source.name)
            return {
                ...source,
                url: absUrl(`metrics/${source.name}`),
                permitted: this.portal.hasPerm(`GET /api/metrics/${source.name}`),
                status,
                controllable: this.statusKnown && !!status && !status.enabledUnknown && this.canUpdate,
                state: this.sourceState(status),
            }
        })
    }

    filteredSites(stats: MetricsGatewayStats) {
        const query = this.searchText[stats.name].trim().toLowerCase()
        return stats.sites.filter(s => s.site.toLowerCase().includes(query))
    }

    summaryCards(stats: MetricsGatewayStats) {
        const s = stats.summary
        return [
            { label: '请求总数', value: this.formatCount(s.requests), icon: 'fas fa-arrow-right-arrow-left', color: 'bg-cyan-500' },
            { label: '平均 QPS', value: this.formatRate(s.rps), icon: 'fas fa-gauge-high', color: 'bg-indigo-500' },
            { label: '5xx 占比', value: (s.errorRate * 100).toFixed(2) + '%', icon: 'fas fa-triangle-exclamation', color: 'bg-amber-500', warning: s.errorRate > 0.01 },
            { label: '平均耗时', value: this.formatMs(s.avg), icon: 'fas fa-stopwatch', color: 'bg-violet-500' },
            { label: '响应流量', value: formatMonitorBytes(s.bytesOut), icon: 'fas fa-arrow-up', color: 'bg-emerald-500' },
            { label: '2xx 请求数', value: this.formatCount(s.s2xx), icon: 'fas fa-circle-check', color: 'bg-emerald-500' },
            { label: '3xx 请求数', value: this.formatCount(s.s3xx), icon: 'fas fa-arrow-right-arrow-left', color: 'bg-blue-500' },
            { label: '4xx 请求数', value: this.formatCount(s.s4xx), icon: 'fas fa-circle-exclamation', color: 'bg-amber-500' },
            { label: '5xx 请求数', value: this.formatCount(s.s5xx), icon: 'fas fa-circle-xmark', color: 'bg-red-500', warning: s.errorRate > 0.01 },
        ]
    }

    sourceState(status?: MetricsSourceStatus) {
        if (this.statusFailed || status?.error) {
            return { label: '检查失败', color: 'text-red-500 font-medium' }
        }
        if (!status) {
            return { label: this.statusLoaded ? '未配置' : '状态未查询', color: 'text-slate-500' }
        }
        if (!status.enabled) {
            return { label: '指标未开启', color: 'text-amber-600 font-medium' }
        }
        if (status.name === 'containers' && !status.available) {
            return { label: 'Docker 不可用', color: 'text-red-500 font-medium' }
        }
        return status.scrapable
            ? { label: '可抓取', color: 'text-emerald-600 font-medium' }
            : { label: '不可抓取', color: 'text-red-500 font-medium' }
    }

    // ─── 数据与操作 ───

    @Watch('portal.initialized')
    onInitialized(initialized: boolean) {
        if (initialized) {
            this.load()
        }
    }

    async load(silent = false) {
        if (!this.portal.initialized || this.destroyed || (silent && this.loading)) {
            return
        }
        const id = ++this.requestId
        this.loading = true
        try {
            const [statuses, stats] = await Promise.allSettled([
                this.canReadStatus ? api.metricsStatus() : Promise.resolve({ payload: [] }),
                this.canReadStats ? api.metricsStats(this.range) : Promise.resolve({ payload: [] }),
            ])
            if (this.destroyed || id !== this.requestId) {
                return
            }
            this.statusFailed = statuses.status === 'rejected'
            this.statsFailed = stats.status === 'rejected'
            if (statuses.status === 'fulfilled') {
                this.statuses = statuses.value.payload || []
                this.statusLoaded = this.canReadStatus
            }
            if (stats.status === 'fulfilled') {
                this.stats = stats.value.payload || []
            }
            await this.$nextTick()
            this.renderCharts()
        } finally {
            if (id === this.requestId) {
                this.loading = false
            }
        }
    }

    canToggle(status: MetricsSourceStatus) {
        return this.canUpdate && !this.loading && !this.statusFailed && !status.enabledUnknown && !this.busy[status.name]
            && (status.name === 'host' || status.name === 'containers' || status.available)
    }

    async updateSource(name: MetricsSourceName, enabled: boolean) {
        const status = this.statuses.find(s => s.name === name)
        if (!status || !this.canToggle(status)) {
            return
        }
        this.busy[name] = true
        try {
            await api.metricsSourceUpdate(name, enabled)
            if (!this.destroyed) {
                status.enabled = enabled
                await this.load()
            }
        } finally {
            this.busy[name] = false
        }
    }

    async copyEndpoint(url: string) {
        const ok = await copyToClipboard(url)
        this.portal.showNotification(ok ? 'success' : 'error', this.$t(ok ? '已复制' : '复制失败'))
    }

    @Watch('range')
    onRangeChange() {
        this.load()
    }

    // ─── 格式化与图表 ───

    formatCount(n: number) {
        return n.toLocaleString(this.portal.locale, { maximumFractionDigits: 0 })
    }

    formatRate(n: number) {
        return n.toLocaleString(this.portal.locale, { maximumFractionDigits: 2 })
    }

    formatMs(n: number) {
        return n >= 1000 ? (n / 1000).toFixed(2) + ' s' : n.toFixed(1) + ' ms'
    }

    private chartOptions(format: (v: number) => string, stacked: boolean): ChartOptions<'line'> {
        return {
            responsive: true, maintainAspectRatio: false, animation: false,
            interaction: { intersect: false, mode: 'index' },
            plugins: {
                legend: { display: false },
                tooltip: {
                    backgroundColor: 'rgba(15,23,42,0.9)', titleFont: { size: 10 }, bodyFont: { size: 10 }, padding: 8, cornerRadius: 6,
                    callbacks: { label: ctx => `${ctx.dataset.label}: ${format(ctx.parsed.y ?? 0)}` },
                },
            },
            scales: {
                x: { grid: { display: false }, border: { display: false }, ticks: { font: { size: 9 }, maxTicksLimit: 6, maxRotation: 0, color: '#94a3b8' } },
                y: {
                    beginAtZero: true, stacked, grid: { color: 'rgba(148,163,184,0.08)' }, border: { display: false },
                    ticks: { font: { size: 9 }, maxTicksLimit: 4, padding: 4, color: '#94a3b8', callback: v => format(Number(v)) },
                },
            },
            elements: { point: { radius: 0, hoverRadius: 3 }, line: { tension: 0.25, borderWidth: 1.5 } },
        }
    }

    private destroyCharts() {
        this.charts.forEach(c => c.destroy())
        this.charts = []
    }

    toggleRequestSeries(name: MetricsGatewayName, index: number) {
        const chart = this.charts.find(c => c.canvas.dataset.gateway === name && c.canvas.dataset.kind === 'requests')
        if (!chart) {
            return
        }
        const hidden = chart.isDatasetVisible(index)
        this.hiddenRequestSeries[name][index] = hidden
        chart.setDatasetVisibility(index, !hidden)
        chart.update()
    }

    renderCharts() {
        this.destroyCharts()
        if (this.destroyed || this.statsFailed || !this.statsRoot) {
            return
        }
        for (const canvas of this.statsRoot.querySelectorAll<HTMLCanvasElement>('canvas[data-gateway]')) {
            const stats = this.stats.find(s => s.name === canvas.dataset.gateway)
            if (!stats?.series.length) {
                continue
            }
            const series = stats.series
            const requests = canvas.dataset.kind === 'requests'
            const labels = series.map(p => monitorTimeLabel(p.ts))
            const datasets = requests ? this.requestSeries.map((item, index) => ({
                ...makeLineDataset(series.map(p => p[item.key]), item.color, item.label),
                hidden: !!this.hiddenRequestSeries[stats.name][index],
            })) : [{ ...makeLineDataset(series.map(p => p.avg), '#06b6d4', this.$t('平均耗时')), fill: false }]
            this.charts.push(markRaw(new Chart(canvas, {
                type: 'line',
                data: { labels, datasets },
                options: requests
                    ? this.chartOptions(v => this.formatRate(v) + ' req/s', true)
                    : this.chartOptions(v => this.formatMs(v), false),
            })))
        }
    }

    // ─── 生命周期 ───

    mounted() {
        this.load()
        this.timer = window.setInterval(() => {
            if (!document.hidden) {
                this.load(true)
            }
        }, POLL_INTERVAL * 6)
    }

    unmounted() {
        this.destroyed = true
        window.clearInterval(this.timer)
        this.destroyCharts()
    }
}

export default toNative(MonitoringMetrics)
</script>

<template>
  <div class="page">
    <div class="page-toolbar">
      <div class="toolbar-desktop">
        <div class="title-group-static">
          <div class="page-icon bg-cyan-500"><i class="fas fa-chart-line text-white"></i></div>
          <div class="min-w-0">
            <h1 class="title-text">{{ $t('监控指标') }}</h1>
            <p class="text-xs text-slate-500 truncate">{{ $t('主机、容器与网关的指标出口及访问统计') }}</p>
          </div>
        </div>
        <div class="action-group">
          <select v-if="canReadStats" v-model.number="range" class="select-sm" :aria-label="$t('时间范围')">
            <option v-for="r in ranges" :key="r.value" :value="r.value">{{ $t(r.label) }}</option>
          </select>
          <button class="btn btn-secondary" :disabled="loading" :aria-busy="loading" @click="load()">
            <i :class="loading ? 'fas fa-spinner fa-spin' : 'fas fa-rotate'"></i>{{ $t('刷新') }}
          </button>
        </div>
      </div>
      <div class="toolbar-mobile">
        <div class="title-group">
          <div class="page-icon bg-cyan-500"><i class="fas fa-chart-line text-white"></i></div>
          <div class="min-w-0">
            <h1 class="title-text">{{ $t('监控指标') }}</h1>
            <p class="text-xs text-slate-500 truncate">{{ $t('主机、容器与网关') }}</p>
          </div>
        </div>
        <div class="action-group-sm">
          <select v-if="canReadStats" v-model.number="range" class="select-sm max-w-28" :aria-label="$t('时间范围')">
            <option v-for="r in ranges" :key="r.value" :value="r.value">{{ $t(r.label) }}</option>
          </select>
          <button class="btn btn-secondary btn-square" :title="$t('刷新')" :aria-label="$t('刷新')" :disabled="loading" :aria-busy="loading" @click="load()">
            <i :class="loading ? 'fas fa-spinner fa-spin' : 'fas fa-rotate'"></i>
          </button>
        </div>
      </div>
    </div>

    <div class="card-body">
      <div ref="statsRoot" class="space-y-4">
        <section :aria-label="$t('指标来源')">
          <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <ToggleCard
              v-for="source in sources"
              :key="source.name"
              :model-value="!!source.status?.enabled"
              always-expanded
              :label="$t(source.label)"
              :disabled="!source.controllable || !source.status || !canToggle(source.status)"
              :aria-busy="busy[source.name]"
              @update:model-value="updateSource(source.name, $event)"
            >
              <template #label>
                <div class="inline-info flex-wrap">
                  <span class="text-sm text-slate-600">{{ $t(source.label) }}</span>
                  <i v-if="busy[source.name]" class="fas fa-spinner fa-spin text-xs text-slate-400" aria-hidden="true"></i>
                  <span v-if="source.controllable" class="text-xs" :class="source.state.color">{{ $t(source.state.label) }}</span>
                </div>
              </template>
              <template #desc>
                {{ $t(source.desc) }}
                <span v-if="source.status?.error" class="block text-red-500 break-all">{{ source.status.error }}</span>
              </template>
              <template v-if="!source.controllable" #control>
                <span class="text-xs flex-shrink-0" :class="source.state.color">{{ $t(source.state.label) }}</span>
              </template>
              <div class="space-y-4">
                <div class="flex items-start gap-2">
                  <code class="detail-value-mono flex-1 min-w-0">{{ source.url }}</code>
                  <button class="btn-icon-slate flex-shrink-0" :title="$t('复制地址')" :aria-label="$t('复制地址')" :disabled="!source.permitted" @click="copyEndpoint(source.url)">
                    <i class="fas fa-copy"></i>
                  </button>
                </div>
                <p v-if="!source.permitted" class="text-xs text-amber-600">{{ $t('未授予此出口的抓取权限') }}</p>
              </div>
            </ToggleCard>
          </div>
        </section>

        <template v-if="canReadStats && (!statusKnown || visibleGatewayStatuses.length)">
          <div v-if="loading && !stats.length" class="card-body">
            <div class="empty-state"><div class="spinner-lg"></div><p class="text-slate-500">{{ $t('加载中...') }}</p></div>
          </div>
          <div v-else-if="statsFailed" class="card-body">
            <div class="empty-state">
              <div class="empty-state-icon"><i class="fas fa-triangle-exclamation text-amber-500"></i></div>
              <p class="text-slate-500">{{ $t('统计暂不可用，请刷新重试') }}</p>
            </div>
          </div>
          <template v-else-if="gateways.length">
            <section v-for="gateway in gateways" :key="gateway.name" class="space-y-4" :aria-labelledby="`metrics-${gateway.name}-stats`">
              <div class="config-section-heading">
                <h2 :id="`metrics-${gateway.name}-stats`" class="config-section-title">{{ gateway.label }}</h2>
                <p v-if="!gateway.error" class="config-section-description">{{ $t('30 秒采样，最多保留 24 小时；重启或重载后清空。') }}</p>
              </div>
              <div v-if="gateway.error" class="empty-note">{{ gateway.error }}</div>
              <div v-else-if="!gateway.hasSeries" class="empty-note">{{ $t('暂无统计数据。开启指标并产生访问后，约 1 分钟内开始出现。') }}</div>
              <template v-else>
                <div class="space-y-4">
                  <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
                    <div v-for="item in gateway.summaryCards" :key="item.label" class="stat-card">
                      <div class="page-icon" :class="item.color"><i :class="item.icon" class="text-white"></i></div>
                      <div class="min-w-0 flex-1">
                        <p class="text-xs text-slate-400 mb-0.5">{{ $t(item.label) }}</p>
                        <p class="font-semibold truncate" :class="item.warning ? 'text-red-500' : 'text-slate-800'">{{ item.value }}</p>
                      </div>
                    </div>
                  </div>
                  <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
                    <div class="metric-card">
                      <div class="inline-info flex-wrap mb-3">
                        <div class="inline-info">
                          <div class="card-icon bg-cyan-500"><i class="fas fa-chart-line text-white text-xs"></i></div>
                          <h3 class="text-sm font-semibold text-slate-700">{{ $t('请求速率（按状态码）') }}</h3>
                        </div>
                        <div class="metric-legend ml-auto !mb-0 !gap-2 !text-slate-500">
                          <button
                            v-for="(item, index) in requestSeries"
                            :key="item.key"
                            type="button"
                            class="monitor-legend-item rounded-lg py-1 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-cyan-500"
                            :class="{ 'line-through opacity-50': hiddenRequestSeries[gateway.name][index] }"
                            :aria-pressed="!hiddenRequestSeries[gateway.name][index]"
                            @click="toggleRequestSeries(gateway.name, index)"
                          >
                            <span class="w-2 h-2 border" :style="{ borderColor: item.color, backgroundColor: item.color + '1a' }" aria-hidden="true"></span>
                            {{ item.label }}
                          </button>
                        </div>
                      </div>
                      <div class="monitor-chart-box"><canvas :data-gateway="gateway.name" data-kind="requests" role="img" :aria-label="gateway.label + ' ' + $t('请求速率（按状态码）')"></canvas></div>
                    </div>
                    <div class="metric-card">
                      <div class="inline-info mb-3">
                        <div class="card-icon bg-violet-500"><i class="fas fa-stopwatch text-white text-xs"></i></div>
                        <h3 class="text-sm font-semibold text-slate-700">{{ $t('平均耗时') }}</h3>
                      </div>
                      <div class="monitor-chart-box"><canvas :data-gateway="gateway.name" data-kind="latency" role="img" :aria-label="gateway.label + ' ' + $t('平均耗时')"></canvas></div>
                    </div>
                  </div>
                </div>
                <div class="card">
                  <div class="card-header">
                    <h3 class="text-sm font-semibold text-slate-700">{{ $t(gateway.siteTitle) }}</h3>
                    <div class="ml-auto hidden md:block">
                      <PageSearch v-model="searchText[gateway.name]" :search-key="`metrics-${gateway.name}-sites`" focus-color="cyan" :placeholder="$t('搜索站点或路由...')" :type-to-search="gateway.name === gateways.find(g => g.hasSeries)?.name" />
                    </div>
                  </div>
                  <div class="mobile-search">
                    <PageSearch v-model="searchText[gateway.name]" :search-key="`metrics-${gateway.name}-sites`" focus-color="cyan" width-class="w-full" :placeholder="$t('搜索站点或路由...')" />
                  </div>
                  <div v-if="!gateway.sites.length" class="card-body">
                    <div class="empty-state"><p class="text-slate-500">{{ $t('暂无数据') }}</p></div>
                  </div>
                  <template v-else>
                    <div class="card-table hidden md:block">
                      <table class="w-full">
                        <thead>
                          <tr class="bg-slate-100 border-b border-slate-200">
                            <th class="th">{{ $t(gateway.siteLabel) }}</th>
                            <th class="th">{{ $t('请求数') }}</th>
                            <th class="th">4xx</th>
                            <th class="th">5xx</th>
                            <th class="th">{{ $t('平均耗时') }}</th>
                            <th class="th">{{ $t('响应流量') }}</th>
                          </tr>
                        </thead>
                        <tbody class="divide-y divide-slate-100">
                          <tr v-for="s in gateway.sites" :key="s.site" class="hover:bg-slate-50 transition-colors">
                            <td class="px-4 py-3 max-w-[280px]">
                              <div class="inline-info">
                                <div class="row-icon bg-cyan-400"><i class="fas fa-route text-white text-sm"></i></div>
                                <div class="min-w-0"><span class="item-title" :title="s.site">{{ s.site }}</span></div>
                              </div>
                            </td>
                            <td class="td-text">{{ formatCount(s.requests) }}</td>
                            <td class="px-4 py-3 text-sm" :class="s.s4xx ? 'text-amber-600 font-medium' : 'text-slate-600'">{{ formatCount(s.s4xx) }}</td>
                            <td class="px-4 py-3 text-sm" :class="s.s5xx ? 'text-red-500 font-medium' : 'text-slate-600'">{{ formatCount(s.s5xx) }}</td>
                            <td class="td-text">{{ formatMs(s.avg) }}</td>
                            <td class="td-text">{{ formatMonitorBytes(s.bytesOut) }}</td>
                          </tr>
                        </tbody>
                      </table>
                    </div>
                    <div class="card-body md:hidden space-y-3">
                      <div v-for="s in gateway.sites" :key="s.site" class="card-interactive">
                        <div class="card-info-row">
                          <div class="list-icon bg-cyan-400"><i class="fas fa-route text-white text-base"></i></div>
                          <div class="min-w-0">
                            <span class="item-title-sm" :title="s.site">{{ s.site }}</span>
                            <span class="item-subtitle">{{ $t(gateway.siteLabel) }}</span>
                          </div>
                        </div>
                        <div class="card-prop-row"><span class="text-xs text-slate-400">{{ $t('请求数') }}</span><span class="text-xs text-slate-500">{{ formatCount(s.requests) }}</span></div>
                        <div class="card-prop-row">
                          <span class="text-xs text-slate-400">4xx / 5xx</span>
                          <span class="text-xs" :class="s.s4xx ? 'text-amber-600 font-medium' : 'text-slate-500'">{{ formatCount(s.s4xx) }}</span>
                          <span class="text-xs text-slate-400">/</span>
                          <span class="text-xs" :class="s.s5xx ? 'text-red-500 font-medium' : 'text-slate-500'">{{ formatCount(s.s5xx) }}</span>
                        </div>
                        <div class="card-prop-row"><span class="text-xs text-slate-400">{{ $t('平均耗时') }}</span><span class="text-xs text-slate-500">{{ formatMs(s.avg) }}</span></div>
                        <div class="card-prop-row"><span class="text-xs text-slate-400">{{ $t('响应流量') }}</span><span class="text-xs text-slate-500">{{ formatMonitorBytes(s.bytesOut) }}</span></div>
                      </div>
                    </div>
                  </template>
                </div>
              </template>
            </section>
          </template>
        </template>
      </div>
    </div>
  </div>
</template>
