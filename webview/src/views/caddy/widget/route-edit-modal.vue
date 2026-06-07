<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { CaddyRoute, CaddyRouteUpsert, CaddyHandlerKind, CaddyHandlerKindCard, CaddyHeaderOp, DockerContainerInfo } from '@/service/types'

import { parseHostPort } from '@/helper/utils'

import BaseModal from '@/component/modal.vue'

import ContainerPortSelect from '@/views/docker/widget/container-port-select.vue'
import ContainerSelect from '@/views/docker/widget/container-select.vue'

const HANDLER_KIND_CARDS: CaddyHandlerKindCard[] = [
    { value: 'reverse_proxy', title: '反向代理', desc: '将请求转发到一个或多个上游服务器', icon: 'fa-diagram-project', tone: 'indigo' },
    { value: 'file_server', title: '静态文件服务', desc: '提供本地目录中的静态文件访问', icon: 'fa-folder-open', tone: 'emerald' },
    { value: 'static_response', title: '静态响应', desc: '直接返回指定的 HTTP 状态码和响应体', icon: 'fa-bolt', tone: 'amber' },
    { value: 'rewrite', title: 'URI 重写', desc: '重写请求 URI、去除路径前后缀或替换子串', icon: 'fa-pen-to-square', tone: 'violet' }
]

const TONE_CARD_ACTIVE: Record<string, string> = {
    indigo: 'border-indigo-300 bg-indigo-50 text-indigo-700',
    emerald: 'border-emerald-300 bg-emerald-50 text-emerald-700',
    amber: 'border-amber-300 bg-amber-50 text-amber-700',
    violet: 'border-violet-300 bg-violet-50 text-violet-700',
    rose: 'border-rose-300 bg-rose-50 text-rose-700'
}
const TONE_ICON_ACTIVE: Record<string, string> = {
    indigo: 'bg-indigo-100', emerald: 'bg-emerald-100', amber: 'bg-amber-100', violet: 'bg-violet-100', rose: 'bg-rose-100'
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
    rawText: '',
    enableHeaders: false
})

@Component({
    expose: ['show'],
    components: { BaseModal, ContainerPortSelect, ContainerSelect },
    emits: ['success']
})
class RouteEditModal extends Vue {
    portal = usePortal()

    isOpen = false
    modalLoading = false
    isEditMode = false
    editingIndex = -1
    containers: DockerContainerInfo[] = []
    showRawPreview = false

    // 匹配请求头多行编辑列表
    matchHeaderList: { key: string; value: string }[] = []

