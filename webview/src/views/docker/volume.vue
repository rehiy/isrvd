<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { DockerVolumeDetail } from '@/service/types'

import { formatFileSize, formatTime } from '@/helper/utils'

import { usePortal } from '@/stores'

@Component
class VolumeDetail extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    detailData: DockerVolumeDetail | null = null
    loading = false
    formatFileSize = formatFileSize
    formatTime = formatTime

    get volumeName() {
        return decodeURIComponent(this.$route.params.name as string)
    }

    // ─── 方法 ───
    async loadDetail() {
        this.loading = true
        try {
            const res = await api.dockerVolume(this.volumeName)
            this.detailData = res.payload ?? null
        } catch {
            this.portal.showNotification('error', '获取数据卷详情失败')
        }
        this.loading = false
    }
    // ─── 生命周期 ───
    mounted() {
        this.loadDetail()
    }
}

export default toNative(VolumeDetail)
</script>

<template>
  <div>
    <div class="card mb-4">
      <!-- Toolbar -->
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-amber-500 flex items-center justify-center">
              <i class="fas fa-database text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">数据卷详情</h1>
              <p class="text-xs text-slate-600 font-mono truncate max-w-xs">{{ detailData?.name || volumeName }}</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <button class="btn btn-sm btn-secondary" @click="loadDetail()">
              <i class="fas fa-rotate"></i>刷新
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="w-9 h-9 rounded-lg bg-amber-500 flex items-center justify-center flex-shrink-0">
              <i class="fas fa-database text-white"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">数据卷详情</h1>
              <p class="text-xs text-slate-600 font-mono truncate">{{ detailData?.name || volumeName }}</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <button class="btn btn-sm btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadDetail()">
              <i class="fas fa-rotate text-sm"></i>
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
      <div v-else-if="detailData" class="p-4 md:p-6 space-y-6 text-sm">
        <!-- 基本信息 -->
        <div>
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">基本信息</h2>
          <div class="grid grid-cols-2 gap-3">
            <div class="col-span-2">
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">名称</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700 break-all">{{ detailData.name }}</div>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">驱动</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ detailData.driver }}</div>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">范围</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ detailData.scope }}</div>
            </div>
            <div class="col-span-2">
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">挂载点</label>
              <code class="block px-3 py-2 bg-slate-50 rounded-lg text-xs font-mono text-slate-700 break-all">{{ detailData.mountpoint }}</code>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">创建时间</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ formatTime(detailData.createdAt) }}</div>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">占用空间</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ detailData.size > 0 ? formatFileSize(detailData.size) : '-' }}</div>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">引用数</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ detailData.refCount || 0 }}</div>
            </div>
          </div>
        </div>

        <!-- 使用此卷的容器 -->
        <div>
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">
            使用此卷的容器
            <span v-if="detailData.usedBy" class="text-slate-400 normal-case font-normal ml-1">({{ detailData.usedBy.length }})</span>
          </h2>
          <div v-if="detailData.usedBy && detailData.usedBy.length > 0" class="border border-slate-200 rounded-xl overflow-hidden">
            <table class="w-full">
              <thead>
                <tr class="bg-slate-50 border-b border-slate-200">
                  <th class="px-3 py-2 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">名称</th>
                  <th class="px-3 py-2 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">挂载路径</th>
                  <th class="px-3 py-2 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">权限</th>
                </tr>
              </thead>
              <tbody class="bg-white divide-y divide-slate-100">
                <tr v-for="ct in detailData.usedBy" :key="ct.id" class="hover:bg-slate-50 transition-colors">
                  <td class="px-3 py-2">
                    <div class="flex items-center gap-1.5">
                      <div class="w-6 h-6 rounded bg-amber-100 flex items-center justify-center">
                        <i class="fas fa-box text-amber-500 text-xs"></i>
                      </div>
                      <span class="text-sm text-slate-800">{{ ct.name || ct.id }}</span>
                    </div>
                  </td>
                  <td class="px-3 py-2 font-mono text-xs text-slate-600">{{ ct.mountPath }}</td>
                  <td class="px-3 py-2 text-xs text-slate-600">{{ ct.readOnly ? '只读' : '读写' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div v-else class="text-sm text-slate-400 py-8 text-center bg-slate-50 rounded-xl">
            暂无容器使用此数据卷
          </div>
        </div>
      </div>

      <!-- Empty -->
      <div v-else class="flex flex-col items-center justify-center py-20">
        <div class="w-16 h-16 rounded-lg bg-slate-100 flex items-center justify-center mb-4">
          <i class="fas fa-database text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium">未找到数据卷详情</p>
      </div>
    </div>
  </div>
</template>
