<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { AllSettings } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'
import { buildInstallScript, isMarketplaceInstallPayload } from '@/helper/marketplace'
import type { MarketplaceInstallPayload } from '@/helper/marketplace'

interface ScriptPreview {
    appKey: string
    appTitle: string
    instance: string
    version: string
    script: string
}

@Component({})
class Marketplace extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── Refs ───
    declare $refs: { iframe?: HTMLIFrameElement }

    // ─── 数据属性 ───
    loading = true
    iframeUrl = ''
    iframeOrigin = ''
    preview: ScriptPreview | null = null
    copied = false

    private messageHandler: ((e: MessageEvent) => void) | null = null

    // ─── 生命周期 ───
    async mounted() {
        await this.loadUrl()
        this.messageHandler = this.onMessage.bind(this)
        window.addEventListener('message', this.messageHandler)
    }

    unmounted() {
        if (this.messageHandler) {
            window.removeEventListener('message', this.messageHandler)
            this.messageHandler = null
        }
    }

    // ─── 方法 ───
    async loadUrl() {
        this.loading = true
        try {
            const res = await api.getSettings()
            const payload = res.payload as AllSettings
            const url = payload.marketplace?.url || ''
            this.iframeUrl = url
            if (url) {
                try {
                    this.iframeOrigin = new URL(url).origin
                } catch {
                    this.iframeOrigin = ''
                }
            }
        } catch (e) {
            this.actions.showNotification('error', '加载应用市场配置失败')
        }
        this.loading = false
    }

    onMessage(e: MessageEvent) {
        // 非 marketplace 协议的消息一律忽略（浏览器扩展/其它 iframe 也会发 message）
        if (!isMarketplaceInstallPayload(e.data)) return

        // 首选校验：消息来源窗口必须是当前 iframe
        const iframeWin = this.$refs.iframe?.contentWindow
        if (iframeWin && e.source !== iframeWin) {
            return
        }

        // 次要校验：若配置 URL 能解析出 origin，则 origin 应一致（不一致时给出告警但仍处理，
        // 以兼容 sandbox iframe 中 origin 为 "null" 的情况）
        if (this.iframeOrigin && e.origin !== this.iframeOrigin && e.origin !== 'null') {
            console.warn('[marketplace] message origin mismatch:', e.origin, 'expected:', this.iframeOrigin)
        }

        const payload = e.data as MarketplaceInstallPayload
        const script = buildInstallScript(payload)
        this.preview = {
            appKey: payload.app.key,
            appTitle: payload.app.title || payload.app.name || payload.app.key,
            instance: payload.instance.name,
            version: payload.version.value,
            script,
        }
        this.copied = false
    }

    async copyScript() {
        if (!this.preview) return
        try {
            await navigator.clipboard.writeText(this.preview.script)
            this.copied = true
            setTimeout(() => { this.copied = false }, 1500)
        } catch {
            const ta = document.createElement('textarea')
            ta.value = this.preview.script
            document.body.appendChild(ta)
            ta.select()
            try {
                document.execCommand('copy')
                this.copied = true
                setTimeout(() => { this.copied = false }, 1500)
            } catch {
                this.actions.showNotification('error', '复制失败，请手动选中复制')
            }
            document.body.removeChild(ta)
        }
    }

    closePreview() {
        this.preview = null
        this.copied = false
    }

    openSettings() {
        this.$router.push('/system/settings')
    }
}

export default toNative(Marketplace)
</script>

