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
}

export default toNative(PortSelect)
</script>

<template>
  <Combobox
    :model-value="modelValue"
    :placeholder="placeholder"
    :disabled="disabled"
    max-height="280px"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <template #hint-extra="{ query }">
      <span class="text-xs text-slate-400">{{ filtered(query).length }} 个匹配</span>
    </template>

    <template #default="{ query, select }">
      <div v-if="filtered(query).length > 0" class="px-2 py-1.5 grid grid-cols-1 gap-0.5">
        <button
          v-for="item in filtered(query)"
          :key="item.port + '-' + item.proto"
          type="button"
          @click="select(item.port)"
          :class="[
            'w-full flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-left transition-all duration-150',
            modelValue === item.port ? 'bg-primary-50 border border-primary-200' : 'hover:bg-slate-50 border border-transparent'
          ]"
        >
          <div class="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0 bg-slate-100">
            <i class="fas fa-plug text-xs" :class="protoColor(item.proto)"></i>
          </div>
          <div class="flex-1 min-w-0">
            <div class="text-sm font-medium text-slate-700">
              {{ item.port }}<span class="text-xs text-slate-400 ml-1">/{{ item.proto }}</span>
            </div>
            <div class="text-xs text-slate-400 truncate">
              <template v-if="item.published">宿主: {{ item.published }}</template>
              <template v-else>仅容器内暴露</template>
            </div>
          </div>
          <i v-if="modelValue === item.port" class="fas fa-check text-primary-500 text-xs"></i>
        </button>
      </div>
    </template>

    <template #empty="{ query }">
      <div v-if="filtered(query).length === 0" class="py-8 text-center">
        <i class="fas fa-search text-slate-300 text-2xl mb-2"></i>
        <p class="text-sm text-slate-400">
          {{ parsedPorts.length === 0 ? '该容器未暴露端口，请手动输入' : '无匹配端口' }}
        </p>
      </div>
    </template>

    <template #footer>
      <div v-if="parsedPorts.length > 0" class="px-3 py-2 bg-slate-50 border-t border-slate-100 flex items-center justify-between">
        <span class="text-xs text-slate-400">共 <strong class="text-slate-700">{{ parsedPorts.length }}</strong> 个容器端口</span>
        <span class="text-xs text-slate-400">回车使用输入值</span>
      </div>
    </template>
  </Combobox>
</template>
