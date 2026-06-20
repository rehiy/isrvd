<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { ApisixRoute } from '@/service/types'

import BaseModal from '@/component/modal.vue'
import PageSearch from '@/component/page-search.vue'

import WhitelistEditModal from './widget/whitelist-edit-modal.vue'

interface AddUserState {
    open: boolean
    loading: boolean
    route: ApisixRoute | null
    newUsername: string
    newKey: string
}

@Component({
    components: { BaseModal, PageSearch, WhitelistEditModal }
})
class Whitelist extends Vue {
    portal = usePortal()

    @Ref readonly editModalRef!: InstanceType<typeof WhitelistEditModal>

    whitelist: ApisixRoute[] = []
    loading = false
    searchText = ''

    addUser: AddUserState = {
        open: false,
        loading: false,
        route: null,
        newUsername: '',
        newKey: '',
    }

    get filteredWhitelist() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.whitelist
        return this.whitelist.filter((r: ApisixRoute) =>
            (r.name || '').toLowerCase().includes(keyword) ||
            (r.id || '').toLowerCase().includes(keyword) ||
            (r.consumers || []).some((c: string) => c.toLowerCase().includes(keyword))
        )
    }

    async loadWhitelist() {
        this.loading = true
        try {
            this.whitelist = (await api.apisixWhitelistInspect()).payload || []
        } catch {
            this.portal.showNotification('error', '加载白名单失败')
        } finally {
            this.loading = false
        }
    }

    getRouteUri(r: ApisixRoute) {
        return r.uris?.length ? r.uris.join(', ') : (r.uri || '-')
    }

    getRouteHost(r: ApisixRoute) {
        return r.hosts?.length ? r.hosts.join(', ') : (r.host || '*')
    }

    openCreateModal() {
        this.editModalRef?.show()
    }

    openEditModal(route: ApisixRoute) {
        this.editModalRef?.show(route)
    }

    openAddUserModal(route: ApisixRoute) {
        Object.assign(this.addUser, {
            open: true,
            loading: false,
            route,
            newUsername: '',
            newKey: '',
        })
    }

    async handleAddUser() {
        const route = this.addUser.route
        if (!route?.id) return

        const username = this.addUser.newUsername.trim()
        const key = this.addUser.newKey.trim()
        if (!username) return this.portal.showNotification('error', '请输入用户名')
        if (!key) return this.portal.showNotification('error', '请输入 key-auth key')
        if ((route.consumers || []).includes(username)) return this.portal.showNotification('error', `用户 "${username}" 已在白名单中`)

        const routeKeyAuth = (route.plugins?.['key-auth'] as Record<string, unknown>) || {}
        const keyAuthConfig = {
            header: (routeKeyAuth.header as string) || 'token',
            query: (routeKeyAuth.query as string) || undefined,
            hide_credentials: (routeKeyAuth.hide_credentials as boolean) || false,
        }

        this.addUser.loading = true
        try {
            await api.apisixWhitelistUserCreate({ route_id: route.id, username, key, key_auth: keyAuthConfig })
            this.portal.showNotification('success', `用户 "${username}" 已创建并加入白名单`)
            this.addUser.open = false
            this.loadWhitelist()
        } catch (e: unknown) {
            this.portal.showNotification('error', (e instanceof Error ? e.message : '') || '操作失败')
        } finally {
            this.addUser.loading = false
        }
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadWhitelist()
    }
}

export default toNative(Whitelist)
</script>