<template>
  <div>
    <div class="card mb-4">
      <!-- Toolbar -->
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3">
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-amber-500 flex items-center justify-center">
              <i class="fas fa-store text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">应用市场</h1>
              <p class="text-xs text-slate-500">浏览并安装容器化应用，安装事件通过 postMessage 接收</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <button type="button" @click="loadUrl()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button type="button" @click="openSettings()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-gear"></i>配置
            </button>
          </div>
        </div>
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="w-9 h-9 rounded-lg bg-amber-500 flex items-center justify-center flex-shrink-0">
              <i class="fas fa-store text-white"></i>
            </div>
            <div class="min-w-0 flex-1">
              <h1 class="text-base font-semibold text-slate-800 truncate">应用市场</h1>
            </div>
          </div>
          <div class="flex items-center gap-2 flex-shrink-0">
            <button type="button" @click="loadUrl()" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 flex items-center justify-center transition-colors" title="刷新">
              <i class="fas fa-rotate"></i>
            </button>
            <button type="button" @click="openSettings()" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 flex items-center justify-center transition-colors" title="配置">
              <i class="fas fa-gear"></i>
            </button>
          </div>
        </div>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <!-- Empty -->
      <div v-else-if="!iframeUrl" class="flex flex-col items-center justify-center py-20 px-4 text-center">
        <div class="w-16 h-16 rounded-2xl bg-amber-100 flex items-center justify-center mb-4">
          <i class="fas fa-store text-amber-500 text-2xl"></i>
        </div>
        <h2 class="text-base font-semibold text-slate-800 mb-1">尚未配置应用市场</h2>
        <p class="text-sm text-slate-500 mb-4">请前往「系统设置 → 应用市场」配置站点 URL</p>
        <button type="button" @click="openSettings()" class="px-4 py-2 rounded-lg bg-blue-500 hover:bg-blue-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
          <i class="fas fa-gear"></i>前往配置
        </button>
      </div>

      <!-- Iframe -->
      <div v-else class="p-0">
        <iframe
          ref="iframe"
          :src="iframeUrl"
          class="w-full border-0 rounded-b-2xl"
          style="height: calc(100vh - 10rem);"
          referrerpolicy="no-referrer"
          sandbox="allow-scripts allow-same-origin allow-popups allow-forms allow-downloads allow-modals"
        ></iframe>
      </div>
    </div>

    <!-- 安装脚本预览弹窗 -->
    <div v-if="preview" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50" @click.self="closePreview()">
      <div class="bg-white rounded-2xl shadow-xl w-full max-w-3xl max-h-[90vh] flex flex-col">
        <div class="flex items-center justify-between px-5 py-3 border-b border-slate-200">
          <div class="flex items-center gap-3 min-w-0">
            <div class="w-9 h-9 rounded-lg bg-emerald-500 flex items-center justify-center flex-shrink-0">
              <i class="fas fa-terminal text-white"></i>
            </div>
            <div class="min-w-0">
              <h2 class="text-base font-semibold text-slate-800 truncate">安装 {{ preview.appTitle }}</h2>
              <p class="text-xs text-slate-500 truncate">实例 {{ preview.instance }} · 版本 {{ preview.version }}</p>
            </div>
          </div>
            <button type="button" @click="closePreview()" class="w-8 h-8 rounded-lg hover:bg-slate-50 text-slate-500 flex items-center justify-center">
            <i class="fas fa-times"></i>
          </button>
        </div>

        <div class="px-5 py-3 bg-amber-50 border-b border-amber-100 text-xs text-amber-800 flex items-start gap-2">
          <i class="fas fa-triangle-exclamation mt-0.5 flex-shrink-0"></i>
          <span>复制下方脚本到目标服务器以 <code class="px-1 py-0.5 rounded bg-white/60">bash</code> 执行即可完成安装；脚本依赖 <code class="px-1 py-0.5 rounded bg-white/60">docker</code> / <code class="px-1 py-0.5 rounded bg-white/60">curl</code> / <code class="px-1 py-0.5 rounded bg-white/60">unzip</code>。</span>
        </div>

        <div class="flex-1 overflow-auto p-4">
          <pre class="text-xs font-mono bg-slate-900 text-slate-100 rounded-lg p-4 whitespace-pre overflow-auto leading-5">{{ preview.script }}</pre>
        </div>

        <div class="flex items-center justify-end gap-2 px-5 py-3 border-t border-slate-200">
          <button type="button" @click="closePreview()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium transition-colors">
            关闭
          </button>
          <button type="button" @click="copyScript()" class="px-3 py-1.5 rounded-lg bg-blue-500 hover:bg-blue-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
            <i :class="['fas', copied ? 'fa-check' : 'fa-copy']"></i>
            {{ copied ? '已复制' : '复制脚本' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
