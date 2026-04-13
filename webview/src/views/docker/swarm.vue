<script setup>
import { inject, onMounted, ref } from 'vue'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const actions = inject(APP_ACTIONS_KEY)

// ─── Tab ───
const activeTab = ref('overview') // overview | nodes | services | tasks

// ─── 概览 ───
const swarmInfo = ref(null)
const infoLoading = ref(false)

const loadInfo = async () => {
  infoLoading.value = true
  try {
    const res = await api.swarmInfo()
    swarmInfo.value = res.payload || {}
  } catch (e) {
    actions.showNotification('error', '获取 Swarm 信息失败，请确认集群已初始化')
    swarmInfo.value = null
  }
  infoLoading.value = false
}

// ─── 节点 ───
const nodes = ref([])
const nodesLoading = ref(false)

const loadNodes = async () => {
  nodesLoading.value = true
  try {
    const res = await api.swarmListNodes()
    nodes.value = res.payload || []
  } catch (e) {
    actions.showNotification('error', '获取节点列表失败')
  }
  nodesLoading.value = false
}

const handleNodeAction = (node, action) => {
  const labels = { drain: '排空', active: '激活', pause: '暂停', remove: '移除' }
  const label = labels[action] || action
  actions.showConfirm({
    title: `${label}节点`,
    message: `确定要${label}节点 <strong class="text-slate-900">${node.hostname}</strong> 吗？`,
    icon: action === 'remove' ? 'fa-trash' : 'fa-server',
    iconColor: action === 'remove' ? 'red' : 'amber',
    confirmText: `确认${label}`,
    danger: action === 'remove',
    onConfirm: async () => {
      await api.swarmNodeAction(node.id, action)
      actions.showNotification('success', `节点${label}成功`)
      loadNodes()
    }
  })
}

// ─── 服务 ───
const services = ref([])
const servicesLoading = ref(false)
const scaleOpen = ref(false)
const scaleService = ref(null)
const scaleReplicas = ref(1)
const scaleLoading = ref(false)

const loadServices = async () => {
  servicesLoading.value = true
  try {
    const res = await api.swarmListServices()
    services.value = res.payload || []
  } catch (e) {
    actions.showNotification('error', '获取服务列表失败')
  }
  servicesLoading.value = false
}

const openScaleModal = (svc) => {
  scaleService.value = svc
  scaleReplicas.value = svc.replicas ?? 1
  scaleOpen.value = true
}

const handleScale = async () => {
  if (!scaleService.value) return
  scaleLoading.value = true
  try {
    await api.swarmServiceAction(scaleService.value.id, 'scale', scaleReplicas.value)
    actions.showNotification('success', '服务扩缩容成功')
    scaleOpen.value = false
    loadServices()
  } catch (e) {}
  scaleLoading.value = false
}

const handleServiceRemove = (svc) => {
  actions.showConfirm({
    title: '删除服务',
    message: `确定要删除服务 <strong class="text-slate-900">${svc.name}</strong> 吗？`,
    icon: 'fa-trash',
    iconColor: 'red',
    confirmText: '确认删除',
    danger: true,
    onConfirm: async () => {
      await api.swarmServiceAction(svc.id, 'remove')
      actions.showNotification('success', '服务删除成功')
      loadServices()
    }
  })
}

// ─── 任务 ───
const tasks = ref([])
const tasksLoading = ref(false)
const filterServiceID = ref('')

const loadTasks = async () => {
  tasksLoading.value = true
  try {
    const res = await api.swarmListTasks(filterServiceID.value)
    tasks.value = res.payload || []
  } catch (e) {
    actions.showNotification('error', '获取任务列表失败')
  }
  tasksLoading.value = false
}

// ─── Tab 切换 ───
const switchTab = (tab) => {
  activeTab.value = tab
  if (tab === 'overview') loadInfo()
  else if (tab === 'nodes') loadNodes()
  else if (tab === 'services') loadServices()
  else if (tab === 'tasks') loadTasks()
}

// ─── 状态样式 ───
const nodeStateClass = (state) => {
  if (state === 'ready') return 'bg-emerald-100 text-emerald-700'
  if (state === 'down') return 'bg-red-100 text-red-700'
  return 'bg-slate-100 text-slate-600'
}
const availabilityClass = (avail) => {
  if (avail === 'active') return 'bg-emerald-100 text-emerald-700'
  if (avail === 'drain') return 'bg-amber-100 text-amber-700'
  if (avail === 'pause') return 'bg-slate-100 text-slate-600'
  return 'bg-slate-100 text-slate-500'
}
const taskStateClass = (state) => {
  if (state === 'running') return 'bg-emerald-100 text-emerald-700'
  if (state === 'failed') return 'bg-red-100 text-red-700'
  if (state === 'complete') return 'bg-blue-100 text-blue-700'
  if (state === 'shutdown') return 'bg-slate-100 text-slate-500'
  return 'bg-amber-100 text-amber-700'
}

