<script lang="ts">
import { Component, Prop, Vue, Watch, toNative } from 'vue-facing-decorator'

import Dropdown from '@/component/dropdown.vue'
import type { ImageInfo } from '@/service/types'

@Component({
    components: { Dropdown },
    emits: ['update:modelValue']
})
class ImageSelect extends Vue {
    @Prop({ type: String, default: '' }) readonly modelValue!: string
    @Prop({ type: Array, default: () => [] }) readonly images!: ImageInfo[]
    @Prop({ type: String, default: '选择或输入镜像名称' }) readonly placeholder!: string
    @Prop({ type: Boolean, default: false }) readonly disabled!: boolean

    // ─── 数据属性 ───
    dropdownOpen = false
    searchQuery = ''
    justSelected = false

    // ─── 生命周期 ───
    mounted() {
        this.searchQuery = this.modelValue
    }

    // ─── 监听器 ───
    @Watch('modelValue')
    onModelValueChange(val: string) {
        if (val !== this.searchQuery) this.searchQuery = val
    }

    @Watch('dropdownOpen')
    onDropdownOpenChange(open: boolean) {
        if (open) {
            this.searchQuery = ''
        } else if (!this.justSelected) {
            this.searchQuery = this.modelValue
        }
        this.justSelected = false
    }

    // ─── 常量 ───
    readonly colorPalette = [
        '#0ea5e9', '#6366f1', '#10b981', '#f59e0b', '#ef4444',
        '#8b5cf6', '#ec4899', '#14b8a6', '#f97316', '#06b6d4',
        '#84cc16', '#e11d48', '#a855f7', '#3b82f6', '#d946ef'
    ]

    // ─── 方法 ───
    stringToIndex(str: string) {
        let hash = 0
        for (let i = 0; i < str.length; i++) hash = ((hash << 5) - hash + str.charCodeAt(i)) | 0
        return Math.abs(hash) % this.colorPalette.length
    }

    getDomainColor(domain: string) { return this.colorPalette[this.stringToIndex(domain)] }

    getDomain(repoTag: string) {
        if (!repoTag) return '本地镜像'
        const parts = repoTag.split('/')
        if (parts.length >= 3) return parts[0]
        if (parts.length === 2 && parts[0].includes('.')) return parts[0]
        return 'docker.io'
    }

    getDomainIcon(domain: string) {
        if (domain === 'docker.io') return 'fa-docker fab'
        if (domain.includes('aliyuncs.com')) return 'fa-cloud fas'
        if (domain.includes('tencentyun.com')) return 'fa-cloud fas'
        if (domain === 'ghcr.io') return 'fa-github fab'
        if (domain === 'gcr.io') return 'fa-google fab'
        if (domain === 'quay.io') return 'fa-ship fas'
        if (domain === '本地镜像') return 'fa-home fas'
        return 'fa-server fas'
    }

    getDomainLabel(domain: string) {
        if (domain === 'docker.io') return 'Docker Hub'
        if (domain.includes('aliyuncs.com')) return '阿里云'
        if (domain.includes('tencentyun.com')) return '腾讯云'
        if (domain === 'ghcr.io') return 'GitHub'
        if (domain === 'gcr.io') return 'Google'
        if (domain === 'quay.io') return 'Quay'
        if (domain === '本地镜像') return '本地镜像'
        return domain
    }

    get filteredImages() {
        if (!this.searchQuery.trim()) return this.images
        const query = this.searchQuery.toLowerCase()
        return this.images.filter((img: ImageInfo) =>
            img.repoTags && img.repoTags.some((tag: string) => tag.toLowerCase().includes(query))
        )
    }

    get groupedImages() {
        const groups: Record<string, { domain: string; images: ImageInfo[] }> = {}
        for (const img of this.filteredImages) {
            const tag = img.repoTags?.[0] || ''
            const domain = this.getDomain(tag)
            if (!groups[domain]) groups[domain] = { domain, images: [] }
            groups[domain].images.push(img)
        }
        return Object.values(groups).sort((a, b) => {
            if (a.domain === 'docker.io') return -1
            if (b.domain === 'docker.io') return 1
            if (a.domain === '本地镜像') return 1
            if (b.domain === '本地镜像') return -1
            return a.domain.localeCompare(b.domain)
        })
    }

