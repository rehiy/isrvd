<script lang="ts">
import { nextTick } from 'vue'
import { Component, Prop, Ref, Vue, Watch, toNative } from 'vue-facing-decorator'

@Component({
    emits: ['update:open']
})
class Dropdown extends Vue {
    @Prop({ type: Boolean, default: false }) readonly open!: boolean
    @Prop({ type: String, default: 'auto' }) readonly placement!: string
    @Prop({ type: Number, default: 6 }) readonly offset!: number
    @Prop({ type: String, default: 'left' }) readonly align!: string
    @Prop({ type: String, default: undefined }) readonly maxWidth!: string | undefined
    @Prop({ type: String, default: '360px' }) readonly maxHeight!: string
    @Prop({ type: Boolean, default: false }) readonly closeOnClick!: boolean

    // ─── Refs ───
    @Ref readonly containerRef!: HTMLDivElement

    // ─── 数据属性 ───
    actualPlacement = 'bottom'

    // ─── 计算属性 ───
    get isOpen() {
        return this.open
    }

    set isOpen(val: boolean) {
        this.$emit('update:open', val)
    }

    // ─── 监听器 ───
    @Watch('open')
    onOpenChange(val: boolean) {
        if (val) this.computePlacement()
    }

    // ─── 方法 ───
    async computePlacement() {
        if (this.placement !== 'auto') {
            this.actualPlacement = this.placement
            return
        }

        await nextTick()

        if (!this.containerRef) {
            this.actualPlacement = 'bottom'
            return
        }

        const rect = this.containerRef.getBoundingClientRect()
        const viewportHeight = window.innerHeight
        const spaceBelow = viewportHeight - rect.bottom
        const spaceAbove = rect.top

        if (spaceBelow < 200 && spaceAbove > spaceBelow) {
            this.actualPlacement = 'top'
        } else {
            this.actualPlacement = 'bottom'
        }
    }

    handleClickOutside(e: MouseEvent) {
        if (!this.containerRef) return
        if (!this.containerRef.contains(e.target as Node)) {
            this.isOpen = false
        }
    }

    // ─── 生命周期 ───
    mounted() {
        document.addEventListener('click', this.handleClickOutside, true)
        window.addEventListener('resize', this.computePlacement)
        window.addEventListener('scroll', this.computePlacement, true)
    }

    unmounted() {
        document.removeEventListener('click', this.handleClickOutside, true)
        window.removeEventListener('resize', this.computePlacement)
        window.removeEventListener('scroll', this.computePlacement, true)
    }
}

export default toNative(Dropdown)
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
          'absolute z-50 min-w-max bg-white rounded-xl shadow-lg border border-slate-200 overflow-hidden',
          align === 'right' ? 'right-0' : 'left-0',
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
