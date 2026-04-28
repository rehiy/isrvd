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
    @Prop({ type: String, default: 'left' }) readonly align!: 'left' | 'right' | 'match' | 'match-right'
    @Prop({ type: String, default: undefined }) readonly maxWidth!: string | undefined
    @Prop({ type: String, default: '360px' }) readonly maxHeight!: string
    @Prop({ type: Boolean, default: false }) readonly closeOnClick!: boolean

    // ─── Refs ───
    @Ref readonly containerRef!: HTMLDivElement
    @Ref readonly panelRef!: HTMLDivElement

    // ─── 数据属性 ───
    actualPlacement = 'bottom'
    panelStyle: Record<string, string> = {}

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
        if (val) {
            this.computePlacement()
        } else {
            this.panelStyle = {}
        }
    }

    // ─── 方法 ───
    async computePlacement(immediate = false) {
        if (!this.containerRef) return

        if (!immediate) await nextTick()

        const rect = this.containerRef.getBoundingClientRect()
        const viewportHeight = window.innerHeight

        // 计算上下空间
        const spaceBelow = viewportHeight - rect.bottom
        const spaceAbove = rect.top

        // 解析 maxHeight 以获取面板最大可能高度
        const panelMaxH = parseInt(this.maxHeight) || 360

        if (this.placement === 'auto') {
            // 根据面板实际高度（若有）或最大高度判断空间是否充足
            const panelH = this.panelRef?.offsetHeight || panelMaxH
            this.actualPlacement = (spaceBelow < panelH && spaceAbove > spaceBelow) ? 'top' : 'bottom'
        } else {
            this.actualPlacement = this.placement
        }

        // 计算 fixed 定位坐标
        const style: Record<string, string> = {
            position: 'fixed',
            zIndex: '9999',
        }

        if (this.actualPlacement === 'top') {
            style.bottom = (viewportHeight - rect.top + this.offset) + 'px'
            // 上方展开时限制最大高度不超过上方可用空间
            const availableHeight = spaceAbove - this.offset
            style.maxHeight = Math.min(panelMaxH, availableHeight) + 'px'
        } else {
            style.top = (rect.bottom + this.offset) + 'px'
            // 下方展开时限制最大高度不超过下方可用空间
            const availableHeight = spaceBelow - this.offset
            style.maxHeight = Math.min(panelMaxH, availableHeight) + 'px'
        }

        // 水平对齐
        if (this.align === 'right' || this.align === 'match-right') {
            style.right = (window.innerWidth - rect.right) + 'px'
        } else {
            style.left = rect.left + 'px'
        }

        // 宽度匹配
        if (this.align === 'match' || this.align === 'match-right') {
            style.width = rect.width + 'px'
        }

        if (this.maxWidth) {
            style.maxWidth = this.maxWidth
        }

        this.panelStyle = style
    }

    handleClickOutside(e: MouseEvent) {
        if (!this.containerRef) return
        const target = e.target as Node
        // 点击在容器外 且 不在面板内
        if (!this.containerRef.contains(target) && !(this.panelRef && this.panelRef.contains(target))) {
            this.isOpen = false
        }
    }

    handleScroll(e: Event) {
        // 如果滚动发生在面板内部，不关闭
        if (this.panelRef && this.panelRef.contains(e.target as Node)) return
        if (this.isOpen) this.isOpen = false
    }

    // ─── 生命周期 ───
    mounted() {
        document.addEventListener('click', this.handleClickOutside, true)
        window.addEventListener('resize', this.handleScroll)
        document.addEventListener('scroll', this.handleScroll, true)
    }

    unmounted() {
        document.removeEventListener('click', this.handleClickOutside, true)
        window.removeEventListener('resize', this.handleScroll)
        document.removeEventListener('scroll', this.handleScroll, true)
    }
}

export default toNative(Dropdown)
</script>

<template>
  <div class="relative" ref="containerRef">
    <!-- 触发区域 -->
    <slot name="trigger" :open="isOpen" :toggle="() => isOpen = !isOpen" />
  </div>

  <!-- 下拉面板 - Teleport 到 body 以避免 overflow 裁剪 -->
  <Teleport to="body">
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
        ref="panelRef"
        :class="[
          'bg-white rounded-xl shadow-lg border border-slate-200 overflow-hidden flex flex-col',
          (align !== 'match' && align !== 'match-right') ? 'min-w-max' : '',
        ]"
        :style="panelStyle"
        @click="closeOnClick && (isOpen = false)"
      >
        <!-- 搜索提示条 -->
        <slot name="search-hint" />

        <!-- 分组列表 -->
        <div class="overflow-y-auto flex-1">
          <slot />
        </div>

        <!-- 空状态 -->
        <slot name="empty" />

        <!-- 底部统计栏 -->
        <slot name="footer" />
      </div>
    </Transition>
  </Teleport>
</template>
