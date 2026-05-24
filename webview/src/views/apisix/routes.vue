<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { ApisixRoute } from '@/service/types'

import { formatRouteUpstreamSummary, formatRouteUpstreamType, formatRouteUpstreamNodes, normalizeUpstreamNodes } from '@/helper/apisix'

import PageSearch from '@/component/page-search.vue'

import { usePortal } from '@/stores'

import RouteEditModal from './widget/route-edit-modal.vue'
import RouteGroupedList from './widget/route-grouped-list.vue'

type ViewMode = 'route' | 'host'

@Component({
    components: { PageSearch, RouteEditModal, RouteGroupedList }
})
class Routes extends Vue {
    portal = usePortal()

    // ─── Refs ───
    @Ref readonly editModalRef!: InstanceType<typeof RouteEditModal>

    // ─── 数据属性 ───
    routes: ApisixRoute[] = []
    loading = false
    searchText = ''
    viewMode: ViewMode = 'route'

    // ─── 计算属性 ───
    get filteredRoutes() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.routes
        return this.routes.filter((r: ApisixRoute) => {
            const upstreamSummary = this.getRouteUpstreamSummary(r).toLowerCase()
            return (
                (r.name || '').toLowerCase().includes(keyword) ||
                (r.id || '').toLowerCase().includes(keyword) ||
                (r.uri || '').toLowerCase().includes(keyword) ||
                (r.uris || []).some((u: string) => u.toLowerCase().includes(keyword)) ||
                (r.desc || '').toLowerCase().includes(keyword) ||
                upstreamSummary.includes(keyword)
            )
        })
    }

    // ─── 方法 ───
    sortRoutes(data: ApisixRoute[]) {
        data.sort((a: ApisixRoute, b: ApisixRoute) => {
            const hostA = (a.hosts?.[0]) || a.host || ''
            const hostB = (b.hosts?.[0]) || b.host || ''
            const hc = hostA.localeCompare(hostB)
            if (hc !== 0) return hc
            const uriA = (a.uris?.[0]) || a.uri || ''
            const uriB = (b.uris?.[0]) || b.uri || ''
            return uriA.localeCompare(uriB)
        })
        return data
    }

    async loadRoutes() {
        this.loading = true
        try {
            this.routes = this.sortRoutes((await api.apisixRouteList()).payload || [])
        } catch {
            this.portal.showNotification('error', '加载路由列表失败')
        } finally {
            this.loading = false
        }
    }

    setViewMode(mode: ViewMode) {
        this.viewMode = mode
    }

    openCreateModal() {
        this.editModalRef?.show(null, this.routes)
    }

    openEditModal(route: ApisixRoute | null) {
        this.editModalRef?.show(route, this.routes)
    }

    getRouteUri(r: ApisixRoute) {
        return r.uris?.length ? r.uris.join(', ') : (r.uri || '-')
    }

    getRouteHost(r: ApisixRoute) {
        return r.hosts?.length ? r.hosts.join(', ') : (r.host || '*')
    }

    getRouteUpstreamSummary(r: ApisixRoute) {
        return formatRouteUpstreamSummary(r)
    }

    getRouteUpstreamType(r: ApisixRoute) {
        return formatRouteUpstreamType(r)
    }

    getRouteUpstreamNodes(r: ApisixRoute) {
        return formatRouteUpstreamNodes(r)
    }

    getRouteUpstreamTagClass(r: ApisixRoute) {
        if (r.upstream_id) return 'bg-emerald-50 text-emerald-700'
        if (normalizeUpstreamNodes(r.upstream).length > 0) return 'bg-indigo-50 text-indigo-700'
        return 'bg-slate-100 text-slate-500'
    }

    toggleStatus(route: ApisixRoute) {
        const id = route.id
        if (!id) return
        const ns = route.status === 1 ? 0 : 1
        const label = ns === 1 ? '启用' : '禁用'
        this.portal.showConfirm({
            title: `${label}路由`,
            message: `确定要${label}路由 <strong class="text-slate-900">${route.name}</strong> 吗？`,
            icon: ns === 1 ? 'fa-toggle-on' : 'fa-toggle-off',
            iconColor: ns === 1 ? 'emerald' : 'amber',
            confirmText: `确认${label}`,
            onConfirm: async () => {
                await api.apisixRouteStatus(id, ns)
                this.portal.showNotification('success', `路由已${label}`)
                this.loadRoutes()
            }
        })
    }

    deleteRoute(route: ApisixRoute) {
        const id = route.id
        if (!id) return
        this.portal.showConfirm({
            title: '删除路由',
            message: `确定要删除路由 <strong class="text-slate-900">${route.name || id}</strong> 吗？此操作不可恢复。`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                await api.apisixRouteDelete(id)
                this.portal.showNotification('success', '删除成功')
                this.loadRoutes()
            }
        })
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadRoutes()
    }
}

