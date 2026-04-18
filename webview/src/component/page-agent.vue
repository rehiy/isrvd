<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'
import { PageAgent } from 'page-agent'

@Component
class PageAgentModal extends Vue {
    agent = null as PageAgent | null
    panelVisible = false

    // 绑定后的事件回调引用（用于 add/remove 时保持一致）
    private boundOnDispose: (() => void) | null = null
    private boundOnStatusChange: (() => void) | null = null

    mounted() {
        this.createAgent()
    }

    unmounted() {
        this.destroyAgent()
    }

    // ─── 对外行为 ───

    togglePanel() {
        if (!this.isAgentAlive()) {
            this.createAgent()
            this.showPanel()
            return
        }
        if (this.panelVisible) {
            this.hidePanel()
        } else {
            this.showPanel()
        }
    }

    // ─── 事件回调（普通方法，通过 bind(this) 注册） ───

    onAgentDispose() {
        // 面板 wrapper 已被 remove()，本地状态同步置空
        this.agent = null
        this.panelVisible = false
    }

    onAgentStatusChange() {
        if (this.agent?.status === 'running') {
            this.panelVisible = true
        }
    }

    // ─── 内部：agent 生命周期 ───

    createAgent() {
        this.destroyAgent()

        // 使用本地 JWT token 作为 apiKey，以便请求能通过后端认证中间件
        const token = localStorage.getItem('app-token') || ''
        const agent = new PageAgent({
            baseURL: '/api/agent/proxy',
            apiKey: token,
            model: 'proxy',
            language: 'zh-CN'
        })

        this.boundOnDispose = this.onAgentDispose.bind(this)
        this.boundOnStatusChange = this.onAgentStatusChange.bind(this)
        agent.addEventListener('dispose', this.boundOnDispose)
        agent.addEventListener('statuschange', this.boundOnStatusChange)

        this.agent = agent
        this.panelVisible = false
    }

    destroyAgent() {
        if (!this.agent) return
        if (this.boundOnDispose) {
            this.agent.removeEventListener('dispose', this.boundOnDispose)
        }
        if (this.boundOnStatusChange) {
            this.agent.removeEventListener('statuschange', this.boundOnStatusChange)
        }
        try {
            this.agent.dispose()
        } catch {
            // 可能已由面板内置 X 按钮 dispose
        }
        this.agent = null
        this.panelVisible = false
        this.boundOnDispose = null
        this.boundOnStatusChange = null
    }

    isAgentAlive(): boolean {
        const wrapper = this.agent?.panel?.wrapper
        return !!wrapper && wrapper.isConnected
    }

    // ─── 内部：显隐控制 ───

    showPanel() {
        if (!this.isAgentAlive()) return
        this.agent!.panel.show()
        this.panelVisible = true
    }

    hidePanel() {
        if (!this.isAgentAlive()) return
        this.agent!.panel.hide()
        this.panelVisible = false
    }
}

export default toNative(PageAgentModal)
</script>

<template>
  <button
    @click="togglePanel"
    :title="panelVisible ? '关闭 AI 助手' : '打开 AI 助手'"
    class="btn-ghost px-4 py-2 text-sm font-medium rounded-lg transition-all duration-200 flex items-center gap-2"
    :class="panelVisible
      ? 'text-primary-600 bg-primary-50 hover:bg-primary-100'
      : 'text-slate-600 hover:text-primary-600 hover:bg-primary-50'"
  >
    <i class="fas fa-wand-magic-sparkles"></i>
    <span class="hidden sm:inline">AI 助手</span>
  </button>
</template>
