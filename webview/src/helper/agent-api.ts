import axios from 'axios'

import { http } from '@/service/client'

import { packToolResult } from '@/helper/agent-blob'
import { sanitizeAgentValue } from '@/helper/agent-sanitize'

export interface AgentAPIArgs {
    callRef: string
    arguments?: string
}

export interface AgentAPIPreview {
    method: string
    path: string
    summary: string
    query?: Record<string, unknown>
    body?: unknown
    error?: string
}

export type AgentAPIMode = 'query' | 'mutation'

interface OpenAPIParameter {
    in?: unknown
    name?: unknown
    required?: unknown
    schema?: unknown
}

interface OpenAPIOperation {
    method: string
    path: string
    operationId: string
    summary: string
    tag: string
    parameters?: OpenAPIParameter[]
    requestBody?: unknown
    requestBodyRequired: boolean
    toolUnsupportedReason: string
}

interface RegisteredOperation extends OpenAPIOperation {
    expiresAt: number
    failures: Map<string, number>
    loaded: boolean
}

interface BusinessArguments {
    path: Record<string, unknown>
    query: Record<string, unknown>
    body?: unknown
}

interface ResolvedCall {
    method: string
    path: string
    query: Record<string, unknown>
    body?: unknown
}

const callRefTTL = 30 * 60 * 1000
const maxIdenticalFailures = 2
const callRefs = new Map<string, RegisteredOperation>()

// registerAgentAPILookup 将 OpenAPI 查询结果转换为当前页面会话内可执行的引用。
// callRef 只存在于内存中，刷新页面后失效，模型无法自行拼接 method/path。
export function registerAgentAPILookup(payload: unknown): unknown {
    cleanupExpiredCallRefs()
    if (!isRecord(payload)) return payload

    if (payload.mode === 'detail') {
        return registerOperationPayload(payload)
    }
    if (payload.mode === 'list' && Array.isArray(payload.operations)) {
        const operations = payload.operations.map(operation => isRecord(operation) ? registerOperationPayload(operation) : operation)
        return {
            ...payload,
            hint: operations.length
                ? '从 operations 选择带 callRef 的匹配项；没有 callRef 时按 toolUnsupportedReason 的说明操作。'
                : payload.hint,
            operations,
        }
    }
    return payload
}

export function resetAgentAPICallRefs() {
    callRefs.clear()
}

export function previewAgentAPICall(args: Partial<AgentAPIArgs>): AgentAPIPreview {
    const operation = getOperation(String(args.callRef || ''))
    if (!operation) {
        return { method: '', path: '', summary: '', error: '调用引用不存在或已过期，请重新调用 lookup_api。' }
    }
    const parsed = parseBusinessArguments(args.arguments)
    if ('error' in parsed) {
        return { method: operation.method.toUpperCase(), path: operation.path, summary: operation.summary, error: parsed.error }
    }
    const path = fillPath(operation.path, parsed.value.path)
    return {
        method: operation.method.toUpperCase(),
        path: 'error' in path ? operation.path : path.value,
        summary: operation.summary,
        query: parsed.value.query,
        body: parsed.value.body,
        error: 'error' in path ? path.error : undefined,
    }
}