export default toNative(Routes)
</script>

<template>
  <div>
    <div class="card mb-4">
      <div class="card-toolbar">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="page-icon bg-indigo-500"><i class="fas fa-route text-white"></i></div>
            <div><h1 class="text-lg font-semibold text-slate-800">路由管理</h1><p class="text-xs text-slate-500">管理 APISIX 路由，配置匹配规则、上游转发与插件</p></div>
          </div>
          <div class="flex items-center gap-2">
            <PageSearch v-model="searchText" search-key="apisix-routes" placeholder="搜索路由、URI、描述或上游..." width-class="w-64" focus-color="indigo" type-to-search />
            <div class="tab-group">
              <button class="tab-btn" :class="viewMode === 'route' ? 'tab-btn-active text-indigo-600' : 'tab-btn-inactive'" @click="setViewMode('route')">
                <i class="fas fa-list text-xs"></i>默认
              </button>
              <button class="tab-btn" :class="viewMode === 'host' ? 'tab-btn-active text-indigo-600' : 'tab-btn-inactive'" @click="setViewMode('host')">
                <i class="fas fa-layer-group text-xs"></i>Host 分组
              </button>
            </div>
            <button class="btn btn-secondary" @click="loadRoutes()"><i class="fas fa-rotate"></i>刷新</button>
            <button v-if="portal.hasPerm('POST /api/apisix/route')" class="btn btn-indigo" @click="openCreateModal()"><i class="fas fa-plus"></i>新建路由</button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="page-icon bg-indigo-500"><i class="fas fa-route text-white"></i></div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">路由管理</h1>
              <p class="text-xs text-slate-500 truncate">配置匹配规则、上游与插件</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadRoutes()">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button v-if="portal.hasPerm('POST /api/apisix/route')" class="btn btn-indigo w-9 h-9 !px-0" title="新建路由" @click="openCreateModal()">
              <i class="fas fa-plus text-sm"></i>
            </button>
          </div>
        </div>
      </div>
      <!-- 移动端搜索栏 -->
      <div class="mobile-search space-y-2">
        <PageSearch v-model="searchText" search-key="apisix-routes" placeholder="搜索路由、URI、上游..." width-class="w-full" focus-color="indigo" />
        <div class="tab-group w-full">
            <button class="tab-btn flex-1 justify-center" :class="viewMode === 'route' ? 'tab-btn-active text-indigo-600' : 'tab-btn-inactive'" @click="setViewMode('route')">
              <i class="fas fa-list text-xs"></i>默认
            </button>
            <button class="tab-btn flex-1 justify-center" :class="viewMode === 'host' ? 'tab-btn-active text-indigo-600' : 'tab-btn-inactive'" @click="setViewMode('host')">
              <i class="fas fa-layer-group text-xs"></i>Host 分组
            </button>
          </div>
      </div>
      <div v-if="loading" class="empty-state"><div class="w-12 h-12 spinner mb-3"></div><p class="text-slate-500">加载中...</p></div>
      <div v-else-if="filteredRoutes.length === 0" class="empty-state">
        <div class="empty-state-icon"><i class="fas fa-route text-4xl text-slate-300"></i></div>
        <p class="text-slate-600 font-medium mb-1">{{ routes.length === 0 ? '暂无路由' : '未找到匹配路由' }}</p>
        <p class="text-sm text-slate-400">{{ routes.length === 0 ? '点击「新建路由」开始创建' : '尝试更换关键词或清空搜索条件' }}</p>
      </div>

      <RouteGroupedList
        v-else-if="viewMode === 'host'"
        :routes="routes"
        :search-text="searchText"
        @toggle-status="toggleStatus"
        @edit="openEditModal"
        @delete="deleteRoute"
      />

      <div v-else class="space-y-3">
        <!-- 桌面端表格视图 -->
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="th">名称</th>
                <th class="th">Host</th>
                <th class="th">URI</th>
                <th class="th">策略</th>
                <th class="th">上游</th>
                <th class="w-40 th-right">操作</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="route in filteredRoutes" :key="route.id" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3 max-w-[280px]">
                  <div class="flex items-center gap-2 min-w-0">
                    <div class="row-icon bg-indigo-400">
                      <i class="fas fa-route text-white text-sm"></i>
                    </div>
                    <div class="min-w-0">
                      <span class="font-medium text-slate-800 truncate block">{{ route.name || route.id }}</span>
                      <span v-if="route.desc" class="text-xs text-slate-400 truncate block mt-0.5">{{ route.desc }}</span>
                    </div>
                  </div>
                </td>
                <td class="px-4 py-3"><span :class="getRouteHost(route) === '*' ? 'text-slate-400' : 'text-teal-600 font-medium'" class="text-sm break-all">{{ getRouteHost(route) }}</span></td>
                <td class="px-4 py-3"><code class="text-xs font-mono text-slate-700 break-all">{{ getRouteUri(route) }}</code></td>
                <td class="px-4 py-3"><span class="text-xs text-slate-600">{{ getRouteUpstreamType(route) || '-' }}</span></td>
                <td class="px-4 py-3"><span :class="getRouteUpstreamTagClass(route)" class="inline-block text-xs px-2 py-0.5 rounded-lg font-mono break-all">{{ getRouteUpstreamNodes(route) }}</span></td>
                <td class="px-4 py-3">
                  <div class="flex justify-end items-center gap-1">
                    <button v-if="portal.hasPerm('PATCH /api/apisix/route/:id/status')" :class="['btn-icon', route.status === 1 ? 'text-amber-500 hover:bg-amber-50' : 'text-emerald-600 hover:bg-emerald-50']" :title="route.status === 1 ? '禁用' : '启用'" @click="toggleStatus(route)">
                      <i :class="route.status === 1 ? 'fas fa-ban' : 'fas fa-play'" class="text-xs"></i>
                    </button>
                    <button v-if="portal.hasPerm('PUT /api/apisix/route/:id')" class="btn-icon btn-icon-indigo" title="编辑" @click="openEditModal(route)"><i class="fas fa-pen text-xs"></i></button>
                    <button v-if="portal.hasPerm('DELETE /api/apisix/route/:id')" class="btn-icon btn-icon-red" title="删除" @click="deleteRoute(route)"><i class="fas fa-trash text-xs"></i></button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 移动端卡片视图 -->
        <div class="md:hidden space-y-3 p-4">
          <div v-for="route in filteredRoutes" :key="route.id" class="card-interactive">
            <!-- 顶部：路由信息和状态 -->
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-3 min-w-0 flex-1">
                <div class="list-icon bg-indigo-400">
                  <i class="fas fa-route text-white text-base"></i>
                </div>
                <div class="min-w-0">
                  <div class="font-medium text-sm text-slate-800 truncate">{{ route.name || route.id }}</div>
                  <div v-if="route.desc" class="text-xs text-slate-400 mt-0.5 truncate">{{ route.desc }}</div>
                </div>
              </div>
            </div>

            <!-- 中间：URI和Host信息 -->
            <div class="flex items-start gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">URI</span>
              <code class="text-xs font-mono text-slate-700 break-all">{{ getRouteUri(route) }}</code>
            </div>

            <div class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">Host</span>
              <span :class="getRouteHost(route) === '*' ? 'text-slate-400' : 'text-teal-600 font-medium'" class="text-xs break-all">{{ getRouteHost(route) }}</span>
            </div>

            <div class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">策略</span>
              <span class="text-xs text-slate-500">{{ getRouteUpstreamType(route) || '-' }}</span>
            </div>
            <div class="flex items-start gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">上游</span>
              <span :class="getRouteUpstreamTagClass(route)" class="inline-block text-xs px-2 py-0.5 rounded-lg font-mono break-all">{{ getRouteUpstreamNodes(route) }}</span>
            </div>

            <!-- 底部：操作按钮 -->
            <div class="card-actions">
              <button v-if="portal.hasPerm('PATCH /api/apisix/route/:id/status')" :class="['btn-icon', route.status === 1 ? 'text-amber-500 hover:bg-amber-50' : 'text-emerald-600 hover:bg-emerald-50']" :title="route.status === 1 ? '禁用' : '启用'" @click="toggleStatus(route)">
                <i :class="route.status === 1 ? 'fas fa-ban' : 'fas fa-play'" class="text-xs"></i><span class="text-xs ml-1">{{ route.status === 1 ? '禁用' : '启用' }}</span>
              </button>
              <button v-if="portal.hasPerm('PUT /api/apisix/route/:id')" class="btn-icon btn-icon-indigo" title="编辑" @click="openEditModal(route)">
                <i class="fas fa-pen text-xs"></i><span class="text-xs ml-1">编辑</span>
              </button>
              <button v-if="portal.hasPerm('DELETE /api/apisix/route/:id')" class="btn-icon btn-icon-red" title="删除" @click="deleteRoute(route)">
                <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <RouteEditModal ref="editModalRef" @success="loadRoutes" />
  </div>
</template>
