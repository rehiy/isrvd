<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import { APP_ACTIONS_KEY, APP_STATE_KEY } from '@/store/state'
import type { AppActions, AppState } from '@/store/state'

@Component
class ConfirmModal extends Vue {
    @Inject({ from: APP_STATE_KEY }) readonly state!: AppState
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 计算属性 ───
    get iconColorClass() {
        const colors: Record<string, string> = {
            blue: 'bg-blue-100 text-blue-500',
            emerald: 'bg-emerald-100 text-emerald-500',
            amber: 'bg-amber-100 text-amber-500',
            red: 'bg-red-100 text-red-500',
            slate: 'bg-slate-100 text-slate-500'
        }
        return colors[this.state.confirm.iconColor] || colors.blue
    }
}

export default toNative(ConfirmModal)
</script>

<template>
  <Transition
    enter-active-class="transition duration-300 ease-out"
    enter-from-class="opacity-0"
    enter-to-class="opacity-100"
    leave-active-class="transition duration-200 ease-in"
    leave-from-class="opacity-100"
    leave-to-class="opacity-0"
  >
    <div 
      v-if="state.confirm.show" 
      class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-sm"
      @click.self="actions.closeConfirm"
    >
      <div class="w-full max-w-3xl modal-card animate-scale-in">
        <!-- Header -->
        <div class="flex items-center justify-between px-6 py-4 border-b border-slate-200/50">
          <h3 class="text-lg font-semibold text-slate-800">{{ state.confirm.title }}</h3>
          <button 
            type="button" 
            class="w-8 h-8 flex items-center justify-center rounded-lg text-slate-400 hover:text-slate-600 hover:bg-slate-100 transition-all duration-200"
            @click="actions.closeConfirm"
            :disabled="state.confirm.loading"
          >
            <i class="fas fa-times"></i>
          </button>
        </div>

        <!-- Body -->
        <div class="px-6 py-6">
          <div class="text-center">
            <div 
              class="w-20 h-20 rounded-full flex items-center justify-center mx-auto mb-4"
              :class="iconColorClass.split(' ')[0]"
            >
              <i 
                class="fas text-3xl"
                :class="[state.confirm.icon, iconColorClass.split(' ')[1]]"
              ></i>
            </div>
            <p class="text-lg text-slate-700" v-html="state.confirm.message"></p>
            <p v-if="state.confirm.danger" class="text-sm text-red-600 flex items-center justify-center mt-3">
              <i class="fas fa-exclamation-triangle mr-2"></i>
              此操作不可恢复！
            </p>
          </div>
        </div>

        <!-- Footer -->
        <div class="flex justify-end gap-3 px-6 py-4 border-t border-slate-200/50 bg-slate-50/50">
          <button 
            type="button" 
            class="btn-secondary"
            @click="actions.closeConfirm"
            :disabled="state.confirm.loading"
          >
            取消
          </button>
          <button 
            type="button" 
            :class="state.confirm.danger ? 'btn-danger' : 'btn-primary'"
            @click="actions.handleConfirm"
            :disabled="state.confirm.loading"
          >
            <i class="fas fa-spinner fa-spin mr-2" v-if="state.confirm.loading"></i>
            {{ state.confirm.loading ? '处理中...' : state.confirm.confirmText }}
          </button>
        </div>
      </div>
    </div>
  </Transition>
</template>
