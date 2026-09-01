<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { CaddyServerInfo } from '@/service/types'

import PageSearch from '@/component/page-search.vue'

import ServerEditModal from './widget/server-edit-modal.vue'

@Component({
    components: { PageSearch, ServerEditModal }
})
class CaddyServers extends Vue {
    portal = usePortal()

    @Ref readonly editModalRef!: InstanceType<typeof ServerEditModal>

    servers: CaddyServerInfo[] = []
    loading = false
    searchText = ''
    editingServer = ''

    get filteredServers() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.servers
        return this.servers.filter(server =>
            `${server.name} ${(server.listen || []).join(' ')} ${(server.protocols || []).join(' ')}`.toLowerCase().includes(keyword)
        )
    }

    async loadServers() {
        this.loading = true
        try {
            this.servers = (await api.caddyServerList()).payload || []
        } catch {} finally {
            this.loading = false
        }
    }

    openCreateModal() {
        this.editModalRef?.show(null)
    }

    get canEditServer() {
        return this.portal.hasPerm('GET /api/caddy/server/:name') &&
            this.portal.hasPerm('PUT /api/caddy/server/:name')
    }

    get canDeleteServers() {
        return this.portal.hasPerm('DELETE /api/caddy/server/:name')
    }

    async openEditModal(server: CaddyServerInfo) {
        if (this.editingServer) return
        const id = this.serverID(server)
        this.editingServer = id
        try {
            const detail = (await api.caddyServerInspect(id)).payload
            if (!detail) {
                this.portal.showNotification('error', '服务详情不存在')
                return
            }
            this.editModalRef?.show(detail)
        } catch {} finally {
            this.editingServer = ''
        }
    }

    serverID(server: CaddyServerInfo) {
        return server.id
    }

    isDefaultServer(server: CaddyServerInfo) {
        return server.name === 'srv0'
    }

    canDeleteServer(server: CaddyServerInfo) {
        return !this.isDefaultServer(server) && this.servers.length > 1
    }

    deleteDisabledTitle(server: CaddyServerInfo) {
        return this.isDefaultServer(server) ? 'srv0 是结构化 API 的默认服务，不可删除' : '至少保留一个服务'
    }

    autoHTTPSLabel(server: CaddyServerInfo) {
        if (!server.automatic_https) return '默认'
        return server.automatic_https.disable ? '已禁用' : '已启用'
    }

    autoHTTPSClass(server: CaddyServerInfo) {
        if (!server.automatic_https) return 'text-slate-500'
        return server.automatic_https.disable ? 'text-red-500 font-medium' : 'text-emerald-600 font-medium'
    }

    deleteServer(server: CaddyServerInfo) {
        if (!this.canDeleteServer(server)) return
        const id = this.serverID(server)
        this.portal.showConfirm({
            title: '删除服务',
            message: '确定删除这个服务吗？其下全部路由也会被删除。',
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                try {
                    await api.caddyServerDelete(id)
                    this.portal.showNotification('success', '服务删除成功')
                    this.loadServers()
                } catch {}
            }
        })
    }

    mounted() {
        this.loadServers()
    }
}

export default toNative(CaddyServers)
</script>

