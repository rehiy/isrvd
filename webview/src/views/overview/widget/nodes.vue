<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { MonitorHostRecord, NodeInfo } from '@/service/types'

import { POLL_INTERVAL } from '@/helper/format'
import { hostCpuUsage, monitorCapacity } from '@/helper/monitor'
import { nodePath, nodeStatus } from '@/helper/node'

@Component
class NodeResources extends Vue {
    portal = usePortal()
    @Prop({ default: undefined }) readonly currentReading?: MonitorHostRecord | null

    nodeStatus = nodeStatus
    readings = new Map<string, MonitorHostRecord | null>()
    private request: AbortController | null = null
    private timer: ReturnType<typeof setInterval> | null = null

    get nodes() {
        const system = this.reading('')?.data.system
        return [
            {
                id: '', name: this.$t('中控'), online: true, status: 'active',
                hostname: system?.hostName, os: system?.platform, arch: system?.kernelArch
            },
            ...this.portal.nodes.filter(node => node.status === 'active')
        ]
    }

    get onlineCount() {
        return this.nodes.filter(node => node.online).length
    }

    reading(id: string) {
        return id === this.portal.currentNodeId ? this.currentReading : this.readings.get(id)
    }

    nodeOverviewPath(id: string) {
        return nodePath(id) + '#/overview'
    }

    metricValues(node: Pick<NodeInfo, 'id' | 'online'>) {
        const system = node.online ? this.reading(node.id)?.data.system : undefined
        return [
            { label: 'CPU', percent: hostCpuUsage(system), color: 'sky', detail: '' },
            { label: '内存', color: 'violet', ...monitorCapacity(system?.memoryUsed, system?.memoryTotal) },
            { label: '磁盘', color: 'amber', ...monitorCapacity(system?.diskUsed, system?.diskTotal) }
        ]
    }

    async load() {
        if (this.request) {
            return
        }
        const request = new AbortController()
        this.request = request
        try {
            await this.portal.nodeLoad(true)
            if (request.signal.aborted || this.portal.nodesLoadFailed) {
                return
            }
            const online = this.nodes.filter(node => node.online && node.id !== this.portal.currentNodeId)
            const onlineIDs = new Set(online.map(node => node.id))
            for (const id of this.readings.keys()) {
                if (!onlineIDs.has(id)) {
                    this.readings.delete(id)
                }
            }
            const concurrency = 4
            for (let start = 0; start < online.length && !request.signal.aborted; start += concurrency) {
                await Promise.all(online.slice(start, start + concurrency).map(node => this.loadSample(node.id, request.signal)))
            }
        } finally {
            this.request = null
        }
    }

    async loadSample(id: string, signal: AbortSignal) {
        let record: MonitorHostRecord | null
        try {
            const { payload } = await api.nodeHostSample(id, signal)
            record = payload?.data?.system ? payload : null
        } catch {
            record = null
        }
        if (!signal.aborted) {
            this.readings.set(id, record)
        }
    }

    handleVisibility() {
        if (!document.hidden) {
            this.load()
        }
    }

    mounted() {
        this.load()
        document.addEventListener('visibilitychange', this.handleVisibility)
        this.timer = setInterval(this.handleVisibility, POLL_INTERVAL * 3)
    }

    unmounted() {
        this.request?.abort()
        if (this.timer) {
            clearInterval(this.timer)
        }
        document.removeEventListener('visibilitychange', this.handleVisibility)
    }
}

export default toNative(NodeResources)
</script>

<template>
  <section class="card resource-panel">
    <div class="resource-section-header">
      <div class="resource-section-heading">
        <span class="row-icon resource-section-icon"><i class="fas fa-sitemap"></i></span>
        <h2 class="resource-section-title">{{ $t('节点资源') }}</h2>
      </div>
      <span v-if="nodes.length" class="badge-sm badge-muted tabular-nums">{{ $t('{0} / {1} 在线', onlineCount, nodes.length) }}</span>
    </div>
    <p v-if="portal.nodesLoadFailed" class="mt-3 text-sm text-amber-700">{{ $t('加载失败，请刷新重试') }}</p>
    <p v-else-if="portal.nodesLoading" class="mt-3 text-sm text-slate-500">{{ $t('加载中...') }}</p>
    <div class="mt-3 grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3" :class="{ 'resource-nodes-single': nodes.length === 1 }">
      <article
        v-for="node in nodes" :key="node.id"
        :aria-current="node.id === portal.currentNodeId ? 'true' : undefined"
        :class="[
          'resource-node',
          node.id === portal.currentNodeId ? 'border-primary-200 bg-primary-50/40' : 'border-slate-200/70 bg-slate-50/60',
          node.online && node.id !== portal.currentNodeId && 'transition-colors hover:border-primary-300 focus-within:ring-2 focus-within:ring-primary-500'
        ]"
      >
        <div class="flex items-center gap-2.5">
          <span class="row-icon bg-white text-primary-600"><i :class="['fas text-xs', node.id ? 'fa-server' : 'fa-sitemap']"></i></span>
          <div class="min-w-0 flex-1">
            <a
              v-if="node.online && node.id !== portal.currentNodeId"
              :href="nodeOverviewPath(node.id)" :title="node.name"
              class="item-title-sm hover:text-primary-600 after:absolute after:inset-0"
            >
              {{ node.name }} <i class="fas fa-arrow-right ml-1 text-[9px]" aria-hidden="true"></i>
            </a>
            <span v-else class="item-title-sm" :title="node.name">{{ node.name }}</span>
            <p v-if="node.hostname || node.os || node.arch" class="truncate text-xs text-slate-500">{{ [node.hostname, node.os, node.arch].filter(Boolean).join(' · ') }}</p>
          </div>
          <span v-if="node.online && node.id === portal.currentNodeId" class="badge-sm badge-primary shrink-0">{{ $t('当前节点') }}</span>
          <span v-else :class="['inline-flex items-center gap-1.5 shrink-0 text-xs', nodeStatus(node).cls]">
            <span class="size-1.5 rounded-full bg-current"></span>{{ $t(nodeStatus(node).text) }}
          </span>
        </div>
        <dl class="resource-node-metrics">
          <div v-for="metric in metricValues(node)" :key="metric.label" class="min-w-0" :title="metric.detail">
            <dt class="text-xs text-slate-500">{{ $t(metric.label) }}</dt>
            <dd class="mt-0.5 text-base font-semibold tabular-nums text-slate-900">
              {{ metric.percent === null ? '—' : metric.percent.toFixed(1) + '%' }}
            </dd>
            <div v-if="metric.percent !== null" :class="['resource-meter mt-1.5', 'resource-metric-' + metric.color]" aria-hidden="true">
              <span :style="{ width: Math.min(100, Math.max(0, metric.percent)) + '%' }"></span>
            </div>
          </div>
        </dl>
        <p v-if="node.online && reading(node.id) === null" class="mt-2 text-xs text-amber-700">{{ $t('获取监控数据失败') }}</p>
      </article>
    </div>
  </section>
</template>