    formData = defaultFormData()

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
        this.formData.kind = kind
    }

    // 根据当前表单数据生成 handle 数组 JSON（用于原始 JSON 预览）
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

    buildProxyRewritePayload() {
        const f = this.formData
        if (!f.proxyRewriteEnabled) return undefined
        const payload = {
            method: f.proxyRewriteMethod.trim() ? f.proxyRewriteMethod.trim().toUpperCase() : undefined,
            rewriteUri: f.proxyRewriteUri.trim() || undefined,
            stripPathPrefix: f.proxyStripPathPrefix.trim() || undefined,
            stripPathSuffix: f.proxyStripPathSuffix.trim() || undefined,
            uriSubstringFind: f.proxyUriSubstringFind.trim() || undefined,
            uriSubstringReplace: f.proxyUriSubstringFind.trim() ? (f.proxyUriSubstringReplace.trim() || undefined) : undefined
        }
        return Object.values(payload).some(Boolean) ? payload : undefined
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
        try {
            const res = await api.dockerContainerList()
            this.containers = (res.payload || []).filter(c => c.state === 'running')
        } catch {
            this.containers = []
        }
    }

    show(route: CaddyRoute | null) {
        Object.assign(this.formData, defaultFormData())
        this.showRawPreview = false
        this.matchHeaderList = []
        void this.loadContainers()
        if (route) {
            this.isEditMode = true
            this.editingIndex = route.index
            const m = route.match
            this.formData.hosts = (m?.hosts || []).join('\n')
            this.formData.paths = (m?.paths || []).join('\n')
            this.formData.methods = (m?.methods || []).join(' ')
            this.formData.protocol = m?.protocol || ''
            this.matchHeaderList = this.headersToList(m?.headers)
            const h = route.handler
            if (h) {
                this.formData.kind = h.kind
                this.formData.upstreams = (h.upstreams || []).join('\n')
                this.syncSelectedFromText()
                this.formData.fastcgi = !!h.fastcgi
                this.formData.fastcgiRoot = h.fastcgiRoot || ''
                this.formData.dialTimeout = h.dialTimeout || ''
                this.formData.readTimeout = h.readTimeout || ''
                this.formData.writeTimeout = h.writeTimeout || ''
                const proxyRewrite = h.proxyRewrite
                this.formData.proxyRewriteEnabled = !!proxyRewrite
                this.formData.proxyRewriteMethod = proxyRewrite?.method || ''
                this.formData.proxyRewriteUri = proxyRewrite?.rewriteUri || ''
                this.formData.proxyStripPathPrefix = proxyRewrite?.stripPathPrefix || ''
                this.formData.proxyStripPathSuffix = proxyRewrite?.stripPathSuffix || ''
                this.formData.proxyUriSubstringFind = proxyRewrite?.uriSubstringFind || ''
                this.formData.proxyUriSubstringReplace = proxyRewrite?.uriSubstringReplace || ''

                this.formData.root = h.root || ''
                this.formData.browse = !!h.browse
                this.formData.statusCode = h.statusCode || 200
                this.formData.body = h.body || ''
                this.formData.rewriteUri = h.rewriteUri || ''
                this.formData.stripPathPrefix = h.stripPathPrefix || ''
                this.formData.stripPathSuffix = h.stripPathSuffix || ''
                this.formData.uriSubstringFind = h.uriSubstringFind || ''
                this.formData.uriSubstringReplace = h.uriSubstringReplace || ''
                this.formData.requestHeaders = h.requestHeaders ? [...h.requestHeaders] : []
                this.formData.responseHeaders = h.responseHeaders ? [...h.responseHeaders] : []
                // 如果已有header配置，自动启用header操作
                this.formData.enableHeaders = !!(h.requestHeaders || h.responseHeaders)
                if (h.kind === 'raw') {
                    try { this.formData.rawText = h.raw !== null && h.raw !== undefined ? JSON.stringify(h.raw, null, 2) : '' }
                    catch { this.formData.rawText = '' }
                }
            }
        } else {
            this.isEditMode = false
            this.editingIndex = -1
        }
        this.isOpen = true
    }

    buildPayload(): CaddyRouteUpsert | null {
        const splitLines = (s: string) => s.split('\n').map(t => t.trim()).filter(Boolean)
        const splitSpaces = (s: string) => s.split(/\s+/).map(t => t.trim()).filter(Boolean)

        const hosts = splitLines(this.formData.hosts)
        const paths = splitLines(this.formData.paths)
        const methods = splitSpaces(this.formData.methods).map(s => s.toUpperCase())
        const headers = this.listToHeaders(this.matchHeaderList)
        const protocol = this.formData.protocol || undefined

        const match = (hosts.length || paths.length || methods.length || headers || protocol)
            ? { hosts, paths, methods, headers, protocol }
            : undefined

        const f = this.formData
        let handler: CaddyRouteUpsert['handler']
        switch (f.kind) {
            case 'reverse_proxy': {
                const upstreams = splitLines(f.upstreams)
                if (!upstreams.length) {
                    this.portal.showNotification('error', '请填写至少一个上游 host:port')
                    return null
                }
                handler = { kind: 'reverse_proxy',
                    upstreams,
                    fastcgi: f.fastcgi || undefined,
                    fastcgiRoot: f.fastcgi && f.fastcgiRoot.trim() ? f.fastcgiRoot.trim() : undefined,
                    dialTimeout: !f.fastcgi && f.dialTimeout.trim() ? f.dialTimeout.trim() : undefined,
                    readTimeout: !f.fastcgi && f.readTimeout.trim() ? f.readTimeout.trim() : undefined,
                    writeTimeout: !f.fastcgi && f.writeTimeout.trim() ? f.writeTimeout.trim() : undefined,
                    proxyRewrite: this.buildProxyRewritePayload(),
                    requestHeaders: f.enableHeaders ? this.buildHeaderOps('request') : undefined,
                    responseHeaders: f.enableHeaders ? this.buildHeaderOps('response') : undefined
                }
                break
            }
            case 'file_server': {
                if (!f.root.trim()) {
                    this.portal.showNotification('error', '请填写文件根目录')
                    return null
                }
                handler = { kind: 'file_server', root: f.root.trim(), browse: f.browse,
                    requestHeaders: f.enableHeaders ? this.buildHeaderOps('request') : undefined,
                    responseHeaders: f.enableHeaders ? this.buildHeaderOps('response') : undefined
                }
                break
            }
            case 'static_response': {
                if (!f.statusCode && !f.body) {
                    this.portal.showNotification('error', '请填写状态码或响应体')
                    return null
                }
                handler = { kind: 'static_response', statusCode: f.statusCode || 200, body: f.body,
                    requestHeaders: f.enableHeaders ? this.buildHeaderOps('request') : undefined,
                    responseHeaders: f.enableHeaders ? this.buildHeaderOps('response') : undefined
                }
                break
            }
            case 'rewrite': {
                if (!f.rewriteUri.trim() && !f.stripPathPrefix.trim() && !f.stripPathSuffix.trim() && !f.uriSubstringFind.trim()) {
                    this.portal.showNotification('error', '请至少填写一个重写规则')
                    return null
                }
                handler = {
                    kind: 'rewrite',
                    rewriteUri: f.rewriteUri.trim() || undefined,
                    stripPathPrefix: f.stripPathPrefix.trim() || undefined,
                    stripPathSuffix: f.stripPathSuffix.trim() || undefined,
                    uriSubstringFind: f.uriSubstringFind.trim() || undefined,
                    uriSubstringReplace: f.uriSubstringFind.trim() ? (f.uriSubstringReplace.trim() || undefined) : undefined,
                    requestHeaders: f.enableHeaders ? this.buildHeaderOps('request') : undefined,
                    responseHeaders: f.enableHeaders ? this.buildHeaderOps('response') : undefined
                }
                break
            }
            case 'raw': {
                let parsed: unknown
                try {
                    parsed = JSON.parse(f.rawText)
                } catch {
                    this.portal.showNotification('error', '原始 JSON 解析失败')
                    return null
                }
                if (!Array.isArray(parsed) || !parsed.length) {
                    this.portal.showNotification('error', '原始 handle 必须为非空数组')
                    return null
                }
                handler = { kind: 'raw', raw: parsed }
                break
            }
        }

        return { match, handler }
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

        <div>
          <label class="form-label">匹配请求头 <span class="text-slate-400 font-normal">（可选）</span></label>
          <div v-if="matchHeaderList.length" class="space-y-2 mb-2">
            <div v-for="(entry, idx) in matchHeaderList" :key="idx" class="flex gap-2 items-center">
              <input v-model="entry.key" type="text" class="input font-mono text-sm flex-1" placeholder="请输入请求头名称" />
              <input v-model="entry.value" type="text" class="input font-mono text-sm flex-1" placeholder="请输入值（可选，留空表示存在即匹配）" />
              <button type="button" class="text-slate-400 hover:text-red-500 shrink-0" @click="removeMatchHeaderRow(idx)"><i class="fas fa-trash text-sm"></i></button>
            </div>
          </div>
          <button type="button" class="btn-add-row" @click="addMatchHeaderRow">
            <i class="fas fa-plus text-xs"></i>添加匹配请求头
          </button>
        </div>
      </div>

      <!-- ── 处理器 ── -->
      <div class="border border-slate-200 rounded-xl p-4 space-y-4">
        <p class="section-title">处理器</p>

        <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
          <button v-for="item in handlerKindCards" :key="item.value" type="button" :class="[modeCardClass(item), 'flex flex-col']" @click="setKind(item.value)">
            <div class="flex items-center gap-2 mb-1.5">
              <div :class="modeCardIconClass(item)"><i class="fas text-sm" :class="item.icon"></i></div>
              <span class="text-sm font-semibold leading-tight">{{ item.title }}</span>
            </div>
            <div class="text-xs opacity-75 leading-4">{{ item.desc }}</div>
          </button>
        </div>

        <!-- raw 模式提示 -->
        <div v-if="formData.kind === 'raw'" class="flex items-center gap-2 px-3 py-2 rounded-lg bg-amber-50 border border-amber-200 text-amber-700">
          <i class="fas fa-triangle-exclamation text-sm"></i>
          <span class="text-sm">当前路由包含多个 handler 或不支持的类型，仅支持直接编辑 JSON</span>
        </div>

        <!-- reverse_proxy -->
        <div v-if="formData.kind === 'reverse_proxy'" class="space-y-3">
          <div>
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
          <div class="toggle-row">
            <div>
              <span class="text-sm text-slate-600">FastCGI 协议</span>
              <p class="text-xs text-slate-400 mt-0.5">适用于 PHP-FPM 等 FastCGI 进程</p>
            </div>
            <button type="button" class="toggle" :class="{ 'toggle-on': formData.fastcgi }" role="switch" :aria-checked="formData.fastcgi" @click="formData.fastcgi = !formData.fastcgi">
              <span class="toggle-thumb" />
            </button>
          </div>
          <div v-if="formData.fastcgi">
            <label class="form-label">FastCGI 文档根目录</label>
            <input v-model="formData.fastcgiRoot" type="text" class="input font-mono text-sm" placeholder="请输入 FastCGI 根目录（可选）" />
            <p class="text-xs text-slate-400 mt-1">设置 <code class="px-1 bg-slate-100 rounded">DOCUMENT_ROOT</code>，留空不传 root，例如：/var/www/html</p>
          </div>
          <div class="toggle-row">
            <div>
              <span class="text-sm text-slate-600">代理前 URI 重写</span>
              <p class="text-xs text-slate-400 mt-0.5">写入 <code class="px-1 bg-slate-100 rounded">reverse_proxy.rewrite</code>，仅影响转发给上游的请求 URI</p>
            </div>
            <button type="button" class="toggle" :class="{ 'toggle-on': formData.proxyRewriteEnabled }" role="switch" :aria-checked="formData.proxyRewriteEnabled" @click="formData.proxyRewriteEnabled = !formData.proxyRewriteEnabled">
              <span class="toggle-thumb" />
            </button>
          </div>
          <div v-if="formData.proxyRewriteEnabled" class="space-y-3 rounded-lg border border-indigo-100 bg-indigo-50/40 p-3">
            <div class="grid grid-cols-[1fr_2fr] gap-3">
              <div>
                <label class="form-label">请求方法</label>
                <input v-model="formData.proxyRewriteMethod" type="text" class="input font-mono text-sm" placeholder="如 GET" />
              </div>
              <div>
                <label class="form-label">URI 替换</label>
                <input v-model="formData.proxyRewriteUri" type="text" class="input font-mono text-sm" placeholder="请输入 URI 替换规则" />
                <p class="text-xs text-slate-400 mt-1">完整替换转发给上游的 URI，例如：/backend/{http.request.uri.path.1}</p>
              </div>
            </div>
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="form-label">去掉路径前缀</label>
                <input v-model="formData.proxyStripPathPrefix" type="text" class="input font-mono text-sm" placeholder="请输入要去掉的前缀" />
              </div>
              <div>
                <label class="form-label">去掉路径后缀</label>
                <input v-model="formData.proxyStripPathSuffix" type="text" class="input font-mono text-sm" placeholder="请输入要去掉的后缀" />
              </div>
            </div>
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="form-label">子串查找</label>
                <input v-model="formData.proxyUriSubstringFind" type="text" class="input font-mono text-sm" placeholder="请输入要查找的子串" />
              </div>
              <div>
                <label class="form-label">子串替换</label>
                <input v-model="formData.proxyUriSubstringReplace" type="text" class="input font-mono text-sm" placeholder="请输入替换内容" />
              </div>
            </div>
          </div>
        </div>

        <!-- file_server -->
        <div v-else-if="formData.kind === 'file_server'" class="space-y-3">
          <div>
            <label class="form-label">根目录 <span class="text-red-500">*</span></label>
            <input v-model="formData.root" type="text" class="input font-mono text-sm" placeholder="请输入根目录" />
            <p class="text-xs text-slate-400 mt-1">静态文件根目录，例如：/var/www/html</p>
          </div>
          <div class="toggle-row">
            <div>
              <span class="text-sm text-slate-600">启用目录浏览</span>
              <p class="text-xs text-slate-400 mt-0.5">允许访客浏览目录内文件列表</p>
            </div>
            <button type="button" class="toggle" :class="{ 'toggle-on': formData.browse }" role="switch" :aria-checked="formData.browse" @click="formData.browse = !formData.browse">
              <span class="toggle-thumb" />
            </button>
          </div>
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
          <label class="form-label">原始 handle 数组（JSON）<span class="text-red-500">*</span></label>
          <textarea v-model="formData.rawText" rows="10" class="input font-mono text-xs"></textarea>
          <p class="text-xs text-slate-400">直接编辑 caddy json 中 routes[i].handle 的原始数组</p>
        </div>
      </div>

      <!-- ── 请求头 / 响应头操作 ── -->
      <div v-if="formData.kind !== 'raw'" class="toggle-row">
        <div>
          <span class="text-sm text-slate-600">配置请求头/响应头操作</span>
          <p class="text-xs text-slate-400 mt-0.5">串联中间件，在处理器前执行，支持占位符如 <code class="px-1 bg-slate-100 rounded">{http.request.remote.host}</code></p>
        </div>
        <button type="button" class="toggle" :class="{ 'toggle-on': formData.enableHeaders }" role="switch" :aria-checked="formData.enableHeaders" @click="formData.enableHeaders = !formData.enableHeaders">
          <span class="toggle-thumb" />
        </button>
      </div>
      <div v-if="formData.kind !== 'raw' && formData.enableHeaders" class="border border-slate-200 rounded-xl p-4 space-y-4">
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

      <!-- ── 原始 JSON 预览 ── -->
      <div v-if="formData.kind !== 'raw'" class="toggle-row">
        <div>
          <span class="text-sm text-slate-600">显示原始 JSON 预览</span>
          <p class="text-xs text-slate-400 mt-0.5">查看当前配置对应的原始 JSON 格式</p>
        </div>
        <button type="button" class="toggle" :class="{ 'toggle-on': showRawPreview }" role="switch" :aria-checked="showRawPreview" @click="showRawPreview = !showRawPreview">
          <span class="toggle-thumb" />
        </button>
      </div>
      <div v-if="formData.kind !== 'raw' && showRawPreview">
        <textarea :value="buildRawFromCurrent()" rows="10" class="input font-mono text-xs" readonly></textarea>
      </div>
    </div>
  
    <template #confirm-text>
      确认{{ isEditMode ? '更新' : '新建' }}
    </template>
  </BaseModal>
</template>