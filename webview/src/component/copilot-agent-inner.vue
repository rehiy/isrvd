<script setup lang="ts">
import { computed, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { CopilotChatConfigurationProvider, CopilotModalHeader, CopilotSidebar, useCopilotReadable, useFrontendTool } from '@copilotkit/vue'
import type { CopilotChatLabels } from '@copilotkit/vue'

import { http } from '@/service/client'
import { usePortal, useAgentStore } from '@/stores'
import CopilotSidebarToggleBridge from '@/component/copilot-sidebar-bridge.vue'
import { getPageController, disposePageController } from '@/helper/page-controller'
import { systemInstruction, getPageInstruction } from '@/helper/instructions'

const route = useRoute()
const portal = usePortal()
const agent = useAgentStore()

// 无 agent 权限时不出侧栏、不注册工具；Provider 仍由外层挂载以支撑注入
const hasAgent = computed(() => portal.hasPerm('agent'))

// ─── 页面上下文 ───

const pageInstruction = computed(() => getPageInstruction(route.fullPath || '/'))

useCopilotReadable(
    {
        description: '当前页面功能说明',
        value: pageInstruction.value,
        convert: (_desc, val) => String(val ?? ''),
    },
    [pageInstruction],
)
useCopilotReadable({ description: 'iSrvd 助手系统提示与 API 调用规范', value: systemInstruction.trim() })

// ─── 前端工具 ───

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

useFrontendTool({
    name: 'isrvd_api',
    description:
        '调用 iSrvd REST API 完成运维操作。path 为相对 api/ 的路径（如 docker/containers），' +
        'body 为 JSON 字符串请求体，params 为 JSON 字符串查询参数。禁止用于读取密钥类配置。',
    parameters: [
        { name: 'method', type: 'string', description: 'HTTP 方法：get / post / put / patch / delete', required: true },
        { name: 'path', type: 'string', description: '相对 api/ 的路径，如 docker/containers（单条路由为 docker/container/:id 单数形式）', required: true },
        { name: 'body', type: 'string', description: '请求体 JSON 字符串，仅 post/put/patch 使用', required: false },
        { name: 'params', type: 'string', description: '查询参数 JSON 字符串', required: false },
    ],
    handler: async ({ method, path, body, params }) => {
        try {
            const config = { params: params ? JSON.parse(params) : undefined }
            const target = String(path).replace(/^\/+/, '')
            const payload = body ? JSON.parse(body) : {}
            let res
            if (method === 'get') {
                res = await http.get(target, config)
            } else if (method === 'delete') {
                res = await http.delete(target, config)
            } else if (method === 'put') {
                res = await http.put(target, payload, config)
            } else if (method === 'patch') {
                res = await http.patch(target, payload, config)
            } else {
                res = await http.post(target, payload, config)
            }
            return { success: res?.success ?? true, message: res?.message ?? '', payload: res?.payload ?? null }
        } catch (e) {
            return { success: false, message: e instanceof Error ? e.message : '请求失败' }
        }
    },
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

// 卸载时释放页面控制器，清理高亮与遮罩
onUnmounted(disposePageController)

// 侧栏卸载时清掉顶栏按钮的开关引用
watch(hasAgent, v => { if (!v) agent.unbindSidebar() }, { immediate: true })
</script>

<template>
  <CopilotChatConfigurationProvider :labels="chineseLabels">
    <CopilotSidebar v-if="hasAgent" :default-open="false">
      <!-- 标题改为 Chat iSrvd：labels 是字面量联合类型改不了文案，故直接换掉 header -->
      <template #header>
        <CopilotModalHeader title="Chat iSrvd" />
      </template>
      <!-- 开合控制转存到 store，按钮由顶栏渲染，避免 z-1200 的侧栏盖住 header -->
      <template #toggle-button="{ isOpen, toggle }">
        <CopilotSidebarToggleBridge :is-open="isOpen" :toggle="toggle" />
      </template>
    </CopilotSidebar>
  </CopilotChatConfigurationProvider>
  <slot />
</template>
