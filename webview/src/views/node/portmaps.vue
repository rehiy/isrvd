<script lang="ts">
import { Component, Ref, Vue, Watch, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { NodePortMapInfo, NodeInfo } from '@/service/types'

import { POLL_INTERVAL } from '@/helper/format'

import PageSearch from '@/component/page-search.vue'

import NodePortMapModal from './widget/node-portmap-modal.vue'
import PortMapActions from './widget/portmap-actions.vue'

@Component({
    components: { PageSearch, PortMapActions, NodePortMapModal }
})
class NodePortMaps extends Vue {
    portal = usePortal()
    @Ref readonly portmapModalRef!: InstanceType<typeof NodePortMapModal>

    // ─── 数据属性 ───
    nodes: NodeInfo[] = []
    loading = true
    searchText = ''
    private pollTimer: ReturnType<typeof setInterval> | null = null
    private polling = false

    // ─── 计算属性 ───
    // 把各节点的规则展开成一张表；已吊销的节点不会保留规则
    get items(): NodePortMapInfo[] {
        return this.nodes.flatMap(node =>
            (node.portmaps || []).map(f => ({
                ...f,
                nodeId: node.id,
                nodeName: node.name,
                nodeOnline: node.online,
                nodeStatus: node.status
            }))
        )
    }

    get filteredItems() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.items
        return this.items.filter(item =>
            [String(item.listen), item.proto, item.target, item.name, item.nodeName]
                .some(value => (value || '').toLowerCase().includes(keyword))
        )
    }

    get canUpdate() {
        return this.portal.hasPerm('PUT /api/node/:id/portmap')
    }

    // 启动数据（探活与权限）加载完成后才能判断节点管理是否启用
    get ready() {
        return this.portal.initialized
    }

    // ─── 监听器 ───
    @Watch('ready', { immediate: true })
    onReadyChange(ready: boolean) {
        if (!ready) return
        // 未启用中控模式或无节点列表权限：直接用地址访问时回到概览，而不是展示一个只会报错的页面
        if (!this.portal.hasPerm('GET /api/node/nodes')) {
            this.$router.replace('/overview')
            return
        }
        this.load()
        this.startPoll()
    }

    // ─── 展示辅助 ───
    nodeStatusText(item: NodePortMapInfo) {
        if (item.nodeStatus === 'pending') return this.$t('待接入')
        return item.nodeOnline ? this.$t('在线') : this.$t('离线')
    }

    nodeStatusClass(item: NodePortMapInfo) {
        if (item.nodeStatus === 'pending') return 'text-slate-500'
        return item.nodeOnline ? 'text-emerald-600 font-medium' : 'text-red-500 font-medium'
    }

    itemKey(item: NodePortMapInfo) {
        return `${item.nodeId}/${item.proto}/${item.listen}`
    }

    // ─── 数据加载 ───
    async load(silent = false) {
        if (this.polling) return
        this.polling = true
        if (!silent) this.loading = true
        try {
            const res = await api.nodeList()
            this.nodes = res.payload || []
        } catch {} finally {
            this.loading = false
            this.polling = false
        }
    }

    startPoll() {
        if (this.pollTimer) return
        this.pollTimer = setInterval(() => this.load(true), POLL_INTERVAL)
    }

    stopPoll() {
        if (!this.pollTimer) return
        clearInterval(this.pollTimer)
        this.pollTimer = null
    }

    // ─── 操作 ───
    openAdd() {
        if (!this.nodes.some(node => node.status !== 'revoked')) {
            this.portal.showNotification('error', this.$t('暂无可用节点，请先在「节点管理」中接入节点'))
            return
        }
        this.portmapModalRef?.show(this.nodes)
    }

    openEdit(item: NodePortMapInfo) {
        this.portmapModalRef?.show(this.nodes, item)
    }

    // ─── 生命周期 ───
    unmounted() {
        this.stopPoll()
    }
}

export default toNative(NodePortMaps)
</script>

