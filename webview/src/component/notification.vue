<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state'

@Component
class NotificationManager extends Vue {
    @Inject({ from: APP_STATE_KEY }) readonly state!: any
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: any

    // ─── 方法 ───
    notificationStyle(type: string) {
        return type === 'error'
            ? 'bg-red-500 text-white shadow-lg shadow-red-500/30'
            : 'bg-emerald-500 text-white shadow-lg shadow-emerald-500/30'
    }

    notificationIcon(type: string) {
        return type === 'error' ? 'fa-circle-exclamation' : 'fa-circle-check'
    }
}

export default toNative(NotificationManager)
</script>

<template>
  <Teleport to="body">
    <div v-if="state.notifications.length" class="fixed top-6 right-6 z-[9999] space-y-3">
      <TransitionGroup
        enter-active-class="transition duration-300 ease-out"
        enter-from-class="opacity-0 translate-x-4"
        enter-to-class="opacity-100 translate-x-0"
        leave-active-class="transition duration-200 ease-in"
        leave-from-class="opacity-100 translate-x-0"
        leave-to-class="opacity-0 translate-x-4"
      >
        <div 
          v-for="item in state.notifications" 
          :key="item.id" 
          :class="['flex items-center px-5 py-4 rounded-2xl min-w-80 animate-slide-down', notificationStyle(item.type)]"
        >
          <div class="w-8 h-8 rounded-full bg-white/20 flex items-center justify-center mr-3">
            <i :class="['fas', notificationIcon(item.type)]"></i>
          </div>
          <span class="flex-1 font-medium">{{ item.message }}</span>
          <button 
            type="button" 
            class="ml-4 w-8 h-8 flex items-center justify-center rounded-lg hover:bg-white/20 transition-colors"
            @click="actions.clearNotification(item.id)"
          >
            <i class="fas fa-times"></i>
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>
