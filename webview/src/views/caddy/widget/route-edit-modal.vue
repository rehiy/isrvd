<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { CaddyRoute, CaddyRouteUpsert, CaddyHandlerKind, CaddyHandlerKindCard, DockerContainerInfo } from '@/service/types'

import BaseModal from '@/component/modal.vue'

import ContainerPortSelect from '@/views/docker/widget/container-port-select.vue'
import ContainerSelect from '@/views/docker/widget/container-select.vue'

import { usePortal } from '@/stores'

const HANDLER_KIND_CARDS: CaddyHandlerKindCard[] = [
    { value: 'reverse_proxy', title: '反向代理', desc: '将请求转发到一个或多个上游服务器', icon: 'fa-diagram-project', tone: 'indigo' },
    { value: 'file_server', title: '静态文件服务', desc: '提供本地目录中的静态文件访问', icon: 'fa-folder-open', tone: 'emerald' },
    { value: 'static_response', title: '静态响应', desc: '直接返回指定的 HTTP 状态码和响应体', icon: 'fa-bolt', tone: 'amber' },
    { value: 'raw', title: '原始 JSON', desc: '直接编辑 Caddy 原生 handle 数组', icon: 'fa-code', tone: 'slate' }
]

const TONE_CARD_ACTIVE: Record<string, string> = {
    indigo: 'border-indigo-300 bg-indigo-50 text-indigo-700',
    emerald: 'border-emerald-300 bg-emerald-50 text-emerald-700',
    amber: 'border-amber-300 bg-amber-50 text-amber-700',
    slate: 'border-slate-300 bg-slate-100 text-slate-700'
}
const TONE_ICON_ACTIVE: Record<string, string> = {
    indigo: 'bg-indigo-100', emerald: 'bg-emerald-100', amber: 'bg-amber-100', slate: 'bg-slate-200'
}

