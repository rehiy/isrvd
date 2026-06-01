<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { CaddyCert, CaddyCertSource, CaddyCertSourceCard } from '@/service/types'

import BaseModal from '@/component/modal.vue'

const SOURCE_CARDS: CaddyCertSourceCard[] = [
    { value: 'file', title: '磁盘文件', desc: '从本地磁盘路径加载证书和私钥文件', icon: 'fa-file-shield', tone: 'indigo' },
    { value: 'pem', title: '内联 PEM', desc: '直接将证书和私钥的 PEM 内容写入配置', icon: 'fa-key', tone: 'emerald' },
    { value: 'automate', title: '自动签发', desc: '由 Caddy 通过 ACME 协议自动申请和续签证书', icon: 'fa-bolt', tone: 'amber' }
]

const TONE_CARD_ACTIVE: Record<string, string> = {
    indigo: 'border-indigo-300 bg-indigo-50 text-indigo-700',
    emerald: 'border-emerald-300 bg-emerald-50 text-emerald-700',
    amber: 'border-amber-300 bg-amber-50 text-amber-700'
}
const TONE_ICON_ACTIVE: Record<string, string> = {
    indigo: 'bg-indigo-100', emerald: 'bg-emerald-100', amber: 'bg-amber-100'
}

const defaultFormData = () => ({
    source: 'file' as CaddyCertSource,
    certificate: '',
    keyContent: '',
    tags: '',
    format: '',
    subject: ''
})

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class CertEditModal extends Vue {
    portal = usePortal()

    isOpen = false
    modalLoading = false
    isEditMode = false
    editingKey = ''

    formData = defaultFormData()

    readonly sourceCards = SOURCE_CARDS

    sourceCardClass(item: CaddyCertSourceCard) {
        const active = this.formData.source === item.value
        // 编辑模式不允许切换 source（避免跨数组删改混乱）
        if (this.isEditMode && !active) {
            return 'option-card option-card-disabled'
        }
        return `option-card ${active ? TONE_CARD_ACTIVE[item.tone] : 'option-card-inactive'}`
    }

    sourceCardIconClass(item: CaddyCertSourceCard) {
        const active = this.formData.source === item.value
        return `option-card-icon ${active ? TONE_ICON_ACTIVE[item.tone] : 'bg-slate-100'}`
    }

    setSource(source: CaddyCertSource) {
        if (this.isEditMode) return
        this.formData.source = source
    }

    show(cert: CaddyCert | null) {
        Object.assign(this.formData, defaultFormData())
        if (cert) {
            this.isEditMode = true
            this.editingKey = cert.key || ''
            this.formData.source = cert.source
            this.formData.certificate = cert.certificate || ''
            this.formData.keyContent = cert.keyContent || ''
            this.formData.tags = (cert.tags || []).join(', ')
            this.formData.format = cert.format || ''
            this.formData.subject = cert.subject || ''
        } else {
            this.isEditMode = false
            this.editingKey = ''
        }
        this.isOpen = true
    }

    buildPayload(): CaddyCert | null {
        const f = this.formData
        const tags = f.tags.split(/[,\s]+/).map(s => s.trim()).filter(Boolean)
        const cert: CaddyCert = { source: f.source }

        switch (f.source) {
            case 'file':
                if (!f.certificate.trim()) {
                    this.portal.showNotification('error', '请填写证书文件路径')
                    return null
                }
                if (!this.isEditMode && !f.keyContent.trim()) {
                    this.portal.showNotification('error', '请填写私钥文件路径')
                    return null
                }
                cert.certificate = f.certificate.trim()
                if (f.keyContent.trim()) cert.keyContent = f.keyContent.trim()
                if (tags.length) cert.tags = tags
                if (f.format.trim()) cert.format = f.format.trim()
                break
            case 'pem':
                if (!f.certificate.trim()) {
                    this.portal.showNotification('error', '请填写证书 PEM 内容')
                    return null
                }
                if (!this.isEditMode && !f.keyContent.trim()) {
                    this.portal.showNotification('error', '请填写私钥 PEM 内容')
                    return null
                }
                cert.certificate = f.certificate
                if (f.keyContent.trim()) cert.keyContent = f.keyContent
                if (tags.length) cert.tags = tags
                break
            case 'automate':
                if (!f.subject.trim()) {
                    this.portal.showNotification('error', '请填写需要自动签发的主机名')
                    return null
                }
                cert.subject = f.subject.trim()
                break
        }
        return cert
    }

    async handleConfirm() {
        const payload = this.buildPayload()
        if (!payload) return

        this.modalLoading = true
        try {
            if (this.isEditMode) {
                await api.caddyCertUpdate(this.editingKey, payload)
                this.portal.showNotification('success', '证书更新成功')
            } else {
                await api.caddyCertCreate(payload)
                this.portal.showNotification('success', '证书创建成功')
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

export default toNative(CertEditModal)
</script>

<template>
  <BaseModal v-model="isOpen" :title="isEditMode ? '编辑证书' : '新建证书'" :loading="modalLoading" confirm-class="btn-cyan" @confirm="handleConfirm">
    <div class="space-y-4 p-1">
      <!-- 来源选择：mode cards，直接平铺 -->
      <div>
        <label class="block section-title">证书来源</label>
        <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
          <button v-for="item in sourceCards" :key="item.value" type="button" :class="sourceCardClass(item)" :disabled="isEditMode && formData.source !== item.value" @click="setSource(item.value)">
            <div class="flex items-center gap-2 mb-1">
              <div :class="sourceCardIconClass(item)"><i class="fas text-sm" :class="item.icon"></i></div>
              <span class="text-sm font-semibold">{{ item.title }}</span>
            </div>
            <div class="text-xs opacity-80 leading-5">{{ item.desc }}</div>
          </button>
        </div>
        <p v-if="isEditMode" class="text-xs text-slate-400 mt-2"><i class="fas fa-circle-info mr-1"></i>编辑时不允许切换证书来源类型</p>
      </div>

      <!-- file -->
      <div v-if="formData.source === 'file'" class="space-y-3">
        <div>
          <label class="form-label">证书路径 <span class="text-red-500">*</span></label>
          <input v-model="formData.certificate" type="text" class="input font-mono text-sm" placeholder="请输入证书路径" />
          <p class="text-xs text-slate-400 mt-1">证书文件绝对路径，例如：/etc/caddy/cert.pem</p>
        </div>
        <div>
          <label class="form-label">
            私钥路径 <span v-if="!isEditMode" class="text-red-500">*</span>
          </label>
          <input v-model="formData.keyContent" type="text" class="input font-mono text-sm" :placeholder="isEditMode ? '留空则保持不变' : '请输入私钥路径'" />
          <p v-if="!isEditMode" class="text-xs text-slate-400 mt-1">私钥文件绝对路径，例如：/etc/caddy/key.pem</p>
        </div>
        <div>
          <label class="form-label">格式</label>
          <input v-model="formData.format" type="text" class="input" placeholder="请输入证书格式（可选）" />
          <p class="text-xs text-slate-400 mt-1">证书格式，留空使用默认 PEM</p>
        </div>
        <div>
          <label class="form-label">标签</label>
          <input v-model="formData.tags" type="text" class="input" placeholder="请输入标签（可选）" />
          <p class="text-xs text-slate-400 mt-1">多个标签用逗号分隔，例如：example,prod</p>
        </div>
      </div>

      <!-- pem -->
      <div v-else-if="formData.source === 'pem'" class="space-y-3">
        <div>
          <label class="form-label">证书 PEM <span class="text-red-500">*</span></label>
          <textarea v-model="formData.certificate" rows="6" class="input font-mono text-xs leading-5" placeholder="请输入证书 PEM 内容"></textarea>
          <p class="text-xs text-slate-400 mt-1">格式：-----BEGIN CERTIFICATE----- ... -----END CERTIFICATE-----</p>
        </div>
        <div>
          <label class="form-label">
            私钥 PEM <span v-if="!isEditMode" class="text-red-500">*</span>
          </label>
          <textarea v-model="formData.keyContent" rows="6" class="input font-mono text-xs leading-5" :placeholder="isEditMode ? '留空则保持不变' : '请输入私钥 PEM 内容'"></textarea>
          <p v-if="!isEditMode" class="text-xs text-slate-400 mt-1">格式：-----BEGIN PRIVATE KEY----- ... -----END PRIVATE KEY-----</p>
        </div>
        <div>
          <label class="form-label">标签</label>
          <input v-model="formData.tags" type="text" class="input" placeholder="请输入标签（可选）" />
          <p class="text-xs text-slate-400 mt-1">多个标签用逗号分隔，例如：example,prod</p>
        </div>
      </div>

      <!-- automate -->
      <div v-else-if="formData.source === 'automate'" class="space-y-3">
        <div>
          <label class="form-label">主机名 <span class="text-red-500">*</span></label>
          <input v-model="formData.subject" type="text" class="input font-mono text-sm" placeholder="请输入主机名" />
          <p class="text-xs text-slate-400 mt-1">Caddy 将通过 ACME 自动为该主机申请并续期证书，例如：example.com</p>
        </div>
      </div>
    </div>

    <template #confirm-text>
      确认{{ isEditMode ? '更新' : '提交' }}
    </template>
  </BaseModal>
</template>
