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
} from '@/helper/apisix'

import BaseModal from '@/component/modal.vue'
import HostSelect from './host-select.vue'
import PortSelect from './port-select.vue'

// ─── 模块级静态常量 ───

const UPSTREAM_MODE_CARDS: ApisixRouteUpstreamModeCard[] = [
    { value: 'nodes', title: '内联多上游节点', desc: '一个路由直连多个节点，并为每个节点设置权重', icon: 'fa-circle-nodes', tone: 'indigo' },
    { value: 'upstream_id', title: '引用已有上游', desc: '复用 APISIX 中已经创建好的上游对象', icon: 'fa-diagram-project', tone: 'emerald' },
    { value: 'none', title: '空上游', desc: '暂不配置转发目标，先保存路由规则', icon: 'fa-ban', tone: 'slate' }
]

const UPSTREAM_TYPE_OPTIONS: Array<{ value: ApisixUpstreamType; label: string; desc: string }> = [
    { value: 'roundrobin', label: 'roundrobin', desc: '按权重轮询分配请求' },
    { value: 'least_conn', label: 'least_conn', desc: '优先选择当前连接数更少的节点' },
    { value: 'ewma', label: 'ewma', desc: '根据历史延迟动态选择更快的节点' },
    { value: 'chash', label: 'chash', desc: '一致性哈希，适合会话粘性场景' }
]

const HASH_ON_OPTIONS: Array<{ value: ApisixUpstreamHashOn; label: string; keyPlaceholder: string; keyHint: string }> = [
    { value: 'vars', label: 'vars（Nginx 变量）', keyPlaceholder: 'remote_addr', keyHint: 'Nginx 变量名，不带 $ 前缀，如 remote_addr、uri' },
    { value: 'header', label: 'header（请求头）', keyPlaceholder: 'X-User-Id', keyHint: '请求头名称，如 X-User-Id' },
    { value: 'cookie', label: 'cookie', keyPlaceholder: 'session_id', keyHint: 'Cookie 名称（大小写敏感），如 session_id' },
    { value: 'consumer', label: 'consumer（消费者）', keyPlaceholder: 'consumer_name', keyHint: '通常填 consumer_name，由 APISIX 自动注入' },
    { value: 'vars_combinations', label: 'vars_combinations', keyPlaceholder: '$remote_addr$uri', keyHint: '多个 Nginx 变量组合，如 $remote_addr$uri' }
]

const TYPE_DEFAULTS: Record<string, string | number | boolean | unknown[] | Record<string, unknown>> = { string: '', integer: 0, number: 0, boolean: false, array: [], object: {} }

const TONE_CARD_ACTIVE: Record<string, string> = {
    indigo: 'border-indigo-300 bg-indigo-50 text-indigo-700',
    emerald: 'border-emerald-300 bg-emerald-50 text-emerald-700',
    slate: 'border-slate-300 bg-slate-100 text-slate-700'
}
const TONE_ICON_ACTIVE: Record<string, string> = { indigo: 'bg-indigo-100', emerald: 'bg-emerald-100', slate: 'bg-slate-200' }

