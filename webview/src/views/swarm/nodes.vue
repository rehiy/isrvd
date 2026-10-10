<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { SwarmNodeInfo } from '@/service/types'

import { copyToClipboard } from '@/helper/dom'

import BaseModal from '@/component/modal.vue'
import PageSearch from '@/component/page-search.vue'

@Component({ components: { BaseModal, PageSearch } })
class Nodes extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    nodes: SwarmNodeInfo[] = []
    loading = false
    searchText = ''
    showJoinModal = false
    joinTokens: { worker: string; manager: string } | null = null
    joinTokensLoading = false
    joinTokenRole: 'worker' | 'manager' = 'worker'
    joinAddr = ''
    copied = false

    get filteredNodes() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.nodes
        return this.nodes.filter((n: SwarmNodeInfo) =>
            (n.hostname || '').toLowerCase().includes(keyword) ||
            (n.id || '').toLowerCase().includes(keyword) ||
            (n.role || '').toLowerCase().includes(keyword) ||
            (n.availability || '').toLowerCase().includes(keyword) ||
            (n.state || '').toLowerCase().includes(keyword) ||
            (n.addr || '').toLowerCase().includes(keyword) ||
            (n.engineVersion || '').toLowerCase().includes(keyword)
        )
    }

    // ─── 方法 ───
    async loadNodes() {
        this.loading = true
        try {
            const res = await api.swarmNodeList()
            const list = res.payload || []
            // leader 节点排最前
            this.nodes = list.sort((a: SwarmNodeInfo, b: SwarmNodeInfo) => (b.leader ? 1 : 0) - (a.leader ? 1 : 0))
        } finally {
            this.loading = false
        }
    }

    handleNodeAction(node: SwarmNodeInfo, action: string) {
        const labels: Record<string, string> = { drain: this.$t('排空'), active: this.$t('激活'), pause: this.$t('暂停'), remove: this.$t('移除') }
        const label = labels[action] || action
        this.portal.showConfirm({
            title: this.$t('{0}节点', label),
            message: this.$t('确定要{0}节点 <strong class="text-slate-900">{1}</strong> 吗？', label, node.hostname),
            icon: action === 'remove' ? 'fa-trash' : 'fa-server',
            iconColor: action === 'remove' ? 'red' : 'amber',
            confirmText: this.$t('确认{0}', label),
            danger: action === 'remove',
            onConfirm: async () => {
                await api.swarmNodeAction(node.id, action)
                this.portal.showNotification('success', this.$t('节点{0}成功', label))
                this.loadNodes()
            }
        })
    }

    async openJoinModal() {
        this.showJoinModal = true
        this.joinTokensLoading = true
        this.joinTokens = null
        this.copied = false
        try {
            const res = await api.swarmJoinToken()
            this.joinTokens = res.payload || null
        } finally {
            this.joinTokensLoading = false
        }
    }

    get joinCommand() {
        if (!this.joinTokens) return ''
        const token = this.joinRole === 'worker' ? this.joinTokens.worker : this.joinTokens.manager
        const addr = this.joinAddr.trim()
        return addr
            ? `docker swarm join --token ${token} ${addr}`
            : `docker swarm join --token ${token} <manager-addr>:2377`
    }

    get joinRole() {
        return this.joinTokenRole
    }

    async copyJoinCommand() {
        const ok = await copyToClipboard(this.joinCommand)
        if (ok) {
            this.copied = true
            setTimeout(() => { this.copied = false }, 2000)
        } else {
            this.portal.showNotification('error', this.$t('复制失败，请手动复制'))
        }
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadNodes()
    }
}

export default toNative(Nodes)
</script>

