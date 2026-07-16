<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { CaddyRoute, CaddyRouteUpsert, CaddyHandlerKind, CaddyHandlerKindCard, CaddyHeaderOp, CaddyHandler, CaddyHandlerReverseProxy, CaddyHandlerFileServer, CaddyHandlerStaticResponse, CaddyHandlerRewrite, CaddyHandlerHeaders, DockerContainerInfo } from '@/service/types'

import { parseHostPort } from '@/helper/format'

import BaseModal from '@/component/modal.vue'
import ToggleCard from '@/component/toggle-card.vue'

import ContainerPortSelect from '@/views/docker/widget/container-port-select.vue'
import ContainerSelect from '@/views/docker/widget/container-select.vue'

const HANDLER_KIND_CARDS: CaddyHandlerKindCard[] = [
    { value: 'reverse_proxy', title: '反向代理', desc: '将请求转发到一个或多个上游服务器', icon: 'fa-diagram-project', tone: 'indigo' },
    { value: 'file_server', title: '静态文件服务', desc: '提供本地目录中的静态文件访问', icon: 'fa-folder-open', tone: 'emerald' },
    { value: 'static_response', title: '静态响应', desc: '直接返回指定的 HTTP 状态码和响应体', icon: 'fa-bolt', tone: 'amber' },
    { value: 'rewrite', title: 'URI 重写', desc: '重写请求 URI、去除路径前后缀或替换子串', icon: 'fa-pen-to-square', tone: 'violet' },
    { value: 'raw', title: '自定义 JSON', desc: '直接编辑 Caddy 原生 JSON，适用于复杂场景', icon: 'fa-code', tone: 'slate' }
]

const TONE_CARD_ACTIVE: Record<string, string> = {
    indigo: 'border-indigo-300 bg-indigo-50 text-indigo-700',
    emerald: 'border-emerald-300 bg-emerald-50 text-emerald-700',
    amber: 'border-amber-300 bg-amber-50 text-amber-700',
    violet: 'border-violet-300 bg-violet-50 text-violet-700',
    rose: 'border-rose-300 bg-rose-50 text-rose-700',
    slate: 'border-slate-300 bg-slate-50 text-slate-700'
}
const TONE_ICON_ACTIVE: Record<string, string> = {
    indigo: 'bg-indigo-100', emerald: 'bg-emerald-100', amber: 'bg-amber-100', violet: 'bg-violet-100', rose: 'bg-rose-100', slate: 'bg-slate-200'
}

const defaultFormData = () => ({
    kind: 'reverse_proxy' as CaddyHandlerKind,
    hosts: '',
    paths: '',
    methods: '',
    protocol: '',
    upstreams: '',
    upstreamHost: '',
    upstreamPort: '',
    fastcgi: false,
    fastcgiRoot: '',
    dialTimeout: '',
    readTimeout: '',
    writeTimeout: '',
    proxyRewriteEnabled: false,
    proxyRewriteMethod: '',
    proxyRewriteUri: '',
    proxyStripPathPrefix: '',
    proxyStripPathSuffix: '',
    proxyUriSubstringFind: '',
    proxyUriSubstringReplace: '',
    root: '',
    browse: false,
    statusCode: 200,
    body: '',
    rewriteUri: '',
    stripPathPrefix: '',
    stripPathSuffix: '',
    uriSubstringFind: '',
    uriSubstringReplace: '',
    requestHeaders: [] as CaddyHeaderOp[],
    responseHeaders: [] as CaddyHeaderOp[],
    enableHeaders: false
})

@Component({
    expose: ['show'],
    components: { BaseModal, ContainerPortSelect, ContainerSelect, ToggleCard },
    emits: ['success']
})
class RouteEditModal extends Vue {
    portal = usePortal()

    isOpen = false
    modalLoading = false
    isEditMode = false
    editingIndex = -1
    containers: DockerContainerInfo[] = []

    get canLoadDockerContainers() {
        return this.portal.hasPerm('GET /api/docker/containers')
    }

    // 匹配请求头多行编辑列表
    matchHeaderList: { key: string; value: string }[] = []
    enableMatchHeaders = false

    formData = defaultFormData()
    // raw 模式下用户直接编辑的完整路由 JSON
    rawJson = ''
    // rawJson 是否已被手动编辑或从已有路由加载。
    // 为 true 时切换到 raw 模式不会用表单内容覆盖它，避免配置丢失。
    rawDirty = false

    readonly handlerKindCards = HANDLER_KIND_CARDS

    // ─── 模式卡片样式 ───
    modeCardClass(item: CaddyHandlerKindCard) {
        const active = this.formData.kind === item.value
        return `option-card ${active ? TONE_CARD_ACTIVE[item.tone] : 'option-card-inactive'}`
    }

    modeCardIconClass(item: CaddyHandlerKindCard) {
        const active = this.formData.kind === item.value
        return `option-card-icon ${active ? TONE_ICON_ACTIVE[item.tone] : 'bg-slate-100'}`
    }

