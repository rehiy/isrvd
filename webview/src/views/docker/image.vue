<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { DockerImageDetail } from '@/service/types'

import { formatFileSize, formatTime } from '@/helper/utils'

@Component
class ImageDetail extends Vue {
    portal = usePortal()
    inspectData: DockerImageDetail | null = null
    loading = false

    get imageId() {
        return this.$route.params.id as string
    }

    get primaryTag() {
        const tag = this.repoTags.find(t => t && t !== '<none>:<none>')
        if (tag) return tag

        const digestName = this.digestImageName(this.repoDigests[0] || '')
        return digestName || this.shortHash(this.inspectData?.shortId || this.inspectData?.id || this.imageId) || '镜像详情'
    }

    get imageSubtitle() {
        return this.shortHash(this.inspectData?.id || this.imageId)
    }

    get repoTags() {
        return this.inspectData?.repoTags || []
    }

    get repoDigests() {
        return this.inspectData?.repoDigests || []
    }

    get envList() {
        return this.inspectData?.env || []
    }

    get labelEntries() {
        if (!this.inspectData?.labels) return []
        return Object.entries(this.inspectData.labels)
    }

    get layerDetails() {
        return this.inspectData?.layerDetails || []
    }

    get entrypointText() {
        return (this.inspectData?.entrypoint || []).join(' ')
    }

    get cmdText() {
        return (this.inspectData?.cmd || []).join(' ')
    }

    get exposedPortsText() {
        return (this.inspectData?.exposedPorts || []).join(', ')
    }

    stripSha256(value: string) {
        return value.startsWith('sha256:') ? value.slice(7) : value
    }

    shortHash(value: string) {
        if (!value) return ''
        const hash = this.stripSha256(value)
        return `sha256:${hash.slice(0, 12)}`
    }

    digestImageName(digest: string) {
        if (!digest) return ''
        const atIndex = digest.indexOf('@')
        if (atIndex <= 0) return ''
        return digest.slice(0, atIndex)
    }

    formatDigest(digest: string) {
        const atIndex = digest.indexOf('@')
        if (atIndex <= 0) return this.shortHash(digest)
        return `${digest.slice(0, atIndex)}@${this.shortHash(digest.slice(atIndex + 1))}`
    }

    async loadDetail() {
        this.loading = true
        try {
            const res = await api.dockerImage(this.imageId)
            this.inspectData = res.payload ?? null
        } finally {
            this.loading = false
        }
    }

    mounted() {
        this.loadDetail()
    }

    formatFileSize = formatFileSize
    formatTime = formatTime
}

export default toNative(ImageDetail)
</script>

