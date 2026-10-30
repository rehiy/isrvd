<script lang="ts">
import { Component, Ref, Vue, Watch, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import type { NodeInfo } from '@/service/types'

import { enterNodePage, nodeMissingFeatures, nodeStatus } from '@/helper/node'

import PageSearch from '@/component/page-search.vue'

import NodeActions from './widget/node-actions.vue'
import NodeCreateModal from './widget/node-create-modal.vue'
import NodeEditModal from './widget/node-edit-modal.vue'

@Component({
    components: { PageSearch, NodeActions, NodeCreateModal, NodeEditModal }
})
class NodeManage extends Vue {
    portal = usePortal()
    @Ref readonly editModalRef!: InstanceType<typeof NodeEditModal>
    @Ref readonly createModalRef!: InstanceType<typeof NodeCreateModal>

    // ─── 数据属性 ───
    searchText = ''
    private stopPoll: (() => void) | null = null

    // ─── 计算属性 ───
    get filteredNodes() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.portal.nodes
        return this.portal.nodes.filter(node =>
            [node.name, node.hostname, node.id, node.os, node.arch, node.agentVersion, this.statusText(node)]
                .some(value => (value || '').toLowerCase().includes(keyword))
        )
    }

    // 启动数据（探活与权限）加载完成后才能判断节点管理是否启用
    get ready() {
        return this.portal.initialized
    }

    // ─── 监听器 ───
    @Watch('ready', { immediate: true })
    onReadyChange(ready: boolean) {
        if (ready) this.stopPoll ??= enterNodePage(this.portal, this.$router)
    }

    // ─── 展示辅助 ───
    statusText(node: NodeInfo) {
        return this.$t(nodeStatus(node).text)
    }

    // 协商成功但节点程序缺少部分可选能力时的提示；基础转发不受影响
    featureHint(node: NodeInfo) {
        if (node.status !== 'active' || !node.online) return ''
        const missing = nodeMissingFeatures(node).map(name => this.$t(name))
        return missing.length ? this.$t('部分功能不可用：{0}，请升级节点程序', missing.join('、')) : ''
    }

    statusClass(node: NodeInfo) {
        return nodeStatus(node).cls
    }

    platformText(node: NodeInfo) {
        return node.os ? `${node.os} / ${node.arch}` : '-'
    }

    latencyText(node: NodeInfo) {
        return node.online ? `${node.latency} ms` : '-'
    }

    // ─── 操作 ───
    openAdd() {
        this.createModalRef?.show()
    }

    openEdit(node: NodeInfo) {
        this.editModalRef?.show(node)
    }

    // ─── 生命周期 ───
    unmounted() {
        this.stopPoll?.()
    }
}

export default toNative(NodeManage)
</script>

