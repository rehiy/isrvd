<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { ApisixSSLCreate, ApisixSSL, ApisixSSLUpdate } from '@/service/types'

import BaseModal from '@/component/modal.vue'

import { usePortal } from '@/stores'

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
    portal = usePortal()
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

    buildPayload(): ApisixSSLCreate | ApisixSSLUpdate {
        const payload: ApisixSSLCreate = {
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
            this.portal.showNotification('error', '至少需要填写一个 SNI 域名')
            return
        }
        if (!this.isEditMode && !this.formData.cert.trim()) {
            this.portal.showNotification('error', '证书内容不能为空')
            return
        }
        if (!this.isEditMode && !this.formData.key.trim()) {
            this.portal.showNotification('error', '私钥内容不能为空')
            return
        }

        this.modalLoading = true
        try {
            const payload = this.buildPayload()
            if (this.isEditMode) {
                await api.apisixSSLUpdate(this.formData.id, payload)
            } else {
                await api.apisixSSLCreate(payload)
            }
            this.isOpen = false
            this.$emit('success')
        } catch (e: unknown) {
            this.portal.showNotification('error', (e instanceof Error ? e.message : '') || '操作失败')
        } finally {
            this.modalLoading = false
        }
    }
}

export default toNative(SSLEditModal)
</script>

<template>
  <BaseModal
    v-model="isOpen"
    :title="isEditMode ? '编辑证书' : '新建证书'"
    :loading="modalLoading"
    confirm-class="btn-cyan"
    @confirm="handleConfirm"
  >
    <div class="max-w-3xl space-y-4 p-1">
      <div>
        <label class="form-label">SNI 域名 <span class="text-red-500">*</span></label>
        <textarea v-model="formData.snisText" rows="3" class="input font-mono" placeholder="example.com&#10;*.example.com"></textarea>
        <p class="text-xs text-slate-400 mt-1">每行一个域名，也支持用英文逗号分隔。</p>
      </div>

      <div>
        <label class="form-label">证书内容 <span v-if="!isEditMode" class="text-red-500">*</span></label>
        <textarea v-model="formData.cert" rows="8" class="input font-mono text-xs" :placeholder="isEditMode ? '留空保持不变' : '-----BEGIN CERTIFICATE-----'" autocomplete="new-password"></textarea>
        <p class="text-xs text-slate-400 mt-1">PEM 格式证书，编辑时留空表示不修改当前证书。</p>
      </div>

      <div>
        <label class="form-label">私钥内容 <span v-if="!isEditMode" class="text-red-500">*</span></label>
        <textarea v-model="formData.key" rows="8" class="input font-mono text-xs" :placeholder="isEditMode ? '留空保持不变' : '-----BEGIN PRIVATE KEY-----'" autocomplete="new-password"></textarea>
        <p class="text-xs text-slate-400 mt-1">PEM 格式私钥，编辑时留空表示不修改当前私钥。</p>
      </div>

      <div class="grid grid-cols-1 gap-4">
        <div>
          <label class="form-label">状态</label>
          <select v-model.number="formData.status" class="input">
            <option :value="1">启用</option>
            <option :value="0">禁用</option>
          </select>
        </div>
      </div>
    </div>

    <template #confirm-text>
      确认{{ isEditMode ? '更新' : '新建' }}
    </template>
  </BaseModal>
</template>