    formatSize(bytes: number) {
        if (!bytes) return ''
        if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
        if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
        return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB'
    }

    selectImage(imageName: string) {
        this.$emit('update:modelValue', imageName)
        this.searchQuery = imageName
        this.justSelected = true
        this.dropdownOpen = false
    }

    handleInput() {
        if (!this.dropdownOpen) this.dropdownOpen = true
    }
}

export default toNative(ImageSelect)
</script>

<template>
  <Dropdown v-model:open="dropdownOpen" max-height="360px">
    <!-- 触发区域：输入框 -->
    <template #trigger="{ open }">
      <div
        class="input min-h-[42px] !px-3 !py-2 cursor-text flex items-center gap-2"
        :class="open ? '!border-primary-400' : ''"
        @click="inputRef?.focus(); dropdownOpen = true"
      >
        <input
          ref="inputRef"
          v-model="searchQuery"
          type="text"
          class="flex-1 min-w-[80px] border-0 outline-none bg-transparent text-sm text-slate-700 placeholder:text-slate-400 p-0 focus:ring-0 focus:border-0 focus:shadow-none"
          :placeholder="placeholder"
          :disabled="disabled"
          @focus="dropdownOpen = true"
          @input="handleInput"
          @keydown.enter.prevent="justSelected = true; selectImage(searchQuery.trim())"
        />
        <i :class="['fas fa-chevron-down text-slate-400 text-xs transition-transform duration-200', open ? 'rotate-180' : '']"></i>
      </div>
    </template>

    <!-- 搜索提示 -->
    <template #search-hint>
      <div v-if="searchQuery.trim()" class="px-3 py-2 bg-slate-50 border-b border-slate-100 flex items-center justify-between">
        <span class="text-xs text-slate-500">搜索: <code class="bg-slate-200 px-1.5 py-0.5 rounded text-slate-700">{{ searchQuery.trim() }}</code></span>
        <span class="text-xs text-slate-400">{{ filteredImages.length }} 个匹配</span>
      </div>
    </template>

    <!-- 分组列表 -->
    <template #default>
      <div v-for="group in groupedImages" :key="group.domain" class="border-b border-slate-100 last:border-0">
        <div class="px-3 py-2 bg-slate-50/80 flex items-center gap-2 sticky top-0 z-10">
          <i :class="['text-xs', getDomainIcon(group.domain)]" :style="{ color: getDomainColor(group.domain) }"></i>
          <span class="text-xs font-semibold text-slate-600">{{ getDomainLabel(group.domain) }}</span>
          <span class="text-xs text-slate-400">{{ group.images.length }}</span>
        </div>
        <div class="px-2 py-1.5 grid grid-cols-1 gap-0.5">
          <button
            v-for="img in group.images"
            :key="img.id + '-' + img.repoTags[0]"
            type="button"
            @click="selectImage(img.repoTags[0])"
            :class="[
              'w-full flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-left transition-all duration-150 group',
              modelValue === img.repoTags[0]
                ? 'bg-primary-50 border border-primary-200'
                : 'hover:bg-slate-50 border border-transparent'
            ]"
          >
            <div
              class="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0"
              :style="{ background: getDomainColor(group.domain) + '15' }"
            >
              <i :class="['text-xs', getDomainIcon(group.domain)]" :style="{ color: getDomainColor(group.domain) }"></i>
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

    <!-- 空状态 -->
    <template #empty>
      <div v-if="groupedImages.length === 0 && images.length > 0" class="py-8 text-center">
        <i class="fas fa-search text-slate-300 text-2xl mb-2"></i>
        <p class="text-sm text-slate-400">无匹配镜像</p>
      </div>
    </template>

    <!-- 底部统计 -->
    <template #footer>
      <div v-if="images.length > 0" class="px-3 py-2 bg-slate-50 border-t border-slate-100 flex items-center justify-between">
        <span class="text-xs text-slate-400">
          共 <strong class="text-slate-700">{{ images.length }}</strong> 个镜像
        </span>
      </div>
    </template>
  </Dropdown>
</template>
