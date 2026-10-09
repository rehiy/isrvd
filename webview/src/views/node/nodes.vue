<script lang="ts">
import { Component, Ref, Vue, Watch, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { NodeCodeInfo, NodeInfo } from '@/service/types'

import { POLL_INTERVAL, formatTime } from '@/helper/format'
import { switchNode } from '@/helper/node'

import PageSearch from '@/component/page-search.vue'

import CodeCreateModal from './widget/code-create-modal.vue'
import NodeEditModal from './widget/node-edit-modal.vue'

// 列表行：已注册的节点，或尚未使用的注册码（等待受管机接入）
interface NodeRow {
    key: string
    icon: string
    title: string
    subtitle: string
    statusText: string
    statusClass: string
    node?: NodeInfo
    code?: NodeCodeInfo
}

@Component({
    components: { PageSearch, CodeCreateModal, NodeEditModal }
})
class NodeManage extends Vue {
    portal = usePortal()
    @Ref readonly editModalRef!: InstanceType<typeof NodeEditModal>
    @Ref readonly codeModalRef!: InstanceType<typeof CodeCreateModal>

    // ─── 数据属性 ───
    nodes: NodeInfo[] = []
    codes: NodeCodeInfo[] = []
    loading = true
    searchText = ''
    private pollTimer: ReturnType<typeof setInterval> | null = null
    private polling = false

    // ─── 计算属性 ───
    // 节点在前，待接入的注册码在后
    get rows(): NodeRow[] {
        const nodeRows = this.nodes.map((node): NodeRow => ({
            key: `node-${node.id}`,
            icon: 'fa-server',
            title: node.name,
            subtitle: node.hostname || node.id,
            statusText: this.statusText(node),
            statusClass: this.statusClass(node),
            node
        }))
        const codeRows = this.codes.map((code): NodeRow => ({
            key: `code-${code.id}`,
            icon: 'fa-ticket',
            title: code.name || '未命名注册码',
            subtitle: '一次性注册码',
            statusText: '待接入',
            statusClass: 'text-slate-500',
            code
        }))
        return [...nodeRows, ...codeRows]
    }

    get filteredRows() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.rows
        return this.rows.filter((row: NodeRow) => {
            const { node, code } = row
            return [row.title, row.subtitle, row.statusText, node?.os, node?.arch, node?.agentVersion, code?.createdBy]
                .some(v => (v || '').toLowerCase().includes(keyword))
        })
    }

    // 启动数据（探活与权限）加载完成后才能判断节点管理是否启用
    get ready() {
        return this.portal.initialized
    }

    // ─── 监听器 ───
    @Watch('ready', { immediate: true })
    onReadyChange(ready: boolean) {
        if (!ready) return
        // 未启用中控模式或不是创始人：直接用地址访问时回到概览，而不是展示一个只会报错的页面
        if (!this.portal.hasPerm('GET /api/node/nodes')) {
            this.$router.replace('/overview')
            return
        }
        this.load()
        this.startPoll()
    }

    // ─── 展示辅助 ───
    statusText(node: NodeInfo) {
        if (node.status === 'pending') return '待审批'
        if (node.status === 'revoked') return '已吊销'
        if (!node.online) return '离线'
        return node.compatible ? '在线' : '版本不兼容'
    }

    statusClass(node: NodeInfo) {
        if (node.status === 'revoked') return 'text-slate-500'
        if (node.status === 'pending') return 'text-amber-600 font-medium'
        if (!node.online) return 'text-red-500 font-medium'
        return node.compatible ? 'text-emerald-600 font-medium' : 'text-amber-600 font-medium'
    }

    canOpen(node: NodeInfo) {
        return node.status === 'approved' && node.online && node.compatible
    }

    platformText(node: NodeInfo) {
        return node.os ? `${node.os} / ${node.arch}` : '-'
    }

    latencyText(node: NodeInfo) {
        return node.online ? `${node.latency} ms` : '-'
    }

    approveText(code: NodeCodeInfo) {
        return code.autoApprove ? '自动审批' : '需人工审批'
    }

    timeText(value?: string) {
        return formatTime(value)
    }

    // ─── 数据加载 ───
    async load(silent = false) {
        if (this.polling) return
        this.polling = true
        if (!silent) this.loading = true
        try {
            const [nodes, codes] = await Promise.all([api.nodeList(), api.nodeCodeList()])
            this.nodes = nodes.payload || []
            this.codes = codes.payload || []
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
    openNode(node: NodeInfo) {
        switchNode(node.id, '#/overview')
    }

    openAdd() {
        this.codeModalRef?.show()
    }

    openCode(code: NodeCodeInfo) {
        this.codeModalRef?.view(code)
    }

    openEdit(node: NodeInfo) {
        this.editModalRef?.show(node)
    }

    async handleApprove(node: NodeInfo) {
        try {
            await api.nodeApprove(node.id)
            this.portal.showNotification('success', '节点已通过审批')
            this.load(true)
        } catch {}
    }

    handleRevoke(node: NodeInfo) {
        this.portal.showConfirm({
            title: '吊销节点',
            message: `确定要吊销节点 <strong class="text-slate-900">${node.name}</strong> 吗？吊销后令牌立即失效并断开连接，需要删除后重新注册才能再次接入。`,
            icon: 'fa-ban',
            iconColor: 'amber',
            confirmText: '确认吊销',
            danger: true,
            onConfirm: async () => {
                try {
                    await api.nodeRevoke(node.id)
                    this.portal.showNotification('success', '节点已吊销')
                    this.load(true)
                } catch {}
            }
        })
    }

    handleDelete(node: NodeInfo) {
        this.portal.showConfirm({
            title: '删除节点',
            message: `确定要删除节点 <strong class="text-slate-900">${node.name}</strong> 吗？在线的节点会被立即断开。`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                try {
                    await api.nodeDelete(node.id)
                    this.portal.showNotification('success', '节点已删除')
                    this.load(true)
                } catch {}
            }
        })
    }

    handleCodeDelete(code: NodeCodeInfo) {
        this.portal.showConfirm({
            title: '撤销注册码',
            message: `确定要撤销注册码 <strong class="text-slate-900">${code.name || '未命名注册码'}</strong> 吗？撤销后尚未使用该码接入的受管机将无法注册。`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认撤销',
            danger: true,
            onConfirm: async () => {
                try {
                    await api.nodeCodeDelete(code.id)
                    this.portal.showNotification('success', '注册码已撤销')
                    this.load(true)
                } catch {}
            }
        })
    }

    // ─── 生命周期 ───
    unmounted() {
        this.stopPoll()
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
            <h1 class="title-text">节点管理</h1>
            <p class="text-xs text-slate-500">接入受管机，并切换到节点进行管理</p>
          </div>
        </div>
        <div class="action-group">
          <PageSearch v-model="searchText" search-key="node" placeholder="搜索节点名、主机名、系统或创建人..." focus-color="orange" type-to-search />
          <button class="btn btn-secondary" @click="load()">
            <i class="fas fa-rotate"></i>刷新
          </button>
          <button v-if="portal.hasPerm('POST /api/node/code')" class="btn btn-orange" @click="openAdd">
            <i class="fas fa-plus"></i>接入节点
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
            <h1 class="title-text">节点管理</h1>
            <p class="text-xs text-slate-500 truncate">接入并管理受管机</p>
          </div>
        </div>
        <div class="action-group-sm">
          <button class="btn btn-secondary btn-square" title="刷新" @click="load()">
            <i class="fas fa-rotate text-sm"></i>
          </button>
          <button v-if="portal.hasPerm('POST /api/node/code')" class="btn btn-orange btn-square" title="接入节点" @click="openAdd">
            <i class="fas fa-plus text-sm"></i>
          </button>
        </div>
      </div>
    </div>

    <!-- 移动端搜索 -->
    <div class="mobile-search">
      <PageSearch v-model="searchText" search-key="node" placeholder="搜索节点名、主机名、系统或创建人..." width-class="w-full" focus-color="orange" />
    </div>

    <!-- Loading -->
    <div v-if="loading" class="card-body">
      <div class="empty-state">
        <div class="spinner-lg"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
    </div>

    <div v-else-if="filteredRows.length === 0" class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-sitemap text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">{{ rows.length === 0 ? '暂无受管节点' : '未找到匹配节点' }}</p>
        <p class="text-sm text-slate-400">{{ rows.length === 0 ? '点击右上角「接入节点」生成注册码，并在受管机上以 agent 模式启动' : '尝试更换关键词或清空搜索条件' }}</p>
      </div>
    </div>

    <template v-else>
      <!-- 桌面端表格 -->
      <div class="card-table hidden md:block">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-100 border-b border-slate-200">
              <th class="th">节点</th>
              <th class="w-32 th">状态</th>
              <th class="th">系统</th>
              <th class="th">版本</th>
              <th class="w-24 th">延迟</th>
              <th class="w-44 th-right">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-for="row in filteredRows" :key="row.key" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3 max-w-[280px]">
                <div class="inline-info">
                  <div class="row-icon bg-orange-400">
                    <i class="fas text-white text-sm" :class="row.icon"></i>
                  </div>
                  <div class="min-w-0">
                    <span class="item-title">{{ row.title }}</span>
                    <span class="item-subtitle">{{ row.subtitle }}</span>
                  </div>
                </div>
              </td>
              <td class="td-text">
                <span :class="row.statusClass">{{ row.statusText }}</span>
              </td>
              <template v-if="row.node">
                <td class="td-text">{{ platformText(row.node) }}</td>
                <td class="td-text">{{ row.node.agentVersion || '-' }}</td>
                <td class="td-text">{{ latencyText(row.node) }}</td>
              </template>
              <td v-else-if="row.code" class="td-text" colspan="3">
                {{ approveText(row.code) }} · 创建人 {{ row.code.createdBy }} · {{ timeText(row.code.expiresAt) }} 过期
              </td>
              <td class="px-4 py-3">
                <div class="table-actions">
                  <template v-if="row.node">
                    <button v-if="canOpen(row.node)" class="btn-icon btn-icon-slate" title="进入节点" @click="openNode(row.node)">
                      <i class="fas fa-arrow-right-to-bracket text-xs"></i>
                    </button>
                    <button v-if="row.node.status === 'pending' && portal.hasPerm('POST /api/node/item/:id/approve')" class="btn-icon btn-icon-emerald" title="审批通过" @click="handleApprove(row.node)">
                      <i class="fas fa-check text-xs"></i>
                    </button>
                    <button v-if="portal.hasPerm('PUT /api/node/item/:id')" class="btn-icon btn-icon-blue" title="重命名" @click="openEdit(row.node)">
                      <i class="fas fa-pen text-xs"></i>
                    </button>
                    <button v-if="row.node.status === 'approved' && portal.hasPerm('POST /api/node/item/:id/revoke')" class="btn-icon btn-icon-amber" title="吊销" @click="handleRevoke(row.node)">
                      <i class="fas fa-ban text-xs"></i>
                    </button>
                    <button v-if="portal.hasPerm('DELETE /api/node/item/:id')" class="btn-icon btn-icon-red" title="删除" @click="handleDelete(row.node)">
                      <i class="fas fa-trash text-xs"></i>
                    </button>
                  </template>
                  <template v-else-if="row.code">
                    <button v-if="portal.hasPerm('GET /api/node/codes')" class="btn-icon btn-icon-slate" title="查看接入命令" @click="openCode(row.code)">
                      <i class="fas fa-eye text-xs"></i>
                    </button>
                    <button v-if="portal.hasPerm('DELETE /api/node/code/:id')" class="btn-icon btn-icon-red" title="撤销" @click="handleCodeDelete(row.code)">
                      <i class="fas fa-trash text-xs"></i>
                    </button>
                  </template>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 移动端卡片 -->
      <div class="card-body md:hidden space-y-3">
        <div v-for="row in filteredRows" :key="row.key" class="card-interactive">
          <div class="card-info-row">
            <div class="list-icon bg-orange-400 flex-shrink-0">
              <i class="fas text-white text-base" :class="row.icon"></i>
            </div>
            <div class="min-w-0">
              <span class="item-title-sm">{{ row.title }}</span>
              <span class="item-subtitle">{{ row.subtitle }}</span>
            </div>
          </div>

          <div class="card-prop-row">
            <span class="text-xs text-slate-400 flex-shrink-0">状态</span>
            <span class="text-xs" :class="row.statusClass">{{ row.statusText }}</span>
          </div>
          <template v-if="row.node">
            <div class="card-prop-row">
              <span class="text-xs text-slate-400 flex-shrink-0">系统</span>
              <span class="text-xs text-slate-500">{{ platformText(row.node) }}</span>
            </div>
            <div class="card-prop-row">
              <span class="text-xs text-slate-400 flex-shrink-0">版本</span>
              <span class="text-xs text-slate-500">{{ row.node.agentVersion || '-' }}</span>
            </div>
            <div class="card-prop-row">
              <span class="text-xs text-slate-400 flex-shrink-0">延迟</span>
              <span class="text-xs text-slate-500">{{ latencyText(row.node) }}</span>
            </div>
          </template>
          <template v-else-if="row.code">
            <div class="card-prop-row">
              <span class="text-xs text-slate-400 flex-shrink-0">审批</span>
              <span class="text-xs text-slate-500">{{ approveText(row.code) }}</span>
            </div>
            <div class="card-prop-row">
              <span class="text-xs text-slate-400 flex-shrink-0">创建人</span>
              <span class="text-xs text-slate-500">{{ row.code.createdBy }}</span>
            </div>
            <div class="card-prop-row">
              <span class="text-xs text-slate-400 flex-shrink-0">过期</span>
              <span class="text-xs text-slate-500">{{ timeText(row.code.expiresAt) }}</span>
            </div>
          </template>

          <div class="card-actions">
            <template v-if="row.node">
              <button v-if="canOpen(row.node)" class="btn-icon btn-icon-slate" title="进入节点" @click="openNode(row.node)">
                <i class="fas fa-arrow-right-to-bracket text-xs"></i><span class="text-xs ml-1">进入</span>
              </button>
              <button v-if="row.node.status === 'pending' && portal.hasPerm('POST /api/node/item/:id/approve')" class="btn-icon btn-icon-emerald" title="审批通过" @click="handleApprove(row.node)">
                <i class="fas fa-check text-xs"></i><span class="text-xs ml-1">审批</span>
              </button>
              <button v-if="portal.hasPerm('PUT /api/node/item/:id')" class="btn-icon btn-icon-blue" title="重命名" @click="openEdit(row.node)">
                <i class="fas fa-pen text-xs"></i><span class="text-xs ml-1">重命名</span>
              </button>
              <button v-if="row.node.status === 'approved' && portal.hasPerm('POST /api/node/item/:id/revoke')" class="btn-icon btn-icon-amber" title="吊销" @click="handleRevoke(row.node)">
                <i class="fas fa-ban text-xs"></i><span class="text-xs ml-1">吊销</span>
              </button>
              <button v-if="portal.hasPerm('DELETE /api/node/item/:id')" class="btn-icon btn-icon-red" title="删除" @click="handleDelete(row.node)">
                <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
              </button>
            </template>
            <template v-else-if="row.code">
              <button v-if="portal.hasPerm('GET /api/node/codes')" class="btn-icon btn-icon-slate" title="查看接入命令" @click="openCode(row.code)">
                <i class="fas fa-eye text-xs"></i><span class="text-xs ml-1">查看命令</span>
              </button>
              <button v-if="portal.hasPerm('DELETE /api/node/code/:id')" class="btn-icon btn-icon-red" title="撤销" @click="handleCodeDelete(row.code)">
                <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">撤销</span>
              </button>
            </template>
          </div>
        </div>
      </div>
    </template>
  </div>

  <CodeCreateModal ref="codeModalRef" @success="load(true)" />
  <NodeEditModal ref="editModalRef" @success="load(true)" />
</template>
