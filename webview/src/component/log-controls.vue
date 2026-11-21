<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import type { LogStream } from '@/helper/log'

@Component({ emits: ['tail-change', 'refresh', 'start', 'stop'] })
class LogControls extends Vue {
    @Prop({ type: Object, required: true }) readonly logs!: LogStream
    @Prop({ type: Boolean, default: false }) readonly canStream!: boolean
    @Prop({ type: Boolean, default: false }) readonly mobile!: boolean

    tails = [
        { value: '50', label: '显示 50 行', short: '50 行' },
        { value: '100', label: '显示 100 行', short: '100 行' },
        { value: '200', label: '显示 200 行', short: '200 行' },
        { value: '500', label: '显示 500 行', short: '500 行' },
        { value: '1000', label: '显示 1000 行', short: '1000 行' },
    ]

    get tail() {
        return this.logs.tail
    }

    set tail(value: string) {
        this.$emit('tail-change', value)
    }
}

export default toNative(LogControls)
</script>

<template>
  <select v-model="tail" class="select-sm" :class="{ 'w-28': !mobile }">
    <option v-for="item in tails" :key="item.value" :value="item.value">
      {{ $t(mobile ? item.short : item.label) }}
    </option>
  </select>
  <button type="button" class="btn btn-secondary" :class="{ 'btn-square': mobile }" :title="mobile ? $t('刷新') : undefined" :disabled="logs.active" @click="$emit('refresh')">
    <i class="fas fa-rotate" :class="{ 'text-sm': mobile }"></i>
    <template v-if="!mobile">{{ $t('刷新') }}</template>
  </button>
  <button v-if="!logs.active && canStream" type="button" class="btn btn-emerald" :class="{ 'btn-square': mobile }" :title="mobile ? $t('实时') : undefined" @click="$emit('start')">
    <i class="fas fa-play" :class="{ 'text-sm': mobile }"></i>
    <template v-if="!mobile">{{ $t('实时') }}</template>
  </button>
  <button v-else-if="logs.active" type="button" class="btn btn-secondary" :class="{ 'btn-square': mobile }" :title="mobile ? $t('停止') : undefined" @click="$emit('stop')">
    <i :class="[logs.state === 'connecting' ? 'fas fa-spinner fa-spin' : 'fas fa-stop', { 'text-sm': mobile }]"></i>
    <template v-if="!mobile">{{ $t('停止') }}</template>
  </button>
</template>
