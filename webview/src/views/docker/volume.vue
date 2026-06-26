<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { DockerVolumeDetail } from '@/service/types'

import { formatFileSize, formatTime } from '@/helper/utils'

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
            const res = await api.dockerVolumeInspect(this.volumeName)
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

export default toNative(VolumeDetail)
</script>

<template>
  <div class="page">
    <!-- Toolbar -->
    <div class="page-toolbar">
      <!-- 桌面端 -->
      <div class="toolbar-desktop">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-amber-500">
            <i class="fas fa-database text-white"></i>
          </div>
          <div>
            <h1 class="text-lg font-semibold text-slate-800">数据卷详情</h1>
            <p class="text-xs text-slate-600 font-mono truncate max-w-xs">{{ detailData?.name || volumeName }}</p>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <button class="btn btn-secondary" @click="loadDetail()">
            <i class="fas fa-rotate"></i>刷新
          </button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="toolbar-mobile">
        <div class="title-group">
          <div class="page-icon bg-amber-500">
            <i class="fas fa-database text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="title-text">数据卷详情</h1>
            <p class="text-xs text-slate-600 font-mono truncate">{{ detailData?.name || volumeName }}</p>
          </div>
        </div>
        <div class="action-group-sm">
          <button class="btn btn-secondary btn-square" title="刷新" @click="loadDetail()">
            <i class="fas fa-rotate text-sm"></i>
          </button>
        </div>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="card-body">
      <div class="empty-state">
        <div class="spinner-lg"></div>
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
          <div>
            <label class="form-label">驱动</label>
            <div class="detail-value">{{ detailData.driver }}</div>
          </div>
          <div>
            <label class="form-label">范围</label>
            <div class="detail-value">{{ detailData.scope }}</div>
          </div>
          <div class="col-span-2">
            <label class="form-label">挂载点</label>
            <code class="detail-value-mono">{{ detailData.mountpoint }}</code>
          </div>
          <div>
            <label class="form-label">创建时间</label>
            <div class="detail-value">{{ formatTime(detailData.createdAt) }}</div>
          </div>
          <div>
            <label class="form-label">占用空间</label>
            <div class="detail-value">{{ detailData.size > 0 ? formatFileSize(detailData.size) : '-' }}</div>
          </div>
          <div>
            <label class="form-label">引用数</label>
            <div class="detail-value">{{ detailData.refCount >= 0 ? detailData.refCount : '-' }}</div>
          </div>
        </div>
      </div>

      <!-- 使用此卷的容器 -->
      <div>
        <h2 class="section-title section-title-table">
          使用此卷的容器
          <span v-if="detailData.usedBy" class="text-slate-400 normal-case font-normal ml-1">({{ detailData.usedBy.length }})</span>
        </h2>
        <div v-if="detailData.usedBy && detailData.usedBy.length > 0" class="border-x border-b border-slate-200 rounded-b-xl overflow-hidden">
          <table class="w-full">
            <thead>
              <tr class="bg-slate-100 border-b border-slate-200">
                <th class="th-sm">名称</th>
                <th class="th-sm">挂载路径</th>
                <th class="th-sm">权限</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-100">
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
        <div v-else class="detail-value text-slate-400 py-8 text-center rounded-xl">
          暂无容器使用此数据卷
        </div>
      </div>
    </div>

    <!-- Empty -->
    <div v-else class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-database text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium">未找到数据卷详情</p>
      </div>
    </div>
  </div>
</template>
