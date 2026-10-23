<script setup lang="ts">
import type { Message } from '@ag-ui/core'
import { CopilotChatAssistantMessage, CopilotChatMessageView } from '@copilotkit/vue'
import { computed } from 'vue'

import { groupAssistantProcesses } from '@/helper/copilot/messages'

import MessageProcess from '@/component/copilot/message-process.vue'

const props = defineProps<{ messages: Message[]; isRunning: boolean }>()
const processes = computed(() => groupAssistantProcesses(props.messages, props.isRunning))
const showingProgress = computed(() => [...processes.value.values()].some(process => process.active && !process.hasAnswer))
</script>

<template>
  <CopilotChatMessageView :messages="messages" :is-running="isRunning">
    <template #assistant-message="{ message }">
      <MessageProcess
        v-if="processes.get(message.id)?.id === message.id"
        :process="processes.get(message.id)!"
        :messages="messages"
      />
      <CopilotChatAssistantMessage
        v-else-if="!processes.has(message.id)"
        :message="message"
        :messages="messages"
        :is-running="isRunning"
      />
      <!-- 空插槽会触发上游的默认消息渲染，保留隐藏元素以显式覆盖回退。 -->
      <span v-else hidden aria-hidden="true"></span>
    </template>
    <template v-if="showingProgress" #cursor>
      <span class="sr-only" role="status">{{ $t('进行中') }}</span>
    </template>
  </CopilotChatMessageView>
</template>
