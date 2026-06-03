<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { ApiTokenResult } from '@/service/types'

import { copyToClipboard } from '@/helper/utils'

@Component
class AccountTokens extends Vue {
    portal = usePortal()

    tokenForm = {
        name: '',
        expiresIn: 86400 // 默认 24 小时
    }
    tokenLoading = false
    newToken: ApiTokenResult | null = null

    expiryOptions = [
        { label: '1 小时', value: 3600 },
        { label: '24 小时', value: 86400 },
        { label: '7 天', value: 604800 },
        { label: '30 天', value: 2592000 },
        { label: '90 天', value: 7776000 },
        { label: '365 天', value: 31536000 }
    ]

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
        } finally {
            this.tokenLoading = false
        }
    }

    async copyToken(token: string) {
        const ok = await copyToClipboard(token)
        this.portal.showNotification(ok ? 'success' : 'error', ok ? '令牌已复制到剪贴板' : '复制失败，请手动复制')
    }

    dismissNewToken() {
        this.newToken = null
    }
}

export default toNative(AccountTokens)
</script>

<template>
  <div class="card">
    <div class="card-toolbar">
      <div class="flex items-center justify-between w-full gap-3">
        <div class="flex items-center gap-3 min-w-0">
          <div class="page-icon bg-emerald-500">
            <i class="fas fa-key text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">API Key</h1>
            <p class="text-xs text-slate-500 truncate">创建用于自动化调用的 API 令牌</p>
          </div>
        </div>
        <button
          v-if="portal.hasPerm('POST /api/account/token')"
          type="submit"
          form="apikey-form"
          :disabled="tokenLoading || !tokenForm.name.trim()"
          class="btn btn-emerald flex-shrink-0"
        >
          <i v-if="!tokenLoading" class="fas fa-plus"></i>
          <i v-else class="fas fa-spinner fa-spin"></i>
          创建 API Key
        </button>
      </div>
    </div>

    <div v-if="portal.hasPerm('POST /api/account/token')" class="card-body">
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

      <form id="apikey-form" class="max-w-3xl space-y-4" @submit.prevent="handleCreateToken">
        <div>
          <label class="form-label">令牌名称</label>
          <input v-model="tokenForm.name" type="text" class="input" placeholder="请输入令牌名称" maxlength="64" />
          <p class="text-xs text-slate-400 mt-1">用于标识令牌用途，如 CI/CD Pipeline；最长 64 字符</p>
        </div>

        <div>
          <label class="form-label">有效期</label>
          <select v-model="tokenForm.expiresIn" class="input">
            <option v-for="opt in expiryOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
          </select>
        </div>
      </form>

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

    <div v-else class="card-body">
      <div class="flex flex-col items-center justify-center py-12 text-slate-400">
        <i class="fas fa-lock text-4xl mb-3"></i>
        <p class="text-sm">无权限创建 API Key</p>
      </div>
    </div>
  </div>
</template>
