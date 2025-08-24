<script setup>
import { inject } from 'vue'

import api from '@/service/api.js'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state.js'

const state = inject(APP_STATE_KEY)
const actions = inject(APP_ACTIONS_KEY)

const handleLogout = async () => {
  await api.logout()
  actions.clearAuth()
}
</script>

<template>
  <button class="btn btn-outline-light btn-sm" @click="handleLogout" :disabled="state.loading">
    <i class="fas fa-spinner fa-spin" v-if="state.loading"></i>
    <i class="fas fa-sign-out-alt" v-else></i>
    {{ state.loading ? '注销中...' : '注销' }}
  </button>
</template>
