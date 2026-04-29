<script lang="ts">
import { Component, Inject, Ref, Vue, toNative } from 'vue-facing-decorator'

import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import api from '@/service/api'
import type { ApisixPluginConfig } from '@/service/types'

import PluginConfigEditModal from './widget/plugin-config-edit-modal.vue'

@Component({
    components: { PluginConfigEditModal }
})
class PluginConfigs extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    @Ref readonly editModalRef!: InstanceType<typeof PluginConfigEditModal>

    configs: ApisixPluginConfig[] = []
    loading = false
    searchText = ''

    get filteredConfigs() {
        if (!this.searchText) return this.configs
        const s = this.searchText.toLowerCase()
        return this.configs.filter((config: ApisixPluginConfig) => {
            const plugins = this.getPluginNames(config).join(' ').toLowerCase()
            return (
                (config.id || '').toLowerCase().includes(s) ||
                (config.desc || '').toLowerCase().includes(s) ||
                plugins.includes(s)
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
            this.configs = this.sortConfigs((await api.apisixListPluginConfigs()).payload || [])
        } catch {
            this.actions.showNotification('error', '加载插件配置列表失败')
        }
        this.loading = false
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
        this.actions.showConfirm({
            title: '删除插件配置',
            message: `确定要删除插件配置 <strong class="text-slate-900">${id}</strong> 吗？仍被路由引用时 APISIX 可能拒绝删除。`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                await api.apisixDeletePluginConfig(id)
                this.actions.showNotification('success', '删除成功')
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
              <p class="text-xs text-slate-500">管理 APISIX Plugin Config，供多个路由复用</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <div class="relative">
              <input v-model="searchText" type="text" placeholder="搜索 ID、描述或插件..." class="pl-8 pr-3 py-1.5 text-xs border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-rose-500 focus:border-transparent w-56" />
              <i class="fas fa-search absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400 text-xs"></i>
            </div>
            <button @click="loadConfigs()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button v-if="actions.hasPerm('apisix', true)" @click="openCreateModal()" class="px-3 py-1.5 rounded-lg bg-rose-500 hover:bg-rose-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-plus"></i>创建
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
              <p class="text-xs text-slate-500 truncate">管理可复用插件配置</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <button @click="loadConfigs()" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 flex items-center justify-center text-slate-600 transition-colors" title="刷新">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button v-if="actions.hasPerm('apisix', true)" @click="openCreateModal()" class="w-9 h-9 rounded-lg bg-rose-500 hover:bg-rose-600 flex items-center justify-center text-white transition-colors" title="创建">
              <i class="fas fa-plus text-sm"></i>
            </button>
          </div>
        </div>
      </div>

      <div class="md:hidden px-4 py-2 border-b border-slate-100">
        <div class="relative">
          <input v-model="searchText" type="text" placeholder="搜索插件配置..." class="w-full pl-8 pr-3 py-1.5 text-xs border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-rose-500 focus:border-transparent" />
          <i class="fas fa-search absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400 text-xs"></i>
        </div>
      </div>

      <div v-if="loading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <div v-else-if="filteredConfigs.length === 0" class="flex flex-col items-center justify-center py-20">
        <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
          <i class="fas fa-puzzle-piece text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">暂无插件配置</p>
        <p class="text-sm text-slate-400">点击「创建」添加可复用 Plugin Config</p>
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
                  <div class="flex flex-wrap gap-1">
                    <span v-for="name in getPluginNames(config)" :key="name" class="text-xs bg-rose-50 px-2 py-1 rounded text-rose-700">{{ name }}</span>
                    <span v-if="getPluginNames(config).length === 0" class="text-xs text-slate-400">未配置</span>
                  </div>
                </td>
                <td class="px-4 py-3 whitespace-nowrap text-sm text-slate-600">{{ formatTs(config.create_time) }}</td>
                <td class="px-4 py-3">
                  <div class="flex justify-end items-center gap-0.5">
                    <button v-if="actions.hasPerm('apisix', true)" @click="openEditModal(config)" class="btn-icon text-blue-600 hover:bg-blue-50" title="编辑">
                      <i class="fas fa-pen text-xs"></i>
                    </button>
                    <button v-if="actions.hasPerm('apisix', true)" @click="deleteConfig(config)" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
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

            <div class="flex items-start gap-2 mb-2">
              <span class="text-xs text-slate-400 flex-shrink-0">插件</span>
              <span class="text-xs bg-rose-50 px-2 py-1 rounded text-rose-700 break-all">{{ getPluginSummary(config) }}</span>
            </div>
            <div class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">创建</span>
              <span class="text-xs text-slate-600">{{ formatTs(config.create_time) }}</span>
            </div>

            <div class="flex flex-wrap gap-1 pt-2 border-t border-slate-100">
              <button v-if="actions.hasPerm('apisix', true)" @click="openEditModal(config)" class="btn-icon text-blue-600 hover:bg-blue-50" title="编辑">
                <i class="fas fa-pen text-xs"></i><span class="text-xs ml-1">编辑</span>
              </button>
              <button v-if="actions.hasPerm('apisix', true)" @click="deleteConfig(config)" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
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
