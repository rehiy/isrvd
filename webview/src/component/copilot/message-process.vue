<script setup lang="ts">
import type { AssistantMessage, Message } from '@ag-ui/core'
import { CopilotChatAssistantMessage, useCopilotKit } from '@copilotkit/vue'
import { computed, ref, useId } from 'vue'

import { toolMessageResults, toolResultFailed } from '@/helper/copilot/messages'
import type { AssistantProcess } from '@/helper/copilot/messages'

const props = defineProps<{ process: AssistantProcess; messages: Message[] }>()
const { executingToolCallIds } = useCopilotKit()
const expanded = ref(false)
const detailsID = useId()
const results = computed(() => toolMessageResults(props.messages))
const calls = computed(() => props.process.messages.flatMap(message => message.toolCalls ?? []))
const running = computed(() => (props.process.active && !props.process.hasAnswer) || calls.value.some(call => executingToolCallIds.value.has(call.id)))
const waitingApproval = computed(() => calls.value.some(call => call.function.name === 'isrvd_mutation' && executingToolCallIds.value.has(call.id) && !results.value.has(call.id)))
const hasFailure = computed(() => calls.value.some(call => toolResultFailed(results.value.get(call.id))))
const latestID = computed(() => props.process.messages[props.process.messages.length - 1]?.id)
const status = computed(() => {
    if (waitingApproval.value) return '等待确认'
    if (running.value) return '进行中'
    if (hasFailure.value) return '含失败或取消记录'
    return '已结束'
})

function needsAttention(message: AssistantMessage): boolean {
    return (message.toolCalls ?? []).some(call =>
        (call.function.name === 'isrvd_mutation' && !results.value.has(call.id)) || toolResultFailed(results.value.get(call.id)),
    )
}

function isVisible(message: AssistantMessage): boolean {
    return expanded.value || needsAttention(message) || (!props.process.hasAnswer && message.id === latestID.value)
}
</script>

<template>
  <section class="copilot-message-process my-3 min-w-0" :aria-label="$t('执行过程')">
    <button
      type="button"
      class="flex items-center gap-2 w-full py-2 text-left text-xs text-slate-500 hover:text-slate-700 rounded-lg focus-visible:outline-2 focus-visible:outline-primary-500"
      :aria-expanded="expanded"
      :aria-controls="detailsID"
      @click="expanded = !expanded"
    >
      <i
        class="fas w-4 text-center flex-shrink-0"
        :class="waitingApproval ? 'fa-shield-halved text-amber-600' : running ? 'fa-circle-notch fa-spin text-primary-500' : hasFailure ? 'fa-circle-exclamation text-amber-600' : 'fa-list-check'"
        aria-hidden="true"
      ></i>
      <span class="font-medium">{{ $t('执行过程') }}</span>
      <span class="min-w-0 flex-1 truncate">{{ calls.length ? $t('{0} 次工具调用', calls.length) : $t('{0} 条进度', process.messages.length) }}</span>
      <span class="flex-shrink-0" :class="{ 'text-amber-600': waitingApproval || hasFailure }">{{ $t(status) }}</span>
      <i class="fas text-[10px] flex-shrink-0" :class="expanded ? 'fa-chevron-up' : 'fa-chevron-down'" aria-hidden="true"></i>
    </button>
    <div :id="detailsID" class="space-y-3 border-l border-slate-200 pl-3 ml-2">
      <!-- v-show 保留工具渲染器，折叠/展开不会重挂审批或丢失卡片状态。 -->
      <div v-for="message in process.messages" v-show="isVisible(message)" :key="message.id" class="min-w-0">
        <CopilotChatAssistantMessage :message="message" :messages="messages" :is-running="running" :toolbar-visible="false" />
      </div>
    </div>
  </section>
</template>

<style scoped>
.copilot-message-process :deep(.copilot-chat-assistant-markdown) {
    font-size: 0.8125rem;
    line-height: 1.6;
    color: var(--color-slate-500);
}

.copilot-message-process :deep(.copilot-chat-assistant-markdown > :first-child) {
    margin-top: 0;
}

.copilot-message-process :deep(.copilot-chat-assistant-markdown > :last-child) {
    margin-bottom: 0.5rem;
}
</style>