    setKind(kind: CaddyHandlerKind) {
        if (kind === 'raw') {
            // 切换到自定义 JSON 时，将当前表单生成的完整路由 JSON 填入。
            // 仅在 raw 内容未被手动编辑/加载时自动生成，否则保留原内容，避免覆盖导致配置丢失。
            const prevKind = this.formData.kind
            if (prevKind !== 'raw' && !this.rawDirty) {
                try {
                    const rawStr = this.buildRawFromCurrent()
                    const handle = rawStr ? JSON.parse(rawStr) : []
                    const splitLines = (s: string) => s.split('\n').map(t => t.trim()).filter(Boolean)
                    const splitSpaces = (s: string) => s.split(/\s+/).map(t => t.trim()).filter(Boolean)
                    const hosts = splitLines(this.formData.hosts)
                    const paths = splitLines(this.formData.paths)
                    const methods = splitSpaces(this.formData.methods).map(s => s.toUpperCase())
                    const headers = this.listToHeaders(this.matchHeaderList)
                    const protocol = this.formData.protocol || undefined
                    const matchSet: Record<string, unknown> = {}
                    if (hosts.length) matchSet.host = hosts
                    if (paths.length) matchSet.path = paths
                    if (methods.length) matchSet.method = methods
                    if (headers) matchSet.header = headers
                    if (protocol) matchSet.protocol = protocol
                    const match = Object.keys(matchSet).length ? [matchSet] : undefined
                    const payload: Record<string, unknown> = { handle }
                    if (match) payload.match = match
                    this.rawJson = JSON.stringify(payload, null, 2)
                } catch {
                    this.rawJson = '{\n  "handle": []\n}'
                }
            }
        }
        this.formData.kind = kind
    }

    buildRawFromCurrent(): string {
        const f = this.formData
        let terminalHandler: Record<string, unknown> | null = null
        switch (f.kind) {
            case 'reverse_proxy': {
                const upstreams = f.upstreams.split('\n').map(t => t.trim()).filter(Boolean)
                if (upstreams.length) {
                    terminalHandler = {
                        handler: 'reverse_proxy',
                        upstreams: upstreams.map(d => ({ dial: d }))
                    }
                    if (f.fastcgi) {
                        const transport: Record<string, unknown> = { protocol: 'fastcgi' }
                        if (f.fastcgiRoot.trim()) transport.root = f.fastcgiRoot.trim()
                        terminalHandler.transport = transport
                    } else if (f.dialTimeout.trim() || f.readTimeout.trim() || f.writeTimeout.trim()) {
                        const transport: Record<string, unknown> = { protocol: 'http' }
                        if (f.dialTimeout.trim()) transport.dial_timeout = f.dialTimeout.trim()
                        if (f.readTimeout.trim()) transport.response_header_timeout = f.readTimeout.trim()
                        if (f.writeTimeout.trim()) transport.write_timeout = f.writeTimeout.trim()
                        terminalHandler.transport = transport
                    }
                    const rewrite = this.buildProxyRewriteRaw()
                    if (rewrite) terminalHandler.rewrite = rewrite
                }
                break
            }
            case 'file_server': {
                terminalHandler = { handler: 'file_server' }
                if (f.root.trim()) terminalHandler.root = f.root.trim()
                if (f.browse) terminalHandler.browse = {}
                break
            }
            case 'static_response': {
                terminalHandler = { handler: 'static_response' }
                if (f.statusCode) terminalHandler.status_code = f.statusCode
                if (f.body) terminalHandler.body = f.body
                break
            }
            case 'rewrite': {
                terminalHandler = { handler: 'rewrite' }
                if (f.rewriteUri.trim()) terminalHandler.uri = f.rewriteUri.trim()
                if (f.stripPathPrefix.trim()) terminalHandler.strip_path_prefix = f.stripPathPrefix.trim()
                if (f.stripPathSuffix.trim()) terminalHandler.strip_path_suffix = f.stripPathSuffix.trim()
                if (f.uriSubstringFind.trim()) {
                    terminalHandler.uri_substring = [{ find: f.uriSubstringFind.trim(), replace: f.uriSubstringReplace.trim() }]
                }
                break
            }
        }
        if (!terminalHandler) return ''
        const buildOps = (ops: CaddyHeaderOp[]) => {
            const set: Record<string, string[]> = {}
            const add: Record<string, string[]> = {}
            const del: string[] = []
            ops.forEach(op => {
                if (op.op === 'set') set[op.field] = [op.value]
                else if (op.op === 'add') { add[op.field] = add[op.field] || []; add[op.field].push(op.value) }
                else if (op.op === 'delete') del.push(op.field)
            })
            const out: Record<string, unknown> = {}
            if (Object.keys(set).length) out.set = set
            if (Object.keys(add).length) out.add = add
            if (del.length) out.delete = del
            return out
        }
        const handles: unknown[] = []
        // 只有当启用header操作时才添加headers中间件
        if (f.enableHeaders) {
            const reqOps = f.requestHeaders.filter(op => op.field.trim())
            const respOps = f.responseHeaders.filter(op => op.field.trim())
            if (reqOps.length || respOps.length) {
                const hh: Record<string, unknown> = { handler: 'headers' }
                if (reqOps.length) hh.request = buildOps(reqOps)
                if (respOps.length) hh.response = buildOps(respOps)
                handles.push(hh)
            }
        }
        handles.push(terminalHandler)
        try { return JSON.stringify(handles, null, 2) } catch { return '' }
    }

