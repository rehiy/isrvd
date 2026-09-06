<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { ApisixRoute } from '@/service/types'

import { formatRouteUpstreamNodes, formatRouteUpstreamSummary, normalizeUpstreamNodes } from '@/helper/apisix'

import PageSearch from '@/component/page-search.vue'

import RouteEditModal from './widget/route-edit-modal.vue'

type RouteListRow =
    | { type: 'host'; key: string; host: string; count: number; active: number; disabled: number }
    | { type: 'route'; key: string; host: string; route: ApisixRoute }

@Component({
    components: { PageSearch, RouteEditModal }
})
class Routes extends Vue {
    portal = usePortal()

    // ─── Refs ───
    @Ref readonly editModalRef!: InstanceType<typeof RouteEditModal>

    // ─── 数据属性 ───
    routes: ApisixRoute[] = []
    loading = false
    searchText = ''
    collapsedHosts: string[] = []
    formatRouteUpstreamNodes = formatRouteUpstreamNodes

    // ─── 计算属性 ───
    get filteredRoutes() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.routes
        return this.routes.filter((r: ApisixRoute) => {
            return (
                (r.name || '').toLowerCase().includes(keyword) ||
                (r.id || '').toLowerCase().includes(keyword) ||
                (r.uri || '').toLowerCase().includes(keyword) ||
                (r.uris || []).some((u: string) => u.toLowerCase().includes(keyword)) ||
                this.getRouteHost(r).toLowerCase().includes(keyword) ||
                (r.desc || '').toLowerCase().includes(keyword) ||
                formatRouteUpstreamSummary(r).toLowerCase().includes(keyword)
            )
        })
    }

    get groupByHost() {
        const counts = new Map<string, number>()
        for (const route of this.routes) {
            const host = this.getRouteGroupHost(route)
            counts.set(host, (counts.get(host) || 0) + 1)
        }
        return [...counts.values()].some(count => count > 1)
    }

    get routeListRows(): RouteListRow[] {
        const routes = [...this.filteredRoutes].sort((a, b) => {
            const hostCompare = this.getRouteGroupHost(a).localeCompare(this.getRouteGroupHost(b))
            return hostCompare || this.getRouteUri(a).localeCompare(this.getRouteUri(b))
        })

        // 统计每个 host 下的路由数量与状态分布
        const hostStats = new Map<string, { count: number; active: number; disabled: number }>()
        for (const route of routes) {
            const host = this.getRouteGroupHost(route)
            const stat = hostStats.get(host) || { count: 0, active: 0, disabled: 0 }
            stat.count += 1
            if (route.status === 1) stat.active += 1
            else stat.disabled += 1
            hostStats.set(host, stat)
        }

        const rows: RouteListRow[] = []
        let previousHost = ''
        for (const [index, route] of routes.entries()) {
            const host = this.getRouteGroupHost(route)
            if (this.groupByHost && host !== previousHost) {
                const stat = hostStats.get(host)
                if (stat) rows.push({ type: 'host', key: `host-${host}`, host, count: stat.count, active: stat.active, disabled: stat.disabled })
                previousHost = host
            }
            rows.push({ type: 'route', key: `route-${route.id || index}`, host, route })
        }
        return rows
    }

    // ─── 方法 ───
    async loadRoutes() {
        this.loading = true
        try {
            this.routes = (await api.apisixRouteList()).payload || []
        } catch {} finally {
            this.loading = false
        }
    }

    toggleHost(host: string) {
        if (this.collapsedHosts.includes(host)) {
            this.collapsedHosts = this.collapsedHosts.filter(item => item !== host)
            return
        }
        this.collapsedHosts = [...this.collapsedHosts, host]
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

    getRouteGroupHost(r: ApisixRoute) {
        return r.hosts?.[0] || r.host || '*'
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
  <div class="page">
    <div class="page-toolbar">
      <!-- 桌面端 -->
      <div class="toolbar-desktop">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-indigo-500"><i class="fas fa-route text-white"></i></div>
          <div><h1 class="title-text">路由</h1><p class="text-xs text-slate-500">管理 APISIX 路由，配置匹配规则、上游转发与插件</p></div>
        </div>
        <div class="action-group">
          <PageSearch v-model="searchText" search-key="apisix-routes" placeholder="搜索路由、URI、描述或上游..." focus-color="indigo" type-to-search />
          <button class="btn btn-secondary" @click="loadRoutes()"><i class="fas fa-rotate"></i>刷新</button>
          <button v-if="portal.hasPerm('POST /api/apisix/route')" class="btn btn-indigo" @click="openCreateModal()"><i class="fas fa-plus"></i>新建路由</button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="toolbar-mobile">
        <div class="title-group">
          <div class="page-icon bg-indigo-500"><i class="fas fa-route text-white"></i></div>
          <div class="min-w-0">
            <h1 class="title-text">路由</h1>
            <p class="text-xs text-slate-500 truncate">配置匹配规则、上游与插件</p>
          </div>
        </div>
        <div class="action-group-sm">
          <button class="btn btn-secondary btn-square" title="刷新" @click="loadRoutes()">
            <i class="fas fa-rotate text-sm"></i>
          </button>
          <button v-if="portal.hasPerm('POST /api/apisix/route')" class="btn btn-indigo btn-square" title="新建路由" @click="openCreateModal()">
            <i class="fas fa-plus text-sm"></i>
          </button>
        </div>
      </div>
    </div>
    <!-- 移动端搜索栏 -->
    <div class="mobile-search">
      <PageSearch v-model="searchText" search-key="apisix-routes" placeholder="搜索路由、URI、上游..." width-class="w-full" focus-color="indigo" />
    </div>
    <div v-if="loading" class="card-body">
      <div class="empty-state"><div class="spinner-lg"></div><p class="text-slate-500">加载中...</p></div>
    </div>
    <div v-else-if="filteredRoutes.length === 0" class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon"><i class="fas fa-route text-4xl text-slate-300"></i></div>
        <p class="text-slate-600 font-medium mb-1">{{ routes.length === 0 ? '暂无路由' : '未找到匹配路由' }}</p>
        <p class="text-sm text-slate-400">{{ routes.length === 0 ? '点击「新建路由」开始创建' : '尝试更换关键词或清空搜索条件' }}</p>
      </div>
    </div>

    <template v-else>
      <!-- 桌面端表格视图 -->
      <div class="card-table hidden md:block">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-100 border-b border-slate-200">
              <th class="th">名称</th>
              <th class="th">Host</th>
              <th class="th">URI</th>
              <th class="th">上游</th>
              <th class="w-40 th-right">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <template v-for="row in routeListRows" :key="row.key">
              <tr v-if="row.type === 'host'" class="bg-slate-50">
                <td colspan="5" class="px-4 py-2">
                  <button type="button" class="flex w-full items-center gap-2 text-left" :aria-expanded="!collapsedHosts.includes(row.host)" @click="toggleHost(row.host)">
                    <i class="fas fa-chevron-right text-slate-400 text-xs transition-transform" :class="{ 'rotate-90': !collapsedHosts.includes(row.host) }"></i>
                    <i class="fas fa-globe text-teal-500 text-xs"></i>
                    <span class="text-xs text-slate-400">Host</span>
                    <span :class="row.host === '*' ? 'text-slate-400' : 'text-teal-600 font-medium'" class="text-sm">{{ row.host }}</span>
                    <span class="ml-auto flex items-center gap-3 text-xs">
                      <span class="text-slate-400">共 <span class="font-medium text-slate-600">{{ row.count }}</span> 条</span>
                      <span class="text-emerald-600">启用 {{ row.active }}</span>
                      <span class="text-amber-600">禁用 {{ row.disabled }}</span>
                    </span>
                  </button>
                </td>
              </tr>
              <tr v-else v-show="!groupByHost || !collapsedHosts.includes(row.host)" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3 max-w-[280px]">
                  <div class="inline-info">
                    <div class="row-icon bg-indigo-400">
                      <i class="fas fa-route text-white text-sm"></i>
                    </div>
                    <div class="min-w-0">
                      <span class="item-title">{{ row.route.name || row.route.id }}</span>
                      <span v-if="row.route.desc" class="item-subtitle">{{ row.route.desc }}</span>
                    </div>
                  </div>
                </td>
                <td class="px-4 py-3"><span :class="getRouteHost(row.route) === '*' ? 'text-slate-400' : 'text-teal-600 font-medium'" class="text-sm break-all">{{ getRouteHost(row.route) }}</span></td>
                <td class="px-4 py-3"><code class="text-xs font-mono text-slate-700 break-all">{{ getRouteUri(row.route) }}</code></td>
                <td class="px-4 py-3"><span :class="getRouteUpstreamTagClass(row.route)" class="code-chip">{{ formatRouteUpstreamNodes(row.route) }}</span></td>
                <td class="px-4 py-3">
                  <div class="table-actions">
                    <button v-if="portal.hasPerm('PATCH /api/apisix/route/:id/status')" :class="['btn-icon', row.route.status === 1 ? 'btn-icon-amber' : 'btn-icon-emerald']" :title="row.route.status === 1 ? '禁用' : '启用'" @click="toggleStatus(row.route)">
                      <i :class="row.route.status === 1 ? 'fas fa-ban' : 'fas fa-play'" class="text-xs"></i>
                    </button>
                    <button v-if="portal.hasPerm('PUT /api/apisix/route/:id')" class="btn-icon btn-icon-indigo" title="编辑" @click="openEditModal(row.route)"><i class="fas fa-pen text-xs"></i></button>
                    <button v-if="portal.hasPerm('DELETE /api/apisix/route/:id')" class="btn-icon btn-icon-red" title="删除" @click="deleteRoute(row.route)"><i class="fas fa-trash text-xs"></i></button>
                  </div>
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>

      <!-- 移动端卡片视图 -->
      <div class="card-body md:hidden space-y-3">
        <template v-for="row in routeListRows" :key="row.key">
          <button v-if="row.type === 'host'" type="button" class="flex w-full items-center gap-2 border-y border-slate-100 bg-slate-50 px-3 py-2 text-left" :aria-expanded="!collapsedHosts.includes(row.host)" @click="toggleHost(row.host)">
            <i class="fas fa-chevron-right text-slate-400 text-xs transition-transform" :class="{ 'rotate-90': !collapsedHosts.includes(row.host) }"></i>
            <i class="fas fa-globe text-teal-500 text-xs"></i>
            <span class="text-xs text-slate-400">Host</span>
            <span :class="row.host === '*' ? 'text-slate-400' : 'text-teal-600 font-medium'" class="text-sm truncate">{{ row.host }}</span>
            <span class="ml-auto flex items-center gap-2 text-xs whitespace-nowrap">
              <span class="text-slate-400">共 <span class="font-medium text-slate-600">{{ row.count }}</span></span>
              <span class="text-emerald-600">{{ row.active }} 启用</span>
              <span class="text-amber-600">{{ row.disabled }} 禁用</span>
            </span>
          </button>
          <div v-else v-show="!groupByHost || !collapsedHosts.includes(row.host)" class="card-interactive">
            <!-- 顶部：路由信息和状态 -->
            <div class="flex items-center justify-between mb-3">
              <div class="title-group">
                <div class="list-icon bg-indigo-400">
                  <i class="fas fa-route text-white text-base"></i>
                </div>
                <div class="min-w-0">
                  <div class="font-medium text-sm text-slate-800 truncate">{{ row.route.name || row.route.id }}</div>
                  <div v-if="row.route.desc" class="text-xs text-slate-400 mt-0.5 truncate">{{ row.route.desc }}</div>
                </div>
              </div>
            </div>

            <!-- 中间：URI和Host信息 -->
            <div class="card-prop-row-start">
              <span class="prop-label-start">URI</span>
              <code class="text-xs font-mono text-slate-700 break-all">{{ getRouteUri(row.route) }}</code>
            </div>

            <div class="card-prop-row">
              <span class="text-xs text-slate-400 flex-shrink-0">Host</span>
              <span :class="getRouteHost(row.route) === '*' ? 'text-slate-400' : 'text-teal-600 font-medium'" class="text-xs break-all">{{ getRouteHost(row.route) }}</span>
            </div>

            <div class="card-prop-row-start">
              <span class="prop-label-start">上游</span>
              <span :class="getRouteUpstreamTagClass(row.route)" class="code-chip">{{ formatRouteUpstreamNodes(row.route) }}</span>
            </div>

            <!-- 底部：操作按钮 -->
            <div class="card-actions">
              <button v-if="portal.hasPerm('PATCH /api/apisix/route/:id/status')" :class="['btn-icon', row.route.status === 1 ? 'btn-icon-amber' : 'btn-icon-emerald']" :title="row.route.status === 1 ? '禁用' : '启用'" @click="toggleStatus(row.route)">
                <i :class="row.route.status === 1 ? 'fas fa-ban' : 'fas fa-play'" class="text-xs"></i><span class="text-xs ml-1">{{ row.route.status === 1 ? '禁用' : '启用' }}</span>
              </button>
              <button v-if="portal.hasPerm('PUT /api/apisix/route/:id')" class="btn-icon btn-icon-indigo" title="编辑" @click="openEditModal(row.route)">
                <i class="fas fa-pen text-xs"></i><span class="text-xs ml-1">编辑</span>
              </button>
              <button v-if="portal.hasPerm('DELETE /api/apisix/route/:id')" class="btn-icon btn-icon-red" title="删除" @click="deleteRoute(row.route)">
                <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
              </button>
            </div>
          </div>
        </template>
      </div>
    </template>
  </div>

  <RouteEditModal ref="editModalRef" @success="loadRoutes" />
</template>
