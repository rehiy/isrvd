<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { ApisixUpstream } from '@/service/types'

import { normalizeUpstreamNodes } from '@/helper/apisix'

import PageSearch from '@/component/page-search.vue'

import { usePortal } from '@/stores'

import UpstreamEditModal from './widget/upstream-edit-modal.vue'

@Component({
    components: { PageSearch, UpstreamEditModal }
})
class Upstreams extends Vue {
    portal = usePortal()

    @Ref readonly editModalRef!: InstanceType<typeof UpstreamEditModal>

    upstreams: ApisixUpstream[] = []
    loading = false
    searchText = ''

    get filteredUpstreams() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.upstreams
        return this.upstreams.filter((upstream: ApisixUpstream) => {
            const nodes = this.getUpstreamNodes(upstream).toLowerCase()
            return (
                (upstream.name || '').toLowerCase().includes(keyword) ||
                (upstream.id || '').toLowerCase().includes(keyword) ||
                (upstream.desc || '').toLowerCase().includes(keyword) ||
                (upstream.type || '').toLowerCase().includes(keyword) ||
                nodes.includes(keyword)
            )
        })
    }

    sortUpstreams(data: ApisixUpstream[]) {
        data.sort((a: ApisixUpstream, b: ApisixUpstream) => (a.name || a.id || '').localeCompare(b.name || b.id || ''))
        return data
    }

    async loadUpstreams() {
        this.loading = true
        try {
            this.upstreams = this.sortUpstreams((await api.apisixUpstreamList()).payload || [])
        } catch {
            this.portal.showNotification('error', '加载上游列表失败')
        } finally {
            this.loading = false
        }
    }

    openCreateModal() {
        this.editModalRef?.show()
    }

    openEditModal(upstream: ApisixUpstream | null) {
        this.editModalRef?.show(upstream)
    }

    getUpstreamNodes(upstream: ApisixUpstream) {
        const nodes = normalizeUpstreamNodes(upstream)
        if (nodes.length === 0) return '未配置'
        const first = nodes[0]
        const firstLabel = `${first.host || '-'}:${first.port || '-'}`
        if (nodes.length === 1) return firstLabel
        return `${firstLabel} 等 ${nodes.length} 个节点`
    }

    getUpstreamNodeCount(upstream: ApisixUpstream) {
        return normalizeUpstreamNodes(upstream).length
    }

    getUpstreamTypeClass(type?: string) {
        if (type === 'chash') return 'bg-emerald-50 text-emerald-700'
        if (type === 'ewma') return 'bg-cyan-50 text-cyan-700'
        if (type === 'least_conn') return 'bg-amber-50 text-amber-700'
        return 'bg-indigo-50 text-indigo-700'
    }

    formatTs(ts?: number) {
        if (!ts) return '-'
        return new Date(ts * 1000).toLocaleString()
    }

    deleteUpstream(upstream: ApisixUpstream) {
        const id = upstream.id
        if (!id) return
        this.portal.showConfirm({
            title: '删除上游',
            message: `确定要删除上游 <strong class="text-slate-900">${upstream.name || id}</strong> 吗？仍被路由引用时 APISIX 可能拒绝删除。`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                await api.apisixUpstreamDelete(id)
                this.portal.showNotification('success', '删除成功')
                this.loadUpstreams()
            }
        })
    }

    mounted() {
        this.loadUpstreams()
    }
}

export default toNative(Upstreams)
</script>

