<script lang="ts">
import { Component, Inject, Ref, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { ApisixConsumer, ApisixRoute } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import ConsumerEditModal from '@/views/apisix/widget/consumer-edit-modal.vue'

@Component({
    components: { ConsumerEditModal }
})
class Consumers extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── Refs ───
    @Ref readonly editModalRef!: InstanceType<typeof ConsumerEditModal>

    // ─── 数据属性 ───
    consumers: ApisixConsumer[] = []
    whitelist: ApisixRoute[] = []
    loading = false
    searchText = ''

    // ─── 计算属性 ───
    get filteredConsumers() {
        if (!this.searchText) return this.consumers
        const s = this.searchText.toLowerCase()
        return this.consumers.filter((c: ApisixConsumer) =>
            (c.username || '').toLowerCase().includes(s) ||
            (c.desc || '').toLowerCase().includes(s)
        )
    }

    // ─── 方法 ───
    async loadConsumers() {
        this.loading = true
        try {
            const [consRes, wlRes] = await Promise.all([api.apisixListConsumers(), api.apisixGetWhitelist()])
            this.consumers = consRes.payload || []
            this.whitelist = wlRes.payload || []
        } catch (e) {
            this.actions.showNotification('error', '加载用户列表失败')
        }
        this.loading = false
    }

    getConsumerRoutes(username: string) {
        return this.whitelist.filter((r: ApisixRoute) => (r.consumers || []).includes(username)).map((r: ApisixRoute) => r.name || r.id)
    }

    formatTs(ts: number) {
        if (!ts) return '-'
        return new Date(ts * 1000).toLocaleString()
    }

    openCreateModal() {
        this.editModalRef?.show()
    }

    openEditModal(consumer: ApisixConsumer | null) {
        this.editModalRef?.show(consumer)
    }

    deleteConsumer(consumer: ApisixConsumer) {
        this.actions.showConfirm({
            title: '删除用户',
            message: `确定要删除用户 <strong class="text-slate-900">${consumer.username}</strong> 吗？此操作不可恢复。`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                await api.apisixDeleteConsumer(consumer.username)
                this.actions.showNotification('success', '删除成功')
                this.loadConsumers()
            }
        })
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadConsumers()
    }
}

export default toNative(Consumers)
</script>

