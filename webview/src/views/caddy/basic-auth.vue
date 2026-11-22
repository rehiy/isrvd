<script lang="ts">
import { Component, Ref, Vue, Watch, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { CaddyBasicAuthRoute, CaddyServerInfo } from '@/service/types'

import BaseModal from '@/component/modal.vue'
import PageSearch from '@/component/page-search.vue'

import BasicAuthEditModal from './widget/basic-auth-edit-modal.vue'

@Component({
    components: { BaseModal, BasicAuthEditModal, PageSearch }
})
class CaddyBasicAuth extends Vue {
    portal = usePortal()

    @Ref readonly editModalRef!: InstanceType<typeof BasicAuthEditModal>

    routes: CaddyBasicAuthRoute[] = []
    servers: CaddyServerInfo[] = []
    selectedServer = 'srv0'
    loading = false
    searchText = ''
    loadRequestID = 0
    deleteOpen = false
    deleteLoading = false
    deleteStage: 1 | 2 = 1
    deleteAcknowledged = false
    pendingDelete: { route: CaddyBasicAuthRoute; username: string; server: string } | null = null

    get canDelete() {
        return this.portal.hasPerm('DELETE /api/caddy/basic-auth/:index/users/:username')
    }

    get deleteTitle() {
        return this.deleteStage === 2 ? this.$t('解除路由认证') : this.$t('删除认证账号')
    }

    get deleteConfirmText() {
        return this.deleteStage === 2 ? this.$t('删除账号并解除认证') : this.$t('删除')
    }

    get deleteDisabled() {
        return !this.canDelete || !this.pendingDelete?.route.revision || (this.deleteStage === 2 && !this.deleteAcknowledged)
    }

    get canListServers() {
        return this.portal.hasPerm('GET /api/caddy/servers')
    }

    get showServerSelect() {
        return this.canListServers && this.servers.length > 0
    }

    get selectedServerName() {
        return this.servers.find(server => server.id === this.selectedServer)?.name || 'srv0'
    }

    get filteredRoutes() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) {
            return this.routes
        }
        return this.routes.filter(r =>
            (r.name || '').toLowerCase().includes(keyword) ||
            r.users.some(u => u.username.toLowerCase().includes(keyword))
        )
    }

    async load() {
        const requestID = ++this.loadRequestID
        const selection = this.selectedServer
        const server = this.selectedServerName
        this.loading = true
        this.routes = []
        try {
            const routes = (await api.caddyBasicAuthList(server)).payload || []
            if (requestID !== this.loadRequestID || selection !== this.selectedServer) {
                return
            }
            this.routes = routes
        } finally {
            if (requestID === this.loadRequestID) {
                this.loading = false
            }
        }
    }

    async loadServers() {
        if (!this.canListServers) {
            await this.load()
            return
        }

        let servers: CaddyServerInfo[]
        try {
            servers = (await api.caddyServerList()).payload || []
        } catch {
            await this.load()
            return
        }

        this.servers = servers
        const defaultServer = this.servers.find(server => server.name === 'srv0')
        const nextServer = this.servers.some(server => this.serverID(server) === this.selectedServer)
            ? this.selectedServer
            : (defaultServer ? this.serverID(defaultServer) : (this.servers[0] ? this.serverID(this.servers[0]) : 'srv0'))
        if (nextServer === this.selectedServer) {
            await this.load()
        } else {
            this.selectedServer = nextServer
        }
    }

    @Watch('selectedServer')
    onSelectedServerChange() {
        this.load()
    }

    serverID(server: CaddyServerInfo) {
        return server.id
    }

    getRouteName(r: CaddyBasicAuthRoute) {
        return r.name || this.$t('路由 #{0}', r.index)
    }

    getRouteDesc(r: CaddyBasicAuthRoute) {
        const proxy = r.handlers?.find(h => h.handler === 'reverse_proxy')
        if (proxy) {
            const ups = (proxy.upstreams as Array<{ dial: string }> | undefined) || []
            return ups.map(u => u.dial).join(', ') || 'reverse_proxy'
        }
        const fs = r.handlers?.find(h => h.handler === 'file_server')
        if (fs) {
            return 'file_server'
        }
        return r.handlers?.map(h => h.handler).join(', ') || '—'
    }

    openAddModal() {
        if (this.loading || this.deleteOpen || this.deleteLoading) {
            return
        }
        this.editModalRef?.showSetup()
    }

    openEditModal(route: CaddyBasicAuthRoute) {
        if (this.loading || this.deleteOpen || this.deleteLoading) {
            return
        }
        this.editModalRef?.showEdit(route)
    }

    openAddUserModal(route: CaddyBasicAuthRoute) {
        if (this.loading || this.deleteOpen || this.deleteLoading) {
            return
        }
        this.editModalRef?.showAddUser(route)
    }

    openDeleteUser(route: CaddyBasicAuthRoute, username: string) {
        if (this.loading || this.deleteOpen || this.deleteLoading || !this.canDelete || !route.revision) {
            return
        }
        this.pendingDelete = { route, username, server: this.selectedServerName }
        this.deleteStage = 1
        this.deleteAcknowledged = false
        this.deleteOpen = true
    }

    async confirmDeleteUser() {
        const pending = this.pendingDelete
        if (!pending || !this.deleteOpen || this.deleteLoading || this.deleteDisabled) {
            return
        }
        if (this.deleteStage === 1 && pending.route.users.length === 1) {
            this.deleteStage = 2
            this.deleteAcknowledged = false
            return
        }
        this.deleteLoading = true
        try {
            await api.caddyBasicAuthUserDelete(pending.route.index, pending.username, pending.server, pending.route.revision)
            this.portal.showNotification('success', this.$t('账号删除成功'))
        } finally {
            this.deleteOpen = false
            this.pendingDelete = null
            this.deleteAcknowledged = false
            try {
                await this.load()
            } finally {
                this.deleteLoading = false
            }
        }
    }

    mounted() {
        this.loadServers()
    }
}

