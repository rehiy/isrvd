<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { PasskeyCredential } from '@/service/types'

import { registerPasskey, isWebAuthnSupported } from '@/helper/webauthn'

import BaseModal from '@/component/modal.vue'

@Component({
    components: { BaseModal }
})
class AccountPasskeys extends Vue {
    portal = usePortal()

    passkeyLoading = false
    passkeyCredentials: Array<PasskeyCredential> = []

    // ─── 注册 ───
    showRegisterDialog = false
    registerLoading = false
    registerDisplayName = ''

    // ─── 重命名 ───
    renamingCredId = ''
    renameValue = ''

    mounted() {
        this.loadPasskeyCredentials()
    }

    async loadPasskeyCredentials() {
        this.passkeyLoading = true
        try {
            const { payload } = await api.accountPasskeyListCredentials()
            this.passkeyCredentials = payload || []
        } finally {
            this.passkeyLoading = false
        }
    }

    openRegisterDialog() {
        if (!isWebAuthnSupported()) {
            this.portal.showNotification('error', '当前浏览器或环境不支持 Passkey（需要 HTTPS 且浏览器支持 WebAuthn）')
            return
        }
        this.registerDisplayName = ''
        this.showRegisterDialog = true
    }

    closeRegisterDialog() {
        this.showRegisterDialog = false
        this.registerDisplayName = ''
    }

    async handleRegisterPasskey() {
        this.registerLoading = true
        try {
            await registerPasskey(this.registerDisplayName || undefined)
            this.portal.showNotification('success', 'Passkey 绑定成功！')
            this.showRegisterDialog = false
            await this.loadPasskeyCredentials()
        } finally {
            this.registerLoading = false
        }
    }

    // ─── 重命名 ───
    startRename(cred: PasskeyCredential) {
        this.renamingCredId = cred.idBase64
        this.renameValue = cred.displayName || ''
    }

    cancelRename() {
        this.renamingCredId = ''
        this.renameValue = ''
    }

    async confirmRename(cred: PasskeyCredential) {
        const name = this.renameValue.trim()
        if (!name || name === cred.displayName) {
            this.cancelRename()
            return
        }
        try {
            await api.accountPasskeyRenameCredential(cred.idBase64, name)
            cred.displayName = name
            this.portal.showNotification('success', '凭证已重命名')
        } finally {
            this.cancelRename()
        }
    }

    // ─── 删除 ───
    handleDeletePasskey(credentialId: string) {
        this.portal.showConfirm({
            title: '删除 Passkey',
            message: '确定要删除这个 Passkey 凭证吗？',
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                try {
                    await api.accountPasskeyDeleteCredential(credentialId)
                    this.portal.showNotification('success', 'Passkey 凭证已删除')
                    await this.loadPasskeyCredentials()
                } catch {}
            }
        })
    }}

export default toNative(AccountPasskeys)
</script>

<template>
  <div class="card">
    <div class="card-toolbar">
      <div class="flex items-center justify-between w-full gap-3">
        <div class="flex items-center gap-3 min-w-0">
          <div class="page-icon bg-purple-500">
            <i class="fas fa-fingerprint text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">Passkey</h1>
            <p class="text-xs text-slate-500 truncate">绑定或移除用于免密登录的 Passkey</p>
          </div>
        </div>
        <button
          type="button"
          class="btn btn-purple flex-shrink-0"
          :disabled="passkeyLoading"
          @click="openRegisterDialog"
        >
          <i class="fas fa-plus mr-2"></i>
          绑定新 Passkey
        </button>
      </div>
    </div>

    <div class="card-body">
      <div v-if="passkeyLoading" class="card-body">
        <div class="empty-state">
          <div class="w-12 h-12 spinner mb-3"></div>
          <p class="text-slate-500">加载中...</p>
        </div>
      </div>

      <div v-else-if="passkeyCredentials.length === 0" class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-fingerprint text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">暂无 Passkey</p>
        <p class="text-sm text-slate-400">点击上方「绑定新 Passkey」开始设置</p>
      </div>

      <div v-else class="space-y-4">
        <div
          v-for="cred in passkeyCredentials"
          :key="cred.idBase64"
          class="flex items-center justify-between p-4 bg-slate-50 rounded-xl border border-slate-200"
        >
          <div class="flex items-center gap-4 min-w-0">
            <div class="list-icon bg-purple-100 text-purple-600">
              <i class="fas fa-fingerprint"></i>
            </div>
            <div class="min-w-0">
              <!-- 重命名编辑态 -->
              <div v-if="renamingCredId === cred.idBase64" class="flex items-center gap-2">
                <input
                  v-model="renameValue"
                  class="input input-sm py-0.5 px-2 text-sm w-40"
                  autofocus
                  @keyup.enter="confirmRename(cred)"
                  @keyup.escape="cancelRename"
                />
                <button class="btn-icon btn-icon-emerald" @click="confirmRename(cred)">
                  <i class="fas fa-check text-xs"></i>
                </button>
                <button class="btn-icon btn-icon-slate" @click="cancelRename">
                  <i class="fas fa-times text-xs"></i>
                </button>
              </div>
              <!-- 展示态 -->
              <div v-else class="flex items-center gap-1 group">
                <h4 class="text-sm font-medium text-slate-800 truncate">
                  {{ cred.displayName || 'Passkey #' + cred.idBase64.slice(0, 8) }}
                </h4>
                <button
                  class="btn-icon btn-icon-slate"
                  title="重命名"
                  @click="startRename(cred)"
                >
                  <i class="fas fa-pen text-xs"></i>
                </button>
              </div>
              <div class="flex items-center gap-3 text-xs text-slate-500 mt-1">
                <span>添加于 {{ new Date(cred.addedAt).toLocaleDateString() }}</span>
              </div>
            </div>
          </div>
          <button
            class="btn-icon btn-icon-red flex-shrink-0"
            title="删除"
            @click="handleDeletePasskey(cred.idBase64)"
          >
            <i class="fas fa-trash-alt text-xs"></i>
          </button>
        </div>
      </div>
    </div>

    <!-- 注册弹窗 -->
    <BaseModal
      v-model="showRegisterDialog"
      title="绑定新 Passkey"
      :loading="registerLoading"
      confirm-class="btn-purple"
      @confirm="handleRegisterPasskey"
      @cancel="closeRegisterDialog"
    >
      <div class="space-y-4">
        <div>
          <label class="form-label">凭证名称 <span class="text-slate-400 font-normal">（可选）</span></label>
          <input
            v-model="registerDisplayName"
            type="text"
            class="input"
            placeholder="如：MacBook Touch ID"
            maxlength="50"
            @keyup.enter="handleRegisterPasskey"
          />
        </div>
        <p class="text-sm text-slate-500">
          请确保您的设备支持 Passkey（如 Touch ID、Face ID 或安全密钥）。
        </p>
      </div>
      <template #confirm-text>
        {{ registerLoading ? '绑定中...' : '开始绑定' }}
      </template>
    </BaseModal>
  </div>
</template>
