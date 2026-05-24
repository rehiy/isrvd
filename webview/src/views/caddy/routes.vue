<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { CaddyRoute } from '@/service/types'

import PageSearch from '@/component/page-search.vue'

import RouteEditModal from './widget/route-edit-modal.vue'

@Component({
    components: { PageSearch, RouteEditModal }
})
class CaddyRoutes extends Vue {
    portal = usePortal()

    @Ref readonly editModalRef!: InstanceType<typeof RouteEditModal>

    routes: CaddyRoute[] = []
    loading = false
    searchText = ''

    get filteredRoutes() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.routes
        return this.routes.filter((r: CaddyRoute) => {
            const m = r.match
            const h = r.handler
            return (
                (m?.hosts || []).some((s: string) => s.toLowerCase().includes(keyword)) ||
                (m?.paths || []).some((s: string) => s.toLowerCase().includes(keyword)) ||
                (h?.upstreams || []).some((s: string) => s.toLowerCase().includes(keyword)) ||
                (h?.root || '').toLowerCase().includes(keyword) ||
                (h?.kind || '').toLowerCase().includes(keyword)
            )
        })
    }

    async loadRoutes() {
        this.loading = true
        try {
            this.routes = (await api.caddyRouteList()).payload || []
        } catch {
            this.portal.showNotification('error', '加载路由列表失败')
        } finally {
            this.loading = false
        }
    }

    openCreateModal() {
        this.editModalRef?.show(null)
    }

    openEditModal(route: CaddyRoute) {
        this.editModalRef?.show(route)
    }

    getRouteHosts(r: CaddyRoute) {
        const hosts = r.match?.hosts || []
        return hosts.length ? hosts.join(', ') : '*'
    }

    getRoutePaths(r: CaddyRoute) {
        const paths = r.match?.paths || []
        return paths.length ? paths.join(', ') : '/*'
    }

    getRouteMethods(r: CaddyRoute) {
        const methods = r.match?.methods || []
        return methods.length ? methods.join(' ') : 'ANY'
    }

    getHandlerKindLabel(kind?: string) {
        const map: Record<string, string> = {
            reverse_proxy: '反向代理',
            file_server: '文件服务',
            static_response: '静态响应',
            raw: '原始 JSON'
        }
        return map[kind || ''] || kind || '-'
    }

    getHandlerSummary(r: CaddyRoute) {
        const h = r.handler
        if (!h) return '-'
        switch (h.kind) {
            case 'reverse_proxy': return (h.upstreams || []).join(', ')
            case 'file_server': return h.root || '-'
            case 'static_response': return `${h.statusCode || 200} ${h.body ? '+' : ''}`
            case 'raw': return '(自定义)'
        }
        return '-'
    }

    getHandlerTagClass(r: CaddyRoute) {
        const kind = r.handler?.kind
        if (kind === 'reverse_proxy') return 'bg-indigo-50 text-indigo-700'
        if (kind === 'file_server') return 'bg-emerald-50 text-emerald-700'
        if (kind === 'static_response') return 'bg-amber-50 text-amber-700'
        return 'bg-slate-100 text-slate-500'
    }

    deleteRoute(route: CaddyRoute) {
        this.portal.showConfirm({
            title: '删除路由',
            message: `确定要删除路由 <strong class="text-slate-900">#${route.index}</strong> 吗？此操作不可恢复。`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                try {
                    await api.caddyRouteDelete(route.index)
                    this.portal.showNotification('success', '删除成功')
                    this.loadRoutes()
                } catch (e: unknown) {
                    this.portal.showNotification('error', (e instanceof Error ? e.message : '') || '删除失败')
                }
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
  <div>
    <div class="card mb-4">
      <div class="card-toolbar">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="page-icon bg-indigo-500"><i class="fas fa-route text-white"></i></div>
            <div class="min-w-0"><h1 class="text-lg font-semibold text-slate-800 truncate">Caddy 路由</h1><p class="text-xs text-slate-500 truncate">配置请求匹配规则与处理器，支持多种转发方式</p></div>
          </div>
          <div class="flex items-center gap-2 flex-shrink-0">
            <PageSearch v-model="searchText" search-key="caddy-routes" placeholder="搜索 host、path、上游..." width-class="w-64" focus-color="indigo" type-to-search />
            <button class="btn btn-secondary" @click="loadRoutes()"><i class="fas fa-rotate"></i>刷新</button>
            <button v-if="portal.hasPerm('POST /api/caddy/route')" class="btn btn-indigo" @click="openCreateModal()"><i class="fas fa-plus"></i>新建路由</button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="page-icon bg-indigo-500"><i class="fas fa-route text-white"></i></div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">Caddy 路由</h1>
              <p class="text-xs text-slate-500 truncate">配置匹配规则与处理器</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadRoutes()">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button v-if="portal.hasPerm('POST /api/caddy/route')" class="btn btn-indigo w-9 h-9 !px-0" title="新建路由" @click="openCreateModal()">
              <i class="fas fa-plus text-sm"></i>
            </button>
          </div>
        </div>
      </div>
      <!-- 移动端搜索栏 -->
      <div class="mobile-search">
        <PageSearch v-model="searchText" search-key="caddy-routes" placeholder="搜索 host、path、上游..." width-class="w-full" focus-color="indigo" />
      </div>
      <div v-if="loading" class="empty-state"><div class="w-12 h-12 spinner mb-3"></div><p class="text-slate-500">加载中...</p></div>
      <div v-else-if="filteredRoutes.length === 0" class="empty-state">
        <div class="empty-state-icon"><i class="fas fa-route text-4xl text-slate-300"></i></div>
        <p class="text-slate-600 font-medium mb-1">{{ routes.length === 0 ? '暂无路由' : '未找到匹配路由' }}</p>
        <p class="text-sm text-slate-400">{{ routes.length === 0 ? '点击「新建路由」开始创建' : '尝试更换关键词或清空搜索条件' }}</p>
      </div>
      <div v-else class="space-y-3">
        <!-- 桌面端表格 -->
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="th">序号</th>
                <th class="th">Host</th>
                <th class="th">Path</th>
                <th class="th">Method</th>
                <th class="th">类型</th>
                <th class="th">处理器</th>
                <th class="w-32 th-right">操作</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="route in filteredRoutes" :key="route.index" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3">
                  <div class="flex items-center gap-2 min-w-0">
                    <div class="row-icon bg-indigo-400">
                      <i class="fas fa-route text-white text-sm"></i>
                    </div>
                    <span class="font-medium text-slate-800">{{ route.index }}</span>
                  </div>
                </td>
                <td class="px-4 py-3"><span :class="getRouteHosts(route) === '*' ? 'text-slate-400' : 'text-teal-600 font-medium'" class="text-sm break-all">{{ getRouteHosts(route) }}</span></td>
                <td class="px-4 py-3"><code class="text-xs font-mono text-slate-700 break-all">{{ getRoutePaths(route) }}</code></td>
                <td class="px-4 py-3"><span class="text-xs text-slate-600">{{ getRouteMethods(route) }}</span></td>
                <td class="px-4 py-3"><span :class="getHandlerTagClass(route)" class="inline-block text-xs px-2 py-0.5 rounded-lg">{{ getHandlerKindLabel(route.handler?.kind) }}</span></td>
                <td class="px-4 py-3"><code class="text-xs font-mono text-slate-700 break-all">{{ getHandlerSummary(route) }}</code></td>
                <td class="px-4 py-3">
                  <div class="flex justify-end items-center gap-1">
                    <button v-if="portal.hasPerm('PUT /api/caddy/route/:index')" class="btn-icon btn-icon-blue" title="编辑" @click="openEditModal(route)"><i class="fas fa-pen text-xs"></i></button>
                    <button v-if="portal.hasPerm('DELETE /api/caddy/route/:index')" class="btn-icon btn-icon-red" title="删除" @click="deleteRoute(route)"><i class="fas fa-trash text-xs"></i></button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 移动端卡片 -->
        <div class="md:hidden space-y-3 p-4">
          <div v-for="route in filteredRoutes" :key="route.index" class="card-interactive">
            <div class="card-info-row">
              <div class="list-icon bg-indigo-400">
                <i class="fas fa-route text-white text-base"></i>
              </div>
              <div class="min-w-0">
                <span class="font-medium text-slate-800 text-sm truncate block">路由 #{{ route.index }}</span>
                <span class="text-xs text-slate-400 truncate block mt-0.5">{{ getHandlerKindLabel(route.handler?.kind) }}</span>
              </div>
            </div>

            <div class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">Host</span>
              <span :class="getRouteHosts(route) === '*' ? 'text-slate-400' : 'text-teal-600 font-medium'" class="text-xs break-all">{{ getRouteHosts(route) }}</span>
            </div>
            <div class="flex items-start gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">Path</span>
              <code class="text-xs font-mono text-slate-700 break-all">{{ getRoutePaths(route) }}</code>
            </div>
            <div class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">Method</span>
              <span class="text-xs text-slate-500">{{ getRouteMethods(route) }}</span>
            </div>
            <div class="flex items-start gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">后端</span>
              <code :class="getHandlerTagClass(route)" class="inline-block text-xs px-2 py-0.5 rounded-lg font-mono break-all">{{ getHandlerSummary(route) }}</code>
            </div>

            <div class="card-actions">
              <button v-if="portal.hasPerm('PUT /api/caddy/route/:index')" class="btn-icon btn-icon-blue" title="编辑" @click="openEditModal(route)">
                <i class="fas fa-pen text-xs"></i><span class="text-xs ml-1">编辑</span>
              </button>
              <button v-if="portal.hasPerm('DELETE /api/caddy/route/:index')" class="btn-icon btn-icon-red" title="删除" @click="deleteRoute(route)">
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
