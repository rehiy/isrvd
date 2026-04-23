<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import { FA_ICONS } from '@/helper/icons'
import type { FaIcon } from '@/helper/icons'

import Combobox from '@/component/combobox.vue'

interface IconGroup {
    label: string
    icons: FaIcon[]
}

const GROUPS: { label: string; names: string[] }[] = [
    { label: '通用', names: ['fa-link','fa-globe','fa-house','fa-home','fa-bookmark','fa-star','fa-heart','fa-flag','fa-tag','fa-tags','fa-bell','fa-envelope','fa-circle-info','fa-circle-question','fa-magnifying-glass','fa-rocket','fa-bolt','fa-crown','fa-wand-magic-sparkles'] },
    { label: '数据 & 图表', names: ['fa-chart-line','fa-chart-bar','fa-chart-pie','fa-chart-area','fa-gauge','fa-gauge-high','fa-table','fa-table-columns','fa-list-ol','fa-list-check','fa-database'] },
    { label: '基础设施', names: ['fa-server','fa-cloud','fa-cloud-arrow-up','fa-cloud-arrow-down','fa-network-wired','fa-sitemap','fa-layer-group','fa-microchip','fa-memory','fa-hard-drive','fa-temperature-half','fa-warehouse','fa-circle-nodes','fa-route','fa-ship','fa-cube','fa-cubes','fa-box','fa-compact-disc'] },
    { label: '开发 & 运维', names: ['fa-terminal','fa-code','fa-code-branch','fa-bug','fa-robot','fa-gear','fa-gears','fa-cogs','fa-wrench','fa-screwdriver-wrench','fa-hammer','fa-plug','fa-puzzle-piece','fa-rotate','fa-arrows-rotate','fa-play','fa-stop','fa-pause','fa-power-off','fa-fan','fa-tasks','fa-store'] },
    { label: '安全', names: ['fa-shield','fa-shield-halved','fa-key','fa-lock','fa-lock-open','fa-user-shield','fa-certificate'] },
    { label: '文件 & 文档', names: ['fa-file','fa-file-code','fa-file-lines','fa-file-alt','fa-file-import','fa-file-circle-plus','fa-file-zipper','fa-folder','fa-folder-open','fa-book','fa-book-open','fa-newspaper','fa-clipboard','fa-note-sticky','fa-download','fa-upload'] },
    { label: '用户 & 团队', names: ['fa-user','fa-user-tie','fa-user-tag','fa-users','fa-user-group','fa-address-card'] },
    { label: '品牌', names: ['fa-docker','fa-github','fa-gitlab','fa-git-alt','fa-linux','fa-python','fa-js','fa-node-js','fa-react','fa-vuejs','fa-aws','fa-google','fa-slack','fa-confluence','fa-jira','fa-jenkins','fa-grafana','fa-prometheus','fa-kubernetes','fa-markdown','fa-html5','fa-css3-alt','fa-php','fa-java'] },
]

// 按分组整理图标，保留顺序
const iconMap = new Map(FA_ICONS.map(i => [i.name, i]))
const ALL_GROUPS: IconGroup[] = GROUPS.map(g => ({
    label: g.label,
    icons: g.names.map(n => iconMap.get(n)).filter(Boolean) as FaIcon[],
}))

@Component({
    components: { Combobox },
    emits: ['update:modelValue']
})
class IconSelect extends Vue {
    @Prop({ type: String, default: '' }) readonly modelValue!: string
    @Prop({ type: String, default: '选择图标' }) readonly placeholder!: string

    get groups(): IconGroup[] {
        return ALL_GROUPS
    }

    filteredGroups(query: string): IconGroup[] {
        if (!query) return this.groups
        const q = query.toLowerCase()
        return this.groups
            .map(g => ({ ...g, icons: g.icons.filter(i => i.name.includes(q) || i.label.includes(q)) }))
            .filter(g => g.icons.length > 0)
    }
}

export default toNative(IconSelect)
</script>

<template>
  <Combobox :model-value="modelValue" :placeholder="placeholder" @update:model-value="$emit('update:modelValue', $event)">
    <template #default="{ query, select, isSelected }">
      <template v-for="group in filteredGroups(query)" :key="group.label">
        <!-- 分组标题（搜索时隐藏） -->
        <div v-if="!query" class="px-3 pt-2 pb-1 text-[10px] font-semibold text-slate-400 uppercase tracking-wider select-none">
          {{ group.label }}
        </div>
        <button
          v-for="icon in group.icons"
          :key="icon.name"
          type="button"
          class="w-full flex items-center gap-2 px-3 py-1.5 text-sm hover:bg-slate-50 transition-colors overflow-hidden"
          :class="isSelected(icon.name) ? 'text-primary-600 bg-primary-50' : 'text-slate-700'"
          @click="select(icon.name)"
        >
          <i :class="[icon.prefix ?? 'fas', icon.name, 'w-4 shrink-0 text-center']" :style="isSelected(icon.name) ? '' : 'color:#94a3b8'"></i>
          <span class="flex-1 text-left truncate">{{ icon.label }}</span>
          <code class="shrink-0 text-[10px] text-slate-400 truncate max-w-[110px]">{{ icon.name }}</code>
        </button>
      </template>
    </template>
  </Combobox>
</template>
