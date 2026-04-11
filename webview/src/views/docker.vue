<script setup>
import { inject, onMounted, ref } from 'vue'

import { formatFileSize, formatTime } from '@/helper/utils.js'
import api from '@/service/api.js'
import { APP_ACTIONS_KEY, APP_STATE_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const state = inject(APP_STATE_KEY)
const actions = inject(APP_ACTIONS_KEY)

// 当前标签页
const activeTab = ref('containers')
const tabs = [
  { key: 'containers', label: '容器', icon: 'fa-box' },
  { key: 'images', label: '镜像', icon: 'fa-compact-disc' },
  { key: 'networks', label: '网络', icon: 'fa-network-wired' },
  { key: 'volumes', label: '卷', icon: 'fa-hdd' },
]

// 概览数据
const dockerInfo = ref(null)
const loading = ref(false)

// 数据列表
const containers = ref([])
const images = ref([])
const networks = ref([])
const volumes = ref([])

const showAll = ref(false)
const showAllImages = ref(false)

// ==================== 模态框状态 ====================
const modalOpen = ref(false)
const modalTitle = ref('')
const modalLoading = ref(false)
const formData = ref({})
const logContent = ref('')
const selectedContainer = ref(null)
const showImageDropdown = ref(false) // 镜像下拉菜单显示状态
const imageInputRef = ref(null) // 镜像输入框引用
const dropdownPosition = ref({ top: 0, left: 0, width: 0 }) // 下拉菜单位置

// ==================== 加载数据 ====================
const loadInfo = async () => {
  try {
    const data = await api.dockerInfo()
    dockerInfo.value = data.payload
  } catch (e) {
    console.error('Failed to load Docker info:', e)
  }
}

const loadContainers = async () => {
  loading.value = true
  try {
    const data = await api.listContainers(showAll.value)
    containers.value = data.payload || []
  } catch (e) {
    console.error('Failed to load containers:', e)
  }
  loading.value = false
}

const loadImages = async () => {
  try {
    const data = await api.listImages(showAllImages.value)
    images.value = data.payload || []
  } catch (e) {
    console.error('Failed to load images:', e)
  }
}

const loadNetworks = async () => {
  try {
    const data = await api.listNetworks()
    networks.value = data.payload || []
  } catch (e) {
    console.error('Failed to load networks:', e)
  }
}

const loadVolumes = async () => {
  try {
    const data = await api.listVolumes()
    volumes.value = data.payload || []
  } catch (e) {
    console.error('Failed to load volumes:', e)
  }
}

const refreshCurrentTab = () => {
  switch (activeTab.value) {
    case 'containers': loadContainers(); break
    case 'images': loadImages(); break
    case 'networks': loadNetworks(); break
    case 'volumes': loadVolumes(); break
  }
}

// ==================== 容器操作 ====================
const handleContainerAction = async (container, action) => {
  if (!confirm(`确定要${getActionName(action)}容器 "${container.name}" 吗？`)) return
  
  try {
    await api.containerAction(container.id, action)
    actions.showNotification('success', `容器 ${getActionName(action)} 成功`)
    loadContainers()
    loadInfo()
  } catch (e) {
    // error handled by interceptor
  }
}

const getActionName = (action) => {
  return { start: '启动', stop: '停止', restart: '重启', remove: '删除', pause: '暂停', unpause: '恢复' }[action] || action
}

const getStateBadgeClass = (state) => {
  switch (state) {
    case 'running': return 'bg-emerald-100 text-emerald-700 border-emerald-200'
    case 'exited': case 'dead': return 'bg-red-100 text-red-700 border-red-200'
    case 'paused': return 'bg-amber-100 text-amber-700 border-amber-200'
    default: return 'bg-slate-100 text-slate-700 border-slate-200'
  }
}

// 查看日志
const viewLogs = async (container) => {
  selectedContainer.value = container
  logContent.value = ''
  modalTitle.value = `日志: ${container.name}`
  modalOpen.value = true
  modalLoading.value = true

  try {
    const data = await api.containerLogs(container.id, '200')
    logContent.value = (data.payload.logs || []).join('\n')
  } catch (e) {
    logContent.value = '加载日志失败'
  }
  modalLoading.value = false
}

// ==================== 镜像操作 ====================
const pullImageModal = () => {
  formData.value = { image: '', tag: '' }
  modalTitle.value = '拉取镜像'
  modalOpen.value = true
  logContent.value = ''
}

const handlePullImage = async () => {
  if (!formData.value.image.trim()) return
  modalLoading.value = true
  try {
    await api.pullImage(formData.value.image, formData.value.tag)
    actions.showNotification('success', '镜像拉取成功')
    modalOpen.value = false
    loadImages()
    loadInfo()
  } catch (e) {
    // error handled by interceptor
  }
  modalLoading.value = false
}

const handleImageAction = async (image, action) => {
  if (!confirm(`确定要删除镜像 "${image.repoTags[0] || image.id}" 吗？`)) return
  try {
    await api.imageAction(image.id, action)
    actions.showNotification('success', '镜像删除成功')
    loadImages()
    loadInfo()
  } catch (e) {
    // error handled
  }
}

// ==================== 网络操作 ====================
const createNetworkModal = () => {
  formData.value = { name: '', driver: 'bridge', subnet: '' }
  modalTitle.value = '创建网络'
  modalOpen.value = true
  logContent.value = ''
}

const handleCreateNetwork = async () => {
  if (!formData.value.name.trim()) return
  modalLoading.value = true
  try {
    await api.createNetwork(formData.value)
    actions.showNotification('success', '网络创建成功')
    modalOpen.value = false
    loadNetworks()
    loadInfo()
  } catch (e) {}
  modalLoading.value = false
}

const handleNetworkAction = async (net, action) => {
  if (!confirm(`确定要删除网络 "${net.name}" 吗？`)) return
  try {
    await api.networkAction(net.id, action)
    actions.showNotification('success', '网络删除成功')
    loadNetworks()
    loadInfo()
  } catch (e) {}
}

// ==================== 卷操作 ====================
const createVolumeModal = () => {
  formData.value = { name: '', driver: 'local' }
  modalTitle.value = '创建卷'
  modalOpen.value = true
  logContent.value = ''
}

const handleCreateVolume = async () => {
  if (!formData.value.name.trim()) return
  modalLoading.value = true
  try {
    await api.createVolume(formData.value)
    actions.showNotification('success', '卷创建成功')
    modalOpen.value = false
    loadVolumes()
    loadInfo()
  } catch (e) {}
  modalLoading.value = false
}

const handleVolumeAction = async (vol, action) => {
  if (!confirm(`确定要删除卷 "${vol.name}" 吗？`)) return
  try {
    await api.volumeAction(vol.name, action)
    actions.showNotification('success', '卷删除成功')
    loadVolumes()
    loadInfo()
  } catch (e) {}
}


// 打开镜像下拉菜单
const openImageDropdown = () => {
  if (imageInputRef.value) {
    const rect = imageInputRef.value.getBoundingClientRect()
    dropdownPosition.value = {
      top: rect.bottom + window.scrollY,
      left: rect.left + window.scrollX,
      width: rect.width
    }
  }
  showImageDropdown.value = true
}

// 选择镜像
const selectImage = (imageName) => {
  formData.value.image = imageName
  showImageDropdown.value = false
}

// 创建容器弹窗
const createContainerModal = () => {
  formData.value = { image: '', name: '', env: [], ports: {}, cmd: '' }
  modalTitle.value = '创建容器'
  modalOpen.value = true
  logContent.value = ''
}

const handleCreateContainer = async () => {
  if (!formData.value.image.trim()) return
  modalLoading.value = true
  try {
    // 构建请求数据
    const data = {
      image: formData.value.image,
      name: formData.value.name || undefined,
      env: formData.value.envStr ? formData.value.envStr.split('\n').filter(e => e.trim()) : [],
      ports: {},
      remove: formData.value.remove || false,
    }
    // 处理端口映射
    if (formData.value.hostPort && formData.value.containerPort) {
      data.ports[formData.value.hostPort] = formData.value.containerPort
    }
    // 处理启动命令：将字符串按空格分割为数组
    if (formData.value.cmd && formData.value.cmd.trim()) {
      data.cmd = formData.value.cmd.trim().split(/\s+/)
    }
    await api.createContainer(data)
    actions.showNotification('success', '容器创建成功')
    modalOpen.value = false
    loadContainers()
    loadInfo()
  } catch (e) {}
  modalLoading.value = false
}

// 切换 tab 时加载数据
const switchTab = (tab) => {
  activeTab.value = tab
  refreshCurrentTab()
}

onMounted(() => {
  loadInfo()
  loadContainers()
})
</script>

<template>
  <div class="h-[calc(100vh-100px)]">
    <div class="h-full card flex flex-col overflow-hidden">
      <!-- Toolbar -->
      <div class="bg-slate-50 border-b border-slate-200 px-6 py-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center space-x-3">
            <div class="w-10 h-10 rounded-xl bg-blue-500 flex items-center justify-center">
              <i class="fab fa-docker text-white text-lg"></i>
            </div>
            <div>
              <h3 class="font-semibold text-slate-800">Docker 管理</h3>
              <p class="text-xs text-slate-500">管理容器、镜像、网络和数据卷</p>
            </div>
          </div>
          <div v-if="dockerInfo" class="flex items-center gap-4 text-sm">
            <div class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-emerald-50 text-emerald-700">
              <i class="fas fa-play-circle text-xs"></i>
              <span class="font-medium">{{ dockerInfo.containersRunning }}</span>
              <span class="text-emerald-600">运行中</span>
            </div>
            <div class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-slate-100 text-slate-600">
              <i class="fas fa-pause-circle text-xs"></i>
              <span class="font-medium">{{ dockerInfo.containersStopped }}</span>
              <span>已停止</span>
            </div>
            <div class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-blue-50 text-blue-700">
              <i class="fas fa-compact-disc text-xs"></i>
              <span class="font-medium">{{ dockerInfo.imagesTotal }}</span>
              <span>镜像</span>
            </div>
            <div class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-purple-50 text-purple-700">
              <i class="fas fa-hdd text-xs"></i>
              <span class="font-medium">{{ dockerInfo.volumesTotal }}</span>
              <span>卷</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Tab Navigation -->
      <div class="border-b border-slate-200 px-6 bg-white">
        <div class="flex items-center space-x-1">
          <button 
            v-for="tab in tabs" :key="tab.key"
            @click="switchTab(tab.key)"
            :class="[
              'px-4 py-3 text-sm font-medium border-b-2 transition-all duration-200 flex items-center gap-2',
              activeTab === tab.key 
                ? 'border-primary-500 text-primary-600 bg-primary-50/50' 
                : 'border-transparent text-slate-500 hover:text-slate-700 hover:bg-slate-50'
            ]"
          >
            <i :class="['fas', tab.icon]"></i>
            {{ tab.label }}
          </button>
        </div>
      </div>

      <!-- Content Area -->
      <div class="flex-1 overflow-auto p-6">

        <!-- ========== 容器面板 ========== -->
        <div v-if="activeTab === 'containers'" class="space-y-4">
          <!-- 操作栏 -->
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <button @click="createContainerModal()" class="btn-primary text-sm py-2">
                <i class="fas fa-plus mr-1.5"></i>创建容器
              </button>
              <label class="flex items-center gap-2 text-sm text-slate-600 cursor-pointer select-none">
                <input type="checkbox" v-model="showAll" @change="loadContainers()" class="rounded border-slate-300">
                显示已停止的容器
              </label>
            </div>
            <button @click="refreshCurrentTab()" class="btn-secondary text-sm py-2">
              <i class="fas fa-rotate mr-1.5"></i>刷新
            </button>
          </div>

          <!-- Loading -->
          <div v-if="loading" class="flex flex-col items-center justify-center py-20">
            <div class="w-12 h-12 spinner mb-3"></div>
            <p class="text-slate-500">加载中...</p>
          </div>

          <!-- Container Table -->
          <div v-else-if="containers.length > 0" class="overflow-x-auto rounded-xl border border-slate-200">
            <table class="w-full">
              <thead class="bg-slate-50">
                <tr>
                  <th class="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase">名称</th>
                  <th class="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase">镜像</th>
                  <th class="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase w-24">状态</th>
                  <th class="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase w-40">端口</th>
                  <th class="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase w-32">创建时间</th>
                  <th class="text-right px-4 py-3 text-xs font-semibold text-slate-500 uppercase w-56">操作</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-100">
                <tr v-for="ct in containers" :key="ct.id" class="hover:bg-slate-50/50 transition-colors">
                  <td class="px-4 py-3">
                    <div class="flex items-center gap-2">
                      <div :class="['w-8 h-8 rounded-lg flex items-center justify-center', ct.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
                        <i class="fas fa-box text-white text-sm"></i>
                      </div>
                      <span class="font-medium text-slate-800">{{ ct.name || ct.id }}</span>
                    </div>
                  </td>
                  <td class="px-4 py-3"><code class="text-xs bg-slate-100 px-2 py-1 rounded">{{ ct.image }}</code></td>
                  <td class="px-4 py-3">
                    <span :class="['inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium border', getStateBadgeClass(ct.state)]">
                      {{ ct.status }}
                    </span>
                  </td>
                  <td class="px-4 py-3 text-sm text-slate-600 font-mono">{{ ct.ports || '-' }}</td>
                  <td class="px-4 py-3 text-sm text-slate-500 whitespace-nowrap">{{ formatTime(new Date(ct.created * 1000).toISOString()) }}</td>
                  <td class="px-4 py-3">
                    <div class="flex items-center justify-end gap-1">
                      <button v-if="ct.state !== 'running'" @click="handleContainerAction(ct, 'start')" class="btn-icon text-emerald-600 hover:bg-emerald-50" title="启动">
                        <i class="fas fa-play text-xs"></i>
                      </button>
                      <button v-if="ct.state === 'running'" @click="handleContainerAction(ct, 'stop')" class="btn-icon text-amber-600 hover:bg-amber-50" title="停止">
                        <i class="fas fa-stop text-xs"></i>
                      </button>
                      <button v-if="ct.state === 'running'" @click="handleContainerAction(ct, 'restart')" class="btn-icon text-blue-600 hover:bg-blue-50" title="重启">
                        <i class="fas fa-redo text-xs"></i>
                      </button>
                      <button v-if="ct.state === 'running'" @click="viewLogs(ct)" class="btn-icon text-cyan-600 hover:bg-cyan-50" title="日志">
                        <i class="fas fa-file-alt text-xs"></i>
                      </button>
                      <button @click="handleContainerAction(ct, 'remove')" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
                        <i class="fas fa-trash text-xs"></i>
                      </button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <div v-else class="flex flex-col items-center justify-center py-20">
            <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
              <i class="fab fa-docker text-4xl text-slate-300"></i>
            </div>
            <p class="text-slate-600 font-medium mb-1">暂无容器</p>
            <p class="text-sm text-slate-400">点击「创建容器」开始使用 Docker</p>
          </div>
        </div>

        <!-- ========== 镜像面板 ========== -->
        <div v-if="activeTab === 'images'" class="space-y-4">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <button @click="pullImageModal()" class="btn-primary text-sm py-2">
                <i class="fas fa-download mr-1.5"></i>拉取镜像
              </button>
              <label class="flex items-center gap-2 text-sm text-slate-600 cursor-pointer select-none">
                <input type="checkbox" v-model="showAllImages" @change="loadImages()" class="rounded border-slate-300">
                显示中间层镜像
              </label>
            </div>
            <button @click="refreshCurrentTab()" class="btn-secondary text-sm py-2">
              <i class="fas fa-rotate mr-1.5"></i>刷新
            </button>
          </div>

          <div v-if="images.length > 0" class="overflow-x-auto rounded-xl border border-slate-200">
            <table class="w-full">
              <thead class="bg-slate-50">
                <tr>
                  <th class="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase">镜像</th>
                  <th class="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase">ID</th>
                  <th class="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase w-32">大小</th>
                  <th class="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase w-36">创建时间</th>
                  <th class="text-right px-4 py-3 text-xs font-semibold text-slate-500 uppercase w-24">操作</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-100">
                <tr v-for="img in images" :key="img.id" class="hover:bg-slate-50/50 transition-colors">
                  <td class="px-4 py-3">
                    <div class="flex items-center gap-2">
                      <div class="w-8 h-8 rounded-lg bg-blue-400 flex items-center justify-center">
                        <i class="fas fa-compact-disc text-white text-sm"></i>
                      </div>
                      <span class="font-medium text-slate-800">{{ img.repoTags[0] || '<none>' }}</span>
                    </div>
                  </td>
                  <td class="px-4 py-3"><code class="text-xs text-slate-500 font-mono">{{ img.shortId }}</code></td>
                  <td class="px-4 py-3 text-sm text-slate-600">{{ formatFileSize(img.size) }}</td>
                  <td class="px-4 py-3 text-sm text-slate-500 whitespace-nowrap">{{ formatTime(new Date(img.created * 1000).toISOString()) }}</td>
                  <td class="px-4 py-3">
                    <div class="flex items-center justify-end">
                      <button @click="handleImageAction(img, 'remove')" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
                        <i class="fas fa-trash text-xs"></i>
                      </button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <div v-else class="flex flex-col items-center justify-center py-20">
            <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
              <i class="fas fa-compact-disc text-4xl text-slate-300"></i>
            </div>
            <p class="text-slate-600 font-medium mb-1">暂无镜像</p>
            <p class="text-sm text-slate-400">点击「拉取镜像」从 Registry 获取镜像</p>
          </div>
        </div>

        <!-- ========== 网络面板 ========== -->
        <div v-if="activeTab === 'networks'" class="space-y-4">
          <div class="flex items-center justify-between">
            <button @click="createNetworkModal()" class="btn-primary text-sm py-2">
              <i class="fas fa-plus mr-1.5"></i>创建网络
            </button>
            <button @click="refreshCurrentTab()" class="btn-secondary text-sm py-2">
              <i class="fas fa-rotate mr-1.5"></i>刷新
            </button>
          </div>

          <div v-if="networks.length > 0" class="overflow-x-auto rounded-xl border border-slate-200">
            <table class="w-full">
              <thead class="bg-slate-50">
                <tr>
                  <th class="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase">名称</th>
                  <th class="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase w-28">驱动</th>
                  <th class="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase w-44">子网</th>
                  <th class="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase w-24">范围</th>
                  <th class="text-right px-4 py-3 text-xs font-semibold text-slate-500 uppercase w-24">操作</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-100">
                <tr v-for="net in networks" :key="net.id" class="hover:bg-slate-50/50 transition-colors">
                  <td class="px-4 py-3">
                    <div class="flex items-center gap-2">
                      <div class="w-8 h-8 rounded-lg bg-purple-400 flex items-center justify-center">
                        <i class="fas fa-network-wired text-white text-sm"></i>
                      </div>
                      <span class="font-medium text-slate-800">{{ net.name }}</span>
                    </div>
                  </td>
                  <td class="px-4 py-3"><code class="text-xs bg-slate-100 px-2 py-1 rounded">{{ net.driver }}</code></td>
                  <td class="px-4 py-3 text-sm text-slate-600 font-mono">{{ net.subnet || '-' }}</td>
                  <td class="px-4 py-3 text-sm text-slate-500">{{ net.scope }}</td>
                  <td class="px-4 py-3">
                    <div class="flex items-center justify-end">
                      <button v-if="net.driver !== 'bridge' && net.driver !== 'host' && net.driver !== 'none'" @click="handleNetworkAction(net, 'remove')" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
                        <i class="fas fa-trash text-xs"></i>
                      </button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <div v-else class="flex flex-col items-center justify-center py-20">
            <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
              <i class="fas fa-network-wired text-4xl text-slate-300"></i>
            </div>
            <p class="text-slate-600 font-medium mb-1">暂无自定义网络</p>
            <p class="text-sm text-slate-400">点击「创建网络」添加自定义网络</p>
          </div>
        </div>

        <!-- ========== 卷面板 ========== -->
        <div v-if="activeTab === 'volumes'" class="space-y-4">
          <div class="flex items-center justify-between">
            <button @click="createVolumeModal()" class="btn-primary text-sm py-2">
              <i class="fas fa-plus mr-1.5"></i>创建卷
            </button>
            <button @click="refreshCurrentTab()" class="btn-secondary text-sm py-2">
              <i class="fas fa-rotate mr-1.5"></i>刷新
            </button>
          </div>

          <div v-if="volumes.length > 0" class="overflow-x-auto rounded-xl border border-slate-200">
            <table class="w-full">
              <thead class="bg-slate-50">
                <tr>
                  <th class="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase">名称</th>
                  <th class="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase w-24">驱动</th>
                  <th class="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase">挂载点</th>
                  <th class="text-left px-4 py-3 text-xs font-semibold text-slate-500 uppercase w-40">创建时间</th>
                  <th class="text-right px-4 py-3 text-xs font-semibold text-slate-500 uppercase w-24">操作</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-100">
                <tr v-for="vol in volumes" :key="vol.name" class="hover:bg-slate-50/50 transition-colors">
                  <td class="px-4 py-3">
                    <div class="flex items-center gap-2">
                      <div class="w-8 h-8 rounded-lg bg-indigo-400 flex items-center justify-center">
                        <i class="fas fa-hdd text-white text-sm"></i>
                      </div>
                      <span class="font-medium text-slate-800">{{ vol.name }}</span>
                    </div>
                  </td>
                  <td class="px-4 py-3"><code class="text-xs bg-slate-100 px-2 py-1 rounded">{{ vol.driver }}</code></td>
                  <td class="px-4 py-3 text-sm text-slate-600 font-mono truncate max-w-xs">{{ vol.mountpoint || '-' }}</td>
                  <td class="px-4 py-3 text-sm text-slate-500 whitespace-nowrap">{{ vol.createdAt ? new Date(vol.createdAt).toLocaleString('zh-CN') : '-' }}</td>
                  <td class="px-4 py-3">
                    <div class="flex items-center justify-end">
                      <button @click="handleVolumeAction(vol, 'remove')" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
                        <i class="fas fa-trash text-xs"></i>
                      </button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <div v-else class="flex flex-col items-center justify-center py-20">
            <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
              <i class="fas fa-hdd text-4xl text-slate-300"></i>
            </div>
            <p class="text-slate-600 font-medium mb-1">暂无数据卷</p>
            <p class="text-sm text-slate-400">点击「创建卷」添加持久化存储</p>
          </div>
        </div>

      </div>
    </div>

    <!-- 通用模态框 -->
    <BaseModal 
      v-model="modalOpen" 
      :title="modalTitle" 
      :loading="modalLoading"
      :show-footer="!!logContent || ['拉取镜像', '创建网络', '创建卷', '创建容器'].includes(modalTitle)"
      @confirm="() => {
        if (modalTitle === '拉取镜像') handlePullImage()
        else if (modalTitle === '创建网络') handleCreateNetwork()
        else if (modalTitle === '创建卷') handleCreateVolume()
        else if (modalTitle === '创建容器') handleCreateContainer()
      }"
    >
      <!-- 日志查看 -->
      <template v-if="logContent && !['拉取镜像','创建网络','创建卷','创建容器'].includes(modalTitle)">
        <pre class="bg-slate-900 text-green-400 p-4 rounded-xl overflow-auto max-h-96 text-sm font-mono whitespace-pre-wrap">{{ logContent }}</pre>
      </template>

      <!-- 拉取镜像表单 -->
      <template v-else-if="modalTitle === '拉取镜像'">
        <form @submit.prevent="handlePullImage" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">镜像名称</label>
            <input type="text" v-model="formData.image" placeholder="例如: nginx, redis:alpine, ubuntu:22.04" required class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">Tag（可选）</label>
            <input type="text" v-model="formData.tag" placeholder="默认 latest" class="input" />
          </div>
        </form>
      </template>

      <!-- 创建网络表单 -->
      <template v-else-if="modalTitle === '创建网络'">
        <form @submit.prevent="handleCreateNetwork" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">网络名称</label>
            <input type="text" v-model="formData.name" placeholder="例如: my-network" required class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">驱动类型</label>
            <select v-model="formData.driver" class="input">
              <option value="bridge">bridge (桥接)</option>
              <option value="host">host (主机)</option>
              <option value="overlay">overlay (覆盖)</option>
              <option value="macvlan">macvlan</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">子网 CIDR（可选）</label>
            <input type="text" v-model="formData.subnet" placeholder="例如: 172.20.0.0/16" class="input" />
          </div>
        </form>
      </template>

      <!-- 创建卷表单 -->
      <template v-else-if="modalTitle === '创建卷'">
        <form @submit.prevent="handleCreateVolume" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">卷名称</label>
            <input type="text" v-model="formData.name" placeholder="例如: my-data" required class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">驱动类型</label>
            <select v-model="formData.driver" class="input">
              <option value="local">local (本地)</option>
            </select>
          </div>
        </form>
      </template>

      <!-- 创建容器表单 -->
      <template v-else-if="modalTitle === '创建容器'">
        <form @submit.prevent="handleCreateContainer" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">镜像 <span class="text-red-500">*</span></label>
            <div class="relative" ref="imageInputRef">
              <input 
                type="text" 
                v-model="formData.image" 
                @focus="openImageDropdown"
                @blur="showImageDropdown = false"
                placeholder="选择或输入镜像名称" 
                required 
                class="input pr-10" 
              />
              <div class="absolute inset-y-0 right-0 flex items-center pr-3 pointer-events-none">
                <i class="fas fa-chevron-down text-slate-400 text-sm"></i>
              </div>
            </div>
            <p class="mt-1 text-xs text-slate-400">可从下拉列表选择已有镜像，或手动输入新镜像</p>
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">容器名称（可选）</label>
            <input type="text" v-model="formData.name" placeholder="例如: my-web-server" class="input" />
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">端口映射（可选）</label>
            <div class="grid grid-cols-2 gap-3">
              <input type="text" v-model="formData.containerPort" placeholder="容器端口 (如 80)" class="input text-sm" />
              <input type="text" v-model="formData.hostPort" placeholder="主机端口 (如 8080)" class="input text-sm" />
            </div>
            <p class="mt-1 text-xs text-slate-400">将主机端口映射到容器端口</p>
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">环境变量（可选，每行一个）</label>
            <textarea v-model="formData.envStr" rows="3" placeholder="MYSQL_ROOT_PASSWORD=secret&#10;APP_ENV=production" class="input font-mono text-sm"></textarea>
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">启动命令（可选）</label>
            <input type="text" v-model="formData.cmd" placeholder="例如: nginx -g 'daemon off;'" class="input font-mono text-sm" />
            <p class="mt-1 text-xs text-slate-400">覆盖镜像默认的启动命令</p>
          </div>
          <div class="flex items-center gap-2">
            <input type="checkbox" id="autoRemove" v-model="formData.remove" class="rounded border-slate-300" />
            <label for="autoRemove" class="text-sm text-slate-600">容器停止后自动删除</label>
          </div>
        </form>
      </template>
    </BaseModal>

    <!-- 镜像下拉列表 - 使用 Teleport 渲染到 body -->
    <Teleport to="body">
      <div 
        v-if="showImageDropdown && images.length > 0" 
        class="fixed bg-white border border-slate-200 rounded-lg shadow-xl max-h-60 overflow-auto"
        :style="{ top: dropdownPosition.top + 'px', left: dropdownPosition.left + 'px', width: dropdownPosition.width + 'px', zIndex: 9999 }"
        @mousedown.prevent
      >
        <div 
          v-for="img in images.filter(i => i.repoTags && i.repoTags[0] && (!formData.image || i.repoTags[0].toLowerCase().includes(formData.image.toLowerCase())))" 
          :key="img.id"
          @click="selectImage(img.repoTags[0])"
          class="px-3 py-2 hover:bg-blue-50 cursor-pointer flex items-center justify-between text-sm"
        >
          <span class="font-medium text-slate-700">{{ img.repoTags[0] }}</span>
          <span class="text-xs text-slate-400">{{ formatFileSize(img.size) }}</span>
        </div>
        <div v-if="!images.filter(i => i.repoTags && i.repoTags[0] && (!formData.image || i.repoTags[0].toLowerCase().includes(formData.image.toLowerCase()))).length" class="px-3 py-2 text-sm text-slate-400 text-center">
          无匹配镜像
        </div>
      </div>
    </Teleport>
  </div>
</template>
