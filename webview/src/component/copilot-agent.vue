<script setup lang="ts">
import { HttpAgent } from '@ag-ui/client'
import { CopilotKitProvider } from '@copilotkit/vue'

import { usePortal } from '@/stores'

import { absUrl } from '@/service/client'

import CopilotAgentInner from '@/component/copilot-agent-inner.vue'

const portal = usePortal()

// HttpAgent 直连后端 /api/agui，不经 CopilotKit 的 Node 运行时。
// Provider 必须在应用根同步挂载：composable 的 provide/inject 依赖它，
// 异步加载会导致子树注入失败而整页崩溃。
const agents = { default: new HttpAgent({ url: absUrl('agui') }) }

// JWT 随登录态变化，经 headers prop 注入每个 agent 请求（core 值优先级最高）
function authHeaders(): Record<string, string> {
    const token = portal.token || ''
    return token ? { Authorization: `Bearer ${token}` } : {}
}

function onError(event: { error: Error }) {
    console.error('[copilot]', event.error)
}
</script>

<template>
  <CopilotKitProvider :self-managed-agents="agents" :headers="authHeaders()" :on-error="onError">
    <CopilotAgentInner />
  </CopilotKitProvider>
</template>

<style>
/*
 * 侧栏默认 fixed right-0 top-0 h-100vh z-[1200]，会覆盖顶栏右侧的用户菜单。
 * 这里让它从顶栏（4rem）下方开始，按 data 属性精确命中侧栏元素。
 */
aside[data-copilot-sidebar] {
    top: 4rem;
    height: calc(100vh - 4rem);
    height: calc(100dvh - 4rem);
}

/*
 * 侧栏打开时会给 body 注入 margin-inline-end，压缩整个页面布局。
 * 此处保持覆盖式呈现，主内容不让位。
 */
@media (min-width: 768px) {
    body {
        margin-inline-end: 0 !important;
    }
}

/*
 * CopilotChatInput 的 expanded 布局会把按钮移到第二行；输入内容在紧凑宽度
 * 与展开宽度之间变化时，内部布局判断会产生高度振荡。保持三列单行结构，
 * 让 textarea 自身增高，避免输入框在两种网格高度之间来回切换。
 */
@media (min-width: 768px) {
    /* 统一 CopilotChatInput 的 compact/expanded 两种状态，避免换行时宽度反馈振荡 */
    [data-testid='copilot-chat-input-shell'] [data-layout] {
        grid-template-columns: auto minmax(0, 1fr) auto !important;
        grid-template-rows: auto !important;
        align-items: center !important;
    }

    [data-testid='copilot-chat-input-shell'] [data-layout] > :nth-child(1),
    [data-testid='copilot-chat-input-shell'] [data-layout] > :nth-child(2),
    [data-testid='copilot-chat-input-shell'] [data-layout] > :nth-child(3) {
        grid-row-start: 1 !important;
    }

    [data-testid='copilot-chat-input-shell'] [data-layout] > :nth-child(1) {
        grid-column: 1 !important;
    }

    [data-testid='copilot-chat-input-shell'] [data-layout] > :nth-child(2) {
        grid-column: 2 !important;
        grid-column-end: auto !important;
    }

    [data-testid='copilot-chat-input-shell'] [data-layout] > :nth-child(3) {
        grid-column: 3 !important;
    }

    [data-testid='copilot-chat-input-shell'] textarea {
        padding-left: 1.25rem !important;
        padding-right: 1.25rem !important;
    }
}
</style>
