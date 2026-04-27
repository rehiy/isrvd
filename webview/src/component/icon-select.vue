<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'
import iconFamilies from '@fortawesome/fontawesome-free/metadata/icon-families.json'
import iconCategoriesYml from '@fortawesome/fontawesome-free/metadata/categories.yml?raw'
import { load as yamlLoad } from 'js-yaml'

import BaseModal from '@/component/modal.vue'

interface FaIcon {
    name: string
    label: string
    category: string
}

interface IconCategory {
    key: string
    label: string
    count: number
}

const { ALL_ICONS, CATEGORIES } = (() => {
    // 构建图标名→分类映射
    const iconCategoryMap = new Map<string, string>()
    const categoryEntries: IconCategory[] = []
    const iconCategories = yamlLoad(iconCategoriesYml) as Record<string, { label: string; icons: string[] }>

    for (const [key, info] of Object.entries(iconCategories)) {
        let count = 0
        for (const icon of info.icons) {
            if (!iconCategoryMap.has(icon)) {
                iconCategoryMap.set(icon, key)
                count++
            }
        }
        if (count > 0) {
            categoryEntries.push({ key, label: info.label, count })
        }
    }

    // 构建全部图标列表
    const icons: FaIcon[] = []
    for (const [name, info] of Object.entries(iconFamilies as Record<string, any>)) {
        for (const s of info.familyStylesByLicense?.free ?? []) {
            if (s.style === 'solid') {
                icons.push({ name: `fas fa-${name}`, label: info.label, category: iconCategoryMap.get(name) ?? '' })
            } else if (s.style === 'brands') {
                icons.push({ name: `fab fa-${name}`, label: info.label, category: iconCategoryMap.get(name) ?? '' })
            }
        }
    }

    return { ALL_ICONS: icons, CATEGORIES: categoryEntries }
})()

@Component({
    components: { BaseModal },
    emits: ['update:modelValue']
})
class IconSelect extends Vue {
    @Prop({ type: String, default: '' }) readonly modelValue!: string
    @Prop({ type: String, default: '选择图标' }) readonly placeholder!: string

    isOpen = false
    searchQuery = ''
    activeCategory = ''
    categoryExpanded = false

    get categories(): IconCategory[] {
        return CATEGORIES
    }

    get visibleCategories(): IconCategory[] {
        return this.categoryExpanded ? CATEGORIES : CATEGORIES.slice(0, 7)
    }

    get hasMoreCategories(): boolean {
        return CATEGORIES.length > 7
    }

    get filteredIcons(): FaIcon[] {
        const q = this.searchQuery.trim().toLowerCase()
        let result = ALL_ICONS

        // 搜索时忽略分类，全局搜索
        if (q) {
            return result.filter(i => i.name.includes(q) || i.label.toLowerCase().includes(q))
        }

        // 按分类筛选
        if (this.activeCategory) {
            result = result.filter(i => i.category === this.activeCategory)
        }

        return result
    }

    openModal() {
        this.searchQuery = ''
        this.activeCategory = ''
        this.categoryExpanded = false
        this.isOpen = true
    }

    selectIcon(icon: FaIcon) {
        this.$emit('update:modelValue', icon.name)
        this.isOpen = false
    }

    setCategory(key: string) {
        if (this.activeCategory === key) {
            this.activeCategory = ''
        } else {
            this.activeCategory = key
        }
    }
}

export default toNative(IconSelect)
</script>

<template>
  <div>
    <!-- 触发按钮 -->
    <div
      class="input flex items-center gap-2 cursor-pointer hover:border-primary-400 transition-colors min-w-0 w-full min-h-[46px]"
      :title="modelValue || placeholder"
      @click="openModal"
    >
      <i :class="[modelValue || 'fas fa-icons', 'w-4 flex-shrink-0 text-center', modelValue ? 'text-slate-500' : 'text-slate-300']"></i>
      <span class="truncate text-sm flex-1 text-left" :class="modelValue ? 'text-slate-600' : 'text-slate-400'">
        {{ modelValue || placeholder }}
      </span>
      <i class="fas fa-chevron-right text-slate-300 text-xs flex-shrink-0"></i>
    </div>

    <!-- 图标选择 Modal -->
    <BaseModal v-model="isOpen" title="选择图标" :show-footer="false">
      <!-- 搜索框 -->
      <div class="mb-3">
        <div class="relative">
          <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
            <i class="fas fa-search text-slate-400 text-sm"></i>
          </div>
          <input
            v-model="searchQuery"
            type="text"
            class="input pl-9"
            placeholder="搜索图标名称..."
            autofocus
          />
          <div class="absolute inset-y-0 right-0 pr-3 flex items-center pointer-events-none">
            <span class="text-xs text-slate-400">{{ filteredIcons.length }} 个</span>
          </div>
        </div>
      </div>

      <!-- 分类标签 -->
      <div class="mb-3" v-if="!searchQuery">
        <div class="flex flex-wrap gap-1.5">
          <button
            type="button"
            class="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium transition-colors"
            :class="!activeCategory
              ? 'bg-primary-100 text-primary-700 ring-1 ring-primary-300'
              : 'bg-slate-100 text-slate-500 hover:bg-slate-200'"
            @click="activeCategory = ''"
          >
            全部
          </button>
          <button
            v-for="cat in visibleCategories"
            :key="cat.key"
            type="button"
            class="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium transition-colors"
            :class="activeCategory === cat.key
              ? 'bg-primary-100 text-primary-700 ring-1 ring-primary-300'
              : 'bg-slate-100 text-slate-500 hover:bg-slate-200'"
            @click="setCategory(cat.key)"
          >
            {{ cat.label }}
          </button>
          <button
            v-if="hasMoreCategories"
            type="button"
            class="inline-flex items-center gap-0.5 px-2 py-0.5 rounded-full text-xs text-primary-500 hover:text-primary-600 hover:bg-primary-50 transition-colors"
            @click="categoryExpanded = !categoryExpanded"
          >
            <i :class="categoryExpanded ? 'fas fa-chevron-up' : 'fas fa-chevron-down'" class="text-[10px]"></i>
            {{ categoryExpanded ? '收起' : '更多' }}
          </button>
        </div>
      </div>

      <!-- 图标网格 -->
      <div class="grid grid-cols-6 gap-1 sm:grid-cols-8 md:grid-cols-10">
        <button
          v-for="icon in filteredIcons"
          :key="icon.name"
          type="button"
          :title="icon.label + '\n' + icon.name"
          class="w-16 flex flex-col items-center justify-center gap-1.5 p-2.5 rounded-lg transition-all overflow-hidden"
          :class="modelValue === icon.name
            ? 'bg-primary-50 text-primary-600 ring-1 ring-primary-300'
            : 'text-slate-500 hover:bg-slate-100 hover:text-slate-700'"
          @click="selectIcon(icon)"
        >
          <i :class="[icon.name, 'text-xl leading-none']"></i>
          <span class="truncate w-full text-center text-[10px] leading-tight text-slate-400">{{ icon.label }}</span>
        </button>
      </div>

      <!-- 空状态 -->
      <div v-if="filteredIcons.length === 0" class="py-12 text-center text-slate-400">
        <i class="fas fa-search text-3xl mb-3 block opacity-30"></i>
        <p class="text-sm">未找到匹配的图标</p>
      </div>
    </BaseModal>
  </div>
</template>