<template>
  <div class="page">
    <div class="page-toolbar">
      <!-- 桌面端 -->
      <div class="toolbar-desktop">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-orange-500">
            <i class="fas fa-right-left text-white"></i>
          </div>
          <div>
            <h1 class="title-text">{{ $t('端口转发') }}</h1>
            <p class="text-xs text-slate-500">{{ $t('中控监听端口，经隧道转发到节点（TCP / UDP）') }}</p>
          </div>
        </div>
        <div class="action-group">
          <PageSearch v-model="searchText" search-key="node-portmaps" :placeholder="$t('搜索监听端口、目标、节点或备注...')" focus-color="orange" type-to-search />
          <button class="btn btn-secondary" @click="load()">
            <i class="fas fa-rotate"></i>{{ $t('刷新') }}
          </button>
          <button v-if="canUpdate" class="btn btn-orange" @click="openAdd">
            <i class="fas fa-plus"></i>{{ $t('添加转发') }}
          </button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="toolbar-mobile">
        <div class="title-group">
          <div class="page-icon bg-orange-500">
            <i class="fas fa-right-left text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="title-text">{{ $t('端口转发') }}</h1>
            <p class="text-xs text-slate-500 truncate">{{ $t('节点端口转发') }}</p>
          </div>
        </div>
        <div class="action-group-sm">
          <button class="btn btn-secondary btn-square" :title="$t('刷新')" @click="load()">
            <i class="fas fa-rotate text-sm"></i>
          </button>
          <button v-if="canUpdate" class="btn btn-orange btn-square" :title="$t('添加转发')" @click="openAdd">
            <i class="fas fa-plus text-sm"></i>
          </button>
        </div>
      </div>
    </div>

    <!-- 移动端搜索 -->
    <div class="mobile-search">
      <PageSearch v-model="searchText" search-key="node-portmaps" :placeholder="$t('搜索监听端口、目标、节点或备注...')" width-class="w-full" focus-color="orange" />
    </div>

    <!-- Loading -->
    <div v-if="loading" class="card-body">
      <div class="empty-state">
        <div class="spinner-lg"></div>
        <p class="text-slate-500">{{ $t('加载中...') }}</p>
      </div>
    </div>

    <div v-else-if="filteredItems.length === 0" class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-right-left text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">{{ items.length === 0 ? $t('暂无转发规则') : $t('未找到匹配规则') }}</p>
        <p class="text-sm text-slate-400">{{ items.length === 0 ? $t('点击右上角「添加转发」，把节点端口转发到中控') : $t('尝试更换关键词或清空搜索条件') }}</p>
      </div>
    </div>

    <template v-else>
      <!-- 桌面端表格 -->
      <div class="card-table hidden md:block">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-100 border-b border-slate-200">
              <th class="th">{{ $t('监听端口') }}</th>
              <th class="w-24 th">{{ $t('协议') }}</th>
              <th class="th">{{ $t('节点') }}</th>
              <th class="th">{{ $t('目标地址') }}</th>
              <th class="w-32 th-right">{{ $t('操作') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-for="item in filteredItems" :key="itemKey(item)" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3 max-w-[280px]">
                <div class="inline-info">
                  <div class="row-icon bg-orange-400">
                    <i class="fas fa-right-left text-white text-sm"></i>
                  </div>
                  <div class="min-w-0">
                    <span class="item-title font-mono">{{ item.listen }}</span>
                    <span v-if="item.name" class="item-subtitle">{{ item.name }}</span>
                  </div>
                </div>
              </td>
              <td class="td-text">
                <span class="badge-sm bg-orange-50 text-orange-700">{{ item.proto.toUpperCase() }}</span>
              </td>
              <td class="px-4 py-3 max-w-[240px]">
                <span class="text-sm text-slate-600 truncate block">{{ item.nodeName }}</span>
                <span class="text-xs mt-0.5 block" :class="nodeStatusClass(item)">{{ nodeStatusText(item) }}</span>
              </td>
              <td class="td-text font-mono">{{ item.target }}</td>
              <td class="px-4 py-3">
                <PortMapActions :item="item" compact @edit="openEdit" @success="load(true)" />
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 移动端卡片 -->
      <div class="card-body md:hidden space-y-3">
        <div v-for="item in filteredItems" :key="itemKey(item)" class="card-interactive">
          <div class="card-info-row">
            <div class="list-icon bg-orange-400 flex-shrink-0">
              <i class="fas fa-right-left text-white text-base"></i>
            </div>
            <div class="min-w-0">
              <span class="item-title-sm font-mono">{{ item.listen }}</span>
              <span v-if="item.name" class="item-subtitle">{{ item.name }}</span>
            </div>
          </div>

          <div class="card-prop-row-start">
            <span class="prop-label-start">{{ $t('协议') }}</span>
            <span class="badge-sm bg-orange-50 text-orange-700">{{ item.proto.toUpperCase() }}</span>
          </div>
          <div class="card-prop-row">
            <span class="text-xs text-slate-400 flex-shrink-0">{{ $t('节点') }}</span>
            <span class="text-xs text-slate-500">{{ item.nodeName }}</span>
            <span class="text-xs" :class="nodeStatusClass(item)">{{ nodeStatusText(item) }}</span>
          </div>
          <div class="card-prop-row-start">
            <span class="prop-label-start">{{ $t('目标地址') }}</span>
            <code class="text-xs bg-slate-100 px-2 py-0.5 rounded-lg break-all">{{ item.target }}</code>
          </div>

          <PortMapActions :item="item" @edit="openEdit" @success="load(true)" />
        </div>
      </div>
    </template>
  </div>

  <NodePortMapModal ref="portmapModalRef" @success="load(true)" />
</template>
