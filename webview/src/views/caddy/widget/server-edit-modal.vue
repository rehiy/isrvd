<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { CaddyAutomaticHTTPS, CaddyServerCreate, CaddyServerDetail, CaddyServerUpdate } from '@/service/types'

import BaseModal from '@/component/modal.vue'
import OptionMultiSelect from '@/component/option-multi-select.vue'
import ToggleCard from '@/component/toggle-card.vue'

const protocolOptions = [
    { value: 'h1', label: 'H1' },
    { value: 'h2', label: 'H2' },
    { value: 'h2c', label: 'H2C' },
    { value: 'h3', label: 'H3' }
]

type StrictSNIHostMode = 'auto' | 'enabled' | 'disabled'

const defaultFormData = () => ({
    name: '',
    listen: '',
    protocols: ['h1', 'h2', 'h3'] as string[],
    idleTimeout: '',
    readTimeout: '',
    writeTimeout: '',
    maxHeaderBytes: 0,
    strictSNIHost: 'auto' as StrictSNIHostMode,
    autoHTTPSDisabled: false,
    redirectsDisabled: false
})

@Component({
    expose: ['show'],
    components: { BaseModal, OptionMultiSelect, ToggleCard },
    emits: ['success']
})
class ServerEditModal extends Vue {
    portal = usePortal()

    isOpen = false
    loading = false
    editingID = ''
    originalConfig: Partial<CaddyServerUpdate> = {}
    formData = defaultFormData()

    readonly protocolOptions = protocolOptions

    get isEditMode() {
        return Boolean(this.editingID)
    }

    show(server: CaddyServerDetail | null) {
        this.formData = defaultFormData()
        this.editingID = server?.id || ''
        this.originalConfig = server ? this.serverConfig(server) : {}
        if (server) {
            this.formData.name = server.name
            this.formData.listen = (server.listen || []).join('\n')
            this.formData.protocols = server.protocols ? [...server.protocols] : []
            this.formData.idleTimeout = server.idle_timeout || ''
            this.formData.readTimeout = server.read_timeout || ''
            this.formData.writeTimeout = server.write_timeout || ''
            this.formData.maxHeaderBytes = server.max_header_bytes || 0
            this.formData.strictSNIHost = server.strict_sni_host === undefined
                ? 'auto'
                : (server.strict_sni_host ? 'enabled' : 'disabled')
            this.formData.autoHTTPSDisabled = server.automatic_https?.disable || false
            this.formData.redirectsDisabled = server.automatic_https?.disable_redirects || false
        }
        this.isOpen = true
    }

    serverConfig(server: CaddyServerDetail): CaddyServerUpdate {
        return Object.fromEntries(Object.entries(server).filter(([key]) => !['id', 'name', 'routeCount'].includes(key))) as CaddyServerUpdate
    }

    buildPayload(): CaddyServerUpdate | null {
        if (!this.formData.name) {
            this.portal.showNotification('error', '请输入服务名称')
            return null
        }
        const listen = this.formData.listen.split(/[\s,]+/).map(item => item.trim()).filter(Boolean)
        if (!listen.length) {
            this.portal.showNotification('error', '请至少填写一个监听地址')
            return null
        }
        const protocols = [...this.formData.protocols]
        if ((protocols.includes('h2') || protocols.includes('h2c')) && !protocols.includes('h1')) {
            this.portal.showNotification('error', '启用 H2 或 H2C 时必须同时启用 H1')
            return null
        }

        let automaticHTTPS: CaddyAutomaticHTTPS | undefined
        if (this.originalConfig.automatic_https || this.formData.autoHTTPSDisabled || this.formData.redirectsDisabled) {
            automaticHTTPS = {
                disable: this.formData.autoHTTPSDisabled,
                disable_redirects: this.formData.redirectsDisabled
            }
        }

        const payload: CaddyServerUpdate = {
            ...this.originalConfig,
            listen,
            protocols,
            idle_timeout: this.formData.idleTimeout.trim() || undefined,
            read_timeout: this.formData.readTimeout.trim() || undefined,
            write_timeout: this.formData.writeTimeout.trim() || undefined,
            max_header_bytes: this.formData.maxHeaderBytes || undefined,
            automatic_https: automaticHTTPS
        }
        if (this.formData.strictSNIHost === 'auto') {
            delete payload.strict_sni_host
        } else {
            payload.strict_sni_host = this.formData.strictSNIHost === 'enabled'
        }
        return payload
    }

