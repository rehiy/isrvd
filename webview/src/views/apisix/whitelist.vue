<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { ApisixRoute } from '@/service/types'

import PageSearch from '@/component/page-search.vue'

import { usePortal } from '@/stores'

@Component({
    components: { PageSearch }
})
class Whitelist extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    whitelist: ApisixRoute[] = []
    loading = false
    searchText = ''

    // ─── 计算属性 ───
    get filteredWhitelist() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.whitelist
        return this.whitelist.filter((r: ApisixRoute) =>
            (r.name || '').toLowerCase().includes(keyword) ||
            (r.id || '').toLowerCase().includes(keyword) ||
            (r.consumers || []).some((c: string) => c.toLowerCase().includes(keyword))
        )
    }

    // ─── 方法 ───
    async loadWhitelist() {
        this.loading = true
        try {
            this.whitelist = (await api.apisixWhitelist()).payload || []
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

    revokeConsumer(route: ApisixRoute, consumer: string) {
        const routeId = route.id
        if (!routeId) return
        this.portal.showConfirm({
            title: '撤销白名单',
            message: `确定要将用户 <strong class="text-slate-900">${consumer}</strong> 从路由 <strong class="text-slate-900">${route.name || routeId}</strong> 的白名单中移除吗？`,
            icon: 'fa-user-minus',
            iconColor: 'red',
            confirmText: '确认撤销',
            danger: true,
            onConfirm: async () => {
                await api.apisixWhitelistRevoke({ routeId, consumer })
                this.portal.showNotification('success', '撤销成功')
                this.loadWhitelist()
            }
        })
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadWhitelist()
    }
}

export default toNative(Whitelist)
</script>

<template>
  <div>
    <div class="card mb-4">
      <!-- Toolbar -->
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-amber-500 flex items-center justify-center">
              <i class="fas fa-shield-halved text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">白名单管理</h1>
              <p class="text-xs text-slate-500">配置路由级别的 Consumer 访问白名单</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <PageSearch v-model="searchText" search-key="apisix-whitelist" placeholder="搜索路由或用户..." width-class="w-48" focus-color="amber" type-to-search />
            <button class="btn btn-sm btn-secondary" @click="loadWhitelist()">
              <i class="fas fa-rotate"></i>刷新
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="w-9 h-9 rounded-lg bg-amber-500 flex items-center justify-center flex-shrink-0">
              <i class="fas fa-shield-halved text-white"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">白名单管理</h1>
              <p class="text-xs text-slate-500 truncate">路由级 Consumer 白名单</p>
            </div>
          </div>
          <button class="btn btn-sm btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadWhitelist()">
            <i class="fas fa-rotate text-sm"></i>
          </button>
        </div>
      </div>

      <!-- 移动端搜索 -->
      <div class="md:hidden px-4 py-2 border-b border-slate-100">
        <PageSearch v-model="searchText" search-key="apisix-whitelist" placeholder="搜索路由或用户..." width-class="w-full" focus-color="amber" />
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <!-- Empty -->
      <div v-else-if="filteredWhitelist.length === 0" class="flex flex-col items-center justify-center py-20">
        <div class="w-16 h-16 rounded-lg bg-slate-100 flex items-center justify-center mb-4">
          <i class="fas fa-shield-halved text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">{{ whitelist.length === 0 ? '暂无白名单数据' : '未找到匹配白名单' }}</p>
        <p class="text-sm text-slate-400">{{ whitelist.length === 0 ? '配置路由的 Consumer 白名单后将在此显示' : '尝试更换关键词或清空搜索条件' }}</p>
      </div>

      <!-- 列表 -->
      <div v-else class="space-y-3">
        <!-- 桌面端表格 -->
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">路由</th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">描述</th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">白名单用户</th>
                <th class="w-24 px-4 py-3 text-right text-xs font-semibold text-slate-600 uppercase tracking-wider">用户数</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="route in filteredWhitelist" :key="route.id" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3 max-w-[280px]">
                  <div class="flex items-center gap-2 min-w-0">
                    <div class="w-8 h-8 rounded-lg bg-amber-400 flex items-center justify-center flex-shrink-0">
                      <i class="fas fa-shield-halved text-white text-sm"></i>
                    </div>
                    <div class="min-w-0">
                      <span class="font-medium text-slate-800 truncate block">{{ getRouteUri(route) }}</span>
                      <span class="text-xs text-slate-400 truncate block mt-0.5">{{ getRouteHost(route) }}</span>
                    </div>
                  </div>
                </td>
                <td class="px-4 py-3 text-xs text-slate-500">{{ route.name || route.id }}</td>
                <td class="px-4 py-3">
                  <div class="flex flex-wrap gap-1.5">
                    <span v-for="consumer in (route.consumers || [])" :key="consumer" class="inline-flex items-center gap-1.5 px-2 py-0.5 bg-amber-50 text-amber-800 rounded-lg text-xs group">
                      <i class="fas fa-user text-amber-500 text-[10px]"></i>
                      <span class="break-all">{{ consumer }}</span>
                      <button v-if="portal.hasPerm('POST /api/apisix/whitelist/revoke')" class="opacity-0 group-hover:opacity-100 hover:text-red-500 transition-all" title="撤销" @click="revokeConsumer(route, consumer)"><i class="fas fa-xmark text-[10px]"></i></button>
                    </span>
                  </div>
                </td>
                <td class="px-4 py-3 text-right text-xs text-slate-500">{{ (route.consumers || []).length }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 移动端卡片 -->
        <div class="md:hidden space-y-3 p-4">
          <div v-for="route in filteredWhitelist" :key="route.id" class="rounded-xl border border-slate-200 bg-white p-4 transition-all hover:shadow-sm">
            <!-- 顶部：路由标识 -->
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-3 min-w-0 flex-1">
                <div class="w-10 h-10 rounded-lg bg-amber-400 flex items-center justify-center flex-shrink-0">
                  <i class="fas fa-shield-halved text-white text-base"></i>
                </div>
                <div class="min-w-0">
                  <div class="font-medium text-slate-800 text-sm truncate">{{ getRouteUri(route) }}</div>
                  <div class="text-xs text-slate-400 truncate mt-0.5">{{ getRouteHost(route) }}</div>
                </div>
              </div>
            </div>

            <!-- 主机 -->
            <div class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">描述</span>
              <span class="text-xs text-slate-500">{{ route.name || route.id }}</span>
            </div>

            <!-- 用户列表 -->
            <div class="flex items-start gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">用户</span>
              <div class="flex flex-wrap gap-1.5">
                <span v-for="consumer in (route.consumers || [])" :key="consumer" class="inline-flex items-center gap-1.5 px-2.5 py-1 bg-amber-50 text-amber-800 rounded-lg text-xs">
                  <i class="fas fa-user text-amber-500 text-[10px]"></i>
                  <span class="break-all">{{ consumer }}</span>
                  <button v-if="portal.hasPerm('POST /api/apisix/whitelist/revoke')" class="hover:text-red-500 transition-colors" title="撤销" @click="revokeConsumer(route, consumer)"><i class="fas fa-xmark text-[10px]"></i></button>
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
