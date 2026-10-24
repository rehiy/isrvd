import type { AssistantMessage, Message, ToolMessage } from '@ag-ui/core'

export interface AssistantProcess {
    id: string
    messages: AssistantMessage[]
    active: boolean
    hasAnswer: boolean
}

// 只整理显示层：工具结果、消息 ID 和发给模型的对话历史保持原样。
export function groupAssistantProcesses(messages: Message[], isRunning: boolean): Map<string, AssistantProcess> {
    let turn = new Map<string, AssistantMessage>()
    const turns = [turn]
    for (const message of messages) {
        if (message.role === 'user') {
            turn = new Map<string, AssistantMessage>()
            turns.push(turn)
        }
        if (message.role !== 'assistant') continue
        const previous = turn.get(message.id)
        turn.set(message.id, previous
            ? { ...previous, ...message, content: message.content || previous.content, toolCalls: message.toolCalls ?? previous.toolCalls }
            : message)
    }

    const groups = new Map<string, AssistantProcess>()
    turns.forEach((items, index) => {
        const turn = [...items.values()]
        const first = turn[0]
        const last = turn[turn.length - 1]
        if (!first || !last || (turn.length === 1 && !first.toolCalls?.length)) return
        // 只有最后一条不带工具的正文作为答复；流式正文之后若出现工具，会归回过程。
        const hasAnswer = !!last.content?.trim() && !last.toolCalls?.length
        const progress = hasAnswer ? turn.slice(0, -1) : turn
        if (!progress.length) return
        const group = { id: first.id, messages: progress, active: isRunning && index === turns.length - 1, hasAnswer }
        for (const message of progress) groups.set(message.id, group)
    })
    return groups
}

export function toolMessageResults(messages: Message[]): Map<string, ToolMessage> {
    return new Map(messages.filter((message): message is ToolMessage => message.role === 'tool').map(message => [message.toolCallId, message]))
}

export function toolResultFailed(message: ToolMessage | undefined): boolean {
    if (!message || typeof message.content !== 'string') return false
    try {
        return JSON.parse(message.content)?.success === false
    } catch {
        return false
    }
}
