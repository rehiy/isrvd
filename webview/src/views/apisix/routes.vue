<script lang="ts">
import { Component, Inject, Ref, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { ApisixRoute } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'
import RouteEditModal from '@/views/apisix/widget/route-edit-modal.vue'

@Component({
    components: { RouteEditModal }
})
class Routes extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── Refs ───
    @Ref readonly editModalRef!: InstanceType<typeof RouteEditModal>

    // ─── 数据属性 ───
    routes: ApisixRoute[] = []
    loading = false
    searchText = ''

    // ─── 计算属性 ───
    get filteredRoutes() {
        if (!this.searchText) return this.routes
        const s = this.searchText.toLowerCase()
        return this.routes.filter((r: ApisixRoute) =>
            (r.name || '').toLowerCase().includes(s) ||
            (r.id || '').toLowerCase().includes(s) ||
            (r.uri || '').toLowerCase().includes(s) ||
            (r.uris || []).some((u: string) => u.toLowerCase().includes(s)) ||
            (r.desc || '').toLowerCase().includes(s)
        )
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
            this.routes = this.sortRoutes((await api.apisixListRoutes()).payload || [])
        } catch {
            this.actions.showNotification('error', '加载路由列表失败')
        }
        this.loading = false
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

    toggleStatus(route: ApisixRoute) {
        const id = route.id
        if (!id) return
        const ns = route.status === 1 ? 0 : 1
        const label = ns === 1 ? '启用' : '禁用'
        this.actions.showConfirm({
            title: `${label}路由`,
            message: `确定要${label}路由 <strong class="text-slate-900">${route.name}</strong> 吗？`,
            icon: ns === 1 ? 'fa-toggle-on' : 'fa-toggle-off',
            iconColor: ns === 1 ? 'emerald' : 'amber',
            confirmText: `确认${label}`,
            onConfirm: async () => {
                await api.apisixPatchRouteStatus(id, ns)
                this.actions.showNotification('success', `路由已${label}`)
                this.loadRoutes()
            }
        })
    }

    deleteRoute(route: ApisixRoute) {
        const id = route.id
        if (!id) return
        this.actions.showConfirm({
            title: '删除路由',
            message: `确定要删除路由 <strong class="text-slate-900">${route.name || id}</strong> 吗？此操作不可恢复。`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                await api.apisixDeleteRoute(id)
                this.actions.showNotification('success', '删除成功')
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
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-6 py-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-indigo-500 flex items-center justify-center"><i class="fas fa-route text-white"></i></div>
            <div><h1 class="text-lg font-semibold text-slate-800">路由管理</h1><p class="text-xs text-slate-500">管理 APISIX 路由（共 {{ routes.length }} 条）</p></div>
          </div>
          <div class="flex items-center gap-2">
            <div class="relative hidden md:block">
              <input v-model="searchText" type="text" placeholder="搜索路由..." class="pl-8 pr-3 py-1.5 text-xs border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent w-48" />
              <i class="fas fa-search absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400 text-xs"></i>
            </div>
            <button @click="loadRoutes()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors"><i class="fas fa-rotate"></i>刷新</button>
            <button @click="openCreateModal()" class="px-3 py-1.5 rounded-lg bg-indigo-500 hover:bg-indigo-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors"><i class="fas fa-plus"></i>创建</button>
          </div>
        </div>
      </div>
      <!-- 移动端搜索栏 -->
      <div class="md:hidden px-4 py-2 border-b border-slate-100">
        <div class="relative">
          <input v-model="searchText" type="text" placeholder="搜索路由..." class="w-full pl-8 pr-3 py-1.5 text-xs border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent" />
          <i class="fas fa-search absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400 text-xs"></i>
        </div>
      </div>
      <div v-if="loading" class="flex flex-col items-center justify-center py-20"><div class="w-12 h-12 spinner mb-3"></div><p class="text-slate-500">加载中...</p></div>
      <div v-else-if="filteredRoutes.length === 0" class="flex flex-col items-center justify-center py-20">
        <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4"><i class="fas fa-route text-4xl text-slate-300"></i></div>
        <p class="text-slate-600 font-medium mb-1">暂无路由</p>
        <p class="text-sm text-slate-400">点击「创建」添加新路由</p>
      </div>
      <div v-else class="space-y-3">
        <!-- 桌面端表格视图 -->
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead><tr class="bg-slate-50 border-b border-slate-200">
              <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">名称</th>
              <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">URI</th>
              <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">Host</th>
              <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">状态</th>
              <th class="w-32 px-4 py-3 text-right text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
            </tr></thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="route in filteredRoutes" :key="route.id" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3">
                  <div class="font-medium text-sm text-slate-800">{{ route.name || route.id }}</div>
                  <div v-if="route.desc" class="text-xs text-slate-400 mt-0.5 truncate max-w-xs">{{ route.desc }}</div>
                </td>
                <td class="px-4 py-3"><code class="text-xs bg-slate-100 px-2 py-1 rounded text-slate-700">{{ getRouteUri(route) }}</code></td>
                <td class="px-4 py-3"><span class="text-sm text-slate-600">{{ getRouteHost(route) }}</span></td>
                <td class="px-4 py-3">
                  <button @click="toggleStatus(route)" :class="['inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium cursor-pointer transition-colors', route.status === 1 ? 'bg-emerald-50 text-emerald-700 hover:bg-emerald-100' : 'bg-slate-100 text-slate-500 hover:bg-slate-200']">
                    <i :class="route.status === 1 ? 'fas fa-circle text-emerald-500' : 'fas fa-circle text-slate-400'" class="text-[6px]"></i>
                    {{ route.status === 1 ? '启用' : '禁用' }}
                  </button>
                </td>
                <td class="px-4 py-3">
                  <div class="flex justify-end items-center gap-1">
                    <button @click="openEditModal(route)" class="btn-icon text-indigo-600 hover:bg-indigo-50" title="编辑"><i class="fas fa-pen-to-square text-xs"></i></button>
                    <button @click="deleteRoute(route)" class="btn-icon text-red-600 hover:bg-red-50" title="删除"><i class="fas fa-trash text-xs"></i></button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 移动端卡片视图 -->
        <div class="md:hidden space-y-3">
          <div 
            v-for="route in filteredRoutes" 
            :key="route.id"
            class="rounded-xl border border-slate-200 bg-white p-4 transition-all hover:shadow-sm"
          >
            <!-- 顶部：路由信息和状态 -->
            <div class="flex items-center justify-between mb-3">
              <div class="min-w-0">
                <div class="font-medium text-sm text-slate-800">{{ route.name || route.id }}</div>
                <div v-if="route.desc" class="text-xs text-slate-400 mt-0.5 truncate">{{ route.desc }}</div>
              </div>
              <button @click="toggleStatus(route)" :class="['inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium cursor-pointer transition-colors', route.status === 1 ? 'bg-emerald-50 text-emerald-700 hover:bg-emerald-100' : 'bg-slate-100 text-slate-500 hover:bg-slate-200']">
                <i :class="route.status === 1 ? 'fas fa-circle text-emerald-500' : 'fas fa-circle text-slate-400'" class="text-[6px]"></i>
                {{ route.status === 1 ? '启用' : '禁用' }}
              </button>
            </div>
            
            <!-- 中间：URI和Host信息 -->
            <div class="mb-3">
              <p class="text-xs text-slate-500 mb-1">URI</p>
              <code class="text-xs bg-slate-100 px-2 py-1 rounded text-slate-700 break-all">{{ getRouteUri(route) }}</code>
            </div>
            
            <div class="mb-3">
              <p class="text-xs text-slate-500 mb-1">Host</p>
              <span class="text-sm text-slate-600 break-all">{{ getRouteHost(route) }}</span>
            </div>
            
            <!-- 底部：操作按钮 -->
            <div class="flex flex-wrap gap-1 pt-2 border-t border-slate-100">
              <button @click="openEditModal(route)" class="btn-icon text-indigo-600 hover:bg-indigo-50" title="编辑">
                <i class="fas fa-pen-to-square text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">编辑</span>
              </button>
              <button @click="deleteRoute(route)" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
                <i class="fas fa-trash text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">删除</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <RouteEditModal ref="editModalRef" @success="loadRoutes" />
  </div>
</template>
