<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import Combobox from '@/component/combobox.vue'

interface PortItem {
    port: string
    proto: string
    published?: string
}

@Component({
    components: { Combobox },
    emits: ['update:modelValue']
})
class PortSelect extends Vue {
    @Prop({ type: String, default: '' }) readonly modelValue!: string
    @Prop({ type: Array, default: () => [] }) readonly ports!: string[]
    @Prop({ type: String, default: '8080' }) readonly placeholder!: string
    @Prop({ type: Boolean, default: false }) readonly disabled!: boolean

    // ─── 方法 ───
    // 解析 formatPorts 输出，形如 "80/tcp"、"8080:80/tcp"、"0.0.0.0:8080:80/tcp"
    parsePort(raw: string): PortItem | null {
        const [body, proto = 'tcp'] = raw.split('/')
        if (!body) return null
        const parts = body.split(':')
        const port = parts.pop()
        if (!port || !/^\d+$/.test(port)) return null
        const published = parts.pop()
        return { port, proto, published: published && /^\d+$/.test(published) ? published : undefined }
    }

    // ─── 计算属性 ───
    get parsedPorts(): PortItem[] {
        const seen = new Set<string>()
        const result: PortItem[] = []
        for (const raw of this.ports) {
            const item = this.parsePort(raw)
            if (!item) continue
            const key = item.port + '/' + item.proto
            if (seen.has(key)) continue
            seen.add(key)
            result.push(item)
        }
        return result.sort((a, b) => Number(a.port) - Number(b.port))
    }

    // ─── 方法 ───
    filtered(query: string) {
        if (!query) return this.parsedPorts
        return this.parsedPorts.filter(p => p.port.includes(query) || p.proto.includes(query))
    }

    protoColor(proto: string) {
        if (proto === 'tcp') return 'text-sky-500'
        if (proto === 'udp') return 'text-amber-500'
        return 'text-slate-400'
    }

    protoBg(proto: string) {
        if (proto === 'tcp') return 'bg-sky-50'
        if (proto === 'udp') return 'bg-amber-50'
        return 'bg-slate-50'
    }
}

export default toNative(PortSelect)
</script>

<template>
  <!-- 无端口数据时直接用输入框，避免空下拉面板 -->
  <input
    v-if="parsedPorts.length === 0"
    type="text"
    class="input"
    :value="modelValue"
    :placeholder="placeholder"
    :disabled="disabled"
    @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
  />
  <!-- 有端口数据时用 Combobox 下拉选择 -->
  <Combobox
    v-else
    :model-value="modelValue"
    :placeholder="placeholder"
    :disabled="disabled"
    max-height="220px"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <template #hint-extra="{ query }">
      <span class="text-xs text-slate-400">{{ filtered(query).length }} 个匹配</span>
    </template>

    <template #default="{ query, select }">
      <div v-if="filtered(query).length > 0" class="px-1.5 py-1 grid grid-cols-1 gap-0.5">
        <button
          v-for="item in filtered(query)"
          :key="item.port + '-' + item.proto"
          type="button"
          @click="select(item.port)"
          :class="[
            'w-full flex items-center gap-2 px-2 py-1.5 rounded-lg text-left transition-all duration-150',
            modelValue === item.port ? 'bg-primary-50 border border-primary-200' : 'hover:bg-slate-50 border border-transparent'
          ]"
        >
          <span class="inline-flex items-center justify-center px-1.5 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider flex-shrink-0" :class="[protoBg(item.proto), protoColor(item.proto)]">{{ item.proto }}</span>
          <span class="flex-1 min-w-0 text-sm font-medium text-slate-700">{{ item.port }}</span>
          <span v-if="item.published" class="text-xs text-slate-400 truncate">→{{ item.published }}</span>
          <span class="w-4 flex-shrink-0 flex items-center justify-center"><i v-if="modelValue === item.port" class="fas fa-check text-primary-500 text-xs"></i></span>
        </button>
      </div>
    </template>

    <template #empty="{ query }">
      <div v-if="filtered(query).length === 0" class="py-4 text-center">
        <p class="text-xs text-slate-400">无匹配端口</p>
      </div>
    </template>

    <template #footer>
      <div class="px-3 py-1.5 bg-slate-50 border-t border-slate-100 flex items-center justify-between">
        <span class="text-xs text-slate-400">共 <strong class="text-slate-700">{{ parsedPorts.length }}</strong> 个端口</span>
        <span class="text-xs text-slate-400">回车确认</span>
      </div>
    </template>
  </Combobox>
</template>
