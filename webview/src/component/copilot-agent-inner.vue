<script setup lang="ts">
import { CopilotChatConfigurationProvider, CopilotSidebar, useAgent, useCopilotAction, useCopilotReadable, useFrontendTool } from '@copilotkit/vue'
import type { CopilotChatLabels } from '@copilotkit/vue'
import { computed, h, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'

import { usePortal } from '@/stores'

import { http } from '@/service/client'

import { executeAgentAPI, registerAgentAPILookup, resetAgentAPICallRefs } from '@/helper/agent-api'
import { blobRead } from '@/helper/agent-blob'
import { systemInstruction, getPageInstruction } from '@/helper/instructions'
import { getPageController, disposePageController } from '@/helper/page-controller'

import AgentAPICard from '@/component/agent-api-card.vue'
import CopilotSidebarToggleBridge from '@/component/copilot-sidebar-bridge.vue'

const route = useRoute()
const portal = usePortal()
const { agent: copilotAgent } = useAgent({ agentId: 'default', updates: [] })

// 侧栏与顶栏入口使用对话端点权限，并检查 Agent 服务可用性。
const hasAgent = computed(() => portal.serviceAvailability.agent && portal.hasPerm('POST /api/agui'))

// 运行结束后清理页面高亮；工具调用轮结束时先保留，给 CopilotKit 的 follow-up
// 继续使用 read 返回的序号，最终文本回答轮结束后才清理。
watch(copilotAgent, (agent, _previous, onCleanup) => {
    if (!agent) return

    const cleanupWhenAnswerFinished = ({ messages }: { messages: readonly { role: string; toolCalls?: unknown[] }[] }) => {
        const lastMessage = messages[messages.length - 1]
        if (lastMessage?.role === 'assistant' && lastMessage.toolCalls?.length) return
        disposePageController()
    }
    const subscription = agent.subscribe({
        onRunFinalized: cleanupWhenAnswerFinished,
        onRunFailed: () => disposePageController(),
        onRunErrorEvent: () => disposePageController(),
    })
    onCleanup(() => subscription.unsubscribe())
}, { immediate: true })

// ─── 页面上下文 ───

const pageInstruction = computed(() => getPageInstruction(route.fullPath || '/'))

useCopilotReadable(
    { description: '当前页面功能说明', value: pageInstruction.value },
    [pageInstruction],
)
useCopilotReadable({ description: 'iSrvd 助手系统提示', value: systemInstruction.trim() })

// ─── 前端工具 ───

useFrontendTool({
    name: 'lookup_api',
    description:
        '查阅 iSrvd 官方 OpenAPI。调用 isrvd_api 或 isrvd_mutation 前必须先查并使用返回的 callRef。' +
        '不传参数返回模块目录；tag 或 q 返回接口列表；path + method 返回字段。仅支持的接口带 callRef，其余按 toolUnsupportedReason 说明操作。',
    parameters: [
        { name: 'tag', type: 'string', description: '模块标签，如 docker、swarm、apisix、caddy、compose、cron、account、system、filer、ssh、overview、shell、agent', required: false },
        { name: 'q', type: 'string', description: '关键词，匹配路径、摘要、operationId', required: false },
        { name: 'path', type: 'string', description: 'API 路径，如 docker/containers 或 /docker/container/{id}', required: false },
        { name: 'method', type: 'string', description: 'HTTP 方法：get / post / put / patch / delete', required: false },
    ],
    handler: async ({ tag, q, path, method }) => {
        try {
            const res = await http.get('agent/openapi', {
                params: {
                    tag: tag || undefined,
                    q: q || undefined,
                    path: path || undefined,
                    method: method || undefined,
                },
            })
            return { success: res?.success ?? true, message: res?.message ?? '', payload: registerAgentAPILookup(res?.payload ?? null) }
        } catch (e) {
            return { success: false, message: e instanceof Error ? e.message : '查阅 API 文档失败' }
        }
    },
})

// 页面操作：读取可交互元素树后按序号执行动作，对应原 page-agent 的自动操作能力
useFrontendTool({
    name: 'page_action',
    description:
        '直接操作当前页面 UI。先调用 action=read 获取带序号的可交互元素列表，' +
        '再用返回的序号执行 click / input / select / scroll。' +
        'javascript 可直接执行 JS，仅在常规动作无法完成时作为兜底。',
    parameters: [
        {
            name: 'action',
            type: 'string',
            description: 'read（读取页面元素）/ click / input / select / scroll / scroll_horizontal / javascript',
            required: true,
        },
        { name: 'index', type: 'number', description: '目标元素序号，来自 read 的结果', required: false },
        { name: 'text', type: 'string', description: 'input 的输入内容，或 select 的选项文本', required: false },
        { name: 'down', type: 'boolean', description: 'scroll 时是否向下滚动', required: false },
        { name: 'right', type: 'boolean', description: 'scroll_horizontal 时是否向右滚动', required: false },
        { name: 'pixels', type: 'number', description: '滚动像素数，默认一屏', required: false },
        { name: 'script', type: 'string', description: 'javascript 动作要执行的 JS 代码', required: false },
    ],
    handler: async ({ action, index, text, down, right, pixels, script }) => {
        const pc = getPageController()
        try {
            if (action === 'read') {
                const state = await pc.getBrowserState()
                return {
                    url: state.url,
                    title: state.title,
                    header: state.header,
                    elements: state.content,
                    footer: state.footer,
                }
            }
            if (action === 'click') {
                return await pc.clickElement(Number(index))
            }
            if (action === 'input') {
                return await pc.inputText(Number(index), String(text ?? ''))
            }
            if (action === 'select') {
                return await pc.selectOption(Number(index), String(text ?? ''))
            }
            if (action === 'scroll') {
                return await pc.scroll({ down: down !== false, numPages: 1, pixels: Number(pixels ?? 0) || undefined, index })
            }
            if (action === 'scroll_horizontal') {
                return await pc.scrollHorizontally({ right: right !== false, pixels: Number(pixels ?? 0), index })
            }
            if (action === 'javascript') {
                if (!script) return { success: false, message: '缺少 script 参数' }
                return await pc.executeJavascript(String(script))
            }
            return { success: false, message: `未知动作：${action}` }
        } catch (e) {
            return { success: false, message: e instanceof Error ? e.message : '页面操作失败' }
        }
    },
})

const isrvdAPIParameters = [
    { name: 'callRef' as const, type: 'string' as const, description: 'lookup_api 返回的会话级调用引用，禁止自行生成', required: true },
    {
        name: 'arguments' as const,
        type: 'string' as const,
        description: '业务参数 JSON 字符串，结构为 {"path":{},"query":{},"body":{}}；只填写 lookup_api 声明的字段',
        required: false,
    },
]

const renderAgentAPICard = (props: unknown, approval = false) =>
    h(AgentAPICard, { ...(props as Record<string, unknown>), approval })

useFrontendTool({
    name: 'isrvd_api',
    description:
        '使用 lookup_api 返回的 callRef 查询资源，仅执行该引用绑定的 GET 操作。' +
        '禁止自行填写或猜测 HTTP 方法和路径；callRef 不存在或过期时重新查阅。禁止用于读取密钥类配置。' +
        '返回结果过大时会自动暂存并返回 blobId，请改用 result_read 按需读取，不要重复调用原接口拉全量。',
    parameters: isrvdAPIParameters,
    handler: args => executeAgentAPI(args, 'query'),
    render: (props: unknown) => renderAgentAPICard(props),
})

useCopilotAction({
    name: 'isrvd_mutation',
    description:
        '使用 lookup_api 返回的 callRef 执行该引用绑定的写操作。禁止自行填写或猜测 HTTP 方法和路径；' +
        'callRef 不存在或过期时重新查阅。调用后会显示解析后的真实目标和脱敏参数，只有用户确认才会执行。',
    parameters: isrvdAPIParameters,
    renderAndWaitForResponse: (props: unknown) => renderAgentAPICard(props, true),
})

// 大结果读取：API 工具返回 truncated 时，按 blobId 从本会话内存中提取所需部分
useFrontendTool({
    name: 'result_read',
    description:
        '读取 API 工具暂存的大结果。blobId 必填；path 可选，提取子字段' +
        '（支持 .key、[0]、[-1]，如 [-1].data.system.memoryUsed）；' +
        'offset/limit 可选，按字符分段读取原文（limit 最大 8000）。',
    parameters: [
        { name: 'blobId', type: 'string', description: 'API 工具返回的暂存 ID', required: true },
        { name: 'path', type: 'string', description: '字段路径，如 [-1].data.system.memoryUsed', required: false },
        { name: 'offset', type: 'number', description: '分段读取的起始字符位置，默认 0', required: false },
        { name: 'limit', type: 'number', description: '分段读取的字符数，默认 4000，最大 8000', required: false },
    ],
    handler: (p: { blobId?: unknown; path?: unknown; offset?: number; limit?: number }) =>
        blobRead(String(p.blobId ?? ''), { path: p.path ? String(p.path) : undefined, offset: p.offset, limit: p.limit }),
})

// CopilotKit 1.70 将 labels 类型声明为英文字面量，运行时实际接受任意字符串。
// 用 unknown 只绕过上游类型限制，不改变配置结构。
const chineseLabels = {
    chatInputPlaceholder: '请输入消息...',
    chatInputToolbarStartTranscribeButtonLabel: '开始语音输入',
    chatInputToolbarCancelTranscribeButtonLabel: '取消语音输入',
    chatInputToolbarFinishTranscribeButtonLabel: '完成语音输入',
    chatInputToolbarAddButtonLabel: '添加图片或文件',
    chatInputToolbarToolsButtonLabel: '工具',
    assistantMessageToolbarCopyCodeLabel: '复制代码',
    assistantMessageToolbarCopyCodeCopiedLabel: '已复制',
    assistantMessageToolbarCopyMessageLabel: '复制消息',
    assistantMessageToolbarThumbsUpLabel: '有帮助',
    assistantMessageToolbarThumbsDownLabel: '没帮助',
    assistantMessageToolbarReadAloudLabel: '朗读',
    assistantMessageToolbarRegenerateLabel: '重新生成',
    userMessageToolbarCopyMessageLabel: '复制消息',
    userMessageToolbarEditMessageLabel: '编辑消息',
    chatDisclaimerText: 'AI 可能出错，请核实重要信息。',
    chatToggleOpenLabel: '打开聊天',
    chatToggleCloseLabel: '关闭聊天',
    modalHeaderTitle: 'Chat iSrvd',
    welcomeMessageText: '你好，我能帮你做什么？',
} as unknown as Partial<CopilotChatLabels>

// 卸载时释放页面控制器和会话级 API 调用引用。
onUnmounted(() => {
    disposePageController()
    resetAgentAPICallRefs()
})
</script>

<template>
  <CopilotChatConfigurationProvider :labels="chineseLabels">
    <CopilotSidebar v-if="hasAgent" :default-open="false">
      <!-- 开合控制转存到 store，按钮由顶栏渲染，避免 z-1200 的侧栏盖住 header -->
      <template #toggle-button="{ isOpen, toggle }">
        <CopilotSidebarToggleBridge :is-open="isOpen" :toggle="toggle" />
      </template>
    </CopilotSidebar>
  </CopilotChatConfigurationProvider>
</template>
