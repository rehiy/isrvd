<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'

@Component
class AccountPassword extends Vue {
    portal = usePortal()

    passwordForm = {
        oldPassword: '',
        newPassword: '',
        confirmPassword: ''
    }
    passwordLoading = false

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
        } catch (e: unknown) {
            const err = e as { response?: { data?: { message?: string } } }
            this.portal.showNotification('error', err.response?.data?.message || '密码修改失败')
        } finally {
            this.passwordLoading = false
        }
    }
}

export default toNative(AccountPassword)
</script>

<template>
  <div class="card">
    <div class="card-toolbar">
      <div class="flex items-center justify-between w-full gap-3">
        <div class="flex items-center gap-3 min-w-0">
          <div class="page-icon bg-blue-500">
            <i class="fas fa-lock text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">修改密码</h1>
            <p class="text-xs text-slate-500 truncate">更新当前账户的登录密码</p>
          </div>
        </div>
        <button
          type="submit"
          form="password-form"
          :disabled="passwordLoading || !passwordForm.newPassword"
          class="btn btn-blue flex-shrink-0"
        >
          <i v-if="!passwordLoading" class="fas fa-check"></i>
          <i v-else class="fas fa-spinner fa-spin"></i>
          确认修改
        </button>
      </div>
    </div>

    <div class="card-body">
      <form id="password-form" class="max-w-3xl space-y-4" @submit.prevent="handleChangePassword">
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
          <input v-model="passwordForm.confirmPassword" type="password" class="input" placeholder="请输入确认密码" autocomplete="new-password" />
        </div>
      </form>
    </div>
  </div>
</template>
