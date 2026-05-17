<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

@Component
class NotificationManager extends Vue {
    portal = usePortal()

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
    <div v-if="portal.notifications.length" class="fixed top-6 right-6 z-[9999] space-y-3">
      <TransitionGroup
        enter-active-class="transition duration-300 ease-out"
        enter-from-class="opacity-0 translate-x-4"
        enter-to-class="opacity-100 translate-x-0"
        leave-active-class="transition duration-200 ease-in"
        leave-from-class="opacity-100 translate-x-0"
        leave-to-class="opacity-0 translate-x-4"
      >
        <div 
          v-for="item in portal.notifications" 
          :key="item.id" 
          :class="['flex items-center px-5 py-4 rounded-2xl min-w-80 animate-slide-down', notificationStyle(item.type)]"
        >
          <div class="w-8 h-8 rounded-full bg-white/20 flex items-center justify-center mr-3">
            <i :class="['fas', notificationIcon(item.type)]"></i>
          </div>
          <span class="flex-1 font-medium">{{ item.message }}</span>
          <button 
            type="button" 
            class="btn-icon hover:bg-white/20!"
            @click="portal.clearNotification(item.id)"
          >
            <i class="fas fa-times"></i>
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>
