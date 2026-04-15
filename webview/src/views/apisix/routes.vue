<script setup>
import BaseModal from '@/component/modal.vue'
import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'
import { parseUpstreamNode, buildRoutePayload } from '@/helper/utils.js'
import { computed, inject, onMounted, reactive, ref } from 'vue'

const actions = inject(APP_ACTIONS_KEY)
const routes = ref([])
const pluginConfigs = ref([])
const upstreams = ref([])
const availablePlugins = ref({})
const loading = ref(false)
const searchText = ref('')
const modalOpen = ref(false)
const modalTitle = ref('')
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

const formData = reactive({
  name: '', desc: '', uris: '', hosts: '',
  status: 1, priority: 0, enable_websocket: false,
  plugin_config_id: '', upstream_host: '', upstream_port: '',
  plugins: {}, pluginsJson: '{}', pluginsJsonError: '',
})

const sortRoutes = (data) => {
  data.sort((a, b) => {
    const hostA = (a.hosts?.[0]) || a.host || ''
    const hostB = (b.hosts?.[0]) || b.host || ''
    const hc = hostA.localeCompare(hostB)
    if (hc !== 0) return hc
    const uriA = (a.uris?.[0]) || a.uri || ''
    const uriB = (b.uris?.[0]) || b.uri || ''
    return uriA.localeCompare(uriB)
  })
  return data
}

const filteredRoutes = computed(() => {
  if (!searchText.value) return routes.value
  const s = searchText.value.toLowerCase()
  return routes.value.filter(r =>
    (r.name || '').toLowerCase().includes(s) ||
    (r.id || '').toLowerCase().includes(s) ||
    (r.uri || '').toLowerCase().includes(s) ||
    (r.uris || []).some(u => u.toLowerCase().includes(s)) ||
    (r.desc || '').toLowerCase().includes(s)
  )
})

const currentPluginNames = computed(() => Object.keys(formData.plugins || {}))
const filteredAvailablePlugins = computed(() => {
  const all = Object.keys(availablePlugins.value)
  if (!pluginSearchKeyword.value) return all
  return all.filter(n => n.toLowerCase().includes(pluginSearchKeyword.value.toLowerCase()))
})

const loadRoutes = async () => {
  loading.value = true
  try { routes.value = sortRoutes((await api.apisixListRoutes()).payload || []) }
  catch { actions.showNotification('error', '加载路由列表失败') }
  loading.value = false
}

const loadResources = async () => {
  try {
    const [pc, us, pl] = await Promise.all([api.apisixListPluginConfigs(), api.apisixListUpstreams(), api.apisixListPlugins()])
    pluginConfigs.value = pc.payload || []; upstreams.value = us.payload || []; availablePlugins.value = pl.payload || {}
  } catch {}
}

const getRouteUri = (r) => r.uris?.length ? r.uris.join(', ') : (r.uri || '-')
const getRouteHost = (r) => r.hosts?.length ? r.hosts.join(', ') : (r.host || '*')

const resetForm = () => {
  Object.assign(formData, { name: '', desc: '', uris: '', hosts: '', status: 1, priority: 0, enable_websocket: false, plugin_config_id: '', upstream_host: '', upstream_port: '', plugins: {}, pluginsJson: '{}', pluginsJsonError: '' })
  editingRouteId.value = ''; showPluginPanel.value = false; showImportPanel.value = false; importRouteId.value = ''; importRoutePlugins.value = {}; selectedImportPlugins.value = new Set()
}

const openCreateModal = () => { isEditMode.value = false; modalTitle.value = '创建路由'; resetForm(); modalOpen.value = true }

