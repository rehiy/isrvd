<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { ApiTokenResult } from '@/service/types'

import { copyToClipboard } from '@/helper/utils'

@Component
class Profile extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    activeTab: 'password' | 'token' = 'password'

    // 密码表单
    passwordForm = {
        oldPassword: '',
        newPassword: '',
        confirmPassword: ''
    }
    passwordLoading = false

    // Token 表单
    tokenForm = {
        name: '',
        expiresIn: 86400 // 默认 24 小时
    }
    tokenLoading = false
    newToken: ApiTokenResult | null = null

    // ─── 过期选项 ───
    expiryOptions = [
        { label: '1 小时', value: 3600 },
        { label: '24 小时', value: 86400 },
        { label: '7 天', value: 604800 },
        { label: '30 天', value: 2592000 },
        { label: '90 天', value: 7776000 },
        { label: '365 天', value: 31536000 }
    ]

    // ─── 方法 ───
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
        }
        this.passwordLoading = false
    }

    async handleCreateToken() {
        if (!this.tokenForm.name.trim()) {
            this.portal.showNotification('error', '请输入令牌名称')
            return
        }
        this.tokenLoading = true
        try {
            const res = await api.accountTokenCreate({
                name: this.tokenForm.name.trim(),
                expiresIn: this.tokenForm.expiresIn
            })
            this.newToken = res.payload ?? null
            this.tokenForm.name = ''
            this.portal.showNotification('success', '令牌创建成功')
        } catch {
            this.portal.showNotification('error', '令牌创建失败')
        }
        this.tokenLoading = false
    }

    async copyToken(token: string) {
        const ok = await copyToClipboard(token)
        this.portal.showNotification(ok ? 'success' : 'error', ok ? '令牌已复制到剪贴板' : '复制失败，请手动复制')
    }

    dismissNewToken() {
        this.newToken = null
    }
}

export default toNative(Profile)
</script>

