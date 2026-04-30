<script lang="ts">
import { Component, Inject, Prop, Ref, Vue, Watch, toNative } from 'vue-facing-decorator'

import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import api from '@/service/api'
import type { SystemAllSettings, ComposeMarketplacePick } from '@/service/types'

// 应用市场 postMessage 协议：仅本组件使用，故就近定义
interface MarketplaceInstallPayload {
    // 协议识别
    source: 'marketplace'
    type: 'install'

    // 业务字段
    name: string                                          // 实例名（作为目录名 / compose project 名，需满足 [a-zA-Z0-9][a-zA-Z0-9_.-]*）
    compose: string                                       // 已完成 ${VAR} 插值的完整 compose.yml 文本
    initURL?: string                                      // 可选：附加运行文件 zip 下载地址
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

@Component({
    emits: ['update:modelValue', 'pick']
})
class MarketplaceModal extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions
    @Prop({ type: Boolean, default: false }) readonly modelValue!: boolean

    // ─── Refs ───
    @Ref readonly iframeRef!: HTMLIFrameElement

    // ─── 数据属性 ───
    loading = false
    iframeUrl = ''
    iframeOrigin = ''

    private messageHandler: ((e: MessageEvent) => void) | null = null
    private escHandler: ((e: KeyboardEvent) => void) | null = null

    // ─── 监听器 ───
    @Watch('modelValue', { immediate: true })
    onVisibilityChange(val: boolean) {
        if (val) {
            // 打开时加载 URL 并绑定事件
            this.loadUrl()
            this.bindEvents()
        } else {
            this.unbindEvents()
        }
    }

    // ─── 生命周期 ───
    unmounted() {
        this.unbindEvents()
    }

    // ─── 方法 ───
    async loadUrl() {
        this.loading = true
        try {
            const res = await api.getSettings()
            const payload = res.payload as SystemAllSettings
            const url = payload.marketplace?.url || ''
            this.iframeUrl = url
            if (url) {
                try {
                    this.iframeOrigin = new URL(url).origin
                } catch {
                    this.iframeOrigin = ''
                }
            } else {
                this.iframeOrigin = ''
            }
        } catch {
            this.actions.showNotification('error', '加载应用市场配置失败')
        }
        this.loading = false
    }

    bindEvents() {
        if (!this.messageHandler) {
            this.messageHandler = this.onMessage.bind(this)
            window.addEventListener('message', this.messageHandler)
        }
        if (!this.escHandler) {
            this.escHandler = (e: KeyboardEvent) => {
                if (e.key === 'Escape' && this.modelValue) this.close()
            }
            document.addEventListener('keydown', this.escHandler)
        }
    }

    unbindEvents() {
        if (this.messageHandler) {
            window.removeEventListener('message', this.messageHandler)
            this.messageHandler = null
        }
        if (this.escHandler) {
            document.removeEventListener('keydown', this.escHandler)
            this.escHandler = null
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
        this.$emit('pick', {
            name: payload.name,
            compose: payload.compose,
            initURL: payload.initURL,
        } satisfies ComposeMarketplacePick)
        this.close()
    }

    close() {
        this.$emit('update:modelValue', false)
    }

    handleBackdropClick(e: MouseEvent) {
        if (e.target === e.currentTarget) {
            this.close()
        }
    }

    openSettings() {
        this.close()
        this.$router.push('/system/settings')
    }
}

export default toNative(MarketplaceModal)
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-300 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition duration-200 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="modelValue"
        class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-sm"
        @click="handleBackdropClick"
      >
        <div class="w-full max-w-5xl modal-card animate-scale-in flex flex-col" style="height: calc(100vh - 4rem);">
          <!-- Header -->
          <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3 flex-shrink-0">
            <!-- 桌面端 -->
            <div class="hidden md:flex items-center justify-between">
              <div class="flex items-center gap-3">
                <div class="w-9 h-9 rounded-lg bg-amber-500 flex items-center justify-center">
                  <i class="fas fa-store text-white"></i>
                </div>
                <div>
                  <h3 class="text-lg font-semibold text-slate-800">应用市场</h3>
                  <p class="text-xs text-slate-500">选择应用后将自动回填到部署表单</p>
                </div>
              </div>
              <div class="flex items-center gap-2">
                <button type="button" @click="loadUrl()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
                  <i class="fas fa-rotate"></i>刷新
                </button>
                <button type="button" @click="close()" class="w-8 h-8 flex items-center justify-center rounded-lg text-slate-400 hover:text-slate-600 hover:bg-slate-100 transition-all duration-200">
                  <i class="fas fa-times"></i>
                </button>
              </div>
            </div>
            <!-- 移动端 -->
            <div class="flex md:hidden items-center justify-between">
              <div class="flex items-center gap-3 min-w-0 flex-1">
                <div class="w-9 h-9 rounded-lg bg-amber-500 flex items-center justify-center flex-shrink-0">
                  <i class="fas fa-store text-white"></i>
                </div>
                <div class="min-w-0">
                  <h3 class="text-lg font-semibold text-slate-800 truncate">应用市场</h3>
                  <p class="text-xs text-slate-500 truncate">选择应用后将自动回填到部署表单</p>
                </div>
              </div>
              <div class="flex items-center gap-1.5 flex-shrink-0">
                <button type="button" @click="loadUrl()" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 flex items-center justify-center text-slate-600 transition-colors" title="刷新">
                  <i class="fas fa-rotate text-sm"></i>
                </button>
                <button type="button" @click="close()" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 flex items-center justify-center text-slate-600 transition-colors" title="关闭">
                  <i class="fas fa-times text-sm"></i>
                </button>
              </div>
            </div>
          </div>

          <!-- Body -->
          <div class="flex-1 min-h-0 overflow-hidden">
            <!-- Loading -->
            <div v-if="loading" class="h-full flex flex-col items-center justify-center">
              <div class="w-12 h-12 spinner mb-3"></div>
              <p class="text-slate-500">加载中...</p>
            </div>

            <!-- Empty -->
            <div v-else-if="!iframeUrl" class="h-full flex flex-col items-center justify-center px-4 text-center">
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
            <iframe
              v-else
              ref="iframeRef"
              :src="iframeUrl"
              class="w-full h-full border-0"
              referrerpolicy="no-referrer"
              sandbox="allow-scripts allow-popups allow-forms allow-downloads allow-modals allow-same-origin"
            ></iframe>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