const defaultFormData = () => ({
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
    timeout_read: '' as string | number,
})

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

    formData = defaultFormData()

    // 暴露常量给模板
    readonly upstreamModeCards = UPSTREAM_MODE_CARDS
    readonly upstreamTypeOptions = UPSTREAM_TYPE_OPTIONS
    readonly hashOnOptions = HASH_ON_OPTIONS

    // ─── 计算属性 ───
    get currentPluginNames() { return Object.keys(this.formData.plugins || {}) }

    get filteredAvailablePlugins() {
        const kw = this.pluginSearchKeyword.toLowerCase()
        const all = Object.keys(this.availablePlugins)
        return kw ? all.filter(n => n.toLowerCase().includes(kw)) : all
    }

    get selectedUpstreamTypeOption() {
        return UPSTREAM_TYPE_OPTIONS.find(o => o.value === this.formData.upstream_type) ?? UPSTREAM_TYPE_OPTIONS[0]
    }

    get selectedHashOnOption() {
        return HASH_ON_OPTIONS.find(o => o.value === this.formData.upstream_hash_on) ?? HASH_ON_OPTIONS[0]
    }

    get routeValidationMessage() {
        const mode = this.formData.upstream_mode
        if (mode === 'upstream_id') return this.formData.upstream_id.trim() ? '' : '请选择要引用的上游对象'
        if (mode !== 'nodes') return ''

        const rows = this.formData.upstream_nodes
        if (!rows.some(n => n.host.trim() && String(n.port).trim())) return '请至少配置一个完整的上游节点'

        for (const [i, node] of rows.entries()) {
            const hasHost = !!node.host.trim()
            const hasPort = !!String(node.port).trim()
            if (hasHost !== hasPort) return `第 ${i + 1} 个上游节点的主机和端口需要同时填写`
            if (hasPort && !/^\d+$/.test(String(node.port).trim())) return `第 ${i + 1} 个上游节点端口必须为数字`
            if ((hasHost || hasPort) && Number(node.weight) < 0) return `第 ${i + 1} 个上游节点权重不能为负数`
        }

        if (rows.length > 1 && this.formData.upstream_type === 'chash' && !this.formData.upstream_key?.trim()) {
            return '使用 chash 策略时，哈希键（key）不能为空'
        }

        return ''
    }

    // ─── 模式卡片样式 ───
    modeCardClass(item: ApisixRouteUpstreamModeCard) {
        const base = 'text-left rounded-xl border p-4 transition-colors'
        const active = this.formData.upstream_mode === item.value
        return `${base} ${active ? TONE_CARD_ACTIVE[item.tone] : 'border-slate-200 bg-white text-slate-600 hover:border-slate-300'}`
    }

    modeCardIconClass(item: ApisixRouteUpstreamModeCard) {
        const active = this.formData.upstream_mode === item.value
        return `w-10 h-10 rounded-xl flex items-center justify-center ${active ? TONE_ICON_ACTIVE[item.tone] : 'bg-slate-100'}`
    }

    // ─── 方法 ───
    resetForm() {
        Object.assign(this.formData, defaultFormData())
        this.editingRouteId = ''
        this.originalUpstream = null
        this.showPluginPanel = false
        this.showImportPanel = false
        this.importRouteId = ''
        this.importRoutePlugins = {}
        this.selectedImportPlugins = new Set()
        this.pluginSearchKeyword = ''
    }

    createUpstreamNode(): ApisixRouteUpstreamFormNode {
        return { host: '', port: '', weight: 1 }
    }

    // ─── 上游节点管理 ───
    addUpstreamNode() {
        this.formData.upstream_nodes = [...this.formData.upstream_nodes, this.createUpstreamNode()]
    }

    removeUpstreamNode(index: number) {
        const next = this.formData.upstream_nodes.filter((_, i) => i !== index)
        this.formData.upstream_nodes = next.length > 0 ? next : [this.createUpstreamNode()]
    }

    setUpstreamMode(mode: ApisixRouteUpstreamMode) {
        this.formData.upstream_mode = mode
        if (mode === 'nodes' && !this.formData.upstream_nodes.length) {
            this.formData.upstream_nodes = [this.createUpstreamNode()]
        }
    }

    getPortsByHost(host: string): string[] {
        return this.containers.find(c => c.name === host.trim())?.ports || []
    }

    updateUpstreamNode(index: number, field: 'host' | 'port', value: string) {
        const next = [...this.formData.upstream_nodes]
        const node = next[index]
        if (!node) return
        node[field] = value
        if (field === 'host' && value.trim()) {
            const port = (this.getPortsByHost(value)[0] || '').split('/')[0].split(':').pop() || ''
            if (port) node.port = port
        }
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
        if (route?.id) {
            this.isEditMode = true
            this.resetForm()
            this.editingRouteId = route.id
            this.modalLoading = true
            this.isOpen = true
            try {
                const r = (await api.apisixGetRoute(route.id)).payload
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
                    timeout_read: r.timeout?.read ?? '',
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

    buildPluginDefault(schema: { properties?: Record<string, { type: string; default?: unknown }>; required?: string[] }) {
        if (!schema?.properties) return {}
        const required = new Set(schema.required || [])
        const result: Record<string, unknown> = {}
        for (const [key, def] of Object.entries(schema.properties)) {
            if (key === 'disable') continue
            if (required.has(key) || def.default !== undefined) {
                result[key] = def.default !== undefined ? def.default : (TYPE_DEFAULTS[def.type] ?? null)
            }
        }
        return result
    }

    addPresetPlugin(name: string) {
        if (this.formData.plugins[name] !== undefined) {
            return this.actions.showNotification('warning', `插件 ${name} 已存在`)
        }
        const p = { ...this.formData.plugins, [name]: this.buildPluginDefault(this.availablePlugins[name]?.schema) }
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
        if (!this.selectedImportPlugins.size) return this.actions.showNotification('warning', '请至少勾选一个插件')
        const toImport = Object.fromEntries(
            [...this.selectedImportPlugins]
                .filter(n => this.importRoutePlugins[n] !== undefined)
                .map(n => [n, this.importRoutePlugins[n]])
        )
        const merged = { ...this.formData.plugins, ...toImport }
        this.formData.plugins = merged
        this.formData.pluginsJson = JSON.stringify(merged, null, 2)
        this.showImportPanel = false
        this.importRouteId = ''
        this.importRoutePlugins = {}
        this.selectedImportPlugins = new Set()
        this.actions.showNotification('success', `已导入 ${Object.keys(toImport).length} 个插件`)
    }

    async handleConfirm() {
        if (!this.formData.name.trim()) return this.actions.showNotification('error', '路由名称不能为空')
        if (!this.formData.uris.split('\n').map(s => s.trim()).filter(Boolean).length) return this.actions.showNotification('error', 'URI 不能为空')
        if (this.formData.pluginsJsonError) return this.actions.showNotification('error', '请修正 Plugin JSON 格式错误')
        if (this.routeValidationMessage) return this.actions.showNotification('error', this.routeValidationMessage)

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
    <div class="space-y-4 p-1">
      <div class="grid grid-cols-2 gap-3">
        <div><label class="block text-sm font-medium text-slate-700 mb-2">路由名称 <span class="text-red-500">*</span></label><input v-model="formData.name" type="text" class="input" placeholder="路由名称" /></div>
        <div><label class="block text-sm font-medium text-slate-700 mb-2">优先级</label><input v-model.number="formData.priority" type="number" class="input" placeholder="0" min="0" /></div>
      </div>
      <div><label class="block text-sm font-medium text-slate-700 mb-2">描述</label><textarea v-model="formData.desc" rows="2" class="input" placeholder="路由描述"></textarea></div>
      <div><label class="block text-sm font-medium text-slate-700 mb-2">URI（每行一个）<span class="text-red-500">*</span></label><textarea v-model="formData.uris" rows="3" class="input font-mono text-sm" placeholder="/api/v1/*&#10;/api/v2/*"></textarea></div>
      <div><label class="block text-sm font-medium text-slate-700 mb-2">Host（每行一个，留空匹配所有）</label><textarea v-model="formData.hosts" rows="2" class="input font-mono text-sm" placeholder="example.com"></textarea></div>

      <div class="border border-slate-200 rounded-xl p-4">
        <div class="flex items-center justify-between mb-3">
          <div>
            <label class="block text-sm font-medium text-slate-700">上游配置</label>
            <p class="text-xs text-slate-400 mt-1">支持多节点、引用已有上游或暂不配置上游</p>
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-3 gap-3 mb-4">
          <button v-for="item in upstreamModeCards" :key="item.value" type="button" :class="modeCardClass(item)" @click="setUpstreamMode(item.value)">
            <div class="flex items-center gap-2 mb-1">
              <div :class="modeCardIconClass(item)"><i class="fas text-sm" :class="item.icon"></i></div>
              <span class="text-sm font-semibold">{{ item.title }}</span>
            </div>
            <div class="text-xs opacity-80 leading-5">{{ item.desc }}</div>
          </button>
        </div>

        <div v-if="formData.upstream_mode === 'nodes'" class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="text-sm font-medium text-slate-700">上游节点</label>
            <button @click="addUpstreamNode()" type="button" class="btn-icon text-indigo-600 hover:bg-indigo-50" title="添加节点"><i class="fas fa-plus text-xs"></i></button>
          </div>
          <div class="space-y-2">
            <div class="grid grid-cols-[2fr_1fr_1fr_32px] gap-2 text-xs font-semibold text-slate-500 uppercase tracking-wider">
              <span class="whitespace-nowrap">主机</span><span class="whitespace-nowrap">端口</span><span class="whitespace-nowrap">权重</span><span v-if="formData.upstream_nodes.length > 1"></span>
            </div>
            <div v-for="(node, index) in formData.upstream_nodes" :key="index" class="grid grid-cols-[2fr_1fr_1fr_32px] gap-2 items-center">
              <HostSelect :model-value="node.host" :containers="containers" placeholder="127.0.0.1 或 容器名" @update:modelValue="updateUpstreamNode(index, 'host', $event)" />
              <PortSelect :model-value="node.port" :ports="getPortsByHost(node.host)" placeholder="80" @update:modelValue="updateUpstreamNode(index, 'port', $event)" />
              <input v-model.number="node.weight" type="number" class="input" placeholder="1" min="0" />
              <button :disabled="formData.upstream_nodes.length < 2" @click="removeUpstreamNode(index)" type="button" class="btn-icon text-red-400 hover:text-red-600 hover:bg-red-50 disabled:opacity-50" title="移除">
                <i class="fas fa-xmark text-xs"></i>
              </button>
            </div>
          </div>
          <div v-if="formData.upstream_nodes.length > 1" class="space-y-2">
            <label class="block text-sm font-medium text-slate-700">负载均衡策略</label>
            <div class="grid grid-cols-2 md:grid-cols-4 gap-2">
              <button
                v-for="o in upstreamTypeOptions"
                :key="o.value"
                type="button"
                @click="formData.upstream_type = o.value"
                :class="['text-left rounded-lg border px-3 py-2 transition-colors', formData.upstream_type === o.value ? 'border-indigo-300 bg-indigo-50 text-indigo-700' : 'border-slate-200 bg-white text-slate-600 hover:border-slate-300']"
              >
                <div class="text-xs font-semibold">{{ o.label }}</div>
                <div class="text-xs opacity-70 mt-0.5 leading-4">{{ o.desc }}</div>
              </button>
            </div>
            <div v-if="formData.upstream_type === 'chash'" class="grid grid-cols-1 md:grid-cols-2 gap-3 pt-1">
              <div>
                <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">哈希依据（hash_on）</label>
                <select v-model="formData.upstream_hash_on" class="input">
                  <option v-for="o in hashOnOptions" :key="o.value" :value="o.value">{{ o.label }}</option>
                </select>
                <p class="text-xs text-slate-400 mt-1">{{ selectedHashOnOption.keyHint }}</p>
              </div>
              <div>
                <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">哈希键（key）</label>
                <input v-model="formData.upstream_key" type="text" class="input" :placeholder="selectedHashOnOption.keyPlaceholder" />
              </div>
            </div>
          </div>
          <div v-if="routeValidationMessage" class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-700">{{ routeValidationMessage }}</div>
        </div>

        <div v-else-if="formData.upstream_mode === 'upstream_id'" class="space-y-3">
          <div>
            <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">选择已有上游</label>
            <select v-model="formData.upstream_id" class="input">
              <option value="">请选择已有上游</option>
              <option v-for="upstream in upstreams" :key="upstream.id" :value="upstream.id">{{ upstream.name || upstream.id }}（{{ upstream.type || 'roundrobin' }}）</option>
            </select>
          </div>
          <div v-if="routeValidationMessage" class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-700">{{ routeValidationMessage }}</div>
        </div>
        <div v-else class="text-sm text-slate-500 leading-6">当前保存只会提交路由匹配规则、状态和插件配置，不会附带内联节点，也不会引用已有上游。</div>
      </div>

      <div class="border border-slate-200 rounded-xl p-4">
        <div class="mb-3">
          <label class="block text-sm font-medium text-slate-700">复用插件配置</label>
          <p class="text-xs text-slate-400 mt-1 mb-2">选择已有的插件配置对象，与独立插件配置合并生效</p>
          <select v-model="formData.plugin_config_id" class="input">
            <option value="">不使用</option>
            <option v-for="pc in pluginConfigs" :key="pc.id" :value="pc.id">{{ pc.desc || pc.id }}</option>
          </select>
        </div>

        <div class="pt-3 border-t border-slate-100 space-y-3">
          <!-- 独立插件配置标题行 -->
          <div class="flex items-center justify-between">
            <div>
              <label class="block text-sm font-medium text-slate-700">独立插件配置</label>
              <p class="text-xs text-slate-400 mt-0.5">可直接编辑 JSON，也支持从现有路由导入</p>
            </div>
            <div class="flex items-center gap-2">
              <button @click="showPluginPanel = !showPluginPanel; showImportPanel = false" :class="['px-3 py-1.5 text-xs rounded-lg border transition-colors', showPluginPanel ? 'border-indigo-300 text-indigo-600 bg-indigo-50' : 'border-slate-200 text-slate-600 hover:border-slate-300 hover:bg-slate-50']"><i class="fas fa-puzzle-piece mr-1"></i>添加插件</button>
              <button @click="showImportPanel = !showImportPanel; showPluginPanel = false" :class="['px-3 py-1.5 text-xs rounded-lg border transition-colors', showImportPanel ? 'border-indigo-300 text-indigo-600 bg-indigo-50' : 'border-slate-200 text-slate-600 hover:border-slate-300 hover:bg-slate-50']"><i class="fas fa-file-import mr-1"></i>从路由导入</button>
            </div>
          </div>

          <!-- 插件选择面板 -->
          <div v-if="showPluginPanel" class="rounded-lg border border-slate-200 overflow-hidden">
            <div class="px-3 py-2 bg-slate-50 border-b border-slate-100">
              <div class="relative">
                <i class="fas fa-search absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400 text-xs pointer-events-none"></i>
                <input v-model="pluginSearchKeyword" type="text" placeholder="搜索插件..." class="w-full pl-7 pr-3 py-1.5 text-xs bg-white border border-slate-200 rounded-md focus:outline-none focus:ring-1 focus:ring-indigo-400 focus:border-indigo-300" />
              </div>
            </div>
            <div class="max-h-44 overflow-y-auto p-2 grid grid-cols-2 md:grid-cols-3 gap-1">
              <button
                v-for="name in filteredAvailablePlugins"
                :key="name"
                :disabled="formData.plugins[name] !== undefined"
                :class="['px-2.5 py-1.5 text-xs rounded-lg text-left truncate transition-colors border', formData.plugins[name] !== undefined ? 'bg-indigo-50 text-indigo-500 border-indigo-100 cursor-default' : 'bg-white hover:bg-slate-50 text-slate-700 border-slate-200 hover:border-slate-300']"
                @click="addPresetPlugin(name)"
              ><i v-if="formData.plugins[name] !== undefined" class="fas fa-check text-[9px] mr-1"></i>{{ name }}</button>
            </div>
          </div>

          <!-- 从路由导入面板 -->
          <div v-if="showImportPanel" class="rounded-lg border border-slate-200 overflow-hidden">
            <div class="px-3 py-2 bg-slate-50 border-b border-slate-100">
              <select v-model="importRouteId" @change="onImportRouteChange" class="w-full text-xs bg-white border border-slate-200 rounded-md px-2.5 py-1.5 focus:outline-none focus:ring-1 focus:ring-indigo-400 focus:border-indigo-300">
                <option value="">选择来源路由...</option>
                <option v-for="r in routes" :key="r.id" :value="r.id">{{ r.name || r.id }}</option>
              </select>
            </div>
            <div v-if="importRoutePluginsLoading" class="py-5 text-center text-xs text-slate-400"><i class="fas fa-spinner fa-spin mr-1"></i>加载中...</div>
            <div v-else-if="Object.keys(importRoutePlugins).length > 0" class="p-2 space-y-2">
              <div class="max-h-32 overflow-y-auto space-y-0.5">
                <label v-for="(_, name) in importRoutePlugins" :key="name" class="flex items-center gap-2 text-xs cursor-pointer rounded-lg px-2.5 py-1.5 hover:bg-slate-50 transition-colors">
                  <input type="checkbox" :checked="selectedImportPlugins.has(name)" @change="toggleImportPlugin(name)" class="rounded border-slate-300 text-indigo-500 focus:ring-indigo-500" />
                  <span class="text-slate-700">{{ name }}</span>
                </label>
              </div>
              <button @click="importPluginsFromRoute" class="w-full py-1.5 text-xs bg-indigo-500 text-white rounded-lg hover:bg-indigo-600 transition-colors font-medium">导入选中 {{ selectedImportPlugins.size }} 个插件</button>
            </div>
            <div v-else-if="importRouteId" class="py-5 text-center text-xs text-slate-400">该路由没有插件配置</div>
          </div>

          <!-- 已添加插件 tags -->
          <div v-if="currentPluginNames.length > 0" class="flex flex-wrap gap-1">
            <span v-for="name in currentPluginNames" :key="name" class="inline-flex items-center gap-1 px-2 py-0.5 bg-indigo-50 text-indigo-700 rounded text-xs">{{ name }}<button @click="removePlugin(name)" class="hover:text-red-500 transition-colors"><i class="fas fa-xmark text-[10px]"></i></button></span>
          </div>

          <!-- JSON 编辑器 -->
          <textarea v-model="formData.pluginsJson" @blur="syncPluginsFromJson" rows="8" :class="['input font-mono text-sm', formData.pluginsJsonError ? 'border-red-300 bg-red-50' : '']" placeholder='{"key-auth": {}, "proxy-rewrite": {"uri": "/new-path"}}'></textarea>
          <p v-if="formData.pluginsJsonError" class="text-xs text-red-500 mt-1">{{ formData.pluginsJsonError }}</p>
        </div>
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
