<script lang="ts">
import { Component, Prop, Ref, Vue, Watch, toNative } from 'vue-facing-decorator'

@Component({
    expose: ['open', 'close'],
    emits: ['update:modelValue', 'confirm', 'cancel']
})
class BaseModal extends Vue {
    @Prop({ type: Boolean, default: false }) readonly modelValue!: boolean
    @Prop({ type: String, default: '' }) readonly title!: string
    @Prop({ type: Boolean, default: false }) readonly loading!: boolean
    @Prop({ type: Boolean, default: true }) readonly showFooter!: boolean
    @Prop({ type: Boolean, default: false }) readonly confirmDisabled!: boolean
    @Prop({ type: String, default: 'btn-primary' }) readonly confirmClass!: string
    @Prop({ type: Boolean, default: true }) readonly showConfirm!: boolean
    @Prop({ type: String, default: 'max-w-3xl' }) readonly maxWidthClass!: string
    @Prop({ type: String, default: '' }) readonly cardClass!: string
    @Prop({ type: String, default: 'px-6 py-6 overflow-y-auto' }) readonly bodyClass!: string

    // ─── Refs ───
    @Ref readonly modalRef!: HTMLDivElement

    // ─── 数据属性 ───
    isOpen = this.modelValue
    mouseDownTarget: EventTarget | null = null

    // ─── 监听器 ───
    @Watch('modelValue')
    onModelValueChange(val: boolean) {
        this.isOpen = val
    }

    // ─── 方法 ───
    open() {
        this.isOpen = true
        this.$emit('update:modelValue', true)
    }

    close() {
        this.isOpen = false
        this.$emit('update:modelValue', false)
    }

    handleConfirm() {
        this.$emit('confirm')
    }

    handleCancel() {
        this.$emit('cancel')
        this.close()
    }

    handleBackdropMouseDown(e: MouseEvent) {
        this.mouseDownTarget = e.target
    }

    handleBackdropClick(e: MouseEvent) {
        // 只有 mousedown 和 mouseup 都落在遮罩层本身时才关闭
        // 防止在表单内拖拽选中文字后鼠标松开偏移到遮罩层触发误关闭
        if (e.target === this.modalRef && this.mouseDownTarget === this.modalRef) {
            this.handleCancel()
        }
        this.mouseDownTarget = null
    }

    handleEscape(e: KeyboardEvent) {
        if (e.key === 'Escape' && this.isOpen) {
            this.handleCancel()
        }
    }

    // ─── 生命周期 ───
    mounted() {
        document.addEventListener('keydown', this.handleEscape)
    }

    unmounted() {
        document.removeEventListener('keydown', this.handleEscape)
    }
}

export default toNative(BaseModal)
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-300 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition duration-200 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div 
        v-if="isOpen" 
        ref="modalRef"
        class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-sm"
        @mousedown="handleBackdropMouseDown"
        @click="handleBackdropClick"
      >
        <div :class="['w-full max-h-[calc(100vh-2rem)] modal-card animate-scale-in flex flex-col overflow-hidden', maxWidthClass, cardClass]">
          <!-- Header -->
          <div class="flex items-center justify-between px-6 py-4 border-b border-slate-200/50 flex-shrink-0">
            <div class="min-w-0 flex-1 pr-4">
              <slot name="title">
                <h1 class="text-lg font-semibold text-slate-800 truncate">{{ title }}</h1>
              </slot>
            </div>
            <div class="flex items-center gap-2 flex-shrink-0">
              <slot name="header-actions"></slot>
              <button type="button" class="btn-icon-sm" :disabled="loading" @click="handleCancel">
                <i class="fas fa-times"></i>
              </button>
            </div>
          </div>

          <!-- Body -->
          <div :class="['flex-1 min-h-0', bodyClass]">
            <slot></slot>
          </div>

          <!-- Footer -->
          <div v-if="showFooter" class="flex justify-end gap-3 px-6 py-4 border-t border-slate-200/50 bg-slate-50/50 flex-shrink-0">
            <slot name="footer">
              <button type="button" class="btn btn-secondary" :disabled="loading" @click="handleCancel">
                <slot name="cancel-text">取消</slot>
              </button>
              <button v-if="showConfirm" type="button" class="btn" :class="confirmClass" :disabled="loading || confirmDisabled" @click="handleConfirm">
                <i v-if="loading" class="fas fa-spinner fa-spin"></i>
                <slot name="confirm-text">确认</slot>
              </button>
            </slot>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
