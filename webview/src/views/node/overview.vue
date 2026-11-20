<script lang="ts">
import { Component, Vue, Watch, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { NodeOverviewInfo, NodeOverviewItem } from '@/service/types'

import { POLL_INTERVAL, formatFileSize } from '@/helper/format'
import { NODE_LIST_PERM, nodeStatus, switchNode } from '@/helper/node'

import PageSearch from '@/component/page-search.vue'

@Component({
    components: { PageSearch }
})
class NodeOverview extends Vue {
    portal = usePortal()
    snapshot: NodeOverviewInfo | null = null
    searchText = ''
    loading = false
    loadFailed = false
    private pollTimer: ReturnType<typeof setInterval> | null = null

    get ready() {
        return this.portal.initialized
    }

    get nodes() {
        const keyword = this.searchText.trim().toLowerCase()
        const nodes = this.snapshot?.nodes || []
        const filtered = keyword
            ? nodes.filter(node => [node.name, node.hostname, node.os, node.arch, node.agentVersion, this.$t(nodeStatus(node).text)]
                .some(value => (value || '').toLowerCase().includes(keyword)))
            : nodes
        // 优先列出已激活但离线或采集失败的节点。
        const priority = (node: NodeOverviewItem) => {
            if (node.status !== 'active') {
                return 2
            }
            return !node.online || node.monitorError ? 0 : 1
        }
        return [...filtered].sort((a, b) => priority(a) - priority(b) || a.name.localeCompare(b.name))
    }

    get offlineCount() {
        return this.snapshot?.nodes.filter(node => node.status === 'active' && !node.online).length ?? 0
    }

    get monitorErrorCount() {
        return this.snapshot?.nodes.filter(node => node.online && node.monitorError).length ?? 0
    }

    @Watch('ready', { immediate: true })
    onReadyChange(ready: boolean) {
        if (!ready || this.pollTimer) {
            return
        }
        if (!this.portal.hasPerm(NODE_LIST_PERM) || !this.portal.hasPerm('GET /api/node/overview')) {
            this.$router.replace('/overview')
            return
        }
        this.load()
        this.pollTimer = setInterval(() => this.load(true), POLL_INTERVAL * 6)
    }

    async load(silent = false) {
        if (this.loading) {
            return
        }
        if (!silent || !this.snapshot) {
            this.loading = true
        }
        try {
            const response = await api.nodeOverviewStats()
            if (response.payload) {
                this.snapshot = response.payload
                this.loadFailed = false
            } else {
                this.loadFailed = true
            }
        } catch {
            this.loadFailed = true
        } finally {
            this.loading = false
        }
    }

    statusText(node: NodeOverviewItem) {
        return this.$t(nodeStatus(node).text)
    }

    statusClass(node: NodeOverviewItem) {
        return nodeStatus(node).cls
    }

    percent(used: number, total: number) {
        return total > 0 ? (used / total * 100).toFixed(0) + '%' : '-'
    }

    cpuText(node: NodeOverviewItem) {
        return node.system?.cpuPercent == null ? '-' : node.system.cpuPercent.toFixed(0) + '%'
    }

    memoryText(node: NodeOverviewItem) {
        return node.system ? this.percent(node.system.memoryUsed, node.system.memoryTotal) : '-'
    }

    diskText(node: NodeOverviewItem) {
        return node.system ? this.percent(node.system.diskUsed, node.system.diskTotal) : '-'
    }

    memoryHint(node: NodeOverviewItem) {
        return node.system ? formatFileSize(node.system.memoryUsed) + ' / ' + formatFileSize(node.system.memoryTotal) : ''
    }

    dockerText(node: NodeOverviewItem) {
        return node.docker ? node.docker.containersRunning + ' / ' + node.docker.containersStopped : '-'
    }

    swarmText(node: NodeOverviewItem) {
        return node.swarm ? String(node.swarm.services) : '-'
    }

    openNode(node: NodeOverviewItem) {
        if (node.online) {
            switchNode(node.id, '#/overview')
        }
    }

    unmounted() {
        if (this.pollTimer) {
            clearInterval(this.pollTimer)
            this.pollTimer = null
        }
    }
}

export default toNative(NodeOverview)
</script>

<template>
  <div class="page">
    <div class="page-toolbar">
      <div class="toolbar-desktop">
        <div class="title-group-static">
          <div class="page-icon bg-orange-500">
            <i class="fas fa-network-wired text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="title-text">{{ $t('全局概览') }}</h1>
            <p class="text-xs text-slate-500 truncate">{{ $t('集中查看受管节点状态与资源使用情况') }}</p>
          </div>
        </div>
        <div class="action-group">
          <PageSearch v-model="searchText" search-key="node-overview" :placeholder="$t('搜索节点名、主机名或系统...')" focus-color="orange" type-to-search />
          <button class="btn btn-secondary" :disabled="loading" @click="load()">
            <i :class="loading ? 'fas fa-spinner fa-spin' : 'fas fa-rotate'"></i>{{ $t('刷新') }}
          </button>
        </div>
      </div>
      <div class="toolbar-mobile">
        <div class="title-group">
          <div class="page-icon bg-orange-500">
            <i class="fas fa-network-wired text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="title-text">{{ $t('全局概览') }}</h1>
            <p class="text-xs text-slate-500 truncate">{{ $t('受管节点状态与资源概况') }}</p>
          </div>
        </div>
        <button class="btn btn-secondary btn-square" :disabled="loading" :title="$t('刷新')" @click="load()">
          <i :class="loading ? 'fas fa-spinner fa-spin' : 'fas fa-rotate'"></i>
        </button>
      </div>
    </div>

    <div class="mobile-search">
      <PageSearch v-model="searchText" search-key="node-overview" :placeholder="$t('搜索节点名、主机名或系统...')" width-class="w-full" focus-color="orange" />
    </div>

    <div v-if="loading && !snapshot" class="overview-loading">
      <div class="spinner-md"></div>
      <span class="text-slate-400 text-sm">{{ $t('加载中...') }}</span>
    </div>
    <div v-else-if="!snapshot" class="overview-unavailable">
      <i class="fas fa-triangle-exclamation text-2xl text-slate-300"></i>
      <span class="text-sm text-slate-500">{{ $t('加载失败，请刷新重试') }}</span>
    </div>
    <template v-else>
      <div v-if="loadFailed" class="mx-4 mt-3 text-sm text-amber-600" role="status">{{ $t('刷新失败，当前展示上次采集结果') }}</div>
      <div class="p-4 space-y-4">
        <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
          <div v-for="card in [
            { label: '节点总数', value: snapshot.nodes.length, icon: 'fa-server', color: 'bg-blue-500' },
            { label: '在线节点', value: snapshot.online, icon: 'fa-circle-check', color: 'bg-emerald-500' },
            { label: '离线节点', value: offlineCount, icon: 'fa-link-slash', color: 'bg-red-500' },
            { label: '指标采集失败', value: monitorErrorCount, icon: 'fa-triangle-exclamation', color: 'bg-amber-500' }
          ]" :key="card.label" class="stat-card items-center">
            <div :class="['page-icon', card.color]">
              <i :class="['fas', card.icon, 'text-white']"></i>
            </div>
            <div class="min-w-0 flex-1">
              <p class="text-xs text-slate-400 mb-0.5">{{ $t(card.label) }}</p>
              <p class="font-semibold text-slate-800">{{ card.value }}</p>
            </div>
          </div>
        </div>
        <p class="text-xs text-slate-500">
          {{ $t('仅统计受管节点；数据按需采集，每 30 秒刷新。Docker 为当前节点容器数；Swarm 服务数为集群口径，多个节点可能重复。') }}
        </p>
      </div>

      <div v-if="snapshot.nodes.length === 0" class="card-body">
        <div class="empty-state">
          <div class="empty-state-icon">
            <i class="fas fa-server text-4xl text-slate-300"></i>
          </div>
          <p class="text-sm text-slate-500">{{ $t('暂无受管节点') }}</p>
        </div>
      </div>
      <div v-else-if="nodes.length === 0" class="card-body">
        <div class="empty-state">
          <p class="text-sm text-slate-500">{{ $t('未找到匹配节点') }}</p>
        </div>
      </div>
      <template v-else>
        <div class="card-table hidden md:block">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-100 border-b border-slate-200">
                <th class="th">{{ $t('节点') }}</th>
                <th class="th">{{ $t('状态') }}</th>
                <th class="th">{{ $t('CPU') }}</th>
                <th class="th">{{ $t('内存') }}</th>
                <th class="th">{{ $t('磁盘') }}</th>
                <th class="th">{{ $t('容器运行/停止') }}</th>
                <th class="th">{{ $t('Swarm 服务数') }}</th>
                <th class="th-right">{{ $t('操作') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-100">
              <tr v-for="node in nodes" :key="node.id" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3 max-w-[250px]">
                  <div class="inline-info">
                    <div class="row-icon bg-orange-400"><i class="fas fa-server text-white text-sm"></i></div>
                    <div class="min-w-0">
                      <span class="item-title">{{ node.name }}</span>
                      <span class="item-subtitle">{{ node.hostname || node.id }}</span>
                    </div>
                  </div>
                </td>
                <td class="td-text">
                  <span :class="statusClass(node)">{{ statusText(node) }}</span>
                  <span v-if="node.online && node.monitorError" class="block text-xs text-amber-600">{{ $t('指标采集失败') }}</span>
                </td>
                <td class="td-text">{{ cpuText(node) }}</td>
                <td class="td-text" :title="memoryHint(node)">{{ memoryText(node) }}</td>
                <td class="td-text">{{ diskText(node) }}</td>
                <td class="td-text">{{ dockerText(node) }}</td>
                <td class="td-text">{{ swarmText(node) }}</td>
                <td class="px-4 py-3 text-right">
                  <button v-if="node.online" class="btn-icon btn-icon-blue" :title="$t('查看节点')" @click="openNode(node)">
                    <i class="fas fa-arrow-up-right-from-square"></i>
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="card-body md:hidden space-y-3">
          <div v-for="node in nodes" :key="node.id" class="card-interactive">
            <div class="card-info-row">
              <div class="list-icon bg-orange-400"><i class="fas fa-server text-white"></i></div>
              <div class="min-w-0">
                <span class="item-title-sm">{{ node.name }}</span>
                <span class="item-subtitle">{{ node.hostname || node.id }}</span>
              </div>
            </div>
            <div class="card-prop-row">
              <span class="text-xs text-slate-400">{{ $t('状态') }}</span>
              <span :class="statusClass(node)">{{ statusText(node) }}</span>
            </div>
            <div v-if="node.online && node.monitorError" class="text-xs text-amber-600">{{ $t('指标采集失败') }}</div>
            <div class="card-prop-row"><span class="text-xs text-slate-400">CPU</span><span class="text-xs text-slate-600">{{ cpuText(node) }}</span></div>
            <div class="card-prop-row"><span class="text-xs text-slate-400">{{ $t('内存') }}</span><span class="text-xs text-slate-600">{{ memoryText(node) }}</span></div>
            <div class="card-prop-row"><span class="text-xs text-slate-400">{{ $t('磁盘') }}</span><span class="text-xs text-slate-600">{{ diskText(node) }}</span></div>
            <div class="card-prop-row"><span class="text-xs text-slate-400">{{ $t('容器运行/停止') }}</span><span class="text-xs text-slate-600">{{ dockerText(node) }}</span></div>
            <div class="card-prop-row"><span class="text-xs text-slate-400">{{ $t('Swarm 服务数') }}</span><span class="text-xs text-slate-600">{{ swarmText(node) }}</span></div>
            <button v-if="node.online" class="btn btn-secondary w-full" @click="openNode(node)">{{ $t('查看节点') }}</button>
          </div>
        </div>
      </template>
    </template>
  </div>
</template>
