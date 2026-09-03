import { http } from '@/service/client'

import { packToolResult } from '@/helper/agent-blob'

export interface AgentAPIArgs {
    method: string
    path: string
    body?: string
    params?: string
}

export type AgentAPIMode = 'query' | 'mutation'

// executeAgentAPI 是 Chat 工具调用 iSrvd REST API 的唯一入口。
// 查询与写入共用实现，只在这里校验各自允许的 HTTP 方法。
export async function executeAgentAPI(args: AgentAPIArgs, mode: AgentAPIMode): Promise<unknown> {
    const method = String(args.method || '').toLowerCase()
    const target = String(args.path || '').replace(/^\/+/, '')

    if (!target) return { success: false, message: '缺少 API 路径' }
    if (target.includes('://') || target.split('/').includes('..')) {
        return { success: false, message: 'API 路径必须是相对 api/ 的站内路径' }
    }
    if (mode === 'query' && method !== 'get') {
        return { success: false, message: 'isrvd_api 只允许 GET 查询；写操作请使用 isrvd_mutation' }
    }
    if (mode === 'mutation' && !['post', 'put', 'patch', 'delete'].includes(method)) {
        return { success: false, message: 'isrvd_mutation 只允许 POST、PUT、PATCH、DELETE' }
    }

    try {
        const params = args.params ? JSON.parse(args.params) : undefined
        const body = args.body ? JSON.parse(args.body) : {}
        const config = { params }
        let res

        switch (method) {
            case 'get':
                res = await http.get(target, config)
                break
            case 'delete':
                res = await http.delete(target, config)
                break
            case 'put':
                res = await http.put(target, body, config)
                break
            case 'patch':
                res = await http.patch(target, body, config)
                break
            case 'post':
                res = await http.post(target, body, config)
                break
            default:
                return { success: false, message: `不支持的 HTTP 方法：${method || '空'}` }
        }

        return packToolResult({
            success: res?.success ?? true,
            message: res?.message ?? '',
            payload: res?.payload ?? null,
        })
    } catch (e) {
        return { success: false, message: e instanceof Error ? e.message : '请求失败' }
    }
}
