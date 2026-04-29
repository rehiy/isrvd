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
    DockerContainerInfo
} from '@/service/types'

import {
    buildRoutePayload,
    detectRouteUpstreamMode,
    normalizeUpstreamFormNodes
} from '@/helper/apisix'

import BaseModal from '@/component/modal.vue'
import HostSelect from './host-select.vue'
import PluginConfigPanel from '@/views/apisix/widget/plugin-config-panel.vue'
import PortSelect from './port-select.vue'

// ─── 模块级静态常量 ───

const UPSTREAM_MODE_CARDS: ApisixRouteUpstreamModeCard[] = [
    { value: 'nodes', title: '内联上游节点', desc: '为当前路由配置一个后端服务地址', icon: 'fa-server', tone: 'indigo' },
    { value: 'upstream_id', title: '引用已有上游', desc: '复用已经创建好的上游对象', icon: 'fa-diagram-project', tone: 'emerald' },
    { value: 'none', title: '空上游', desc: '不配置转发目标，仅保存路由规则', icon: 'fa-ban', tone: 'slate' }
]

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
    upstream_mode: 'nodes' as ApisixRouteUpstreamMode,
    upstream_id: '',
    upstream_nodes: [{ host: '', port: '', weight: 1 }] as ApisixRouteUpstreamFormNode[],
    timeout_connect: '' as string | number,
    timeout_send: '' as string | number,
    timeout_read: '' as string | number,
})