<template>
  <div class="card">
    <div class="card-toolbar">
      <!-- 桌面端 -->
      <div class="hidden md:flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-blue-500">
            <i class="fas fa-compact-disc text-white text-sm"></i>
          </div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">{{ primaryTag }}</h1>
            <p class="text-xs text-slate-600 font-mono truncate max-w-xs">{{ imageSubtitle }}</p>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <button class="btn btn-secondary" :disabled="loading" @click="loadDetail">
            <i class="fas fa-rotate"></i>刷新
          </button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="flex md:hidden items-center justify-between">
        <div class="flex items-center gap-3 min-w-0 flex-1">
          <div class="page-icon bg-blue-500 flex-shrink-0">
            <i class="fas fa-compact-disc text-white text-sm"></i>
          </div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">{{ primaryTag }}</h1>
            <p class="text-xs text-slate-600 font-mono truncate">{{ imageSubtitle }}</p>
          </div>
        </div>
        <div class="flex items-center gap-1 flex-shrink-0">
          <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" :disabled="loading" @click="loadDetail">
            <i class="fas fa-rotate text-sm"></i>
          </button>
        </div>
      </div>
    </div>

    <div v-if="loading" class="empty-state">
      <div class="w-12 h-12 spinner mb-3"></div>
      <p class="text-slate-500">加载中...</p>
    </div>

    <div v-else-if="inspectData" class="card-body space-y-4">
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <!-- 基本信息 -->
        <div class="detail-card detail-card-blue">
          <div class="detail-card-bar detail-card-bar-blue"></div>
          <h2 class="detail-card-title"><div class="detail-card-icon bg-gradient-to-br from-blue-500 to-blue-600"><i class="fas fa-circle-info text-white text-[9px]"></i></div>基本信息</h2>
          <div class="detail-card-body">
            <div>
              <div class="text-xs text-slate-500 mb-1">镜像 ID</div>
              <code class="detail-code-block">{{ inspectData.id }}</code>
            </div>
            <div class="detail-row"><span class="detail-label">短 ID</span><code class="detail-value code">{{ shortHash(inspectData.shortId || inspectData.id) }}</code></div>
            <div class="detail-row"><span class="detail-label">创建时间</span><span class="detail-value">{{ formatTime(inspectData.created) }}</span></div>
            <div class="detail-row"><span class="detail-label">大小</span><span class="detail-value">{{ formatFileSize(inspectData.size) }}</span></div>
            <div class="detail-row"><span class="detail-label">层数</span><span class="detail-value">{{ inspectData.layers }}</span></div>
            <div v-if="inspectData.author" class="detail-row"><span class="detail-label">作者</span><span class="detail-value">{{ inspectData.author }}</span></div>
          </div>
        </div>

        <!-- 平台信息 -->
        <div class="detail-card detail-card-indigo">
          <div class="detail-card-bar detail-card-bar-indigo"></div>
          <h2 class="detail-card-title"><div class="detail-card-icon bg-gradient-to-br from-indigo-500 to-indigo-600"><i class="fas fa-microchip text-white text-[9px]"></i></div>平台信息</h2>
          <div class="detail-card-body">
            <div class="detail-row"><span class="detail-label">操作系统</span><span class="detail-value">{{ inspectData.os || '-' }}</span></div>
            <div class="detail-row"><span class="detail-label">架构</span><span class="detail-value">{{ inspectData.architecture || '-' }}</span></div>
            <div class="detail-row"><span class="detail-label">标签数</span><span class="detail-value">{{ repoTags.length }}</span></div>
            <div class="detail-row"><span class="detail-label">Digest 数</span><span class="detail-value">{{ repoDigests.length }}</span></div>
          </div>
        </div>
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <!-- 标签 -->
        <div class="detail-card detail-card-blue">
          <div class="detail-card-bar detail-card-bar-blue"></div>
          <h2 class="detail-card-title"><div class="detail-card-icon bg-gradient-to-br from-blue-500 to-blue-600"><i class="fas fa-tags text-white text-[9px]"></i></div>标签</h2>
          <div v-if="repoTags.length" class="space-y-2">
            <code v-for="tag in repoTags" :key="tag" class="detail-code-block">{{ tag }}</code>
          </div>
          <p v-else class="detail-empty">无标签</p>
        </div>

        <!-- Digest -->
        <div class="detail-card detail-card-slate">
          <div class="detail-card-bar detail-card-bar-slate"></div>
          <h2 class="detail-card-title"><div class="detail-card-icon bg-gradient-to-br from-slate-500 to-slate-600"><i class="fas fa-fingerprint text-white text-[9px]"></i></div>Digest</h2>
          <div v-if="repoDigests.length" class="space-y-2">
            <code v-for="digest in repoDigests" :key="digest" class="detail-code-block text-slate-700" :title="digest">{{ formatDigest(digest) }}</code>
          </div>
          <p v-else class="detail-empty">无 Digest</p>
        </div>
      </div>

      <!-- 运行配置 -->
      <div class="detail-card detail-card-amber">
        <div class="detail-card-bar detail-card-bar-amber"></div>
        <h2 class="detail-card-title"><div class="detail-card-icon bg-gradient-to-br from-amber-500 to-amber-600"><i class="fas fa-sliders text-white text-[9px]"></i></div>运行配置</h2>
        <div class="space-y-4">
          <div>
            <div class="text-xs text-slate-500 mb-1">工作目录</div>
            <code class="detail-code-block">{{ inspectData.workingDir || '-' }}</code>
          </div>
          <div>
            <div class="text-xs text-slate-500 mb-1">Entrypoint</div>
            <code class="detail-code-block">{{ entrypointText || '-' }}</code>
          </div>
          <div>
            <div class="text-xs text-slate-500 mb-1">CMD</div>
            <code class="detail-code-block">{{ cmdText || '-' }}</code>
          </div>
          <div>
            <div class="text-xs text-slate-500 mb-1">暴露端口</div>
            <code class="detail-code-block">{{ exposedPortsText || '-' }}</code>
          </div>
        </div>
      </div>

      <!-- 环境变量 -->
      <div class="detail-card detail-card-slate">
        <div class="detail-card-bar detail-card-bar-slate"></div>
        <h2 class="detail-card-title"><div class="detail-card-icon bg-gradient-to-br from-slate-500 to-slate-600"><i class="fas fa-terminal text-white text-[9px]"></i></div>环境变量</h2>
        <div v-if="envList.length" class="detail-grid">
          <code v-for="env in envList" :key="env" class="detail-grid-item">{{ env }}</code>
        </div>
        <p v-else class="detail-empty">无环境变量</p>
      </div>

      <!-- Labels -->
      <div class="detail-card detail-card-purple">
        <div class="detail-card-bar detail-card-bar-purple"></div>
        <h2 class="detail-card-title"><div class="detail-card-icon bg-gradient-to-br from-purple-500 to-purple-600"><i class="fas fa-tags text-white text-[9px]"></i></div>Labels</h2>
        <div v-if="labelEntries.length" class="detail-grid">
          <div v-for="[key, value] in labelEntries" :key="key" class="detail-grid-item">
            <span class="text-slate-500">{{ key }}</span><span class="text-slate-300 mx-1">=</span><span class="text-slate-700">{{ value }}</span>
          </div>
        </div>
        <p v-else class="detail-empty">无标签</p>
      </div>

      <!-- 层信息 -->
      <div class="detail-card detail-card-indigo">
        <div class="detail-card-bar detail-card-bar-indigo"></div>
        <h2 class="detail-card-title"><div class="detail-card-icon bg-gradient-to-br from-indigo-500 to-indigo-600"><i class="fas fa-layer-group text-white text-[9px]"></i></div>层信息（{{ inspectData.layers }} 个实际层，共 {{ layerDetails.length }} 步）</h2>
        <div v-if="layerDetails.length" class="space-y-2">
          <div v-for="(layer, idx) in layerDetails" :key="idx" class="detail-list-item grid grid-cols-[1.75rem_3.75rem_minmax(0,1fr)_8.5rem] gap-2 items-center" :class="layer.empty ? 'bg-slate-100/70' : 'bg-white'">
            <span
              class="w-6 h-6 rounded-lg flex items-center justify-center text-xs font-bold"
              :class="layer.empty ? 'bg-slate-200 text-slate-400' : 'bg-blue-50 text-blue-600'"
            >{{ layerDetails.length - idx }}</span>
            <span v-if="!layer.empty" class="text-xs text-slate-500 text-right tabular-nums">{{ formatFileSize(layer.size) }}</span>
            <span v-else class="text-xs text-slate-400 text-right">空层</span>
            <code class="text-xs font-mono text-slate-700 truncate min-w-0">{{ layer.createdBy || '(无命令)' }}</code>
            <code v-if="!layer.empty && layer.digest" class="text-xs font-mono text-slate-400 truncate text-right" :title="layer.digest">{{ formatDigest(layer.digest) }}</code>
            <span v-else class="text-xs text-slate-300 text-right">-</span>
          </div>
        </div>
        <p v-else class="detail-empty">无层信息</p>
      </div>
    </div>

    <div v-else class="empty-state">
      <div class="empty-state-icon">
        <i class="fas fa-compact-disc text-4xl text-slate-300"></i>
      </div>
      <p class="text-slate-600 font-medium">未找到镜像详情</p>
    </div>
  </div>
</template>
