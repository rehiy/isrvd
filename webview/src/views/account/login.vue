<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'

@Component
class Login extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    loading = false
    loginForm = {
        username: '',
        password: ''
    }

    // ─── 方法 ───
    async handleLogin() {
        this.loading = true
        try {
            const { payload } = await api.accountLogin(this.loginForm)
            if (!payload) return

            this.portal.setAuth({ authMode: 'jwt', ...payload })
            await this.portal.initialize()
            this.loginForm.username = ''
            this.loginForm.password = ''
        } finally {
            this.loading = false
        }
    }

    handleOIDCLogin() {
        window.location.href = 'api/account/oidc/login'
    }
}

export default toNative(Login)
</script>

<template>
  <div class="min-h-screen flex items-center justify-center p-6 bg-gradient-to-br from-slate-50 to-slate-100">
    <!-- Background Decoration -->
    <div class="fixed inset-0 overflow-hidden pointer-events-none">
      <div class="absolute -top-40 -right-40 w-80 h-80 rounded-3xl bg-primary-300/20 blur-3xl"></div>
      <div class="absolute -bottom-40 -left-40 w-80 h-80 rounded-3xl bg-primary-300/20 blur-3xl"></div>
    </div>

    <!-- Login Card -->
    <div class="w-full max-w-md relative animate-fade-in">
      <div class="card p-8">
        <!-- Header -->
        <div class="text-center mb-8">
          <img src="@/assets/logo.svg" alt="iSrvd" class="inline-flex w-28 object-contain my-5 transform hover:scale-105 transition-transform duration-300">
          <h1 class="text-2xl font-bold text-slate-800 mb-2">欢迎回来</h1>
          <p class="text-slate-500">登录到 iSrvd 管理面板</p>
        </div>

        <!-- Form -->
        <form class="space-y-5" @submit.prevent="handleLogin">
          <div>
            <label for="username" class="form-label">
              用户名
            </label>
            <div class="relative">
              <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                <i class="fas fa-user text-slate-400"></i>
              </div>
              <input id="username" v-model="loginForm.username" type="text" required class="input pl-11" placeholder="请输入用户名">
            </div>
          </div>

          <div>
            <label for="password" class="form-label">
              密码
            </label>
            <div class="relative">
              <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                <i class="fas fa-lock text-slate-400"></i>
              </div>
              <input id="password" v-model="loginForm.password" type="password" required class="input pl-11" placeholder="请输入密码">
            </div>
          </div>

          <button type="submit" :disabled="loading" class="btn btn-primary w-full mt-6">
            <i v-if="loading" class="fas fa-spinner fa-spin mr-2"></i>
            <i v-else class="fas fa-sign-in-alt mr-2"></i>
            {{ loading ? '登录中...' : '登录' }}
          </button>

          <template v-if="portal.oidcEnabled">
            <div class="relative py-1">
              <div class="absolute inset-0 flex items-center">
                <div class="w-full border-t border-slate-200"></div>
              </div>
              <div class="relative flex justify-center text-xs">
                <span class="bg-white px-2 text-slate-400">或</span>
              </div>
            </div>
            <button type="button" class="btn btn-secondary w-full" @click="handleOIDCLogin">
              <i class="fas fa-right-to-bracket mr-2"></i>
              使用 OIDC 登录
            </button>
          </template>
        </form>
      </div>

      <!-- Footer -->
      <p class="text-center text-sm text-slate-400 mt-6 flex items-center justify-center gap-2">
        <span>© 2024 - {{ new Date().getFullYear() }} <a href="https://isrvd.rehiy.com" target="_blank">iSrvd</a>. All rights reserved.</span>
      </p>
    </div>
  </div>
</template>
