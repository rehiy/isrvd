// 工具结果与卡片共用脱敏逻辑；不修改原始 API 对象。
export function sanitizeCopilotValue(value: unknown, key = ''): unknown {
    if (/(password|passphrase|secret|token|api.?key|private.?key|authorization|jwt)|^key$/i.test(key)) return '••••••'
    if (typeof value === 'string') {
        return value.replace(/-----BEGIN ([A-Z ]*PRIVATE KEY)-----[\s\S]*?-----END \1-----/g, '••••••')
    }
    if (Array.isArray(value)) return value.map(item => sanitizeCopilotValue(item))
    if (typeof value !== 'object' || value === null) return value
    return Object.fromEntries(Object.entries(value).map(([name, item]) => [name, sanitizeCopilotValue(item, name)]))
}
