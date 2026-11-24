<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { DockerContainerInfo } from '@/service/types'

import { POLL_INTERVAL } from '@/helper/format'
import { containerResourceValues, formatMonitorBytes } from '@/helper/monitor'

interface ContainerReading {
    receivedAt: number
    cpuPercent: number | null
    memoryUsage: number | null
}

@Component({ emits: ['self-container'] })
class WorkloadsOverview extends Vue {
    portal = usePortal()
    containers: DockerContainerInfo[] | null = null
    readings = new Map<string, ContainerReading>()
    unhealthyIDs: Set<string> | null = null
    ranking: 'cpu' | 'memory' = 'cpu'
    showMetricHelp = false
    loading = false
    sampling = false
    healthLoading = false
    failed = false
    clock = performance.now()
    readonly rankings = [{ key: 'cpu', label: 'CPU' }, { key: 'memory', label: '内存' }] as const
    private timer: ReturnType<typeof setInterval> | null = null
    private clockTimer: ReturnType<typeof setInterval> | null = null
    private readonly lifetime = new AbortController()

    get canMonitor() {
        return this.portal.hasPerm('GET /api/overview/monitor')
    }

    get canInspect() {
        return this.portal.hasPerm('GET /api/docker/container/:id')
    }

    get runningContainers() {
        return (this.containers ?? []).filter(container => container.state === 'running')
    }

    get sampledContainers() {
        if (!this.canMonitor || this.failed) {
            return []
        }
        return this.runningContainers.flatMap(container => {
            const reading = this.readings.get(container.id)
            return reading && this.clock - reading.receivedAt <= 90_000 ? [{ ...container, ...reading }] : []
        })
    }

    get memorySamples() {
        return this.sampledContainers.filter(container => container.memoryUsage !== null)
    }

    get totalMemory() {
        if (!this.runningContainers.length || this.memorySamples.length !== this.runningContainers.length) {
            return null
        }
        return this.memorySamples.reduce((total, container) => total + (container.memoryUsage ?? 0), 0)
    }

    get anomalyCount() {
        if (this.unhealthyIDs === null) {
            return null
        }
        return (this.containers ?? []).filter(container =>
            container.state === 'restarting' || container.state === 'dead'
            || (container.state === 'running' && this.unhealthyIDs?.has(container.id))
        ).length
    }

    get memoryHint() {
        if (!this.canMonitor) {
            return '无容器监控权限'
        }
        if (!this.runningContainers.length) {
            return '暂无运行中容器'
        }
        if (this.totalMemory !== null) {
            return '已扣除非活跃文件缓存'
        }
        if (this.sampling) {
            return '加载中...'
        }
        return this.memorySamples.length ? '采集不完整，未计算总量' : '暂无监控数据'
    }

    get summaryCards() {
        return [
            { label: '运行中', icon: 'fa-play', color: 'bg-emerald-50 text-emerald-600', value: this.runningContainers.length, hint: '' },
            { label: '已停止', icon: 'fa-stop', color: 'bg-slate-100 text-slate-500', value: (this.containers ?? []).filter(container => ['exited', 'stopped'].includes(container.state)).length, hint: '' },
            { label: '异常', icon: 'fa-triangle-exclamation', color: 'bg-amber-50 text-amber-600', value: this.anomalyCount ?? '—', hint: this.healthLoading ? '加载中...' : this.unhealthyIDs === null ? '健康检查数据不可用' : '重启、终止或不健康' },
            { label: '内存占用', icon: 'fa-memory', color: 'bg-violet-50 text-violet-600', value: this.totalMemory === null ? '—' : formatMonitorBytes(this.totalMemory), hint: this.memoryHint },
        ]
    }

    get rankingSamples() {
        const field = this.ranking === 'cpu' ? 'cpuPercent' : 'memoryUsage'
        return this.sampledContainers.filter(container => container[field] !== null)
            .sort((a, b) => (b[field] ?? 0) - (a[field] ?? 0) || (a.name || a.id).localeCompare(b.name || b.id))
    }

    get topContainers() {
        const field = this.ranking === 'cpu' ? 'cpuPercent' : 'memoryUsage'
        const maximum = this.rankingSamples[0]?.[field] ?? 0
        return this.rankingSamples.slice(0, 3).map(container => {
            const value = container[field] ?? 0
            return {
                id: container.id,
                name: container.name || container.id.slice(0, 12),
                path: this.canInspect ? '/docker/container/' + container.id : undefined,
                value: this.ranking === 'cpu' ? value.toFixed(1) + '%' : formatMonitorBytes(value),
                width: maximum > 0 ? value / maximum * 100 : 0,
            }
        })
    }

