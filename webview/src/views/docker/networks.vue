<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { DockerNetworkInfo } from '@/service/types'

import PageSearch from '@/component/page-search.vue'

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
        } finally {
            this.loading = false
        }
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
  <!-- Toolbar Bar -->
  <div class="page">
    <div class="page-toolbar">
      <!-- 桌面端 -->
      <div class="toolbar-desktop">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-purple-500">
            <i class="fas fa-network-wired text-white"></i>
          </div>
          <div>
            <h1 class="title-text">网络管理</h1>
            <p class="text-xs text-slate-500">管理 Docker 网络，配置容器间通信</p>
          </div>
        </div>
        <div class="action-group">
          <PageSearch v-model="searchText" search-key="docker-networks" placeholder="搜索网络名称、ID、驱动或子网..." focus-color="purple" type-to-search />
          <button class="btn btn-secondary" @click="loadNetworks()">
            <i class="fas fa-rotate"></i>刷新
          </button>
          <button v-if="portal.hasPerm('POST /api/docker/network')" class="btn btn-purple" @click="createModalRef?.show()">
            <i class="fas fa-plus"></i>新建网络
          </button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="toolbar-mobile">
        <div class="title-group">
          <div class="page-icon bg-purple-500">
            <i class="fas fa-network-wired text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="title-text">网络管理</h1>
            <p class="text-xs text-slate-500 truncate">管理容器网络</p>
          </div>
        </div>
        <div class="action-group-sm">
          <button class="btn btn-secondary btn-square" title="刷新" @click="loadNetworks()">
            <i class="fas fa-rotate text-sm"></i>
          </button>
          <button v-if="portal.hasPerm('POST /api/docker/network')" class="btn btn-purple btn-square" title="新建网络" @click="createModalRef?.show()">
            <i class="fas fa-plus text-sm"></i>
          </button>
        </div>
      </div>
    </div>

    <div class="mobile-search">
      <PageSearch v-model="searchText" search-key="docker-networks" placeholder="搜索网络名称、ID、驱动..." width-class="w-full" focus-color="purple" />
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="card-body">
      <div class="empty-state">
        <div class="spinner-lg"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
    </div>

    <!-- Network List -->
    <template v-else-if="filteredNetworks.length > 0">
      <!-- 桌面端表格视图 -->
      <div class="card-table hidden md:block">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-50 border-b border-slate-200">
              <th class="th">名称</th>
              <th class="w-24 th">驱动</th>
              <th class="th">子网</th>
              <th class="w-24 th">范围</th>
              <th class="w-32 th-right">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-for="net in filteredNetworks" :key="net.id" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3 max-w-[280px]">
                <div class="inline-info">
                  <div class="row-icon bg-purple-400">
                    <i class="fas fa-network-wired text-white text-sm"></i>
                  </div>
                  <div class="min-w-0">
                    <router-link v-if="portal.hasPerm('GET /api/docker/network/:id')" :to="'/docker/network/' + net.id" class="font-medium text-slate-800 hover:text-purple-600 transition-colors truncate block">{{ net.name }}</router-link>
                    <span v-else class="item-title">{{ net.name }}</span>
                    <code class="item-subtitle-mono">{{ net.id }}</code>
                  </div>
                </div>
              </td>
              <td class="px-4 py-3"><span class="badge-sm bg-purple-50 text-purple-700">{{ net.driver }}</span></td>
              <td class="px-4 py-3 font-mono text-sm text-slate-600">{{ net.subnet || '-' }}</td>
              <td class="td-text">{{ net.scope }}</td>
              <td class="px-4 py-3">
                <div class="table-actions">
                  <button v-if="portal.hasPerm('GET /api/docker/network/:id')" class="btn-icon btn-icon-slate" title="详情" @click="viewNetworkDetail(net)">
                    <i class="fas fa-circle-info text-xs"></i>
                  </button>
                  <button v-if="canDeleteNetwork(net) && portal.hasPerm('POST /api/docker/network/:id/action')" class="btn-icon btn-icon-red" title="删除" @click="handleNetworkAction(net, 'remove')">
                    <i class="fas fa-trash text-xs"></i>
                  </button>
                  <button v-else disabled class="btn-icon text-slate-300 cursor-not-allowed" :title="getDeleteDisabledReason(net)">
                    <i class="fas fa-trash text-xs"></i>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 移动端卡片视图 -->
      <div class="card-body md:hidden space-y-3">
        <div v-for="net in filteredNetworks" :key="net.id" class="card-interactive">
          <!-- 顶部：网络信息和图标 -->
          <div class="card-info-row">
            <div class="list-icon bg-purple-400">
              <i class="fas fa-network-wired text-white text-base"></i>
            </div>
            <div class="min-w-0">
              <router-link v-if="portal.hasPerm('GET /api/docker/network/:id')" :to="'/docker/network/' + net.id" class="font-medium text-slate-800 hover:text-purple-600 transition-colors text-sm truncate block">{{ net.name }}</router-link>
              <span v-else class="item-title-sm">{{ net.name }}</span>
              <code class="item-subtitle-mono">{{ net.id }}</code>
            </div>
          </div>

          <!-- 驱动 / 范围 -->
          <div class="card-prop-row-start">
            <span class="prop-label-start">驱动</span>
            <span class="badge-sm bg-purple-50 text-purple-700">{{ net.driver }}</span>
          </div>
          <div class="card-prop-row">
            <span class="text-xs text-slate-400 flex-shrink-0">范围</span>
            <span class="text-xs text-slate-500">{{ net.scope }}</span>
          </div>

          <div class="card-prop-row-start">
            <span class="prop-label-start">子网</span>
            <code class="font-mono text-xs text-slate-500">{{ net.subnet || '-' }}</code>
          </div>
          
          <!-- 底部：操作按钮 -->
          <div class="card-actions">
            <button v-if="portal.hasPerm('GET /api/docker/network/:id')" class="btn-icon btn-icon-slate" title="详情" @click="viewNetworkDetail(net)">
              <i class="fas fa-circle-info text-xs"></i><span class="text-xs ml-1">详情</span>
            </button>
            <button v-if="canDeleteNetwork(net) && portal.hasPerm('POST /api/docker/network/:id/action')" class="btn-icon btn-icon-red" title="删除" @click="handleNetworkAction(net, 'remove')">
              <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
            </button>
            <button v-else disabled class="btn-icon text-slate-300 cursor-not-allowed" :title="getDeleteDisabledReason(net)">
              <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
            </button>
          </div>
        </div>
      </div>
    </template>

    <!-- Empty State -->
    <div v-else class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-network-wired text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">{{ networks.length === 0 ? '暂无自定义网络' : '未找到匹配网络' }}</p>
        <p class="text-sm text-slate-400">{{ networks.length === 0 ? '点击「新建网络」添加自定义网络' : '尝试更换关键词或清空搜索条件' }}</p>
      </div>
    </div>
  </div>

  <NetworkCreateModal ref="createModalRef" @success="loadNetworks" />
</template>
