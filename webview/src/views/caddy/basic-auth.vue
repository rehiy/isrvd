<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { CaddyBasicAuthRoute } from '@/service/types'

import PageSearch from '@/component/page-search.vue'

import BasicAuthEditModal from './widget/basic-auth-edit-modal.vue'

@Component({
    components: { BasicAuthEditModal, PageSearch }
})
class CaddyBasicAuth extends Vue {
    portal = usePortal()

    @Ref readonly editModalRef!: InstanceType<typeof BasicAuthEditModal>

    routes: CaddyBasicAuthRoute[] = []
    loading = false
    searchText = ''

    get filteredRoutes() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.routes
        return this.routes.filter(r =>
            (r.name || '').toLowerCase().includes(keyword) ||
            r.users.some(u => u.username.toLowerCase().includes(keyword))
        )
    }

    async load() {
        this.loading = true
        try {
            this.routes = (await api.caddyBasicAuthList()).payload || []
        } catch {
            this.portal.showNotification('error', '加载失败')
        } finally {
            this.loading = false
        }
    }

    getRouteName(r: CaddyBasicAuthRoute) {
        return r.name || `路由 #${r.index}`
    }

    getRouteDesc(r: CaddyBasicAuthRoute) {
        const proxy = r.handlers?.find(h => h.handler === 'reverse_proxy')
        if (proxy) {
            const ups = (proxy.upstreams as Array<{ dial: string }> | undefined) || []
            return ups.map(u => u.dial).join(', ') || 'reverse_proxy'
        }
        const fs = r.handlers?.find(h => h.handler === 'file_server')
        if (fs) return 'file_server'
        return r.handlers?.map(h => h.handler).join(', ') || '—'
    }

    openAddModal() {
        this.editModalRef?.showSetup()
    }

    openEditModal(route: CaddyBasicAuthRoute) {
        this.editModalRef?.showEdit(route)
    }

    openAddUserModal(route: CaddyBasicAuthRoute) {
        this.editModalRef?.showAddUser(route)
    }

    mounted() {
        this.load()
    }
}

export default toNative(CaddyBasicAuth)
</script>

