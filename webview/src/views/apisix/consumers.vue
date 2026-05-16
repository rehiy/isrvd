<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { ApisixConsumer, ApisixRoute } from '@/service/types'

import PageSearch from '@/component/page-search.vue'

import { usePortal } from '@/stores'

import ConsumerEditModal from './widget/consumer-edit-modal.vue'

@Component({
    components: { PageSearch, ConsumerEditModal }
})
class Consumers extends Vue {
    portal = usePortal()

    // ─── Refs ───
    @Ref readonly editModalRef!: InstanceType<typeof ConsumerEditModal>

    // ─── 数据属性 ───
    consumers: ApisixConsumer[] = []
    whitelist: ApisixRoute[] = []
    loading = false
    searchText = ''

    // ─── 计算属性 ───
    get filteredConsumers() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.consumers
        return this.consumers.filter((c: ApisixConsumer) =>
            (c.username || '').toLowerCase().includes(keyword) ||
            (c.desc || '').toLowerCase().includes(keyword)
        )
    }

    // ─── 方法 ───
    async loadConsumers() {
        this.loading = true
        try {
            const [consRes, wlRes] = await Promise.all([api.apisixConsumerList(), api.apisixWhitelist()])
            this.consumers = consRes.payload || []
            this.whitelist = wlRes.payload || []
        } catch {
            this.portal.showNotification('error', '加载消费者列表失败')
        } finally {
            this.loading = false
        }
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
        this.portal.showConfirm({
            title: '删除消费者',
            message: `确定要删除消费者 <strong class="text-slate-900">${consumer.username}</strong> 吗？此操作不可恢复。`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                await api.apisixConsumerDelete(consumer.username)
                this.portal.showNotification('success', '删除成功')
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
              <h1 class="text-lg font-semibold text-slate-800">消费者管理</h1>
              <p class="text-xs text-slate-500">管理 APISIX Consumer 及其认证凭据</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <PageSearch v-model="searchText" search-key="apisix-consumers" placeholder="搜索消费者..." width-class="w-48" focus-color="violet" type-to-search />
            <button class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors" @click="loadConsumers()">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button v-if="portal.hasPerm('POST /api/apisix/consumer')" class="px-3 py-1.5 rounded-lg bg-violet-500 hover:bg-violet-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors" @click="openCreateModal()">
              <i class="fas fa-plus"></i>新建消费者
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
              <h1 class="text-lg font-semibold text-slate-800 truncate">消费者管理</h1>
              <p class="text-xs text-slate-500 truncate">管理 Consumer 与凭据</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <button class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 flex items-center justify-center text-slate-600 transition-colors" title="刷新" @click="loadConsumers()">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button v-if="portal.hasPerm('POST /api/apisix/consumer')" class="w-9 h-9 rounded-lg bg-violet-500 hover:bg-violet-600 flex items-center justify-center text-white transition-colors" title="新建消费者" @click="openCreateModal()">
              <i class="fas fa-plus text-sm"></i>
            </button>
          </div>
        </div>
      </div>
      <!-- 移动端搜索栏 -->
      <div class="md:hidden px-4 py-2 border-b border-slate-100">
        <PageSearch v-model="searchText" search-key="apisix-consumers" placeholder="搜索消费者..." width-class="w-full" focus-color="violet" />
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <!-- 空状态 -->
      <div v-else-if="filteredConsumers.length === 0" class="flex flex-col items-center justify-center py-20">
        <div class="w-16 h-16 rounded-lg bg-slate-100 flex items-center justify-center mb-4">
          <i class="fas fa-users text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">{{ consumers.length === 0 ? '暂无消费者' : '未找到匹配消费者' }}</p>
        <p class="text-sm text-slate-400">{{ consumers.length === 0 ? '点击「新建消费者」开始创建' : '尝试更换关键词或清空搜索条件' }}</p>
      </div>

      <!-- 消费者列表 -->
      <div v-else class="space-y-3">
        <!-- 桌面端表格视图 -->
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">名称</th>
                <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">插件配置</th>
                <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">授权路由</th>
                <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">创建时间</th>
                <th class="w-32 px-4 py-3 text-right text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="consumer in filteredConsumers" :key="consumer.username" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3 max-w-[280px]">
                  <div class="flex items-center gap-2 min-w-0">
                    <div class="w-8 h-8 rounded-lg bg-violet-400 flex items-center justify-center flex-shrink-0">
                      <i class="fas fa-user text-white text-sm"></i>
                    </div>
                    <div class="min-w-0">
                      <span class="font-medium text-slate-800 truncate block">{{ consumer.username }}</span>
                      <span v-if="consumer.desc" class="text-xs text-slate-400 truncate block mt-0.5">{{ consumer.desc }}</span>
                    </div>
                  </div>
                </td>
                <td class="px-4 py-3">
                  <div v-if="Object.keys(consumer.plugins || {}).length > 0" class="flex flex-wrap gap-1">
                    <span v-for="(_, name) in consumer.plugins" :key="name" class="inline-flex items-center px-1.5 py-0.5 bg-blue-50 text-blue-700 rounded text-xs">{{ name }}</span>
                  </div>
                  <span v-else class="text-xs text-slate-400">-</span>
                </td>
                <td class="px-4 py-3">
                  <div v-if="getConsumerRoutes(consumer.username).length > 0" class="flex flex-wrap gap-1">
                    <span v-for="name in getConsumerRoutes(consumer.username)" :key="name" class="inline-flex items-center px-1.5 py-0.5 bg-amber-50 text-amber-700 rounded text-xs">{{ name }}</span>
                  </div>
                  <span v-else class="text-xs text-slate-400">-</span>
                </td>
                <td class="px-4 py-3 whitespace-nowrap text-sm text-slate-600">
                  {{ formatTs(consumer.create_time) }}
                </td>
                <td class="px-4 py-3">
                  <div class="flex justify-end items-center gap-1">
                    <button v-if="portal.hasPerm('PUT /api/apisix/consumer/:username')" class="btn-icon text-violet-600 hover:bg-violet-50" title="编辑" @click="openEditModal(consumer)">
                      <i class="fas fa-pen text-xs"></i>
                    </button>
                    <button v-if="portal.hasPerm('DELETE /api/apisix/consumer/:username')" class="btn-icon text-red-600 hover:bg-red-50" title="删除" @click="deleteConsumer(consumer)">
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
            <!-- 顶部：消费者信息和图标 -->
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-3 min-w-0 flex-1">
                <div class="w-10 h-10 rounded-lg bg-violet-400 flex items-center justify-center flex-shrink-0">
                  <i class="fas fa-user text-white text-base"></i>
                </div>
                <div class="min-w-0">
                  <span class="font-medium text-slate-800 text-sm truncate block">{{ consumer.username }}</span>
                  <span v-if="consumer.desc" class="text-xs text-slate-400 truncate block mt-0.5">{{ consumer.desc }}</span>
                </div>
              </div>
            </div>
            
            <!-- 中间：API Key和创建时间 -->
            <div class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">创建</span>
              <span class="text-xs text-slate-500">{{ formatTs(consumer.create_time) }}</span>
            </div>
            
            <div class="flex items-start gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">插件</span>
              <div v-if="Object.keys(consumer.plugins || {}).length > 0" class="flex flex-wrap gap-1">
                <span v-for="(_, name) in consumer.plugins" :key="name" class="inline-flex items-center px-1.5 py-0.5 bg-blue-50 text-blue-700 rounded text-xs">{{ name }}</span>
              </div>
              <span v-else class="text-xs text-slate-400">-</span>
            </div>

            <!-- 授权路由 -->
            <div class="flex items-start gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">路由</span>
              <div v-if="getConsumerRoutes(consumer.username).length > 0" class="flex flex-wrap gap-1">
                <span v-for="name in getConsumerRoutes(consumer.username)" :key="name" class="inline-flex items-center px-1.5 py-0.5 bg-amber-50 text-amber-700 rounded text-xs">{{ name }}</span>
              </div>
              <span v-else class="text-xs text-slate-400">-</span>
            </div>
            
            <!-- 底部：操作按钮 -->
            <div class="flex flex-wrap gap-1.5 pt-2 border-t border-slate-100">
              <button v-if="portal.hasPerm('PUT /api/apisix/consumer/:username')" class="btn-icon text-violet-600 hover:bg-violet-50" title="编辑" @click="openEditModal(consumer)">
                <i class="fas fa-pen text-xs"></i><span class="text-xs ml-1">编辑</span>
              </button>
              <button v-if="portal.hasPerm('DELETE /api/apisix/consumer/:username')" class="btn-icon text-red-600 hover:bg-red-50" title="删除" @click="deleteConsumer(consumer)">
                <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <ConsumerEditModal ref="editModalRef" @success="loadConsumers" />
  </div>
</template>