<template>
  <div class="page">
    <div class="page-toolbar">
      <!-- 桌面端 -->
      <div class="toolbar-desktop">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-orange-500">
            <i class="fas fa-sitemap text-white"></i>
          </div>
          <div>
            <h1 class="title-text">{{ $t('节点管理') }}</h1>
            <p class="text-xs text-slate-500">{{ $t('接入受管机，并切换到节点进行管理') }}</p>
          </div>
        </div>
        <div class="action-group">
          <PageSearch v-model="searchText" search-key="node" :placeholder="$t('搜索节点名、主机名或系统...')" focus-color="orange" type-to-search />
          <button class="btn btn-secondary" @click="portal.nodeLoad()">
            <i class="fas fa-rotate"></i>{{ $t('刷新') }}
          </button>
          <button v-if="portal.hasPerm('POST /api/node')" class="btn btn-orange" @click="openAdd">
            <i class="fas fa-plus"></i>{{ $t('接入节点') }}
          </button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="toolbar-mobile">
        <div class="title-group">
          <div class="page-icon bg-orange-500">
            <i class="fas fa-sitemap text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="title-text">{{ $t('节点管理') }}</h1>
            <p class="text-xs text-slate-500 truncate">{{ $t('接入并管理受管机') }}</p>
          </div>
        </div>
        <div class="action-group-sm">
          <button class="btn btn-secondary btn-square" :title="$t('刷新')" @click="portal.nodeLoad()">
            <i class="fas fa-rotate text-sm"></i>
          </button>
          <button v-if="portal.hasPerm('POST /api/node')" class="btn btn-orange btn-square" :title="$t('接入节点')" @click="openAdd">
            <i class="fas fa-plus text-sm"></i>
          </button>
        </div>
      </div>
    </div>

    <!-- 移动端搜索 -->
    <div class="mobile-search">
      <PageSearch v-model="searchText" search-key="node" :placeholder="$t('搜索节点名、主机名或系统...')" width-class="w-full" focus-color="orange" />
    </div>

    <!-- Loading -->
    <div v-if="portal.nodesLoading" class="card-body">
      <div class="empty-state">
        <div class="spinner-lg"></div>
        <p class="text-slate-500">{{ $t('加载中...') }}</p>
      </div>
    </div>

    <div v-else-if="filteredNodes.length === 0" class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-sitemap text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">{{ portal.nodes.length === 0 ? $t('暂无受管节点') : $t('未找到匹配节点') }}</p>
        <p class="text-sm text-slate-400">{{ portal.nodes.length === 0 ? $t('点击右上角「接入节点」生成令牌，并在受管机上以 agent 模式启动') : $t('尝试更换关键词或清空搜索条件') }}</p>
      </div>
    </div>

    <template v-else>
      <!-- 桌面端表格 -->
      <div class="card-table hidden md:block">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-100 border-b border-slate-200">
              <th class="th">{{ $t('节点') }}</th>
              <th class="w-32 th">{{ $t('状态') }}</th>
              <th class="th">{{ $t('系统') }}</th>
              <th class="th">{{ $t('版本') }}</th>
              <th class="w-24 th">{{ $t('延迟') }}</th>
              <th class="w-44 th-right">{{ $t('操作') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-for="node in filteredNodes" :key="node.id" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3 max-w-[280px]">
                <div class="inline-info">
                  <div class="row-icon bg-orange-400">
                    <i class="fas fa-server text-white text-sm"></i>
                  </div>
                  <div class="min-w-0">
                    <span class="item-title">{{ node.name }}</span>
                    <span class="item-subtitle">{{ node.hostname || node.id }}</span>
                  </div>
                </div>
              </td>
              <td class="td-text">
                <span :class="statusClass(node)">{{ statusText(node) }}</span>
                <i v-if="featureHint(node)" class="fas fa-triangle-exclamation text-amber-500 text-xs ml-1.5" :title="featureHint(node)"></i>
              </td>
              <td class="td-text">{{ platformText(node) }}</td>
              <td class="td-text">{{ node.agentVersion || '-' }}</td>
              <td class="td-text">{{ latencyText(node) }}</td>
              <td class="px-4 py-3">
                <NodeActions :node="node" compact @edit="openEdit" @success="portal.nodeLoad(true)" />
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 移动端卡片 -->
      <div class="card-body md:hidden space-y-3">
        <div v-for="node in filteredNodes" :key="node.id" class="card-interactive">
          <div class="card-info-row">
            <div class="list-icon bg-orange-400 flex-shrink-0">
              <i class="fas fa-server text-white text-base"></i>
            </div>
            <div class="min-w-0">
              <span class="item-title-sm">{{ node.name }}</span>
              <span class="item-subtitle">{{ node.hostname || node.id }}</span>
            </div>
          </div>

          <div class="card-prop-row">
            <span class="text-xs text-slate-400 flex-shrink-0">{{ $t('状态') }}</span>
            <span class="text-xs" :class="statusClass(node)">{{ statusText(node) }}</span>
            <i v-if="featureHint(node)" class="fas fa-triangle-exclamation text-amber-500 text-xs ml-1.5" :title="featureHint(node)"></i>
          </div>
          <div class="card-prop-row">
            <span class="text-xs text-slate-400 flex-shrink-0">{{ $t('系统') }}</span>
            <span class="text-xs text-slate-500">{{ platformText(node) }}</span>
          </div>
          <div class="card-prop-row">
            <span class="text-xs text-slate-400 flex-shrink-0">{{ $t('版本') }}</span>
            <span class="text-xs text-slate-500">{{ node.agentVersion || '-' }}</span>
          </div>
          <div class="card-prop-row">
            <span class="text-xs text-slate-400 flex-shrink-0">{{ $t('延迟') }}</span>
            <span class="text-xs text-slate-500">{{ latencyText(node) }}</span>
          </div>

          <NodeActions :node="node" @edit="openEdit" @success="portal.nodeLoad(true)" />
        </div>
      </div>
    </template>
  </div>

  <NodeCreateModal ref="createModalRef" @success="portal.nodeLoad(true)" />
  <NodeEditModal ref="editModalRef" @success="portal.nodeLoad(true)" />
</template>