<template>
  <div>
    <!-- Toolbar -->
    <div class="card mb-4">
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-violet-500 flex items-center justify-center">
              <i class="fas fa-users text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">用户管理</h1>
              <p class="text-xs text-slate-500">管理 APISIX Consumer</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <div class="relative">
              <input
                v-model="searchText"
                type="text"
                placeholder="搜索用户..."
                class="pl-8 pr-3 py-1.5 text-xs border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent w-48"
              />
              <i class="fas fa-search absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400 text-xs"></i>
            </div>
            <button @click="loadConsumers()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button @click="openCreateModal()" class="px-3 py-1.5 rounded-lg bg-violet-500 hover:bg-violet-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-plus"></i>创建
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="w-9 h-9 rounded-lg bg-violet-500 flex items-center justify-center flex-shrink-0">
              <i class="fas fa-users text-white"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">用户管理</h1>
              <p class="text-xs text-slate-500 truncate">管理 APISIX Consumer</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <button @click="loadConsumers()" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 flex items-center justify-center text-slate-600 transition-colors" title="刷新">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button @click="openCreateModal()" class="w-9 h-9 rounded-lg bg-violet-500 hover:bg-violet-600 flex items-center justify-center text-white transition-colors" title="创建">
              <i class="fas fa-plus text-sm"></i>
            </button>
          </div>
        </div>
      </div>
      <!-- 移动端搜索栏 -->
      <div class="md:hidden px-4 py-2 border-b border-slate-100">
        <div class="relative">
          <input
            v-model="searchText"
            type="text"
            placeholder="搜索用户..."
            class="w-full pl-8 pr-3 py-1.5 text-xs border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent"
          />
          <i class="fas fa-search absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400 text-xs"></i>
        </div>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <!-- 空状态 -->
      <div v-else-if="filteredConsumers.length === 0" class="flex flex-col items-center justify-center py-20">
        <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
          <i class="fas fa-users text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">暂无用户</p>
        <p class="text-sm text-slate-400">点击「创建」添加 Consumer 用户</p>
      </div>

      <!-- 用户列表 -->
      <div v-else class="space-y-3">
        <!-- 桌面端表格视图 -->
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">用户名</th>
                <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">描述</th>
                <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">API Key</th>
                <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">关联路由</th>
                <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">创建时间</th>
                <th class="w-32 px-4 py-3 text-right text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="consumer in filteredConsumers" :key="consumer.username" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3">
                  <div class="flex items-center gap-2">
                    <div class="w-8 h-8 rounded-lg bg-violet-400 flex items-center justify-center flex-shrink-0">
                      <i class="fas fa-user text-white text-sm"></i>
                    </div>
                    <span class="font-medium text-slate-800">{{ consumer.username }}</span>
                  </div>
                </td>
                <td class="px-4 py-3">
                  <span class="text-sm text-slate-600">{{ consumer.desc || '-' }}</span>
                </td>
                <td class="px-4 py-3">
                  <code class="text-xs bg-slate-100 px-2 py-1 rounded text-slate-600">{{ (consumer.plugins?.['key-auth'] as Record<string, unknown>)?.key || '-' }}</code>
                </td>
                <td class="px-4 py-3">
                  <div v-if="getConsumerRoutes(consumer.username).length > 0" class="flex flex-wrap gap-1">
                    <span v-for="name in getConsumerRoutes(consumer.username)" :key="name" class="inline-flex items-center px-1.5 py-0.5 bg-violet-50 text-violet-700 rounded text-xs">{{ name }}</span>
                  </div>
                  <span v-else class="text-xs text-slate-400">-</span>
                </td>
                <td class="px-4 py-3 whitespace-nowrap text-sm text-slate-600">
                  {{ formatTs(consumer.create_time) }}
                </td>
                <td class="px-4 py-3">
                  <div class="flex justify-end items-center gap-0.5">
                    <button @click="openEditModal(consumer)" class="btn-icon text-violet-600 hover:bg-violet-50" title="编辑">
                      <i class="fas fa-pen-to-square text-xs"></i>
                    </button>
                    <button @click="deleteConsumer(consumer)" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
                      <i class="fas fa-trash text-xs"></i>
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 移动端卡片视图 -->
        <div class="md:hidden space-y-3 p-4">
          <div 
            v-for="consumer in filteredConsumers" 
            :key="consumer.username"
            class="rounded-xl border border-slate-200 bg-white p-4 transition-all hover:shadow-sm"
          >
            <!-- 顶部：用户信息和图标 -->
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-3 min-w-0 flex-1">
                <div class="w-10 h-10 rounded-lg bg-violet-400 flex items-center justify-center flex-shrink-0">
                  <i class="fas fa-user text-white text-base"></i>
                </div>
                <div class="min-w-0">
                  <div class="flex items-center gap-2">
                    <span class="font-medium text-slate-800 text-sm truncate">{{ consumer.username }}</span>
                  </div>
                  <div class="text-xs text-slate-500 mt-1">{{ formatTs(consumer.create_time) }}</div>
                </div>
              </div>
            </div>
            
            <!-- 中间：描述和API Key信息 -->
            <div class="mb-3">
              <p class="text-xs text-slate-500 mb-1">描述</p>
              <span class="text-sm text-slate-600">{{ consumer.desc || '-' }}</span>
            </div>
            
            <div class="mb-3">
              <p class="text-xs text-slate-500 mb-1">API Key</p>
              <code class="text-xs bg-slate-100 px-2 py-1 rounded text-slate-600 break-all">{{ (consumer.plugins?.['key-auth'] as Record<string, unknown>)?.key || '-' }}</code>
            </div>
            
            <!-- 关联路由 -->
            <div class="mb-3">
              <p class="text-xs text-slate-500 mb-1">关联路由</p>
              <div v-if="getConsumerRoutes(consumer.username).length > 0" class="flex flex-wrap gap-1">
                <span v-for="name in getConsumerRoutes(consumer.username)" :key="name" class="inline-flex items-center px-1.5 py-0.5 bg-violet-50 text-violet-700 rounded text-xs">{{ name }}</span>
              </div>
              <span v-else class="text-xs text-slate-400">-</span>
            </div>
            
            <!-- 底部：操作按钮 -->
            <div class="flex flex-wrap gap-1 pt-2 border-t border-slate-100">
              <button @click="openEditModal(consumer)" class="btn-icon text-violet-600 hover:bg-violet-50" title="编辑">
                <i class="fas fa-pen-to-square text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">编辑</span>
              </button>
              <button @click="deleteConsumer(consumer)" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
                <i class="fas fa-trash text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">删除</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <ConsumerEditModal ref="editModalRef" @success="loadConsumers" />
  </div>
</template>