export default toNative(CaddyBasicAuth)
</script>

<template>
  <div class="page">
    <!-- Toolbar -->
    <div class="page-toolbar">
      <!-- 桌面端 -->
      <div class="toolbar-desktop">
        <div class="title-group">
          <div class="page-icon bg-cyan-500">
            <i class="fas fa-lock text-white text-sm"></i>
          </div>
          <div class="min-w-0">
            <h1 class="title-text">{{ $t('基础认证') }}</h1>
            <p class="text-xs text-slate-500 truncate">{{ $t('基于用户名 + 密码的路由 HTTP Basic 认证') }}</p>
          </div>
        </div>
        <div class="action-group">
          <select v-if="showServerSelect" v-model="selectedServer" class="select-sm max-w-40" :title="$t('服务')" :aria-label="$t('服务')">
            <option v-for="server in servers" :key="serverID(server)" :value="serverID(server)">{{ server.name }}</option>
          </select>
          <PageSearch v-model="searchText" search-key="caddy-basic-auth" :placeholder="$t('搜索路由或用户名...')" focus-color="cyan" type-to-search />
          <button class="btn btn-secondary" :disabled="loading || deleteLoading" :aria-busy="loading" @click="load()">
            <i :class="loading ? 'fas fa-spinner fa-spin' : 'fas fa-rotate'"></i>{{ $t('刷新') }}
          </button>
          <button v-if="portal.hasPerm('POST /api/caddy/basic-auth/:index/users')" class="btn btn-cyan" :disabled="loading || deleteLoading" @click="openAddModal()">
            <i class="fas fa-plus"></i>{{ $t('配置认证') }}
          </button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="toolbar-mobile">
        <div class="title-group">
          <div class="page-icon bg-cyan-500">
            <i class="fas fa-lock text-white text-sm"></i>
          </div>
          <div class="min-w-0">
            <h1 class="title-text">{{ $t('基础认证') }}</h1>
            <p class="text-xs text-slate-500 truncate">{{ $t('路由 HTTP Basic 认证') }}</p>
          </div>
        </div>
        <div class="action-group-sm">
          <select v-if="showServerSelect" v-model="selectedServer" class="select-sm max-w-28" :title="$t('服务')" :aria-label="$t('服务')">
            <option v-for="server in servers" :key="serverID(server)" :value="serverID(server)">{{ server.name }}</option>
          </select>
          <button class="btn btn-secondary btn-square" :disabled="loading || deleteLoading" :aria-busy="loading" :title="$t('刷新')" @click="load()">
            <i :class="loading ? 'fas fa-spinner fa-spin text-sm' : 'fas fa-rotate text-sm'"></i>
          </button>
          <button v-if="portal.hasPerm('POST /api/caddy/basic-auth/:index/users')" class="btn btn-cyan btn-square" :disabled="loading || deleteLoading" :title="$t('配置认证')" @click="openAddModal()">
            <i class="fas fa-plus text-sm"></i>
          </button>
        </div>
      </div>
    </div>

    <!-- 移动端搜索 -->
    <div class="mobile-search">
      <PageSearch v-model="searchText" search-key="caddy-basic-auth" :placeholder="$t('搜索路由或用户名...')" width-class="w-full" focus-color="cyan" />
    </div>

    <!-- Loading -->
    <div v-if="loading" class="card-body">
      <div class="empty-state">
        <div class="spinner-lg"></div>
        <p class="text-slate-500">{{ $t('加载中...') }}</p>
      </div>
    </div>

    <!-- Empty -->
    <div v-else-if="filteredRoutes.length === 0" class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-lock text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">
          {{ routes.length === 0 ? $t('暂无基础认证') : $t('未找到匹配路由') }}
        </p>
        <p class="text-sm text-slate-400">
          {{ routes.length === 0 ? $t('点击「配置认证」为任意路由启用 HTTP Basic 认证') : $t('尝试更换关键词或清空搜索条件') }}
        </p>
      </div>
    </div>

    <!-- 列表 -->
    <template v-else>
      <!-- 桌面端表格 -->
      <div class="card-table hidden md:block">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-100 border-b border-slate-200">
              <th class="th">{{ $t('路由') }}</th>
              <th class="th">{{ $t('转发头') }}</th>
              <th class="th">{{ $t('账号') }}</th>
              <th class="w-16 th-right">{{ $t('数量') }}</th>
              <th class="w-24 th-right">{{ $t('操作') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-for="route in filteredRoutes" :key="route.index" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3 max-w-[280px]">
                <div class="inline-info">
                  <div class="row-icon bg-cyan-400">
                    <i class="fas fa-lock text-white text-sm"></i>
                  </div>
                  <div class="min-w-0">
                    <span class="item-title">{{ getRouteName(route) }}</span>
                    <span class="item-subtitle">{{ getRouteDesc(route) }}</span>
                  </div>
                </div>
              </td>
              <td class="td-text">
                <code v-if="route.forwardHeader" class="text-xs bg-slate-100 px-2 py-0.5 rounded-lg text-slate-600 font-mono">{{ route.forwardHeader }}</code>
                <span v-else>—</span>
              </td>
              <td class="px-4 py-3">
                <div class="flex flex-wrap gap-1.5">
                  <span
                    v-for="user in route.users"
                    :key="user.username"
                    class="badge-sm gap-1.5 bg-cyan-50 text-cyan-700"
                  >
                    <i class="fas fa-user text-cyan-500 text-[10px]"></i>
                    <span>{{ user.username }}</span>
                    <button
                      v-if="canDelete"
                      type="button"
                      class="btn-tag-remove"
                      :disabled="deleteLoading"
                      :title="$t('删除账号「{0}」', user.username)"
                      :aria-label="$t('删除账号「{0}」', user.username)"
                      @click="openDeleteUser(route, user.username)"
                    >
                      <i class="fas fa-xmark text-[10px]"></i>
                    </button>
                  </span>
                </div>
              </td>
              <td class="px-4 py-3 text-right text-sm text-slate-600">{{ route.users.length }}</td>
              <td class="px-4 py-3">
                <div class="table-actions">
                  <button
                    v-if="portal.hasPerm('PUT /api/caddy/basic-auth/:index/config')"
                    class="btn-icon btn-icon-blue"
                    :title="$t('编辑配置')"
                    @click="openEditModal(route)"
                  >
                    <i class="fas fa-pen text-xs"></i>
                  </button>
                  <button
                    v-if="portal.hasPerm('POST /api/caddy/basic-auth/:index/users')"
                    class="btn-icon btn-icon-emerald"
                    :title="$t('添加账号')"
                    @click="openAddUserModal(route)"
                  >
                    <i class="fas fa-user-plus text-xs"></i>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 移动端卡片 -->
      <div class="card-body md:hidden space-y-3">
        <div v-for="route in filteredRoutes" :key="route.index" class="card-interactive">
          <div class="card-info-row">
            <div class="list-icon bg-cyan-400">
              <i class="fas fa-lock text-white text-base"></i>
            </div>
            <div class="min-w-0">
              <span class="item-title-sm">{{ getRouteName(route) }}</span>
              <span class="item-subtitle">{{ getRouteDesc(route) }}</span>
            </div>
          </div>

          <div v-if="route.forwardHeader" class="card-prop-row-start">
            <span class="prop-label-start">{{ $t('转发头') }}</span>
            <code class="text-xs bg-slate-100 px-2 py-0.5 rounded-lg text-slate-600 font-mono break-all">{{ route.forwardHeader }}</code>
          </div>

          <div class="card-prop-row-start">
            <span class="prop-label-start">{{ $t('账号') }}</span>
            <div class="flex flex-wrap gap-1.5">
              <span
                v-for="user in route.users"
                :key="user.username"
                class="badge bg-cyan-50 text-cyan-700 gap-1.5"
              >
                <i class="fas fa-user text-cyan-500 text-[10px]"></i>
                <span>{{ user.username }}</span>
                <button
                  v-if="canDelete"
                  type="button"
                  class="btn-tag-remove"
                  :disabled="deleteLoading"
                  :title="$t('删除账号「{0}」', user.username)"
                  :aria-label="$t('删除账号「{0}」', user.username)"
                  @click="openDeleteUser(route, user.username)"
                >
                  <i class="fas fa-xmark text-[10px]"></i>
                </button>
              </span>
            </div>
          </div>

          <div class="card-actions">
            <button
              v-if="portal.hasPerm('PUT /api/caddy/basic-auth/:index/config')"
              class="btn-icon btn-icon-blue"
              :title="$t('编辑配置')"
              @click="openEditModal(route)"
            >
              <i class="fas fa-pen text-xs"></i>
              <span class="text-xs ml-1">{{ $t('编辑配置') }}</span>
            </button>
            <button
              v-if="portal.hasPerm('POST /api/caddy/basic-auth/:index/users')"
              class="btn-icon btn-icon-emerald"
              :title="$t('添加账号')"
              @click="openAddUserModal(route)"
            >
              <i class="fas fa-user-plus text-xs"></i>
              <span class="text-xs ml-1">{{ $t('添加账号') }}</span>
            </button>
          </div>
        </div>
      </div>
    </template>
  </div>

  <BasicAuthEditModal ref="editModalRef" :server="selectedServerName" @success="load" @refresh="load" />
  <BaseModal
    v-model="deleteOpen"
    :title="deleteTitle"
    :loading="deleteLoading"
    :confirm-disabled="deleteDisabled"
    confirm-class="btn-danger"
    max-width-class="max-w-lg"
    @confirm="confirmDeleteUser"
  >
    <div v-if="pendingDelete" class="space-y-4">
      <p class="text-sm text-slate-600">{{ getRouteName(pendingDelete.route) }}</p>
      <p v-if="deleteStage === 1" class="text-sm text-slate-700">{{ $t('确定删除账号「{0}」吗？', pendingDelete.username) }}</p>
      <template v-else>
        <p class="text-sm text-red-500">{{ $t('删除最后一个账号将解除此路由的 HTTP Basic 认证，任何人都可能访问受保护的资源。确定继续吗？') }}</p>
        <label class="check-label">
          <input v-model="deleteAcknowledged" type="checkbox" class="rounded border-slate-300 text-red-500 focus:ring-red-500" />
          <span class="text-sm text-slate-600">{{ $t('我了解这会解除路由认证') }}</span>
        </label>
      </template>
    </div>
    <template #confirm-text>{{ deleteConfirmText }}</template>
  </BaseModal>
</template>
