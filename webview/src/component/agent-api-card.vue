<script setup lang="ts">
import { computed, ref } from 'vue'

import { executeAgentAPI, previewAgentAPICall } from '@/helper/agent-api'
import type { AgentAPIArgs } from '@/helper/agent-api'

type ToolStatus = 'inProgress' | 'executing' | 'complete'

interface ResourceItem {
    title: string
    subtitle: string
}

interface ResourceRow {
    key: string
    value: string
}

const props = withDefaults(defineProps<{
    args?: Partial<AgentAPIArgs>
    status?: ToolStatus
    result?: unknown
    respond?: (result: unknown) => Promise<void>
    approval?: boolean
}>(), {
    args: () => ({}),
    status: 'inProgress',
    result: undefined,
    respond: undefined,
    approval: false,
})

const submitting = ref(false)
const localError = ref('')

const preview = computed(() => previewAgentAPICall(props.args))
const method = computed(() => preview.value.method)
const path = computed(() => preview.value.path.replace(/^\/+/, ''))
const previewError = computed(() => preview.value.error || '')
const isDanger = computed(() => {
    const content = `${method.value} ${path.value} ${preview.value.summary} ${JSON.stringify(preview.value.body ?? '')}`
    return method.value === 'DELETE' || /(delete|remove|prune|stop|restart|redeploy|force)/i.test(content)
})
const methodClass = computed(() => {
    if (method.value === 'GET') return 'bg-blue-50 text-blue-700'
    if (isDanger.value) return 'bg-red-50 text-red-700'
    return 'bg-amber-50 text-amber-700'
})
const cardTitle = computed(() => {
    if (props.approval && props.status === 'executing') return '等待确认'
    if (props.status === 'inProgress') return '正在准备操作'
    if (props.status === 'executing') return method.value === 'GET' ? '正在查询资源' : '正在执行变更'
    if (resultCanceled.value) return '操作已取消'
    return resultSuccess.value ? '操作完成' : '操作失败'
})

const parsedResult = computed(() => parseResult(props.result))
const resultRecord = computed(() => isRecord(parsedResult.value) ? parsedResult.value : null)
const resultSuccess = computed(() => resultRecord.value?.success !== false)
const resultCanceled = computed(() => resultRecord.value?.canceled === true)
const resultMessage = computed(() => {
    const message = resultRecord.value?.message
    return typeof message === 'string' ? message : ''
})
const payload = computed(() => {
    const result = resultRecord.value
    if (!result) return sanitizeValue(parsedResult.value)
    if ('payload' in result) return sanitizeValue(result.payload)

    const rest = Object.fromEntries(
        Object.entries(result).filter(([key]) => !['success', 'message', 'canceled'].includes(key)),
    )
    return Object.keys(rest).length ? sanitizeValue(rest) : null
})
const resourceItems = computed<ResourceItem[]>(() => {
    if (!Array.isArray(payload.value)) return []
    return payload.value.slice(0, 6).map((item, index) => formatResourceItem(item, index))
})
const resourceRows = computed<ResourceRow[]>(() => {
    if (!isRecord(payload.value)) return []
    return Object.entries(payload.value).slice(0, 10).map(([key, value]) => ({
        key,
        value: formatValue(value),
    }))
})
const resourceText = computed(() => {
    if (payload.value === null || payload.value === undefined || Array.isArray(payload.value) || isRecord(payload.value)) return ''
    return formatValue(payload.value)
})
const bodyPreview = computed(() => formatJSONPreview(preview.value.body))
const paramsPreview = computed(() => formatJSONPreview(preview.value.query))

async function approve() {
    if (!props.respond || submitting.value) return
    submitting.value = true
    localError.value = ''
    try {
        const result = await executeAgentAPI(normalizeArgs(props.args), 'mutation')
        await props.respond(result)
    } catch (e) {
        localError.value = e instanceof Error ? e.message : '执行失败'
    } finally {
        submitting.value = false
    }
}

