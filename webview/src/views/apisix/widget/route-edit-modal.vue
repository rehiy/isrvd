<script setup>
import { computed, inject, reactive, ref } from 'vue'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'
import { parseUpstreamNode, buildRoutePayload } from '@/helper/utils.js'

import BaseModal from '@/component/modal.vue'

const actions = inject(APP_ACTIONS_KEY)

const emit = defineEmits(['success'])

const isOpen = ref(false)
const modalLoading = ref(false)
const isEditMode = ref(false)
const editingRouteId = ref('')
const showPluginPanel = ref(false)
const showImportPanel = ref(false)
const importRouteId = ref('')
const importRoutePlugins = ref({})
const importRoutePluginsLoading = ref(false)
const selectedImportPlugins = ref(new Set())
const pluginSearchKeyword = ref('')

const pluginConfigs = ref([])
const upstreams = ref([])
const availablePlugins = ref({})
const routes = ref([])

const formData = reactive({
  name: '', desc: '', uris: '', hosts: '',
  status: 1, priority: 0, enable_websocket: false,
  plugin_config_id: '', upstream_host: '', upstream_port: '',
  plugins: {}, pluginsJson: '{}', pluginsJsonError: '',
})

const currentPluginNames = computed(() => Object.keys(formData.plugins || {}))
const filteredAvailablePlugins = computed(() => {
  const all = Object.keys(availablePlugins.value)
  if (!pluginSearchKeyword.value) return all
  return all.filter(n => n.toLowerCase().includes(pluginSearchKeyword.value.toLowerCase()))
})

const resetForm = () => {
  Object.assign(formData, { name: '', desc: '', uris: '', hosts: '', status: 1, priority: 0, enable_websocket: false, plugin_config_id: '', upstream_host: '', upstream_port: '', plugins: {}, pluginsJson: '{}', pluginsJsonError: '' })
  editingRouteId.value = ''; showPluginPanel.value = false; showImportPanel.value = false; importRouteId.value = ''; importRoutePlugins.value = {}; selectedImportPlugins.value = new Set()
}

const loadResources = async (allRoutes) => {
  routes.value = allRoutes || []
  try {
    const [pc, us, pl] = await Promise.all([api.apisixListPluginConfigs(), api.apisixListUpstreams(), api.apisixListPlugins()])
    pluginConfigs.value = pc.payload || []; upstreams.value = us.payload || []; availablePlugins.value = pl.payload || {}
  } catch {}
}

const show = async (route, allRoutes) => {
  await loadResources(allRoutes)
  if (route) {
    isEditMode.value = true
    editingRouteId.value = route.id
    resetForm()
    modalLoading.value = true
    isOpen.value = true
    try {
      const r = (await api.apisixGetRoute(route.id)).payload
      const plugins = r.plugins || {}
      const { host: uH, port: uP } = parseUpstreamNode(r.upstream)
      Object.assign(formData, { name: r.name || '', desc: r.desc || '', uris: (r.uris?.length ? r.uris : [r.uri || '']).join('\n'), hosts: (r.hosts?.length ? r.hosts : [r.host || '']).join('\n'), status: r.status ?? 0, priority: r.priority ?? 0, enable_websocket: r.enable_websocket || false, plugin_config_id: r.plugin_config_id || '', upstream_host: uH, upstream_port: uP, plugins, pluginsJson: JSON.stringify(plugins, null, 2), pluginsJsonError: '' })
      editingRouteId.value = route.id
    } catch (e) {
      actions.showNotification('error', '加载路由详情失败')
      isOpen.value = false
    }
    modalLoading.value = false
  } else {
    isEditMode.value = false
    resetForm()
    isOpen.value = true
  }
}

const syncPluginsFromJson = () => {
  try { formData.plugins = JSON.parse(formData.pluginsJson || '{}'); formData.pluginsJsonError = '' }
  catch (e) { formData.pluginsJsonError = 'JSON 格式错误: ' + e.message }
}

const removePlugin = (name) => { const p = { ...formData.plugins }; delete p[name]; formData.plugins = p; formData.pluginsJson = JSON.stringify(p, null, 2) }

const TYPE_DEFAULTS = { string: '', integer: 0, number: 0, boolean: false, array: [], object: {} }
const buildPluginDefault = (schema) => {
  if (!schema?.properties) return {}
  const required = new Set(schema.required || [])
  const result = {}
  for (const [key, def] of Object.entries(schema.properties)) {
    if (key === 'disable') continue
    if (required.has(key) || def.default !== undefined) result[key] = def.default !== undefined ? def.default : (TYPE_DEFAULTS[def.type] ?? null)
  }
  return result
}

const addPresetPlugin = (name) => {
  if (formData.plugins[name] !== undefined) return actions.showNotification('warning', `插件 ${name} 已存在`)
  const schema = availablePlugins.value[name]?.schema
  const p = { ...formData.plugins, [name]: buildPluginDefault(schema) }
  formData.plugins = p; formData.pluginsJson = JSON.stringify(p, null, 2); showPluginPanel.value = false; pluginSearchKeyword.value = ''
}

const onImportRouteChange = async () => {
  importRoutePlugins.value = {}; selectedImportPlugins.value = new Set()
  if (!importRouteId.value) return
  importRoutePluginsLoading.value = true
  try { const src = (await api.apisixGetRoute(importRouteId.value)).payload?.plugins || {}; importRoutePlugins.value = src; selectedImportPlugins.value = new Set(Object.keys(src)) }
  catch (e) { actions.showNotification('error', '加载路由插件失败') }
  importRoutePluginsLoading.value = false
}

const toggleImportPlugin = (name) => { const s = new Set(selectedImportPlugins.value); s.has(name) ? s.delete(name) : s.add(name); selectedImportPlugins.value = s }

const importPluginsFromRoute = () => {
  if (!importRouteId.value) return actions.showNotification('warning', '请先选择要导入的路由')
  if (selectedImportPlugins.value.size === 0) return actions.showNotification('warning', '请至少勾选一个插件')
  const toImport = {}
  for (const name of selectedImportPlugins.value) { if (importRoutePlugins.value[name] !== undefined) toImport[name] = importRoutePlugins.value[name] }
  const merged = { ...formData.plugins, ...toImport }
  formData.plugins = merged; formData.pluginsJson = JSON.stringify(merged, null, 2)
  showImportPanel.value = false; importRouteId.value = ''; importRoutePlugins.value = {}; selectedImportPlugins.value = new Set()
  actions.showNotification('success', `已导入 ${Object.keys(toImport).length} 个插件`)
}

const handleConfirm = async () => {
  if (!formData.name.trim()) return actions.showNotification('error', '路由名称不能为空')
  if (!formData.uris.split('\n').map(s => s.trim()).filter(Boolean).length) return actions.showNotification('error', 'URI 不能为空')
  if (formData.pluginsJsonError) return actions.showNotification('error', '请修正 Plugin JSON 格式错误')
  modalLoading.value = true
  try {
    const payload = buildRoutePayload(formData)
    if (isEditMode.value) { await api.apisixUpdateRoute(editingRouteId.value, payload); actions.showNotification('success', '路由更新成功') }
    else { await api.apisixCreateRoute(payload); actions.showNotification('success', '路由创建成功') }
    isOpen.value = false
    resetForm()
    emit('success')
  } catch (e) { actions.showNotification('error', e.message || '操作失败') }
  modalLoading.value = false
}

defineExpose({ show })
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
