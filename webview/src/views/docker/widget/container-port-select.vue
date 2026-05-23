<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

interface PortItem {
    port: string
    proto: string
}

@Component({
    emits: ['update:modelValue']
})
class ContainerPortSelect extends Vue {
    @Prop({ type: String, default: '' }) readonly modelValue!: string
    @Prop({ type: Array, default: () => [] }) readonly ports!: string[]
    @Prop({ type: String, default: '80' }) readonly placeholder!: string
    @Prop({ type: Boolean, default: false }) readonly disabled!: boolean

    parsePort(raw: string): PortItem | null {
        const [body, proto = 'tcp'] = raw.split('/')
        if (!body) return null
        // Docker 端口格式：[ip:]publicPort:privatePort
        // IPv6 通配地址可能为 :::8080:80，取最后一段作为端口号
        const port = body.split(':').pop()
        if (!port || !/^\d+$/.test(port)) return null
        return { port, proto }
    }

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
}

export default toNative(ContainerPortSelect)
</script>

<template>
  <input
    v-if="parsedPorts.length === 0"
    type="text"
    class="input"
    :value="modelValue"
    :placeholder="placeholder"
    :disabled="disabled"
    @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
  />
  <select v-else class="input" :value="modelValue" :disabled="disabled" @change="$emit('update:modelValue', ($event.target as HTMLSelectElement).value)">
    <option value="" disabled>选择端口</option>
    <option v-for="item in parsedPorts" :key="item.port + '-' + item.proto" :value="item.port">
      {{ item.port }} / {{ item.proto }}
    </option>
  </select>
</template>