    buildProxyRewriteRaw(): Record<string, unknown> | null {
        const f = this.formData
        if (!f.proxyRewriteEnabled) return null
        const rewrite: Record<string, unknown> = {}
        if (f.proxyRewriteMethod.trim()) rewrite.method = f.proxyRewriteMethod.trim().toUpperCase()
        if (f.proxyRewriteUri.trim()) rewrite.uri = f.proxyRewriteUri.trim()
        if (f.proxyStripPathPrefix.trim()) rewrite.strip_path_prefix = f.proxyStripPathPrefix.trim()
        if (f.proxyStripPathSuffix.trim()) rewrite.strip_path_suffix = f.proxyStripPathSuffix.trim()
        if (f.proxyUriSubstringFind.trim()) {
            rewrite.uri_substring = [{ find: f.proxyUriSubstringFind.trim(), replace: f.proxyUriSubstringReplace.trim() }]
        }
        return Object.keys(rewrite).length ? rewrite : null
    }

    // ── 匹配请求头操作 ──
    headersToList(headers: Record<string, string[]> | undefined): { key: string; value: string }[] {
        const rows: { key: string; value: string }[] = []
        for (const [k, vals] of Object.entries(headers || {})) {
            if (vals.length === 0) {
                rows.push({ key: k, value: '' })
            } else {
                for (const v of vals) rows.push({ key: k, value: v })
            }
        }
        return rows
    }

    listToHeaders(list: { key: string; value: string }[]): Record<string, string[]> | undefined {
        const headers: Record<string, string[]> = {}
        for (const row of list) {
            const k = row.key.trim()
            if (!k) continue
            if (!headers[k]) headers[k] = []
            if (row.value.trim()) headers[k].push(row.value.trim())
        }
        return Object.keys(headers).length ? headers : undefined
    }

    addMatchHeaderRow() {
        this.matchHeaderList.push({ key: '', value: '' })
    }

    removeMatchHeaderRow(idx: number) {
        this.matchHeaderList.splice(idx, 1)
    }
    // ─── 请求头/响应头中间件操作 ───
    buildHeaderOps(target: 'request' | 'response'): CaddyHeaderOp[] | undefined {
        const list = target === 'request' ? this.formData.requestHeaders : this.formData.responseHeaders
        const ops = list.filter(op => op.field.trim())
        return ops.length ? ops : undefined
    }

    addHeaderOp(target: 'request' | 'response') {
        const list = target === 'request' ? this.formData.requestHeaders : this.formData.responseHeaders
        list.push({ op: 'set', field: '', value: '' })
    }

    removeHeaderOp(target: 'request' | 'response', idx: number) {
        const list = target === 'request' ? this.formData.requestHeaders : this.formData.responseHeaders
        list.splice(idx, 1)
    }

    getPortsByHost(host: string): string[] {
        return this.containers.find(c => c.name === host.trim())?.ports || []
    }

    parseUpstream(upstream: string) {
        return parseHostPort(upstream)
    }

    syncSelectedUpstream() {
        const host = this.formData.upstreamHost.trim()
        const port = this.formData.upstreamPort.trim()
        if (!host) return
        const lines = this.formData.upstreams.split('\n').map(t => t.trim()).filter(Boolean)
        const upstream = port ? `${host}:${port}` : host
        this.formData.upstreams = [upstream, ...lines.slice(1)].join('\n')
    }

    setUpstreamHost(host: string) {
        this.formData.upstreamHost = host
        const port = (this.getPortsByHost(host)[0] || '').split('/')[0].split(':').pop() || ''
        if (port) this.formData.upstreamPort = port
        this.syncSelectedUpstream()
    }

    setUpstreamPort(port: string) {
        this.formData.upstreamPort = port
        this.syncSelectedUpstream()
    }

    syncSelectedFromText() {
        const first = this.formData.upstreams.split('\n').map(t => t.trim()).filter(Boolean)[0] || ''
        const { host, port } = this.parseUpstream(first)
        this.formData.upstreamHost = host
        this.formData.upstreamPort = port
    }

    async loadContainers() {
        this.containers = []
        if (!this.canLoadDockerContainers) return
        try {
            const res = await api.dockerContainerList()
            this.containers = (res.payload || []).filter(c => c.state === 'running')
        } catch {}
    }