async function cancel() {
    if (!props.respond || submitting.value) return
    submitting.value = true
    try {
        await props.respond({ success: false, canceled: true, message: '用户取消了操作' })
    } catch (e) {
        localError.value = e instanceof Error ? e.message : '取消失败'
        submitting.value = false
    }
}

function normalizeArgs(args: Partial<AgentAPIArgs>): AgentAPIArgs {
    return {
        callRef: String(args.callRef || ''),
        arguments: args.arguments ? String(args.arguments) : undefined,
    }
}

function parseResult(value: unknown): unknown {
    if (typeof value !== 'string') return value
    try {
        return JSON.parse(value)
    } catch {
        return value
    }
}

function isRecord(value: unknown): value is Record<string, unknown> {
    return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function sanitizeValue(value: unknown, key = ''): unknown {
    if (/(password|passphrase|secret|token|api.?key|private.?key|authorization|jwt)/i.test(key)) return '••••••'
    if (Array.isArray(value)) return value.map(item => sanitizeValue(item))
    if (!isRecord(value)) return value
    return Object.fromEntries(Object.entries(value).map(([name, item]) => [name, sanitizeValue(item, name)]))
}

function formatJSONPreview(value: unknown): string {
    if (value === undefined || value === null) return ''
    return JSON.stringify(sanitizeValue(value), null, 2)
}

function formatResourceItem(value: unknown, index: number): ResourceItem {
    if (!isRecord(value)) return { title: `#${index + 1}`, subtitle: formatValue(value) }

    const titleKey = ['name', 'title', 'hostname', 'host', 'id', 'shortId', 'path', 'image'].find(key => value[key] !== undefined)
    const title = titleKey ? formatValue(value[titleKey]) : `#${index + 1}`
    const subtitle = Object.entries(value)
        .filter(([key, item]) => key !== titleKey && item !== null && item !== undefined && typeof item !== 'object')
        .slice(0, 3)
        .map(([key, item]) => `${key}: ${formatValue(item)}`)
        .join(' · ')
    return { title, subtitle }
}

function formatValue(value: unknown): string {
    if (value === null || value === undefined || value === '') return '-'
    if (typeof value === 'string') return value.length > 180 ? `${value.slice(0, 180)}…` : value
    if (typeof value === 'number' || typeof value === 'boolean') return String(value)
    const text = JSON.stringify(value)
    return text.length > 180 ? `${text.slice(0, 180)}…` : text
}
</script>

<template>
  <div class="card w-full bg-white">
    <div class="card-header justify-between">
      <div class="inline-info">
        <div class="card-icon" :class="approval && status === 'executing' ? 'bg-amber-100 text-amber-600' : 'bg-primary-100 text-primary-600'">
          <i class="fas" :class="approval && status === 'executing' ? 'fa-shield-halved' : 'fa-terminal'"></i>
        </div>
        <div class="min-w-0">
          <span class="item-title-sm">{{ cardTitle }}</span>
          <code class="item-subtitle-mono">{{ path || props.args.callRef || '等待调用引用' }}</code>
        </div>
      </div>
      <span v-if="method" class="inline-flex px-2 py-0.5 rounded-lg text-xs font-semibold" :class="methodClass">{{ method }}</span>
    </div>

    <div class="card-body space-y-3">
      <div v-if="status === 'inProgress'" class="flex items-center gap-2 text-sm text-slate-500">
        <span class="spinner w-4 h-4"></span>
        <span>正在生成请求参数…</span>
      </div>

      <template v-else-if="approval && status === 'executing'">
        <div v-if="previewError" class="rounded-lg bg-red-50 border border-red-200 p-3">
          <p class="text-sm font-medium text-red-600">{{ previewError }}</p>
        </div>
        <div class="rounded-lg bg-amber-50 border border-amber-200 p-3">
          <p class="text-sm font-medium text-amber-700">此请求会修改服务器状态，请确认后执行。</p>
        </div>

        <details v-if="paramsPreview || bodyPreview" class="rounded-lg border border-slate-200 bg-slate-50">
          <summary class="px-3 py-2 text-xs font-medium text-slate-600 cursor-pointer">查看请求参数</summary>
          <div class="border-t border-slate-200 p-3 space-y-3">
            <div v-if="paramsPreview">
              <p class="text-xs text-slate-400 mb-1">Query</p>
              <pre class="text-xs text-slate-600 whitespace-pre-wrap break-all">{{ paramsPreview }}</pre>
            </div>
            <div v-if="bodyPreview">
              <p class="text-xs text-slate-400 mb-1">Body</p>
              <pre class="text-xs text-slate-600 whitespace-pre-wrap break-all">{{ bodyPreview }}</pre>
            </div>
          </div>
        </details>

        <p v-if="localError" class="text-xs text-red-600">{{ localError }}</p>
        <div class="flex items-center justify-end gap-2 pt-1">
          <button type="button" class="btn btn-secondary" :disabled="submitting" @click="cancel">
            取消
          </button>
          <button type="button" class="btn" :class="isDanger ? 'btn-danger' : 'btn-primary'" :disabled="submitting || !!previewError" @click="approve">
            <span v-if="submitting" class="spinner w-3.5 h-3.5"></span>
            {{ submitting ? '执行中…' : '确认执行' }}
          </button>
        </div>
      </template>

      <div v-else-if="status === 'executing'" class="flex items-center gap-2 text-sm text-slate-500">
        <span class="spinner w-4 h-4"></span>
        <span>{{ method === 'GET' ? '正在读取资源…' : '正在执行操作…' }}</span>
      </div>

      <template v-else>
        <div v-if="resultCanceled" class="rounded-lg bg-slate-50 px-3 py-2 text-sm text-slate-500">
          {{ resultMessage || '用户取消了操作' }}
        </div>
        <div v-else-if="!resultSuccess" class="rounded-lg bg-red-50 border border-red-200 px-3 py-2 text-sm text-red-600">
          {{ resultMessage || '请求失败' }}
        </div>
        <p v-else-if="resultMessage" class="text-sm text-emerald-600 font-medium">{{ resultMessage }}</p>

        <template v-if="resultSuccess && Array.isArray(payload)">
          <div class="flex items-center justify-between text-xs text-slate-400">
            <span>资源列表</span>
            <span>共 {{ payload.length }} 项</span>
          </div>
          <div v-if="resourceItems.length" class="divide-y divide-slate-100 rounded-lg border border-slate-200 overflow-hidden">
            <div v-for="(item, index) in resourceItems" :key="index" class="px-3 py-2 min-w-0">
              <span class="item-title-sm">{{ item.title }}</span>
              <span v-if="item.subtitle" class="item-subtitle">{{ item.subtitle }}</span>
            </div>
          </div>
          <p v-if="payload.length > resourceItems.length" class="text-xs text-slate-400">仅展示前 {{ resourceItems.length }} 项，完整结果仍会提供给 Agent。</p>
        </template>

        <dl v-else-if="resultSuccess && resourceRows.length" class="divide-y divide-slate-100 rounded-lg border border-slate-200 overflow-hidden">
          <div v-for="row in resourceRows" :key="row.key" class="flex items-start gap-3 px-3 py-2">
            <dt class="w-24 flex-shrink-0 text-xs text-slate-400 break-all">{{ row.key }}</dt>
            <dd class="min-w-0 text-xs text-slate-600 break-all">{{ row.value }}</dd>
          </div>
        </dl>

        <p v-else-if="resultSuccess && resourceText" class="text-sm text-slate-600 break-words">{{ resourceText }}</p>
        <p v-else-if="resultSuccess && !resultMessage" class="text-sm text-emerald-600 font-medium">请求执行成功</p>
      </template>
    </div>
  </div>
</template>
