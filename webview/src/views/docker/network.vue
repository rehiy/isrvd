<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { DockerNetworkDetail } from '@/service/types'

@Component
class NetworkDetail extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    detailData: DockerNetworkDetail | null = null
    loading = false

    get networkId() {
        return this.$route.params.id as string
    }

    async loadDetail() {
        this.loading = true
        try {
            const res = await api.dockerNetwork(this.networkId)
            this.detailData = res.payload ?? null
        } finally {
            this.loading = false
        }
    }
    // ─── 生命周期 ───
    mounted() {
        this.loadDetail()
    }
}

export default toNative(NetworkDetail)
</script>

<template>
  <div class="card">
    <!-- Toolbar -->
    <div class="card-toolbar">
      <!-- 桌面端 -->
      <div class="hidden md:flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-purple-500">
            <i class="fas fa-network-wired text-white"></i>
          </div>
          <div>
            <h1 class="text-lg font-semibold text-slate-800">网络详情</h1>
            <p class="text-xs text-slate-600 font-mono truncate max-w-xs">{{ detailData?.name || networkId }}</p>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <button class="btn btn-secondary" @click="loadDetail()">
            <i class="fas fa-rotate"></i>刷新
          </button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="flex md:hidden items-center justify-between">
        <div class="flex items-center gap-3 min-w-0 flex-1">
          <div class="page-icon bg-purple-500">
            <i class="fas fa-network-wired text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">网络详情</h1>
            <p class="text-xs text-slate-600 font-mono truncate">{{ detailData?.name || networkId }}</p>
          </div>
        </div>
        <div class="flex items-center gap-1 flex-shrink-0">
          <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadDetail()">
            <i class="fas fa-rotate text-sm"></i>
          </button>
        </div>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="card-body">
      <div class="empty-state">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
    </div>

    <!-- Detail Content -->
    <div v-else-if="detailData" class="card-body space-y-4 text-sm">
      <!-- 基本信息 -->
      <div>
        <h2 class="section-title">基本信息</h2>
        <div class="grid grid-cols-2 gap-3">
          <div class="col-span-2">
            <label class="form-label">名称</label>
            <div class="detail-value">{{ detailData.name }}</div>
          </div>
          <div class="col-span-2">
            <label class="form-label">ID</label>
            <code class="detail-value-mono">{{ detailData.id }}</code>
          </div>
          <div>
            <label class="form-label">驱动</label>
            <div class="detail-value">{{ detailData.driver }}</div>
          </div>
          <div>
            <label class="form-label">范围</label>
            <div class="detail-value">{{ detailData.scope }}</div>
          </div>
          <div>
            <label class="form-label">子网</label>
            <div class="detail-value">{{ detailData.subnet || '-' }}</div>
          </div>
          <div>
            <label class="form-label">网关</label>
            <div class="detail-value">{{ detailData.gateway || '-' }}</div>
          </div>
          <div>
            <label class="form-label">内部网络</label>
            <div class="detail-value">{{ detailData.internal ? '是' : '否' }}</div>
          </div>
          <div>
            <label class="form-label">IPv6</label>
            <div class="detail-value">{{ detailData.enableIPv6 ? '已启用' : '未启用' }}</div>
          </div>
        </div>
      </div>

      <!-- 已连接的容器 -->
      <div>
        <h2 class="section-title section-title-table">
          已连接容器
          <span v-if="detailData.containers" class="text-slate-400 normal-case font-normal ml-1">({{ detailData.containers.length }})</span>
        </h2>
        <div v-if="detailData.containers && detailData.containers.length > 0" class="border-x border-b border-slate-200 rounded-b-xl overflow-hidden">
          <table class="w-full">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="th-sm">名称</th>
                <th class="th-sm">IPv4</th>
                <th class="th-sm">MAC 地址</th>
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
        <div v-else class="detail-value text-slate-400 py-8 text-center rounded-xl">
          暂无容器连接到此网络
        </div>
      </div>
    </div>

    <!-- Empty -->
    <div v-else class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-network-wired text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium">未找到网络详情</p>
      </div>
    </div>
  </div>
</template>
