<script lang="ts">
import QRCode from 'qrcode'
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'

import { copyToClipboard } from '@/helper/utils'

@Component
class AccountPassword extends Vue {
    portal = usePortal()

    passwordForm = { oldPassword: '', newPassword: '', confirmPassword: '' }
    passwordLoading = false

    twoFactorLoading = false
    twoFactorEnabled = false
    totpSetup = { secret: '', qrcode: '', code: '' }
    totpDisableCode = ''

    // ─── 权限 ───
    get canViewTOTP() { return this.portal.hasPerm('GET /api/account/2fa/status') }
    get canEnableTOTP() {
        return this.portal.hasPerm('POST /api/account/2fa/totp/begin')
            && this.portal.hasPerm('POST /api/account/2fa/totp/enable')
    }
    get canDisableTOTP() { return this.portal.hasPerm('POST /api/account/2fa/totp/disable') }

    async mounted() {
        if (!this.canViewTOTP) return
        await this.loadTwoFactorStatus()
        if (!this.twoFactorEnabled && this.canEnableTOTP) {
            await this.handleTOTPBegin()
        }
    }

    async loadTwoFactorStatus() {
        this.twoFactorLoading = true
        try {
            const { payload } = await api.accountTwoFactorStatus()
            this.twoFactorEnabled = !!payload?.enabled
        } finally {
            this.twoFactorLoading = false
        }
    }

    async handleTOTPBegin() {
        this.twoFactorLoading = true
        try {
            const { payload } = await api.accountTOTPBegin()
            const uri = payload?.uri || ''
            const qrcode = uri ? await QRCode.toDataURL(uri, { width: 200, margin: 1 }) : ''
            this.totpSetup = { secret: payload?.secret || '', qrcode, code: '' }
        } finally {
            this.twoFactorLoading = false
        }
    }

    async handleTOTPEnable() {
        if (!this.totpSetup.code) {
            this.portal.showNotification('error', '请输入认证器中的验证码')
            return
        }
        this.twoFactorLoading = true
        try {
            await api.accountTOTPEnable({ secret: this.totpSetup.secret, code: this.totpSetup.code })
            this.portal.showNotification('success', 'TOTP 二次验证已启用')
            this.totpSetup = { secret: '', qrcode: '', code: '' }
            await this.loadTwoFactorStatus()
        } finally {
            this.twoFactorLoading = false
        }
    }

    async handleTOTPDisable() {
        if (!this.totpDisableCode) {
            this.portal.showNotification('error', '请输入当前验证码')
            return
        }
        this.twoFactorLoading = true
        try {
            await api.accountTOTPDisable({ code: this.totpDisableCode })
            this.portal.showNotification('success', 'TOTP 二次验证已禁用')
            this.totpDisableCode = ''
            await this.loadTwoFactorStatus()
            if (!this.twoFactorEnabled) {
                await this.handleTOTPBegin()
            }
        } finally {
            this.twoFactorLoading = false
        }
    }

    async copyTOTPSecret() {
        const ok = await copyToClipboard(this.totpSetup.secret)
        this.portal.showNotification(ok ? 'success' : 'error', ok ? '密钥已复制到剪贴板' : '复制失败，请手动复制')
    }

    async handleChangePassword() {
        if (!this.passwordForm.newPassword) {
            this.portal.showNotification('error', '请输入新密码')
            return
        }
        if (this.passwordForm.newPassword !== this.passwordForm.confirmPassword) {
            this.portal.showNotification('error', '两次输入的密码不一致')
            return
        }
        if (this.passwordForm.newPassword.length < 6) {
            this.portal.showNotification('error', '密码长度至少 6 位')
            return
        }
        this.passwordLoading = true
        try {
            await api.accountPasswordChange({
                oldPassword: this.passwordForm.oldPassword,
                newPassword: this.passwordForm.newPassword
            })
            this.portal.showNotification('success', '密码修改成功')
            this.passwordForm = { oldPassword: '', newPassword: '', confirmPassword: '' }
        } finally {
            this.passwordLoading = false
        }
    }
}

export default toNative(AccountPassword)
</script>

