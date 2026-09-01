<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { CaddyHandler, CaddyRoute, CaddyServerInfo } from '@/service/types'

import PageSearch from '@/component/page-search.vue'

import RouteEditModal from './widget/route-edit-modal.vue'

const handlerLabels: Record<string, string> = {
    reverse_proxy: '反向代理',
    file_server: '文件服务',
    static_response: '静态响应',
    raw: '原始 JSON'
}

const escapeHTML = (value: string) => value.replace(/[&<>"']/g, char => ({
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;'
})[char] || char)

type RouteListRow =
    | { type: 'host'; key: string; host: string; count: number }
    | { type: 'route'; key: string; host: string; grouped: boolean; route: RouteWithServer }

type RouteWithServer = CaddyRoute & {
    serverID: string
    serverName: string
}

@Component({
    components: { PageSearch, RouteEditModal }
})
class CaddyRoutes extends Vue {
    portal = usePortal()

    @Ref readonly editModalRef!: InstanceType<typeof RouteEditModal>

    routes: RouteWithServer[] = []
    servers: CaddyServerInfo[] = []
    loading = false
    searchText = ''
    collapsedHosts: string[] = []
    loadRequestID = 0

    get canListServers() {
        return this.portal.hasPerm('GET /api/caddy/servers')
    }

    get routeServerOptions() {
        if (this.servers.length > 0) {
            return this.servers.map(server => ({ id: server.id, name: server.name }))
        }
        return [{ id: 'srv0', name: 'srv0' }]
    }

