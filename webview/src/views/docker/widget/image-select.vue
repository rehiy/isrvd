<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import Combobox from '@/component/combobox.vue'
import type { DockerImageInfo } from '@/service/types'

interface ImageGroup {
    domain: string
    images: DockerImageInfo[]
}

@Component({
    components: { Combobox },
    emits: ['update:modelValue']
})
class ImageSelect extends Vue {
    @Prop({ type: String, default: '' }) readonly modelValue!: string
    @Prop({ type: Array, default: () => [] }) readonly images!: DockerImageInfo[]
    @Prop({ type: String, default: '选择或输入镜像名称' }) readonly placeholder!: string
    @Prop({ type: Boolean, default: false }) readonly disabled!: boolean

    // ─── 常量 ───
    readonly colorPalette = [
        '#0ea5e9', '#6366f1', '#10b981', '#f59e0b', '#ef4444',
        '#8b5cf6', '#ec4899', '#14b8a6', '#f97316', '#06b6d4',
        '#84cc16', '#e11d48', '#a855f7', '#3b82f6', '#d946ef'
    ]

    // ─── 方法 ───
    domainColor(domain: string) {
        let hash = 0
        for (let i = 0; i < domain.length; i++) hash = ((hash << 5) - hash + domain.charCodeAt(i)) | 0
        return this.colorPalette[Math.abs(hash) % this.colorPalette.length]
    }

    domainOf(repoTag: string) {
        if (!repoTag) return '本地镜像'
        const parts = repoTag.split('/')
        if (parts.length >= 3) return parts[0]
        if (parts.length === 2 && parts[0].includes('.')) return parts[0]
        return 'docker.io'
    }

    domainIcon(domain: string) {
        if (domain === 'docker.io') return 'fa-docker fab'
        if (domain.includes('aliyuncs.com')) return 'fa-cloud fas'
        if (domain.includes('tencentyun.com')) return 'fa-cloud fas'
        if (domain === 'ghcr.io') return 'fa-github fab'
        if (domain === 'gcr.io') return 'fa-google fab'
        if (domain === 'quay.io') return 'fa-ship fas'
        if (domain === '本地镜像') return 'fa-home fas'
        return 'fa-server fas'
    }

    domainLabel(domain: string) {
        if (domain === 'docker.io') return 'Docker Hub'
        if (domain.includes('aliyuncs.com')) return '阿里云'
        if (domain.includes('tencentyun.com')) return '腾讯云'
        if (domain === 'ghcr.io') return 'GitHub'
        if (domain === 'gcr.io') return 'Google'
        if (domain === 'quay.io') return 'Quay'
        return domain
    }

    formatSize(bytes: number) {
        if (!bytes) return ''
        if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
        if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
        return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB'
    }

    groupedBy(query: string): ImageGroup[] {
        const list = query
            ? this.images.filter(img => img.repoTags?.some(t => t.toLowerCase().includes(query)))
            : this.images
        const groups: Record<string, ImageGroup> = {}
        for (const img of list) {
            const domain = this.domainOf(img.repoTags?.[0] || '')
            ;(groups[domain] ||= { domain, images: [] }).images.push(img)
        }
        return Object.values(groups).sort((a, b) => {
            if (a.domain === 'docker.io') return -1
            if (b.domain === 'docker.io') return 1
            if (a.domain === '本地镜像') return 1
            if (b.domain === '本地镜像') return -1
            return a.domain.localeCompare(b.domain)
        })
    }

    matchCount(query: string) {
        if (!query) return this.images.length
        return this.images.filter(img => img.repoTags?.some(t => t.toLowerCase().includes(query))).length
    }
}

export default toNative(ImageSelect)
</script>

<template>
  <Combobox
    :model-value="modelValue"
    :placeholder="placeholder"
    :disabled="disabled"
    max-height="360px"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <template #hint-extra="{ query }">
      <span class="text-xs text-slate-400">{{ matchCount(query.toLowerCase()) }} 个匹配</span>
    </template>

    <template #default="{ query, select }">
      <div v-for="group in groupedBy(query)" :key="group.domain" class="border-b border-slate-100 last:border-0">
        <div class="px-3 py-2 bg-slate-50/80 flex items-center gap-2 sticky top-0 z-10">
          <i :class="['text-xs', domainIcon(group.domain)]" :style="{ color: domainColor(group.domain) }"></i>
          <span class="text-xs font-semibold text-slate-600">{{ domainLabel(group.domain) }}</span>
          <span class="text-xs text-slate-400">{{ group.images.length }}</span>
        </div>
        <div class="px-2 py-1.5 grid grid-cols-1 gap-0.5">
          <button
            v-for="img in group.images"
            :key="img.id + '-' + img.repoTags[0]"
            type="button"
            @click="select(img.repoTags[0])"
            :class="[
              'w-full flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-left transition-all duration-150',
              modelValue === img.repoTags[0] ? 'bg-primary-50 border border-primary-200' : 'hover:bg-slate-50 border border-transparent'
            ]"
          >
            <div class="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0" :style="{ background: domainColor(group.domain) + '15' }">
              <i :class="['text-xs', domainIcon(group.domain)]" :style="{ color: domainColor(group.domain) }"></i>
            </div>
            <div class="flex-1 min-w-0">
              <div class="text-sm font-medium text-slate-700 truncate">{{ img.repoTags[0] }}</div>
              <div class="text-xs text-slate-400">{{ img.shortId }} · {{ formatSize(img.size) }}</div>
            </div>
            <i v-if="modelValue === img.repoTags[0]" class="fas fa-check text-primary-500 text-xs"></i>
          </button>
        </div>
      </div>
    </template>

    <template #empty="{ query }">
      <div v-if="groupedBy(query.toLowerCase()).length === 0" class="py-8 text-center">
        <i class="fas fa-search text-slate-300 text-2xl mb-2"></i>
        <p class="text-sm text-slate-400">{{ images.length === 0 ? '无可用镜像' : '无匹配镜像' }}</p>
      </div>
    </template>

    <template #footer>
      <div v-if="images.length > 0" class="px-3 py-2 bg-slate-50 border-t border-slate-100 flex items-center justify-between">
        <span class="text-xs text-slate-400">共 <strong class="text-slate-700">{{ images.length }}</strong> 个镜像</span>
        <span class="text-xs text-slate-400">回车使用输入值</span>
      </div>
    </template>
  </Combobox>
</template>