<template>
  <div class="card mb-4">
    <div class="card-toolbar">
      <div class="hidden md:flex items-center justify-between gap-3">
        <div class="flex items-center gap-3 min-w-0">
          <div class="page-icon bg-blue-500">
            <i class="fas fa-lock text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">账号安全</h1>
            <p class="text-xs text-slate-500 truncate">更新登录密码，并管理密码登录的二次验证</p>
          </div>
        </div>
        <button
          type="button"
          :disabled="passwordLoading || !passwordForm.newPassword || passwordForm.newPassword !== passwordForm.confirmPassword"
          class="btn btn-blue flex-shrink-0"
          @click="handleChangePassword"
        >
          <i v-if="!passwordLoading" class="fas fa-check"></i>
          <i v-else class="fas fa-spinner fa-spin"></i>
          确认修改
        </button>
      </div>

      <div class="flex md:hidden items-center justify-between gap-3">
        <div class="flex items-center gap-3 min-w-0 flex-1">
          <div class="page-icon bg-blue-500 flex-shrink-0">
            <i class="fas fa-lock text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">账号安全</h1>
            <p class="text-xs text-slate-500 truncate">密码与二次验证</p>
          </div>
        </div>
        <button
          type="button"
          :disabled="passwordLoading || !passwordForm.newPassword || passwordForm.newPassword !== passwordForm.confirmPassword"
          class="btn btn-blue flex-shrink-0"
          @click="handleChangePassword"
        >
          <i v-if="!passwordLoading" class="fas fa-check"></i>
          <i v-else class="fas fa-spinner fa-spin"></i>
          修改
        </button>
      </div>
    </div>

    <div class="card-body space-y-4">
      <!-- 修改密码 -->
      <section class="max-w-3xl space-y-4">
        <div>
          <label class="form-label">原密码</label>
          <input v-model="passwordForm.oldPassword" type="password" class="input" placeholder="请输入原密码" autocomplete="current-password" />
        </div>
        <div>
          <label class="form-label">新密码</label>
          <input v-model="passwordForm.newPassword" type="password" class="input" placeholder="请输入新密码（至少 6 位）" autocomplete="new-password" />
        </div>
        <div>
          <label class="form-label">确认密码</label>
          <input v-model="passwordForm.confirmPassword" type="password" class="input" placeholder="请再次输入新密码" autocomplete="new-password" />
        </div>
        <p class="text-xs text-slate-400">修改密码后，已签发的 API Key 将自动失效，请按需重新创建。</p>
      </section>

      <!-- 二次验证 -->
      <section v-if="canViewTOTP" class="max-w-3xl space-y-4 border-t border-slate-200 pt-4">
        <div class="flex items-center justify-between gap-3">
          <div class="flex items-center gap-2">
            <span class="card-icon bg-indigo-100 text-indigo-600"><i class="fas fa-shield-halved"></i></span>
            <div>
              <h2 class="text-sm font-semibold text-slate-700">二次验证（TOTP）</h2>
              <p class="text-xs text-slate-400 mt-0.5">仅账号密码登录触发，不影响 Passkey、OIDC 和 API Token。</p>
            </div>
          </div>
          <span class="badge flex-shrink-0" :class="twoFactorEnabled ? 'badge-success' : 'badge-muted'">
            {{ twoFactorEnabled ? '已启用' : '未启用' }}
          </span>
        </div>

        <!-- 未启用：绑定入口 -->
        <div v-if="!twoFactorEnabled && canEnableTOTP" class="flex flex-col sm:flex-row gap-4 items-start">
          <img v-if="totpSetup.qrcode" :src="totpSetup.qrcode" alt="TOTP QR Code" class="w-40 h-40 rounded-lg border border-slate-200 flex-shrink-0" />
          <div v-else class="w-40 h-40 rounded-lg border border-slate-200 flex-shrink-0 flex items-center justify-center bg-slate-50">
            <i class="fas fa-spinner fa-spin text-slate-400 text-2xl"></i>
          </div>
          <div class="flex-1 min-w-0 space-y-3">
            <div>
              <label class="form-label">无法扫码？手动输入密钥</label>
              <div class="flex gap-2">
                <input class="input font-mono flex-1 min-w-0" readonly :value="totpSetup.secret" />
                <button type="button" class="btn btn-secondary flex-shrink-0" @click="copyTOTPSecret">
                  <i class="fas fa-copy"></i>
                  复制
                </button>
              </div>
            </div>
            <div>
              <label class="form-label">验证码</label>
              <div class="flex gap-2">
                <input v-model="totpSetup.code" type="text" inputmode="numeric" autocomplete="one-time-code" class="input flex-1 font-mono" placeholder="请输入 App 中的 6 位验证码" />
                <button type="button" class="btn btn-indigo flex-shrink-0" :disabled="twoFactorLoading || !totpSetup.code" @click="handleTOTPEnable">
                  <i v-if="twoFactorLoading" class="fas fa-spinner fa-spin"></i>
                  <i v-else class="fas fa-check"></i> 绑定
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- 已启用：禁用入口 -->
        <div v-else-if="twoFactorEnabled && canDisableTOTP" class="space-y-2">
          <label class="form-label">输入当前验证码以禁用</label>
          <div class="flex gap-2">
            <input
              v-model="totpDisableCode"
              type="text"
              inputmode="numeric"
              autocomplete="one-time-code"
              class="input flex-1 font-mono"
              placeholder="请输入认证器中的 6 位验证码"
            />
            <button type="button" class="btn btn-danger flex-shrink-0" :disabled="twoFactorLoading || !totpDisableCode" @click="handleTOTPDisable">
              <i v-if="twoFactorLoading" class="fas fa-spinner fa-spin"></i>
              <i v-else class="fas fa-ban"></i>
              禁用
            </button>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>