@Component({
    expose: ['show'],
    components: { BaseModal, HostSelect, PluginConfigPanel, PortSelect },
    emits: ['success']
})
class RouteEditModal extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    isEditMode = false
    editingRouteId = ''
    originalUpstream: ApisixUpstreamConfig | null = null
    declare $refs: { pluginPanel: InstanceType<typeof PluginConfigPanel> }

    pluginConfigs: ApisixPluginConfig[] = []
    upstreams: ApisixUpstream[] = []
    containers: DockerContainerInfo[] = []
    availablePlugins: Record<string, { schema: Record<string, unknown> }> = {}
    routes: ApisixRoute[] = []

    formData = defaultFormData()

    // 暴露常量给模板
    readonly upstreamModeCards = UPSTREAM_MODE_CARDS

    // ─── 计算属性 ───
    get routeValidationMessage() {
        const mode = this.formData.upstream_mode
        if (mode === 'upstream_id') return this.formData.upstream_id.trim() ? '' : '请选择要引用的上游对象'
        if (mode !== 'nodes') return ''

        const node = this.formData.upstream_nodes[0] || this.createUpstreamNode()
        const hasHost = !!node.host.trim()
        const hasPort = !!String(node.port).trim()
        if (!hasHost && !hasPort) return '请填写上游主机和端口'
        if (hasHost !== hasPort) return '上游主机和端口需要同时填写'
        if (hasPort && !/^\d+$/.test(String(node.port).trim())) return '上游端口必须为数字'

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

    upstreamNodeSummary(upstream: ApisixUpstream) {
        const nodes = upstream.nodes
        if (Array.isArray(nodes)) {
            const labels = nodes.map(node => `${node.host || '-'}:${node.port || '-'}`)
            return labels.length > 2 ? `${labels.slice(0, 2).join(', ')} 等 ${labels.length} 个节点` : labels.join(', ')
        }
        if (nodes && typeof nodes === 'object') {
            const labels = Object.keys(nodes)
            return labels.length > 2 ? `${labels.slice(0, 2).join(', ')} 等 ${labels.length} 个节点` : labels.join(', ')
        }
        return '无节点'
    }

    upstreamOptionLabel(upstream: ApisixUpstream) {
        const name = upstream.name || upstream.id || '未命名上游'
        const type = upstream.type || 'roundrobin'
        const desc = upstream.desc ? `描述: ${upstream.desc}` : ''
        return [name, `类型: ${type}`, `节点: ${this.upstreamNodeSummary(upstream)}`, desc].filter(Boolean).join(' ｜ ')
    }

    // ─── 方法 ───
    resetForm() {
        Object.assign(this.formData, defaultFormData())
        this.editingRouteId = ''
        this.originalUpstream = null
    }

    createUpstreamNode(): ApisixRouteUpstreamFormNode {
        return { host: '', port: '', weight: 1 }
    }

    // ─── 上游节点管理 ───
    setUpstreamMode(mode: ApisixRouteUpstreamMode) {
        this.formData.upstream_mode = mode
        if (mode === 'nodes') {
            this.formData.upstream_nodes = [this.formData.upstream_nodes[0] || this.createUpstreamNode()]
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
                    upstream_mode: detectRouteUpstreamMode(r),
                    upstream_id: r.upstream_id || '',
                    upstream_nodes: normalizeUpstreamFormNodes(r.upstream).slice(0, 1),
                    timeout_connect: r.upstream?.timeout?.connect ?? '',
                    timeout_send: r.upstream?.timeout?.send ?? '',
                    timeout_read: r.upstream?.timeout?.read ?? '',
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

    onPluginsUpdate(plugins: Record<string, unknown>) {
        this.formData.plugins = plugins
    }

    async handleConfirm() {
        if (!this.formData.name.trim()) return this.actions.showNotification('error', '路由名称不能为空')
        if (!this.formData.uris.split('\n').map(s => s.trim()).filter(Boolean).length) return this.actions.showNotification('error', 'URI 不能为空')
        if (this.$refs.pluginPanel?.pluginsJsonError) return this.actions.showNotification('error', '请修正 Plugin JSON 格式错误')
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
        <div><label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">路由名称 <span class="text-red-500">*</span></label><input v-model="formData.name" type="text" class="input" placeholder="路由名称" /></div>
        <div><label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">优先级</label><input v-model.number="formData.priority" type="number" class="input" placeholder="0" min="0" /></div>
      </div>
      <div><label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">描述</label><textarea v-model="formData.desc" rows="2" class="input" placeholder="路由描述"></textarea></div>
      <div><label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">URI（每行一个）<span class="text-red-500">*</span></label><textarea v-model="formData.uris" rows="3" class="input font-mono text-sm" placeholder="/api/v1/*&#10;/api/v2/*"></textarea></div>
      <div><label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">Host（每行一个，留空匹配所有）</label><textarea v-model="formData.hosts" rows="2" class="input font-mono text-sm" placeholder="example.com"></textarea></div>

      <div class="border border-slate-200 rounded-xl p-4">
        <div class="flex items-center justify-between mb-3">
          <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider">上游配置</label>
            <p class="text-xs text-slate-400 mt-1">支持直接输入单个上游、引用已有上游或暂不配置上游</p>
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
          <div>
            <div class="grid grid-cols-[2fr_1fr] gap-2 items-center">
              <HostSelect :model-value="formData.upstream_nodes[0]?.host || ''" :containers="containers" placeholder="127.0.0.1 或 容器名" @update:modelValue="updateUpstreamNode(0, 'host', $event)" />
              <PortSelect :model-value="formData.upstream_nodes[0]?.port || ''" :ports="getPortsByHost(formData.upstream_nodes[0]?.host || '')" placeholder="80" @update:modelValue="updateUpstreamNode(0, 'port', $event)" />
            </div>
            <p class="text-xs text-slate-400 mt-2">直接输入模式仅提交一个上游节点；如需多节点负载均衡，请先在「上游管理」中创建后再引用。</p>
          </div>

          <div>
            <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">超时时间（秒）</label>
            <div class="grid grid-cols-1 md:grid-cols-3 gap-2">
              <input v-model.number="formData.timeout_connect" type="number" min="0" class="input" placeholder="连接 connect" />
              <input v-model.number="formData.timeout_send" type="number" min="0" class="input" placeholder="发送 send" />
              <input v-model.number="formData.timeout_read" type="number" min="0" class="input" placeholder="读取 read" />
            </div>
            <p class="text-xs text-slate-400 mt-1">留空或 0 表示使用 APISIX 默认超时。</p>
          </div>
          <div v-if="routeValidationMessage" class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-700">{{ routeValidationMessage }}</div>
        </div>

        <div v-else-if="formData.upstream_mode === 'upstream_id'" class="space-y-3">
          <div>
            <select v-model="formData.upstream_id" class="input">
              <option value="">请选择已有上游</option>
              <option v-for="upstream in upstreams" :key="upstream.id" :value="upstream.id">{{ upstreamOptionLabel(upstream) }}</option>
            </select>
          </div>
          <div v-if="routeValidationMessage" class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-700">{{ routeValidationMessage }}</div>
        </div>
        <div v-else class="text-sm text-slate-500 leading-6">当前保存只会提交路由匹配规则、状态和插件配置，不会附带内联节点，也不会引用已有上游。</div>
      </div>

      <div class="border border-slate-200 rounded-xl p-4">
        <div class="mb-3">
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">引用插件配置</label>
          <p class="text-xs text-slate-400 mt-1 mb-2">选择已有的插件配置对象，与独立插件配置合并生效</p>
          <select v-model="formData.plugin_config_id" class="input">
            <option value="">不使用</option>
            <option v-for="pc in pluginConfigs" :key="pc.id" :value="pc.id">{{ pc.desc || pc.id }}</option>
          </select>
        </div>

        <div class="pt-3 border-t border-slate-100">
          <PluginConfigPanel
            :plugins="formData.plugins"
            :available-plugins="availablePlugins"
            :show-import="true"
            :routes="routes"
            @update:plugins="onPluginsUpdate"
            ref="pluginPanel"
          />
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