    get pausedCount() {
        return (this.containers ?? []).filter(container => container.state === 'paused').length
    }

    async loadHealth() {
        const { signal } = this.lifetime
        this.healthLoading = true
        try {
            const { payload } = await api.dockerContainerList(true, { health: ['unhealthy'] }, { signal, timeout: 5000, inlineErrors: true })
            if (!signal.aborted) {
                this.unhealthyIDs = Array.isArray(payload) ? new Set(payload.map(container => container.id)) : null
            }
        } catch {
            if (!signal.aborted) {
                this.unhealthyIDs = null
            }
        } finally {
            this.healthLoading = false
        }
    }

    async loadReadings() {
        const { signal } = this.lifetime
        if (!this.canMonitor) {
            this.readings.clear()
            return
        }
        const containers = this.runningContainers
        const readings = new Map<string, ContainerReading>()
        let index = 0
        this.sampling = containers.length > 0
        try {
            await Promise.all(Array.from({ length: Math.min(4, containers.length) }, async () => {
                while (!signal.aborted && !document.hidden && index < containers.length) {
                    const container = containers[index++]
                    try {
                        const { payload } = await api.overviewMonitor({ type: 'container', id: container.id, since: 0 }, { signal, timeout: 5000, inlineErrors: true })
                        if (!signal.aborted && payload?.data && Number.isFinite(payload.ts)) {
                            readings.set(container.id, { receivedAt: performance.now(), ...containerResourceValues(payload) })
                        }
                    } catch {
                        if (signal.aborted) {
                            return
                        }
                    }
                }
            }))
            if (!signal.aborted) {
                this.readings = readings
                this.clock = performance.now()
            }
        } finally {
            this.sampling = false
        }
    }

    async load() {
        if (this.loading) {
            return
        }
        const { signal } = this.lifetime
        this.loading = true
        this.clock = performance.now()
        try {
            const { payload } = await api.dockerContainerList(true, undefined, { signal, timeout: 10000 })
            if (signal.aborted) {
                return
            }
            if (!Array.isArray(payload)) {
                throw new Error('Invalid container list')
            }
            this.containers = payload
            this.failed = false
            this.$emit('self-container', payload.find(container => container.isSelf) ?? null)
            await Promise.all([this.loadHealth(), this.loadReadings()])
        } catch {
            if (!signal.aborted) {
                this.failed = true
                this.readings.clear()
                this.unhealthyIDs = null
                this.$emit('self-container', null)
            }
        } finally {
            this.loading = false
        }
    }

    handleVisibility() {
        this.clock = performance.now()
        if (!document.hidden) {
            this.load()
        }
    }

    mounted() {
        this.load()
        document.addEventListener('visibilitychange', this.handleVisibility)
        this.timer = setInterval(this.handleVisibility, POLL_INTERVAL * 6)
        this.clockTimer = setInterval(() => { this.clock = performance.now() }, POLL_INTERVAL)
    }

    unmounted() {
        this.lifetime.abort()
        document.removeEventListener('visibilitychange', this.handleVisibility)
        if (this.timer) {
            clearInterval(this.timer)
        }
        if (this.clockTimer) {
            clearInterval(this.clockTimer)
        }
    }
}

export default toNative(WorkloadsOverview)
</script>

