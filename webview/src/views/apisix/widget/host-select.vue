<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import type { DockerContainerInfo } from '@/service/types'

import Combobox from '@/component/combobox.vue'

interface HostGroup {
    network: string
    containers: DockerContainerInfo[]
}

@Component({
    components: { Combobox },
    emits: ['update:modelValue']
})
class HostSelect extends Vue {
    @Prop({ type: String, default: '' }) readonly modelValue!: string
    @Prop({ type: Array, default: () => [] }) readonly containers!: DockerContainerInfo[]
    @Prop({ type: String, default: '127.0.0.1 或 容器名' }) readonly placeholder!: string
    @Prop({ type: Boolean, default: false }) readonly disabled!: boolean

    // ─── 常量 ───
    readonly colorPalette = [
        '#0ea5e9', '#6366f1', '#10b981', '#f59e0b', '#ef4444',
        '#8b5cf6', '#ec4899', '#14b8a6', '#f97316', '#06b6d4',
        '#84cc16', '#e11d48', '#a855f7', '#3b82f6', '#d946ef'
    ]

    // ─── 方法 ───
    networkColor(network: string) {
        let hash = 0
        for (let i = 0; i < network.length; i++) hash = ((hash << 5) - hash + network.charCodeAt(i)) | 0
        return this.colorPalette[Math.abs(hash) % this.colorPalette.length]
    }

    stateColor(state: string) {
        if (state === 'running') return 'text-emerald-500'
        if (state === 'paused') return 'text-amber-500'
        return 'text-slate-400'
    }

    groupedBy(query: string): HostGroup[] {
        const list = query
            ? this.containers.filter(c => c.name.toLowerCase().includes(query))
            : this.containers
        const groups: Record<string, HostGroup> = {}
        for (const c of list) {
            const nets = c.networks?.length ? c.networks : ['默认']
            for (const net of nets) {
                (groups[net] ||= { network: net, containers: [] }).containers.push(c)
            }
        }
        return Object.values(groups).sort((a, b) => {
            if (a.network === '默认') return 1
            if (b.network === '默认') return -1
            return a.network.localeCompare(b.network)
        })
    }

    matchCount(query: string) {
        return query ? this.containers.filter(c => c.name.toLowerCase().includes(query)).length : this.containers.length
    }
}

export default toNative(HostSelect)
</script>

<template>
  <Combobox
    :model-value="modelValue"
    :placeholder="placeholder"
    :disabled="disabled"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <template #hint-extra="{ query }">
      <span class="text-xs text-slate-400">{{ matchCount(query.toLowerCase()) }} 个匹配</span>
    </template>

    <template #default="{ query, select }">
      <div v-for="group in groupedBy(query)" :key="group.network" class="border-b border-slate-100 last:border-0">
        <div class="px-3 py-2 bg-slate-50/80 flex items-center gap-2 sticky top-0 z-10">
          <i class="fas fa-network-wired text-xs" :style="{ color: networkColor(group.network) }"></i>
          <span class="text-xs font-semibold text-slate-600">{{ group.network }}</span>
          <span class="text-xs text-slate-400">{{ group.containers.length }}</span>
        </div>
        <div class="px-2 py-1.5 grid grid-cols-1 gap-0.5">
          <button
            v-for="c in group.containers"
            :key="group.network + '-' + c.id"
            type="button"
            @click="select(c.name)"
            :class="[
              'w-full flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-left transition-all duration-150',
              modelValue === c.name ? 'bg-primary-50 border border-primary-200' : 'hover:bg-slate-50 border border-transparent'
            ]"
          >
            <div class="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0" :style="{ background: networkColor(group.network) + '15' }">
              <i class="fas fa-cube text-xs" :style="{ color: networkColor(group.network) }"></i>
            </div>
            <div class="flex-1 min-w-0">
              <div class="text-sm font-medium text-slate-700 truncate">{{ c.name }}</div>
              <div class="text-xs text-slate-400 truncate">
                <i :class="['fas fa-circle text-[6px] mr-1', stateColor(c.state)]"></i>
                {{ c.state }} · {{ c.id }}
              </div>
            </div>
            <i v-if="modelValue === c.name" class="fas fa-check text-primary-500 text-xs"></i>
          </button>
        </div>
      </div>
    </template>

    <template #empty="{ query }">
      <div v-if="groupedBy(query.toLowerCase()).length === 0" class="py-8 text-center">
        <i class="fas fa-search text-slate-300 text-2xl mb-2"></i>
        <p class="text-sm text-slate-400">{{ containers.length === 0 ? '无可用容器' : '无匹配容器' }}</p>
      </div>
    </template>

    <template #footer>
      <div v-if="containers.length > 0" class="px-3 py-2 bg-slate-50 border-t border-slate-100 flex items-center justify-between">
        <span class="text-xs text-slate-400">共 <strong class="text-slate-700">{{ containers.length }}</strong> 个运行中容器</span>
        <span class="text-xs text-slate-400">回车使用输入值</span>
      </div>
    </template>
  </Combobox>
</template>
