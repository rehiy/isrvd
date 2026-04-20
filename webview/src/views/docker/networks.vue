<script lang="ts">
import { Component, Inject, Ref, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { NetworkInfo } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import NetworkCreateModal from '@/views/docker/widget/network-create-modal.vue'

@Component({
    expose: ['load', 'show'],
    components: { NetworkCreateModal }
})
class Networks extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── Refs ───
    @Ref readonly createModalRef!: InstanceType<typeof NetworkCreateModal>

    // ─── 数据属性 ───
    networks: NetworkInfo[] = []
    loading = false

    // ─── 方法 ───
    async loadNetworks() {
        this.loading = true
        try {
            const res = await api.listNetworks()
            this.networks = res.payload || []
        } catch (e) {
            this.actions.showNotification('error', '加载网络列表失败')
        }
        this.loading = false
    }

    handleNetworkAction(net: NetworkInfo, action: string) {
        this.actions.showConfirm({
            title: '删除网络',
            message: `确定要删除网络 <strong class="text-slate-900">${net.name}</strong> 吗？`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                await api.networkAction(net.id, action)
                this.actions.showNotification('success', '网络删除成功')
                this.loadNetworks()
            }
        })
    }

    viewNetworkDetail(net: NetworkInfo) {
        this.$router.push('/docker/network/' + net.id)
    }

    canDeleteNetwork(net: NetworkInfo) {
        const undeletableNames = ['bridge', 'host', 'none']
        return !undeletableNames.includes(net.name)
    }

    getDeleteDisabledReason(net: NetworkInfo) {
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
            <button @click="loadNetworks()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button @click="createModalRef?.show()" class="px-3 py-1.5 rounded-lg bg-purple-500 hover:bg-purple-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
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
            <button @click="loadNetworks()" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 flex items-center justify-center text-slate-600 transition-colors" title="刷新">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button @click="createModalRef?.show()" class="w-9 h-9 rounded-lg bg-purple-500 hover:bg-purple-600 flex items-center justify-center text-white transition-colors" title="创建">
              <i class="fas fa-plus text-sm"></i>
            </button>
          </div>
        </div>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <!-- Network List -->
      <div v-else-if="networks.length > 0" class="space-y-3">
        <!-- 桌面端表格视图 -->
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="w-1/4 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">名称</th>
                <th class="w-24 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">驱动</th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">子网</th>
                <th class="w-24 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">范围</th>
                <th class="w-32 px-4 py-3 text-right text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="net in networks" :key="net.id" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3">
                  <div class="flex items-center gap-2">
                    <div class="w-8 h-8 rounded-lg bg-purple-400 flex items-center justify-center">
                      <i class="fas fa-network-wired text-white text-sm"></i>
                    </div>
                    <span class="font-medium text-slate-800">{{ net.name }}</span>
                  </div>
                </td>
                <td class="px-4 py-3"><code class="text-xs bg-slate-100 px-2 py-1 rounded">{{ net.driver }}</code></td>
                <td class="px-4 py-3 font-mono text-sm text-slate-600">{{ net.subnet || '-' }}</td>
                <td class="px-4 py-3 text-sm text-slate-600">{{ net.scope }}</td>
                <td class="px-4 py-3">
                  <div class="flex justify-end items-center gap-0.5">
                    <button @click="viewNetworkDetail(net)" class="btn-icon text-purple-600 hover:bg-purple-50" title="详情">
                      <i class="fas fa-info-circle text-xs"></i>
                    </button>
                    <button
                      v-if="canDeleteNetwork(net)"
                      @click="handleNetworkAction(net, 'remove')"
                      class="btn-icon text-red-600 hover:bg-red-50"
                      title="删除"
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
            v-for="net in networks" 
            :key="net.id"
            class="rounded-xl border border-slate-200 bg-white p-4 transition-all hover:shadow-sm"
          >
            <!-- 顶部：网络信息和图标 -->
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-3 min-w-0 flex-1">
                <div class="w-10 h-10 rounded-lg bg-purple-400 flex items-center justify-center flex-shrink-0">
                  <i class="fas fa-network-wired text-white text-base"></i>
                </div>
                <div class="min-w-0">
                  <div class="flex items-center gap-2">
                    <span class="font-medium text-slate-800 text-sm truncate">{{ net.name }}</span>
                  </div>
                  <div class="flex items-center gap-3 mt-1">
                    <span class="text-xs text-slate-500">{{ net.scope }}</span>
                    <span class="text-xs text-slate-500">{{ net.driver }}</span>
                  </div>
                </div>
              </div>
            </div>
            
            <!-- 中间：子网信息 -->
            <div class="mb-3">
              <p class="text-xs text-slate-500 mb-1">子网</p>
              <code class="font-mono text-sm text-slate-600 break-all">{{ net.subnet || '-' }}</code>
            </div>
            
            <!-- 底部：操作按钮 -->
            <div class="flex flex-wrap gap-1 pt-2 border-t border-slate-100">
              <button @click="viewNetworkDetail(net)" class="btn-icon text-purple-600 hover:bg-purple-50" title="详情">
                <i class="fas fa-info-circle text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">详情</span>
              </button>
              <button
                v-if="canDeleteNetwork(net)"
                @click="handleNetworkAction(net, 'remove')"
                class="btn-icon text-red-600 hover:bg-red-50"
                title="删除"
              >
                <i class="fas fa-trash text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">删除</span>
              </button>
              <button
                v-else
                disabled
                class="btn-icon text-slate-300 cursor-not-allowed"
                :title="getDeleteDisabledReason(net)"
              >
                <i class="fas fa-trash text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">删除</span>
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Empty State -->
      <div v-else class="flex flex-col items-center justify-center py-20">
        <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
          <i class="fas fa-network-wired text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">暂无自定义网络</p>
        <p class="text-sm text-slate-400">点击「创建网络」添加自定义网络</p>
      </div>
    </div>

    <NetworkCreateModal ref="createModalRef" @success="loadNetworks" />
  </div>
</template>
