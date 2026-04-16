<script setup>
import { computed, ref, watch } from 'vue'

import Dropdown from '@/component/dropdown.vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  images: { type: Array, default: () => [] },
  placeholder: { type: String, default: '选择或输入镜像名称' },
  disabled: { type: Boolean, default: false },
})

const emit = defineEmits(['update:modelValue'])

const dropdownOpen = ref(false)
const searchQuery = ref(props.modelValue)
const inputRef = ref(null)
const justSelected = ref(false)

// 当外部设置 modelValue 时（如编辑容器），同步到输入框
watch(() => props.modelValue, (val) => {
  if (val !== searchQuery.value) {
    searchQuery.value = val
  }
})

// 打开下拉时清空搜索词，显示全部镜像；取消关闭时恢复显示
watch(dropdownOpen, (open) => {
  if (open) {
    searchQuery.value = ''
  } else if (!justSelected.value) {
    searchQuery.value = props.modelValue
  }
  justSelected.value = false
})

// Tailwind 标准色 500 色阶，与项目 badge/button 色彩体系一致
const colorPalette = [
  '#0ea5e9', // sky-500    (与项目 primary 一致)
  '#6366f1', // indigo-500
  '#10b981', // emerald-500
  '#f59e0b', // amber-500
  '#ef4444', // red-500
  '#8b5cf6', // violet-500
  '#ec4899', // pink-500
  '#14b8a6', // teal-500
  '#f97316', // orange-500
  '#06b6d4', // cyan-500
  '#84cc16', // lime-500
  '#e11d48', // rose-500
  '#a855f7', // purple-500
  '#3b82f6', // blue-500
  '#d946ef', // fuchsia-500
]

// 根据字符串生成稳定的索引（同一域名始终映射同一颜色）
const stringToIndex = (str) => {
  let hash = 0
  for (let i = 0; i < str.length; i++) {
    hash = ((hash << 5) - hash + str.charCodeAt(i)) | 0
  }
  return Math.abs(hash) % colorPalette.length
}

// 获取域名颜色
const getDomainColor = (domain) => colorPalette[stringToIndex(domain)]

// 提取镜像域名
const getDomain = (repoTag) => {
  if (!repoTag) return '本地镜像'
  const parts = repoTag.split('/')
  if (parts.length >= 3) return parts[0]
  if (parts.length === 2 && parts[0].includes('.')) return parts[0]
  return 'docker.io'
}

// 获取域名图标
const getDomainIcon = (domain) => {
  if (domain === 'docker.io') return 'fa-docker fab'
  if (domain.includes('aliyuncs.com')) return 'fa-cloud fas'
  if (domain.includes('tencentyun.com')) return 'fa-cloud fas'
  if (domain === 'ghcr.io') return 'fa-github fab'
  if (domain === 'gcr.io') return 'fa-google fab'
  if (domain === 'quay.io') return 'fa-ship fas'
  if (domain === '本地镜像') return 'fa-home fas'
  return 'fa-server fas'
}

// 获取域名显示名称
const getDomainLabel = (domain) => {
  if (domain === 'docker.io') return 'Docker Hub'
  if (domain.includes('aliyuncs.com')) return '阿里云'
  if (domain.includes('tencentyun.com')) return '腾讯云'
  if (domain === 'ghcr.io') return 'GitHub'
  if (domain === 'gcr.io') return 'Google'
  if (domain === 'quay.io') return 'Quay'
  if (domain === '本地镜像') return '本地镜像'
  return domain
}

// 搜索过滤
const filteredImages = computed(() => {
  if (!searchQuery.value.trim()) return props.images
  const query = searchQuery.value.toLowerCase()
  return props.images.filter(img =>
    img.repoTags && img.repoTags.some(tag => tag.toLowerCase().includes(query))
  )
})

// 按域名分组的镜像
const groupedImages = computed(() => {
  const groups = {}
  const images = filteredImages.value
  for (const img of images) {
    const tag = img.repoTags?.[0] || ''
    const domain = getDomain(tag)
    if (!groups[domain]) {
      groups[domain] = { domain, images: [] }
    }
    groups[domain].images.push(img)
  }
  const sorted = Object.values(groups).sort((a, b) => {
    if (a.domain === 'docker.io') return -1
    if (b.domain === 'docker.io') return 1
    if (a.domain === '本地镜像') return 1
    if (b.domain === '本地镜像') return -1
    return a.domain.localeCompare(b.domain)
  })
  return sorted
})

// 格式化镜像大小
const formatSize = (bytes) => {
  if (!bytes) return ''
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB'
}

// 选择镜像
const selectImage = (imageName) => {
  emit('update:modelValue', imageName)
  searchQuery.value = imageName
  justSelected.value = true
  dropdownOpen.value = false
}

// 处理输入 - 仅用于搜索，不更新 modelValue
const handleInput = () => {
  if (!dropdownOpen.value) dropdownOpen.value = true
}
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
