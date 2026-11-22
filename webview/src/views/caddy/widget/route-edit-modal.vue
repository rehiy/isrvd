<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { CaddyRoute, CaddyRouteSnapshot, CaddyRouteUpsert, DockerContainerInfo } from '@/service/types'

import { formatHostPort, parseHostPort } from '@/helper/format'

import CodeEditor from '@/component/code-editor.vue'
import ContainerPortSelect from '@/component/docker/container-port-select.vue'
import ContainerSelect from '@/component/docker/container-select.vue'
import BaseModal from '@/component/modal.vue'
import ToggleCard from '@/component/toggle-card.vue'

import { buildRouteMatch, buildStructuredRoute, caddyNativeRoute, caddyRouteInsertionIndex, caddyRouteJSONError, defaultFormData, readStructuredRoute } from './route-form'
import type { CaddyHandlerKind, MatchHeader } from './route-form'

// handler kind 卡片
interface CaddyHandlerKindCard {
    value: CaddyHandlerKind
    title: string
    desc: string
    icon: string
    tone: 'indigo' | 'emerald' | 'amber' | 'violet' | 'rose' | 'slate'
}

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

interface RouteServerOption {
    id: string
    name: string
}

@Component({
    expose: ['show'],
    components: { BaseModal, CodeEditor, ContainerPortSelect, ContainerSelect, ToggleCard },
    emits: ['success']
})
class RouteEditModal extends Vue {
    portal = usePortal()

    @Prop({ type: Array, default: () => [{ id: 'srv0', name: 'srv0' }] }) readonly servers!: RouteServerOption[]

    isOpen = false
    modalLoading = false
    isEditMode = false
    editingIndex = -1
    editingRevision = ''
    targetServer = 'srv0'
    containers: DockerContainerInfo[] = []
    routeSnapshot: CaddyRouteSnapshot | null = null
    positionsLoading = false
    positionsRequestID = 0
    insertPosition: 'recommended' | 'append' | number = 'recommended'

    get recommendedPosition() {
        if (!this.routeSnapshot || this.formData.kind === 'raw') {
            return undefined
        }
        const route = this.buildStructuredRoute()
        return route ? caddyRouteInsertionIndex(route, this.routeSnapshot.routes) : undefined
    }

    get selectedPosition() {
        if (typeof this.insertPosition === 'number') {
            return this.insertPosition
        }
        if (this.insertPosition === 'recommended' && this.recommendedPosition !== undefined) {
            return this.recommendedPosition
        }
        return this.routeSnapshot?.routes.length ?? 0
    }

    async loadRoutePositions() {
        const requestID = ++this.positionsRequestID
        this.routeSnapshot = null
        this.positionsLoading = true
        this.insertPosition = 'recommended'
        try {
            const snapshot = (await api.caddyRouteSnapshot(this.targetServer)).payload
            if (requestID === this.positionsRequestID) {
                this.routeSnapshot = snapshot || null
            }
        } finally {
            if (requestID === this.positionsRequestID) {
                this.positionsLoading = false
            }
        }
    }

    positionLabel(route: CaddyRoute) {
        const hosts = route.match?.flatMap(match => match.host || []) || []
        return this.$t('在第 {0} 条路由前插入：{1}', route.index + 1, hosts.join(', ') || '*')
    }

    get canLoadDockerContainers() {
        return this.portal.hasPerm('GET /api/docker/containers')
    }

    get defaultServerName() {
        return this.servers.find(server => server.name === 'srv0')?.name || this.servers[0]?.name || 'srv0'
    }

