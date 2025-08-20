<template>
  <nav class="navbar navbar-expand-lg navbar-dark bg-dark mb-3">
    <div class="container-fluid">
      <a class="navbar-brand fw-semibold" href="#" @click="goHome">
        <i class="fas fa-folder-open me-2"></i>服务器管理
      </a>
      <div v-if="state.user" class="d-flex align-items-center">
        <span class="navbar-text me-3">欢迎, {{ state.user }}</span>
        <LogoutButton />
      </div>
    </div>
  </nav>
</template>

<script>
import { defineComponent, inject } from 'vue'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '../helpers/state.js'
import LogoutButton from './logout_button.vue'

export default defineComponent({
  name: 'NavigationBar',
  components: {
    LogoutButton
  },
  setup() {
    const state = inject(APP_STATE_KEY)
    const actions = inject(APP_ACTIONS_KEY)

    const goHome = () => {
      actions.loadFiles('/')
    }

    return {
      state,
      goHome
    }
  }
})
</script>
