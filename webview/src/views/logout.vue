<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import { APP_ACTIONS_KEY, APP_STATE_KEY } from '@/store/state'
import type { AppActions, AppState } from '@/store/state'

@Component
class Logout extends Vue {
    @Inject({ from: APP_STATE_KEY }) readonly state!: AppState
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 方法 ───
    async handleLogout() {
        await api.logout()
        this.actions.clearAuth()
    }
}

export default toNative(Logout)
</script>

<template>
  <button 
    v-if="state.authMode !== 'header'"
    class="btn-ghost px-4 py-2 text-sm font-medium text-slate-600 hover:text-red-600 hover:bg-red-50 rounded-lg transition-all duration-200 flex items-center gap-2"
    @click="handleLogout" 
    :disabled="state.loading"
  >
    <i class="fas fa-spinner fa-spin" v-if="state.loading"></i>
    <i class="fas fa-sign-out-alt" v-else></i>
    {{ state.loading ? '刷新' : '注销' }}
  </button>
</template>