    show(route: CaddyRoute | null) {
        Object.assign(this.formData, defaultFormData())
        this.matchHeaderList = []
        this.enableMatchHeaders = false
        this.rawJson = ''
        this.rawDirty = false
        void this.loadContainers()
        if (route) {
            this.isEditMode = true
            this.editingIndex = route.index
            // 解析 match：取第一个 MatchSet
            const matchSet = route.match?.[0]
            if (matchSet) {
                this.formData.hosts = (matchSet.host || []).join('\n')
                this.formData.paths = (matchSet.path || []).join('\n')
                this.formData.methods = (matchSet.method || []).join(' ')
                this.formData.protocol = matchSet.protocol || ''
                this.matchHeaderList = this.headersToList(matchSet.header)
                this.enableMatchHeaders = this.matchHeaderList.length > 0
            }
            // 解析 handle
            const handles = route.handle || []
            if (handles.length === 0) {
                this.isEditMode = false
                this.editingIndex = -1
                this.isOpen = true
                return
            }
            // 识别 [headers, <终止handler>] 组合
            let headersHandler: CaddyHandler | null = null
            let workHandles = handles
            if (handles.length === 2 && handles[0].handler === 'headers') {
                headersHandler = handles[0]
                workHandles = handles.slice(1)
            }
            if (workHandles.length > 1) {
                // 复杂路由无法解析，降级到 raw 模式
                this.formData.kind = 'raw'
                try { this.rawJson = JSON.stringify({ match: route.match, handle: handles }, null, 2) } catch { this.rawJson = '' }
                this.rawDirty = true
                this.isOpen = true
                return
            }
            const h = workHandles[0]
            switch (h.handler) {
                case 'reverse_proxy': {
                    this.formData.kind = 'reverse_proxy'
                    const rph = h as CaddyHandlerReverseProxy
                    this.formData.upstreams = (rph.upstreams || []).map(u => u.dial || '').filter(Boolean).join('\n')
                    this.syncSelectedFromText()
                    const transport = rph.transport
                    if (transport?.protocol === 'fastcgi') {
                        this.formData.fastcgi = true
                        this.formData.fastcgiRoot = transport.root || ''
                    } else if (transport) {
                        this.formData.dialTimeout = transport.dial_timeout || ''
                        this.formData.readTimeout = transport.response_header_timeout || ''
                        this.formData.writeTimeout = transport.write_timeout || ''
                    }
                    const rw = rph.rewrite
                    if (rw) {
                        this.formData.proxyRewriteEnabled = true
                        this.formData.proxyRewriteMethod = rw.method || ''
                        this.formData.proxyRewriteUri = rw.uri || ''
                        this.formData.proxyStripPathPrefix = rw.strip_path_prefix || ''
                        this.formData.proxyStripPathSuffix = rw.strip_path_suffix || ''
                        if (rw.uri_substring?.[0]) {
                            this.formData.proxyUriSubstringFind = rw.uri_substring[0].find || ''
                            this.formData.proxyUriSubstringReplace = rw.uri_substring[0].replace || ''
                        }
                    }
                    break
                }
                case 'file_server': {
                    this.formData.kind = 'file_server'
                    const fsh = h as CaddyHandlerFileServer
                    this.formData.root = fsh.root || ''
                    this.formData.browse = 'browse' in fsh
                    break
                }
                case 'static_response': {
                    this.formData.kind = 'static_response'
                    const srh = h as CaddyHandlerStaticResponse
                    const sc = srh.status_code
                    this.formData.statusCode = typeof sc === 'number' ? sc : (typeof sc === 'string' ? parseInt(sc) : 200)
                    this.formData.body = srh.body || ''
                    break
                }
                case 'rewrite': {
                    this.formData.kind = 'rewrite'
                    const rwh = h as CaddyHandlerRewrite
                    this.formData.rewriteUri = rwh.uri || ''
                    this.formData.stripPathPrefix = rwh.strip_path_prefix || ''
                    this.formData.stripPathSuffix = rwh.strip_path_suffix || ''
                    if (rwh.uri_substring?.[0]) {
                        this.formData.uriSubstringFind = rwh.uri_substring[0].find || ''
                        this.formData.uriSubstringReplace = rwh.uri_substring[0].replace || ''
                    }
                    break
                }
                default: {
                    // 未知 handler 类型，降级到 raw 模式
                    this.formData.kind = 'raw'
                    try { this.rawJson = JSON.stringify({ match: route.match, handle: handles }, null, 2) } catch { this.rawJson = '' }
                    this.rawDirty = true
                    this.isOpen = true
                    return
                }
            }
            // 解析 headers 中间件
            if (headersHandler) {
                const hh = headersHandler as CaddyHandlerHeaders
                if (hh.request) this.formData.requestHeaders = this.parseHeadersOps(hh.request)
                if (hh.response) this.formData.responseHeaders = this.parseHeadersOps(hh.response)
                this.formData.enableHeaders = true
            }
        } else {
            this.isEditMode = false
            this.editingIndex = -1
        }
        this.isOpen = true
    }

