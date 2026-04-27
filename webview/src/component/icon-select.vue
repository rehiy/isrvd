<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'
import iconFamilies from '@fortawesome/fontawesome-free/metadata/icon-families.json'

import BaseModal from '@/component/modal.vue'

interface FaIcon {
    name: string
    label: string
}

const ALL_ICONS: FaIcon[] = (() => {
    const result: FaIcon[] = []
    for (const [name, info] of Object.entries(iconFamilies as Record<string, any>)) {
        for (const s of info.familyStylesByLicense?.free ?? []) {
            if (s.style === 'solid') result.push({ name: `fas fa-${name}`, label: info.label })
            else if (s.style === 'brands') result.push({ name: `fab fa-${name}`, label: info.label })
        }
    }
    return result
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

    get filteredIcons(): FaIcon[] {
        const q = this.searchQuery.trim().toLowerCase()
        if (!q) return ALL_ICONS
        return ALL_ICONS.filter(i => i.name.includes(q) || i.label.toLowerCase().includes(q))
    }

    openModal() {
        this.searchQuery = ''
        this.isOpen = true
    }

    selectIcon(icon: FaIcon) {
        this.$emit('update:modelValue', icon.name)
        this.isOpen = false
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
      <div class="mb-4">
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
