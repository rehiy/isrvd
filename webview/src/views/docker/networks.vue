<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { DockerNetworkInfo } from '@/service/types'

import PageSearch from '@/component/page-search.vue'

import { usePortal } from '@/stores'

import NetworkCreateModal from './widget/network-create-modal.vue'

@Component({
    expose: ['load', 'show'],
    components: { PageSearch, NetworkCreateModal }
})
class Networks extends Vue {
    portal = usePortal()

    // ─── Refs ───
    @Ref readonly createModalRef!: InstanceType<typeof NetworkCreateModal>

    // ─── 数据属性 ───
    networks: DockerNetworkInfo[] = []
    loading = false
    searchText = ''

    get filteredNetworks() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.networks
        return this.networks.filter((network: DockerNetworkInfo) =>
            network.name.toLowerCase().includes(keyword) ||
            network.id.toLowerCase().includes(keyword) ||
            (network.driver || '').toLowerCase().includes(keyword) ||
            (network.subnet || '').toLowerCase().includes(keyword) ||
            (network.scope || '').toLowerCase().includes(keyword)
        )
    }

    // ─── 方法 ───
    async loadNetworks() {
        this.loading = true
        try {
            const res = await api.dockerNetworkList()
            this.networks = res.payload || []
        } catch {
            this.portal.showNotification('error', '加载网络列表失败')
        }
        this.loading = false
    }

    handleNetworkAction(net: DockerNetworkInfo, action: string) {
        this.portal.showConfirm({
            title: '删除网络',
            message: `确定要删除网络 <strong class="text-slate-900">${net.name}</strong> 吗？`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                await api.dockerNetworkAction(net.id, action)
                this.portal.showNotification('success', '网络删除成功')
                this.loadNetworks()
            }
        })
    }

    viewNetworkDetail(net: DockerNetworkInfo) {
        this.$router.push('/docker/network/' + net.id)
    }

    canDeleteNetwork(net: DockerNetworkInfo) {
        const undeletableNames = ['bridge', 'host', 'none']
        return !undeletableNames.includes(net.name)
    }

    getDeleteDisabledReason(net: DockerNetworkInfo) {
        const networkNames: Record<string, string> = {
            bridge: '默认桥接网络',
            host: '主机网络',
            none: '空网络'
        }
        return `${networkNames[net.name] || '系统网络'}不可删除`
    }

    createNetworkModal() {
        this.createModalRef?.show()
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadNetworks()
    }
}

export default toNative(Networks)
</script>

