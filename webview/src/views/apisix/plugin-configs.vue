<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { ApisixPluginConfig } from '@/service/types'

import PageSearch from '@/component/page-search.vue'

import { usePortal } from '@/stores'

import PluginConfigEditModal from './widget/plugin-config-edit-modal.vue'

@Component({
    components: { PageSearch, PluginConfigEditModal }
})
class PluginConfigs extends Vue {
    portal = usePortal()

    @Ref readonly editModalRef!: InstanceType<typeof PluginConfigEditModal>

    configs: ApisixPluginConfig[] = []
    loading = false
    searchText = ''

    get filteredConfigs() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.configs
        return this.configs.filter((config: ApisixPluginConfig) => {
            const plugins = this.getPluginNames(config).join(' ').toLowerCase()
            return (
                (config.id || '').toLowerCase().includes(keyword) ||
                (config.desc || '').toLowerCase().includes(keyword) ||
                plugins.includes(keyword)
            )
        })
    }

    sortConfigs(data: ApisixPluginConfig[]) {
        data.sort((a: ApisixPluginConfig, b: ApisixPluginConfig) => (a.id || '').localeCompare(b.id || ''))
        return data
    }

    async loadConfigs() {
        this.loading = true
        try {
            this.configs = this.sortConfigs((await api.apisixPluginConfigList()).payload || [])
        } catch {
            this.portal.showNotification('error', '加载插件配置列表失败')
        } finally {
            this.loading = false
        }
    }

    openCreateModal() {
        this.editModalRef?.show()
    }

    openEditModal(config: ApisixPluginConfig | null) {
        this.editModalRef?.show(config)
    }

    getPluginNames(config: ApisixPluginConfig) {
        return Object.keys(config.plugins || {})
    }

    getPluginSummary(config: ApisixPluginConfig) {
        const names = this.getPluginNames(config)
        if (names.length === 0) return '未配置'
        if (names.length <= 3) return names.join('、')
        return `${names.slice(0, 3).join('、')} 等 ${names.length} 个插件`
    }

    formatTs(ts?: number) {
        if (!ts) return '-'
        return new Date(ts * 1000).toLocaleString()
    }

    deleteConfig(config: ApisixPluginConfig) {
        const id = config.id
        if (!id) return
        this.portal.showConfirm({
            title: '删除插件配置',
            message: `确定要删除插件配置 <strong class="text-slate-900">${id}</strong> 吗？仍被路由引用时 APISIX 可能拒绝删除。`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                await api.apisixPluginConfigDelete(id)
                this.portal.showNotification('success', '删除成功')
                this.loadConfigs()
            }
        })
    }

    mounted() {
        this.loadConfigs()
    }
}

export default toNative(PluginConfigs)
</script>

