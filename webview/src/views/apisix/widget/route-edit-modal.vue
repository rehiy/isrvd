<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import api from '@/service/api'
import type {
    ApisixPluginConfig,
    ApisixRoute,
    ApisixRouteUpstreamFormNode,
    ApisixRouteUpstreamModeCard,
    ApisixRouteUpstreamMode,
    ApisixUpstream,
    ApisixUpstreamConfig,
    ApisixUpstreamHashOn,
    ApisixUpstreamType,
    DockerContainerInfo
} from '@/service/types'

import {
    buildRoutePayload,
    detectRouteUpstreamMode,
    normalizeUpstreamFormNodes,
    normalizeUpstreamType
} from '@/helper/utils'

import BaseModal from '@/component/modal.vue'

import HostSelect from './host-select.vue'
import PortSelect from './port-select.vue'

@Component({
    expose: ['show'],
    components: { BaseModal, HostSelect, PortSelect },
    emits: ['success']
})
class RouteEditModal extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    isEditMode = false
    editingRouteId = ''
    showPluginPanel = false
    showImportPanel = false
    importRouteId = ''
    importRoutePlugins: Record<string, unknown> = {}
    importRoutePluginsLoading = false
    selectedImportPlugins: Set<string> = new Set()
    pluginSearchKeyword = ''
    originalUpstream: ApisixUpstreamConfig | null = null

    pluginConfigs: ApisixPluginConfig[] = []
    upstreams: ApisixUpstream[] = []
    containers: DockerContainerInfo[] = []
    availablePlugins: Record<string, { schema: Record<string, unknown> }> = {}
    routes: ApisixRoute[] = []

    formData = {
        name: '',
        desc: '',
        uris: '',
        hosts: '',
        status: 1,
        priority: 0,
        enable_websocket: false,
        plugin_config_id: '',
        plugins: {} as Record<string, unknown>,
        pluginsJson: '{}',
        pluginsJsonError: '',
        upstream_mode: 'nodes' as ApisixRouteUpstreamMode,
        upstream_type: 'roundrobin' as ApisixUpstreamType,
        upstream_id: '',
        upstream_nodes: [{ host: '', port: '', weight: 1 }] as ApisixRouteUpstreamFormNode[],
        upstream_hash_on: 'vars' as ApisixUpstreamHashOn,
        upstream_key: 'remote_addr',
        timeout_connect: '' as string | number,
        timeout_send: '' as string | number,
        timeout_read: '' as string | number
    }

    // ─── 计算属性 ───
    get currentPluginNames() {
        return Object.keys(this.formData.plugins || {})
    }

    get filteredAvailablePlugins() {
        const all = Object.keys(this.availablePlugins)
        if (!this.pluginSearchKeyword) return all
        return all.filter(n => n.toLowerCase().includes(this.pluginSearchKeyword.toLowerCase()))
    }

    get selectedInlineNodeCount() {
        return this.formData.upstream_nodes.filter(node => node.host.trim() && String(node.port).trim()).length
    }

    get upstreamModeCards(): ApisixRouteUpstreamModeCard[] {
        return [
            {
                value: 'nodes',
                title: '内联多上游节点',
                desc: '一个路由直连多个节点，并为每个节点设置权重',
                icon: 'fa-circle-nodes',
                tone: 'indigo'
            },
            {
                value: 'upstream_id',
                title: '引用已有上游',
                desc: '复用 APISIX 中已经创建好的上游对象',
                icon: 'fa-diagram-project',
                tone: 'emerald'
            },
            {
                value: 'none',
                title: '空上游',
                desc: '暂不配置转发目标，先保存路由规则',
                icon: 'fa-ban',
                tone: 'slate'
            },
        ]
    }

    get upstreamTypeOptions(): Array<{ value: ApisixUpstreamType; label: string; desc: string }> {
        return [
            { value: 'roundrobin', label: 'roundrobin', desc: '按权重轮询分配请求' },
            { value: 'least_conn', label: 'least_conn', desc: '优先选择当前连接数更少的节点' },
            { value: 'ewma', label: 'ewma', desc: '根据历史延迟动态选择更快的节点' },
            { value: 'chash', label: 'chash', desc: '一致性哈希，适合会话粘性场景' }
        ]
    }

    get hashOnOptions(): Array<{ value: ApisixUpstreamHashOn; label: string; keyPlaceholder: string; keyHint: string }> {
        return [
            { value: 'vars',             label: 'vars（Nginx 变量）',    keyPlaceholder: 'remote_addr',  keyHint: 'Nginx 变量名，不带 $ 前缀，如 remote_addr、uri' },
            { value: 'header',           label: 'header（请求头）',      keyPlaceholder: 'X-User-Id',    keyHint: '请求头名称，如 X-User-Id' },
            { value: 'cookie',           label: 'cookie',               keyPlaceholder: 'session_id',   keyHint: 'Cookie 名称（大小写敏感），如 session_id' },
            { value: 'consumer',         label: 'consumer（消费者）',    keyPlaceholder: 'consumer_name', keyHint: '通常填 consumer_name，由 APISIX 自动注入' },
            { value: 'vars_combinations', label: 'vars_combinations',   keyPlaceholder: '$remote_addr$uri', keyHint: '多个 Nginx 变量组合，如 $remote_addr$uri' },
        ]
    }

    get selectedHashOnOption() {
        return this.hashOnOptions.find(o => o.value === this.formData.upstream_hash_on) || this.hashOnOptions[0]
    }

    get selectedUpstreamTypeOption() {
        return this.upstreamTypeOptions.find(option => option.value === this.formData.upstream_type) || this.upstreamTypeOptions[0]
    }

    get routeValidationMessage() {
        if (this.formData.upstream_mode === 'upstream_id') {
            return this.formData.upstream_id.trim() ? '' : '请选择要引用的上游对象'
        }

        if (this.formData.upstream_mode !== 'nodes') return ''

        const rows = this.formData.upstream_nodes
        const validRows = rows.filter(node => node.host.trim() && String(node.port).trim())
        if (validRows.length === 0) return '请至少配置一个完整的上游节点'

        for (const [index, node] of rows.entries()) {
            const hasHost = !!node.host.trim()
            const hasPort = !!String(node.port).trim()
            if (hasHost !== hasPort) return `第 ${index + 1} 个上游节点的主机和端口需要同时填写`
            if (hasPort && !/^\d+$/.test(String(node.port).trim())) return `第 ${index + 1} 个上游节点端口必须为数字`
            if ((hasHost || hasPort) && Number(node.weight) < 0) return `第 ${index + 1} 个上游节点权重不能为负数`
        }

        if (this.formData.upstream_type === 'chash' && !this.formData.upstream_key?.trim()) {
            return '使用 chash 策略时，哈希键（key）不能为空'
        }

        return ''
    }

    // ─── 方法 ───
    resetForm() {
        Object.assign(this.formData, {
            name: '',
            desc: '',
            uris: '',
            hosts: '',
            status: 1,
            priority: 0,
            enable_websocket: false,
            plugin_config_id: '',
            plugins: {},
            pluginsJson: '{}',
            pluginsJsonError: '',
            upstream_mode: 'nodes',
            upstream_type: 'roundrobin',
            upstream_id: '',
            upstream_nodes: [{ host: '', port: '', weight: 1 }],
            upstream_hash_on: 'vars',
            upstream_key: 'remote_addr',
            timeout_connect: '',
            timeout_send: '',
            timeout_read: ''
        })
        this.editingRouteId = ''
        this.originalUpstream = null
        this.showPluginPanel = false
        this.showImportPanel = false
        this.importRouteId = ''
        this.importRoutePlugins = {}
        this.selectedImportPlugins = new Set()
        this.pluginSearchKeyword = ''
    }

    createUpstreamNode(node?: Partial<ApisixRouteUpstreamFormNode>): ApisixRouteUpstreamFormNode {
        return {
            host: node?.host || '',
            port: String(node?.port || ''),
            weight: Number(node?.weight) >= 0 ? Number(node?.weight) : 1
        }
    }

    addUpstreamNode(node?: Partial<ApisixRouteUpstreamFormNode>) {
        this.formData.upstream_nodes = [...this.formData.upstream_nodes, this.createUpstreamNode(node)]
    }

    removeUpstreamNode(index: number) {
        const next = this.formData.upstream_nodes.filter((_, idx) => idx !== index)
        this.formData.upstream_nodes = next.length > 0 ? next : [this.createUpstreamNode()]
    }

    setUpstreamMode(mode: ApisixRouteUpstreamMode) {
        this.formData.upstream_mode = mode
        if (mode === 'nodes' && this.formData.upstream_nodes.length === 0) {
            this.formData.upstream_nodes = [this.createUpstreamNode()]
        }
    }

    getContainerByHost(host: string) {
        const normalized = host.trim()
        return normalized ? this.containers.find(c => c.name === normalized) : undefined
    }

    getPortsByHost(host: string): string[] {
        return this.getContainerByHost(host)?.ports || []
    }

    extractDefaultPort(ports: string[]) {
        const first = ports[0] || ''
        return first.split('/')[0].split(':').pop() || ''
    }

    updateUpstreamNodeHost(index: number, host: string) {
        const next = [...this.formData.upstream_nodes]
        const current = next[index]
        if (!current) return
        current.host = host
        const defaultPort = this.extractDefaultPort(this.getPortsByHost(host))
        if (defaultPort) current.port = defaultPort
        this.formData.upstream_nodes = next
    }

    updateUpstreamNodePort(index: number, port: string) {
        const next = [...this.formData.upstream_nodes]
        const current = next[index]
        if (!current) return
        current.port = port
        this.formData.upstream_nodes = next
    }

    async loadResources(allRoutes: ApisixRoute[]) {
        this.routes = allRoutes || []
        try {
            const [pc, us, pl, ct] = await Promise.all([
                api.apisixListPluginConfigs(),
                api.apisixListUpstreams(),
                api.apisixListPlugins(),
                api.listContainers()
            ])
            this.pluginConfigs = pc.payload || []
            this.upstreams = us.payload || []
            this.availablePlugins = pl.payload || {}
            this.containers = (ct.payload || []).filter(c => c.state === 'running')
        } catch {}
    }

    async show(route: ApisixRoute | null, allRoutes: ApisixRoute[]) {
        await this.loadResources(allRoutes)
        if (route && route.id) {
            const routeID = route.id
            this.isEditMode = true
            this.resetForm()
            this.editingRouteId = routeID
            this.modalLoading = true
            this.isOpen = true
            try {
                const r = (await api.apisixGetRoute(routeID)).payload
                if (!r) {
                    this.actions.showNotification('error', '加载路由详情失败')
                    this.isOpen = false
                    this.modalLoading = false
                    return
                }
                const plugins = r.plugins || {}
                this.originalUpstream = r.upstream ? { ...(r.upstream as ApisixUpstreamConfig) } : null
                Object.assign(this.formData, {
                    name: r.name || '',
                    desc: r.desc || '',
                    uris: (r.uris?.length ? r.uris : [r.uri || '']).filter(Boolean).join('\n'),
                    hosts: (r.hosts?.length ? r.hosts : [r.host || '']).filter(Boolean).join('\n'),
                    status: r.status ?? 0,
                    priority: r.priority ?? 0,
                    enable_websocket: r.enable_websocket || false,
                    plugin_config_id: r.plugin_config_id || '',
                    plugins,
                    pluginsJson: JSON.stringify(plugins, null, 2),
                    pluginsJsonError: '',
                    upstream_mode: detectRouteUpstreamMode(r),
                    upstream_type: normalizeUpstreamType(r.upstream?.type),
                    upstream_id: r.upstream_id || '',
                    upstream_nodes: normalizeUpstreamFormNodes(r.upstream),
                    upstream_hash_on: (r.upstream?.hash_on as ApisixUpstreamHashOn) || 'vars',
                    upstream_key: (r.upstream?.key as string) || 'remote_addr',
                    timeout_connect: r.timeout?.connect ?? '',
                    timeout_send: r.timeout?.send ?? '',
                    timeout_read: r.timeout?.read ?? ''
                })
            } catch {
                this.actions.showNotification('error', '加载路由详情失败')
                this.isOpen = false
            }
            this.modalLoading = false
            return
        }

        this.isEditMode = false
        this.resetForm()
        this.isOpen = true
    }

    syncPluginsFromJson() {
        try {
            this.formData.plugins = JSON.parse(this.formData.pluginsJson || '{}')
            this.formData.pluginsJsonError = ''
        } catch (e: unknown) {
            this.formData.pluginsJsonError = 'JSON 格式错误: ' + (e instanceof Error ? e.message : String(e))
        }
    }

    removePlugin(name: string) {
        const p = { ...this.formData.plugins }
        delete p[name]
        this.formData.plugins = p
        this.formData.pluginsJson = JSON.stringify(p, null, 2)
    }

    readonly TYPE_DEFAULTS: Record<string, string | number | boolean | unknown[] | Record<string, unknown>> = {
        string: '',
        integer: 0,
        number: 0,
        boolean: false,
        array: [],
        object: {}
    }

    buildPluginDefault(schema: { properties?: Record<string, { type: string; default?: unknown }>; required?: string[] }) {
        if (!schema?.properties) return {}
        const required = new Set(schema.required || [])
        const result: Record<string, unknown> = {}
        for (const [key, def] of Object.entries(schema.properties)) {
            if (key === 'disable') continue
            if (required.has(key) || def.default !== undefined) {
                result[key] = def.default !== undefined ? def.default : (this.TYPE_DEFAULTS[def.type] ?? null)
            }
        }
        return result
    }

    addPresetPlugin(name: string) {
        if (this.formData.plugins[name] !== undefined) {
            return this.actions.showNotification('warning', `插件 ${name} 已存在`)
        }
        const schema = this.availablePlugins[name]?.schema
        const p = { ...this.formData.plugins, [name]: this.buildPluginDefault(schema) }
        this.formData.plugins = p
        this.formData.pluginsJson = JSON.stringify(p, null, 2)
        this.showPluginPanel = false
        this.pluginSearchKeyword = ''
    }

    async onImportRouteChange() {
        this.importRoutePlugins = {}
        this.selectedImportPlugins = new Set()
        if (!this.importRouteId) return
        this.importRoutePluginsLoading = true
        try {
            const src = (await api.apisixGetRoute(this.importRouteId)).payload?.plugins || {}
            this.importRoutePlugins = src
            this.selectedImportPlugins = new Set(Object.keys(src))
        } catch {
            this.actions.showNotification('error', '加载路由插件失败')
        }
        this.importRoutePluginsLoading = false
    }

    toggleImportPlugin(name: string) {
        const s = new Set(this.selectedImportPlugins)
        s.has(name) ? s.delete(name) : s.add(name)
        this.selectedImportPlugins = s
    }

    importPluginsFromRoute() {
        if (!this.importRouteId) return this.actions.showNotification('warning', '请先选择要导入的路由')
        if (this.selectedImportPlugins.size === 0) return this.actions.showNotification('warning', '请至少勾选一个插件')
        const toImport: Record<string, unknown> = {}
        for (const name of this.selectedImportPlugins) {
            if (this.importRoutePlugins[name] !== undefined) toImport[name] = this.importRoutePlugins[name]
        }
        const merged = { ...this.formData.plugins, ...toImport }
        this.formData.plugins = merged
        this.formData.pluginsJson = JSON.stringify(merged, null, 2)
        this.showImportPanel = false
        this.importRouteId = ''
        this.importRoutePlugins = {}
        this.selectedImportPlugins = new Set()
        this.actions.showNotification('success', `已导入 ${Object.keys(toImport).length} 个插件`)
    }

    validateInlineNodes() {
        if (this.routeValidationMessage) {
            this.actions.showNotification('error', this.routeValidationMessage)
            return false
        }
        return true
    }

    async handleConfirm() {
        if (!this.formData.name.trim()) return this.actions.showNotification('error', '路由名称不能为空')
        if (!this.formData.uris.split('\n').map((s: string) => s.trim()).filter(Boolean).length) return this.actions.showNotification('error', 'URI 不能为空')
        if (this.formData.pluginsJsonError) return this.actions.showNotification('error', '请修正 Plugin JSON 格式错误')
        if (this.formData.upstream_mode === 'upstream_id' && !this.formData.upstream_id.trim()) return this.actions.showNotification('error', '请选择要引用的上游')
        if (this.formData.upstream_mode === 'nodes' && !this.validateInlineNodes()) return

        this.modalLoading = true
        try {
            const payload = buildRoutePayload(this.formData, this.originalUpstream)
            if (this.isEditMode) {
                await api.apisixUpdateRoute(this.editingRouteId, payload)
                this.actions.showNotification('success', '路由更新成功')
            } else {
                await api.apisixCreateRoute(payload)
                this.actions.showNotification('success', '路由创建成功')
            }
            this.isOpen = false
            this.resetForm()
            this.$emit('success')
        } catch (e: unknown) {
            this.actions.showNotification('error', (e instanceof Error ? e.message : '') || '操作失败')
        }
        this.modalLoading = false
    }
}