// executeAgentAPI 是 Chat 工具调用 iSrvd REST API 的唯一入口。
// 模型只能提交 lookup_api 签发的 callRef，method/path 均从 OpenAPI 快照解析。
export async function executeAgentAPI(args: AgentAPIArgs, mode: AgentAPIMode): Promise<unknown> {
    const operation = getOperation(String(args.callRef || ''))
    if (!operation) {
        return agentError('UNKNOWN_CALL_REF', '调用引用不存在或已过期，请重新调用 lookup_api。', true)
    }

    const failureKey = canonicalArguments(args.arguments)
    if ((operation.failures.get(failureKey) || 0) >= maxIdenticalFailures) {
        return agentError('RETRY_LIMIT', '相同调用已连续失败两次，请重新查询接口或调整参数后再试。', false)
    }

    const detail = await ensureOperationDetail(operation)
    if ('error' in detail) return recordFailure(operation, failureKey, detail.error)
    if (operation.toolUnsupportedReason) {
        return agentError('UNSUPPORTED_OPERATION', operation.toolUnsupportedReason, false)
    }

    const resolved = resolveCall(operation, args.arguments, mode)
    if ('error' in resolved) return recordFailure(operation, failureKey, resolved.error)

    try {
        const config = { params: resolved.value.query }
        let res
        switch (resolved.value.method) {
            case 'get':
                res = await http.get(resolved.value.path, config)
                break
            case 'delete':
                res = await http.delete(resolved.value.path, config)
                break
            case 'put':
                res = await http.put(resolved.value.path, resolved.value.body ?? {}, config)
                break
            case 'patch':
                res = await http.patch(resolved.value.path, resolved.value.body ?? {}, config)
                break
            case 'post':
                res = await http.post(resolved.value.path, resolved.value.body ?? {}, config)
                break
            default:
                return recordFailure(operation, failureKey, agentError('UNKNOWN_OPERATION', 'OpenAPI 中的 HTTP 方法不受支持。', false))
        }

        if (!isRecord(res) || typeof res.success !== 'boolean') {
            return recordFailure(operation, failureKey, agentError('UNSUPPORTED_RESPONSE', '接口未返回标准 JSON 响应，请改用页面操作。', false))
        }
        operation.failures.delete(failureKey)
        return packToolResult({
            success: res?.success ?? true,
            message: res?.message ?? '',
            payload: res?.payload ?? null,
        })
    } catch (error) {
        return recordFailure(operation, failureKey, httpError(error))
    }
}

function registerOperationPayload(payload: Record<string, unknown>): Record<string, unknown> {
    const operation = operationFromPayload(payload)
    if (!operation) return payload
    if (operation.toolUnsupportedReason) return { ...payload, hint: operation.toolUnsupportedReason }

    const bytes = new Uint8Array(16)
    globalThis.crypto.getRandomValues(bytes)
    const suffix = Array.from(bytes, byte => byte.toString(16).padStart(2, '0')).join('')
    const callRef = `call_${suffix}`
    callRefs.set(callRef, {
        ...operation,
        expiresAt: Date.now() + callRefTTL,
        failures: new Map(),
        loaded: payload.mode === 'detail',
    })
    return {
        ...payload,
        callRef,
        hint: payload.mode === 'detail'
            ? '使用 callRef 调用 isrvd_api 或 isrvd_mutation；arguments 为 JSON 字符串，结构为 {"path":{},"query":{},"body":{}}。不要自行填写 HTTP 方法或路径。'
            : payload.hint,
    }
}

function operationFromPayload(payload: Record<string, unknown>): OpenAPIOperation | null {
    const method = stringValue(payload.method).toLowerCase()
    const path = stringValue(payload.path)
    const operationId = stringValue(payload.operationId)
    if (!method || !path || !operationId) return null
    return {
        method,
        path,
        operationId,
        summary: stringValue(payload.summary),
        tag: stringValue(payload.tag),
        parameters: Array.isArray(payload.parameters) ? payload.parameters.filter(isRecord) : undefined,
        requestBody: payload.requestBody,
        requestBodyRequired: payload.requestBodyRequired === true,
        toolUnsupportedReason: stringValue(payload.toolUnsupportedReason),
    }
}

async function ensureOperationDetail(operation: RegisteredOperation): Promise<{ value: RegisteredOperation } | { error: Record<string, unknown> }> {
    if (operation.loaded) return { value: operation }
    try {
        const res = await http.get('agent/openapi', { params: { path: operation.path, method: operation.method } })
        const payload = res?.payload
        if (!isRecord(payload)) {
            return { error: agentError('OPERATION_CHANGED', '接口定义已变化，请重新调用 lookup_api。', true) }
        }
        const detail = operationFromPayload(payload)
        if (!detail || payload.mode !== 'detail' || detail.operationId !== operation.operationId) {
            return { error: agentError('OPERATION_CHANGED', '接口定义已变化，请重新调用 lookup_api。', true) }
        }
        Object.assign(operation, detail)
        operation.loaded = true
        return { value: operation }
    } catch (error) {
        return { error: httpError(error, '无法刷新接口定义') }
    }
}

