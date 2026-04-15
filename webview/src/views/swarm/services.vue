<script setup>
import { inject, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

import ImageSelect from '@/component/docker/image-select.vue'
import BaseModal from '@/component/modal.vue'

const actions = inject(APP_ACTIONS_KEY)
const router = useRouter()

const services = ref([])
const servicesLoading = ref(false)

// 扩缩容
const scaleOpen = ref(false)
const scaleService = ref(null)
const scaleReplicas = ref(1)
const scaleLoading = ref(false)

// 创建服务
const createOpen = ref(false)
const createLoading = ref(false)
const createForm = ref({
  name: '', image: '', mode: 'replicated', replicas: 1,
  env: '', args: '', network: '', ports: '', mounts: ''
})

// 镜像和网络列表（创建服务时加载）
const createImages = ref([])
const createNetworks = ref([])
const showCreateAdvanced = ref(false)

const loadCreateResources = async () => {
  try {
    const [imgRes, netRes] = await Promise.all([
      api.listImages(false),
      api.listNetworks()
    ])
    createImages.value = imgRes.payload || []
    // 只保留 overlay 和 host 网络（适合 Swarm 服务）
    createNetworks.value = (netRes.payload || []).filter(n =>
      n.driver === 'overlay' || n.driver === 'host' || n.driver === 'bridge'
    )
  } catch (e) {
    // 静默失败
  }
}

// 日志（已移至服务详情页，保留变量避免模板报错）
const logsOpen = ref(false)
const logsLoading = ref(false)
const logsContent = ref('')
const logsService = ref(null)
const logsTail = ref('200')

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

const handleRedeploy = (svc) => {
  actions.showConfirm({
    title: '强制重部署',
    message: `重新拉取并部署服务 <strong class="text-slate-900">${svc.name}</strong>，正在运行的副本会滚动更新。`,
    icon: 'fa-rotate',
    iconColor: 'blue',
    confirmText: '确认重部署',
    onConfirm: async () => {
      await api.swarmRedeployService(svc.id)
      actions.showNotification('success', '已触发强制重部署')
      loadServices()
    }
  })
}

const openCreateModal = () => {
  createForm.value = { name: '', image: '', mode: 'replicated', replicas: 1, env: '', args: '', network: '', ports: '', mounts: '' }
  showCreateAdvanced.value = false
  createOpen.value = true
  loadCreateResources()
}

const handleCreate = async () => {
  createLoading.value = true
  try {
    const parseLines = (s) => s.split('\n').map(l => l.trim()).filter(Boolean)
    const parsePorts = (s) => parseLines(s).map(l => {
      const [pub, rest] = l.split(':')
      const [tgt, proto] = (rest || pub).split('/')
      return { published: parseInt(pub) || 0, target: parseInt(tgt), protocol: proto || 'tcp' }
    })
    const parseMounts = (s) => parseLines(s).map(l => {
      const parts = l.split(':')
      return { type: 'bind', source: parts[0], target: parts[1] || parts[0] }
    })

    await api.swarmCreateService({
      name: createForm.value.name,
      image: createForm.value.image,
      mode: createForm.value.mode,
      replicas: createForm.value.replicas,
      env: parseLines(createForm.value.env),
      args: parseLines(createForm.value.args),
      networks: createForm.value.network ? [createForm.value.network] : [],
      ports: parsePorts(createForm.value.ports),
      mounts: parseMounts(createForm.value.mounts),
    })
    actions.showNotification('success', '服务创建成功')
    createOpen.value = false
    loadServices()
  } catch (e) {
    actions.showNotification('error', '服务创建失败')
  }
  createLoading.value = false
}

const openLogsModal = async (svc) => {
  logsService.value = svc
  logsContent.value = ''
  logsOpen.value = true
  await fetchLogs(svc.id)
}

const fetchLogs = async (id) => {
  logsLoading.value = true
  try {
    const res = await api.swarmServiceLogs(id, logsTail.value)
    logsContent.value = res.payload?.logs || '（暂无日志）'
  } catch (e) {
    logsContent.value = '获取日志失败'
  }
  logsLoading.value = false
}

onMounted(() => loadServices())
</script>

<template>
  <div>
    <div class="card mb-4">
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-6 py-3 flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="w-9 h-9 rounded-lg bg-emerald-500 flex items-center justify-center">
            <i class="fas fa-cubes text-white"></i>
          </div>
          <div>
            <h1 class="text-lg font-semibold text-slate-800">服务管理</h1>
            <p class="text-xs text-slate-500">管理 Swarm 服务</p>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <button @click="loadServices" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
            <i class="fas fa-rotate"></i>刷新
          </button>
          <button @click="openCreateModal" class="px-3 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
            <i class="fas fa-plus"></i>创建服务
          </button>
        </div>
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
<th class="w-44 px-4 py-3 text-right text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
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
<div class="flex justify-end items-center gap-0.5">
                  <button @click="router.push(`/swarm/service/${svc.id}`)" class="btn-icon text-slate-600 hover:bg-slate-100"  title="查看详情"><i class="fas fa-circle-info text-xs"></i></button>
                  <button @click="handleRedeploy(svc)"     class="btn-icon text-blue-600 hover:bg-blue-50"     title="强制重部署"><i class="fas fa-rotate text-xs"></i></button>
                  <button v-if="svc.mode === 'replicated'" @click="openScaleModal(svc)" class="btn-icon text-indigo-600 hover:bg-indigo-50" title="扩缩容"><i class="fas fa-up-right-and-down-left-from-center text-xs"></i></button>
                  <button @click="handleServiceRemove(svc)" class="btn-icon text-red-600 hover:bg-red-50"      title="删除"><i class="fas fa-trash text-xs"></i></button>
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

    <!-- 扩缩容模态框 -->
    <BaseModal v-model="scaleOpen" title="服务扩缩容" :loading="scaleLoading" show-footer confirm-text="确认扩缩容" @confirm="handleScale">
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

    <!-- 创建服务模态框 -->
    <BaseModal v-model="createOpen" title="创建服务" :loading="createLoading" show-footer confirm-text="创建" @confirm="handleCreate">
      <form @submit.prevent="handleCreate" class="space-y-4">
        <!-- 基础设置 -->
        <div class="grid grid-cols-2 gap-3">
          <div class="col-span-2">
            <label class="block text-sm font-medium text-slate-700 mb-2">镜像 <span class="text-red-500">*</span></label>
            <ImageSelect v-model="createForm.image" :images="createImages" placeholder="选择或输入镜像名称" />
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">服务名 <span class="text-red-500">*</span></label>
            <input v-model="createForm.name" type="text" placeholder="my-service" class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">网络</label>
            <select v-model="createForm.network" class="input">
              <option value="">不指定</option>
              <option v-for="net in createNetworks" :key="net.id" :value="net.name">
                {{ net.name }} ({{ net.driver }})
              </option>
            </select>
          </div>
        </div>

        <!-- 端口映射 -->
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">端口映射</label>
          <textarea v-model="createForm.ports" rows="2" placeholder="8080:80/tcp" class="input font-mono text-sm"></textarea>
          <p class="mt-1 text-xs text-slate-400">每行一条，格式：宿主端口:容器端口/协议</p>
        </div>

        <!-- 目录挂载 -->
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">目录挂载</label>
          <textarea v-model="createForm.mounts" rows="2" placeholder="/data:/app/data" class="input font-mono text-sm"></textarea>
          <p class="mt-1 text-xs text-slate-400">每行一条，格式：宿主路径:容器路径</p>
        </div>

        <!-- 环境变量 -->
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">环境变量</label>
          <textarea v-model="createForm.env" rows="2" placeholder="KEY=value" class="input font-mono text-sm"></textarea>
        </div>

        <!-- 高级选项 -->
        <div class="border-t border-slate-200 pt-4">
          <button type="button" @click="showCreateAdvanced = !showCreateAdvanced" class="flex items-center gap-2 text-sm text-slate-600 hover:text-slate-800">
            <i :class="['fas fa-chevron-down text-xs transition-transform', showCreateAdvanced ? 'rotate-180' : '']"></i>
            高级选项
          </button>
          <div v-if="showCreateAdvanced" class="mt-4 space-y-4">
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="block text-sm font-medium text-slate-700 mb-2">模式</label>
                <select v-model="createForm.mode" class="input">
                  <option value="replicated">Replicated</option>
                  <option value="global">Global</option>
                </select>
              </div>
              <div v-if="createForm.mode === 'replicated'">
                <label class="block text-sm font-medium text-slate-700 mb-2">副本数</label>
                <input v-model.number="createForm.replicas" type="number" min="1" class="input" />
              </div>
            </div>
            <div>
              <label class="block text-sm font-medium text-slate-700 mb-2">启动参数</label>
              <input v-model="createForm.args" type="text" placeholder="覆盖默认启动参数" class="input font-mono text-sm" />
            </div>
          </div>
        </div>
      </form>
    </BaseModal>

    <!-- 日志模态框 -->
    <BaseModal v-model="logsOpen" :title="`服务日志 - ${logsService?.name || ''}`" :loading="logsLoading" show-footer confirm-text="刷新" @confirm="fetchLogs(logsService?.id)">
      <div class="space-y-3">
        <div class="flex items-center gap-2">
          <label class="text-xs text-slate-500 flex-shrink-0">最近行数</label>
          <select v-model="logsTail" @change="fetchLogs(logsService?.id)" class="w-24 px-2 py-1 bg-white border border-slate-200 rounded-lg text-xs text-slate-700">
            <option value="50">50</option>
            <option value="100">100</option>
            <option value="200">200</option>
            <option value="500">500</option>
          </select>
        </div>
        <div v-if="logsLoading" class="flex items-center justify-center py-10">
          <div class="w-8 h-8 spinner"></div>
        </div>
        <pre v-else class="bg-slate-900 text-slate-100 rounded-xl p-4 text-xs font-mono overflow-auto max-h-[420px] whitespace-pre-wrap break-all">{{ logsContent }}</pre>
      </div>
    </BaseModal>
    </div>
  </div>
</template>