<template>
  <div>
    <div class="card mb-4">
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3">
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-rose-500 flex items-center justify-center">
              <i class="fas fa-puzzle-piece text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">插件配置</h1>
              <p class="text-xs text-slate-500">管理可复用的插件集合，供路由引用</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <PageSearch v-model="searchText" search-key="apisix-plugin-configs" placeholder="搜索 ID、描述或插件..." width-class="w-56" focus-color="rose" type-to-search />
            <button class="btn btn-sm btn-secondary" @click="loadConfigs()">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button v-if="portal.hasPerm('POST /api/apisix/plugin-config')" class="btn btn-sm btn-rose" @click="openCreateModal()">
              <i class="fas fa-plus"></i>新建配置
            </button>
          </div>
        </div>

        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="w-9 h-9 rounded-lg bg-rose-500 flex items-center justify-center flex-shrink-0">
              <i class="fas fa-puzzle-piece text-white"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">插件配置</h1>
              <p class="text-xs text-slate-500 truncate">管理可复用插件集合</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <button class="btn btn-sm btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadConfigs()">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button v-if="portal.hasPerm('POST /api/apisix/plugin-config')" class="btn btn-sm btn-rose w-9 h-9 !px-0" title="创建" @click="openCreateModal()">
              <i class="fas fa-plus text-sm"></i>
            </button>
          </div>
        </div>
      </div>

      <div class="md:hidden px-4 py-2 border-b border-slate-100">
        <PageSearch v-model="searchText" search-key="apisix-plugin-configs" placeholder="搜索插件配置..." width-class="w-full" focus-color="rose" />
      </div>

      <div v-if="loading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <div v-else-if="filteredConfigs.length === 0" class="flex flex-col items-center justify-center py-20">
        <div class="w-16 h-16 rounded-lg bg-slate-100 flex items-center justify-center mb-4">
          <i class="fas fa-puzzle-piece text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">{{ configs.length === 0 ? '暂无插件配置' : '未找到匹配插件配置' }}</p>
        <p class="text-sm text-slate-400">{{ configs.length === 0 ? '点击「新建配置」添加可复用 Plugin Config' : '尝试更换关键词或清空搜索条件' }}</p>
      </div>

      <div v-else class="space-y-3">
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">配置</th>
                <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">插件</th>
                <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">创建时间</th>
                <th class="w-32 px-4 py-3 text-right text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="config in filteredConfigs" :key="config.id" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3 max-w-[280px]">
                  <div class="flex items-center gap-2 min-w-0">
                    <div class="w-8 h-8 rounded-lg bg-rose-400 flex items-center justify-center flex-shrink-0">
                      <i class="fas fa-puzzle-piece text-white text-sm"></i>
                    </div>
                    <div class="min-w-0">
                      <span class="font-medium text-slate-800 truncate block font-mono">{{ config.id || '-' }}</span>
                      <span v-if="config.desc" class="text-xs text-slate-400 truncate block mt-0.5">{{ config.desc }}</span>
                    </div>
                  </div>
                </td>
                <td class="px-4 py-3">
                  <div v-if="getPluginNames(config).length > 0" class="flex flex-wrap gap-1">
                    <span v-for="name in getPluginNames(config)" :key="name" class="inline-flex items-center px-1.5 py-0.5 bg-rose-50 text-rose-700 rounded text-xs">{{ name }}</span>
                  </div>
                  <span v-else class="text-xs text-slate-400">未配置</span>
                </td>
                <td class="px-4 py-3 whitespace-nowrap text-sm text-slate-600">{{ formatTs(config.create_time) }}</td>
                <td class="px-4 py-3">
                  <div class="flex justify-end items-center gap-1">
                    <button v-if="portal.hasPerm('PUT /api/apisix/plugin-config/:id')" class="btn-icon text-rose-600 hover:bg-rose-50" title="编辑" @click="openEditModal(config)">
                      <i class="fas fa-pen text-xs"></i>
                    </button>
                    <button v-if="portal.hasPerm('DELETE /api/apisix/plugin-config/:id')" class="btn-icon text-red-600 hover:bg-red-50" title="删除" @click="deleteConfig(config)">
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
            v-for="config in filteredConfigs"
            :key="config.id"
            class="rounded-xl border border-slate-200 bg-white p-4 transition-all hover:shadow-sm"
          >
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-3 min-w-0 flex-1">
                <div class="w-10 h-10 rounded-lg bg-rose-400 flex items-center justify-center flex-shrink-0">
                  <i class="fas fa-puzzle-piece text-white text-base"></i>
                </div>
                <div class="min-w-0">
                  <div class="font-medium text-sm text-slate-800 truncate font-mono">{{ config.id || '-' }}</div>
                  <div v-if="config.desc" class="text-xs text-slate-400 mt-0.5 truncate">{{ config.desc }}</div>
                </div>
              </div>
            </div>

            <div class="flex items-start gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">插件</span>
              <div v-if="getPluginNames(config).length > 0" class="flex flex-wrap gap-1">
                <span v-for="name in getPluginNames(config)" :key="name" class="inline-flex items-center px-1.5 py-0.5 bg-rose-50 text-rose-700 rounded text-xs">{{ name }}</span>
              </div>
              <span v-else class="text-xs text-slate-400">未配置</span>
            </div>
            <div class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">创建</span>
              <span class="text-xs text-slate-500">{{ formatTs(config.create_time) }}</span>
            </div>

            <div class="flex flex-wrap gap-1.5 pt-2 border-t border-slate-100">
              <button v-if="portal.hasPerm('PUT /api/apisix/plugin-config/:id')" class="btn-icon text-rose-600 hover:bg-rose-50" title="编辑" @click="openEditModal(config)">
                <i class="fas fa-pen text-xs"></i><span class="text-xs ml-1">编辑</span>
              </button>
              <button v-if="portal.hasPerm('DELETE /api/apisix/plugin-config/:id')" class="btn-icon text-red-600 hover:bg-red-50" title="删除" @click="deleteConfig(config)">
                <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <PluginConfigEditModal ref="editModalRef" @success="loadConfigs" />
  </div>
</template>
