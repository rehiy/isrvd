<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'

import { loginWithPasskey } from '@/helper/webauthn'

@Component
class Login extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    loading = false
    passkeyLoading = false
    loginForm = {
        username: '',
        password: ''
    }
    totpForm = {
        code: ''
    }
    twoFactorRequired = false

    // ─── 方法 ───
    async handleLogin() {
        this.loading = true
        try {
            const { payload } = await api.accountLogin({
                ...this.loginForm,
                totpCode: this.twoFactorRequired ? this.totpForm.code : undefined
            })
            if (!payload) return
            if (payload.twoFactorRequired) {
                this.twoFactorRequired = true
                this.totpForm.code = ''
                return
            }
            if (!payload.token) return

            this.portal.setAuth({ authMode: 'jwt', token: payload.token, username: payload.username })
            await this.portal.initialize()
            this.loginForm.username = ''
            this.loginForm.password = ''
            this.totpForm.code = ''
            this.twoFactorRequired = false
        } finally {
            this.loading = false
        }
    }

    resetTwoFactor() {
        this.twoFactorRequired = false
        this.totpForm.code = ''
    }

    handleOIDCLogin() {
        window.location.href = 'api/account/oidc/login'
    }

    // ─── Passkey 登录 ───
    async handlePasskeyLogin() {
        this.passkeyLoading = true
        try {
            const result = await loginWithPasskey(this.loginForm.username || undefined)
            this.portal.setAuth({ authMode: 'jwt', ...result })
            await this.portal.initialize()
        } catch (e) {
            console.error('Passkey 登录失败:', e)
        } finally {
            this.passkeyLoading = false
        }
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
              <input id="username" v-model="loginForm.username" type="text" required class="input pl-11" placeholder="请输入用户名" @input="resetTwoFactor">
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
              <input id="password" v-model="loginForm.password" type="password" required class="input pl-11" placeholder="请输入密码" @input="resetTwoFactor">
            </div>
          </div>

          <div v-if="twoFactorRequired">
            <label for="totp-code" class="form-label">
              二次验证码
            </label>
            <div class="relative">
              <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                <i class="fas fa-shield-halved text-slate-400"></i>
              </div>
              <input id="totp-code" v-model="totpForm.code" type="text" inputmode="numeric" autocomplete="one-time-code" required class="input pl-11" placeholder="请输入 6 位验证码">
            </div>
            <p class="text-xs text-slate-400 mt-1">该账户已启用 TOTP 二次验证，请输入认证器 App 中的动态验证码。</p>
          </div>

          <button type="submit" :disabled="loading || (twoFactorRequired && !totpForm.code)" class="btn btn-primary w-full mt-6">
            <i v-if="loading" class="fas fa-spinner fa-spin mr-2"></i>
            <i v-else class="fas fa-sign-in-alt mr-2"></i>
            {{ loading ? '登录中...' : (twoFactorRequired ? '验证并登录' : '登录') }}
          </button>

          <template v-if="portal.oidcEnabled || portal.passkeyEnabled">
            <div class="border-t border-slate-200 pt-4 space-y-3">
              <button
                v-if="portal.passkeyEnabled"
                type="button"
                class="btn btn-emerald w-full"
                :disabled="passkeyLoading"
                @click="handlePasskeyLogin"
              >
                <i v-if="passkeyLoading" class="fas fa-spinner fa-spin mr-2"></i>
                <i v-else class="fas fa-fingerprint mr-2"></i>
                {{ passkeyLoading ? 'Passkey 认证中...' : '使用 Passkey 登录' }}
              </button>
              <button v-if="portal.oidcEnabled" type="button" class="btn btn-indigo w-full" @click="handleOIDCLogin">
                <i class="fas fa-id-badge mr-2"></i>
                {{ portal.oidcLoginLabel || '使用 OIDC 登录' }}
              </button>
            </div>
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