const statCards = [
  { key: 'nodes',    label: '总节点数',  icon: 'fa-server',       color: 'bg-blue-500' },
  { key: 'managers', label: 'Manager',   icon: 'fa-crown',        color: 'bg-indigo-500' },
  { key: 'workers',  label: 'Worker',    icon: 'fa-circle-nodes', color: 'bg-slate-500' },
  { key: 'services', label: '服务总数',  icon: 'fa-cubes',        color: 'bg-emerald-500' },
  { key: 'tasks',    label: '任务总数',  icon: 'fa-tasks',        color: 'bg-amber-500' },
]

onMounted(() => loadInfo())
</script>

<template>
  <div>
    <div class="card mb-4">
      <!-- Toolbar -->
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-6 py-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-cyan-600 flex items-center justify-center">
              <i class="fas fa-circle-nodes text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">Docker Swarm</h1>
              <p class="text-xs text-slate-500">集群节点、服务与任务管理</p>
            </div>
          </div>
          <!-- Tab 切换 -->
          <div class="flex items-center gap-1 bg-slate-100 p-1 rounded-lg">
            <button
              v-for="tab in [
                { key: 'overview',  label: '概览',  icon: 'fa-tachometer-alt' },
                { key: 'nodes',     label: '节点',  icon: 'fa-server' },
                { key: 'services',  label: '服务',  icon: 'fa-cubes' },
                { key: 'tasks',     label: '任务',  icon: 'fa-tasks' },
              ]"
              :key="tab.key"
              @click="switchTab(tab.key)"
              :class="[
                'px-3 py-1.5 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5',
                activeTab === tab.key ? 'bg-white text-cyan-600 shadow-sm' : 'text-slate-500 hover:text-slate-700'
              ]"
            >
              <i :class="['fas', tab.icon]"></i>{{ tab.label }}
            </button>
          </div>
        </div>
      </div>

      <!-- ─── 概览 Tab ─── -->
      <div v-if="activeTab === 'overview'">
        <div v-if="infoLoading" class="flex flex-col items-center justify-center py-20">
          <div class="w-12 h-12 spinner mb-3"></div>
          <p class="text-slate-500">加载中...</p>
        </div>
        <div v-else-if="swarmInfo" class="p-6">
          <div class="grid grid-cols-2 md:grid-cols-5 gap-4 mb-6">
            <div
              v-for="card in statCards" :key="card.key"
              class="relative overflow-hidden rounded-xl border border-slate-200 bg-white p-5 hover:shadow-md transition-shadow"
            >
              <div class="flex items-center gap-4">
                <div :class="['w-12 h-12 rounded-xl flex items-center justify-center', card.color]">
                  <i :class="['fas', card.icon, 'text-white text-lg']"></i>
                </div>
                <div>
                  <p class="text-sm text-slate-500">{{ card.label }}</p>
                  <p class="text-2xl font-bold text-slate-800 mt-1">{{ swarmInfo[card.key] ?? 0 }}</p>
                </div>
              </div>
            </div>
          </div>
          <div class="pt-4 border-t border-slate-200">
            <div class="grid grid-cols-2 gap-4 text-sm text-slate-600">
              <div><span class="text-xs text-slate-400">Cluster ID</span><p class="font-mono text-xs mt-0.5 truncate">{{ swarmInfo.clusterID || '-' }}</p></div>
              <div><span class="text-xs text-slate-400">创建时间</span><p class="mt-0.5">{{ swarmInfo.createdAt || '-' }}</p></div>
            </div>
          </div>
          <div class="mt-4 flex flex-wrap gap-2">
            <button @click="switchTab('nodes')" class="px-4 py-2 rounded-lg bg-blue-50 hover:bg-blue-100 text-blue-700 text-sm font-medium flex items-center gap-2 transition-colors">
              <i class="fas fa-server"></i>节点管理
            </button>
            <button @click="switchTab('services')" class="px-4 py-2 rounded-lg bg-emerald-50 hover:bg-emerald-100 text-emerald-700 text-sm font-medium flex items-center gap-2 transition-colors">
              <i class="fas fa-cubes"></i>服务管理
            </button>
            <button @click="switchTab('tasks')" class="px-4 py-2 rounded-lg bg-amber-50 hover:bg-amber-100 text-amber-700 text-sm font-medium flex items-center gap-2 transition-colors">
              <i class="fas fa-tasks"></i>任务列表
            </button>
          </div>
        </div>
        <div v-else class="flex flex-col items-center justify-center py-20">
          <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
            <i class="fas fa-circle-nodes text-4xl text-slate-300"></i>
          </div>
          <p class="text-slate-600 font-medium mb-1">Swarm 集群未初始化</p>
          <p class="text-sm text-slate-400">请先在 Docker 主机上执行 <code class="bg-slate-100 px-1.5 rounded">docker swarm init</code></p>
        </div>
      </div>

      <!-- ─── 节点 Tab ─── -->
      <div v-if="activeTab === 'nodes'">
        <div class="px-6 py-3 border-b border-slate-100 flex justify-end">
          <button @click="loadNodes" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
            <i class="fas fa-rotate"></i>刷新
          </button>
        </div>
        <div v-if="nodesLoading" class="flex flex-col items-center justify-center py-20">
          <div class="w-12 h-12 spinner mb-3"></div>
          <p class="text-slate-500">加载中...</p>
        </div>
        <div v-else-if="nodes.length > 0" class="overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">主机名</th>
                <th class="w-24 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">角色</th>
                <th class="w-28 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">状态</th>
                <th class="w-28 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">可用性</th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">地址</th>
                <th class="w-28 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">引擎版本</th>
                <th class="w-44 px-4 py-3 text-center text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="n in nodes" :key="n.id" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3">
                  <div class="flex items-center gap-2">
                    <div class="w-8 h-8 rounded-lg bg-blue-400 flex items-center justify-center">
                      <i class="fas fa-server text-white text-sm"></i>
                    </div>
                    <div>
                      <span class="font-medium text-slate-800">{{ n.hostname }}</span>
                      <span v-if="n.leader" class="ml-2 inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-indigo-100 text-indigo-700">
                        <i class="fas fa-crown mr-1 text-[10px]"></i>Leader
                      </span>
                    </div>
                  </div>
                </td>
                <td class="px-4 py-3">
                  <span :class="n.role === 'manager' ? 'bg-indigo-100 text-indigo-700' : 'bg-slate-100 text-slate-600'" class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize">{{ n.role }}</span>
                </td>
                <td class="px-4 py-3">
                  <span :class="nodeStateClass(n.state)" class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize">{{ n.state }}</span>
                </td>
                <td class="px-4 py-3">
                  <span :class="availabilityClass(n.availability)" class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize">{{ n.availability }}</span>
                </td>
                <td class="px-4 py-3 font-mono text-xs text-slate-500">{{ n.addr || '-' }}</td>
                <td class="px-4 py-3 text-xs text-slate-500">{{ n.engineVersion || '-' }}</td>
                <td class="px-4 py-3">
                  <div class="flex justify-center items-center gap-0.5">
                    <button v-if="n.availability !== 'active'" @click="handleNodeAction(n, 'active')" class="btn-icon text-emerald-600 hover:bg-emerald-50" title="激活"><i class="fas fa-play text-xs"></i></button>
                    <button v-if="n.availability !== 'drain'"  @click="handleNodeAction(n, 'drain')"  class="btn-icon text-amber-600 hover:bg-amber-50"   title="排空"><i class="fas fa-arrow-down text-xs"></i></button>
                    <button v-if="n.availability !== 'pause'"  @click="handleNodeAction(n, 'pause')"  class="btn-icon text-slate-600 hover:bg-slate-100"   title="暂停"><i class="fas fa-pause text-xs"></i></button>
                    <button v-if="n.role !== 'manager'"        @click="handleNodeAction(n, 'remove')" class="btn-icon text-red-600 hover:bg-red-50"         title="移除"><i class="fas fa-trash text-xs"></i></button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="flex flex-col items-center justify-center py-20">
          <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
            <i class="fas fa-server text-4xl text-slate-300"></i>
          </div>
          <p class="text-slate-600 font-medium mb-1">暂无节点</p>
        </div>
      </div>

      <!-- ─── 服务 Tab ─── -->
      <div v-if="activeTab === 'services'">
        <div class="px-6 py-3 border-b border-slate-100 flex justify-end">
          <button @click="loadServices" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
            <i class="fas fa-rotate"></i>刷新
          </button>
        </div>
        <div v-if="servicesLoading" class="flex flex-col items-center justify-center py-20">
          <div class="w-12 h-12 spinner mb-3"></div>
          <p class="text-slate-500">加载中...</p>
        </div>
        <div v-else-if="services.length > 0" class="overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="w-1/4 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">服务名</th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">镜像</th>
                <th class="w-24 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">模式</th>
                <th class="w-24 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">副本</th>
                <th class="w-36 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">端口</th>
                <th class="w-36 px-4 py-3 text-center text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="svc in services" :key="svc.id" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3">
                  <div class="flex items-center gap-2">
                    <div class="w-8 h-8 rounded-lg bg-emerald-400 flex items-center justify-center">
                      <i class="fas fa-cubes text-white text-sm"></i>
                    </div>
                    <span class="font-medium text-slate-800">{{ svc.name }}</span>
                  </div>
                </td>
                <td class="px-4 py-3"><code class="text-xs bg-slate-100 px-2 py-1 rounded">{{ svc.image }}</code></td>
                <td class="px-4 py-3"><span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-600 capitalize">{{ svc.mode }}</span></td>
                <td class="px-4 py-3 text-sm text-slate-600">
                  <span class="text-emerald-600 font-medium">{{ svc.runningTasks }}</span>
                  <span v-if="svc.mode === 'replicated'" class="text-slate-400"> / {{ svc.replicas ?? '?' }}</span>
                </td>
                <td class="px-4 py-3 font-mono text-xs text-slate-500">
                  <template v-if="svc.ports && svc.ports.length">
                    <div v-for="p in svc.ports" :key="p.published">{{ p.published }}:{{ p.target }}/{{ p.protocol }}</div>
                  </template>
                  <template v-else>-</template>
                </td>
                <td class="px-4 py-3">
                  <div class="flex justify-center items-center gap-0.5">
                    <button v-if="svc.mode === 'replicated'" @click="openScaleModal(svc)" class="btn-icon text-blue-600 hover:bg-blue-50" title="扩缩容"><i class="fas fa-up-right-and-down-left-from-center text-xs"></i></button>
                    <button @click="handleServiceRemove(svc)" class="btn-icon text-red-600 hover:bg-red-50" title="删除"><i class="fas fa-trash text-xs"></i></button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="flex flex-col items-center justify-center py-20">
          <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
            <i class="fas fa-cubes text-4xl text-slate-300"></i>
          </div>
          <p class="text-slate-600 font-medium mb-1">暂无服务</p>
        </div>
      </div>

      <!-- ─── 任务 Tab ─── -->
      <div v-if="activeTab === 'tasks'">
        <div class="px-6 py-3 border-b border-slate-100 flex items-center justify-between">
          <div class="flex items-center gap-2">
            <label class="text-xs text-slate-500">过滤服务</label>
            <select v-model="filterServiceID" @change="loadTasks" class="w-44 px-3 py-1.5 bg-white border border-slate-200 rounded-lg text-xs text-slate-700 hover:border-slate-300 focus:outline-none">
              <option value="">全部服务</option>
              <option v-for="s in services" :key="s.id" :value="s.id">{{ s.name }}</option>
            </select>
          </div>
          <button @click="loadTasks" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
            <i class="fas fa-rotate"></i>刷新
          </button>
        </div>
        <div v-if="tasksLoading" class="flex flex-col items-center justify-center py-20">
          <div class="w-12 h-12 spinner mb-3"></div>
          <p class="text-slate-500">加载中...</p>
        </div>
        <div v-else-if="tasks.length > 0" class="overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">服务名</th>
                <th class="w-16 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">Slot</th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">镜像</th>
                <th class="w-28 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">状态</th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">消息</th>
                <th class="w-40 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">更新时间</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="t in tasks" :key="t.id" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3 font-medium text-slate-800">{{ t.serviceName || t.serviceID }}</td>
                <td class="px-4 py-3 text-sm text-slate-500">{{ t.slot || '-' }}</td>
                <td class="px-4 py-3"><code class="text-xs bg-slate-100 px-2 py-1 rounded">{{ t.image }}</code></td>
                <td class="px-4 py-3">
                  <span :class="taskStateClass(t.state)" class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize">{{ t.state }}</span>
                </td>
                <td class="px-4 py-3 text-xs text-slate-500 max-w-xs truncate" :title="t.err || t.message">{{ t.err || t.message || '-' }}</td>
                <td class="px-4 py-3 text-xs text-slate-500 whitespace-nowrap">{{ t.updatedAt }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="flex flex-col items-center justify-center py-20">
          <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
            <i class="fas fa-tasks text-4xl text-slate-300"></i>
          </div>
          <p class="text-slate-600 font-medium mb-1">暂无任务</p>
        </div>
      </div>

    </div>

    <!-- 扩缩容模态框 -->
    <BaseModal
      v-model="scaleOpen"
      title="服务扩缩容"
      :loading="scaleLoading"
      show-footer
      confirm-text="确认扩缩容"
      @confirm="handleScale"
    >
      <div v-if="scaleService" class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">服务</label>
          <div class="px-3 py-2 bg-slate-50 rounded-lg text-sm text-slate-600">{{ scaleService.name }}</div>
        </div>
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">目标副本数</label>
          <input type="number" v-model.number="scaleReplicas" min="0" max="100" class="input" />
          <p class="mt-1 text-xs text-slate-400">当前运行中副本：{{ scaleService.runningTasks }} / {{ scaleService.replicas }}</p>
        </div>
      </div>
    </BaseModal>
  </div>
</template>