<template>
  <div class="card">
    <!-- Toolbar -->
    <div class="card-toolbar">
      <!-- 桌面端 -->
      <div class="hidden md:flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-amber-500">
            <i class="fas fa-shield-halved text-white"></i>
          </div>
          <div>
            <h1 class="text-lg font-semibold text-slate-800 truncate">白名单管理</h1>
            <p class="text-xs text-slate-500">配置路由级别的 Consumer 访问白名单</p>
          </div>
        </div>
        <div class="flex items-center gap-2 flex-shrink-0">
          <PageSearch v-model="searchText" search-key="apisix-whitelist" placeholder="搜索路由或用户..." focus-color="amber" type-to-search />
          <button class="btn btn-secondary" @click="loadWhitelist()">
            <i class="fas fa-rotate"></i>刷新
          </button>
          <button v-if="portal.hasPerm('POST /api/apisix/whitelist')" class="btn btn-amber" @click="openCreateModal()">
            <i class="fas fa-plus"></i>配置白名单
          </button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="flex md:hidden items-center justify-between">
        <div class="flex items-center gap-3 min-w-0 flex-1">
          <div class="page-icon bg-amber-500">
            <i class="fas fa-shield-halved text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">白名单管理</h1>
            <p class="text-xs text-slate-500 truncate">路由级 Consumer 白名单</p>
          </div>
        </div>
        <div class="flex items-center gap-1 flex-shrink-0">
          <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadWhitelist()">
            <i class="fas fa-rotate text-sm"></i>
          </button>
          <button v-if="portal.hasPerm('POST /api/apisix/whitelist')" class="btn btn-amber w-9 h-9 !px-0" title="配置白名单" @click="openCreateModal()">
            <i class="fas fa-plus text-sm"></i>
          </button>
        </div>
      </div>
    </div>

    <!-- 移动端搜索 -->
    <div class="mobile-search">
      <PageSearch v-model="searchText" search-key="apisix-whitelist" placeholder="搜索路由或用户..." width-class="w-full" focus-color="amber" />
    </div>

    <!-- Loading -->
    <div v-if="loading" class="card-body">
      <div class="empty-state">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
    </div>

    <!-- Empty -->
    <div v-else-if="filteredWhitelist.length === 0" class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-shield-halved text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">{{ whitelist.length === 0 ? '暂无白名单数据' : '未找到匹配白名单' }}</p>
        <p class="text-sm text-slate-400">{{ whitelist.length === 0 ? '配置路由的 Consumer 白名单后将在此显示' : '尝试更换关键词或清空搜索条件' }}</p>
      </div>
    </div>

    <!-- 列表 -->
    <template v-else>
      <!-- 桌面端表格 -->
      <div class="card-table hidden md:block">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-50 border-b border-slate-200">
              <th class="th">路由</th>
              <th class="th">描述</th>
              <th class="th">白名单用户</th>
              <th class="w-24 th-right">用户数</th>
              <th class="w-24 th-right">操作</th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-slate-100">
            <tr v-for="route in filteredWhitelist" :key="route.id" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3 max-w-[280px]">
                <div class="flex items-center gap-2 min-w-0">
                  <div class="row-icon bg-amber-400">
                    <i class="fas fa-shield-halved text-white text-sm"></i>
                  </div>
                  <div class="min-w-0">
                    <span class="font-medium text-slate-800 truncate block">{{ getRouteUri(route) }}</span>
                    <span class="text-xs text-slate-400 truncate block mt-0.5">{{ getRouteHost(route) }}</span>
                  </div>
                </div>
              </td>
              <td class="px-4 py-3 text-sm text-slate-600">{{ route.name || route.id }}</td>
              <td class="px-4 py-3">
                <div class="flex flex-wrap gap-1.5">
                  <span v-for="consumer in (route.consumers || [])" :key="consumer" class="inline-flex items-center gap-1.5 px-2 py-0.5 bg-amber-50 text-amber-800 rounded-lg text-xs group">
                    <i class="fas fa-user text-amber-500 text-[10px]"></i>
                    <span class="break-all">{{ consumer }}</span>
                  </span>
                </div>
              </td>
              <td class="px-4 py-3 text-right text-sm text-slate-600">{{ (route.consumers || []).length }}</td>
              <td class="px-4 py-3">
                <div class="flex justify-end items-center gap-1">
                  <button
                    v-if="portal.hasPerm('POST /api/apisix/whitelist')"
                    class="btn-icon btn-icon-blue"
                    title="编辑白名单"
                    @click="openEditModal(route)"
                  >
                    <i class="fas fa-pen text-xs"></i>
                  </button>
                  <button
                    v-if="portal.hasPerm('POST /api/apisix/whitelist/user')"
                    class="btn-icon btn-icon-amber"
                    title="新建用户"
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
        <div v-for="route in filteredWhitelist" :key="route.id" class="card-interactive">
          <div class="card-info-row">
            <div class="list-icon bg-amber-400">
              <i class="fas fa-shield-halved text-white text-base"></i>
            </div>
            <div class="min-w-0">
              <span class="font-medium text-slate-800 text-sm truncate block">{{ getRouteUri(route) }}</span>
              <span class="text-xs text-slate-400 truncate block mt-0.5">{{ getRouteHost(route) }}</span>
            </div>
          </div>

          <div class="card-prop-row">
            <span class="text-xs text-slate-400 flex-shrink-0">描述</span>
            <span class="text-xs text-slate-500">{{ route.name || route.id }}</span>
          </div>

          <div class="card-prop-row-start">
            <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">用户</span>
            <div class="flex flex-wrap gap-1.5">
              <span v-for="consumer in (route.consumers || [])" :key="consumer" class="inline-flex items-center gap-1.5 px-2.5 py-1 bg-amber-50 text-amber-800 rounded-lg text-xs">
                <i class="fas fa-user text-amber-500 text-[10px]"></i>
                <span class="break-all">{{ consumer }}</span>
              </span>
            </div>
          </div>

          <div class="card-actions">
            <button v-if="portal.hasPerm('POST /api/apisix/whitelist')" class="btn-icon btn-icon-blue" title="编辑白名单" @click="openEditModal(route)">
              <i class="fas fa-pen text-xs"></i><span class="text-xs ml-1">编辑白名单</span>
            </button>
            <button v-if="portal.hasPerm('POST /api/apisix/whitelist/user')" class="btn-icon btn-icon-amber" title="新建用户" @click="openAddUserModal(route)">
              <i class="fas fa-user-plus text-xs"></i><span class="text-xs ml-1">新建用户</span>
            </button>
          </div>
        </div>
      </div>
    </template>
  </div>

  <WhitelistEditModal ref="editModalRef" @success="loadWhitelist" />

  <BaseModal v-model="addUser.open" title="新建白名单用户" :loading="addUser.loading" confirm-class="btn-amber" @confirm="handleAddUser">
    <div class="max-w-3xl space-y-4 p-1">
      <div v-if="addUser.route">
        <label class="form-label">路由</label>
        <div class="detail-value text-sm text-slate-700">
          {{ addUser.route.name || addUser.route.id }} - {{ getRouteUri(addUser.route) }}
        </div>
        <p class="text-xs text-slate-400 mt-1">新建 Consumer 并加入当前路由白名单</p>
      </div>

      <div class="space-y-3">
        <div>
          <label class="form-label">用户名 <span class="text-red-500">*</span></label>
          <input v-model="addUser.newUsername" type="text" class="input" placeholder="请输入 Consumer 用户名" />
        </div>
        <div>
          <label class="form-label">key-auth key <span class="text-red-500">*</span></label>
          <input v-model="addUser.newKey" type="text" class="input" placeholder="请输入 API Key" />
          <p class="mt-1 text-xs text-slate-400">提交后会先创建 Consumer 并配置 key-auth，再加入路由白名单</p>
        </div>
      </div>
    </div>

    <template #confirm-text>
      创建并加入
    </template>
  </BaseModal>
</template>
