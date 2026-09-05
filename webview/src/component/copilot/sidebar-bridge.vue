<script setup lang="ts">
import { onUnmounted, watchEffect } from 'vue'

import { useCopilotStore } from '@/stores'

// CopilotSidebar 的开合控制只在 toggle-button 插槽作用域内可得，
// 此桥接组件把 toggle/isOpen 同步进全局 store，供顶栏按钮调用。
const props = defineProps<{
    isOpen: boolean
    toggle: () => void
}>()

const copilot = useCopilotStore()

watchEffect(() => {
    copilot.bindSidebar({ isOpen: props.isOpen, toggle: props.toggle })
})

onUnmounted(copilot.unbindSidebar)
</script>

<template>
  <span hidden></span>
</template>