    // parseHeadersOps 从 Caddy headers handler 的 set/add/delete 结构解析为 CaddyHeaderOp 列表
    parseHeadersOps(m: Record<string, unknown>): CaddyHeaderOp[] {
        const ops: CaddyHeaderOp[] = []
        const setMap = m.set as Record<string, string[]> | undefined
        const addMap = m.add as Record<string, string[]> | undefined
        const delArr = m.delete as string[] | undefined
        if (setMap) for (const [k, v] of Object.entries(setMap)) ops.push({ op: 'set', field: k, value: Array.isArray(v) ? v[0] || '' : String(v) })
        if (addMap) for (const [k, v] of Object.entries(addMap)) for (const val of (Array.isArray(v) ? v : [v])) ops.push({ op: 'add', field: k, value: String(val) })
        if (delArr) for (const f of delArr) ops.push({ op: 'delete', field: f, value: '' })
        return ops
    }

    buildPayload(): CaddyRouteUpsert | null {
        // raw 模式：直接解析用户编辑的 JSON
        if (this.formData.kind === 'raw') {
            if (!this.rawJson.trim()) {
                this.portal.showNotification('error', '请输入路由 JSON')
                return null
            }
            let parsed: Record<string, unknown>
            try {
                parsed = JSON.parse(this.rawJson)
            } catch {
                this.portal.showNotification('error', 'JSON 格式错误')
                return null
            }
            const handle = parsed.handle as CaddyHandler[] | undefined
            if (!Array.isArray(handle) || !handle.length) {
                this.portal.showNotification('error', 'handle 字段不能为空')
                return null
            }
            return {
                match: parsed.match as CaddyRouteUpsert['match'],
                handle
            }
        }

        // 表单模式：从 buildRawFromCurrent 生成 handle
        const splitLines = (s: string) => s.split('\n').map(t => t.trim()).filter(Boolean)
        const splitSpaces = (s: string) => s.split(/\s+/).map(t => t.trim()).filter(Boolean)
        const hosts = splitLines(this.formData.hosts)
        const paths = splitLines(this.formData.paths)
        const methods = splitSpaces(this.formData.methods).map(s => s.toUpperCase())
        const headers = this.listToHeaders(this.matchHeaderList)
        const protocol = this.formData.protocol || undefined
        const matchSet: Record<string, unknown> = {}
        if (hosts.length) matchSet.host = hosts
        if (paths.length) matchSet.path = paths
        if (methods.length) matchSet.method = methods
        if (headers) matchSet.header = headers
        if (protocol) matchSet.protocol = protocol
        const match = Object.keys(matchSet).length ? [matchSet] : undefined

        const rawStr = this.buildRawFromCurrent()
        if (!rawStr) {
            this.portal.showNotification('error', '请完善处理器配置')
            return null
        }
        let handle: unknown[]
        try {
            handle = JSON.parse(rawStr)
        } catch {
            this.portal.showNotification('error', 'JSON 解析失败')
            return null
        }
        if (!Array.isArray(handle) || !handle.length) {
            this.portal.showNotification('error', '处理器不能为空')
            return null
        }
        return { match, handle: handle as CaddyHandler[] }
    }

    async handleConfirm() {
        const payload = this.buildPayload()
        if (!payload) return

        this.modalLoading = true
        try {
            if (this.isEditMode) {
                await api.caddyRouteUpdate(this.editingIndex, payload)
                this.portal.showNotification('success', '路由更新成功')
            } else {
                await api.caddyRouteCreate(payload)
                this.portal.showNotification('success', '路由创建成功')
            }
            this.isOpen = false
            this.$emit('success')
        } catch (e: unknown) {
            this.portal.showNotification('error', (e instanceof Error ? e.message : '') || '操作失败')
        } finally {
            this.modalLoading = false
        }
    }
}

export default toNative(RouteEditModal)
</script>

