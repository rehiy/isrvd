<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { DockerInfo } from '@/service/types'

@Component
class DockerOverview extends Vue {
    portal = usePortal()
    @Prop({ default: true }) readonly showContainerCounts!: boolean
    @Prop({ default: false }) readonly compact!: boolean

    // ─── 数据属性 ───
    info: DockerInfo | null = null
    loading = false

    readonly statCards: { key: keyof DockerInfo; label: string; icon: string }[] = [
        { key: 'containersRunning', label: '运行中容器', icon: 'fa-play' },
        { key: 'containersStopped', label: '已停止容器', icon: 'fa-stop' },
        { key: 'containersPaused', label: '已暂停容器', icon: 'fa-pause' },
        { key: 'imagesTotal', label: '镜像', icon: 'fa-compact-disc' },
        { key: 'networksTotal', label: '网络', icon: 'fa-network-wired' },
        { key: 'volumesTotal', label: '数据卷', icon: 'fa-database' },
        { key: 'registryMirrors', label: '镜像源', icon: 'fa-gauge-high' },
    ]

    get visibleStatCards() {
        return this.statCards.filter(card => this.showContainerCounts || !card.key.startsWith('containers'))
    }

    // ─── 方法 ───
    async load() {
        this.loading = true
        try {
            if (!this.portal.hasPerm('GET /api/docker/info')) {
                this.info = null
                return
            }
            const payload = (await api.dockerInfo()).payload
            this.info = payload && Object.keys(payload).length ? { ...payload } : null
        } catch {
            this.info = null
        } finally {
            this.loading = false
        }
    }

    cardValue(key: keyof DockerInfo): string | number {
        if (!this.info) {
            return 0
        }
        if (key === 'registryMirrors') {
            return this.info.registryMirrors?.length ?? 0
        }
        return this.info[key] as number
    }

    // ─── 生命周期 ───
    mounted() {
        this.load()
    }
}

export default toNative(DockerOverview)
</script>

<template>
  <div>
    <!-- Loading -->
    <div v-if="loading" :class="compact ? 'flex items-center gap-2 py-1' : 'overview-loading'">
      <div class="spinner-md"></div>
      <span class="text-slate-400 text-sm">{{ $t('加载中...') }}</span>
    </div>

    <dl v-else-if="info && compact" class="grid grid-cols-2 gap-x-6 gap-y-3 sm:grid-cols-4">
      <div v-for="card in visibleStatCards" :key="card.key" class="flex min-w-0 items-baseline justify-between gap-2">
        <dt class="inline-flex min-w-0 items-center gap-2 text-xs text-slate-500">
          <i :class="['fas text-slate-400', card.icon]" aria-hidden="true"></i>{{ $t(card.label) }}
        </dt>
        <dd class="text-sm font-semibold tabular-nums text-slate-700">{{ cardValue(card.key) }}</dd>
      </div>
    </dl>

    <!-- Stats Grid -->
    <div v-else-if="info" class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
      <div v-for="card in visibleStatCards" :key="card.key" class="stat-card min-w-0">
        <div class="page-icon resource-service-icon">
          <i :class="['fas', card.icon]"></i>
        </div>
        <div class="min-w-0 flex-1">
          <p class="resource-service-label">{{ $t(card.label) }}</p>
          <p class="resource-service-value" :title="String(cardValue(card.key))">{{ cardValue(card.key) }}</p>
        </div>
      </div>
    </div>

    <p v-else-if="compact" class="text-xs text-amber-700" :title="$t('请确认 Docker 服务是否正常运行')">{{ $t('无法获取 Docker 信息') }}</p>

    <!-- Empty/Error State -->
    <div v-else class="overview-unavailable">
      <i class="fab fa-docker text-2xl text-slate-300"></i>
      <div>
        <p class="text-sm font-medium text-slate-600">{{ $t('无法获取 Docker 信息') }}</p>
        <p class="text-xs text-slate-400">{{ $t('请确认 Docker 服务是否正常运行') }}</p>
      </div>
    </div>
  </div>
</template>
