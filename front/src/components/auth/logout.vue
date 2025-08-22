<script setup>
import axios from 'axios'
import { inject, ref } from 'vue'

import { APP_ACTIONS_KEY } from '@/stores/state.js'

const actions = inject(APP_ACTIONS_KEY)
const loading = ref(false)

const handleLogout = async () => {
  loading.value = true

  try {
    await axios.post('/api/logout')
  } catch (error) {
    console.warn('Logout request failed:', error)
  }

  actions.clearAuth()
  loading.value = false
}
</script>

<template>
  <button class="btn btn-outline-light btn-sm" @click="handleLogout" :disabled="loading">
    <i class="fas fa-spinner fa-spin" v-if="loading"></i>
    <i class="fas fa-sign-out-alt" v-else></i>
    {{ loading ? '注销中...' : '注销' }}
  </button>
</template>
