<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { DockerContainerInfo } from '@/service/types'

import { containerStateTextClass } from '@/helper/docker'
import { POLL_INTERVAL } from '@/helper/format'

const stateLabels = new Map([
    ['running', '运行中'], ['exited', '已退出'], ['paused', '已暂停'], ['created', '已创建'],
    ['restarting', '重启中'], ['removing', '删除中'], ['dead', '已终止']
])

@Component({ emits: ['self-container'] })
class WorkloadsOverview extends Vue {
    portal = usePortal()
    containers: DockerContainerInfo[] | null = null
    filter: 'all' | 'running' | 'inactive' = 'all'
    loading = false
    failed = false
    private timer: ReturnType<typeof setInterval> | null = null
    private readonly lifetime = new AbortController()

    get filters() {
        const total = this.containers?.length ?? 0
        const running = this.containers?.filter(container => container.state === 'running').length ?? 0
        return [
            { key: 'all', label: '全部', count: total },
            { key: 'running', label: '运行中', count: running },
            { key: 'inactive', label: '未运行', count: total - running }
        ] as const
    }

    get filteredContainers() {
        return (this.containers ?? []).filter(container =>
            this.filter === 'all' || (this.filter === 'running') === (container.state === 'running')
        )
    }

    get visibleContainers() {
        return this.filteredContainers.slice(0, 5).map(container => {
            const label = stateLabels.get(container.state)
            return {
                ...container,
                displayName: container.name || container.id.slice(0, 12),
                path: this.canInspect ? '/docker/container/' + container.id : undefined,
                stateLabel: label ? this.$t(label) : container.state || '—',
                stateClass: containerStateTextClass(container.state),
                iconClass: container.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400'
            }
        })
    }

    get canInspect() {
        return this.portal.hasPerm('GET /api/docker/container/:id')
    }

    async load() {
        if (this.loading) {
            return
        }
        const { signal } = this.lifetime
        this.loading = true
        let containers: DockerContainerInfo[]
        try {
            const { payload } = await api.dockerContainerList(true, undefined, { signal })
            containers = payload ?? []
        } catch {
            this.failed = !signal.aborted
            return
        } finally {
            this.loading = false
        }
        if (signal.aborted) {
            return
        }
        this.containers = [...containers].sort((a, b) => {
            const runningFirst = Number(b.state === 'running') - Number(a.state === 'running')
            return runningFirst || (a.name || a.id).localeCompare(b.name || b.id)
        })
        this.failed = false
        this.$emit('self-container', containers.find(container => container.isSelf) ?? null)
    }

    mounted() {
        this.load()
        this.timer = setInterval(() => {
            if (!document.hidden) {
                this.load()
            }
        }, POLL_INTERVAL * 3)
    }

    unmounted() {
        this.lifetime.abort()
        if (this.timer) {
            clearInterval(this.timer)
        }
    }
}

export default toNative(WorkloadsOverview)
</script>

<template>
  <section class="card bg-white">
    <div class="flex flex-wrap items-center gap-x-4 gap-y-3 p-4 sm:px-5 border-b border-slate-200/70">
      <div class="title-group mr-auto">
        <div class="page-icon bg-primary-50 text-primary-600"><i class="fab fa-docker"></i></div>
        <h2 class="text-base font-semibold text-slate-900">{{ $t('容器负载') }}</h2>
      </div>
      <div v-if="containers" class="order-1 w-full md:order-0 md:w-auto">
        <div class="tab-group inline-flex max-w-full">
          <button
            v-for="item in filters" :key="item.key" type="button"
            :class="['tab-btn whitespace-nowrap', filter === item.key ? 'tab-btn-active text-primary-600' : 'tab-btn-inactive']"
            :aria-pressed="filter === item.key"
            @click="filter = item.key"
          >
            {{ $t(item.label) }} <span class="tabular-nums">{{ item.count }}</span>
          </button>
        </div>
      </div>
      <router-link to="/docker/containers" class="btn btn-secondary">
        {{ $t('容器管理') }}<i class="fas fa-arrow-right text-xs"></i>
      </router-link>
    </div>
    <p v-if="failed" class="px-4 py-3 sm:px-5 text-sm text-amber-700">{{ $t('获取容器列表失败') }}</p>
    <div v-if="!containers && !failed" class="overview-loading">
      <span class="spinner-md"></span><span class="text-sm text-slate-500">{{ $t('加载中...') }}</span>
    </div>
    <div v-else-if="containers && !visibleContainers.length" class="px-4 py-6 text-center text-sm text-slate-500">
      {{ $t(containers.length ? '未找到匹配容器' : '暂无容器') }}
    </div>
    <template v-else-if="visibleContainers.length">
      <div class="card-table hidden md:block">
        <table class="w-full table-fixed">
          <thead class="bg-slate-50">
            <tr>
              <th scope="col" class="th w-[32%]">{{ $t('名称') }}</th>
              <th scope="col" class="th w-[27%]">{{ $t('镜像') }}</th>
              <th scope="col" class="th w-[17%]">{{ $t('状态') }}</th>
              <th scope="col" class="th">{{ $t('网络') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-for="container in visibleContainers" :key="container.id" class="hover:bg-slate-50/65 transition-colors">
              <td class="px-4 py-2 max-w-[280px] sm:px-5">
                <div class="inline-info">
                  <span :class="['row-icon shrink-0', container.iconClass]"><i class="fas fa-cube text-white text-xs"></i></span>
                  <component :is="container.path ? 'RouterLink' : 'span'" :to="container.path" class="item-title-sm min-w-0" :class="{ 'hover:text-primary-600': container.path }" :title="container.displayName">
                    {{ container.displayName }}
                  </component>
                </div>
              </td>
              <td class="px-4 py-2 text-sm font-mono text-slate-600 break-all">{{ container.image || '—' }}</td>
              <td class="px-4 py-2">
                <span :class="['inline-flex items-center gap-1.5 whitespace-nowrap text-sm', container.stateClass]" :title="container.status">
                  <span class="size-1.5 rounded-full bg-current"></span>{{ container.stateLabel }}
                </span>
              </td>
              <td class="px-4 py-2 text-sm text-slate-600 break-all">
                <i v-if="container.networks?.length" class="fas fa-network-wired mr-1.5 text-slate-400"></i>{{ container.networks?.join(', ') || '—' }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="card-body md:hidden space-y-3">
        <div v-for="container in visibleContainers" :key="container.id" class="card-interactive">
          <div class="card-info-row">
            <div :class="['list-icon', container.iconClass]"><i class="fas fa-cube text-white text-base"></i></div>
            <div class="min-w-0">
              <component :is="container.path ? 'RouterLink' : 'span'" :to="container.path" class="item-title-sm" :class="{ 'hover:text-primary-600': container.path }" :title="container.displayName">
                {{ container.displayName }}
              </component>
              <span v-if="container.image" class="item-subtitle">{{ container.image }}</span>
              <span :class="['inline-flex items-center gap-1.5 mt-1 text-xs', container.stateClass]">
                <span class="size-1.5 rounded-full bg-current"></span>{{ container.stateLabel }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </template>
    <p v-if="filteredContainers.length > visibleContainers.length" class="border-t border-slate-200/70 px-4 py-2.5 sm:px-5 text-xs text-slate-500">
      {{ $t('显示 {0} / {1} 个容器', visibleContainers.length, filteredContainers.length) }}
    </p>
  </section>
</template>
