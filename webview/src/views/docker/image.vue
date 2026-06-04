<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { DockerImageDetail } from '@/service/types'

import { formatFileSize, formatTime } from '@/helper/utils'

@Component
class ImageDetail extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    inspectData: DockerImageDetail | null = null
    loading = false
    formatFileSize = formatFileSize
    formatTime = formatTime

    get imageId() {
        return this.$route.params.id as string
    }

    // ─── 方法 ───
    async loadDetail() {
        this.loading = true
        try {
            const res = await api.dockerImage(this.imageId)
            this.inspectData = res.payload ?? null
        } finally {
            this.loading = false
        }
    }
    // ─── 生命周期 ───
    mounted() {
        this.loadDetail()
    }
}

export default toNative(ImageDetail)
</script>

<template>
  <div class="card">
    <!-- Toolbar -->
    <div class="card-toolbar">
      <!-- 桌面端 -->
      <div class="hidden md:flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-blue-500">
            <i class="fas fa-compact-disc text-white"></i>
          </div>
          <div>
            <h1 class="text-lg font-semibold text-slate-800">镜像详情</h1>
            <p class="text-xs text-slate-600 font-mono truncate max-w-xs">{{ imageId }}</p>
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
          <div class="page-icon bg-blue-500">
            <i class="fas fa-compact-disc text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">镜像详情</h1>
            <p class="text-xs text-slate-600 font-mono truncate">{{ imageId }}</p>
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
    <div v-else-if="inspectData" class="card-body space-y-4 text-sm">
      <!-- 基本信息 -->
      <div>
        <h2 class="section-title">基本信息</h2>
        <div class="grid grid-cols-2 gap-3">
          <div class="col-span-2">
            <label class="form-label">镜像 ID</label>
            <code class="block px-3 py-2 bg-slate-50 rounded-lg text-xs font-mono text-slate-700 break-all">{{ inspectData.id }}</code>
          </div>
          <div>
            <label class="form-label">架构</label>
            <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ inspectData.architecture || '-' }}</div>
          </div>
          <div>
            <label class="form-label">操作系统</label>
            <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ inspectData.os || '-' }}</div>
          </div>
          <div>
            <label class="form-label">大小</label>
            <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ formatFileSize(inspectData.size) }}</div>
          </div>
          <div>
            <label class="form-label">层数</label>
            <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ inspectData.layers }}</div>
          </div>
          <div class="col-span-2">
            <label class="form-label">创建时间</label>
            <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ formatTime(inspectData.created) }}</div>
          </div>
          <div v-if="inspectData.author" class="col-span-2">
            <label class="form-label">作者</label>
            <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ inspectData.author }}</div>
          </div>
        </div>
      </div>

      <!-- 标签 -->
      <div v-if="inspectData.repoTags && inspectData.repoTags.length > 0">
        <h2 class="section-title">标签</h2>
        <div class="space-y-1">
          <div v-for="tag in inspectData.repoTags" :key="tag" class="text-xs font-mono text-slate-700">{{ tag }}</div>
        </div>
      </div>

      <!-- Digest -->
      <div v-if="inspectData.repoDigests && inspectData.repoDigests.length > 0">
        <h2 class="section-title">Digest</h2>
        <div class="space-y-1">
          <code v-for="d in inspectData.repoDigests" :key="d" class="block px-3 py-2 bg-slate-50 rounded-lg text-xs font-mono text-slate-600 break-all">{{ d }}</code>
        </div>
      </div>

      <!-- 运行配置 -->
      <div>
        <h2 class="section-title">运行配置</h2>
        <div class="space-y-3">
          <div v-if="inspectData.workingDir">
            <label class="block text-xs text-slate-500 mb-1">工作目录</label>
            <code class="block px-3 py-2 bg-slate-50 rounded-lg text-xs font-mono text-slate-700">{{ inspectData.workingDir }}</code>
          </div>
          <div v-if="inspectData.entrypoint && inspectData.entrypoint.length > 0">
            <label class="block text-xs text-slate-500 mb-1">Entrypoint</label>
            <code class="block px-3 py-2 bg-slate-50 rounded-lg text-xs font-mono text-slate-700">{{ inspectData.entrypoint.join(' ') }}</code>
          </div>
          <div v-if="inspectData.cmd && inspectData.cmd.length > 0">
            <label class="block text-xs text-slate-500 mb-1">CMD</label>
            <code class="block px-3 py-2 bg-slate-50 rounded-lg text-xs font-mono text-slate-700">{{ inspectData.cmd.join(' ') }}</code>
          </div>
          <div v-if="inspectData.exposedPorts && inspectData.exposedPorts.length > 0">
            <label class="block text-xs text-slate-500 mb-1">暴露端口</label>
            <div class="text-xs font-mono text-slate-700">{{ inspectData.exposedPorts.join(', ') }}</div>
          </div>
          <div v-if="!inspectData.workingDir && !inspectData.entrypoint?.length && !inspectData.cmd?.length && !inspectData.exposedPorts?.length" class="text-sm text-slate-400">
            无运行配置
          </div>
        </div>
      </div>

      <!-- 环境变量 -->
      <div v-if="inspectData.env && inspectData.env.length > 0">
        <h2 class="section-title">环境变量</h2>
        <div class="border border-slate-200 rounded-lg divide-y divide-slate-100">
          <div v-for="(env, idx) in inspectData.env" :key="idx" class="px-3 py-1.5">
            <code class="text-xs font-mono text-slate-600 break-all">{{ env }}</code>
          </div>
        </div>
      </div>

      <!-- Labels -->
      <div v-if="inspectData.labels && Object.keys(inspectData.labels).length > 0">
        <h2 class="section-title">Labels</h2>
        <div class="border border-slate-200 rounded-lg divide-y divide-slate-100">
          <div v-for="(val, key) in inspectData.labels" :key="key" class="px-3 py-1.5 flex gap-2">
            <code class="text-xs font-mono text-blue-600 shrink-0">{{ key }}</code>
            <span class="text-slate-400">=</span>
            <code class="text-xs font-mono text-slate-600 break-all">{{ val }}</code>
          </div>
        </div>
      </div>

      <!-- 层信息 -->
      <div v-if="inspectData.layerDetails && inspectData.layerDetails.length > 0">
        <h2 class="section-title">
          层信息（{{ inspectData.layers }} 个实际层，共 {{ inspectData.layerDetails.length }} 步）
        </h2>
        <div class="border border-slate-200 rounded-lg divide-y divide-slate-100 overflow-hidden">
          <div v-for="(layer, idx) in inspectData.layerDetails" :key="idx" class="px-3 py-2" :class="layer.empty ? 'bg-slate-50/50' : 'bg-white'">
            <div class="flex items-start gap-2">
              <!-- 层序号 + 类型标记 -->
              <div class="flex items-center gap-1.5 shrink-0 mt-0.5">
                <span
                  class="w-6 h-6 rounded flex items-center justify-center text-xs font-bold"
                  :class="layer.empty ? 'bg-slate-100 text-slate-400' : 'bg-blue-50 text-blue-600'"
                >{{ inspectData.layerDetails.length - idx }}</span>
                <span v-if="!layer.empty" class="text-xs text-slate-500">{{ formatFileSize(layer.size) }}</span>
                <span v-else class="text-xs text-slate-400">空层</span>
              </div>
              <!-- 构建命令 -->
              <div class="flex-1 min-w-0">
                <code class="text-xs font-mono text-slate-700 break-all leading-relaxed">{{ layer.createdBy || '(无命令)' }}</code>
                <div v-if="!layer.empty && layer.digest" class="mt-1">
                  <code class="text-xs font-mono text-slate-400 break-all">{{ layer.digest }}</code>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Empty -->
    <div v-else class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-compact-disc text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium">未找到镜像详情</p>
      </div>
    </div>
  </div>
</template>
