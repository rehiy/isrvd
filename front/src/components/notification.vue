<script setup>
import { inject } from 'vue'

import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/stores/state.js'

const state = inject(APP_STATE_KEY)
const actions = inject(APP_ACTIONS_KEY)

const toastClass = () => {
  return state.notification.type === 'error' ? 'text-bg-danger' : 'text-bg-success'
}

const toastIcon = () => {
  return state.notification.type === 'error' ? 'fa-circle-exclamation' : 'fa-circle-check'
}
</script>

<template>
  <Teleport to="body">
    <div class="toast-container position-fixed top-0 end-0 p-3">
      <div v-if="state.notification.type" :class="['toast show', toastClass()]">
        <div class="toast-header">
          <i :class="['fas', toastIcon(), 'me-2']"></i>
          <strong class="me-auto">
            {{ state.notification.type === 'error' ? '错误' : '成功' }}
          </strong>
          <button type="button" class="btn-close" @click="actions.clearNotification()"></button>
        </div>
        <div class="toast-body">
          {{ state.notification.message }}
        </div>
      </div>
    </div>
  </Teleport>
</template>