<template>
  <BaseModal v-model="isOpen" :title="isEditMode ? '编辑路由' : '新建路由'" :loading="modalLoading" confirm-class="btn-indigo" @confirm="handleConfirm">
    <div class="space-y-5 p-1">
      <!-- ── 匹配条件 ── -->
      <div class="space-y-4">
        <p class="section-title">匹配条件</p>

        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="form-label">Host</label>
            <textarea v-model="formData.hosts" rows="2" class="input font-mono text-sm" placeholder="请输入 Host（可选）"></textarea>
            <p class="text-xs text-slate-400 mt-1">每行一个，留空匹配所有，例如：example.com 或 *.example.com</p>
          </div>
          <div>
            <label class="form-label">Path</label>
            <textarea v-model="formData.paths" rows="2" class="input font-mono text-sm" placeholder="请输入 Path（可选）"></textarea>
            <p class="text-xs text-slate-400 mt-1">每行一个，支持 * 通配符，例如：/api/* 或 /static/*</p>
          </div>
        </div>

        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="form-label">Method</label>
            <input v-model="formData.methods" type="text" class="input font-mono text-sm" placeholder="请输入 Method（可选）" />
            <p class="text-xs text-slate-400 mt-1">空格分隔，留空匹配所有，例如：GET POST</p>
          </div>
          <div>
            <label class="form-label">协议</label>
            <div class="flex gap-2 mt-0.5">
              <button
                v-for="proto in ['', 'http', 'https']"
                :key="proto"
                type="button"
                :class="['btn-proto', formData.protocol === proto ? 'btn-proto-active' : 'btn-proto-inactive']"
                @click="formData.protocol = proto"
              >
                {{ proto === '' ? '不限' : proto.toUpperCase() }}
              </button>
            </div>
            <p class="text-xs text-slate-400 mt-1">留空匹配所有</p>
          </div>
        </div>

        <ToggleCard
          v-model="enableMatchHeaders"
          label="匹配请求头"
          desc="仅转发包含指定请求头的请求，留空匹配所有"
          @update:model-value="(v: boolean) => { if (!v) matchHeaderList = []; else if (!matchHeaderList.length) addMatchHeaderRow() }"
        >
          <div class="space-y-2">
            <div v-for="(entry, idx) in matchHeaderList" :key="idx" class="flex gap-2 items-center">
              <input v-model="entry.key" type="text" class="input font-mono text-sm flex-1" placeholder="请求头名称" />
              <input v-model="entry.value" type="text" class="input font-mono text-sm flex-1" placeholder="值（留空表示存在即匹配）" />
              <button type="button" class="text-slate-400 hover:text-red-500 shrink-0" @click="removeMatchHeaderRow(idx)"><i class="fas fa-trash text-sm"></i></button>
            </div>
            <button type="button" class="btn-add-row" @click="addMatchHeaderRow">
              <i class="fas fa-plus text-xs"></i>添加匹配请求头
            </button>
          </div>
        </ToggleCard>
      </div>

      <!-- ── 处理器 ── -->
      <div class="border-t border-slate-100 pt-4 space-y-4">
        <p class="section-title">处理器</p>

        <div class="grid grid-cols-2 md:grid-cols-3 gap-3">
          <button v-for="item in handlerKindCards" :key="item.value" type="button" :class="[modeCardClass(item), 'flex flex-col']" @click="setKind(item.value)">
            <div class="flex items-center gap-2 mb-1.5">
              <div :class="modeCardIconClass(item)"><i class="fas text-sm" :class="item.icon"></i></div>
              <span class="text-sm font-semibold leading-tight">{{ item.title }}</span>
            </div>
            <div class="text-xs opacity-75 leading-4">{{ item.desc }}</div>
          </button>
        </div>

        <!-- reverse_proxy -->
        <div v-if="formData.kind === 'reverse_proxy'" class="space-y-3">
          <div v-if="canLoadDockerContainers">
            <label class="form-label">选择容器与端口</label>
            <div class="grid grid-cols-[2fr_1fr] gap-2">
              <ContainerSelect :model-value="formData.upstreamHost" :containers="containers" placeholder="请输入容器名（可选）" @update:model-value="setUpstreamHost" />
              <ContainerPortSelect :model-value="formData.upstreamPort" :ports="getPortsByHost(formData.upstreamHost)" placeholder="请输入端口（可选）" @update:model-value="setUpstreamPort" />
            </div>
            <p class="text-xs text-slate-400 mt-1">可输入容器名或 IP 地址，选择后自动填充下方第一个上游，也可手动输入</p>
          </div>
          <div>
            <label class="form-label">上游地址 <span class="text-red-500">*</span></label>
            <textarea v-model="formData.upstreams" rows="3" class="input font-mono text-sm" placeholder="请输入上游地址（每行一个）" @input="syncSelectedFromText"></textarea>
            <p class="text-xs text-slate-400 mt-1">每行一个 host:port，多个上游自动轮询，例如：backend1:8080</p>
          </div>
          <div v-if="!formData.fastcgi">
            <div class="grid grid-cols-3 gap-3">
              <div>
                <label class="form-label">连接超时</label>
                <input v-model="formData.dialTimeout" type="text" class="input font-mono text-sm" placeholder="如 10s、5m" />
              </div>
              <div>
                <label class="form-label">响应头超时</label>
                <input v-model="formData.readTimeout" type="text" class="input font-mono text-sm" placeholder="如 10s、5m" />
              </div>
              <div>
                <label class="form-label">写入超时</label>
                <input v-model="formData.writeTimeout" type="text" class="input font-mono text-sm" placeholder="如 10s、5m" />
              </div>
            </div>
          </div>
          <ToggleCard v-model="formData.fastcgi" label="FastCGI 协议" desc="适用于 PHP-FPM 等 FastCGI 进程">
            <label class="form-label">FastCGI 文档根目录</label>
            <input v-model="formData.fastcgiRoot" type="text" class="input font-mono text-sm" placeholder="请输入 FastCGI 根目录（可选）" />
            <p class="text-xs text-slate-400 mt-1">设置 <code class="px-1 bg-slate-100 rounded">DOCUMENT_ROOT</code>，留空不传 root，例如：/var/www/html</p>
          </ToggleCard>
          <ToggleCard v-model="formData.proxyRewriteEnabled" label="代理前 URI 重写">
            <template #desc>写入 <code class="px-1 bg-slate-100 rounded">reverse_proxy.rewrite</code>，仅影响转发给上游的请求 URI</template>
            <div class="space-y-4">
              <!-- 覆盖请求方法 -->
              <div>
                <label class="form-label">覆盖请求方法 <span class="text-slate-400 font-normal">（可选）</span></label>
                <input v-model="formData.proxyRewriteMethod" type="text" class="input font-mono text-sm" placeholder="如 GET、POST，留空不修改" />
              </div>
              <!-- URI 整体替换 -->
              <div>
                <label class="form-label">URI 整体替换 <span class="text-slate-400 font-normal">（可选）</span></label>
                <input v-model="formData.proxyRewriteUri" type="text" class="input font-mono text-sm" placeholder="如 /backend/{http.request.uri.path.1}" />
                <p class="text-xs text-slate-400 mt-1">完整替换转发给上游的 URI，留空不修改</p>
              </div>
              <!-- 路径前后缀裁剪 -->
              <div>
                <p class="form-label">路径前后缀裁剪 <span class="text-slate-400 font-normal">（可选）</span></p>
                <div class="grid grid-cols-2 gap-3 mt-1">
                  <div>
                    <label class="form-label">去掉前缀</label>
                    <input v-model="formData.proxyStripPathPrefix" type="text" class="input font-mono text-sm" placeholder="如 /api" />
                  </div>
                  <div>
                    <label class="form-label">去掉后缀</label>
                    <input v-model="formData.proxyStripPathSuffix" type="text" class="input font-mono text-sm" placeholder="如 .php" />
                  </div>
                </div>
              </div>
              <!-- 子串查找替换 -->
              <div>
                <p class="form-label">子串查找替换 <span class="text-slate-400 font-normal">（可选）</span></p>
                <div class="grid grid-cols-2 gap-3 mt-1">
                  <div>
                    <label class="form-label">查找</label>
                    <input v-model="formData.proxyUriSubstringFind" type="text" class="input font-mono text-sm" placeholder="如 /old" />
                  </div>
                  <div>
                    <label class="form-label">替换为</label>
                    <input v-model="formData.proxyUriSubstringReplace" type="text" class="input font-mono text-sm" placeholder="如 /new" />
                  </div>
                </div>
              </div>
            </div>
          </ToggleCard>
        </div>

        <!-- file_server -->
        <div v-else-if="formData.kind === 'file_server'" class="space-y-3">
          <div>
            <label class="form-label">根目录 <span class="text-red-500">*</span></label>
            <input v-model="formData.root" type="text" class="input font-mono text-sm" placeholder="请输入根目录" />
            <p class="text-xs text-slate-400 mt-1">静态文件根目录，例如：/var/www/html</p>
          </div>
          <ToggleCard v-model="formData.browse" label="启用目录浏览" desc="允许访客浏览目录内文件列表" />
        </div>

        <!-- static_response -->
        <div v-else-if="formData.kind === 'static_response'" class="space-y-3">
          <div>
            <label class="form-label">状态码</label>
            <input v-model.number="formData.statusCode" type="number" min="100" max="599" class="input" placeholder="请输入状态码" />
            <p class="text-xs text-slate-400 mt-1">HTTP 状态码，例如：200</p>
          </div>
          <div>
            <label class="form-label">响应体</label>
            <textarea v-model="formData.body" rows="4" class="input font-mono text-sm" placeholder="请输入响应体"></textarea>
            <p class="text-xs text-slate-400 mt-1">留空则返回默认响应，例如：OK</p>
          </div>
        </div>

        <!-- rewrite -->
        <div v-else-if="formData.kind === 'rewrite'" class="space-y-3">
          <div>
            <label class="form-label">URI 替换</label>
            <input v-model="formData.rewriteUri" type="text" class="input font-mono text-sm" placeholder="请输入 URI 替换规则" />
            <p class="text-xs text-slate-400 mt-1">完整替换请求 URI，支持 Caddy 占位符，例如：/new-path/{http.request.uri.path.1}</p>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="form-label">去掉路径前缀</label>
              <input v-model="formData.stripPathPrefix" type="text" class="input font-mono text-sm" placeholder="请输入要去掉的前缀" />
              <p class="text-xs text-slate-400 mt-1">例如：<code class="px-1 bg-slate-100 rounded">/api</code>，效果：<code class="px-1 bg-slate-100 rounded">/api/v1/foo</code> → <code class="px-1 bg-slate-100 rounded">/v1/foo</code></p>
            </div>
            <div>
              <label class="form-label">去掉路径后缀</label>
              <input v-model="formData.stripPathSuffix" type="text" class="input font-mono text-sm" placeholder="请输入要去掉的后缀" />
              <p class="text-xs text-slate-400 mt-1">从路径末尾去掉指定后缀，例如：.php</p>
            </div>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="form-label">子串查找</label>
              <input v-model="formData.uriSubstringFind" type="text" class="input font-mono text-sm" placeholder="请输入要查找的子串" />
              <p class="text-xs text-slate-400 mt-1">例如：/old</p>
            </div>
            <div>
              <label class="form-label">子串替换</label>
              <input v-model="formData.uriSubstringReplace" type="text" class="input font-mono text-sm" placeholder="请输入替换内容" />
              <p class="text-xs text-slate-400 mt-1">例如：/new</p>
            </div>
          </div>
        </div>

        <!-- raw -->
        <div v-else-if="formData.kind === 'raw'" class="space-y-2">
          <label class="form-label">路由 JSON <span class="text-red-500">*</span></label>
          <textarea v-model="rawJson" rows="12" class="input font-mono text-xs" placeholder='{ "match": [...], "handle": [...] }' @input="rawDirty = true"></textarea>
          <p class="text-xs text-slate-400">直接编辑完整的 Caddy 路由 JSON，match 字段可选</p>
        </div>
      </div>

      <!-- ── 请求头 / 响应头操作 ── -->
      <ToggleCard v-if="formData.kind !== 'raw'" v-model="formData.enableHeaders" label="配置请求头/响应头操作">
        <template #desc>串联中间件，在处理器前执行，支持占位符如 <code class="px-1 bg-slate-100 rounded">{http.request.remote.host}</code></template>
        <div class="space-y-4">
          <!-- 请求头 -->
          <div>
            <label class="form-label">请求头</label>
            <p class="text-xs text-slate-400 mt-1 mb-2">例如：字段 X-Real-IP，值 {http.request.remote.host}</p>
            <div v-if="formData.requestHeaders.length" class="space-y-2 mb-2">
              <div v-for="(op, idx) in formData.requestHeaders" :key="idx" class="flex gap-2 items-center">
                <select v-model="op.op" class="input text-sm w-24 shrink-0">
                  <option value="set">覆盖</option>
                  <option value="add">追加</option>
                  <option value="delete">Delete</option>
                </select>
                <input v-model="op.field" type="text" class="input font-mono text-sm flex-1" placeholder="请输入请求头名称" />
                <input v-if="op.op !== 'delete'" v-model="op.value" type="text" class="input font-mono text-sm flex-1" placeholder="请输入请求头值（支持占位符）" />
                <div v-else class="flex-1"></div>
                <button type="button" class="text-slate-400 hover:text-red-500 shrink-0" @click="removeHeaderOp('request', idx)"><i class="fas fa-trash text-sm"></i></button>
              </div>
            </div>
            <button type="button" class="btn-add-row" @click="addHeaderOp('request')">
              <i class="fas fa-plus text-xs"></i>添加请求头操作
            </button>
          </div>
          <!-- 响应头 -->
          <div>
            <label class="form-label">响应头</label>
            <p class="text-xs text-slate-400 mt-1 mb-2">例如：字段 X-Frame-Options，值 SAMEORIGIN</p>
            <div v-if="formData.responseHeaders.length" class="space-y-2 mb-2">
              <div v-for="(op, idx) in formData.responseHeaders" :key="idx" class="flex gap-2 items-center">
                <select v-model="op.op" class="input text-sm w-24 shrink-0">
                  <option value="set">覆盖</option>
                  <option value="add">追加</option>
                  <option value="delete">删除</option>
                </select>
                <input v-model="op.field" type="text" class="input font-mono text-sm flex-1" placeholder="请输入响应头名称" />
                <input v-if="op.op !== 'delete'" v-model="op.value" type="text" class="input font-mono text-sm flex-1" placeholder="请输入响应头值" />
                <div v-else class="flex-1"></div>
                <button type="button" class="text-slate-400 hover:text-red-500 shrink-0" @click="removeHeaderOp('response', idx)"><i class="fas fa-trash text-sm"></i></button>
              </div>
            </div>
            <button type="button" class="btn-add-row" @click="addHeaderOp('response')">
              <i class="fas fa-plus text-xs"></i>添加响应头操作
            </button>
          </div>
        </div>
      </ToggleCard>
    </div>
  
    <template #confirm-text>
      确认{{ isEditMode ? '更新' : '新建' }}
    </template>
  </BaseModal>
</template>
