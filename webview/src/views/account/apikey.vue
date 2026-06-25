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
  <div class="page">
    <div class="page-toolbar">
      <div class="hidden md:flex items-center justify-between">
        <div class="flex items-center gap-3 min-w-0">
          <div class="page-icon bg-blue-500">
            <i class="fas fa-key text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">API Key</h1>
            <p class="text-xs text-slate-500 truncate">创建用于自动化调用的账号令牌</p>
          </div>
        </div>
        <div class="flex items-center gap-2 flex-shrink-0">
          <button
            v-if="portal.hasPerm('POST /api/account/token') && newToken"
            type="button"
            class="btn btn-blue"
            @click="copyToken(newToken.token)"
          >
            <i class="fas fa-copy"></i>
            复制令牌
          </button>
          <button
            v-else-if="portal.hasPerm('POST /api/account/token')"
            type="submit"
            form="apikey-form"
            :disabled="tokenLoading || !tokenForm.name.trim()"
            class="btn btn-blue"
          >
            <i v-if="!tokenLoading" class="fas fa-plus"></i>
            <i v-else class="fas fa-spinner fa-spin"></i>
            创建令牌
          </button>
        </div>
      </div>

      <div class="flex md:hidden items-center justify-between gap-3">
        <div class="flex items-center gap-3 min-w-0 flex-1">
          <div class="page-icon bg-blue-500 flex-shrink-0">
            <i class="fas fa-key text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">API Key</h1>
            <p class="text-xs text-slate-500 truncate">自动化调用令牌</p>
          </div>
        </div>
        <div class="flex items-center gap-1 flex-shrink-0">
          <button
            v-if="portal.hasPerm('POST /api/account/token') && newToken"
            type="button"
            class="btn-icon-sm"
            title="复制令牌"
            @click="copyToken(newToken.token)"
          >
            <i class="fas fa-copy text-sm"></i>
          </button>
          <button
            v-else-if="portal.hasPerm('POST /api/account/token')"
            type="submit"
            form="apikey-form"
            :disabled="tokenLoading || !tokenForm.name.trim()"
            class="btn-icon-sm"
            title="创建令牌"
          >
            <i v-if="!tokenLoading" class="fas fa-plus text-sm"></i>
            <i v-else class="fas fa-spinner fa-spin text-sm"></i>
          </button>
        </div>
      </div>
    </div>

    <div v-if="portal.hasPerm('POST /api/account/token')" class="card-body">
      <div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_320px]">
        <div class="min-w-0 space-y-6">
          <section class="max-w-3xl">
            <form id="apikey-form" class="space-y-4" @submit.prevent="handleCreateToken">
              <div>
                <label class="form-label">令牌名称</label>
                <input v-model="tokenForm.name" type="text" class="input" placeholder="请输入令牌名称" maxlength="64" />
              </div>

              <div>
                <label class="form-label">有效期</label>
                <select v-model="tokenForm.expiresIn" class="input">
                  <option v-for="opt in expiryOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
                </select>
              </div>
            </form>
          </section>

          <section class="max-w-3xl space-y-4 border-t border-slate-200 pt-5">
            <div class="flex items-center justify-between gap-3">
              <div class="flex items-center gap-2 min-w-0">
                <span class="card-icon" :class="newToken ? 'bg-emerald-100 text-emerald-600' : 'bg-slate-100 text-slate-400'">
                  <i class="fas" :class="newToken ? 'fa-check' : 'fa-key'"></i>
                </span>
                <div class="min-w-0">
                  <h2 class="text-sm font-semibold text-slate-700 truncate">生成结果</h2>
                  <p class="text-xs text-slate-400 mt-0.5 truncate">令牌只在创建后显示一次。</p>
                </div>
              </div>
              <button v-if="newToken" class="btn-icon btn-icon-slate flex-shrink-0" title="关闭" @click="dismissNewToken()">
                <i class="fas fa-times"></i>
              </button>
            </div>

            <div v-if="newToken" class="space-y-2">
              <div class="editor-container bg-slate-50">
                <code class="block max-h-32 overflow-auto p-3 font-mono text-xs leading-relaxed text-slate-700 break-all">{{ newToken.token }}</code>
              </div>
            </div>

            <div v-else class="flex items-center gap-3 rounded-lg border border-dashed border-slate-200 px-4 py-5 text-slate-400">
              <i class="fas fa-arrow-up-right-from-square text-sm"></i>
              <span class="text-sm">填写名称后点击创建，生成的令牌会显示在这里。</span>
            </div>
          </section>
        </div>

        <aside class="space-y-5 lg:border-l lg:border-slate-200 lg:pl-6">
          <section class="space-y-3">
            <h2 class="section-title">调用方式</h2>
            <div class="flex items-start gap-2">
              <span class="card-icon bg-indigo-100 text-indigo-600 flex-shrink-0"><i class="fas fa-code"></i></span>
              <code class="text-xs bg-slate-100 px-2 py-1 rounded-lg break-all text-slate-600">Authorization: Bearer &lt;token&gt;</code>
            </div>
          </section>

          <section class="space-y-3 border-t border-slate-200 pt-5">
            <h2 class="section-title">安全策略</h2>
            <div class="space-y-3 text-xs text-slate-500">
              <div class="flex items-start gap-2">
                <i class="fas fa-eye-slash text-slate-400 mt-0.5"></i>
                <span>创建后仅显示一次，请立即复制保存。</span>
              </div>
              <div class="flex items-start gap-2">
                <i class="fas fa-rotate text-slate-400 mt-0.5"></i>
                <span>修改密码后，已签发的 API Key 自动失效。</span>
              </div>
              <div class="flex items-start gap-2">
                <i class="fas fa-clock text-slate-400 mt-0.5"></i>
                <span>按任务选择最短可用有效期。</span>
              </div>
            </div>
          </section>
        </aside>
      </div>
    </div>

    <div v-else class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-lock text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">无权限创建 API Key</p>
        <p class="text-sm text-slate-400">请联系管理员调整账号权限</p>
      </div>
    </div>
  </div>
</template>