<template>
  <div>
    <div class="card mb-4">
      <div class="card-toolbar">
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="page-icon bg-emerald-500">
              <i class="fas fa-diagram-project text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">上游管理</h1>
              <p class="text-xs text-slate-500">管理可复用的后端上游对象与负载均衡策略</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <PageSearch v-model="searchText" search-key="apisix-upstreams" placeholder="搜索上游、节点或策略..." width-class="w-56" focus-color="emerald" type-to-search />
            <button class="btn btn-secondary" @click="loadUpstreams()">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button v-if="portal.hasPerm('POST /api/apisix/upstream')" class="btn btn-emerald" @click="openCreateModal()">
              <i class="fas fa-plus"></i>新建上游
            </button>
          </div>
        </div>

        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="page-icon bg-emerald-500">
              <i class="fas fa-diagram-project text-white"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">上游管理</h1>
              <p class="text-xs text-slate-500 truncate">管理可复用上游对象</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadUpstreams()">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button v-if="portal.hasPerm('POST /api/apisix/upstream')" class="btn btn-emerald w-9 h-9 !px-0" title="新建上游" @click="openCreateModal()">
              <i class="fas fa-plus text-sm"></i>
            </button>
          </div>
        </div>
      </div>

      <div class="mobile-search">
        <PageSearch v-model="searchText" search-key="apisix-upstreams" placeholder="搜索上游、节点..." width-class="w-full" focus-color="emerald" />
      </div>

      <div v-if="loading" class="empty-state">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <div v-else-if="filteredUpstreams.length === 0" class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-diagram-project text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">{{ upstreams.length === 0 ? '暂无上游' : '未找到匹配上游' }}</p>
        <p class="text-sm text-slate-400">{{ upstreams.length === 0 ? '点击「新建上游」开始创建' : '尝试更换关键词或清空搜索条件' }}</p>
      </div>

      <div v-else class="space-y-3">
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="th">名称</th>
                <th class="th">策略</th>
                <th class="th">节点</th>
                <th class="th">创建时间</th>
                <th class="w-32 th-right">操作</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="upstream in filteredUpstreams" :key="upstream.id" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3 max-w-[280px]">
                  <div class="flex items-center gap-2 min-w-0">
                    <div class="row-icon bg-emerald-400">
                      <i class="fas fa-diagram-project text-white text-sm"></i>
                    </div>
                    <div class="min-w-0">
                      <span class="font-medium text-slate-800 truncate block">{{ upstream.name || upstream.id }}</span>
                      <span v-if="upstream.desc" class="text-xs text-slate-400 truncate block mt-0.5">{{ upstream.desc }}</span>
                    </div>
                  </div>
                </td>
                <td class="px-4 py-3">
                  <span :class="['text-xs px-2 py-0.5 rounded', getUpstreamTypeClass(upstream.type)]">{{ upstream.type || '-' }}</span>
                  <span v-if="upstream.type === 'chash' && upstream.key" class="ml-2 text-xs text-slate-400">{{ upstream.hash_on }}: {{ upstream.key }}</span>
                </td>
                <td class="px-4 py-3 text-xs text-slate-600 break-all">{{ getUpstreamNodes(upstream) }}</td>
                <td class="px-4 py-3 whitespace-nowrap text-sm text-slate-600">{{ formatTs(upstream.create_time) }}</td>
                <td class="px-4 py-3">
                  <div class="flex justify-end items-center gap-1">
                    <button v-if="portal.hasPerm('PUT /api/apisix/upstream/:id')" class="btn-icon btn-icon-emerald" title="编辑" @click="openEditModal(upstream)">
                      <i class="fas fa-pen text-xs"></i>
                    </button>
                    <button v-if="portal.hasPerm('DELETE /api/apisix/upstream/:id')" class="btn-icon btn-icon-red" title="删除" @click="deleteUpstream(upstream)">
                      <i class="fas fa-trash text-xs"></i>
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="md:hidden space-y-3 p-4">
          <div
            v-for="upstream in filteredUpstreams"
            :key="upstream.id"
            class="card-interactive"
          >
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-3 min-w-0 flex-1">
                <div class="list-icon bg-emerald-400">
                  <i class="fas fa-diagram-project text-white text-base"></i>
                </div>
                <div class="min-w-0">
                  <div class="font-medium text-sm text-slate-800 truncate">{{ upstream.name || upstream.id }}</div>
                  <div v-if="upstream.desc" class="text-xs text-slate-400 mt-0.5 truncate">{{ upstream.desc }}</div>
                </div>
              </div>
            </div>

            <div class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">策略</span>
              <span class="text-xs text-slate-600">{{ upstream.type || '-' }}</span>
            </div>
            <div v-if="upstream.type === 'chash' && upstream.key" class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">哈希</span>
              <span class="text-xs text-slate-500 break-all">{{ upstream.hash_on }}: {{ upstream.key }}</span>
            </div>
            <div class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">节点</span>
              <span class="text-xs text-slate-600 break-all">{{ getUpstreamNodes(upstream) }}</span>
            </div>
            <div class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">创建</span>
              <span class="text-xs text-slate-500">{{ formatTs(upstream.create_time) }}</span>
            </div>

            <div class="card-actions">
              <button v-if="portal.hasPerm('PUT /api/apisix/upstream/:id')" class="btn-icon btn-icon-emerald" title="编辑" @click="openEditModal(upstream)">
                <i class="fas fa-pen text-xs"></i><span class="text-xs ml-1">编辑</span>
              </button>
              <button v-if="portal.hasPerm('DELETE /api/apisix/upstream/:id')" class="btn-icon btn-icon-red" title="删除" @click="deleteUpstream(upstream)">
                <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <UpstreamEditModal ref="editModalRef" @success="loadUpstreams" />
  </div>
</template>
