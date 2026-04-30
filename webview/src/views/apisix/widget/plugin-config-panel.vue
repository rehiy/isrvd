<script lang="ts">
import { Component, Inject, Prop, Vue, Watch, toNative } from 'vue-facing-decorator'

import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import api from '@/service/api'
import type { ApisixRoute } from '@/service/types'

const TYPE_DEFAULTS: Record<string, string | number | boolean | unknown[] | Record<string, unknown>> = { string: '', integer: 0, number: 0, boolean: false, array: [], object: {} }

@Component({
    expose: ['pluginsJsonError'],
    emits: ['update:plugins']
})
class PluginConfigPanel extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    @Prop({ type: Object, default: () => ({}) }) plugins!: Record<string, unknown>
    @Prop({ type: Object, default: () => ({}) }) availablePlugins!: Record<string, { schema: Record<string, unknown> }>
    @Prop({ type: Boolean, default: false }) showImport!: boolean
    @Prop({ type: Array, default: () => [] }) routes!: ApisixRoute[]

    // ─── 数据属性 ───
    showPluginPanel = false
    showImportPanel = false
    pluginSearchKeyword = ''
    pluginsJson = '{}'
    pluginsJsonError = ''
    isEditingJson = false

    // 从路由导入
    importRouteId = ''
    importRoutePlugins: Record<string, unknown> = {}
    importRoutePluginsLoading = false
    selectedImportPlugins: Set<string> = new Set()

    // ─── 生命周期 ───
    created() {
        this.syncJsonFromPlugins()
    }

    // ─── Watchers ───
    @Watch('plugins', { deep: true })
    onPluginsChanged() {
        if (!this.isEditingJson) {
            this.syncJsonFromPlugins()
        }
    }

    // ─── 计算属性 ───
    get currentPluginNames() { return Object.keys(this.plugins || {}) }

    get filteredAvailablePlugins() {
        const kw = this.pluginSearchKeyword.toLowerCase()
        const all = Object.keys(this.availablePlugins)
        return kw ? all.filter(n => n.toLowerCase().includes(kw)) : all
    }

    // ─── 方法 ───
    syncJsonFromPlugins() {
        const next = JSON.stringify(this.plugins, null, 2)
        if (next !== this.pluginsJson) {
            this.pluginsJson = next
            this.pluginsJsonError = ''
        }
    }

    syncPluginsFromJson() {
        this.isEditingJson = false
        try {
            const parsed = JSON.parse(this.pluginsJson || '{}')
            this.pluginsJsonError = ''
            if (JSON.stringify(parsed) !== JSON.stringify(this.plugins)) {
                this.$emit('update:plugins', parsed)
            }
        } catch (e: unknown) {
            this.pluginsJsonError = 'JSON 格式错误: ' + (e instanceof Error ? e.message : String(e))
        }
    }

    removePlugin(name: string) {
        const p = { ...this.plugins }
        delete p[name]
        this.$emit('update:plugins', p)
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
        if (this.plugins[name] !== undefined) {
            return this.actions.showNotification('warning', `插件 ${name} 已存在`)
        }
        const p = { ...this.plugins, [name]: this.buildPluginDefault(this.availablePlugins[name]?.schema) }
        this.$emit('update:plugins', p)
        this.showPluginPanel = false
        this.pluginSearchKeyword = ''
    }

    onPluginsJsonInput(event: Event) {
        this.isEditingJson = true
        this.pluginsJson = (event.target as HTMLTextAreaElement).value
    }

    // ─── 从路由导入 ───
    openPluginPanel() {
        this.showPluginPanel = true
        this.showImportPanel = false
    }

    openImportPanel() {
        this.showImportPanel = true
        this.showPluginPanel = false
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
        const merged = { ...this.plugins, ...toImport }
        this.$emit('update:plugins', merged)
        this.showImportPanel = false
        this.importRouteId = ''
        this.importRoutePlugins = {}
        this.selectedImportPlugins = new Set()
        this.actions.showNotification('success', `已导入 ${Object.keys(toImport).length} 个插件`)
    }
}

export default toNative(PluginConfigPanel)
</script>

<template>
  <div>
    <!-- 标题行 -->
    <div class="flex items-center justify-between">
      <div>
        <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">插件配置</label>
        <p class="text-xs text-slate-400 mt-1">{{ showImport ? '可直接编辑 JSON，也支持从现有路由导入' : '可直接编辑 JSON，或从插件列表添加' }}</p>
      </div>
      <div class="flex items-center gap-2">
        <button @click="openPluginPanel" :class="['px-3 py-1.5 text-xs rounded-lg border transition-colors', showPluginPanel ? 'border-indigo-300 text-indigo-600 bg-indigo-50' : 'border-slate-200 text-slate-600 hover:border-slate-300 hover:bg-slate-50']"><i class="fas fa-puzzle-piece mr-1"></i>添加插件</button>
        <button v-if="showImport" @click="openImportPanel" :class="['px-3 py-1.5 text-xs rounded-lg border transition-colors', showImportPanel ? 'border-indigo-300 text-indigo-600 bg-indigo-50' : 'border-slate-200 text-slate-600 hover:border-slate-300 hover:bg-slate-50']"><i class="fas fa-file-import mr-1"></i>从路由导入</button>
      </div>
    </div>

    <!-- 插件选择面板 -->
    <div v-if="showPluginPanel" class="rounded-lg border border-slate-200 overflow-hidden mt-3">
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
          :disabled="plugins[name] !== undefined"
          :class="['px-2.5 py-1.5 text-xs rounded-lg text-left truncate transition-colors border', plugins[name] !== undefined ? 'bg-indigo-50 text-indigo-500 border-indigo-100 cursor-default' : 'bg-white hover:bg-slate-50 text-slate-700 border-slate-200 hover:border-slate-300']"
          @click="addPresetPlugin(name)"
        ><i v-if="plugins[name] !== undefined" class="fas fa-check text-[9px] mr-1"></i>{{ name }}</button>
      </div>
    </div>

    <!-- 从路由导入面板 -->
    <div v-if="showImportPanel" class="rounded-lg border border-slate-200 overflow-hidden mt-3">
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
    <div v-if="currentPluginNames.length > 0" class="flex flex-wrap gap-1 mt-3">
      <span v-for="name in currentPluginNames" :key="name" class="inline-flex items-center gap-1 px-2 py-0.5 bg-indigo-50 text-indigo-700 rounded text-xs">{{ name }}<button @click="removePlugin(name)" class="hover:text-red-500 transition-colors"><i class="fas fa-xmark text-[10px]"></i></button></span>
    </div>

    <!-- JSON 编辑器 -->
    <textarea :value="pluginsJson" @blur="syncPluginsFromJson" @input="onPluginsJsonInput" rows="8" :class="['input font-mono text-sm mt-3', pluginsJsonError ? 'border-red-300 bg-red-50' : '']" placeholder='{"key-auth": {"key": "your-api-key"}}'></textarea>
    <p v-if="pluginsJsonError" class="text-xs text-red-500 mt-1">{{ pluginsJsonError }}</p>
  </div>
</template>