<template>
  <div>
    <div class="card mb-4">
      <!-- Toolbar Bar -->
      <div class="card-toolbar">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="page-icon bg-blue-500">
              <i class="fas fa-user-circle text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">个人设置</h1>
              <p class="text-xs text-slate-500">修改密码、管理 API 令牌</p>
            </div>
          </div>
          <div class="tab-group">
            <button type="button" :class="['tab-btn', activeTab === 'password' ? 'tab-btn-active text-blue-600' : 'tab-btn-inactive']" @click="activeTab = 'password'">
              <i class="fas fa-lock"></i>修改密码
            </button>
            <button type="button" :class="['tab-btn', activeTab === 'token' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="activeTab = 'token'">
              <i class="fas fa-key"></i>API 令牌
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="flex md:hidden items-center gap-3">
          <div class="page-icon bg-blue-500">
            <i class="fas fa-user-circle text-white"></i>
          </div>
          <div class="min-w-0 flex-1">
            <h1 class="text-lg font-semibold text-slate-800 truncate">个人设置</h1>
            <p class="text-xs text-slate-500 truncate">修改密码、管理 API 令牌</p>
          </div>
        </div>
        <!-- 移动端 Tab -->
        <div class="tab-group md:hidden mt-3">
          <button type="button" :class="['tab-btn flex-1 justify-center', activeTab === 'password' ? 'tab-btn-active text-blue-600' : 'tab-btn-inactive']" @click="activeTab = 'password'">
            <i class="fas fa-lock"></i>修改密码
          </button>
          <button type="button" :class="['tab-btn flex-1 justify-center', activeTab === 'token' ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="activeTab = 'token'">
            <i class="fas fa-key"></i>API 令牌
          </button>
        </div>
      </div>

      <!-- 修改密码 -->
      <div v-if="activeTab === 'password'" class="p-4 md:p-6">
        <form class="max-w-3xl space-y-4" @submit.prevent="handleChangePassword">
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

          <div class="flex items-center gap-3 pt-2">
            <button
              type="submit"
              :disabled="passwordLoading || !passwordForm.newPassword"
              class="btn btn-blue"
            >
              <i v-if="!passwordLoading" class="fas fa-check"></i>
              <i v-else class="fas fa-spinner fa-spin"></i>
              确认修改
            </button>
          </div>
        </form>
      </div>

      <!-- API 令牌 -->
      <div v-else-if="portal.hasPerm('POST /api/account/token')" class="p-4 md:p-6">
        <!-- 新令牌提示 -->
        <div v-if="newToken" class="mb-4 rounded-xl border border-emerald-200 bg-emerald-50 p-4">
          <div class="flex items-start gap-3">
            <div class="page-icon bg-emerald-500">
              <i class="fas fa-check text-white text-sm"></i>
            </div>
            <div class="flex-1 min-w-0">
              <h3 class="text-sm font-semibold text-emerald-800 mb-1">令牌创建成功</h3>
              <p class="text-xs text-emerald-700 mb-3">请立即复制保存，此令牌仅显示一次：</p>
              <div class="flex flex-col sm:flex-row gap-2">
                <code class="block flex-1 min-w-0 max-h-28 overflow-y-auto px-3 py-2 bg-white rounded-lg text-xs font-mono leading-relaxed text-slate-700 break-all border border-emerald-200">{{ newToken.token }}</code>
                <button class="btn btn-emerald self-start sm:self-stretch flex-shrink-0 text-xs" @click="copyToken(newToken.token)">
                  <i class="fas fa-copy"></i>复制
                </button>
              </div>
            </div>
            <button class="btn-icon btn-icon-emerald" @click="dismissNewToken()">
              <i class="fas fa-times"></i>
            </button>
          </div>
        </div>

        <form class="max-w-3xl space-y-4" @submit.prevent="handleCreateToken">
          <div>
            <label class="form-label">令牌名称</label>
            <input v-model="tokenForm.name" type="text" class="input" placeholder="如：CI/CD Pipeline" maxlength="64" />
            <p class="text-xs text-slate-400 mt-1">用于标识令牌用途，最长 64 字符</p>
          </div>

          <div>
            <label class="form-label">有效期</label>
            <select v-model="tokenForm.expiresIn" class="input">
              <option v-for="opt in expiryOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
            </select>
          </div>

          <div class="flex items-center gap-3 pt-2">
            <button
              type="submit"
              :disabled="tokenLoading || !tokenForm.name.trim()"
              class="btn btn-emerald"
            >
              <i v-if="!tokenLoading" class="fas fa-plus"></i>
              <i v-else class="fas fa-spinner fa-spin"></i>
              创建令牌
            </button>
          </div>
        </form>

        <!-- 使用说明 -->
        <div class="mt-6 rounded-xl border border-slate-200 bg-slate-50 p-4">
          <h3 class="text-xs font-semibold text-slate-600 mb-3 flex items-center gap-1.5">
            <i class="fas fa-circle-info text-slate-400"></i>使用说明
          </h3>
          <div class="space-y-3">
            <div>
              <p class="text-xs font-medium text-slate-600 mb-1">调用方式</p>
              <p class="text-xs text-slate-500">在 HTTP 请求头中添加：<code class="px-1.5 py-0.5 bg-white rounded text-slate-700">Authorization: Bearer &lt;token&gt;</code></p>
            </div>
            <div>
              <p class="text-xs font-medium text-slate-600 mb-1">安全提示</p>
              <ul class="text-xs text-slate-500 space-y-0.5">
                <li>• 令牌创建后仅显示一次，请立即复制保存</li>
                <li>• 修改密码后，所有令牌将自动失效</li>
                <li>• 如需更换令牌，请重新创建新令牌</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
      <div v-else class="p-4 md:p-6">
        <div class="flex flex-col items-center justify-center py-12 text-slate-400">
          <i class="fas fa-lock text-4xl mb-3"></i>
          <p class="text-sm">无权限创建 API 令牌</p>
        </div>
      </div>
    </div>
  </div>
</template>
