import { sanitizeAgentValue } from '@/helper/agent-sanitize'

// AI 助手大结果暂存：仅保存在当前页面内存中，刷新即失效。
// 生命周期与会话一致，无需服务端清理，也不存在多用户串数据的问题。

// 超过该字节数的工具结果会被暂存，仅向模型返回 blobId 与读取指引
export const BLOB_THRESHOLD = 8 * 1024

// result_read 单次返回的最大字符数
const READ_LIMIT_MAX = 8000
const READ_LIMIT_DEFAULT = 4000

// 最多保留的暂存条数，超出后按写入顺序淘汰最旧的一条
const MAX_BLOBS = 20

const blobs = new Map<string, string>()
let seq = 0

// blobPut 暂存文本并返回引用 ID
export function blobPut(text: string): string {
    const id = `b${Date.now().toString(36)}-${(++seq).toString(36)}`
    blobs.set(id, text)
    if (blobs.size > MAX_BLOBS) {
        const oldest = blobs.keys().next().value
        if (oldest) blobs.delete(oldest)
    }
    return id
}

// packToolResult API 工具结果先脱敏，超过阈值时只暂存脱敏后的内容。
export function packToolResult(result: { success: boolean; message: string; payload: unknown }): unknown {
    const sanitized = sanitizeAgentValue(result)
    const text = JSON.stringify(sanitized)
    if (text.length <= BLOB_THRESHOLD) return sanitized

    const blobId = blobPut(text)
    return {
        success: result.success,
        message: sanitizeAgentValue(result.message),
        truncated: true,
        blobId,
        hint: '结果过大，已暂存到本会话内存，用法见 result_read 工具说明；刷新页面后暂存失效',
    }
}

// blobRead 读取暂存内容：path 提取子字段优先，否则按 offset/limit 分段返回原文
export function blobRead(id: string, opts: { path?: string; offset?: number; limit?: number }): unknown {
    const text = blobs.get(id)
    if (text === undefined) {
        return { success: false, message: '暂存结果不存在或已失效（刷新页面会清空暂存），请重新调用原接口获取' }
    }

    if (opts.path) {
        let data: unknown
        try {
            data = JSON.parse(text)
        } catch {
            return { success: false, message: '暂存内容不是合法 JSON，请改用 offset/limit 分段读取' }
        }
        const value = resolvePath(data, opts.path)
        if (value === undefined) {
            return { success: false, message: `路径 ${opts.path} 不存在，请检查层级或改用 offset/limit 分段读取` }
        }
        const content = JSON.stringify(value)
        if (content.length <= READ_LIMIT_MAX) {
            return { success: true, content }
        }
        return {
            success: true,
            truncated: true,
            total: content.length,
            content: content.slice(0, READ_LIMIT_MAX),
            hint: '提取结果仍超限，已截断；可细化 path 或换用 offset/limit 分段读取',
        }
    }

    const offset = Math.max(0, opts.offset ?? 0)
    const limit = Math.min(Math.max(1, opts.limit ?? READ_LIMIT_DEFAULT), READ_LIMIT_MAX)
    return {
        success: true,
        total: text.length,
        offset,
        limit,
        content: text.slice(offset, offset + limit),
    }
}

// resolvePath 解析简易路径：支持 .key、[0]、[-1] 三种写法，如 [-1].data.system.memoryUsed
function resolvePath(data: unknown, path: string): unknown {
    const tokens = path.match(/[^.[\]]+|\[-?\d+\]/g) ?? []
    let cur: unknown = data
    for (const token of tokens) {
        if (cur === null || cur === undefined) return undefined
        const idx = /^\[(-?\d+)\]$/.exec(token)
        if (idx) {
            if (!Array.isArray(cur)) return undefined
            let i = Number(idx[1])
            if (i < 0) i = cur.length + i
            cur = cur[i]
        } else {
            cur = (cur as Record<string, unknown>)[token]
        }
    }
    return cur
}