    get filteredRoutes() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.routes
        return this.routes.filter((r: RouteWithServer) => {
            const m = r.match?.[0]
            const handler = this.getTerminalHandler(r)
            const ups = (handler?.upstreams as Array<{ dial: string }> | undefined) || []
            return (
                (m?.host || []).some((s: string) => s.toLowerCase().includes(keyword)) ||
                (m?.path || []).some((s: string) => s.toLowerCase().includes(keyword)) ||
                r.serverName.toLowerCase().includes(keyword) ||
                ups.some(upstream => upstream.dial?.toLowerCase().includes(keyword)) ||
                String(handler?.root || '').toLowerCase().includes(keyword) ||
                (handler?.handler || '').toLowerCase().includes(keyword)
            )
        })
    }

    get routeListRows(): RouteListRow[] {
        const allEntries = this.routes.flatMap(route =>
            this.getRouteGroupHosts(route).map(host => ({ host, route }))
        )
        const grouped = this.routes.some(route => this.getRouteGroupHosts(route).length > 1) ||
            new Set(allEntries.map(entry => entry.host)).size < allEntries.length
        const entries = this.filteredRoutes.flatMap(route =>
            this.getRouteGroupHosts(route).map(host => ({ host, route }))
        )
        if (grouped) {
            entries.sort((a, b) => {
                const hostCompare = a.host.localeCompare(b.host)
                const serverCompare = a.route.serverName.localeCompare(b.route.serverName)
                return hostCompare || serverCompare || this.getRoutePaths(a.route).localeCompare(this.getRoutePaths(b.route))
            })
        }
        const counts = new Map<string, number>()
        for (const { host } of entries) {
            counts.set(host, (counts.get(host) || 0) + 1)
        }
        const rows: RouteListRow[] = []
        let previousHost = ''
        for (const { host, route } of entries) {
            if (grouped && host !== previousHost) {
                rows.push({ type: 'host', key: `host-${host}`, host, count: counts.get(host) || 0 })
                previousHost = host
            }
            rows.push({ type: 'route', key: `route-${route.serverID}-${host}-${route.index}`, host, grouped, route })
        }
        return rows
    }

    async loadRoutes() {
        const requestID = ++this.loadRequestID
        this.loading = true
        this.routes = []
        this.collapsedHosts = []
        try {
            if (this.canListServers) {
                try {
                    const servers = (await api.caddyServerList()).payload || []
                    if (requestID !== this.loadRequestID) return
                    this.servers = servers
                } catch {
                    if (requestID !== this.loadRequestID) return
                    this.servers = []
                }
            } else {
                this.servers = []
            }
            const routesByServer = await Promise.all(this.routeServerOptions.map(async server => {
                const routes = (await api.caddyRouteList(server.name)).payload || []
                return routes.map(route => ({ ...route, serverID: server.id, serverName: server.name }))
            }))
            if (requestID !== this.loadRequestID) return
            this.routes = routesByServer.flat()
            this.collapsedHosts = []
        } catch {} finally {
            if (requestID === this.loadRequestID) this.loading = false
        }
    }

    openCreateModal() {
        this.editModalRef?.show(null)
    }

    toggleHost(host: string) {
        this.collapsedHosts = this.collapsedHosts.includes(host)
            ? this.collapsedHosts.filter(item => item !== host)
            : [...this.collapsedHosts, host]
    }

    openEditModal(route: RouteWithServer) {
        this.editModalRef?.show(route, route.serverName)
    }

    getRouteHosts(r: CaddyRoute) {
        const hosts = r.match?.[0]?.host || []
        return hosts.length ? hosts.join(', ') : '*'
    }

    getRouteGroupHosts(r: CaddyRoute) {
        const hosts = r.match?.[0]?.host || []
        return hosts.length ? [...new Set(hosts)] : ['*']
    }

    getRoutePaths(r: CaddyRoute) {
        const paths = r.match?.[0]?.path || []
        return paths.length ? paths.join(', ') : '/*'
    }

    getRouteMethods(r: CaddyRoute) {
        const methods = r.match?.[0]?.method || []
        return methods.length ? methods.join(' ') : 'ANY'
    }

    getHandlerKindLabel(kind?: string) {
        return handlerLabels[kind || ''] || kind || '-'
    }

    getTerminalHandler(r: CaddyRoute): CaddyHandler | undefined {
        if (!r.handle?.length) return undefined
        return r.handle[r.handle.length - 1]
    }

    getHandlerSummary(r: CaddyRoute) {
        const h = this.getTerminalHandler(r)
        if (!h) return '-'
        switch (h.handler) {
            case 'reverse_proxy': {
                const ups = (h.upstreams as Array<{ dial: string }> | undefined) || []
                return ups.map(u => u.dial || '').filter(Boolean).join(', ') || '-'
            }
            case 'file_server': return (h.root as string) || '-'
            case 'static_response': return `${h.status_code || 200}${h.body ? ' +' : ''}`
        }
        return '(自定义)'
    }

    getHandlerTagClass(r: CaddyRoute) {
        const kind = this.getTerminalHandler(r)?.handler
        if (kind === 'reverse_proxy') return 'bg-indigo-50 text-indigo-700'
        if (kind === 'file_server') return 'bg-emerald-50 text-emerald-700'
        if (kind === 'static_response') return 'bg-amber-50 text-amber-700'
        return 'bg-slate-100 text-slate-500'
    }

    deleteRoute(route: RouteWithServer) {
        const serverName = escapeHTML(route.serverName)
        this.portal.showConfirm({
            title: '删除路由',
            message: `确定要删除服务 <strong class="text-slate-900">${serverName}</strong> 下的路由 <strong class="text-slate-900">#${route.index}</strong> 吗？此操作不可恢复。`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                try {
                    await api.caddyRouteDelete(route.index, route.serverName)
                    this.portal.showNotification('success', '删除成功')
                    this.loadRoutes()
                } catch {}
            }
        })
    }

    mounted() {
        this.loadRoutes()
    }
}

export default toNative(CaddyRoutes)
</script>

