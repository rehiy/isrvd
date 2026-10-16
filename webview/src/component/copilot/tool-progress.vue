<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
    label: string
    args?: { path?: string; tag?: string; q?: string }
    status?: string
    result?: unknown
}>(), { status: 'inProgress', args: undefined, result: undefined })
const error = computed(() => {
    try {
        const result: unknown = typeof props.result === 'string' ? JSON.parse(props.result) : props.result
        if (!result || typeof result !== 'object' || !('success' in result) || result.success !== false) return ''
        return 'message' in result && typeof result.message === 'string' ? result.message || '请求失败' : '请求失败'
    } catch {
        return ''
    }
})
</script>

<template>
  <div class="not-prose py-1.5 text-xs text-slate-500">
    <div class="inline-info min-w-0">
      <i class="fas w-4 text-center flex-shrink-0" :class="error ? 'fa-circle-exclamation text-red-500' : status === 'complete' ? 'fa-check text-slate-400' : 'fa-circle-notch fa-spin'" aria-hidden="true"></i>
      <span class="flex-shrink-0">{{ $t(label) }}</span>
      <span v-if="args?.path || args?.tag || args?.q" class="truncate text-slate-400">{{ args.path || args.tag || args.q }}</span>
    </div>
    <p v-if="error" class="mt-1 ml-6 text-red-500 break-words">{{ $t(error) }}</p>
  </div>
</template>
