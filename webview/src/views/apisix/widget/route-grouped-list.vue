<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import type { ApisixRoute } from '@/service/types'

import { formatRouteUpstreamSummary, formatRouteUpstreamNodes, normalizeUpstreamNodes } from '@/helper/apisix'

interface ApisixRouteGroupEntry {
    key: string
    route: ApisixRoute
}

interface ApisixRouteGroup {
    key: string
    entries: ApisixRouteGroupEntry[]
    enabled: number
    preview: string
}

@Component({
    emits: ['toggle-status', 'edit', 'delete']
})
class RouteGroupedList extends Vue {
    portal = usePortal()

    @Prop({ type: Array, default: () => [] }) readonly routes!: ApisixRoute[]
    @Prop({ type: String, default: '' }) readonly searchText!: string

    expandedGroupKeys: string[] = []

    get keyword() {
        return this.searchText.trim().toLowerCase()
    }

    get autoExpand() {
        return !!this.keyword
    }

    get routeHostGroups() {
        return this.buildRouteGroups()
    }

    buildRouteGroups(): ApisixRouteGroup[] {
        const groups = new Map<string, ApisixRouteGroup>()

        for (const route of this.routes) {
            const routeMatched = !this.keyword || this.routeMatchesKeyword(route)
            for (const host of this.getRouteHosts(route)) {
                if (this.keyword && !routeMatched && !host.toLowerCase().includes(this.keyword)) continue
                const group = groups.get(host) || { key: host, entries: [], enabled: 0, preview: '' }
                group.entries.push({ key: `${host}::${route.id || route.name || this.getRouteUri(route)}`, route })
                if (route.status === 1) group.enabled++
                groups.set(host, group)
            }
        }

        return Array.from(groups.values())
            .map(group => ({ ...group, preview: this.getRouteGroupPreview(group.entries) }))
            .sort((a, b) => this.compareHosts(a.key, b.key))
    }

    getRouteHosts(r: ApisixRoute) {
        const hosts = (r.hosts?.length ? r.hosts : [r.host || ''])
            .map((host: string) => host.trim())
            .filter(Boolean)
        return hosts.length > 0 ? Array.from(new Set(hosts)) : ['*']
    }

    routeMatchesKeyword(r: ApisixRoute) {
        const text = [
            r.name,
            r.id,
            r.uri,
            ...(r.uris || []),
            r.desc,
            formatRouteUpstreamSummary(r)
        ].join('\n').toLowerCase()
        return text.includes(this.keyword)
    }

    compareHosts(a: string, b: string) {
        if (a === b) return 0
        if (a === '*') return 1
        if (b === '*') return -1
        return a.localeCompare(b)
    }

    getRouteGroupPreview(entries: ApisixRouteGroupEntry[]) {
        return entries
            .map(entry => entry.route.name || entry.route.id || this.getRouteUri(entry.route))
            .join('、')
    }

    isGroupExpanded(group: ApisixRouteGroup) {
        return this.autoExpand || this.expandedGroupKeys.includes(group.key)
    }

    toggleGroup(group: ApisixRouteGroup) {
        if (this.expandedGroupKeys.includes(group.key)) {
            this.expandedGroupKeys = this.expandedGroupKeys.filter(key => key !== group.key)
            return
        }
        this.expandedGroupKeys = [...this.expandedGroupKeys, group.key]
    }

    getRouteUri(r: ApisixRoute) {
        return r.uris?.length ? r.uris.join(', ') : (r.uri || '-')
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
        this.$emit('toggle-status', route)
    }

    editRoute(route: ApisixRoute) {
        this.$emit('edit', route)
    }

    deleteRoute(route: ApisixRoute) {
        this.$emit('delete', route)
    }
}

export default toNative(RouteGroupedList)
</script>

