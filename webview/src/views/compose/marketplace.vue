<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import { MARKETPLACE_PICK_STORAGE_KEY } from '@/service/types'
import type { ComposeMarketplacePick } from '@/service/types'

// 应用市场 postMessage 协议：仅本页面使用，故就近定义
interface MarketplaceInstallPayload {
    // 协议识别
    source: 'marketplace'
    type: 'install'

    // 业务字段
    name: string      // 实例名（作为目录名 / compose project 名，需满足 [a-zA-Z0-9][a-zA-Z0-9_.-]*）
    compose: string   // 已完成 ${VAR} 插值的完整 compose.yml 文本
    initURL?: string  // 可选：附加运行文件 zip 下载地址
}

// 校验 postMessage 数据是否为合法的安装 payload
function isMarketplaceInstallPayload(data: unknown): data is MarketplaceInstallPayload {
    if (!data || typeof data !== 'object') return false
    const d = data as Record<string, unknown>
    if (d.source !== 'marketplace' || d.type !== 'install') return false
    if (typeof d.name !== 'string' || !d.name) return false
    if (typeof d.compose !== 'string' || !d.compose) return false
    if (d.initURL !== undefined && typeof d.initURL !== 'string') return false
    return true
}

@Component
class ComposeMarketplace extends Vue {
    portal = usePortal()

    // ─── Refs ───
    @Ref readonly iframeRef!: HTMLIFrameElement

    // ─── 数据属性 ───
    iframeUrl = ''
    iframeOrigin = ''
    iframeKey = 0
    iframeLoading = false

    private messageHandler: ((e: MessageEvent) => void) | null = null

    // ─── 生命周期 ───
    mounted() {
        this.loadUrl()
        this.bindEvents()
    }

    unmounted() {
        this.unbindEvents()
    }

    // ─── 方法 ───
    loadUrl() {
        const url = this.portal.marketplaceUrl || ''
        this.iframeUrl = url
        this.iframeLoading = !!url
        if (url) {
            try {
                this.iframeOrigin = new URL(url).origin
            } catch {
                this.iframeOrigin = ''
            }
        } else {
            this.iframeOrigin = ''
        }
    }

    refreshMarketplace() {
        this.loadUrl()
        if (this.iframeUrl) this.iframeKey += 1
    }

    onIframeLoad() {
        this.iframeLoading = false
    }

    bindEvents() {
        if (!this.messageHandler) {
            this.messageHandler = this.onMessage.bind(this)
            window.addEventListener('message', this.messageHandler)
        }
    }

    unbindEvents() {
        if (this.messageHandler) {
            window.removeEventListener('message', this.messageHandler)
            this.messageHandler = null
        }
    }

    onMessage(e: MessageEvent) {
        // 非 marketplace 协议的消息一律忽略（浏览器扩展/其它 iframe 也会发 message）
        if (!isMarketplaceInstallPayload(e.data)) return

        // 首选校验：消息来源窗口必须是当前 iframe
        const iframeWin = this.iframeRef?.contentWindow
        if (iframeWin && e.source !== iframeWin) {
            return
        }

        // 次要校验：若配置 URL 能解析出 origin，则 origin 应一致
        if (this.iframeOrigin && e.origin !== this.iframeOrigin) {
            return
        }

        const payload = e.data as MarketplaceInstallPayload
        const pick: ComposeMarketplacePick = {
            name: payload.name,
            compose: payload.compose,
            initURL: payload.initURL,
        }

        // 暂存选中模板，跳转到部署页后由其消费预填
        try {
            sessionStorage.setItem(MARKETPLACE_PICK_STORAGE_KEY, JSON.stringify(pick))
        } catch {
            this.portal.showNotification('error', '暂存模板失败，请重试')
            return
        }
        this.$router.push('/compose/deploy')
    }

    goConfig() {
        this.$router.push('/system/config')
    }
}

export default toNative(ComposeMarketplace)
</script>

<template>
  <div class="card flex flex-col h-[calc(100vh-7rem)]">
    <!-- Toolbar -->
    <div class="card-toolbar">
      <!-- 桌面端 -->
      <div class="hidden md:flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-amber-500"><i class="fas fa-store text-white"></i></div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">应用市场</h1>
            <p class="text-xs text-slate-500 truncate">选择应用后将自动跳转到部署页并回填模板</p>
          </div>
        </div>
        <div class="flex items-center gap-2 flex-shrink-0">
          <button type="button" class="btn btn-secondary" title="刷新" @click="refreshMarketplace()">
            <i :class="iframeLoading ? 'fas fa-spinner fa-spin' : 'fas fa-rotate'"></i><span>刷新</span>
          </button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="flex md:hidden items-center justify-between">
        <div class="flex items-center gap-3 min-w-0 flex-1">
          <div class="page-icon bg-amber-500"><i class="fas fa-store text-white"></i></div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">应用市场</h1>
            <p class="text-xs text-slate-500 truncate">选择后跳转部署页回填</p>
          </div>
        </div>
        <div class="flex items-center gap-1 flex-shrink-0">
          <button type="button" class="btn btn-secondary w-9 h-9 !px-0" title="刷新" @click="refreshMarketplace()">
            <i :class="iframeLoading ? 'fas fa-spinner fa-spin text-sm' : 'fas fa-rotate text-sm'"></i>
          </button>
        </div>
      </div>
    </div>

    <!-- Body -->
    <div class="card-body relative flex-1 min-h-0 p-0 overflow-hidden">
      <!-- Empty -->
      <div v-if="!iframeUrl" class="h-full flex flex-col items-center justify-center px-4 text-center">
        <div class="empty-state-icon bg-amber-100">
          <i class="fas fa-store text-amber-500 text-2xl"></i>
        </div>
        <h1 class="text-lg font-semibold text-slate-800 mb-1">尚未配置应用市场</h1>
        <p class="text-sm text-slate-500 mb-4">请前往「系统设置 → 应用市场」配置站点 URL</p>
        <button type="button" class="btn btn-blue" @click="goConfig()">
          <i class="fas fa-gear"></i>前往配置
        </button>
      </div>

      <!-- Loading -->
      <div v-if="iframeUrl && iframeLoading" class="absolute inset-0 z-10 bg-white/80">
        <div class="empty-state h-full py-0">
          <div class="w-12 h-12 spinner mb-3"></div>
          <p class="text-slate-500">加载中...</p>
        </div>
      </div>

      <!-- Iframe -->
      <iframe
        v-if="iframeUrl"
        :key="iframeKey"
        ref="iframeRef"
        :src="iframeUrl"
        class="w-full h-full border-0"
        referrerpolicy="no-referrer"
        sandbox="allow-scripts allow-popups allow-forms allow-downloads allow-modals allow-same-origin"
        @load="onIframeLoad()"
      ></iframe>
    </div>
  </div>
</template>
