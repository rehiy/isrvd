<script lang="ts">
import { Component, Prop, Ref, Vue, Watch, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { AllConfig, ComposeMarketplacePick } from '@/service/types'

import BaseModal from '@/component/modal.vue'

// 应用市场 postMessage 协议：仅本组件使用，故就近定义
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

@Component({
    components: { BaseModal },
    emits: ['update:modelValue', 'pick']
})
class MarketplaceModal extends Vue {
    portal = usePortal()
    @Prop({ type: Boolean, default: false }) readonly modelValue!: boolean

    // ─── Refs ───
    @Ref readonly iframeRef!: HTMLIFrameElement

    // ─── 数据属性 ───
    loading = false
    iframeUrl = ''
    iframeOrigin = ''

    private messageHandler: ((e: MessageEvent) => void) | null = null

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
            const res = await api.systemConfig()
            const payload = res.payload as AllConfig
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
            this.portal.showNotification('error', '加载应用市场配置失败')
        }
        this.loading = false
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

    openConfig() {
        this.close()
        this.$router.push('/system/config')
    }
}

export default toNative(MarketplaceModal)
</script>

<template>
  <BaseModal
    :model-value="modelValue"
    :show-footer="false"
    max-width-class="max-w-5xl"
    card-class="h-[calc(100vh-4rem)]"
    body-class="p-0 overflow-hidden"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <template #title>
      <div class="flex items-center gap-3 min-w-0">
        <div class="page-icon bg-amber-500">
          <i class="fas fa-store text-white"></i>
        </div>
        <div class="min-w-0">
          <h1 class="text-lg font-semibold text-slate-800 truncate">应用市场</h1>
          <p class="text-xs text-slate-500 truncate">选择应用后将自动回填到部署表单</p>
        </div>
      </div>
    </template>

    <template #header-actions>
      <button type="button" class="hidden md:flex btn btn-secondary" title="刷新" @click="loadUrl()">
        <i class="fas fa-rotate"></i><span>刷新</span>
      </button>
      <button type="button" class="md:hidden btn btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadUrl()">
        <i class="fas fa-rotate text-sm"></i>
      </button>
    </template>

    <div class="h-full flex-1 min-h-0 overflow-hidden">
      <!-- Loading -->
      <div v-if="loading" class="h-full flex flex-col items-center justify-center">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <!-- Empty -->
      <div v-else-if="!iframeUrl" class="h-full flex flex-col items-center justify-center px-4 text-center">
        <div class="empty-state-icon bg-amber-100">
          <i class="fas fa-store text-amber-500 text-2xl"></i>
        </div>
        <h1 class="text-lg font-semibold text-slate-800 mb-1">尚未配置应用市场</h1>
        <p class="text-sm text-slate-500 mb-4">请前往「系统设置 → 应用市场」配置站点 URL</p>
        <button type="button" class="btn btn-blue" @click="openConfig()">
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
  </BaseModal>
</template>
