<script setup>
import { inject } from 'vue'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY, APP_STATE_KEY } from '@/store/state.js'

const state = inject(APP_STATE_KEY)
const actions = inject(APP_ACTIONS_KEY)

const handleLogout = async () => {
  await api.logout()
  actions.clearAuth()
}
</script>

<template>
  <button 
    class="btn-ghost px-4 py-2 text-sm font-medium text-slate-600 hover:text-red-600 hover:bg-red-50 rounded-lg transition-all duration-200 flex items-center gap-2"
    @click="handleLogout" 
    :disabled="state.loading"
  >
    <i class="fas fa-spinner fa-spin" v-if="state.loading"></i>
    <i class="fas fa-sign-out-alt" v-else></i>
    {{ state.loading ? '请求...' : '注销' }}
  </button>
</template>