<template>
  <div class="space-y-3">
    <div v-if="routeHostGroups.length === 0" class="empty-state">
      <div class="empty-state-icon"><i class="fas fa-layer-group text-4xl text-slate-300"></i></div>
      <p class="text-slate-600 font-medium mb-1">{{ routes.length === 0 ? '暂无路由' : '未找到匹配 Host' }}</p>
      <p class="text-sm text-slate-400">{{ routes.length === 0 ? '点击「新建路由」开始创建' : '尝试更换关键词或清空搜索条件' }}</p>
    </div>

    <!-- 桌面端 Host 分组表格视图 -->
    <div v-else class="hidden md:block overflow-x-auto">
      <table class="w-full border-collapse">
        <thead>
          <tr class="bg-slate-50 border-b border-slate-200">
            <th class="th">路由</th>
            <th class="th">URI</th>
            <th class="th">上游</th>
            <th class="w-40 th-right">操作</th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-slate-100">
          <template v-for="group in routeHostGroups" :key="group.key">
            <tr class="bg-slate-50/80 hover:bg-slate-100 transition-colors">
              <td colspan="5" class="px-4 py-3">
                <button type="button" class="w-full flex items-center gap-3 text-left" :aria-expanded="isGroupExpanded(group)" @click="toggleGroup(group)">
                  <i class="fas fa-chevron-right text-slate-400 text-xs transition-transform flex-shrink-0" :class="{ 'rotate-90': isGroupExpanded(group) }"></i>
                  <div class="row-icon bg-teal-400">
                    <i class="fas fa-globe text-white text-sm"></i>
                  </div>
                  <div class="min-w-0 w-[260px] flex-shrink-0">
                    <span :class="group.key === '*' ? 'text-slate-400' : 'text-teal-600 font-medium'" class="text-sm truncate block">{{ group.key }}</span>
                    <span class="text-xs text-slate-400 truncate block mt-0.5">{{ group.entries.length }} 条路由 / {{ group.enabled }} 启用</span>
                  </div>
                  <span class="min-w-0 flex-1 text-xs text-slate-500 truncate">{{ group.preview }}</span>
                  <span class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-medium flex-shrink-0 bg-indigo-50 text-indigo-700">{{ group.entries.length }} 条</span>
                  <span class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-medium flex-shrink-0 bg-emerald-50 text-emerald-700">{{ group.enabled }} 启用</span>
                </button>
              </td>
            </tr>
            <tr v-for="entry in group.entries" v-show="isGroupExpanded(group)" :key="entry.key" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3 max-w-[280px]">
                <div class="flex items-center gap-2 min-w-0">
                  <div class="row-icon bg-indigo-400">
                    <i class="fas fa-route text-white text-sm"></i>
                  </div>
                  <div class="min-w-0">
                    <span class="font-medium text-slate-800 truncate block">{{ entry.route.name || entry.route.id }}</span>
                    <span v-if="entry.route.desc" class="text-xs text-slate-400 truncate block mt-0.5">{{ entry.route.desc }}</span>
                  </div>
                </div>
              </td>
              <td class="px-4 py-3"><code class="text-xs font-mono text-slate-700 break-all">{{ getRouteUri(entry.route) }}</code></td>
              <td class="px-4 py-3"><span :class="getRouteUpstreamTagClass(entry.route)" class="inline-block text-xs px-2 py-0.5 rounded-lg font-mono break-all">{{ getRouteUpstreamNodes(entry.route) }}</span></td>
              <td class="px-4 py-3">
                <div class="flex justify-end items-center gap-1">
                  <button v-if="portal.hasPerm('PATCH /api/apisix/route/:id/status')" :class="['btn-icon', entry.route.status === 1 ? 'btn-icon-amber' : 'btn-icon-emerald']" :title="entry.route.status === 1 ? '禁用' : '启用'" @click="toggleStatus(entry.route)">
                    <i :class="entry.route.status === 1 ? 'fas fa-ban' : 'fas fa-play'" class="text-xs"></i>
                  </button>
                  <button v-if="portal.hasPerm('PUT /api/apisix/route/:id')" class="btn-icon btn-icon-indigo" title="编辑" @click="editRoute(entry.route)"><i class="fas fa-pen text-xs"></i></button>
                  <button v-if="portal.hasPerm('DELETE /api/apisix/route/:id')" class="btn-icon btn-icon-red" title="删除" @click="deleteRoute(entry.route)"><i class="fas fa-trash text-xs"></i></button>
                </div>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>

    <!-- 移动端 Host 分组列表视图 -->
    <div v-if="routeHostGroups.length > 0" class="md:hidden space-y-4 p-4">
      <template v-for="group in routeHostGroups" :key="group.key">
        <button type="button" class="w-full flex items-center justify-between gap-3 rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-left" :aria-expanded="isGroupExpanded(group)" @click="toggleGroup(group)">
          <div class="flex items-center gap-3 min-w-0">
            <i class="fas fa-chevron-right text-slate-400 text-xs transition-transform flex-shrink-0" :class="{ 'rotate-90': isGroupExpanded(group) }"></i>
            <div class="list-icon bg-teal-400">
              <i class="fas fa-globe text-white text-base"></i>
            </div>
            <div class="min-w-0">
              <span :class="group.key === '*' ? 'text-slate-400' : 'text-teal-600 font-medium'" class="text-sm truncate block">{{ group.key }}</span>
              <span class="text-xs text-slate-400 truncate block mt-0.5">{{ group.entries.length }} 条路由 / {{ group.enabled }} 启用</span>
            </div>
          </div>
          <span class="text-xs text-slate-400 flex-shrink-0">{{ isGroupExpanded(group) ? '收起' : '展开' }}</span>
        </button>

        <div v-show="isGroupExpanded(group)" class="space-y-3 pl-3 border-l border-slate-200">
          <div v-for="entry in group.entries" :key="entry.key" class="card-interactive">
            <div class="card-info-row">
              <div class="list-icon bg-indigo-400">
                <i class="fas fa-route text-white text-base"></i>
              </div>
              <div class="min-w-0">
                <div class="font-medium text-sm text-slate-800 truncate">{{ entry.route.name || entry.route.id }}</div>
                <div v-if="entry.route.desc" class="text-xs text-slate-400 mt-0.5 truncate">{{ entry.route.desc }}</div>
              </div>
            </div>

            <div class="flex items-start gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">URI</span>
              <code class="text-xs font-mono text-slate-700 break-all">{{ getRouteUri(entry.route) }}</code>
            </div>
            <div class="flex items-start gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">上游</span>
              <span :class="getRouteUpstreamTagClass(entry.route)" class="inline-block text-xs px-2 py-0.5 rounded-lg font-mono break-all">{{ getRouteUpstreamNodes(entry.route) }}</span>
            </div>

            <div class="card-actions">
              <button v-if="portal.hasPerm('PATCH /api/apisix/route/:id/status')" :class="['btn-icon', entry.route.status === 1 ? 'btn-icon-amber' : 'btn-icon-emerald']" :title="entry.route.status === 1 ? '禁用' : '启用'" @click="toggleStatus(entry.route)">
                <i :class="entry.route.status === 1 ? 'fas fa-ban' : 'fas fa-play'" class="text-xs"></i><span class="text-xs ml-1">{{ entry.route.status === 1 ? '禁用' : '启用' }}</span>
              </button>
              <button v-if="portal.hasPerm('PUT /api/apisix/route/:id')" class="btn-icon btn-icon-indigo" title="编辑" @click="editRoute(entry.route)">
                <i class="fas fa-pen text-xs"></i><span class="text-xs ml-1">编辑</span>
              </button>
              <button v-if="portal.hasPerm('DELETE /api/apisix/route/:id')" class="btn-icon btn-icon-red" title="删除" @click="deleteRoute(entry.route)">
                <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
              </button>
            </div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>
