<script setup>
import { inject, ref } from 'vue'

import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/stores/state.js'

import AuthLogout from '@/layouts/logout.vue'
import ShellModal from '@/modals/shell.vue'

const state = inject(APP_STATE_KEY)
const actions = inject(APP_ACTIONS_KEY)

const shellModalRef = ref(null)

const goHome = () => {
  actions.loadFiles('/')
}
</script>

<template>
  <nav class="navbar navbar-expand-lg navbar-dark bg-dark mb-3">
    <div class="container-fluid">
      <a class="navbar-brand fw-semibold" href="#" @click="goHome">
        <i class="fas fa-folder-open me-2"></i> Isrvd
      </a>

      <div v-if="state.user" class="navbar-nav me-auto">
        <a class="nav-link" href="#" @click="shellModalRef.show">
          <i class="fas fa-terminal me-1"></i> 终端
        </a>
      </div>

      <div v-if="state.user" class="d-flex align-items-center">
        <span class="navbar-text me-3">欢迎, {{ state.user }}</span>
        <AuthLogout />
      </div>
    </div>

    <!-- 终端模态框 -->
    <ShellModal ref="shellModalRef" />
  </nav>
</template>
