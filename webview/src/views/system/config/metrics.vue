<script lang="ts">
import { Component, Vue, Watch, toNative } from 'vue-facing-decorator'

import { useConfigStore, usePortal } from '@/stores'

import api from '@/service/api'
import { absUrl } from '@/service/client'
import type { MetricsSourceName, MetricsSourceStatus } from '@/service/types'

import { copyToClipboard } from '@/helper/dom'

import ToggleCard from '@/component/toggle-card.vue'

const SOURCES = [
    { name: 'host', label: '本机监控', desc: 'CPU、内存、磁盘、网络与进程运行时指标' },
    { name: 'containers', label: '容器监控', desc: '最后保存的容器 CPU、内存、网络与磁盘指标' },
    { name: 'apisix', label: 'APISIX', desc: '请求数、状态码、耗时与响应流量' },
    { name: 'caddy', label: 'Caddy', desc: '请求数、状态码、耗时与响应流量' },
] as const

@Component({ components: { ToggleCard } })
class ConfigMetrics extends Vue {
    portal = usePortal()
    config = useConfigStore()
    loading = false
    statuses: MetricsSourceStatus[] = []
    statusFailed = false
    statusLoaded = false
    busy: Partial<Record<MetricsSourceName, boolean>> = {}
    private destroyed = false
    private refreshPending = false
    private controller?: AbortController

    get canReadStatus() { return this.portal.hasPerm('GET /api/metrics/status') }
    get canUpdate() { return this.portal.hasPerm('PUT /api/metrics/gateway/:name') }
    get statusKnown() { return this.canReadStatus && this.statusLoaded && !this.statusFailed }
    get canReadConfig() { return this.portal.hasPerm('GET /api/system/config') && this.config.loaded }
    get canEditConfig() {
        return this.canReadConfig && this.portal.hasPerm('PUT /api/system/config')
            && !this.config.loading && !this.config.saving && !this.destroyed
    }

    isLocal(name: MetricsSourceName) {
        return name === 'host' || name === 'containers'
    }

    sourceEnabled(name: MetricsSourceName, status?: MetricsSourceStatus) {
        if (this.isLocal(name) && this.canReadConfig) {
            return name === 'host' ? this.config.draft.metrics.hostEnabled : this.config.draft.metrics.containersEnabled
        }
        return !!status?.enabled
    }

    sourceAvailable(name: MetricsSourceName) {
        if (name === 'host') {
            return true
        }
        const module = name === 'containers' ? 'docker' : name
        return this.portal.initialized && this.portal.serviceAvailability[module]
            && (!this.statusKnown || this.statuses.some(status => status.name === name && status.available))
    }

    get sources() {
        return SOURCES.filter(source => this.sourceAvailable(source.name)).map(source => {
            const status = this.statuses.find(s => s.name === source.name)
            return {
                ...source,
                url: absUrl(`metrics/${source.name}`),
                permitted: this.portal.hasPerm(`GET /api/metrics/${source.name}`),
                status,
                enabled: this.sourceEnabled(source.name, status),
                local: this.isLocal(source.name),
                controllable: this.isLocal(source.name) ? this.canEditConfig : !!status && this.canToggle(status),
                state: this.sourceState(status),
            }
        })
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
        return status.scrapable
            ? { label: '可抓取', color: 'text-emerald-600 font-medium' }
            : { label: '不可抓取', color: 'text-red-500 font-medium' }
    }

    @Watch('portal.initialized')
    onInitialized(initialized: boolean) {
        if (initialized) {
            this.load()
        }
    }

    @Watch('config.loading')
    onConfigLoaded(loading: boolean) {
        if (!loading && this.config.loaded && !this.config.saving) {
            this.load()
        }
    }

    @Watch('config.saving')
    onConfigSaved(saving: boolean) {
        if (!saving && this.config.loaded) {
            this.load()
        }
    }

    async load() {
        if (!this.portal.initialized || !this.canReadStatus || this.destroyed || this.config.saving) {
            return
        }
        if (this.loading) {
            this.refreshPending = true
            return
        }
        this.refreshPending = false
        this.controller = new AbortController()
        this.loading = true
        try {
            const response = await api.metricsStatus(this.controller.signal)
            if (!this.destroyed) {
                this.statuses = response.payload || []
                this.statusLoaded = true
                this.statusFailed = false
            }
        } catch {
            if (!this.destroyed) {
                this.statusFailed = true
            }
        } finally {
            this.loading = false
            if (this.refreshPending && !this.destroyed) {
                this.load()
            }
        }
    }