<template>
  <div class="page">
    <div class="page-toolbar">
      <!-- 桌面端 -->
      <div class="toolbar-desktop">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-indigo-500"><i class="fas fa-route text-white"></i></div>
          <div class="min-w-0"><h1 class="title-text">路由</h1><p class="text-xs text-slate-500 truncate">配置请求匹配规则与处理器，支持多种转发方式</p></div>
        </div>
        <div class="action-group">
          <PageSearch v-model="searchText" search-key="caddy-routes" placeholder="请输入搜索关键词..." focus-color="indigo" type-to-search />
          <button class="btn btn-secondary" @click="loadRoutes()"><i class="fas fa-rotate"></i>刷新</button>
          <button v-if="portal.hasPerm('POST /api/caddy/route')" class="btn btn-indigo" @click="openCreateModal()"><i class="fas fa-plus"></i>新建路由</button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="toolbar-mobile">
        <div class="title-group">
          <div class="page-icon bg-indigo-500"><i class="fas fa-route text-white"></i></div>
          <div class="min-w-0">
            <h1 class="title-text">路由</h1>
            <p class="text-xs text-slate-500 truncate">配置匹配规则与处理器</p>
          </div>
        </div>
        <div class="action-group-sm">
          <button class="btn btn-secondary btn-square" title="刷新" @click="loadRoutes()">
            <i class="fas fa-rotate text-sm"></i>
          </button>
          <button v-if="portal.hasPerm('POST /api/caddy/route')" class="btn btn-indigo btn-square" title="新建路由" @click="openCreateModal()">
            <i class="fas fa-plus text-sm"></i>
          </button>
        </div>
      </div>
    </div>
    <!-- 移动端搜索栏 -->
    <div class="mobile-search">
      <PageSearch v-model="searchText" search-key="caddy-routes" placeholder="请输入搜索关键词..." width-class="w-full" focus-color="indigo" />
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
      <!-- 桌面端表格 -->
      <div class="card-table hidden md:block">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-100 border-b border-slate-200">
              <th class="th">Host</th>
              <th class="th">服务</th>
              <th class="th">Path</th>
              <th class="th">Method</th>
              <th class="th">类型</th>
              <th class="th">处理器</th>
              <th class="w-32 th-right">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <template v-for="row in routeListRows" :key="row.key">
              <tr v-if="row.type === 'host'" class="bg-slate-50">
                <td colspan="7" class="px-4 py-2">
                  <button type="button" class="flex w-full items-center gap-2 text-left" :aria-expanded="!collapsedHosts.includes(row.host)" @click="toggleHost(row.host)">
                    <i class="fas fa-chevron-right text-slate-400 text-xs transition-transform" :class="{ 'rotate-90': !collapsedHosts.includes(row.host) }"></i>
                    <i class="fas fa-globe text-teal-500 text-xs"></i>
                    <span class="text-xs text-slate-400">Host</span>
                    <span :class="row.host === '*' ? 'text-slate-400' : 'text-teal-600 font-medium'" class="text-sm">{{ row.host }}</span>
                    <span class="ml-auto text-xs text-slate-400">共 <span class="font-medium text-slate-600">{{ row.count }}</span> 条</span>
                  </button>
                </td>
              </tr>
              <tr v-else v-show="!row.grouped || !collapsedHosts.includes(row.host)" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3">
                  <div class="inline-info">
                    <div class="row-icon bg-indigo-400 flex-shrink-0">
                      <i class="fas fa-route text-white text-sm"></i>
                    </div>
                    <span :class="getRouteHosts(row.route) === '*' ? 'text-slate-400' : 'text-teal-600 font-medium'" class="text-sm break-all">{{ getRouteHosts(row.route) }}</span>
                  </div>
                </td>
                <td class="td-text"><span class="font-mono">{{ row.route.serverName }}</span></td>
                <td class="px-4 py-3"><code class="text-xs font-mono text-slate-700 break-all">{{ getRoutePaths(row.route) }}</code></td>
                <td class="px-4 py-3"><span class="text-xs text-slate-600">{{ getRouteMethods(row.route) }}</span></td>
                <td class="px-4 py-3"><span :class="getHandlerTagClass(row.route)" class="inline-block text-xs px-2 py-0.5 rounded-lg">{{ getHandlerKindLabel(getTerminalHandler(row.route)?.handler as string) }}</span></td>
                <td class="px-4 py-3"><code class="text-xs font-mono text-slate-700 break-all">{{ getHandlerSummary(row.route) }}</code></td>
                <td class="px-4 py-3">
                  <div class="table-actions">
                    <button v-if="portal.hasPerm('PUT /api/caddy/route/:index')" class="btn-icon btn-icon-blue" title="编辑" @click="openEditModal(row.route)"><i class="fas fa-pen text-xs"></i></button>
                    <button v-if="portal.hasPerm('DELETE /api/caddy/route/:index')" class="btn-icon btn-icon-red" title="删除" @click="deleteRoute(row.route)"><i class="fas fa-trash text-xs"></i></button>
                  </div>
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>

      <!-- 移动端卡片 -->
      <div class="card-body md:hidden space-y-3">
        <template v-for="row in routeListRows" :key="row.key">
          <button v-if="row.type === 'host'" type="button" class="flex w-full items-center gap-2 border-y border-slate-100 bg-slate-50 px-3 py-2 text-left" :aria-expanded="!collapsedHosts.includes(row.host)" @click="toggleHost(row.host)">
            <i class="fas fa-chevron-right text-slate-400 text-xs transition-transform" :class="{ 'rotate-90': !collapsedHosts.includes(row.host) }"></i>
            <i class="fas fa-globe text-teal-500 text-xs"></i>
            <span class="text-xs text-slate-400">Host</span>
            <span :class="row.host === '*' ? 'text-slate-400' : 'text-teal-600 font-medium'" class="text-sm truncate">{{ row.host }}</span>
            <span class="ml-auto text-xs text-slate-400 whitespace-nowrap">共 <span class="font-medium text-slate-600">{{ row.count }}</span> 条</span>
          </button>
          <div v-else v-show="!row.grouped || !collapsedHosts.includes(row.host)" class="card-interactive">
            <div class="card-info-row">
              <div class="list-icon bg-indigo-400">
                <i class="fas fa-route text-white text-base"></i>
              </div>
              <div class="min-w-0">
                <span class="item-title-sm">{{ getRouteHosts(row.route) }}</span>
                <span class="item-subtitle">{{ getHandlerKindLabel(getTerminalHandler(row.route)?.handler as string) }}</span>
              </div>
            </div>

            <div class="card-prop-row-start">
              <span class="prop-label-start">服务</span>
              <code class="text-xs font-mono text-slate-700 break-all">{{ row.route.serverName }}</code>
            </div>
            <div class="card-prop-row-start">
              <span class="prop-label-start">Path</span>
              <code class="text-xs font-mono text-slate-700 break-all">{{ getRoutePaths(row.route) }}</code>
            </div>
            <div class="card-prop-row">
              <span class="text-xs text-slate-400 flex-shrink-0">Method</span>
              <span class="text-xs text-slate-500">{{ getRouteMethods(row.route) }}</span>
            </div>
            <div class="card-prop-row-start">
              <span class="prop-label-start">后端</span>
              <code :class="getHandlerTagClass(row.route)" class="code-chip">{{ getHandlerSummary(row.route) }}</code>
            </div>

            <div class="card-actions">
              <button v-if="portal.hasPerm('PUT /api/caddy/route/:index')" class="btn-icon btn-icon-blue" title="编辑" @click="openEditModal(row.route)">
                <i class="fas fa-pen text-xs"></i><span class="text-xs ml-1">编辑</span>
              </button>
              <button v-if="portal.hasPerm('DELETE /api/caddy/route/:index')" class="btn-icon btn-icon-red" title="删除" @click="deleteRoute(row.route)">
                <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
              </button>
            </div>
          </div>
        </template>
      </div>
    </template>
  </div>

  <RouteEditModal ref="editModalRef" :servers="routeServerOptions" @success="loadRoutes" />
</template>
