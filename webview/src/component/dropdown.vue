<script setup>
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'

const props = defineProps({
  open: { type: Boolean, default: false },
  placement: { type: String, default: 'auto' }, // 'auto' | 'bottom' | 'top'
  offset: { type: Number, default: 6 },
  maxWidth: { type: String, default: undefined },
  maxHeight: { type: String, default: '360px' },
  closeOnClick: { type: Boolean, default: false },
})

const emit = defineEmits(['update:open'])

const containerRef = ref(null)
const actualPlacement = ref('bottom')

// 双向绑定 open
const isOpen = computed({
  get: () => props.open,
  set: (val) => emit('update:open', val),
})

// 计算实际显示方向
const computePlacement = async () => {
  if (props.placement !== 'auto') {
    actualPlacement.value = props.placement
    return
  }

  await nextTick()

  if (!containerRef.value) {
    actualPlacement.value = 'bottom'
    return
  }

  const rect = containerRef.value.getBoundingClientRect()
  const viewportHeight = window.innerHeight
  const spaceBelow = viewportHeight - rect.bottom
  const spaceAbove = rect.top

  // 下方空间不足 200px 且上方空间更充足时，显示在上方
  if (spaceBelow < 200 && spaceAbove > spaceBelow) {
    actualPlacement.value = 'top'
  } else {
    actualPlacement.value = 'bottom'
  }
}

// 打开时计算方向
watch(() => props.open, (val) => {
  if (val) computePlacement()
})

// 点击外部关闭
const handleClickOutside = (e) => {
  if (!containerRef.value) return
  if (!containerRef.value.contains(e.target)) {
    isOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside, true)
  window.addEventListener('resize', computePlacement)
  window.addEventListener('scroll', computePlacement, true)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside, true)
  window.removeEventListener('resize', computePlacement)
  window.removeEventListener('scroll', computePlacement, true)
})

// 暴露方法
defineExpose({ containerRef })
</script>

<template>
  <div class="relative" ref="containerRef">
    <!-- 触发区域 -->
    <slot name="trigger" :open="isOpen" :toggle="() => isOpen = !isOpen" />

    <!-- 下拉面板 -->
    <Transition
      enter-active-class="transition duration-200 ease-out"
      :enter-from-class="actualPlacement === 'top'
        ? 'opacity-0 translate-y-1'
        : 'opacity-0 -translate-y-1'"
      enter-to-class="opacity-100 translate-y-0"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="opacity-100 translate-y-0"
      :leave-to-class="actualPlacement === 'top'
        ? 'opacity-0 translate-y-1'
        : 'opacity-0 -translate-y-1'"
    >
      <div
        v-if="isOpen"
        :class="[
          'absolute z-50 left-0 right-0 bg-white rounded-xl shadow-lg border border-slate-200 overflow-hidden',
          actualPlacement === 'top' ? 'bottom-full' : 'top-full',
        ]"
        :style="{
          [actualPlacement === 'top' ? 'marginBottom' : 'marginTop']: offset + 'px',
          maxWidth: maxWidth,
        }"
        @click="closeOnClick && (isOpen = false)"
      >
        <!-- 搜索提示条 -->
        <slot name="search-hint" />

        <!-- 分组列表 -->
        <div class="overflow-y-auto" :style="{ maxHeight: maxHeight }">
          <slot />
        </div>

        <!-- 空状态 -->
        <slot name="empty" />

        <!-- 底部统计栏 -->
        <slot name="footer" />
      </div>
    </Transition>
  </div>
</template>
