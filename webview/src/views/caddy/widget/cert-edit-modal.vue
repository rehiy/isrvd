<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { CaddyCert, CaddyCertSource, CaddyCertSourceCard } from '@/service/types'

import BaseModal from '@/component/modal.vue'

import { usePortal } from '@/stores'

const SOURCE_CARDS: CaddyCertSourceCard[] = [
    { value: 'file', title: '磁盘文件', desc: '通过路径加载证书与私钥', icon: 'fa-file-shield', tone: 'indigo' },
    { value: 'pem', title: '内联 PEM', desc: '直接粘贴 PEM 文本到配置中', icon: 'fa-key', tone: 'emerald' },
    { value: 'automate', title: '自动签发', desc: '由 Caddy 通过 ACME 自动申请', icon: 'fa-bolt', tone: 'amber' }
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
        const base = 'text-left rounded-xl border p-3 transition-colors'
        const active = this.formData.source === item.value
        // 编辑模式不允许切换 source（避免跨数组删改混乱）
        if (this.isEditMode && !active) {
            return `${base} border-slate-200 bg-slate-50 text-slate-300 cursor-not-allowed`
        }
        return `${base} ${active ? TONE_CARD_ACTIVE[item.tone] : 'border-slate-200 bg-white text-slate-600 hover:border-slate-300'}`
    }

    sourceCardIconClass(item: CaddyCertSourceCard) {
        const active = this.formData.source === item.value
        return `w-8 h-8 rounded-lg flex items-center justify-center ${active ? TONE_ICON_ACTIVE[item.tone] : 'bg-slate-100'}`
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
  <BaseModal v-model="isOpen" :title="isEditMode ? '编辑证书' : '添加证书'" :loading="modalLoading" confirm-class="btn-indigo" @confirm="handleConfirm">
    <div class="space-y-4 p-1">
      <!-- 来源选择：mode cards，直接平铺 -->
      <div>
        <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">证书来源</label>
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
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">证书路径 <span class="text-red-500">*</span></label>
          <input v-model="formData.certificate" type="text" class="input font-mono text-sm" placeholder="/etc/caddy/cert.pem" />
        </div>
        <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">
            私钥路径 <span v-if="!isEditMode" class="text-red-500">*</span>
          </label>
          <input v-model="formData.keyContent" type="text" class="input font-mono text-sm" :placeholder="isEditMode ? '留空保持不变' : '/etc/caddy/key.pem'" />
        </div>
        <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">格式</label>
          <input v-model="formData.format" type="text" class="input" placeholder="pem（默认）" />
          <p class="text-xs text-slate-400 mt-1">证书格式，留空使用默认 PEM</p>
        </div>
        <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">标签</label>
          <input v-model="formData.tags" type="text" class="input" placeholder="逗号分隔，如 example,prod" />
        </div>
      </div>

      <!-- pem -->
      <div v-else-if="formData.source === 'pem'" class="space-y-3">
        <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">证书 PEM <span class="text-red-500">*</span></label>
          <textarea v-model="formData.certificate" rows="6" class="input font-mono text-xs leading-5" placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"></textarea>
        </div>
        <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">
            私钥 PEM <span v-if="!isEditMode" class="text-red-500">*</span>
          </label>
          <textarea v-model="formData.keyContent" rows="6" class="input font-mono text-xs leading-5" :placeholder="isEditMode ? '留空保持不变' : '-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----'"></textarea>
        </div>
        <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">标签</label>
          <input v-model="formData.tags" type="text" class="input" placeholder="逗号分隔" />
        </div>
      </div>

      <!-- automate -->
      <div v-else-if="formData.source === 'automate'" class="space-y-3">
        <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">主机名 <span class="text-red-500">*</span></label>
          <input v-model="formData.subject" type="text" class="input font-mono text-sm" placeholder="example.com" />
          <p class="text-xs text-slate-400 mt-1">Caddy 将通过 ACME 自动为该主机申请并续期证书</p>
        </div>
      </div>
    </div>

    <template #confirm-text>
      确认{{ isEditMode ? '更新' : '保存' }}
    </template>
  </BaseModal>
</template>