function resolveCall(operation: RegisteredOperation, source: string | undefined, mode: AgentAPIMode): { value: ResolvedCall } | { error: Record<string, unknown> } {
    const isQuery = operation.method === 'get'
    if ((mode === 'query') !== isQuery) {
        const expected = isQuery ? 'isrvd_api' : 'isrvd_mutation'
        return { error: agentError('WRONG_TOOL', `该操作必须使用 ${expected}。`, true) }
    }

    const parsed = parseBusinessArguments(source)
    if ('error' in parsed) return { error: agentError('INVALID_ARGUMENTS', parsed.error, true) }

    const parameters = operation.parameters || []
    const pathParameters = parameters.filter(parameter => parameter.in === 'path')
    const queryParameters = parameters.filter(parameter => parameter.in === 'query')
    const pathError = validateParameterGroup(parsed.value.path, pathParameters, 'path')
    if (pathError) return { error: agentError('INVALID_ARGUMENTS', pathError, true) }
    const queryError = validateParameterGroup(parsed.value.query, queryParameters, 'query')
    if (queryError) return { error: agentError('INVALID_ARGUMENTS', queryError, true) }

    const bodyError = validateRequestBody(parsed.value.body, operation.requestBody, operation.requestBodyRequired)
    if (bodyError) return { error: agentError('INVALID_ARGUMENTS', bodyError, true) }

    const path = fillPath(operation.path, parsed.value.path)
    if ('error' in path) return { error: agentError('INVALID_ARGUMENTS', path.error, true) }
    const target = path.value.replace(/^\/+/, '')
    if (!target || target.includes('://') || target.split('/').includes('..')) {
        return { error: agentError('INVALID_ARGUMENTS', '解析后的 API 路径不是合法的站内路径。', false) }
    }

    return {
        value: {
            method: operation.method,
            path: target,
            query: parsed.value.query,
            body: parsed.value.body,
        },
    }
}

function parseBusinessArguments(source?: string): { value: BusinessArguments } | { error: string } {
    if (!source) return { value: { path: {}, query: {} } }
    let parsed: unknown
    try {
        parsed = JSON.parse(source)
    } catch {
        return { error: 'arguments 不是合法 JSON。' }
    }
    if (!isRecord(parsed)) return { error: 'arguments 必须是 JSON 对象。' }
    const unknown = Object.keys(parsed).filter(key => !['path', 'query', 'body'].includes(key))
    if (unknown.length) return { error: `arguments 包含未知字段：${unknown.join(', ')}。` }
    if (parsed.path !== undefined && !isRecord(parsed.path)) return { error: 'arguments.path 必须是对象。' }
    if (parsed.query !== undefined && !isRecord(parsed.query)) return { error: 'arguments.query 必须是对象。' }
    return {
        value: {
            path: isRecord(parsed.path) ? parsed.path : {},
            query: isRecord(parsed.query) ? parsed.query : {},
            body: parsed.body,
        },
    }
}

function canonicalArguments(source?: string): string {
    // ponytail: 原样字符串做失败去重键，模型重试通常逐字重放；如出现键序绕过 RETRY_LIMIT 再做规范化
    return source?.trim() || '{}'
}

function validateParameterGroup(values: Record<string, unknown>, parameters: OpenAPIParameter[], group: string): string {
    const allowed = new Set(parameters.map(parameter => stringValue(parameter.name)).filter(Boolean))
    const unknown = Object.keys(values).filter(name => !allowed.has(name))
    if (unknown.length) return `${group} 包含接口未声明的参数：${unknown.join(', ')}。`

    for (const parameter of parameters) {
        const name = stringValue(parameter.name)
        const value = values[name]
        if (parameter.required === true && (value === undefined || value === null || value === '')) {
            return `${group}.${name} 是必填参数。`
        }
        if (value !== undefined) {
            const error = validateSchema(value, parameter.schema, `${group}.${name}`)
            if (error) return error
        }
    }
    return ''
}

function validateRequestBody(body: unknown, schema: unknown, required: boolean): string {
    if (schema === undefined || schema === null) return body === undefined ? '' : '该接口不接受 body。'
    if (body === undefined) {
        if (required) return 'body 是必填参数。'
        const requiredFields = isRecord(schema) && Array.isArray(schema.required) ? schema.required : []
        return requiredFields.length ? `body 缺少必填字段：${requiredFields.join(', ')}。` : ''
    }
    return validateSchema(body, schema, 'body')
}

