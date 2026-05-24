<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import type { ApisixRoute } from '@/service/types'

import { formatRouteUpstreamType, formatRouteUpstreamNodes, normalizeUpstreamNodes } from '@/helper/apisix'

import { usePortal } from '@/stores'

import type { ApisixRouteGroup } from './route-grouped-list-types'

@Component({
    emits: ['toggle-group', 'toggle-status', 'edit', 'delete']
})
class RouteGroupedList extends Vue {
    portal = usePortal()

    @Prop({ type: Array, default: () => [] }) readonly groups!: ApisixRouteGroup[]
    @Prop({ type: Array, default: () => [] }) readonly expandedGroupKeys!: string[]
    @Prop({ type: Boolean, default: false }) readonly autoExpand!: boolean
    @Prop({ type: String, default: '分组' }) readonly groupColumnLabel!: string
    @Prop({ type: String, default: 'fa-layer-group' }) readonly groupIcon!: string
    @Prop({ type: String, default: 'bg-slate-400' }) readonly groupIconBgClass!: string
    @Prop({ type: String, default: 'w-[260px]' }) readonly groupWidthClass!: string

    isGroupExpanded(group: ApisixRouteGroup) {
        return this.autoExpand || this.expandedGroupKeys.includes(group.key)
    }

    toggleGroup(group: ApisixRouteGroup) {
        this.$emit('toggle-group', group.key)
    }

    getRouteUri(r: ApisixRoute) {
        return r.uris?.length ? r.uris.join(', ') : (r.uri || '-')
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
  <div>
    <!-- 桌面端分组表格视图 -->
    <div class="hidden md:block overflow-x-auto">
      <table class="w-full border-collapse">
        <thead>
          <tr class="bg-slate-50 border-b border-slate-200">
            <th :class="['th', groupWidthClass]">{{ groupColumnLabel }}</th>
            <th class="th">路由</th>
            <th class="th">URI</th>
            <th class="th">策略</th>
            <th class="th">上游</th>
            <th class="w-40 th-right">操作</th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-slate-100">
          <template v-for="group in groups" :key="group.key">
            <tr class="bg-slate-50 hover:bg-slate-100 transition-colors">
              <td :class="['px-4 py-3 max-w-[280px]', groupWidthClass]">
                <button type="button" class="w-full flex items-center gap-2 min-w-0 text-left" :aria-expanded="isGroupExpanded(group)" @click="toggleGroup(group)">
                  <i class="fas fa-chevron-right text-slate-400 text-xs transition-transform flex-shrink-0" :class="{ 'rotate-90': isGroupExpanded(group) }"></i>
                  <div :class="['row-icon', groupIconBgClass]">
                    <i :class="['fas', groupIcon, 'text-white', 'text-sm']"></i>
                  </div>
                  <div class="min-w-0">
                    <span :class="group.labelClass || 'text-slate-800 font-medium'" class="text-sm truncate block">{{ group.label }}</span>
                    <span class="text-xs text-slate-400 truncate block mt-0.5">{{ group.summary }}</span>
                  </div>
                </button>
              </td>
              <td colspan="3" class="px-4 py-3">
                <span class="text-xs text-slate-500 truncate block">{{ group.preview }}</span>
              </td>
              <td class="px-4 py-3">
                <div class="flex items-center gap-2">
                  <span v-for="stat in group.stats" :key="stat.key" :class="['inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-medium', stat.className]">{{ stat.label }}</span>
                </div>
              </td>
              <td class="px-4 py-3" aria-hidden="true"></td>
            </tr>
            <tr v-for="entry in group.entries" v-show="isGroupExpanded(group)" :key="entry.key" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3 bg-white border-r border-slate-100"></td>
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
              <td class="px-4 py-3"><span class="text-xs text-slate-600">{{ getRouteUpstreamType(entry.route) || '-' }}</span></td>
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

    <!-- 移动端分组卡片视图 -->
    <div class="md:hidden space-y-4 p-4">
      <div v-for="group in groups" :key="group.key" class="space-y-2">
        <button type="button" class="w-full flex items-center justify-between gap-3 rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-left" :aria-expanded="isGroupExpanded(group)" @click="toggleGroup(group)">
          <div class="flex items-center gap-3 min-w-0">
            <i class="fas fa-chevron-right text-slate-400 text-xs transition-transform flex-shrink-0" :class="{ 'rotate-90': isGroupExpanded(group) }"></i>
            <div :class="['list-icon', groupIconBgClass]">
              <i :class="['fas', groupIcon, 'text-white', 'text-base']"></i>
            </div>
            <div class="min-w-0">
              <span :class="group.labelClass || 'text-slate-800 font-medium'" class="text-sm truncate block">{{ group.label }}</span>
              <span class="text-xs text-slate-400 truncate block mt-0.5">{{ group.summary }}</span>
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
            <div class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">策略</span>
              <span class="text-xs text-slate-500">{{ getRouteUpstreamType(entry.route) || '-' }}</span>
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
      </div>
    </div>
  </div>
</template>