const openEditModal = async (route) => {
  isEditMode.value = true; editingRouteId.value = route.id; modalTitle.value = '编辑路由'; resetForm(); modalLoading.value = true; modalOpen.value = true
  try {
    const r = (await api.apisixGetRoute(route.id)).payload
    const plugins = r.plugins || {}; const { host: uH, port: uP } = parseUpstreamNode(r.upstream)
    Object.assign(formData, { name: r.name || '', desc: r.desc || '', uris: (r.uris?.length ? r.uris : [r.uri || '']).join('\n'), hosts: (r.hosts?.length ? r.hosts : [r.host || '']).join('\n'), status: r.status ?? 0, priority: r.priority ?? 0, enable_websocket: r.enable_websocket || false, plugin_config_id: r.plugin_config_id || '', upstream_host: uH, upstream_port: uP, plugins, pluginsJson: JSON.stringify(plugins, null, 2), pluginsJsonError: '' })
    editingRouteId.value = route.id
  } catch (e) { actions.showNotification('error', '加载路由详情失败'); modalOpen.value = false }
  modalLoading.value = false
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

const submitForm = async () => {
  if (!formData.name.trim()) return actions.showNotification('error', '路由名称不能为空')
  if (!formData.uris.split('\n').map(s => s.trim()).filter(Boolean).length) return actions.showNotification('error', 'URI 不能为空')
  if (formData.pluginsJsonError) return actions.showNotification('error', '请修正 Plugin JSON 格式错误')
  modalLoading.value = true
  try {
    const payload = buildRoutePayload(formData)
    if (isEditMode.value) { await api.apisixUpdateRoute(editingRouteId.value, payload); actions.showNotification('success', '路由更新成功') }
    else { await api.apisixCreateRoute(payload); actions.showNotification('success', '路由创建成功') }
    modalOpen.value = false; resetForm(); loadRoutes()
  } catch (e) { actions.showNotification('error', e.message || '操作失败') }
  modalLoading.value = false
}

const toggleStatus = (route) => {
  const ns = route.status === 1 ? 0 : 1; const label = ns === 1 ? '启用' : '禁用'
  actions.showConfirm({ title: `${label}路由`, message: `确定要${label}路由 <strong class="text-slate-900">${route.name}</strong> 吗？`, icon: ns === 1 ? 'fa-toggle-on' : 'fa-toggle-off', iconColor: ns === 1 ? 'emerald' : 'amber', confirmText: `确认${label}`,
    onConfirm: async () => { await api.apisixPatchRouteStatus(route.id, ns); actions.showNotification('success', `路由已${label}`); loadRoutes() } })
}

const deleteRoute = (route) => {
  actions.showConfirm({ title: '删除路由', message: `确定要删除路由 <strong class="text-slate-900">${route.name || route.id}</strong> 吗？此操作不可恢复。`, icon: 'fa-trash', iconColor: 'red', confirmText: '确认删除', danger: true,
    onConfirm: async () => { await api.apisixDeleteRoute(route.id); actions.showNotification('success', '删除成功'); loadRoutes() } })
}

onMounted(() => { loadRoutes(); loadResources() })
</script>

<template>
  <div>
    <div class="card mb-4">
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-6 py-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-indigo-500 flex items-center justify-center"><i class="fas fa-route text-white"></i></div>
            <div><h1 class="text-lg font-semibold text-slate-800">路由管理</h1><p class="text-xs text-slate-500">管理 Apisix 路由（共 {{ routes.length }} 条）</p></div>
          </div>
          <div class="flex items-center gap-2">
            <div class="relative hidden md:block">
              <input v-model="searchText" type="text" placeholder="搜索路由..." class="pl-8 pr-3 py-1.5 text-xs border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent w-48" />
              <i class="fas fa-search absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400 text-xs"></i>
            </div>
            <button @click="loadRoutes()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors"><i class="fas fa-rotate"></i>刷新</button>
            <button @click="openCreateModal()" class="px-3 py-1.5 rounded-lg bg-indigo-500 hover:bg-indigo-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors"><i class="fas fa-plus"></i>创建</button>
          </div>
        </div>
      </div>
      <!-- 移动端搜索栏 -->
      <div class="md:hidden px-4 py-2 border-b border-slate-100">
        <div class="relative">
          <input v-model="searchText" type="text" placeholder="搜索路由..." class="w-full pl-8 pr-3 py-1.5 text-xs border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent" />
          <i class="fas fa-search absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400 text-xs"></i>
        </div>
      </div>
      <div v-if="loading" class="flex flex-col items-center justify-center py-20"><div class="w-12 h-12 spinner mb-3"></div><p class="text-slate-500">加载中...</p></div>
      <div v-else-if="filteredRoutes.length === 0" class="flex flex-col items-center justify-center py-20">
        <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4"><i class="fas fa-route text-4xl text-slate-300"></i></div>
        <p class="text-slate-600 font-medium mb-1">暂无路由</p>
        <p class="text-sm text-slate-400">点击「创建」添加新路由</p>
      </div>
      <div v-else class="space-y-3">
        <!-- 桌面端表格视图 -->
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead><tr class="bg-slate-50 border-b border-slate-200">
              <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">名称</th>
              <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">URI</th>
              <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">Host</th>
              <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">状态</th>
              <th class="w-32 px-4 py-3 text-right text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
            </tr></thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="route in filteredRoutes" :key="route.id" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3">
                  <div class="font-medium text-sm text-slate-800">{{ route.name || route.id }}</div>
                  <div v-if="route.desc" class="text-xs text-slate-400 mt-0.5 truncate max-w-xs">{{ route.desc }}</div>
                </td>
                <td class="px-4 py-3"><code class="text-xs bg-slate-100 px-2 py-1 rounded text-slate-700">{{ getRouteUri(route) }}</code></td>
                <td class="px-4 py-3"><span class="text-sm text-slate-600">{{ getRouteHost(route) }}</span></td>
                <td class="px-4 py-3">
                  <button @click="toggleStatus(route)" :class="['inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium cursor-pointer transition-colors', route.status === 1 ? 'bg-emerald-50 text-emerald-700 hover:bg-emerald-100' : 'bg-slate-100 text-slate-500 hover:bg-slate-200']">
                    <i :class="route.status === 1 ? 'fas fa-circle text-emerald-500' : 'fas fa-circle text-slate-400'" class="text-[6px]"></i>
                    {{ route.status === 1 ? '启用' : '禁用' }}
                  </button>
                </td>
                <td class="px-4 py-3">
                  <div class="flex justify-end items-center gap-1">
                    <button @click="openEditModal(route)" class="btn-icon text-indigo-600 hover:bg-indigo-50" title="编辑"><i class="fas fa-pen-to-square text-xs"></i></button>
                    <button @click="deleteRoute(route)" class="btn-icon text-red-600 hover:bg-red-50" title="删除"><i class="fas fa-trash text-xs"></i></button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 移动端卡片视图 -->
        <div class="md:hidden space-y-3">
          <div 
            v-for="route in filteredRoutes" 
            :key="route.id"
            class="rounded-xl border border-slate-200 bg-white p-4 transition-all hover:shadow-sm"
          >
            <!-- 顶部：路由信息和状态 -->
            <div class="flex items-center justify-between mb-3">
              <div class="min-w-0">
                <div class="font-medium text-sm text-slate-800">{{ route.name || route.id }}</div>
                <div v-if="route.desc" class="text-xs text-slate-400 mt-0.5 truncate">{{ route.desc }}</div>
              </div>
              <button @click="toggleStatus(route)" :class="['inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium cursor-pointer transition-colors', route.status === 1 ? 'bg-emerald-50 text-emerald-700 hover:bg-emerald-100' : 'bg-slate-100 text-slate-500 hover:bg-slate-200']">
                <i :class="route.status === 1 ? 'fas fa-circle text-emerald-500' : 'fas fa-circle text-slate-400'" class="text-[6px]"></i>
                {{ route.status === 1 ? '启用' : '禁用' }}
              </button>
            </div>
            
            <!-- 中间：URI和Host信息 -->
            <div class="mb-3">
              <p class="text-xs text-slate-500 mb-1">URI</p>
              <code class="text-xs bg-slate-100 px-2 py-1 rounded text-slate-700 break-all">{{ getRouteUri(route) }}</code>
            </div>
            
            <div class="mb-3">
              <p class="text-xs text-slate-500 mb-1">Host</p>
              <span class="text-sm text-slate-600 break-all">{{ getRouteHost(route) }}</span>
            </div>
            
            <!-- 底部：操作按钮 -->
            <div class="flex flex-wrap gap-1 pt-2 border-t border-slate-100">
              <button @click="openEditModal(route)" class="btn-icon text-indigo-600 hover:bg-indigo-50" title="编辑">
                <i class="fas fa-pen-to-square text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">编辑</span>
              </button>
              <button @click="deleteRoute(route)" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
                <i class="fas fa-trash text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">删除</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <BaseModal v-model="modalOpen" :title="modalTitle" size="lg">
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
          <button @click="modalOpen = false" class="px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-200 rounded-lg hover:bg-slate-50">取消</button>
          <button @click="submitForm()" :disabled="modalLoading" class="px-4 py-2 text-sm font-medium text-white bg-indigo-500 rounded-lg hover:bg-indigo-600 disabled:opacity-50"><i v-if="modalLoading" class="fas fa-spinner fa-spin mr-1"></i>{{ isEditMode ? '保存' : '创建' }}</button>
        </div>
      </template>
    </BaseModal>
  </div>
</template>