<template>
  <div>
    <div class="page">
      <div class="page-toolbar">
        <!-- 桌面端 -->
        <div class="toolbar-desktop">
          <div class="flex items-center gap-3">
            <div class="page-icon bg-blue-500">
              <i class="fas fa-server text-white"></i>
            </div>
            <div>
              <h1 class="title-text">{{ $t('节点') }}</h1>
              <p class="text-xs text-slate-500">{{ $t('管理 Swarm 集群节点') }}</p>
            </div>
          </div>
          <div class="action-group">
            <PageSearch v-model="searchText" search-key="swarm-nodes" :placeholder="$t('请输入搜索关键词...')" focus-color="blue" type-to-search />
            <button class="btn btn-secondary" @click="loadNodes">
              <i class="fas fa-rotate"></i>{{ $t('刷新') }}
            </button>
            <button v-if="portal.hasPerm('GET /api/swarm/token')" class="btn btn-blue" @click="openJoinModal">
              <i class="fas fa-plus"></i>{{ $t('加入集群') }}
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="toolbar-mobile">
          <div class="title-group">
            <div class="page-icon bg-blue-500">
              <i class="fas fa-server text-white"></i>
            </div>
            <div class="min-w-0">
              <h1 class="title-text">{{ $t('节点') }}</h1>
              <p class="text-xs text-slate-500 truncate">{{ $t('管理 Swarm 集群节点') }}</p>
            </div>
          </div>
          <div class="action-group-sm">
            <button class="btn btn-secondary btn-square" :title="$t('刷新')" @click="loadNodes">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button v-if="portal.hasPerm('GET /api/swarm/token')" class="btn btn-blue btn-square" :title="$t('加入集群')" @click="openJoinModal">
              <i class="fas fa-plus text-sm"></i>
            </button>
          </div>
        </div>
      </div>
      <div class="mobile-search">
        <PageSearch v-model="searchText" search-key="swarm-nodes" :placeholder="$t('请输入搜索关键词...')" width-class="w-full" focus-color="blue" />
      </div>

      <div v-if="loading" class="card-body">
        <div class="empty-state">
          <div class="spinner-lg"></div>
          <p class="text-slate-500">{{ $t('加载中...') }}</p>
        </div>
      </div>
      <template v-else-if="filteredNodes.length > 0">
        <!-- 桌面端表格视图 -->
        <div class="card-table hidden md:block">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-100 border-b border-slate-200">
                <th class="th">{{ $t('主机名') }}</th>
                <th class="w-24 th">{{ $t('角色') }}</th>
                <th class="w-28 th">{{ $t('状态') }}</th>
                <th class="w-28 th">{{ $t('可用性') }}</th>
                <th class="th">{{ $t('IP 地址') }}</th>
                <th class="w-28 th">{{ $t('引擎版本') }}</th>
                <th class="w-44 th-right">{{ $t('操作') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-100">
              <tr v-for="n in filteredNodes" :key="n.id" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3 max-w-[280px]">
                  <div class="inline-info">
                    <div class="row-icon bg-blue-400">
                      <i class="fas fa-server text-white text-sm"></i>
                    </div>
                    <div class="min-w-0">
                      <router-link v-if="portal.hasPerm('GET /api/swarm/node/:id')" :to="`/swarm/node/${n.id}`" class="font-medium text-slate-800 hover:text-blue-600 transition-colors truncate block">{{ n.hostname }}</router-link>
                      <span v-else class="item-title">{{ n.hostname }}</span>
                      <span class="item-subtitle-mono">{{ n.id }}</span>
                    </div>
                  </div>
                </td>
                <td class="px-4 py-3 text-xs capitalize">
                  <span :class="n.role === 'manager' ? 'text-indigo-600 font-medium' : 'text-slate-500'" class="whitespace-nowrap">{{ n.role }}</span>
                </td>
                <td class="px-4 py-3 text-xs capitalize">
                  <span :class="n.state === 'ready' ? 'text-emerald-600 font-medium' : n.state === 'down' ? 'text-red-500 font-medium' : 'text-slate-500'">{{ n.state }}</span>
                </td>
                <td class="px-4 py-3 text-xs capitalize">
                  <span :class="n.availability === 'active' ? 'text-emerald-600 font-medium' : n.availability === 'drain' ? 'text-amber-600 font-medium' : 'text-slate-500'">{{ n.availability }}</span>
                </td>
                <td class="px-4 py-3 font-mono text-xs text-slate-500">{{ n.addr || '-' }}</td>
                <td class="px-4 py-3 text-xs text-slate-500">{{ n.engineVersion || '-' }}</td>
                <td class="px-4 py-3">
                  <div class="table-actions">
                    <button v-if="portal.hasPerm('GET /api/swarm/node/:id')" class="btn-icon btn-icon-slate" :title="$t('查看详情')" @click="$router.push(`/swarm/node/${n.id}`)"><i class="fas fa-circle-info text-xs"></i></button>
                    <button v-if="n.availability !== 'active' && portal.hasPerm('POST /api/swarm/node/:id/action')" class="btn-icon btn-icon-emerald" :title="$t('激活')" @click="handleNodeAction(n, 'active')"><i class="fas fa-play text-xs"></i></button>
                    <button v-if="n.availability !== 'drain' && portal.hasPerm('POST /api/swarm/node/:id/action')" class="btn-icon btn-icon-amber" :title="$t('排空')" @click="handleNodeAction(n, 'drain')"><i class="fas fa-arrow-down text-xs"></i></button>
                    <button v-if="n.availability !== 'pause' && portal.hasPerm('POST /api/swarm/node/:id/action')" class="btn-icon btn-icon-slate" :title="$t('暂停')" @click="handleNodeAction(n, 'pause')"><i class="fas fa-pause text-xs"></i></button>
                    <button v-if="portal.hasPerm('POST /api/swarm/node/:id/action')" :disabled="n.leader" :class="n.leader ? 'btn-icon text-slate-300 cursor-not-allowed' : 'btn-icon btn-icon-red'" :title="n.leader ? $t('不能移除 Leader 节点') : $t('移除')" @click="n.leader ? null : handleNodeAction(n, 'remove')"><i class="fas fa-trash text-xs"></i></button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 移动端卡片视图 -->
        <div class="card-body md:hidden space-y-3">
          <div v-for="n in filteredNodes" :key="n.id" class="card-interactive">
            <!-- 顶部：主机名和图标 -->
            <div class="card-info-row">
              <div class="list-icon bg-blue-400">
                <i class="fas fa-server text-white text-base"></i>
              </div>
              <div class="min-w-0">
                <router-link v-if="portal.hasPerm('GET /api/swarm/node/:id')" :to="`/swarm/node/${n.id}`" class="font-medium text-slate-800 hover:text-blue-600 transition-colors text-sm truncate block">{{ n.hostname }}</router-link>
                <span v-else class="item-title-sm">{{ n.hostname }}</span>
                <span class="item-subtitle-mono">{{ n.id }}</span>
              </div>
            </div>
            
            <!-- 角色 | 状态（关联：节点身份与健康） -->
            <div class="flex items-center gap-1.5 mb-3 flex-nowrap">
              <span class="text-xs text-slate-400 flex-shrink-0">{{ $t('角色') }}</span>
              <span :class="n.role === 'manager' ? 'text-indigo-600 font-medium' : 'text-slate-600'" class="text-xs capitalize whitespace-nowrap flex-shrink-0">{{ n.role }}</span>
              <span class="text-xs text-slate-200">|</span>
              <span class="text-xs text-slate-400">{{ $t('状态') }}</span>
              <span :class="n.state === 'ready' ? 'text-emerald-600 font-medium' : n.state === 'down' ? 'text-red-500 font-medium' : 'text-slate-500'" class="text-xs capitalize">{{ n.state }}</span>
            </div>
            <!-- 可用性（调度状态，独立） -->
            <div class="card-prop-row">
              <span class="text-xs text-slate-400 flex-shrink-0">{{ $t('可用性') }}</span>
              <span :class="n.availability === 'active' ? 'text-emerald-600' : n.availability === 'drain' ? 'text-amber-600' : 'text-slate-500'" class="text-xs capitalize">{{ n.availability }}</span>
            </div>
            <!-- IP 地址（独立） -->
            <div class="card-prop-row">
              <span class="text-xs text-slate-400 flex-shrink-0">{{ $t('IP 地址') }}</span>
              <span class="text-xs text-slate-600 font-mono">{{ n.addr || '-' }}</span>
            </div>
            <!-- 引擎版本（独立） -->
            <div class="card-prop-row">
              <span class="text-xs text-slate-400 flex-shrink-0">{{ $t('引擎版本') }}</span>
              <span class="text-xs text-slate-500">{{ n.engineVersion || '-' }}</span>
            </div>
            
            <!-- 底部：操作按钮 -->
            <div class="card-actions">
              <button v-if="portal.hasPerm('GET /api/swarm/node/:id')" class="btn-icon btn-icon-slate" :title="$t('查看详情')" @click="$router.push(`/swarm/node/${n.id}`)">
                <i class="fas fa-circle-info text-xs"></i><span class="text-xs ml-1">{{ $t('详情') }}</span>
              </button>
              <button v-if="n.availability !== 'active' && portal.hasPerm('POST /api/swarm/node/:id/action')" class="btn-icon btn-icon-emerald" :title="$t('激活')" @click="handleNodeAction(n, 'active')">
                <i class="fas fa-play text-xs"></i><span class="text-xs ml-1">{{ $t('激活') }}</span>
              </button>
              <button v-if="n.availability !== 'drain' && portal.hasPerm('POST /api/swarm/node/:id/action')" class="btn-icon btn-icon-amber" :title="$t('排空')" @click="handleNodeAction(n, 'drain')">
                <i class="fas fa-arrow-down text-xs"></i><span class="text-xs ml-1">{{ $t('排空') }}</span>
              </button>
              <button v-if="n.availability !== 'pause' && portal.hasPerm('POST /api/swarm/node/:id/action')" class="btn-icon btn-icon-slate" :title="$t('暂停')" @click="handleNodeAction(n, 'pause')">
                <i class="fas fa-pause text-xs"></i><span class="text-xs ml-1">{{ $t('暂停') }}</span>
              </button>
              <button v-if="portal.hasPerm('POST /api/swarm/node/:id/action')" :disabled="n.leader" :class="n.leader ? 'btn-icon text-slate-300 cursor-not-allowed' : 'btn-icon btn-icon-red'" :title="n.leader ? $t('不能移除 Leader 节点') : $t('移除')" @click="n.leader ? null : handleNodeAction(n, 'remove')">
                <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">{{ $t('移除') }}</span>
              </button>
            </div>
          </div>
        </div>
      </template>
      <div v-else class="card-body">
        <div class="empty-state">
          <div class="empty-state-icon">
            <i class="fas fa-server text-4xl text-slate-300"></i>
          </div>
          <p class="text-slate-600 font-medium mb-1">{{ nodes.length === 0 ? $t('暂无节点') : $t('未找到匹配节点') }}</p>
          <p class="text-sm text-slate-400">{{ nodes.length === 0 ? $t('当前集群没有可用节点') : $t('尝试更换关键词或清空搜索条件') }}</p>
        </div>
      </div>
    </div>
  </div>

  <!-- 加入集群弹窗 -->
  <BaseModal v-model="showJoinModal" :title="$t('加入集群')" :loading="joinTokensLoading" :show-confirm="false">
    <div v-if="joinTokensLoading" class="empty-state">
      <div class="spinner-lg"></div>
      <p class="text-slate-500">{{ $t('加载中...') }}</p>
    </div>
    <div v-else-if="joinTokens" class="space-y-4">
      <!-- 角色选择 -->
      <div>
        <label class="form-label">{{ $t('节点角色') }}</label>
        <div class="tab-group w-full">
          <button
            type="button"
            class="tab-btn flex-1 justify-center"
            :class="joinTokenRole === 'worker' ? 'tab-btn-active text-blue-600' : 'tab-btn-inactive'"
            @click="joinTokenRole = 'worker'"
          >
            Worker
          </button>
          <button
            type="button"
            class="tab-btn flex-1 justify-center"
            :class="joinTokenRole === 'manager' ? 'tab-btn-active text-blue-600' : 'tab-btn-inactive'"
            @click="joinTokenRole = 'manager'"
          >
            Manager
          </button>
        </div>
      </div>
      <!-- Manager 地址 -->
      <div>
        <label class="form-label">{{ $t('Manager 地址') }}</label>
        <input v-model="joinAddr" type="text" class="input" :placeholder="$t('请输入 Manager 地址')" />
        <p class="mt-1 text-xs text-slate-400">{{ $t('留空则使用占位符，填写后命令可直接使用') }}</p>
      </div>
      <!-- 加入命令 -->
      <div>
        <label class="form-label">{{ $t('加入命令') }}</label>
        <div class="relative">
          <pre class="bg-slate-900 text-emerald-400 rounded-xl p-4 text-xs font-mono overflow-x-auto whitespace-pre-wrap break-all pr-12">{{ joinCommand }}</pre>
          <button
            :class="copied ? 'text-emerald-400' : 'text-slate-400 hover:text-white'"
            class="absolute top-3 right-3 w-7 h-7 flex items-center justify-center rounded transition-colors"
            :title="copied ? $t('已复制') : $t('复制命令')"
            @click="copyJoinCommand"
          >
            <i :class="copied ? 'fas fa-check' : 'fas fa-copy'" class="text-xs"></i>
          </button>
        </div>
      </div>
    </div>
  </BaseModal>
</template>
