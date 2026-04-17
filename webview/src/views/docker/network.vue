<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { NetworkInspectResponse } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

@Component
class NetworkDetail extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    detailData: NetworkInspectResponse | null = null
    loading = false

    get networkId() {
        return this.$route.params.id as string
    }

    // ─── 方法 ───
    async loadDetail() {
        this.loading = true
        try {
            const res = await api.networkInspect(this.networkId)
            this.detailData = res.payload ?? null
        } catch (e) {
            this.actions.showNotification('error', '获取网络详情失败')
        }
        this.loading = false
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadDetail()
    }
}

export default toNative(NetworkDetail)
</script>

<template>
  <div>
    <div class="card mb-4">
      <!-- Toolbar -->
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-6 py-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-purple-500 flex items-center justify-center">
              <i class="fas fa-network-wired text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">网络详情</h1>
              <p class="text-xs text-slate-500 font-mono truncate max-w-xs">{{ detailData?.name || networkId }}</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <button @click="loadDetail()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button @click="$router.back()" class="px-3 py-1.5 rounded-lg bg-purple-500 hover:bg-purple-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-arrow-left"></i>返回
            </button>
          </div>
        </div>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <!-- Detail Content -->
      <div v-else-if="detailData" class="p-6 space-y-6 text-sm">
        <!-- 基本信息 -->
        <div>
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">基本信息</h2>
          <div class="bg-slate-50 rounded-xl p-4">
            <div class="grid grid-cols-2 gap-4">
              <div>
                <span class="text-xs text-slate-500">名称</span>
                <p class="text-sm font-medium text-slate-800 mt-0.5">{{ detailData.name }}</p>
              </div>
              <div class="overflow-hidden">
                <span class="text-xs text-slate-500">ID</span>
                <p class="text-xs font-mono text-slate-600 mt-0.5 truncate" :title="detailData.id">{{ detailData.id }}</p>
              </div>
              <div>
                <span class="text-xs text-slate-500">驱动</span>
                <p class="text-sm font-medium text-slate-800 mt-0.5"><code class="bg-slate-100 px-2 py-0.5 rounded">{{ detailData.driver }}</code></p>
              </div>
              <div>
                <span class="text-xs text-slate-500">范围</span>
                <p class="text-sm font-medium text-slate-800 mt-0.5">{{ detailData.scope }}</p>
              </div>
              <div>
                <span class="text-xs text-slate-500">子网</span>
                <p class="text-sm font-mono text-slate-800 mt-0.5">{{ detailData.subnet || '-' }}</p>
              </div>
              <div>
                <span class="text-xs text-slate-500">网关</span>
                <p class="text-sm font-mono text-slate-800 mt-0.5">{{ detailData.gateway || '-' }}</p>
              </div>
              <div>
                <span class="text-xs text-slate-500">内部网络</span>
                <p class="text-sm font-medium text-slate-800 mt-0.5">{{ detailData.internal ? '是' : '否' }}</p>
              </div>
              <div>
                <span class="text-xs text-slate-500">IPv6</span>
                <p class="text-sm font-medium text-slate-800 mt-0.5">{{ detailData.enableIPv6 ? '已启用' : '未启用' }}</p>
              </div>
            </div>
          </div>
        </div>

        <!-- 已连接的容器 -->
        <div>
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">
            已连接容器
            <span v-if="detailData.containers" class="text-slate-400 normal-case font-normal ml-1">({{ detailData.containers.length }})</span>
          </h2>
          <div v-if="detailData.containers && detailData.containers.length > 0" class="border border-slate-200 rounded-xl overflow-hidden">
            <table class="w-full">
              <thead>
                <tr class="bg-slate-50 border-b border-slate-200">
                  <th class="px-3 py-2 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">名称</th>
                  <th class="px-3 py-2 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">IPv4</th>
                  <th class="px-3 py-2 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">MAC 地址</th>
                </tr>
              </thead>
              <tbody class="bg-white divide-y divide-slate-100">
                <tr v-for="ct in detailData.containers" :key="ct.id" class="hover:bg-slate-50 transition-colors">
                  <td class="px-3 py-2">
                    <div class="flex items-center gap-1.5">
                      <div class="w-6 h-6 rounded bg-purple-100 flex items-center justify-center">
                        <i class="fas fa-box text-purple-500 text-xs"></i>
                      </div>
                      <span class="text-sm text-slate-800">{{ ct.name || ct.id }}</span>
                    </div>
                  </td>
                  <td class="px-3 py-2 font-mono text-xs text-slate-600">{{ ct.ipv4 || '-' }}</td>
                  <td class="px-3 py-2 font-mono text-xs text-slate-600">{{ ct.macAddress || '-' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div v-else class="text-sm text-slate-400 py-8 text-center bg-slate-50 rounded-xl">
            暂无容器连接到此网络
          </div>
        </div>
      </div>

      <!-- Empty -->
      <div v-else class="flex flex-col items-center justify-center py-20">
        <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
          <i class="fas fa-network-wired text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium">未找到网络详情</p>
      </div>
    </div>
  </div>
</template>
