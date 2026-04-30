<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import api from '@/service/api'
import type { ApisixCreateSSLRequest, ApisixSSL, ApisixUpdateSSLRequest } from '@/service/types'

import BaseModal from '@/component/modal.vue'

const defaultFormData = () => ({
    id: '',
    snisText: '',
    cert: '',
    key: '',
    status: 1
})

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class SSLEditModal extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    isOpen = false
    modalLoading = false
    isEditMode = false
    formData = defaultFormData()

    resetForm() {
        this.formData = defaultFormData()
    }

    async show(ssl: ApisixSSL | null = null) {
        if (ssl) {
            this.isEditMode = true
            this.formData = {
                id: ssl.id || '',
                snisText: (ssl.snis || []).join('\n'),
                cert: '',
                key: '',
                status: ssl.status ?? 1
            }
        } else {
            this.isEditMode = false
            this.resetForm()
        }
        this.isOpen = true
    }

    parseSnis() {
        return this.formData.snisText
            .split(/[\n,]/)
            .map(item => item.trim())
            .filter(Boolean)
    }

    buildPayload(): ApisixCreateSSLRequest | ApisixUpdateSSLRequest {
        const payload: ApisixCreateSSLRequest = {
            snis: this.parseSnis(),
            status: Number(this.formData.status)
        }
        if (this.formData.cert.trim()) payload.cert = this.formData.cert.trim()
        if (this.formData.key.trim()) payload.key = this.formData.key.trim()
        return payload
    }

    async handleConfirm() {
        const snis = this.parseSnis()
        if (snis.length === 0) {
            this.actions.showNotification('error', '至少需要填写一个 SNI 域名')
            return
        }
        if (!this.isEditMode && !this.formData.cert.trim()) {
            this.actions.showNotification('error', '证书内容不能为空')
            return
        }
        if (!this.isEditMode && !this.formData.key.trim()) {
            this.actions.showNotification('error', '私钥内容不能为空')
            return
        }

        this.modalLoading = true
        try {
            const payload = this.buildPayload()
            if (this.isEditMode) {
                await api.apisixUpdateSSL(this.formData.id, payload)
            } else {
                await api.apisixCreateSSL(payload)
            }
            this.isOpen = false
            this.$emit('success')
        } catch (e: unknown) {
            this.actions.showNotification('error', (e instanceof Error ? e.message : '') || '操作失败')
        }
        this.modalLoading = false
    }
}

export default toNative(SSLEditModal)
</script>

<template>
  <BaseModal
    v-model="isOpen"
    :title="isEditMode ? '编辑证书' : '创建证书'"
    :loading="modalLoading"
  >
    <div class="max-w-3xl space-y-4 p-1">
      <div>
        <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">SNI 域名 <span class="text-red-500">*</span></label>
        <textarea v-model="formData.snisText" rows="3" class="input font-mono" placeholder="example.com&#10;*.example.com"></textarea>
        <p class="text-xs text-slate-400 mt-1">每行一个域名，也支持用英文逗号分隔。</p>
      </div>

      <div>
        <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">证书内容 <span v-if="!isEditMode" class="text-red-500">*</span></label>
        <textarea v-model="formData.cert" rows="8" class="input font-mono text-xs" :placeholder="isEditMode ? '留空保持不变' : '-----BEGIN CERTIFICATE-----'" autocomplete="new-password"></textarea>
        <p class="text-xs text-slate-400 mt-1">PEM 格式证书，编辑时留空表示不修改当前证书。</p>
      </div>

      <div>
        <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">私钥内容 <span v-if="!isEditMode" class="text-red-500">*</span></label>
        <textarea v-model="formData.key" rows="8" class="input font-mono text-xs" :placeholder="isEditMode ? '留空保持不变' : '-----BEGIN PRIVATE KEY-----'" autocomplete="new-password"></textarea>
        <p class="text-xs text-slate-400 mt-1">PEM 格式私钥，编辑时留空表示不修改当前私钥。</p>
      </div>

      <div class="grid grid-cols-1 gap-4">
        <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">状态</label>
          <select v-model.number="formData.status" class="input">
            <option :value="1">启用</option>
            <option :value="0">禁用</option>
          </select>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-2">
        <button @click="isOpen = false" class="px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-200 rounded-lg hover:bg-slate-50">取消</button>
        <button @click="handleConfirm()" :disabled="modalLoading" class="px-4 py-2 text-sm font-medium text-white bg-cyan-500 rounded-lg hover:bg-cyan-600 disabled:opacity-50 shadow-sm">
          <i v-if="modalLoading" class="fas fa-spinner fa-spin mr-1"></i>{{ isEditMode ? '保存' : '创建' }}
        </button>
      </div>
    </template>
  </BaseModal>
</template>