    canToggle(status: MetricsSourceStatus) {
        return !this.isLocal(status.name) && this.statusKnown && this.canUpdate && !this.destroyed && !this.loading
            && !this.config.saving && !status.enabledUnknown && !this.busy[status.name]
            && status.available && this.sourceAvailable(status.name)
    }

    async updateSource(name: MetricsSourceName, enabled: boolean) {
        if (this.isLocal(name)) {
            if (this.canEditConfig && this.sourceAvailable(name)) {
                if (name === 'host') {
                    this.config.draft.metrics.hostEnabled = enabled
                } else {
                    this.config.draft.metrics.containersEnabled = enabled
                }
            }
            return
        }
        const status = this.statuses.find(s => s.name === name)
        if (!status || status.enabled === enabled || !this.canToggle(status)) {
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

    mounted() {
        this.load()
    }

    unmounted() {
        this.destroyed = true
        this.controller?.abort()
    }
}

export default toNative(ConfigMetrics)
</script>

<template>
  <div class="max-w-4xl space-y-6">
    <section v-if="canReadConfig" class="space-y-4" :aria-label="$t('系统与容器监控采集')">
      <div class="config-section-heading">
        <div>
          <h2 class="config-section-title">{{ $t('监控日志') }}</h2>
          <p class="config-section-description">{{ $t('系统与容器监控采集') }}</p>
        </div>
      </div>
      <div>
        <label for="monitor-interval" class="form-label">{{ $t('监控采集间隔') }}</label>
        <select id="monitor-interval" v-model.number="config.draft.monitor.interval" class="input" :disabled="!canEditConfig">
          <option :value="0">{{ $t('禁用') }}</option>
          <option :value="5">{{ $t('5 秒') }}</option>
          <option :value="15">{{ $t('15 秒') }}</option>
          <option :value="30">{{ $t('30 秒') }}</option>
          <option :value="60">{{ $t('60 秒') }}</option>
        </select>
        <p class="mt-1 text-xs text-slate-400">{{ $t('系统与容器监控数据的采集频率，禁用后停止历史采集与资源阈值告警；应用故障告警仍可独立运行') }}</p>
      </div>
    </section>

    <section class="space-y-4" :aria-label="$t('Prometheus 指标出口')">
      <div class="config-section-heading flex items-start justify-between gap-3">
        <div class="min-w-0">
          <h2 class="config-section-title">{{ $t('Prometheus 指标出口') }}</h2>
          <p class="config-section-description">{{ $t('主机、容器与网关的指标出口与开关') }}</p>
        </div>
        <button type="button" class="btn btn-secondary flex-shrink-0" :title="$t('刷新')" :aria-label="$t('刷新')" :disabled="!canReadStatus || loading || config.saving" :aria-busy="loading" @click="load()">
          <i :class="loading ? 'fas fa-spinner fa-spin' : 'fas fa-rotate'"></i>
          <span class="hidden sm:inline">{{ $t('刷新') }}</span>
        </button>
      </div>
      <div class="grid grid-cols-1 gap-3" :aria-label="$t('指标来源')">
        <ToggleCard
          v-for="source in sources"
          :key="source.name"
          :model-value="source.enabled"
          always-expanded
          :label="$t(source.label)"
          :disabled="!source.controllable"
          :aria-busy="source.local ? config.saving : busy[source.name]"
          @update:model-value="updateSource(source.name, $event)"
        >
          <template #label>
            <div class="inline-info flex-wrap">
              <span class="text-sm text-slate-600">{{ $t(source.label) }}</span>
              <span class="text-xs" :class="source.state.color">{{ $t('状态') }}: {{ $t(source.state.label) }}</span>
              <i v-if="source.local ? config.saving : busy[source.name]" class="fas fa-spinner fa-spin text-xs text-slate-400" aria-hidden="true"></i>
            </div>
          </template>
          <template #desc>
            {{ $t(source.desc) }} · {{ $t(source.local ? '保存后生效' : '立即生效') }}
          </template>
          <template v-if="!source.controllable && !(source.local && canReadConfig)" #control></template>
          <div class="space-y-3">
            <p v-if="source.status?.error" class="text-xs text-red-500 break-all">{{ source.status.error }}</p>
            <div class="flex items-start gap-2">
              <code class="detail-value-mono flex-1 min-w-0">{{ source.url }}</code>
              <button type="button" class="btn-icon-slate flex-shrink-0" :title="$t('复制地址')" :aria-label="$t('复制地址')" :disabled="!source.permitted" @click="copyEndpoint(source.url)">
                <i class="fas fa-copy"></i>
              </button>
            </div>
            <p v-if="!source.permitted" class="text-xs text-amber-600">{{ $t('未授予此出口的抓取权限') }}</p>
          </div>
        </ToggleCard>
      </div>
    </section>
  </div>
</template>
