<script lang="ts">
import { Component, Inject, Ref, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { AllSettings } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import ComposeModal from '@/views/compose/widget/compose-modal.vue'

// 应用市场 postMessage 协议：仅本页使用，故就近定义
interface MarketplaceInstallPayload {
    // 协议识别
    source: 'marketplace'
    type: 'install'

    // 业务字段
    url: string                                           // 应用安装包（zip）下载地址
    name: string                                          // 实例名（作为目录名 / compose project 名，需满足 [a-zA-Z0-9][a-zA-Z0-9_.-]*）
    env: Record<string, string | number | boolean>       // 所有 compose 插值变量（含 CONTAINER_NAME / APP_NAME / NETWORK_NAME 等）
    internalOnly?: boolean                                // 可选：仅内网模式，剥离宿主机端口映射（由 APISIX 等网关代理访问）
}

// 校验 postMessage 数据是否为合法的安装 payload
function isMarketplaceInstallPayload(data: unknown): data is MarketplaceInstallPayload {
    if (!data || typeof data !== 'object') return false
    const d = data as Record<string, unknown>
    if (d.source !== 'marketplace' || d.type !== 'install') return false
    if (typeof d.url !== 'string' || !d.url) return false
    if (typeof d.name !== 'string' || !d.name) return false
    if (typeof d.env !== 'object' || d.env === null) return false
    return true
}

@Component({
    components: { ComposeModal }
})
class Marketplace extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── Refs ───
    declare $refs: { iframe?: HTMLIFrameElement }
    @Ref readonly composeModalRef!: InstanceType<typeof ComposeModal>

    // ─── 数据属性 ───
    loading = true
    installing = false
    iframeUrl = ''
    iframeOrigin = ''

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

        this.installFromPayload(e.data as MarketplaceInstallPayload)
    }

    async installFromPayload(payload: MarketplaceInstallPayload) {
        if (this.installing) {
            this.actions.showNotification('error', '已有安装任务进行中，请稍候')
            return
        }
        const appTitle = payload.name
        this.installing = true
        try {
            const res = await api.composeDeployZip({
                url: payload.url,
                name: payload.name,
                env: payload.env,
                internalOnly: payload.internalOnly,
            })
            const items = res.payload?.items || []
            this.actions.showNotification(
                'success',
                `${appTitle} 安装成功，已创建 ${items.length} 个容器`
            )
        } catch (e) {
            const msg = e instanceof Error ? e.message : String(e)
            this.actions.showNotification('error', `${appTitle} 安装失败：${msg}`)
        }
        this.installing = false
    }

    openSettings() {
        this.$router.push('/system/settings')
    }

    openComposeModal() {
        this.composeModalRef?.show()
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
            <button type="button" @click="openComposeModal()" class="px-3 py-1.5 rounded-lg bg-violet-500 hover:bg-violet-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-file-code"></i>Compose 部署
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
            <button type="button" @click="openComposeModal()" class="w-9 h-9 rounded-lg bg-violet-500 hover:bg-violet-600 flex items-center justify-center text-white transition-colors" title="Compose 部署">
              <i class="fas fa-file-code"></i>
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
      <div v-else class="p-0 relative">
        <iframe
          ref="iframe"
          :src="iframeUrl"
          class="w-full border-0 rounded-b-2xl"
          style="height: calc(100vh - 10rem);"
          referrerpolicy="no-referrer"
          sandbox="allow-scripts allow-same-origin allow-popups allow-forms allow-downloads allow-modals"
        ></iframe>
        <div v-if="installing" class="absolute inset-0 bg-black/30 flex items-center justify-center rounded-b-2xl pointer-events-none">
          <div class="bg-white rounded-xl shadow-lg px-4 py-3 flex items-center gap-3">
            <div class="w-6 h-6 spinner"></div>
            <span class="text-sm text-slate-700">正在安装，请稍候...</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Compose 部署弹窗 -->
    <ComposeModal ref="composeModalRef" />
  </div>
</template>