<template>
  <div class="card">
    <!-- Toolbar -->
    <div class="card-toolbar">
      <!-- 桌面端 -->
      <div class="hidden md:flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-cyan-500">
            <i class="fas fa-lock text-white text-sm"></i>
          </div>
          <div>
            <h1 class="text-lg font-semibold text-slate-800 truncate">基础认证</h1>
            <p class="text-xs text-slate-500">基于用户名 + 密码的路由 HTTP Basic 认证</p>
          </div>
        </div>
        <div class="flex items-center gap-2 flex-shrink-0">
          <PageSearch v-model="searchText" search-key="caddy-basic-auth" placeholder="搜索路由或用户名..." focus-color="cyan" type-to-search />
          <button class="btn btn-secondary" @click="load()">
            <i class="fas fa-rotate"></i>刷新
          </button>
          <button v-if="portal.hasPerm('POST /api/caddy/basic-auth/:index/users')" class="btn btn-cyan" @click="openAddModal()">
            <i class="fas fa-plus"></i>配置认证
          </button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="flex md:hidden items-center justify-between">
        <div class="flex items-center gap-3 min-w-0 flex-1">
          <div class="page-icon bg-cyan-500">
            <i class="fas fa-lock text-white text-sm"></i>
          </div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">基础认证</h1>
            <p class="text-xs text-slate-500 truncate">路由 HTTP Basic 认证</p>
          </div>
        </div>
        <div class="flex items-center gap-1 flex-shrink-0">
          <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" @click="load()">
            <i class="fas fa-rotate text-sm"></i>
          </button>
          <button v-if="portal.hasPerm('POST /api/caddy/basic-auth/:index/users')" class="btn btn-cyan w-9 h-9 !px-0" title="配置认证" @click="openAddModal()">
            <i class="fas fa-plus text-sm"></i>
          </button>
        </div>
      </div>
    </div>

    <!-- 移动端搜索 -->
    <div class="mobile-search">
      <PageSearch v-model="searchText" search-key="caddy-basic-auth" placeholder="搜索路由或用户名..." width-class="w-full" focus-color="cyan" />
    </div>

    <!-- Loading -->
    <div v-if="loading" class="card-body">
      <div class="empty-state">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
    </div>

    <!-- Empty -->
    <div v-else-if="filteredRoutes.length === 0" class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-lock text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">
          {{ routes.length === 0 ? '暂无基础认证' : '未找到匹配路由' }}
        </p>
        <p class="text-sm text-slate-400">
          {{ routes.length === 0 ? '点击「配置认证」为任意路由启用 HTTP Basic 认证' : '尝试更换关键词或清空搜索条件' }}
        </p>
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
              <th class="th">转发头</th>
              <th class="th">账号</th>
              <th class="w-16 th-right">数量</th>
              <th class="w-24 th-right">操作</th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-slate-100">
            <tr v-for="route in filteredRoutes" :key="route.index" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3 max-w-[280px]">
                <div class="flex items-center gap-2 min-w-0">
                  <div class="row-icon bg-cyan-400">
                    <i class="fas fa-lock text-white text-sm"></i>
                  </div>
                  <div class="min-w-0">
                    <span class="font-medium text-slate-800 truncate block">{{ getRouteName(route) }}</span>
                    <span class="text-xs text-slate-400 truncate block mt-0.5">{{ getRouteDesc(route) }}</span>
                  </div>
                </div>
              </td>
              <td class="px-4 py-3 text-sm text-slate-600">
                <code v-if="route.forwardHeader" class="text-xs bg-slate-100 px-2 py-0.5 rounded-lg text-slate-600 font-mono">{{ route.forwardHeader }}</code>
                <span v-else>—</span>
              </td>
              <td class="px-4 py-3">
                <div class="flex flex-wrap gap-1.5">
                  <span
                    v-for="user in route.users"
                    :key="user.username"
                    class="inline-flex items-center gap-1.5 px-2 py-0.5 bg-cyan-50 text-cyan-800 rounded-lg text-xs group"
                  >
                    <i class="fas fa-user text-cyan-500 text-[10px]"></i>
                    <span>{{ user.username }}</span>
                  </span>
                </div>
              </td>
              <td class="px-4 py-3 text-right text-sm text-slate-600">{{ route.users.length }}</td>
              <td class="px-4 py-3">
                <div class="flex justify-end items-center gap-1">
                  <button
                    v-if="portal.hasPerm('PUT /api/caddy/basic-auth/:index/config')"
                    class="btn-icon btn-icon-blue"
                    title="编辑配置"
                    @click="openEditModal(route)"
                  >
                    <i class="fas fa-pen text-xs"></i>
                  </button>
                  <button
                    v-if="portal.hasPerm('POST /api/caddy/basic-auth/:index/users')"
                    class="btn-icon btn-icon-cyan"
                    title="添加账号"
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
              <span class="font-medium text-slate-800 text-sm truncate block">{{ getRouteName(route) }}</span>
              <span class="text-xs text-slate-400 truncate block mt-0.5">{{ getRouteDesc(route) }}</span>
            </div>
          </div>

          <div v-if="route.forwardHeader" class="card-prop-row-start">
            <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">转发头</span>
            <code class="text-xs bg-slate-100 px-2 py-0.5 rounded-lg text-slate-600 font-mono break-all">{{ route.forwardHeader }}</code>
          </div>

          <div class="card-prop-row-start">
            <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">账号</span>
            <div class="flex flex-wrap gap-1.5">
              <span
                v-for="user in route.users"
                :key="user.username"
                class="inline-flex items-center gap-1.5 px-2.5 py-1 bg-cyan-50 text-cyan-800 rounded-lg text-xs"
              >
                <i class="fas fa-user text-cyan-500 text-[10px]"></i>
                <span>{{ user.username }}</span>
              </span>
            </div>
          </div>

          <div class="card-actions">
            <button
              v-if="portal.hasPerm('PUT /api/caddy/basic-auth/:index/config')"
              class="btn-icon btn-icon-blue"
              title="编辑配置"
              @click="openEditModal(route)"
            >
              <i class="fas fa-pen text-xs"></i>
              <span class="text-xs ml-1">编辑配置</span>
            </button>
            <button
              v-if="portal.hasPerm('POST /api/caddy/basic-auth/:index/users')"
              class="btn-icon btn-icon-cyan"
              title="添加账号"
              @click="openAddUserModal(route)"
            >
              <i class="fas fa-user-plus text-xs"></i>
              <span class="text-xs ml-1">添加账号</span>
            </button>
          </div>
        </div>
      </div>
    </template>
  </div>

  <BasicAuthEditModal ref="editModalRef" @success="load" />
</template>