<template>
  <div class="min-w-0">
    <div class="resource-section-header">
      <div class="resource-section-heading flex-wrap">
        <span class="row-icon resource-service-icon"><i class="fab fa-docker"></i></span>
        <h2 class="resource-section-title">{{ $t('Docker 服务') }}</h2>
        <span v-if="containers" class="badge-sm badge-muted tabular-nums" :title="$t('共 {0} 个容器，已暂停 {1} 个', containers.length, pausedCount)">
          {{ $t('{0} 个容器', containers.length) }}
        </span>
        <span v-if="pausedCount" class="badge-sm badge-muted tabular-nums">{{ $t('暂停 {0}', pausedCount) }}</span>
        <span v-if="loading" class="spinner-sm" aria-hidden="true"></span>
      </div>
      <div class="flex items-center gap-2">
        <button
          type="button" class="btn btn-ghost btn-square" :title="$t('指标说明')" :aria-label="$t('指标说明')"
          :aria-expanded="showMetricHelp" aria-controls="container-metric-help" @click="showMetricHelp = !showMetricHelp"
        >
          <i class="fas fa-circle-info" aria-hidden="true"></i>
        </button>
        <router-link to="/docker/containers" class="btn btn-secondary">
          {{ $t('容器管理') }}<i class="fas fa-arrow-right text-xs"></i>
        </router-link>
      </div>
    </div>
    <div v-if="showMetricHelp" id="container-metric-help" class="mt-3 rounded-lg bg-slate-50 p-3 text-xs text-slate-500 space-y-2">
      <p>{{ $t('仅统计重启中、已终止和健康检查不通过的容器；正常停止不算异常') }}</p>
      <p>{{ $t('CPU 100% 表示占用 1 个逻辑核；内存按占用字节排序') }}</p>
      <p>{{ $t('运行中容器内存') }} · {{ $t(memoryHint) }}</p>
    </div>
    <p v-if="failed" class="mt-3 text-sm text-amber-700">{{ $t('获取容器列表失败') }}</p>
    <div v-if="!containers && !failed" class="overview-loading">
      <span class="spinner-md"></span><span class="text-sm text-slate-500">{{ $t('加载中...') }}</span>
    </div>
    <template v-else-if="containers">
      <div class="mt-4 grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
        <div v-for="card in summaryCards" :key="card.label" class="stat-card min-w-0">
          <span :class="['page-icon', card.color]"><i :class="['fas', card.icon]"></i></span>
          <div class="min-w-0 flex-1">
            <p class="resource-service-label" :title="card.hint ? $t(card.hint) : undefined">
              {{ $t(card.label) }}<i v-if="card.value === '—'" class="fas fa-circle-info ml-1 text-slate-400" aria-hidden="true"></i>
            </p>
            <p class="resource-metric-value truncate text-xl sm:text-2xl" :title="card.hint ? $t(card.hint) : String(card.value)">{{ card.value }}</p>
          </div>
        </div>
      </div>
      <div class="resource-section-header mt-4">
        <div class="flex min-w-0 items-center gap-2">
          <h3 class="text-xs font-medium text-slate-500" :title="$t('资源占用前三名')">TOP 3</h3>
          <span
            v-if="canMonitor && runningContainers.length && rankingSamples.length < runningContainers.length"
            class="badge-sm badge-warning tabular-nums" role="status"
            :title="$t('已采集 {0} / {1} 个运行中容器', rankingSamples.length, runningContainers.length)"
            :aria-label="$t('已采集 {0} / {1} 个运行中容器', rankingSamples.length, runningContainers.length)"
          >
            {{ rankingSamples.length }} / {{ runningContainers.length }}
          </span>
          <span v-if="sampling" class="spinner-sm" aria-hidden="true"></span>
        </div>
        <div class="tab-group">
          <button
            v-for="item in rankings" :key="item.key" type="button"
            :class="['tab-btn', ranking === item.key ? 'tab-btn-active text-primary-600' : 'tab-btn-inactive']"
            :aria-pressed="ranking === item.key"
            @click="ranking = item.key"
          >
            {{ $t(item.label) }}
          </button>
        </div>
      </div>
      <ol v-if="topContainers.length" class="grid grid-cols-1 gap-3 mt-3 xl:grid-cols-3">
        <li v-for="(container, index) in topContainers" :key="container.id" class="rounded-lg border border-slate-200/70 bg-slate-50/60 p-3.5 min-w-0">
          <div class="flex items-center gap-2.5">
            <span class="text-xs tabular-nums text-slate-400 shrink-0">{{ String(index + 1).padStart(2, '0') }}</span>
            <component
              :is="container.path ? 'RouterLink' : 'span'" :to="container.path"
              class="item-title-sm min-w-0 flex-1" :class="{ 'hover:text-primary-600': container.path }" :title="container.name"
            >
              {{ container.name }}
            </component>
            <span class="text-sm font-semibold text-slate-800 tabular-nums shrink-0">{{ container.value }}</span>
          </div>
          <div :class="['resource-meter mt-3', ranking === 'cpu' ? 'resource-metric-sky' : 'resource-metric-violet']" aria-hidden="true">
            <span :style="{ width: container.width + '%' }"></span>
          </div>
        </li>
      </ol>
      <p v-else class="mt-3 rounded-lg bg-slate-50 p-4 text-center text-sm text-slate-500">
        {{ $t(!canMonitor ? '无容器监控权限' : !runningContainers.length ? '暂无运行中容器' : sampling ? '加载中...' : '暂无监控数据') }}
      </p>
    </template>
  </div>
</template>
