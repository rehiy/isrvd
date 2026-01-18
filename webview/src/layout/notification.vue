<script setup>
import { inject } from 'vue'

import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state.js'

const state = inject(APP_STATE_KEY)
const actions = inject(APP_ACTIONS_KEY)

const toastClass = (type) => {
  return type === 'error' ? 'text-bg-danger' : 'text-bg-success'
}

const toastIcon = (type) => {
  return type === 'error' ? 'fa-circle-exclamation' : 'fa-circle-check'
}
</script>

<template>
  <Teleport v-if="state.notifications.length" to="body">
    <div class="toast-container position-fixed top-0 end-0 p-3">
      <div v-for="item in state.notifications" :key="item.id" :class="['toast show mb-2', toastClass(item.type)]">
        <div class="d-flex align-items-center p-3">
          <i :class="['fas', toastIcon(item.type), 'me-2']"></i>
          <span class="flex-grow-1">{{ item.message }}</span>
          <button type="button" class="btn-close ms-2" @click="actions.clearNotification(item.id)"></button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.toast-container {
  z-index: 9999;
}
</style>