<template>
  <div>
    <!-- Toolbar Bar -->
    <div class="card mb-4">
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-purple-500 flex items-center justify-center">
              <i class="fas fa-network-wired text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">网络管理</h1>
              <p class="text-xs text-slate-500">管理 Docker 网络</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <PageSearch v-model="searchText" search-key="docker-networks" placeholder="搜索网络名称、ID、驱动或子网..." width-class="w-64" focus-color="purple" type-to-search />
            <button class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors" @click="loadNetworks()">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button v-if="portal.hasPerm('POST /api/docker/network')" class="px-3 py-1.5 rounded-lg bg-purple-500 hover:bg-purple-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors" @click="createModalRef?.show()">
              <i class="fas fa-plus"></i>创建
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="w-9 h-9 rounded-lg bg-purple-500 flex items-center justify-center flex-shrink-0">
              <i class="fas fa-network-wired text-white"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">网络管理</h1>
              <p class="text-xs text-slate-500 truncate">管理 Docker 网络</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <button class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 flex items-center justify-center text-slate-600 transition-colors" title="刷新" @click="loadNetworks()">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button v-if="portal.hasPerm('POST /api/docker/network')" class="w-9 h-9 rounded-lg bg-purple-500 hover:bg-purple-600 flex items-center justify-center text-white transition-colors" title="创建" @click="createModalRef?.show()">
              <i class="fas fa-plus text-sm"></i>
            </button>
          </div>
        </div>
      </div>

      <div class="md:hidden px-4 py-2 border-b border-slate-100">
        <PageSearch v-model="searchText" search-key="docker-networks" placeholder="搜索网络名称、ID、驱动..." width-class="w-full" focus-color="purple" />
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <!-- Network List -->
      <div v-else-if="filteredNetworks.length > 0" class="space-y-3">
        <!-- 桌面端表格视图 -->
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">名称</th>
                <th class="w-24 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">驱动</th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">子网</th>
                <th class="w-24 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">范围</th>
                <th class="w-32 px-4 py-3 text-right text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="net in filteredNetworks" :key="net.id" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3 max-w-[280px]">
                  <div class="flex items-center gap-2 min-w-0">
                    <div class="w-8 h-8 rounded-lg bg-purple-400 flex items-center justify-center flex-shrink-0">
                      <i class="fas fa-network-wired text-white text-sm"></i>
                    </div>
                    <div class="min-w-0">
                      <router-link v-if="portal.hasPerm('GET /api/docker/network/:id')" :to="'/docker/network/' + net.id" class="font-medium text-slate-800 hover:text-purple-600 transition-colors truncate block">{{ net.name }}</router-link>
                      <span v-else class="font-medium text-slate-800 truncate block">{{ net.name }}</span>
                      <code class="text-xs text-slate-400 font-mono truncate block mt-0.5">{{ net.id }}</code>
                    </div>
                  </div>
                </td>
                <td class="px-4 py-3"><span class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-medium bg-purple-50 text-purple-700">{{ net.driver }}</span></td>
                <td class="px-4 py-3 font-mono text-sm text-slate-600">{{ net.subnet || '-' }}</td>
                <td class="px-4 py-3 text-sm text-slate-600">{{ net.scope }}</td>
                <td class="px-4 py-3">
                  <div class="flex justify-end items-center gap-1">
                    <button v-if="portal.hasPerm('GET /api/docker/network/:id')" class="btn-icon text-slate-600 hover:bg-slate-50" title="详情" @click="viewNetworkDetail(net)">
                      <i class="fas fa-circle-info text-xs"></i>
                    </button>
                    <button
                      v-if="canDeleteNetwork(net) && portal.hasPerm('POST /api/docker/network/:id/action')"
                      class="btn-icon text-red-600 hover:bg-red-50"
                      title="删除"
                      @click="handleNetworkAction(net, 'remove')"
                    >
                      <i class="fas fa-trash text-xs"></i>
                    </button>
                    <button
                      v-else
                      disabled
                      class="btn-icon text-slate-300 cursor-not-allowed"
                      :title="getDeleteDisabledReason(net)"
                    >
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
            v-for="net in filteredNetworks" 
            :key="net.id"
            class="rounded-xl border border-slate-200 bg-white p-4 transition-all hover:shadow-sm"
          >
            <!-- 顶部：网络信息和图标 -->
            <div class="flex items-center gap-3 min-w-0 flex-1 mb-3">
              <div class="w-10 h-10 rounded-lg bg-purple-400 flex items-center justify-center flex-shrink-0">
                <i class="fas fa-network-wired text-white text-base"></i>
              </div>
              <div class="min-w-0">
                <router-link v-if="portal.hasPerm('GET /api/docker/network/:id')" :to="'/docker/network/' + net.id" class="font-medium text-slate-800 hover:text-purple-600 transition-colors text-sm truncate block">{{ net.name }}</router-link>
                <span v-else class="font-medium text-slate-800 text-sm truncate block">{{ net.name }}</span>
                <code class="text-xs text-slate-400 font-mono truncate block mt-0.5">{{ net.id }}</code>
              </div>
            </div>

            <!-- 驱动 / 范围 -->
            <div class="flex items-start gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">驱动</span>
              <span class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-medium bg-purple-50 text-purple-700">{{ net.driver }}</span>
            </div>
            <div class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">范围</span>
              <span class="text-xs text-slate-500">{{ net.scope }}</span>
            </div>

            <div class="flex items-start gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">子网</span>
              <code class="font-mono text-xs text-slate-500">{{ net.subnet || '-' }}</code>
            </div>
            
            <!-- 底部：操作按钮 -->
            <div class="flex flex-wrap gap-1.5 pt-2 border-t border-slate-100">
              <button v-if="portal.hasPerm('GET /api/docker/network/:id')" class="btn-icon text-slate-600 hover:bg-slate-50" title="详情" @click="viewNetworkDetail(net)">
                <i class="fas fa-circle-info text-xs"></i><span class="text-xs ml-1">详情</span>
              </button>
              <button
                v-if="canDeleteNetwork(net) && portal.hasPerm('POST /api/docker/network/:id/action')"
                class="btn-icon text-red-600 hover:bg-red-50"
                title="删除"
                @click="handleNetworkAction(net, 'remove')"
              >
                <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
              </button>
              <button
                v-else
                disabled
                class="btn-icon text-slate-300 cursor-not-allowed"
                :title="getDeleteDisabledReason(net)"
              >
                <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Empty State -->
      <div v-else class="flex flex-col items-center justify-center py-20">
        <div class="w-16 h-16 rounded-lg bg-slate-100 flex items-center justify-center mb-4">
          <i class="fas fa-network-wired text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">{{ networks.length === 0 ? '暂无自定义网络' : '未找到匹配网络' }}</p>
        <p class="text-sm text-slate-400">{{ networks.length === 0 ? '点击「创建」添加自定义网络' : '尝试更换关键词或清空搜索条件' }}</p>
      </div>
    </div>

    <NetworkCreateModal ref="createModalRef" @success="loadNetworks" />
  </div>
</template>