const defaultFormData = () => ({
    kind: 'reverse_proxy' as CaddyHandlerKind,
    hosts: '',
    paths: '',
    methods: '',
    upstreams: '',
    upstreamHost: '',
    upstreamPort: '',
    fastcgi: false,
    fastcgiRoot: '',
    root: '',
    browse: false,
    statusCode: 200,
    body: '',
    rawText: ''
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

    formData = defaultFormData()

    readonly handlerKindCards = HANDLER_KIND_CARDS

    // ─── 模式卡片样式 ───
    modeCardClass(item: CaddyHandlerKindCard) {
        const base = 'text-left rounded-xl border p-3 transition-colors'
        const active = this.formData.kind === item.value
        return `${base} ${active ? TONE_CARD_ACTIVE[item.tone] : 'border-slate-200 bg-white text-slate-600 hover:border-slate-300'}`
    }

    modeCardIconClass(item: CaddyHandlerKindCard) {
        const active = this.formData.kind === item.value
        return `w-8 h-8 rounded-lg flex items-center justify-center ${active ? TONE_ICON_ACTIVE[item.tone] : 'bg-slate-100'}`
    }

    setKind(kind: CaddyHandlerKind) {
        this.formData.kind = kind
    }

    getPortsByHost(host: string): string[] {
        return this.containers.find(c => c.name === host.trim())?.ports || []
    }

    parseUpstream(upstream: string) {
        const value = upstream.trim()
        const idx = value.lastIndexOf(':')
        if (idx <= 0) return { host: value, port: '' }
        return { host: value.slice(0, idx), port: value.slice(idx + 1) }
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
        void this.loadContainers()
        if (route) {
            this.isEditMode = true
            this.editingIndex = route.index
            const m = route.match
            this.formData.hosts = (m?.hosts || []).join('\n')
            this.formData.paths = (m?.paths || []).join('\n')
            this.formData.methods = (m?.methods || []).join(' ')
            const h = route.handler
            if (h) {
                this.formData.kind = h.kind
                this.formData.upstreams = (h.upstreams || []).join('\n')
                this.syncSelectedFromText()
                this.formData.fastcgi = !!h.fastcgi
                this.formData.fastcgiRoot = h.fastcgiRoot || ''
                this.formData.root = h.root || ''
                this.formData.browse = !!h.browse
                this.formData.statusCode = h.statusCode || 200
                this.formData.body = h.body || ''
                if (h.kind === 'raw' && h.raw) {
                    try { this.formData.rawText = JSON.stringify(h.raw, null, 2) } catch { this.formData.rawText = '' }
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

        const match = (hosts.length || paths.length || methods.length)
            ? { hosts, paths, methods }
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
                handler = { kind: 'reverse_proxy', upstreams, fastcgi: f.fastcgi || undefined, fastcgiRoot: f.fastcgi && f.fastcgiRoot.trim() ? f.fastcgiRoot.trim() : undefined }
                break
            }
            case 'file_server': {
                if (!f.root.trim()) {
                    this.portal.showNotification('error', '请填写文件根目录')
                    return null
                }
                handler = { kind: 'file_server', root: f.root.trim(), browse: f.browse }
                break
            }
            case 'static_response': {
                if (!f.statusCode && !f.body) {
                    this.portal.showNotification('error', '请填写状态码或响应体')
                    return null
                }
                handler = { kind: 'static_response', statusCode: f.statusCode || 200, body: f.body }
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
    <div class="space-y-4 p-1">
      <!-- 匹配条件：直接平铺，与 apisix 一致 -->
      <div><label class="form-label">Host（每行一个，留空匹配所有）</label><textarea v-model="formData.hosts" rows="2" class="input font-mono text-sm" placeholder="example.com&#10;*.example.com"></textarea></div>
      <div><label class="form-label">Path（每行一个，支持 * 通配符）</label><textarea v-model="formData.paths" rows="2" class="input font-mono text-sm" placeholder="/api/*&#10;/static/*"></textarea></div>
      <div><label class="form-label">Method（空格分隔，留空匹配所有）</label><input v-model="formData.methods" type="text" class="input font-mono text-sm" placeholder="GET POST" /></div>

      <!-- 处理器：含 mode cards，用框包裹（对应 apisix 上游配置框） -->
      <div class="border border-slate-200 rounded-xl p-4">
        <label class="block section-title">处理器类型</label>
        <div class="grid grid-cols-2 md:grid-cols-4 gap-3 mb-4">
          <button v-for="item in handlerKindCards" :key="item.value" type="button" :class="modeCardClass(item)" @click="setKind(item.value)">
            <div class="flex items-center gap-2 mb-1">
              <div :class="modeCardIconClass(item)"><i class="fas text-sm" :class="item.icon"></i></div>
              <span class="text-sm font-semibold">{{ item.title }}</span>
            </div>
            <div class="text-xs opacity-80 leading-5">{{ item.desc }}</div>
          </button>
        </div>

        <!-- reverse_proxy -->
        <div v-if="formData.kind === 'reverse_proxy'" class="space-y-3">
          <div>
            <label class="form-label">选择容器与端口</label>
            <div class="grid grid-cols-[2fr_1fr] gap-2 items-center">
              <ContainerSelect :model-value="formData.upstreamHost" :containers="containers" placeholder="127.0.0.1 或 容器名" @update:model-value="setUpstreamHost" />
              <ContainerPortSelect :model-value="formData.upstreamPort" :ports="getPortsByHost(formData.upstreamHost)" placeholder="80" @update:model-value="setUpstreamPort" />
            </div>
            <p class="text-xs text-slate-400 mt-1">选择后会自动填充下方第一个上游；也可手动输入多个 host:port。</p>
          </div>
          <div>
            <label class="form-label">上游 host:port（每行一个）<span class="text-red-500">*</span></label>
            <textarea v-model="formData.upstreams" rows="3" class="input font-mono text-sm" placeholder="backend1:8080&#10;backend2:8080" @input="syncSelectedFromText"></textarea>
            <p class="text-xs text-slate-400 mt-1">多个上游会做轮询负载</p>
          </div>
          <div class="toggle-row">
            <div>
              <span class="text-sm text-slate-600">使用 FastCGI 协议（PHP-FPM / fcgi）</span>
              <p class="text-xs text-slate-400 mt-0.5">启用后将使用 FastCGI 传输协议与上游通信，适用于 PHP-FPM 等 FastCGI 进程</p>
            </div>
            <button type="button" class="toggle" :class="{ 'toggle-on': formData.fastcgi }" role="switch" :aria-checked="formData.fastcgi" @click="formData.fastcgi = !formData.fastcgi">
              <span class="toggle-thumb" />
            </button>
          </div>
          <div v-if="formData.fastcgi">
            <label class="form-label">FastCGI 文档根目录</label>
            <input v-model="formData.fastcgiRoot" type="text" class="input font-mono text-sm" placeholder="/var/www/html（留空不传 root）" />
            <p class="text-xs text-slate-400 mt-1">上游 FastCGI 服务器的文档根目录，用于设置 <code class="px-1 bg-slate-100 rounded">DOCUMENT_ROOT</code></p>
          </div>
        </div>

        <!-- file_server -->
        <div v-else-if="formData.kind === 'file_server'" class="space-y-3">
          <div>
            <label class="form-label">根目录 <span class="text-red-500">*</span></label>
            <input v-model="formData.root" type="text" class="input font-mono text-sm" placeholder="/var/www/html" />
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
            <input v-model.number="formData.statusCode" type="number" min="100" max="599" class="input" placeholder="200" />
          </div>
          <div>
            <label class="form-label">响应体</label>
            <textarea v-model="formData.body" rows="4" class="input font-mono text-sm" placeholder="OK"></textarea>
          </div>
        </div>

        <!-- raw -->
        <div v-else>
          <label class="form-label">原始 handle 数组（JSON）<span class="text-red-500">*</span></label>
          <textarea v-model="formData.rawText" rows="10" class="input font-mono text-xs" placeholder='[{"handler":"reverse_proxy","upstreams":[{"dial":"x:80"}]}]'></textarea>
          <p class="text-xs text-slate-400 mt-1">直接编辑 caddy json 中 routes[i].handle 的原始数组</p>
        </div>
      </div>
    </div>

    <template #confirm-text>
      确认{{ isEditMode ? '更新' : '新建' }}
    </template>
  </BaseModal>
</template>
