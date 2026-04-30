<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions, AppState } from '@/store/state'

import api from '@/service/api'

@Component
class Login extends Vue {
    @Inject({ from: APP_STATE_KEY }) readonly state!: AppState
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    loginForm = {
        username: '',
        password: ''
    }

    // ─── 方法 ───
    async handleLogin() {
        const { payload } = await api.login(this.loginForm)
        if (!payload) return

        this.actions.setAuth({ authMode: 'jwt', ...payload })
        this.loginForm.username = ''
        this.loginForm.password = ''
    }
}

export default toNative(Login)
</script>

<template>
  <div class="min-h-screen flex items-center justify-center p-6">
    <!-- Background Decoration -->
    <div class="fixed inset-0 overflow-hidden pointer-events-none">
      <div class="absolute -top-40 -right-40 w-80 h-80 rounded-full bg-primary-300/20 blur-3xl"></div>
      <div class="absolute -bottom-40 -left-40 w-80 h-80 rounded-full bg-primary-300/20 blur-3xl"></div>
    </div>

    <!-- Login Card -->
    <div class="w-full max-w-md relative animate-fade-in">
      <div class="card p-8">
        <!-- Header -->
        <div class="text-center mb-8">
          <div class="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-primary-500 mb-4 shadow-glow">
            <i class="fas fa-server text-white text-2xl"></i>
          </div>
          <h1 class="text-2xl font-bold text-slate-800 mb-2">欢迎回来</h1>
          <p class="text-slate-500">登录到 Isrvd 管理面板</p>
        </div>

        <!-- Form -->
        <form @submit.prevent="handleLogin" class="space-y-5">
          <div>
            <label for="username" class="block text-sm font-medium text-slate-700 mb-2">
              用户名
            </label>
            <div class="relative">
              <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                <i class="fas fa-user text-slate-400"></i>
              </div>
              <input 
                type="text" 
                id="username" 
                v-model="loginForm.username" 
                required
                class="input pl-11"
                placeholder="请输入用户名"
              >
            </div>
          </div>

          <div>
            <label for="password" class="block text-sm font-medium text-slate-700 mb-2">
              密码
            </label>
            <div class="relative">
              <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                <i class="fas fa-lock text-slate-400"></i>
              </div>
              <input 
                type="password" 
                id="password" 
                v-model="loginForm.password" 
                required
                class="input pl-11"
                placeholder="请输入密码"
              >
            </div>
          </div>

          <button 
            type="submit" 
            :disabled="state.loading"
            class="btn-primary w-full py-3 text-base font-semibold mt-6"
          >
            <i class="fas fa-spinner fa-spin mr-2" v-if="state.loading"></i>
            <i class="fas fa-sign-in-alt mr-2" v-else></i>
            {{ state.loading ? '登录中...' : '登录' }}
          </button>
        </form>
      </div>

      <!-- Footer -->
      <p class="text-center text-sm text-slate-400 mt-6 flex items-center justify-center gap-2">
        <span>© 2024 - {{ new Date().getFullYear() }} Isrvd. All rights reserved.</span>
        <a
          href="https://github.com/rehiy/isrvd"
          target="_blank"
          rel="noopener noreferrer"
          class="inline-flex items-center text-slate-400 hover:text-slate-600 transition-colors"
          title="GitHub"
        >
          <i class="fab fa-github"></i>
        </a>
      </p>
    </div>
  </div>
</template>
