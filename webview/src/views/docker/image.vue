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
            const res = await api.dockerImageInspect(this.imageId)
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
  <div class="page">
    <div class="page-toolbar">
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

    <div v-else-if="inspectData" class="card-body space-y-4 text-sm">
      <!-- 基本信息 & 平台信息 -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div>
          <h2 class="section-title">基本信息</h2>
          <div class="space-y-3">
            <div>
              <label class="form-label">镜像 ID</label>
              <code class="detail-value-mono">{{ inspectData.id }}</code>
            </div>
            <div>
              <label class="form-label">短 ID</label>
              <code class="detail-value-mono">{{ shortHash(inspectData.shortId || inspectData.id) }}</code>
            </div>
            <div>
              <label class="form-label">创建时间</label>
              <div class="detail-value">{{ formatTime(inspectData.created) }}</div>
            </div>
            <div>
              <label class="form-label">大小</label>
              <div class="detail-value">{{ formatFileSize(inspectData.size) }}</div>
            </div>
            <div>
              <label class="form-label">层数</label>
              <div class="detail-value">{{ inspectData.layers }}</div>
            </div>
            <div v-if="inspectData.author">
              <label class="form-label">作者</label>
              <div class="detail-value">{{ inspectData.author }}</div>
            </div>
          </div>
        </div>

        <div>
          <h2 class="section-title">平台信息</h2>
          <div class="space-y-3">
            <div>
              <label class="form-label">操作系统</label>
              <div class="detail-value">{{ inspectData.os || '-' }}</div>
            </div>
            <div>
              <label class="form-label">架构</label>
              <div class="detail-value">{{ inspectData.architecture || '-' }}</div>
            </div>
            <div>
              <label class="form-label">标签数</label>
              <div class="detail-value">{{ repoTags.length }}</div>
            </div>
            <div>
              <label class="form-label">Digest 数</label>
              <div class="detail-value">{{ repoDigests.length }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- 标签 & Digest -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div>
          <h2 class="section-title">标签</h2>
          <div v-if="repoTags.length" class="space-y-2">
            <code v-for="tag in repoTags" :key="tag" class="detail-value-mono">{{ tag }}</code>
          </div>
          <div v-else class="detail-value text-slate-400 text-sm">无标签</div>
        </div>

        <div>
          <h2 class="section-title">Digest</h2>
          <div v-if="repoDigests.length" class="space-y-2">
            <code v-for="digest in repoDigests" :key="digest" class="detail-value-mono" :title="digest">{{ formatDigest(digest) }}</code>
          </div>
          <div v-else class="detail-value text-slate-400 text-sm">无 Digest</div>
        </div>
      </div>

      <!-- 运行配置 -->
      <div>
        <h2 class="section-title">运行配置</h2>
        <div class="space-y-3">
          <div>
            <label class="form-label">工作目录</label>
            <code class="detail-value-mono">{{ inspectData.workingDir || '-' }}</code>
          </div>
          <div>
            <label class="form-label">Entrypoint</label>
            <code class="detail-value-mono">{{ entrypointText || '-' }}</code>
          </div>
          <div>
            <label class="form-label">CMD</label>
            <code class="detail-value-mono">{{ cmdText || '-' }}</code>
          </div>
          <div>
            <label class="form-label">暴露端口</label>
            <code class="detail-value-mono">{{ exposedPortsText || '-' }}</code>
          </div>
        </div>
      </div>

      <!-- 环境变量 -->
      <div>
        <h2 class="section-title">环境变量</h2>
        <div v-if="envList.length" class="space-y-2">
          <code v-for="env in envList" :key="env" class="detail-value-mono">{{ env }}</code>
        </div>
        <div v-else class="detail-value text-slate-400 text-sm">无环境变量</div>
      </div>

      <!-- Labels -->
      <div>
        <h2 class="section-title">Labels</h2>
        <div v-if="labelEntries.length" class="space-y-2">
          <div v-for="[key, value] in labelEntries" :key="key" class="detail-value-mono">
            <span class="text-slate-500">{{ key }}</span><span class="text-slate-300 mx-1">=</span><span class="text-slate-700">{{ value }}</span>
          </div>
        </div>
        <div v-else class="detail-value text-slate-400 text-sm">无标签</div>
      </div>

      <!-- 层信息 -->
      <div>
        <h2 class="section-title section-title-table">
          层信息
          <span class="text-slate-400 normal-case font-normal ml-1">（{{ inspectData.layers }} 个实际层，共 {{ layerDetails.length }} 步）</span>
        </h2>
        <div v-if="layerDetails.length" class="border-x border-b border-slate-200 rounded-b-xl overflow-hidden">
          <table class="w-full">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="th-sm w-10">#</th>
                <th class="th-sm w-28">大小</th>
                <th class="th-sm">命令</th>
                <th class="th-sm w-36">Digest</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-100">
              <tr v-for="(layer, idx) in layerDetails" :key="idx" class="hover:bg-slate-50 transition-colors" :class="layer.empty ? 'opacity-50' : ''">
                <td class="px-3 py-2 text-center">
                  <span class="text-xs font-bold" :class="layer.empty ? 'text-slate-400' : 'text-slate-600'">{{ layerDetails.length - idx }}</span>
                </td>
                <td class="px-3 py-2 text-right">
                  <span v-if="!layer.empty" class="text-xs text-slate-500 tabular-nums">{{ formatFileSize(layer.size) }}</span>
                  <span v-else class="text-xs text-slate-400">空层</span>
                </td>
                <td class="px-3 py-2">
                  <code class="text-xs font-mono text-slate-700 truncate block max-w-xs">{{ layer.createdBy || '(无命令)' }}</code>
                </td>
                <td class="px-3 py-2 text-right">
                  <code v-if="!layer.empty && layer.digest" class="text-xs font-mono text-slate-400 truncate block" :title="layer.digest">{{ formatDigest(layer.digest) }}</code>
                  <span v-else class="text-xs text-slate-300">-</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="detail-value text-slate-400 text-sm">无层信息</div>
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
