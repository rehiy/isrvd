<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { PasskeyBeginData, PasskeyCredential } from '@/service/types'

import { base64urlToBuffer, bufferToBase64url } from '@/helper/utils'
import BaseModal from '@/component/modal.vue'

@Component({
    components: { BaseModal }
})
class AccountPasskeys extends Vue {
    portal = usePortal()

    passkeyLoading = false
    passkeyCredentials: Array<PasskeyCredential> = []
    showRegisterDialog = false
    registerLoading = false
    registerPrepareLoading = false
    passkeyBeginData: PasskeyBeginData | null = null

    mounted() {
        this.loadPasskeyCredentials()
    }

    async loadPasskeyCredentials() {
        this.passkeyLoading = true
        try {
            const { payload } = await api.accountPasskeyListCredentials()
            this.passkeyCredentials = payload || []
        } catch {
            this.portal.showNotification('error', '加载 Passkey 列表失败')
        } finally {
            this.passkeyLoading = false
        }
    }

    async handleRegisterPasskey() {
        if (!window.PublicKeyCredential || !navigator.credentials?.create) {
            this.portal.showNotification('error', '当前浏览器或环境不支持 Passkey（需要 HTTPS 且浏览器支持 WebAuthn）')
            return
        }

        if (!this.passkeyBeginData) {
            this.portal.showNotification('error', '注册数据未就绪，请稍后重试')
            return
        }

        this.registerLoading = true
        try {
            const beginData = this.passkeyBeginData
            const publicKey = beginData.options.publicKey as any
            const creationOptions: CredentialCreationOptions = {
                publicKey: {
                    ...publicKey,
                    challenge: base64urlToBuffer(publicKey.challenge),
                    user: {
                        ...publicKey.user,
                        id: base64urlToBuffer(publicKey.user.id),
                    },
                    excludeCredentials: (publicKey.excludeCredentials || []).map((c: any) => ({
                        ...c,
                        id: base64urlToBuffer(c.id),
                    })),
                },
            }

            // 直接响应「开始绑定」点击调用，确保 Bitwarden 等扩展能识别用户手势。
            const credential = await navigator.credentials.create(creationOptions) as PublicKeyCredential | null
            if (!credential) {
                throw new Error('用户取消了 Passkey 注册')
            }

            const response = credential.response as AuthenticatorAttestationResponse
            const credentialData = {
                id: credential.id,
                rawId: bufferToBase64url(credential.rawId),
                type: credential.type,
                response: {
                    attestationObject: bufferToBase64url(response.attestationObject),
                    clientDataJSON: bufferToBase64url(response.clientDataJSON),
                },
            }

            await api.accountPasskeyRegisterFinish({
                sessionId: beginData.sessionId,
                credential: credentialData,
            })

            this.portal.showNotification('success', 'Passkey 绑定成功！')
            this.showRegisterDialog = false
            this.passkeyBeginData = null
            await this.loadPasskeyCredentials()
        } catch (e) {
            console.error('Passkey 注册失败:', e)
            const msg = e instanceof Error ? e.message : 'Passkey 注册失败'
            this.portal.showNotification('error', msg)
        } finally {
            this.registerLoading = false
        }
    }

    async openRegisterDialog() {
        if (!window.PublicKeyCredential || !navigator.credentials?.create) {
            this.portal.showNotification('error', '当前浏览器或环境不支持 Passkey（需要 HTTPS 且浏览器支持 WebAuthn）')
            return
        }

        this.passkeyBeginData = null
        this.showRegisterDialog = true
        this.registerPrepareLoading = true

        try {
            const { payload: beginData } = await api.accountPasskeyRegisterBegin()
            if (!beginData) {
                throw new Error('无法开始 Passkey 注册')
            }
            this.passkeyBeginData = beginData
        } catch (e) {
            console.error('获取 Passkey 注册参数失败:', e)
            this.portal.showNotification('error', '获取注册参数失败，请重试')
            this.showRegisterDialog = false
        } finally {
            this.registerPrepareLoading = false
        }
    }

    closeRegisterDialog() {
        this.showRegisterDialog = false
        this.passkeyBeginData = null
    }

    async handleDeletePasskey(credentialId: string) {
        if (!confirm('确定要删除这个 Passkey 凭证吗？')) {
            return
        }

        try {
            await api.accountPasskeyDeleteCredential(credentialId)
            this.portal.showNotification('success', 'Passkey 凭证已删除')
            await this.loadPasskeyCredentials()
        } catch (e) {
            console.error('Passkey 删除失败:', e)
            this.portal.showNotification('error', '删除失败')
        }
    }
}

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
          @click="openRegisterDialog"
          :disabled="passkeyLoading || registerPrepareLoading"
        >
          <i v-if="!registerPrepareLoading" class="fas fa-plus mr-2"></i>
          <i v-else class="fas fa-spinner fa-spin mr-2"></i>
          绑定新 Passkey
        </button>
      </div>
    </div>

    <div class="card-body">
      <div v-if="passkeyLoading" class="text-center py-8">
        <i class="fas fa-spinner fa-spin text-2xl text-slate-400"></i>
        <p class="text-sm text-slate-500 mt-2">加载中...</p>
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
            <div class="w-10 h-10 rounded-lg bg-purple-100 flex items-center justify-center flex-shrink-0">
              <i class="fas fa-fingerprint text-purple-600"></i>
            </div>
            <div class="min-w-0">
              <h4 class="text-sm font-medium text-slate-800 truncate">{{ cred.displayName || 'Passkey #' + cred.idBase64.slice(0, 8) }}</h4>
              <div class="flex items-center gap-3 text-xs text-slate-500 mt-1">
                <span>添加于 {{ new Date(cred.addedAt).toLocaleDateString() }}</span>
                <span>·</span>
                <span>使用次数: {{ cred.authenticator.signCount }}</span>
              </div>
            </div>
          </div>
          <button
            type="button"
            class="btn btn-ghost text-red-500 hover:bg-red-50 flex-shrink-0"
            @click="handleDeletePasskey(cred.idBase64)"
          >
            <i class="fas fa-trash-alt"></i>
          </button>
        </div>
      </div>
    </div>

    <BaseModal
      v-model="showRegisterDialog"
      title="绑定新 Passkey"
      :loading="registerLoading"
      confirm-class="btn-purple"
      @confirm="handleRegisterPasskey"
      @cancel="closeRegisterDialog"
    >
      <p class="text-sm text-slate-500 mb-6">
        点击下方按钮开始绑定 Passkey。请确保您的设备支持 Passkey（如 Touch ID、Face ID 或安全密钥）。
      </p>
      <div v-if="registerPrepareLoading" class="flex items-center gap-2 text-xs text-slate-400 mb-2">
        <i class="fas fa-spinner fa-spin"></i>
        正在准备注册参数...
      </div>
      <template #confirm-text>
        {{ registerLoading ? '绑定中...' : '开始绑定' }}
      </template>
    </BaseModal>
  </div>
</template>
