<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import { APP_ACTIONS_KEY } from '@/store/state'
import { parseUpstreamNode, buildRoutePayload } from '@/helper/utils'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class RouteEditModal extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: any

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    isEditMode = false
    editingRouteId = ''
    showPluginPanel = false
    showImportPanel = false
    importRouteId = ''
    importRoutePlugins: Record<string, any> = {}
    importRoutePluginsLoading = false
    selectedImportPlugins: Set<string> = new Set()
    pluginSearchKeyword = ''

    pluginConfigs: any[] = []
    upstreams: any[] = []
    availablePlugins: Record<string, any> = {}
    routes: any[] = []

    formData = {
        name: '', desc: '', uris: '', hosts: '',
        status: 1, priority: 0, enable_websocket: false,
        plugin_config_id: '', upstream_host: '', upstream_port: '',
        plugins: {} as Record<string, any>, pluginsJson: '{}', pluginsJsonError: ''
    }

    // ─── 计算属性 ───
    get currentPluginNames() { return Object.keys(this.formData.plugins || {}) }

    get filteredAvailablePlugins() {
        const all = Object.keys(this.availablePlugins)
        if (!this.pluginSearchKeyword) return all
        return all.filter(n => n.toLowerCase().includes(this.pluginSearchKeyword.toLowerCase()))
    }

    // ─── 方法 ───
    resetForm() {
        Object.assign(this.formData, {
            name: '', desc: '', uris: '', hosts: '', status: 1, priority: 0,
            enable_websocket: false, plugin_config_id: '', upstream_host: '', upstream_port: '',
            plugins: {}, pluginsJson: '{}', pluginsJsonError: ''
        })
        this.editingRouteId = ''
        this.showPluginPanel = false
        this.showImportPanel = false
        this.importRouteId = ''
        this.importRoutePlugins = {}
        this.selectedImportPlugins = new Set()
    }

    async loadResources(allRoutes: any[]) {
        this.routes = allRoutes || []
        try {
            const [pc, us, pl] = await Promise.all([
                api.apisixListPluginConfigs(), api.apisixListUpstreams(), api.apisixListPlugins()
            ])
            this.pluginConfigs = pc.payload || []
            this.upstreams = us.payload || []
            this.availablePlugins = pl.payload || {}
        } catch {}
    }

    async show(route: any, allRoutes: any[]) {
        await this.loadResources(allRoutes)
        if (route) {
            this.isEditMode = true
            this.editingRouteId = route.id
            this.resetForm()
            this.modalLoading = true
            this.isOpen = true
            try {
                const r = (await api.apisixGetRoute(route.id)).payload
                const plugins = r.plugins || {}
                const { host: uH, port: uP } = parseUpstreamNode(r.upstream)
                Object.assign(this.formData, {
                    name: r.name || '', desc: r.desc || '',
                    uris: (r.uris?.length ? r.uris : [r.uri || '']).join('\n'),
                    hosts: (r.hosts?.length ? r.hosts : [r.host || '']).join('\n'),
                    status: r.status ?? 0, priority: r.priority ?? 0,
                    enable_websocket: r.enable_websocket || false,
                    plugin_config_id: r.plugin_config_id || '',
                    upstream_host: uH, upstream_port: uP,
                    plugins, pluginsJson: JSON.stringify(plugins, null, 2), pluginsJsonError: ''
                })
                this.editingRouteId = route.id
            } catch (e) {
                this.actions.showNotification('error', '加载路由详情失败')
                this.isOpen = false
            }
            this.modalLoading = false
        } else {
            this.isEditMode = false
            this.resetForm()
            this.isOpen = true
        }
    }

    syncPluginsFromJson() {
        try {
            this.formData.plugins = JSON.parse(this.formData.pluginsJson || '{}')
            this.formData.pluginsJsonError = ''
        } catch (e: any) {
            this.formData.pluginsJsonError = 'JSON 格式错误: ' + e.message
        }
    }

    removePlugin(name: string) {
        const p = { ...this.formData.plugins }
        delete p[name]
        this.formData.plugins = p
        this.formData.pluginsJson = JSON.stringify(p, null, 2)
    }

    readonly TYPE_DEFAULTS: Record<string, any> = { string: '', integer: 0, number: 0, boolean: false, array: [], object: {} }

    buildPluginDefault(schema: any) {
        if (!schema?.properties) return {}
        const required = new Set(schema.required || [])
        const result: Record<string, any> = {}
        for (const [key, def] of Object.entries(schema.properties) as any[]) {
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
        } catch (e) {
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
        const toImport: Record<string, any> = {}
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

    async handleConfirm() {
        if (!this.formData.name.trim()) return this.actions.showNotification('error', '路由名称不能为空')
        if (!this.formData.uris.split('\n').map((s: string) => s.trim()).filter(Boolean).length) return this.actions.showNotification('error', 'URI 不能为空')
        if (this.formData.pluginsJsonError) return this.actions.showNotification('error', '请修正 Plugin JSON 格式错误')
        this.modalLoading = true
        try {
            const payload = buildRoutePayload(this.formData)
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
        } catch (e: any) {
            this.actions.showNotification('error', e.message || '操作失败')
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
        <div><label class="block text-sm font-medium text-slate-700 mb-2">优先级</label><input v-model.number="formData.priority" type="number" class="input" placeholder="0" /></div>
      </div>
      <div><label class="block text-sm font-medium text-slate-700 mb-2">描述</label><textarea v-model="formData.desc" rows="2" class="input" placeholder="路由描述"></textarea></div>
      <div><label class="block text-sm font-medium text-slate-700 mb-2">URI（每行一个）<span class="text-red-500">*</span></label><textarea v-model="formData.uris" rows="3" class="input font-mono text-sm" placeholder="/api/v1/*&#10;/api/v2/*"></textarea></div>
      <div><label class="block text-sm font-medium text-slate-700 mb-2">Host（每行一个，留空匹配所有）</label><textarea v-model="formData.hosts" rows="2" class="input font-mono text-sm" placeholder="example.com"></textarea></div>
      <div class="grid grid-cols-2 gap-3">
        <div><label class="block text-sm font-medium text-slate-700 mb-2">上游主机</label><input v-model="formData.upstream_host" type="text" class="input" placeholder="127.0.0.1" /></div>
        <div><label class="block text-sm font-medium text-slate-700 mb-2">上游端口</label><input v-model="formData.upstream_port" type="text" class="input" placeholder="8080" /></div>
      </div>
      <div class="grid grid-cols-2 gap-3">
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">WebSocket</label>
          <select v-model="formData.enable_websocket" class="input">
            <option :value="false">关闭</option>
            <option :value="true">开启</option>
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">路由状态</label>
          <select v-model="formData.status" class="input">
            <option :value="1">启用</option>
            <option :value="0">禁用</option>
          </select>
        </div>
      </div>
      <div><label class="block text-sm font-medium text-slate-700 mb-2">复用插件配置</label>
        <select v-model="formData.plugin_config_id" class="input"><option value="">不使用</option><option v-for="pc in pluginConfigs" :key="pc.id" :value="pc.id">{{ pc.desc || pc.id }}</option></select>
      </div>
      <div>
        <div class="flex items-center justify-between mb-1">
          <label class="text-sm font-medium text-slate-700">独立插件配置</label>
          <div class="flex items-center gap-1">
            <button @click="showPluginPanel = !showPluginPanel; showImportPanel = false" class="px-2 py-0.5 text-xs rounded bg-indigo-50 text-indigo-600 hover:bg-indigo-100 transition-colors"><i class="fas fa-puzzle-piece mr-1"></i>添加插件</button>
            <button @click="showImportPanel = !showImportPanel; showPluginPanel = false" class="px-2 py-0.5 text-xs rounded bg-slate-50 text-slate-600 hover:bg-slate-100 transition-colors"><i class="fas fa-file-import mr-1"></i>从路由导入</button>
          </div>
        </div>
        <div v-if="showPluginPanel" class="mb-2 p-3 bg-slate-50 rounded-lg border border-slate-200">
          <div class="mb-2"><input v-model="pluginSearchKeyword" type="text" placeholder="搜索插件..." class="input text-xs" /></div>
          <div class="max-h-40 overflow-y-auto grid grid-cols-3 gap-1">
            <button v-for="name in filteredAvailablePlugins" :key="name" @click="addPresetPlugin(name)" :class="['px-2 py-1 text-xs rounded text-left truncate transition-colors', formData.plugins[name] !== undefined ? 'bg-indigo-100 text-indigo-700 cursor-default' : 'bg-white hover:bg-indigo-50 text-slate-700']" :disabled="formData.plugins[name] !== undefined">{{ name }}</button>
          </div>
        </div>
        <div v-if="showImportPanel" class="mb-2 p-3 bg-slate-50 rounded-lg border border-slate-200">
          <div class="mb-2"><select v-model="importRouteId" @change="onImportRouteChange" class="input text-xs"><option value="">选择来源路由...</option><option v-for="r in routes" :key="r.id" :value="r.id">{{ r.name || r.id }}</option></select></div>
          <div v-if="importRoutePluginsLoading" class="text-xs text-slate-500 text-center py-2">加载中...</div>
          <div v-else-if="Object.keys(importRoutePlugins).length > 0">
            <div class="max-h-32 overflow-y-auto space-y-1 mb-2"><label v-for="(_, name) in importRoutePlugins" :key="name" class="flex items-center gap-2 text-xs cursor-pointer"><input type="checkbox" :checked="selectedImportPlugins.has(name)" @change="toggleImportPlugin(name)" class="rounded border-slate-300 text-indigo-500 focus:ring-indigo-500" /><span>{{ name }}</span></label></div>
            <button @click="importPluginsFromRoute" class="px-3 py-1 text-xs bg-indigo-500 text-white rounded hover:bg-indigo-600 transition-colors">导入选中插件</button>
          </div>
        </div>
        <div v-if="currentPluginNames.length > 0" class="flex flex-wrap gap-1 mb-2">
          <span v-for="name in currentPluginNames" :key="name" class="inline-flex items-center gap-1 px-2 py-0.5 bg-indigo-50 text-indigo-700 rounded text-xs">{{ name }}<button @click="removePlugin(name)" class="hover:text-red-500 transition-colors"><i class="fas fa-xmark text-[10px]"></i></button></span>
        </div>
        <textarea v-model="formData.pluginsJson" @blur="syncPluginsFromJson" rows="8" :class="['input font-mono text-sm', formData.pluginsJsonError ? 'border-red-300 bg-red-50' : '']" placeholder='{"key-auth": {}, "proxy-rewrite": {"uri": "/new-path"}}'></textarea>
        <p v-if="formData.pluginsJsonError" class="text-xs text-red-500 mt-1">{{ formData.pluginsJsonError }}</p>
      </div>
    </div>
    <template #footer>
      <div class="flex justify-end gap-2">
        <button @click="isOpen = false" class="px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-200 rounded-lg hover:bg-slate-50">取消</button>
        <button @click="handleConfirm()" :disabled="modalLoading" class="px-4 py-2 text-sm font-medium text-white bg-indigo-500 rounded-lg hover:bg-indigo-600 disabled:opacity-50"><i v-if="modalLoading" class="fas fa-spinner fa-spin mr-1"></i>{{ isEditMode ? '保存' : '创建' }}</button>
      </div>
    </template>
  </BaseModal>
</template>