export default toNative(RouteEditModal)
</script>

<template>
  <BaseModal v-model="isOpen" :title="isEditMode ? '编辑路由' : '创建路由'" :loading="modalLoading">
    <div class="space-y-5 p-1">
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">路由名称</label>
          <input v-model="formData.name" type="text" class="input" placeholder="例如：成员服务入口" />
          <p class="text-xs text-slate-400 mt-1">用于在路由列表中识别当前规则。</p>
        </div>
        <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">优先级</label>
          <input v-model.number="formData.priority" type="number" min="0" class="input" placeholder="0" />
          <p class="text-xs text-slate-400 mt-1">数值越大优先级越高，命中冲突时优先匹配。</p>
        </div>
      </div>

      <div>
        <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">描述</label>
        <textarea v-model="formData.desc" rows="2" class="input" placeholder="补充路由用途、环境或业务说明"></textarea>
      </div>

      <div>
        <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2">超时配置（秒）</label>
        <div class="grid grid-cols-3 gap-3">
          <div><input v-model.number="formData.timeout_connect" type="number" class="input" placeholder="连接超时" min="0" /><p class="text-xs text-slate-400 mt-1">connect</p></div>
          <div><input v-model.number="formData.timeout_send" type="number" class="input" placeholder="发送超时" min="0" /><p class="text-xs text-slate-400 mt-1">send</p></div>
          <div><input v-model.number="formData.timeout_read" type="number" class="input" placeholder="读取超时" min="0" /><p class="text-xs text-slate-400 mt-1">read</p></div>
        </div>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">URI（每行一个）</label>
          <textarea v-model="formData.uris" rows="4" class="input font-mono text-sm" placeholder="/api/v1/*&#10;/api/v2/*"></textarea>
          <p class="text-xs text-slate-400 mt-1">支持单条或多条 URI 规则。</p>
        </div>
        <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">Host（每行一个，留空匹配所有）</label>
          <textarea v-model="formData.hosts" rows="4" class="input font-mono text-sm" placeholder="example.com&#10;api.example.com"></textarea>
          <p class="text-xs text-slate-400 mt-1">可选，多 Host 时一行一个。</p>
        </div>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">WebSocket</label>
          <select v-model="formData.enable_websocket" class="input">
            <option :value="false">关闭</option>
            <option :value="true">开启</option>
          </select>
        </div>
        <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">路由状态</label>
          <select v-model="formData.status" class="input">
            <option :value="1">启用</option>
            <option :value="0">禁用</option>
          </select>
        </div>
      </div>

      <div class="rounded-2xl border border-slate-200 bg-gradient-to-br from-slate-50 via-white to-indigo-50/50 p-4 md:p-5 shadow-sm">
        <div class="flex items-start justify-between gap-3 mb-4">
          <div>
            <h3 class="text-sm font-semibold text-slate-800">上游转发策略</h3>
            <p class="text-xs text-slate-500 mt-1">支持在同一个路由下配置多个上游节点，也可以直接引用已有上游。</p>
          </div>

        </div>

        <div class="grid grid-cols-1 md:grid-cols-3 gap-3 mb-4">
          <button
            v-for="item in upstreamModeCards"
            :key="item.value"
            type="button"
            @click="setUpstreamMode(item.value as ApisixRouteUpstreamMode)"
            :class="[
              'text-left rounded-2xl border p-4 transition-colors duration-200 shadow-sm',
              formData.upstream_mode === item.value
                ? item.tone === 'indigo'
                  ? 'border-indigo-300 bg-indigo-50 text-indigo-700 shadow-indigo-100'
                  : item.tone === 'emerald'
                    ? 'border-emerald-300 bg-emerald-50 text-emerald-700 shadow-emerald-100'
                    : 'border-slate-300 bg-slate-100 text-slate-700'
                : 'border-slate-200 bg-white text-slate-600 hover:border-slate-300'
            ]"
          >
            <div class="flex items-center justify-between gap-3 mb-2">
              <div class="w-10 h-10 rounded-xl flex items-center justify-center"
                :class="[
                  formData.upstream_mode === item.value
                    ? item.tone === 'indigo'
                      ? 'bg-indigo-100'
                      : item.tone === 'emerald'
                        ? 'bg-emerald-100'
                        : 'bg-slate-200'
                    : 'bg-slate-100'
                ]"
              >
                <i class="fas text-sm" :class="item.icon"></i>
              </div>
              <i v-if="formData.upstream_mode === item.value" class="fas fa-circle-check text-base"></i>
            </div>
            <div class="text-sm font-semibold">{{ item.title }}</div>
            <div class="text-xs mt-1 opacity-80 leading-5">{{ item.desc }}</div>
          </button>
        </div>

        <div v-if="formData.upstream_mode === 'nodes'" class="space-y-4">
          <div class="flex flex-col md:flex-row md:items-center md:justify-between gap-3 rounded-2xl bg-white border border-slate-200 p-4 shadow-sm">
            <div class="grid grid-cols-2 md:grid-cols-4 gap-3 flex-1">
              <div class="rounded-xl bg-slate-50 border border-slate-200 px-3 py-2">
                <div class="text-[11px] uppercase tracking-wider text-slate-400 font-semibold">节点行数</div>
                <div class="text-lg font-semibold text-slate-800 mt-1">{{ formData.upstream_nodes.length }}</div>
              </div>
              <div class="rounded-xl bg-slate-50 border border-slate-200 px-3 py-2">
                <div class="text-[11px] uppercase tracking-wider text-slate-400 font-semibold">有效节点</div>
                <div class="text-lg font-semibold text-indigo-600 mt-1">{{ selectedInlineNodeCount }}</div>
              </div>
              <div class="col-span-2 md:col-span-2 rounded-xl bg-slate-50 border border-slate-200 px-3 py-2">
                <div class="text-[11px] uppercase tracking-wider text-slate-400 font-semibold">负载均衡策略</div>
                <div class="text-sm font-semibold text-slate-800 mt-1">{{ selectedUpstreamTypeOption.label }}</div>
                <div class="text-xs text-slate-400 mt-1">{{ selectedUpstreamTypeOption.desc }}</div>
              </div>
            </div>
            <button type="button" @click="addUpstreamNode()" class="inline-flex items-center justify-center gap-2 px-4 py-2 rounded-xl bg-indigo-500 hover:bg-indigo-600 text-white text-sm font-medium transition-colors shadow-sm">
              <i class="fas fa-plus"></i>
              添加上游节点
            </button>
          </div>

          <div class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
            <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">负载均衡策略</label>
            <select v-model="formData.upstream_type" class="input">
              <option v-for="option in upstreamTypeOptions" :key="option.value" :value="option.value">
                {{ option.label }}
              </option>
            </select>
            <p class="text-xs text-slate-400 mt-1">{{ selectedUpstreamTypeOption.desc }}</p>
          </div>

          <div v-if="formData.upstream_type === 'chash'" class="rounded-2xl border border-violet-200 bg-violet-50/40 p-4 shadow-sm space-y-3">
            <div class="flex items-center gap-2 mb-1">
              <i class="fas fa-fingerprint text-violet-500 text-sm"></i>
              <span class="text-sm font-semibold text-violet-800">一致性哈希参数</span>
            </div>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
              <div>
                <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">哈希依据（hash_on）</label>
                <select v-model="formData.upstream_hash_on" class="input">
                  <option v-for="o in hashOnOptions" :key="o.value" :value="o.value">{{ o.label }}</option>
                </select>
                <p class="text-xs text-slate-400 mt-1">决定用什么来计算哈希值</p>
              </div>
              <div>
                <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">哈希键（key）</label>
                <input v-model="formData.upstream_key" type="text" class="input" :placeholder="selectedHashOnOption.keyPlaceholder" />
                <p class="text-xs text-slate-400 mt-1">{{ selectedHashOnOption.keyHint }}</p>
              </div>
            </div>
          </div>

          <div class="space-y-3">
            <div
              v-for="(node, index) in formData.upstream_nodes"
              :key="`node-${index}`"
              class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm"
            >
              <div class="flex items-center justify-between gap-3 mb-4">
                <div class="flex items-center gap-3 min-w-0">
                  <div class="w-9 h-9 rounded-xl bg-indigo-100 text-indigo-700 flex items-center justify-center font-semibold text-sm">{{ index + 1 }}</div>
                  <div>
                    <div class="text-sm font-semibold text-slate-800">上游节点 {{ index + 1 }}</div>
                    <div class="text-xs text-slate-400">支持容器选择或直接手动输入主机与端口</div>
                  </div>
                </div>
                <button type="button" @click="removeUpstreamNode(index)" class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-red-50 text-red-600 hover:bg-red-100 text-xs font-medium transition-colors">
                  <i class="fas fa-trash"></i>
                  删除
                </button>
              </div>

              <div class="grid grid-cols-1 md:grid-cols-12 gap-4 items-start">
                <div class="md:col-span-5">
                  <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">上游主机</label>
                  <HostSelect
                    :model-value="node.host"
                    :containers="containers"
                    placeholder="127.0.0.1 或 容器名"
                    @update:modelValue="updateUpstreamNodeHost(index, $event)"
                  />
                </div>
                <div class="md:col-span-4">
                  <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">上游端口</label>
                  <PortSelect
                    :model-value="node.port"
                    :ports="getPortsByHost(node.host)"
                    placeholder="8080"
                    @update:modelValue="updateUpstreamNodePort(index, $event)"
                  />
                </div>
                <div class="md:col-span-3">
                  <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">节点权重</label>
                  <input v-model.number="node.weight" type="number" min="0" class="input" placeholder="1" />
                  <p class="text-xs text-slate-400 mt-1">当前策略 {{ formData.upstream_type }} 下的节点权重。</p>
                </div>
              </div>
            </div>
          </div>

          <div v-if="routeValidationMessage" class="rounded-xl border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-700">
            {{ routeValidationMessage }}
          </div>
        </div>

        <div v-else-if="formData.upstream_mode === 'upstream_id'" class="rounded-2xl border border-emerald-200 bg-white p-4 shadow-sm space-y-3">
          <div>
            <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">选择已有上游</label>
            <select v-model="formData.upstream_id" class="input">
              <option value="">请选择已有上游</option>
              <option v-for="upstream in upstreams" :key="upstream.id" :value="upstream.id">
                {{ upstream.name || upstream.id }}（{{ upstream.type || 'roundrobin' }}）
              </option>
            </select>
            <p class="text-xs text-slate-400 mt-1">适用于复用已在 APISIX 中维护的上游对象。</p>
          </div>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div class="rounded-xl bg-emerald-50 border border-emerald-100 px-3 py-3 text-xs text-emerald-700">
              <div class="font-semibold mb-1">为什么用引用模式</div>
              <div>当多个路由共享同一组上游配置时，只需要维护一个上游对象。</div>
            </div>
            <div class="rounded-xl bg-slate-50 border border-slate-200 px-3 py-3 text-xs text-slate-600">
              <div class="font-semibold mb-1">注意</div>
              <div>切换为引用模式后，本路由将不再提交内联节点列表。</div>
            </div>
          </div>
          <div v-if="routeValidationMessage" class="rounded-xl border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-700">
            {{ routeValidationMessage }}
          </div>
        </div>

        <div v-else class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm space-y-3">
          <div>
            <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">暂不配置上游</label>
            <p class="text-sm text-slate-600 leading-6">当前保存只会提交路由匹配规则、状态和插件配置，不会附带内联节点，也不会引用已有上游。</p>
          </div>
          <div class="rounded-xl bg-slate-50 border border-slate-200 px-3 py-3 text-xs text-slate-600">
            <div class="font-semibold mb-1">适用场景</div>
            <div>先创建占位路由，后续再由手工或其他流程补充上游配置。</div>
          </div>
        </div>

      </div>

      <div>
        <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">复用插件配置</label>
        <select v-model="formData.plugin_config_id" class="input">
          <option value="">不使用</option>
          <option v-for="pc in pluginConfigs" :key="pc.id" :value="pc.id">{{ pc.desc || pc.id }}</option>
        </select>
      </div>

      <div class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
        <div class="flex flex-col md:flex-row md:items-center md:justify-between gap-3 mb-3">
          <div>
            <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">独立插件配置</label>
            <p class="text-xs text-slate-400">可直接编辑 JSON，也支持从现有路由导入。</p>
          </div>
          <div class="flex items-center gap-2">
            <button @click="showPluginPanel = !showPluginPanel; showImportPanel = false" class="px-3 py-1.5 text-xs rounded-lg bg-indigo-50 text-indigo-600 hover:bg-indigo-100 transition-colors border border-indigo-100"><i class="fas fa-puzzle-piece mr-1"></i>添加插件</button>
            <button @click="showImportPanel = !showImportPanel; showPluginPanel = false" class="px-3 py-1.5 text-xs rounded-lg bg-slate-50 text-slate-600 hover:bg-slate-100 transition-colors border border-slate-200"><i class="fas fa-file-import mr-1"></i>从路由导入</button>
          </div>
        </div>

        <div v-if="showPluginPanel" class="mb-3 p-3 bg-slate-50 rounded-xl border border-slate-200">
          <div class="mb-2">
            <input v-model="pluginSearchKeyword" type="text" placeholder="搜索插件..." class="input text-xs" />
          </div>
          <div class="max-h-40 overflow-y-auto grid grid-cols-1 md:grid-cols-3 gap-2">
            <button
              v-for="name in filteredAvailablePlugins"
              :key="name"
              @click="addPresetPlugin(name)"
              :class="[
                'px-3 py-2 text-xs rounded-xl text-left truncate transition-colors border',
                formData.plugins[name] !== undefined
                  ? 'bg-indigo-100 text-indigo-700 border-indigo-200 cursor-default'
                  : 'bg-white hover:bg-indigo-50 text-slate-700 border-slate-200'
              ]"
              :disabled="formData.plugins[name] !== undefined"
            >
              {{ name }}
            </button>
          </div>
        </div>

        <div v-if="showImportPanel" class="mb-3 p-3 bg-slate-50 rounded-xl border border-slate-200">
          <div class="mb-2">
            <select v-model="importRouteId" @change="onImportRouteChange" class="input text-xs">
              <option value="">选择来源路由...</option>
              <option v-for="r in routes" :key="r.id" :value="r.id">{{ r.name || r.id }}</option>
            </select>
          </div>
          <div v-if="importRoutePluginsLoading" class="text-xs text-slate-500 text-center py-2">加载中...</div>
          <div v-else-if="Object.keys(importRoutePlugins).length > 0">
            <div class="max-h-32 overflow-y-auto space-y-1 mb-2">
              <label v-for="(_, name) in importRoutePlugins" :key="name" class="flex items-center gap-2 text-xs cursor-pointer rounded-lg px-2 py-1 hover:bg-white">
                <input type="checkbox" :checked="selectedImportPlugins.has(name)" @change="toggleImportPlugin(name)" class="rounded border-slate-300 text-indigo-500 focus:ring-indigo-500" />
                <span>{{ name }}</span>
              </label>
            </div>
            <button @click="importPluginsFromRoute" class="px-3 py-1.5 text-xs bg-indigo-500 text-white rounded-lg hover:bg-indigo-600 transition-colors">导入选中插件</button>
          </div>
        </div>

        <div v-if="currentPluginNames.length > 0" class="flex flex-wrap gap-1.5 mb-3">
          <span v-for="name in currentPluginNames" :key="name" class="inline-flex items-center gap-1 px-2.5 py-1 bg-indigo-50 text-indigo-700 rounded-full text-xs border border-indigo-100">
            {{ name }}
            <button @click="removePlugin(name)" class="hover:text-red-500 transition-colors"><i class="fas fa-xmark text-[10px]"></i></button>
          </span>
        </div>

        <textarea v-model="formData.pluginsJson" @blur="syncPluginsFromJson" rows="8" :class="['input font-mono text-sm', formData.pluginsJsonError ? 'border-red-300 bg-red-50' : '']" placeholder='{"key-auth": {}, "proxy-rewrite": {"uri": "/new-path"}}'></textarea>
        <p v-if="formData.pluginsJsonError" class="text-xs text-red-500 mt-1">{{ formData.pluginsJsonError }}</p>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-2">
        <button @click="isOpen = false" class="px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-200 rounded-lg hover:bg-slate-50">取消</button>
        <button @click="handleConfirm()" :disabled="modalLoading" class="px-4 py-2 text-sm font-medium text-white bg-indigo-500 rounded-lg hover:bg-indigo-600 disabled:opacity-50 shadow-sm">
          <i v-if="modalLoading" class="fas fa-spinner fa-spin mr-1"></i>{{ isEditMode ? '保存' : '创建' }}
        </button>
      </div>
    </template>
  </BaseModal>
</template>