    // 匹配请求头多行编辑列表
    matchHeaderList: MatchHeader[] = []
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
        if (this.formData.kind === 'raw' && this.rawDirty && item.value !== 'raw') {
            return 'option-card option-card-disabled'
        }
        return `option-card ${active ? TONE_CARD_ACTIVE[item.tone] : 'option-card-inactive'}`
    }

    modeCardIconClass(item: CaddyHandlerKindCard) {
        const active = this.formData.kind === item.value
        return `option-card-icon ${active ? TONE_ICON_ACTIVE[item.tone] : 'bg-slate-100'}`
    }

    setKind(kind: CaddyHandlerKind) {
        if (this.modalLoading || (this.formData.kind === 'raw' && this.rawDirty && kind !== 'raw')) {
            return
        }
        if (kind === 'raw' && this.formData.kind !== 'raw') {
            const route = this.buildStructuredRoute() || { match: this.formMatch, handle: [] }
            this.rawJson = JSON.stringify(route, null, 2)
        }
        this.formData.kind = kind
    }

    get formMatch() {
        return buildRouteMatch(this.formData, this.matchHeaderList)
    }

    buildStructuredRoute(): CaddyRouteUpsert | null {
        return buildStructuredRoute(this.formData, this.formMatch)
    }

    addMatchHeaderRow() {
        this.matchHeaderList.push({ key: '', value: '' })
    }

    removeMatchHeaderRow(idx: number) {
        this.matchHeaderList.splice(idx, 1)
    }

    onToggleMatchHeaders(enabled: boolean) {
        if (!enabled) {
            this.matchHeaderList = []
        } else if (!this.matchHeaderList.length) {
            this.addMatchHeaderRow()
        }
    }
    // ─── 请求头/响应头中间件操作 ───
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

    syncSelectedUpstream() {
        const host = this.formData.upstreamHost.trim()
        const port = this.formData.upstreamPort.trim()
        if (!host) {
            return
        }
        const lines = this.formData.upstreams.split('\n').map(t => t.trim()).filter(Boolean)
        const upstream = formatHostPort(host, port)
        this.formData.upstreams = [upstream, ...lines.slice(1)].join('\n')
    }

    setUpstreamHost(host: string) {
        this.formData.upstreamHost = host
        const port = (this.getPortsByHost(host)[0] || '').split('/')[0].split(':').pop() || ''
        if (port) {
            this.formData.upstreamPort = port
        }
        this.syncSelectedUpstream()
    }

    setUpstreamPort(port: string) {
        this.formData.upstreamPort = port
        this.syncSelectedUpstream()
    }

    syncSelectedFromText() {
        const first = this.formData.upstreams.split('\n').map(t => t.trim()).filter(Boolean)[0] || ''
        const { host, port } = parseHostPort(first)
        this.formData.upstreamHost = host
        this.formData.upstreamPort = port
    }

    async loadContainers() {
        this.containers = []
        if (!this.canLoadDockerContainers) {
            return
        }
        const res = await api.dockerContainerList()
        this.containers = (res.payload || []).filter(c => c.state === 'running')
    }

    show(route: CaddyRoute | null, server = '', revision = '') {
        this.targetServer = server || this.defaultServerName
        this.editingRevision = revision
        this.positionsRequestID++
        this.routeSnapshot = null
        this.positionsLoading = false
        this.insertPosition = 'recommended'
        Object.assign(this.formData, defaultFormData())
        this.matchHeaderList = []
        this.enableMatchHeaders = false
        this.rawJson = ''
        this.rawDirty = false
        void this.loadContainers()
        if (route) {
            this.isEditMode = true
            this.editingIndex = route.index
            const nativeRoute = caddyNativeRoute(route)
            this.rawJson = JSON.stringify(nativeRoute, null, 2)
            const structured = readStructuredRoute(nativeRoute)
            if (structured) {
                Object.assign(this.formData, structured.form)
                this.matchHeaderList = structured.matchHeaders
                this.enableMatchHeaders = this.matchHeaderList.length > 0
                if (this.formData.kind === 'reverse_proxy') {
                    this.syncSelectedFromText()
                }
            } else {
                this.formData.kind = 'raw'
                this.rawDirty = true
            }
        } else {
            this.isEditMode = false
            this.editingIndex = -1
            void this.loadRoutePositions()
        }
        this.isOpen = true
    }

    buildPayload(): CaddyRouteUpsert | null {
        // raw 模式：直接解析用户编辑的 JSON
        if (this.formData.kind === 'raw') {
            if (!this.rawJson.trim()) {
                this.portal.showNotification('error', this.$t('请输入路由 JSON'))
                return null
            }
            let parsed: unknown
            try {
                parsed = JSON.parse(this.rawJson)
            } catch {
                this.portal.showNotification('error', this.$t('JSON 格式错误'))
                return null
            }
            const error = caddyRouteJSONError(parsed)
            if (error) {
                this.portal.showNotification('error', this.$t(error))
                return null
            }
            return parsed as CaddyRouteUpsert
        }

        const route = this.buildStructuredRoute()
        if (!route) {
            this.portal.showNotification('error', this.$t('请完善处理器配置'))
        }
        return route
    }

    async handleConfirm() {
        const revision = this.routeSnapshot?.revision
        if (this.modalLoading || (!this.isEditMode && (this.positionsLoading || !revision))) {
            return
        }
        const payload = this.buildPayload()
        if (!payload) {
            return
        }

        this.modalLoading = true
        try {
            if (this.isEditMode) {
                await api.caddyRouteUpdate(this.editingIndex, payload, this.targetServer, this.editingRevision)
                this.portal.showNotification('success', this.$t('路由更新成功'))
            } else if (revision) {
                await api.caddyRouteCreate(payload, this.targetServer, {
                    index: this.selectedPosition, revision
                })
                this.portal.showNotification('success', this.$t('路由创建成功'))
            }
            this.isOpen = false
            this.$emit('success')
        } finally {
            this.modalLoading = false
        }
    }
}

