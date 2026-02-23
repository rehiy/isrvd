<script setup>
import { inject } from 'vue'

import { APP_STATE_KEY } from '@/store/state.js'

import AuthLogout from '@/views/logout.vue'

const state = inject(APP_STATE_KEY)
</script>

<template>
  <nav class="navbar navbar-expand-lg navbar-dark bg-dark mb-3">
    <div class="container-fluid">
      <a class="navbar-brand fw-semibold">Isrvd</a>

      <div v-if="state.username" class="navbar-nav me-auto">
        <router-link to="/explorer" class="nav-link" active-class="active">
          <i class="fas fa-folder-open me-1"></i> 文件管理
        </router-link>
        <router-link to="/markdown" class="nav-link" active-class="active">
          <i class="fas fa-edit me-1"></i> Markdown
        </router-link>
        <router-link to="/shell" class="nav-link" active-class="active">
          <i class="fas fa-terminal me-1"></i> Shell 终端
        </router-link>
      </div>

      <div v-if="state.username" class="d-flex align-items-center">
        <span class="navbar-text me-3">欢迎, {{ state.username }}</span>
        <AuthLogout />
      </div>
    </div>

  </nav>
</template>