function validateSchema(value: unknown, schema: unknown, path: string): string {
    if (!isRecord(schema)) return ''
    if (Array.isArray(schema.enum) && !schema.enum.some(item => Object.is(item, value))) {
        return `${path} 必须是以下值之一：${schema.enum.map(String).join(', ')}。`
    }

    const type = stringValue(schema.type)
    if (type === 'object') {
        if (!isRecord(value)) return `${path} 必须是对象。`
        const properties = isRecord(schema.properties) ? schema.properties : {}
        const required = Array.isArray(schema.required) ? schema.required.map(String) : []
        for (const name of required) {
            if (value[name] === undefined || value[name] === null || value[name] === '') return `${path}.${name} 是必填字段。`
        }
        if (schema.additionalProperties === false) {
            const unknown = Object.keys(value).filter(name => !(name in properties))
            if (unknown.length) return `${path} 包含未知字段：${unknown.join(', ')}。`
        }
        for (const [name, child] of Object.entries(value)) {
            if (name in properties) {
                const error = validateSchema(child, properties[name], `${path}.${name}`)
                if (error) return error
            }
        }
        return ''
    }
    if (type === 'array') {
        if (!Array.isArray(value)) return `${path} 必须是数组。`
        for (let index = 0; index < value.length; index++) {
            const error = validateSchema(value[index], schema.items, `${path}[${index}]`)
            if (error) return error
        }
        return ''
    }
    if (type === 'string' && typeof value !== 'string') return `${path} 必须是字符串。`
    if (type === 'boolean' && typeof value !== 'boolean') return `${path} 必须是布尔值。`
    if (type === 'number' && typeof value !== 'number') return `${path} 必须是数字。`
    if (type === 'integer' && !Number.isInteger(value)) return `${path} 必须是整数。`
    return ''
}

function fillPath(template: string, values: Record<string, unknown>): { value: string } | { error: string } {
    let path = template
    for (const name of template.match(/\{([^}]+)\}/g)?.map(value => value.slice(1, -1)) || []) {
        const value = values[name]
        if (value === undefined || value === null || value === '') return { error: `path.${name} 是必填参数。` }
        path = path.replace(`{${name}}`, encodeURIComponent(String(value)))
    }
    return { value: path }
}

function recordFailure(operation: RegisteredOperation, key: string, error: Record<string, unknown>): Record<string, unknown> {
    operation.failures.set(key, (operation.failures.get(key) || 0) + 1)
    return error
}

function agentError(kind: string, message: string, recoverable: boolean, status?: number): Record<string, unknown> {
    return {
        success: false,
        message: sanitizeAgentValue(message),
        error: {
            kind,
            recoverable,
            ...(status ? { status } : {}),
        },
    }
}

function httpError(error: unknown, prefix = ''): Record<string, unknown> {
    if (!axios.isAxiosError(error)) {
        return agentError('EXECUTION_FAILED', prefix || (error instanceof Error ? error.message : '请求失败'), false)
    }
    const status = error.response?.status
    const data = error.response?.data
    const detail = isRecord(data) ? stringValue(data.message) : ''
    const message = [prefix, detail || error.message].filter(Boolean).join('：')
    if (!status) return agentError('TRANSIENT_FAILURE', message || '网络请求失败。', true)
    if (status === 400 || status === 422) return agentError('INVALID_ARGUMENTS', message, true, status)
    if (status === 401 || status === 403) return agentError('PERMISSION_DENIED', message, false, status)
    if (status === 404) return agentError('RESOURCE_NOT_FOUND', message, true, status)
    if (status === 409) return agentError('PRECONDITION_FAILED', message, true, status)
    if (status === 429 || status >= 500) return agentError('SERVICE_UNAVAILABLE', message, true, status)
    return agentError('EXECUTION_FAILED', message, false, status)
}

function getOperation(callRef: string): RegisteredOperation | null {
    const operation = callRefs.get(callRef)
    if (!operation) return null
    if (operation.expiresAt <= Date.now()) {
        callRefs.delete(callRef)
        return null
    }
    return operation
}

function cleanupExpiredCallRefs() {
    const now = Date.now()
    for (const [callRef, operation] of callRefs) {
        if (operation.expiresAt <= now) callRefs.delete(callRef)
    }
}

function isRecord(value: unknown): value is Record<string, unknown> {
    return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function stringValue(value: unknown): string {
    return typeof value === 'string' ? value : ''
}