    async handleConfirm() {
        const config = this.buildPayload()
        if (!config) return

        this.loading = true
        try {
            if (this.isEditMode) {
                await api.caddyServerUpdate(this.editingID, config)
                this.portal.showNotification('success', '服务更新成功')
            } else {
                const data: CaddyServerCreate = { name: this.formData.name, ...config }
                await api.caddyServerCreate(data)
                this.portal.showNotification('success', '服务创建成功')
            }
            this.isOpen = false
            this.$emit('success')
        } catch (error: unknown) {
            this.portal.showNotification('error', (error instanceof Error ? error.message : '') || '操作失败')
        } finally {
            this.loading = false
        }
    }
}

export default toNative(ServerEditModal)
</script>

<template>
  <BaseModal v-model="isOpen" :title="isEditMode ? '编辑服务' : '新建服务'" :loading="loading" confirm-class="btn-rose" @confirm="handleConfirm">
    <div class="max-w-3xl space-y-4">
      <div>
        <label class="form-label">服务名称 <span class="text-red-500">*</span></label>
        <input v-model="formData.name" class="input font-mono" :disabled="isEditMode" placeholder="例如 srv0" />
        <p class="text-xs text-slate-400 mt-1">名称创建后不可修改，并会原样保留。</p>
      </div>

      <div>
        <label class="form-label">监听地址 <span class="text-red-500">*</span></label>
        <textarea v-model="formData.listen" rows="3" class="input font-mono text-sm" placeholder=":80&#10;:443"></textarea>
        <p class="text-xs text-slate-400 mt-1">每行一个地址，例如 :80、:443 或 127.0.0.1:8080。</p>
      </div>

      <div>
        <label class="form-label">协议</label>
        <OptionMultiSelect
          v-model="formData.protocols"
          :options="protocolOptions"
          aria-label="服务协议"
          placeholder="请选择服务协议"
          search-placeholder="搜索协议"
          empty-text="未找到匹配协议"
        />
        <p class="text-xs text-slate-400 mt-1">支持多选；未选择时使用 Caddy 默认协议。H2 与 H2C 依赖 H1。</p>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div><label class="form-label">空闲超时</label><input v-model="formData.idleTimeout" class="input" placeholder="例如 5m" /></div>
        <div><label class="form-label">读取超时</label><input v-model="formData.readTimeout" class="input" placeholder="例如 30s" /></div>
        <div><label class="form-label">写入超时</label><input v-model="formData.writeTimeout" class="input" placeholder="例如 30s" /></div>
      </div>

      <div>
        <label class="form-label">最大请求头字节数</label>
        <input v-model.number="formData.maxHeaderBytes" type="number" min="0" class="input" placeholder="0 表示使用默认值" />
      </div>

      <div class="border-t border-slate-200 pt-6 space-y-4">
        <div>
          <label class="form-label">严格校验 SNI Host</label>
          <select v-model="formData.strictSNIHost" class="input">
            <option value="auto">跟随 Caddy 默认规则</option>
            <option value="enabled">启用</option>
            <option value="disabled">禁用</option>
          </select>
          <p class="text-xs text-slate-400 mt-1">默认规则会在配置 TLS 客户端认证时自动启用严格校验。</p>
        </div>
        <ToggleCard v-model="formData.autoHTTPSDisabled" label="禁用自动 HTTPS" desc="关闭该服务的自动证书和 HTTPS 配置。" />
        <ToggleCard v-model="formData.redirectsDisabled" label="禁用 HTTPS 重定向" desc="保留自动 HTTPS，但不将 HTTP 请求重定向到 HTTPS。" />
      </div>
    </div>

    <template #confirm-text>{{ isEditMode ? '保存修改' : '创建服务' }}</template>
  </BaseModal>
</template>
