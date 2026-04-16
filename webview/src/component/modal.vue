<script setup>
import { ref, watch, onMounted, onUnmounted } from 'vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  title: { type: String, default: '' },
  loading: { type: Boolean, default: false },
  showFooter: { type: Boolean, default: true },
  confirmDisabled: { type: Boolean, default: false }
})

const emit = defineEmits(['update:modelValue', 'confirm', 'cancel'])

const modalRef = ref(null)
const isOpen = ref(props.modelValue)

const open = () => {
  isOpen.value = true
  emit('update:modelValue', true)
}

const close = () => {
  isOpen.value = false
  emit('update:modelValue', false)
}

const handleConfirm = () => {
  emit('confirm')
}

const handleCancel = () => {
  emit('cancel')
  close()
}

const handleBackdropClick = (e) => {
  if (e.target === modalRef.value) {
    handleCancel()
  }
}

const handleEscape = (e) => {
  if (e.key === 'Escape' && isOpen.value) {
    handleCancel()
  }
}

watch(() => props.modelValue, (val) => {
  isOpen.value = val
})

onMounted(() => {
  document.addEventListener('keydown', handleEscape)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleEscape)
})

defineExpose({ open, close })
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
        @click="handleBackdropClick"
      >
        <div 
          v-if="isOpen"
          :class="['w-full modal-card animate-scale-in', 'max-w-3xl']"
        >
          <!-- Header -->
          <div class="flex items-center justify-between px-6 py-4 border-b border-slate-200/50">
            <h3 class="text-lg font-semibold text-slate-800">
              <slot name="title">{{ title }}</slot>
            </h3>
            <button 
              type="button" 
              class="w-8 h-8 flex items-center justify-center rounded-lg text-slate-400 hover:text-slate-600 hover:bg-slate-100 transition-all duration-200"
              @click="handleCancel"
              :disabled="loading"
            >
              <i class="fas fa-times"></i>
            </button>
          </div>

          <!-- Body -->
          <div class="px-6 py-6 max-h-[70vh] overflow-y-auto">
            <slot></slot>
          </div>

          <!-- Footer -->
          <div v-if="showFooter" class="flex justify-end gap-3 px-6 py-4 border-t border-slate-200/50 bg-slate-50/50">
            <slot name="footer">
              <button 
                type="button" 
                class="btn-secondary"
                @click="handleCancel"
                :disabled="loading"
              >
                <slot name="cancel-text">取消</slot>
              </button>
              <button 
                type="button" 
                class="btn-primary"
                @click="handleConfirm"
                :disabled="loading || confirmDisabled"
              >
                <i class="fas fa-spinner fa-spin mr-2" v-if="loading"></i>
                <slot name="confirm-text">确认</slot>
              </button>
            </slot>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