export default toNative(RouteEditModal)
</script>

<template>
  <BaseModal v-model="isOpen" :title="isEditMode ? $t('编辑路由') : $t('新建路由')" :loading="modalLoading" :confirm-disabled="!isEditMode && (positionsLoading || !routeSnapshot?.revision)" confirm-class="btn-indigo" @confirm="handleConfirm">
    <div class="space-y-6 p-1">
      <div class="space-y-4">
        <div>
          <label class="form-label">{{ $t('服务') }} <span class="text-red-500">*</span></label>
          <select v-model="targetServer" class="input font-mono" :disabled="isEditMode || modalLoading" @change="loadRoutePositions">
            <option v-for="item in servers" :key="item.id" :value="item.name">{{ item.name }}</option>
          </select>
          <p class="text-xs text-slate-400 mt-1">{{ isEditMode ? $t('路由所属服务不可修改') : $t('选择新路由所属的服务') }}</p>
        </div>

        <div v-if="!isEditMode">
          <label class="form-label">{{ $t('插入位置') }}</label>
          <div class="flex items-center gap-2">
            <select v-model="insertPosition" class="input" :disabled="positionsLoading || !routeSnapshot || modalLoading">
              <option value="recommended">{{ recommendedPosition === undefined ? $t('末尾（默认）') : $t('推荐：在第 {0} 条路由前插入', recommendedPosition + 1) }}</option>
              <option v-if="recommendedPosition !== undefined" value="append">{{ $t('末尾') }}</option>
              <option v-for="route in routeSnapshot?.routes || []" :key="route.index" :value="route.index">{{ positionLabel(route) }}</option>
            </select>
            <button type="button" class="btn btn-secondary" :disabled="positionsLoading || modalLoading" :title="$t('刷新路由位置')" @click="loadRoutePositions"><i class="fas fa-rotate"></i></button>
          </div>
          <p v-if="positionsLoading" class="text-xs text-slate-400 mt-1">{{ $t('加载中...') }}</p>
          <p v-else-if="!routeSnapshot" class="text-xs text-amber-600 mt-1">{{ $t('请刷新路由位置后重试') }}</p>
          <p v-else-if="recommendedPosition !== undefined" class="text-xs mt-1" :class="selectedPosition > recommendedPosition ? 'text-amber-600' : 'text-slate-500'">{{ $t('第 {0} 条路由包含覆盖此域名的通配符或兜底规则，建议插入到它之前。', recommendedPosition + 1) }}</p>
          <p v-else class="text-xs text-slate-400 mt-1">{{ $t('路由按顺序执行；涉及鉴权、重写或复杂匹配时，请手动选择合适的位置。') }}</p>
        </div>
      </div>

      <!-- ── 匹配条件 ── -->
      <div v-if="formData.kind !== 'raw'" class="space-y-4">
        <p class="section-title">{{ $t('匹配条件') }}</p>

        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="form-label">Host</label>
            <textarea v-model="formData.hosts" rows="2" class="input font-mono text-sm"></textarea>
            <p class="text-xs text-slate-400 mt-1">{{ $t('每行一个，留空匹配所有，例如：example.com 或 *.example.com') }}</p>
          </div>
          <div>
            <label class="form-label">Path</label>
            <textarea v-model="formData.paths" rows="2" class="input font-mono text-sm"></textarea>
            <p class="text-xs text-slate-400 mt-1">{{ $t('每行一个，支持 * 通配符，例如：/api/* 或 /static/*') }}</p>
          </div>
        </div>

        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="form-label">Method</label>
            <input v-model="formData.methods" type="text" class="input font-mono text-sm" />
            <p class="text-xs text-slate-400 mt-1">{{ $t('空格分隔，留空匹配所有，例如：GET POST') }}</p>
          </div>
          <div>
            <label class="form-label">{{ $t('协议') }}</label>
            <div class="flex gap-2 mt-0.5">
              <button
                v-for="proto in ['', 'http', 'https']"
                :key="proto"
                type="button"
                :class="['btn-proto', formData.protocol === proto ? 'btn-proto-active' : 'btn-proto-inactive']"
                @click="formData.protocol = proto"
              >
                {{ proto === '' ? $t('不限') : proto.toUpperCase() }}
              </button>
            </div>
            <p class="text-xs text-slate-400 mt-1">{{ $t('留空匹配所有') }}</p>
          </div>
        </div>

        <ToggleCard
          v-model="enableMatchHeaders"
          :label="$t('匹配请求头')"
          :desc="$t('仅转发包含指定请求头的请求，留空匹配所有')"
          @update:model-value="onToggleMatchHeaders"
        >
          <div class="space-y-2">
            <div v-for="(entry, idx) in matchHeaderList" :key="idx" class="flex gap-2 items-center">
              <input v-model="entry.key" type="text" class="input font-mono text-sm flex-1" :placeholder="$t('请求头名称')" />
              <input v-model="entry.value" type="text" class="input font-mono text-sm flex-1" :placeholder="$t('值')" />
              <button type="button" class="btn-icon btn-icon-red shrink-0" :title="$t('删除')" :aria-label="$t('删除')" @click="removeMatchHeaderRow(idx)"><i class="fas fa-trash text-sm"></i></button>
            </div>
            <button type="button" class="btn-add-row" @click="addMatchHeaderRow">
              <i class="fas fa-plus text-xs"></i>{{ $t('添加匹配请求头') }}
            </button>
          </div>
          <p class="text-xs text-slate-400 mt-1">{{ $t('请求头值留空表示存在即匹配') }}</p>
        </ToggleCard>
      </div>

      <!-- ── 处理器 ── -->
      <div class="border-t border-slate-200 pt-6 space-y-4">
        <p class="section-title">{{ $t('处理器') }}</p>

        <div class="grid grid-cols-2 md:grid-cols-3 gap-3">
          <button v-for="item in handlerKindCards" :key="item.value" type="button" :class="[modeCardClass(item), 'flex flex-col']" :disabled="modalLoading || (formData.kind === 'raw' && rawDirty && item.value !== 'raw')" @click="setKind(item.value)">
            <div class="flex items-center gap-2 mb-1.5">
              <div :class="modeCardIconClass(item)"><i class="fas text-sm" :class="item.icon"></i></div>
              <span class="text-sm font-semibold leading-tight">{{ $t(item.title) }}</span>
            </div>
            <div class="text-xs opacity-75 leading-4">{{ $t(item.desc) }}</div>
          </button>
        </div>

        <!-- reverse_proxy -->
        <div v-if="formData.kind === 'reverse_proxy'" class="space-y-4">
          <div v-if="canLoadDockerContainers">
            <label class="form-label">{{ $t('选择容器与端口') }}</label>
            <div class="grid grid-cols-[2fr_1fr] gap-2">
              <ContainerSelect :model-value="formData.upstreamHost" :containers="containers" :placeholder="$t('请输入容器名')" @update:model-value="setUpstreamHost" />
              <ContainerPortSelect :model-value="formData.upstreamPort" :ports="getPortsByHost(formData.upstreamHost)" :placeholder="$t('请输入端口')" @update:model-value="setUpstreamPort" />
            </div>
            <p class="text-xs text-slate-400 mt-1">{{ $t('可输入容器名或 IP 地址，选择后自动填充下方第一个上游，也可手动输入') }}</p>
          </div>
          <div>
            <label class="form-label">{{ $t('上游地址') }} <span class="text-red-500">*</span></label>
            <textarea v-model="formData.upstreams" rows="3" class="input font-mono text-sm" :placeholder="$t('请输入上游地址')" @input="syncSelectedFromText"></textarea>
            <p class="text-xs text-slate-400 mt-1">{{ $t('每行一个 host:port，多个上游自动轮询，例如：backend1:8080') }}</p>
          </div>
          <div v-if="!formData.fastcgi">
            <div class="grid grid-cols-3 gap-3">
              <div>
                <label class="form-label">{{ $t('连接超时') }}</label>
                <input v-model="formData.dialTimeout" type="text" class="input font-mono text-sm" />
              </div>
              <div>
                <label class="form-label">{{ $t('响应头超时') }}</label>
                <input v-model="formData.readTimeout" type="text" class="input font-mono text-sm" />
              </div>
              <div>
                <label class="form-label">{{ $t('写入超时') }}</label>
                <input v-model="formData.writeTimeout" type="text" class="input font-mono text-sm" />
              </div>
            </div>
            <p class="text-xs text-slate-400 mt-1">{{ $t('如 10s、5m') }}{{ $t('，留空使用默认值') }}</p>
          </div>
          <ToggleCard v-model="formData.fastcgi" :label="$t('FastCGI 协议')" :desc="$t('适用于 PHP-FPM 等 FastCGI 进程')">
            <label class="form-label">{{ $t('FastCGI 文档根目录') }}</label>
            <input v-model="formData.fastcgiRoot" type="text" class="input font-mono text-sm" />
            <p class="text-xs text-slate-400 mt-1">{{ $t('设置') }} <code class="px-1 bg-slate-100 rounded">DOCUMENT_ROOT</code>{{ $t('，留空不传 root，例如：/var/www/html') }}</p>
          </ToggleCard>
          <ToggleCard v-model="formData.proxyRewriteEnabled" :label="$t('代理前 URI 重写')">
            <template #desc>{{ $t('写入') }} <code class="px-1 bg-slate-100 rounded">reverse_proxy.rewrite</code>{{ $t('，仅影响转发给上游的请求 URI') }}</template>
            <div class="space-y-4">
              <!-- 覆盖请求方法 -->
              <div>
                <label class="form-label">{{ $t('覆盖请求方法') }}</label>
                <input v-model="formData.proxyRewriteMethod" type="text" class="input font-mono text-sm" />
                <p class="text-xs text-slate-400 mt-1">{{ $t('如 GET、POST，留空不修改') }}</p>
              </div>
              <!-- URI 整体替换 -->
              <div>
                <label class="form-label">{{ $t('URI 整体替换') }}</label>
                <input v-model="formData.proxyRewriteUri" type="text" class="input font-mono text-sm" />
                <p class="text-xs text-slate-400 mt-1">{{ $t('可选，完整替换转发给上游的 URI，留空不修改') }}{{ $t('例如：') }}<code class="px-1 bg-slate-100 rounded">/backend/{http.request.uri.path.1}</code></p>
              </div>
              <!-- 路径前后缀裁剪 -->
              <div>
                <p class="form-label">{{ $t('路径前后缀裁剪') }}</p>
                <div class="grid grid-cols-2 gap-3 mt-1">
                  <div>
                    <label class="form-label">{{ $t('去掉前缀') }}</label>
                    <input v-model="formData.proxyStripPathPrefix" type="text" class="input font-mono text-sm" />
                  </div>
                  <div>
                    <label class="form-label">{{ $t('去掉后缀') }}</label>
                    <input v-model="formData.proxyStripPathSuffix" type="text" class="input font-mono text-sm" />
                  </div>
                </div>
                <p class="text-xs text-slate-400 mt-1">{{ $t('可选') }}{{ $t('例如：') }}<code class="px-1 bg-slate-100 rounded">/api</code> / <code class="px-1 bg-slate-100 rounded">.php</code></p>
              </div>
              <!-- 子串查找替换 -->
              <div>
                <p class="form-label">{{ $t('子串查找替换') }}</p>
                <div class="grid grid-cols-2 gap-3 mt-1">
                  <div>
                    <label class="form-label">{{ $t('查找') }}</label>
                    <input v-model="formData.proxyUriSubstringFind" type="text" class="input font-mono text-sm" />
                  </div>
                  <div>
                    <label class="form-label">{{ $t('替换为') }}</label>
                    <input v-model="formData.proxyUriSubstringReplace" type="text" class="input font-mono text-sm" />
                  </div>
                </div>
                <p class="text-xs text-slate-400 mt-1">{{ $t('可选') }}{{ $t('例如：') }}<code class="px-1 bg-slate-100 rounded">/old</code> → <code class="px-1 bg-slate-100 rounded">/new</code></p>
              </div>
            </div>
          </ToggleCard>
        </div>

        <!-- file_server -->
        <div v-else-if="formData.kind === 'file_server'" class="space-y-4">
          <div>
            <label class="form-label">{{ $t('根目录') }} <span class="text-red-500">*</span></label>
            <input v-model="formData.root" type="text" class="input font-mono text-sm" :placeholder="$t('请输入根目录')" />
            <p class="text-xs text-slate-400 mt-1">{{ $t('静态文件根目录，例如：/var/www/html') }}</p>
          </div>
          <ToggleCard v-model="formData.browse" :label="$t('启用目录浏览')" :desc="$t('允许访客浏览目录内文件列表')" />
        </div>

        <!-- static_response -->
        <div v-else-if="formData.kind === 'static_response'" class="space-y-4">
          <div>
            <label class="form-label">{{ $t('状态码') }}</label>
            <input v-model.number="formData.statusCode" type="number" min="100" max="599" class="input" :placeholder="$t('请输入状态码')" />
            <p class="text-xs text-slate-400 mt-1">{{ $t('HTTP 状态码，例如：200') }}</p>
          </div>
          <div>
            <label class="form-label">{{ $t('响应体') }}</label>
            <CodeEditor v-model="formData.body" :label="$t('响应体')" :disabled="modalLoading" :placeholder="$t('请输入响应体')" />
            <p class="text-xs text-slate-400 mt-1">{{ $t('留空则返回默认响应，例如：OK') }}</p>
          </div>
        </div>

        <!-- rewrite -->
        <div v-else-if="formData.kind === 'rewrite'" class="space-y-4">
          <div>
            <label class="form-label">{{ $t('URI 替换') }}</label>
            <input v-model="formData.rewriteUri" type="text" class="input font-mono text-sm" :placeholder="$t('请输入 URI 替换规则')" />
            <p class="text-xs text-slate-400 mt-1">{{ $t('完整替换请求 URI，支持 Caddy 占位符，例如：/new-path/{http.request.uri.path.1}') }}</p>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="form-label">{{ $t('去掉路径前缀') }}</label>
              <input v-model="formData.stripPathPrefix" type="text" class="input font-mono text-sm" :placeholder="$t('请输入要去掉的前缀')" />
              <p class="text-xs text-slate-400 mt-1">{{ $t('例如：') }}<code class="px-1 bg-slate-100 rounded">/api</code>{{ $t('，效果：') }}<code class="px-1 bg-slate-100 rounded">/api/v1/foo</code> → <code class="px-1 bg-slate-100 rounded">/v1/foo</code></p>
            </div>
            <div>
              <label class="form-label">{{ $t('去掉路径后缀') }}</label>
              <input v-model="formData.stripPathSuffix" type="text" class="input font-mono text-sm" :placeholder="$t('请输入要去掉的后缀')" />
              <p class="text-xs text-slate-400 mt-1">{{ $t('从路径末尾去掉指定后缀，例如：.php') }}</p>
            </div>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="form-label">{{ $t('子串查找') }}</label>
              <input v-model="formData.uriSubstringFind" type="text" class="input font-mono text-sm" :placeholder="$t('请输入要查找的子串')" />
              <p class="text-xs text-slate-400 mt-1">{{ $t('例如：/old') }}</p>
            </div>
            <div>
              <label class="form-label">{{ $t('子串替换') }}</label>
              <input v-model="formData.uriSubstringReplace" type="text" class="input font-mono text-sm" :placeholder="$t('请输入替换内容')" />
              <p class="text-xs text-slate-400 mt-1">{{ $t('例如：/new') }}</p>
            </div>
          </div>
        </div>

        <!-- raw -->
        <div v-else-if="formData.kind === 'raw'" class="space-y-2">
          <label class="form-label">{{ $t('路由 JSON') }} <span class="text-red-500">*</span></label>
          <CodeEditor v-model="rawJson" language="json" :label="$t('路由 JSON')" :disabled="modalLoading" placeholder='{ "match": [...], "handle": [...] }' @update:model-value="rawDirty = true" />
          <p class="text-xs text-slate-400 mt-1">{{ $t('直接编辑完整的 Caddy 路由 JSON，match 和 handle 字段可选') }}</p>
          <p v-if="rawDirty" class="text-xs text-slate-400 mt-1">{{ $t('为避免丢失配置，已加载或修改的 JSON 仅支持在此模式编辑。') }}</p>
        </div>
      </div>

      <!-- ── 请求头 / 响应头操作 ── -->
      <ToggleCard v-if="formData.kind !== 'raw'" v-model="formData.enableHeaders" :label="$t('配置请求头/响应头操作')">
        <template #desc>{{ $t('串联中间件，在处理器前执行，支持占位符如') }} <code class="px-1 bg-slate-100 rounded">{http.request.remote.host}</code></template>
        <div class="space-y-4">
          <!-- 请求头 -->
          <div>
            <label class="form-label">{{ $t('请求头') }}</label>
            <div v-if="formData.requestHeaders.length" class="space-y-2 mb-2">
              <div v-for="(op, idx) in formData.requestHeaders" :key="idx" class="flex gap-2 items-center">
                <select v-model="op.op" class="input text-sm w-24 shrink-0">
                  <option value="set">{{ $t('覆盖') }}</option>
                  <option value="add">{{ $t('追加') }}</option>
                  <option value="delete">{{ $t('删除') }}</option>
                </select>
                <input v-model="op.field" type="text" class="input font-mono text-sm flex-1" :placeholder="$t('请输入请求头名称')" />
                <input v-if="op.op !== 'delete'" v-model="op.value" type="text" class="input font-mono text-sm flex-1" :placeholder="$t('请输入请求头值')" />
                <div v-else class="flex-1"></div>
                <button type="button" class="btn-icon btn-icon-red shrink-0" :title="$t('删除')" :aria-label="$t('删除')" @click="removeHeaderOp('request', idx)"><i class="fas fa-trash text-sm"></i></button>
              </div>
            </div>
            <button type="button" class="btn-add-row" @click="addHeaderOp('request')">
              <i class="fas fa-plus text-xs"></i>{{ $t('添加请求头操作') }}
            </button>
            <p class="text-xs text-slate-400 mt-1">{{ $t('例如：字段 X-Real-IP，值 {http.request.remote.host}') }}</p>
          </div>
          <!-- 响应头 -->
          <div>
            <label class="form-label">{{ $t('响应头') }}</label>
            <div v-if="formData.responseHeaders.length" class="space-y-2 mb-2">
              <div v-for="(op, idx) in formData.responseHeaders" :key="idx" class="flex gap-2 items-center">
                <select v-model="op.op" class="input text-sm w-24 shrink-0">
                  <option value="set">{{ $t('覆盖') }}</option>
                  <option value="add">{{ $t('追加') }}</option>
                  <option value="delete">{{ $t('删除') }}</option>
                </select>
                <input v-model="op.field" type="text" class="input font-mono text-sm flex-1" :placeholder="$t('请输入响应头名称')" />
                <input v-if="op.op !== 'delete'" v-model="op.value" type="text" class="input font-mono text-sm flex-1" :placeholder="$t('请输入响应头值')" />
                <div v-else class="flex-1"></div>
                <button type="button" class="btn-icon btn-icon-red shrink-0" :title="$t('删除')" :aria-label="$t('删除')" @click="removeHeaderOp('response', idx)"><i class="fas fa-trash text-sm"></i></button>
              </div>
            </div>
            <button type="button" class="btn-add-row" @click="addHeaderOp('response')">
              <i class="fas fa-plus text-xs"></i>{{ $t('添加响应头操作') }}
            </button>
            <p class="text-xs text-slate-400 mt-1">{{ $t('例如：字段 X-Frame-Options，值 SAMEORIGIN') }}</p>
          </div>
        </div>
      </ToggleCard>
    </div>
  
    <template #confirm-text>
      {{ isEditMode ? $t('保存') : $t('新建') }}
    </template>
  </BaseModal>
</template>
