<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import type { CaddyCert } from '@/service/types'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal }
})
class CertDetailModal extends Vue {
    isOpen = false
    cert: CaddyCert | null = null

    show(cert: CaddyCert) {
        this.cert = cert
        this.isOpen = true
    }

    sourceLabel(source?: string) {
        const map: Record<string, string> = { file: '磁盘文件', pem: '内联 PEM', automate: '自动签发', cached: '已签发缓存' }
        return source ? map[source] || source : '-'
    }

    sourceTagClass(source?: string) {
        const map: Record<string, string> = {
            file: 'bg-indigo-50 text-indigo-700',
            pem: 'bg-emerald-50 text-emerald-700',
            automate: 'bg-amber-50 text-amber-700',
            cached: 'bg-cyan-50 text-cyan-700',
        }
        return source ? map[source] || 'bg-slate-100 text-slate-500' : 'bg-slate-100 text-slate-500'
    }

    certSummary(cert: CaddyCert) {
        if (cert.subject) return cert.subject
        if (cert.source === 'file') return cert.certificate || '-'
        return cert.certificate?.split('\n')[0].trim() || '(空)'
    }

    formatDate(value?: string) {
        if (!value) return '-'
        const date = new Date(value)
        if (Number.isNaN(date.getTime())) return value
        return date.toLocaleString('zh-CN', { hour12: false })
    }

    expireClass(notAfter?: string) {
        if (!notAfter) return 'text-slate-400'
        const days = (new Date(notAfter).getTime() - Date.now()) / 86400000
        if (days < 0) return 'text-red-500 font-medium'
        if (days < 30) return 'text-amber-600 font-medium'
        return 'text-emerald-600 font-medium'
    }

    expireLabel(notAfter?: string) {
        if (!notAfter) return '-'
        const days = Math.floor((new Date(notAfter).getTime() - Date.now()) / 86400000)
        if (days < 0) return `已过期 ${-days} 天`
        if (days === 0) return '今日过期'
        return `${days} 天后到期`
    }

    certificatePreview(cert: CaddyCert) {
        const text = cert.certificate || ''
        if (!text) return '-'
        if (cert.source === 'pem') {
            return text.split('\n').slice(0, 4).join('\n') + (text.split('\n').length > 4 ? '\n...' : '')
        }
        return text
    }

    sourceDescription(cert: CaddyCert) {
        const map: Record<string, string> = {
            file: '从磁盘路径加载证书和私钥，适合宿主机或容器内已有证书文件。',
            pem: '直接在 Caddy 配置中保存 PEM 证书内容，私钥内容不会返回前端。',
            automate: '由 Caddy ACME 自动签发与续期，详情中的有效期依赖运行时缓存。',
            cached: 'Caddy 运行时已签发证书缓存，只读展示，不支持编辑或删除。',
        }
        return map[cert.source] || '-'
    }
}

export default toNative(CertDetailModal)
</script>

<template>
  <BaseModal v-model="isOpen" :show-footer="false" max-width-class="max-w-4xl" body-class="px-6 py-6 overflow-y-auto">
    <template #title>
      <div v-if="cert" class="flex items-center gap-3 min-w-0">
        <div class="page-icon bg-cyan-500">
          <i class="fas fa-certificate text-white"></i>
        </div>
        <div class="min-w-0">
          <h1 class="text-lg font-semibold text-slate-800 truncate">SSL 证书详情</h1>
          <p class="text-xs text-slate-500 truncate">{{ certSummary(cert) }}</p>
        </div>
      </div>
    </template>

    <div v-if="cert" class="space-y-6 text-sm">
      <section>
        <h2 class="section-title">基本信息</h2>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div>
            <label class="form-label">来源</label>
            <div class="detail-value">
              <span :class="sourceTagClass(cert.source)" class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-medium">{{ sourceLabel(cert.source) }}</span>
            </div>
          </div>
          <div>
            <label class="form-label">状态</label>
            <div class="detail-value" :class="cert.source === 'cached' ? 'text-slate-500' : 'text-emerald-600 font-medium'">
              {{ cert.source === 'cached' ? '运行时缓存，只读' : '配置证书，可管理' }}
            </div>
          </div>
          <div>
            <label class="form-label">Key</label>
            <code class="detail-value-mono">{{ cert.key || '-' }}</code>
          </div>
          <div>
            <label class="form-label">格式</label>
            <div class="detail-value">{{ cert.format || 'PEM' }}</div>
          </div>
          <div class="md:col-span-2">
            <label class="form-label">主体</label>
            <code class="detail-value-mono">{{ certSummary(cert) }}</code>
          </div>
        </div>
      </section>

      <section>
        <h2 class="section-title">有效期</h2>
        <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
          <div>
            <label class="form-label">生效时间</label>
            <div class="detail-value">{{ formatDate(cert.notBefore) }}</div>
          </div>
          <div>
            <label class="form-label">过期时间</label>
            <div class="detail-value">{{ formatDate(cert.notAfter) }}</div>
          </div>
          <div>
            <label class="form-label">剩余时间</label>
            <div class="detail-value" :class="expireClass(cert.notAfter)">{{ expireLabel(cert.notAfter) }}</div>
          </div>
        </div>
      </section>

      <section>
        <h2 class="section-title">证书解析</h2>
        <div class="space-y-3">
          <div>
            <label class="form-label">签发机构</label>
            <div class="detail-value">{{ cert.issuer || '-' }}</div>
          </div>
          <div>
            <label class="form-label">SAN 域名</label>
            <div v-if="cert.sans && cert.sans.length" class="flex flex-wrap gap-1.5">
              <span v-for="san in cert.sans" :key="san" class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs font-mono bg-cyan-50 text-cyan-700">{{ san }}</span>
            </div>
            <div v-else class="detail-value text-slate-400">-</div>
          </div>
        </div>
      </section>

      <section>
        <h2 class="section-title">来源配置</h2>
        <div class="space-y-3">
          <div class="rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-xs text-slate-500">
            {{ sourceDescription(cert) }}
          </div>
          <div v-if="cert.source === 'file' || cert.source === 'pem'">
            <label class="form-label">证书内容 / 路径</label>
            <pre class="bg-slate-900 text-slate-100 rounded-lg p-3 text-xs font-mono overflow-auto whitespace-pre-wrap break-all max-h-48">{{ certificatePreview(cert) }}</pre>
          </div>
          <div v-else>
            <label class="form-label">自动签发域名</label>
            <code class="detail-value-mono">{{ cert.subject || '-' }}</code>
          </div>
          <div v-if="cert.source === 'file' || cert.source === 'pem'">
            <label class="form-label">私钥</label>
            <div class="detail-value text-slate-500">响应不返回私钥；编辑时留空表示保持原值</div>
          </div>
          <div v-if="cert.tags && cert.tags.length">
            <label class="form-label">标签</label>
            <div class="flex flex-wrap gap-1.5">
              <span v-for="tag in cert.tags" :key="tag" class="inline-flex items-center px-2 py-0.5 rounded-lg text-xs bg-slate-100 text-slate-600">{{ tag }}</span>
            </div>
          </div>
        </div>
      </section>
    </div>
  </BaseModal>
</template>