<template>
  <div class="page">
    <div class="page-toolbar">
      <div class="toolbar-desktop">
        <div class="title-group-static">
          <div class="page-icon bg-rose-500"><i class="fas fa-server text-white"></i></div>
          <div class="min-w-0">
            <h1 class="title-text">服务</h1>
            <p class="text-xs text-slate-500 truncate">管理 srv0 等命名服务、监听地址和协议</p>
          </div>
        </div>
        <div class="action-group">
          <PageSearch v-model="searchText" search-key="caddy-servers" placeholder="搜索名称、监听或协议..." focus-color="rose" type-to-search />
          <button class="btn btn-secondary" @click="loadServers()"><i class="fas fa-rotate"></i>刷新</button>
          <button v-if="portal.hasPerm('POST /api/caddy/server')" class="btn btn-rose" @click="openCreateModal()"><i class="fas fa-plus"></i>新建服务</button>
        </div>
      </div>

      <div class="toolbar-mobile">
        <div class="title-group">
          <div class="page-icon bg-rose-500"><i class="fas fa-server text-white"></i></div>
          <div class="min-w-0">
            <h1 class="title-text">服务</h1>
            <p class="text-xs text-slate-500 truncate">管理监听地址和协议</p>
          </div>
        </div>
        <div class="action-group-sm">
          <button class="btn btn-secondary btn-square" title="刷新" @click="loadServers()"><i class="fas fa-rotate text-sm"></i></button>
          <button v-if="portal.hasPerm('POST /api/caddy/server')" class="btn btn-rose btn-square" title="新建服务" @click="openCreateModal()"><i class="fas fa-plus text-sm"></i></button>
        </div>
      </div>
    </div>

    <div class="mobile-search">
      <PageSearch v-model="searchText" search-key="caddy-servers" placeholder="搜索服务..." width-class="w-full" focus-color="rose" />
    </div>

    <div v-if="loading" class="card-body">
      <div class="empty-state">
        <div class="spinner-lg"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
    </div>

    <div v-else-if="filteredServers.length === 0" class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon"><i class="fas fa-server text-4xl text-slate-300"></i></div>
        <p class="text-slate-600 font-medium mb-1">{{ servers.length === 0 ? '暂无服务' : '未找到匹配服务' }}</p>
        <p class="text-sm text-slate-400">{{ servers.length === 0 ? '点击「新建服务」开始创建' : '尝试更换关键词或清空搜索条件' }}</p>
      </div>
    </div>

    <template v-else>
      <div class="card-table hidden md:block">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-100 border-b border-slate-200">
              <th class="th">服务</th>
              <th class="th">协议</th>
              <th class="w-28 th">自动 HTTPS</th>
              <th class="w-24 th">路由</th>
              <th class="w-32 th-right">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-for="server in filteredServers" :key="serverID(server)" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3 max-w-[280px]">
                <div class="inline-info">
                  <div class="row-icon bg-rose-400">
                    <i class="fas fa-server text-white text-sm"></i>
                  </div>
                  <div class="min-w-0">
                    <span class="item-title font-mono">{{ server.name }}</span>
                    <span class="item-subtitle-mono">{{ (server.listen || []).join(', ') || '无监听地址' }}</span>
                  </div>
                </div>
              </td>
              <td class="td-text">
                <div v-if="server.protocols?.length" class="flex flex-wrap gap-1">
                  <span v-for="protocol in server.protocols" :key="protocol" class="badge-sm bg-rose-50 text-rose-700">{{ protocol }}</span>
                </div>
                <span v-else class="text-slate-400">默认</span>
              </td>
              <td class="td-text"><span :class="autoHTTPSClass(server)">{{ autoHTTPSLabel(server) }}</span></td>
              <td class="td-text"><span class="text-emerald-600 font-medium">{{ server.routeCount }}</span></td>
              <td class="px-4 py-3">
                <div class="table-actions">
                  <button
                    v-if="canEditServer"
                    class="btn-icon btn-icon-blue disabled:opacity-50 disabled:cursor-not-allowed"
                    :disabled="Boolean(editingServer)"
                    title="编辑"
                    @click="openEditModal(server)"
                  >
                    <i :class="['fas text-xs', editingServer === serverID(server) ? 'fa-spinner fa-spin' : 'fa-pen']"></i>
                  </button>
                  <button v-if="canDeleteServer(server) && canDeleteServers" class="btn-icon btn-icon-red" title="删除" @click="deleteServer(server)">
                    <i class="fas fa-trash text-xs"></i>
                  </button>
                  <span v-else-if="canDeleteServers" :title="deleteDisabledTitle(server)">
                    <button disabled class="btn-icon text-slate-300 cursor-not-allowed" :aria-label="deleteDisabledTitle(server)">
                      <i class="fas fa-trash text-xs"></i>
                    </button>
                  </span>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="card-body md:hidden space-y-3">
        <div v-for="server in filteredServers" :key="serverID(server)" class="card-interactive">
          <div class="card-info-row">
            <div class="list-icon bg-rose-400">
              <i class="fas fa-server text-white text-base"></i>
            </div>
            <div class="min-w-0">
              <span class="item-title-sm font-mono">{{ server.name }}</span>
              <span class="item-subtitle-mono">{{ (server.listen || []).join(', ') || '无监听地址' }}</span>
            </div>
          </div>

          <div class="card-prop-row-start">
            <span class="prop-label-start">协议</span>
            <span v-if="server.protocols?.length" class="flex flex-wrap gap-1">
              <span v-for="protocol in server.protocols" :key="protocol" class="badge-sm bg-rose-50 text-rose-700">{{ protocol }}</span>
            </span>
            <span v-else class="text-xs text-slate-400">默认</span>
          </div>
          <div class="card-prop-row">
            <span class="text-xs text-slate-400 flex-shrink-0">自动 HTTPS</span>
            <span :class="autoHTTPSClass(server)" class="text-xs">{{ autoHTTPSLabel(server) }}</span>
          </div>
          <div class="card-prop-row">
            <span class="text-xs text-slate-400 flex-shrink-0">路由</span>
            <span class="text-emerald-600 text-xs font-medium">{{ server.routeCount }}</span>
          </div>

          <div class="card-actions">
            <button
              v-if="canEditServer"
              class="btn-icon btn-icon-blue disabled:opacity-50 disabled:cursor-not-allowed"
              :disabled="Boolean(editingServer)"
              title="编辑"
              @click="openEditModal(server)"
            >
              <i :class="['fas text-xs', editingServer === serverID(server) ? 'fa-spinner fa-spin' : 'fa-pen']"></i><span class="text-xs ml-1">编辑</span>
            </button>
            <button v-if="canDeleteServer(server) && canDeleteServers" class="btn-icon btn-icon-red" title="删除" @click="deleteServer(server)">
              <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
            </button>
            <span v-else-if="canDeleteServers" :title="deleteDisabledTitle(server)">
              <button disabled class="btn-icon text-slate-300 cursor-not-allowed" :aria-label="deleteDisabledTitle(server)">
                <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
              </button>
            </span>
          </div>
        </div>
      </div>
    </template>
  </div>

  <ServerEditModal ref="editModalRef" @success="loadServers" />
</template>
