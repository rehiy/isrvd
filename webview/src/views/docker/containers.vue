<script setup>
import { computed, inject, onMounted, ref } from 'vue'

import { useRouter } from 'vue-router'

import { formatTime } from '@/helper/utils.js'
import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

import CapSelect from '@/component/docker/cap-select.vue'
import ImageSelect from '@/component/docker/image-select.vue'
import BaseModal from '@/component/modal.vue'

const actions = inject(APP_ACTIONS_KEY)
const router = useRouter()

// 自己管理数据
const containers = ref([])
const images = ref([])
const networks = ref([])
const loading = ref(false)
const showAll = ref(false)

// 模态框状态
const modalOpen = ref(false)
const modalTitle = ref('')
const modalLoading = ref(false)
const formData = ref({})
const showAdvanced = ref(false)
const showSecurity = ref(false)
const isEditMode = ref(false)

// 批量操作状态
const selectedIds = ref([])
const batchMode = ref(false)

// 加载容器列表
const loadContainers = async () => {
  loading.value = true
  try {
const res = await api.listContainers(showAll.value)
    containers.value = res.payload || []
  } catch (e) {
    actions.showNotification('error', '加载容器列表失败')
  }
  loading.value = false
}

// 加载镜像列表（用于创建容器时选择镜像）
const loadImages = async () => {
  try {
const res = await api.listImages(false)
    images.value = res.payload || []
  } catch (e) {
    // 静默失败
  }
}

// 加载网络列表（用于创建容器时选择网络）
const loadNetworks = async () => {
  try {
    const res = await api.listNetworks()
    networks.value = res.payload || []
  } catch (e) {
    // 静默失败
  }
}

// 操作配置
const actionConfigs = {
  start: { icon: 'fa-play', iconColor: 'emerald', title: '启动容器', confirmText: '启动' },
  stop: { icon: 'fa-stop', iconColor: 'amber', title: '停止容器', confirmText: '停止' },
  restart: { icon: 'fa-redo', iconColor: 'blue', title: '重启容器', confirmText: '重启' },
  remove: { icon: 'fa-trash', iconColor: 'red', title: '删除容器', confirmText: '删除', danger: true },
  pause: { icon: 'fa-pause', iconColor: 'amber', title: '暂停容器', confirmText: '暂停' },
  unpause: { icon: 'fa-play', iconColor: 'emerald', title: '恢复容器', confirmText: '恢复' }
}

// 容器操作 - 显示确认模态框
const handleContainerAction = (container, action) => {
  const config = actionConfigs[action] || {}
  actions.showConfirm({
    title: config.title,
    message: `确定要${config.confirmText}容器 <strong class="text-slate-900">${container.name || container.id}</strong> 吗？`,
    icon: config.icon,
    iconColor: config.iconColor,
    confirmText: `确认${config.confirmText}`,
    danger: config.danger,
    onConfirm: async () => {
      await api.containerAction(container.id, action)
      actions.showNotification('success', `容器 ${config.confirmText} 成功`)
      loadContainers()
    }
  })
}

// 创建容器弹窗
// 重启策略选项
const restartOptions = [
  { value: 'always', label: '总是重启' },
  { value: 'unless-stopped', label: '除非手动停止' },
  { value: 'on-failure', label: '失败时重启' },
  { value: 'no', label: '不重启' }
]

// 网络模式选项（从加载的网络列表生成）
const networkOptions = computed(() => {
  const options = [{ value: '', label: '不指定' }]
  networks.value.forEach(net => {
    options.push({ value: net.name, label: `${net.name} (${net.driver})` })
  })
  return options
})

const createContainerModal = () => {
  isEditMode.value = false
  formData.value = {
    image: '',
    name: '',
    envStr: '',
    portsStr: '',
    cmd: '',
    volumesStr: '',
    restart: 'always',
    network: '',
    memory: '',
    cpus: '',
    workdir: '',
    user: '',
    hostname: '',
    privileged: false,
    capAdd: [],
    capDrop: [],
  }
  modalTitle.value = '创建容器'
  modalOpen.value = true
  loadImages()
  loadNetworks()
}

// 编辑容器配置
const editContainerModal = async (container) => {
  if (!container.name) {
    actions.showNotification('error', '只能编辑有名称的容器')
    return
  }

  isEditMode.value = true
  modalLoading.value = true
  modalTitle.value = '编辑容器配置'
  modalOpen.value = true
  showAdvanced.value = true

  try {
    const res = await api.getContainerConfig(container.name)
    const config = res.payload

    formData.value = {
      name: config.name,
      image: config.image,
      envStr: (config.env || []).join('\n'),
      portsStr: Object.entries(config.ports || {}).map(([h, c]) => `${h}:${c}`).join('\n'),
      volumesStr: (config.volumes || []).map(v => {
        let s = `${v.hostPath}:${v.containerPath}`
        if (v.readOnly) s += ':ro'
        return s
      }).join('\n'),
      cmd: (config.cmd || []).join(' '),
      restart: config.restart || 'always',
      network: config.network || '',
      memory: config.memory || '',
      cpus: config.cpus || '',
      workdir: config.workdir || '',
      user: config.user || '',
      hostname: config.hostname || '',
      privileged: config.privileged || false,
      capAdd: config.capAdd || [],
      capDrop: config.capDrop || [],
    }

    loadImages()
    loadNetworks()
  } catch (e) {
    actions.showNotification('error', '加载容器配置失败: ' + (e.response?.data?.message || e.message))
    modalOpen.value = false
  }
  modalLoading.value = false
}

const handleCreateContainer = async () => {
  if (!formData.value.image.trim()) return
  modalLoading.value = true
  try {
    const data = {
      image: formData.value.image,
      name: formData.value.name || undefined,
      env: formData.value.envStr ? formData.value.envStr.split('\n').filter(e => e.trim()) : [],
      ports: formData.value.portsStr ? Object.fromEntries(
        formData.value.portsStr.split('\n').filter(p => p.trim()).map(p => {
          const [hostPort, containerPort] = p.split(':').map(s => s.trim())
          return [hostPort, containerPort]
        })
      ) : {},
      volumes: formData.value.volumesStr ? formData.value.volumesStr.split('\n').filter(v => v.trim()).map(v => {
        const parts = v.split(':').map(s => s.trim())
        const hostPath = parts[0]
        const containerPath = parts[1]
        const readOnly = parts[2] === 'ro'
        return { hostPath, containerPath, readOnly }
      }) : [],
      restart: formData.value.restart || 'always',
      network: formData.value.network || undefined,
      memory: formData.value.memory ? parseInt(formData.value.memory) : undefined,
      cpus: formData.value.cpus ? parseFloat(formData.value.cpus) : undefined,
      workdir: formData.value.workdir || undefined,
      user: formData.value.user || undefined,
      hostname: formData.value.hostname || undefined,
      privileged: formData.value.privileged || undefined,
      capAdd: formData.value.capAdd.length > 0 ? formData.value.capAdd : undefined,
      capDrop: formData.value.capDrop.length > 0 ? formData.value.capDrop : undefined,
    }
    if (formData.value.cmd && formData.value.cmd.trim()) {
      data.cmd = formData.value.cmd.trim().split(/\s+/)
    }
    await api.createContainer(data)
    actions.showNotification('success', '容器创建成功')
    modalOpen.value = false
    loadContainers()
  } catch (e) {}
  modalLoading.value = false
}

// 更新容器配置
const handleUpdateContainer = async () => {
  if (!formData.value.image.trim() || !formData.value.name) return
  modalLoading.value = true
  try {
    const data = {
      name: formData.value.name,
      image: formData.value.image,
      env: formData.value.envStr ? formData.value.envStr.split('\n').filter(e => e.trim()) : [],
      ports: formData.value.portsStr ? Object.fromEntries(
        formData.value.portsStr.split('\n').filter(p => p.trim()).map(p => {
          const [hostPort, containerPort] = p.split(':').map(s => s.trim())
          return [hostPort, containerPort]
        })
      ) : {},
      volumes: formData.value.volumesStr ? formData.value.volumesStr.split('\n').filter(v => v.trim()).map(v => {
        const parts = v.split(':').map(s => s.trim())
        const hostPath = parts[0]
        const containerPath = parts[1]
        const readOnly = parts[2] === 'ro'
        return { hostPath, containerPath, readOnly }
      }) : [],
      restart: formData.value.restart || 'always',
      network: formData.value.network || undefined,
      memory: formData.value.memory ? parseInt(formData.value.memory) : undefined,
      cpus: formData.value.cpus ? parseFloat(formData.value.cpus) : undefined,
      workdir: formData.value.workdir || undefined,
      user: formData.value.user || undefined,
      hostname: formData.value.hostname || undefined,
      privileged: formData.value.privileged || undefined,
      capAdd: formData.value.capAdd.length > 0 ? formData.value.capAdd : undefined,
      capDrop: formData.value.capDrop.length > 0 ? formData.value.capDrop : undefined,
    }
    if (formData.value.cmd && formData.value.cmd.trim()) {
      data.cmd = formData.value.cmd.trim().split(/\s+/)
    }
    await api.updateContainerConfig(data)
    actions.showNotification('success', '容器配置更新成功，已重建容器')
    modalOpen.value = false
    loadContainers()
  } catch (e) {}
  modalLoading.value = false
}

// 批量操作
const toggleBatchMode = () => {
  batchMode.value = !batchMode.value
  if (!batchMode.value) {
    selectedIds.value = []
  }
}

const toggleSelect = (id) => {
  const idx = selectedIds.value.indexOf(id)
  if (idx >= 0) {
    selectedIds.value.splice(idx, 1)
  } else {
    selectedIds.value.push(id)
  }
}

const selectAll = () => {
  if (selectedIds.value.length === containers.value.length) {
    selectedIds.value = []
  } else {
    selectedIds.value = containers.value.map(ct => ct.id)
  }
}

// 批量操作
const batchAction = (action) => {
  if (selectedIds.value.length === 0) return
  const config = actionConfigs[action] || {}
  actions.showConfirm({
    title: `批量${config.confirmText}`,
    message: `确定要批量${config.confirmText} <strong class="text-slate-900">${selectedIds.value.length}</strong> 个容器吗？`,
    icon: config.icon,
    iconColor: config.iconColor,
    confirmText: `确认批量${config.confirmText}`,
    danger: config.danger,
    onConfirm: async () => {
      const promises = selectedIds.value.map(id => api.containerAction(id, action))
      await Promise.allSettled(promises)
      actions.showNotification('success', `批量${config.confirmText}操作完成`)
      selectedIds.value = []
      loadContainers()
    }
  })
}

// 处理镜像名称，去掉域名部分
const formatImageName = (image) => {
  if (!image) return ''
  // 去掉域名部分，保留镜像名和标签
  return image.replace(/^[^\\/]+\//, '')
}

// 暴露方法给 toolbar 使用
defineExpose({
  loadContainers,
  createContainerModal
})

onMounted(() => {
  loadContainers()
})
</script>

<template>
  <div>
    <!-- Toolbar Bar -->
    <div class="card mb-4">
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-6 py-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-emerald-500 flex items-center justify-center">
              <i class="fab fa-docker text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">容器管理</h1>
              <p class="text-xs text-slate-500">管理 Docker 容器</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <div class="flex items-center gap-1 bg-slate-100 p-1 rounded-lg">
              <button 
                @click="showAll = false; loadContainers()" 
                :class="[
                  'px-3 py-1 text-xs font-medium rounded-md transition-all duration-200',
                  !showAll 
                    ? 'bg-white text-emerald-600 shadow-sm' 
                    : 'text-slate-500 hover:text-slate-700'
                ]"
              >
                <i class="fas fa-play mr-1"></i><span class="hidden sm:inline">运行中</span>
              </button>
              <button 
                @click="showAll = true; loadContainers()" 
                :class="[
                  'px-3 py-1 text-xs font-medium rounded-md transition-all duration-200',
                  showAll 
                    ? 'bg-white text-emerald-600 shadow-sm' 
                    : 'text-slate-500 hover:text-slate-700'
                ]"
              >
                <i class="fas fa-layer-group mr-1"></i><span class="hidden sm:inline">全部</span>
              </button>
            </div>
            <button v-if="batchMode && selectedIds.length > 0" @click="batchAction('start')" class="px-3 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors" title="批量启动">
              <i class="fas fa-play"></i>
            </button>
            <button v-if="batchMode && selectedIds.length > 0" @click="batchAction('stop')" class="px-3 py-1.5 rounded-lg bg-amber-500 hover:bg-amber-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors" title="批量停止">
              <i class="fas fa-stop"></i>
            </button>
            <button v-if="batchMode && selectedIds.length > 0" @click="batchAction('remove')" class="px-3 py-1.5 rounded-lg bg-red-500 hover:bg-red-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors" title="批量删除">
              <i class="fas fa-trash"></i>
            </button>
            <button @click="toggleBatchMode()" :class="['px-3 py-1.5 rounded-lg border text-xs font-medium flex items-center gap-1.5 transition-colors', batchMode ? 'bg-blue-50 border-blue-200 text-blue-600' : 'bg-white border-slate-200 hover:bg-slate-50 text-slate-700']">
              <i class="fas fa-check-double"></i><span class="hidden sm:inline">{{ batchMode ? '取消多选' : '多选' }}</span>
            </button>
            <button @click="loadContainers()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button @click="createContainerModal()" class="px-3 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-plus"></i>创建
            </button>
          </div>
        </div>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <!-- Container List -->
      <div v-if="containers.length > 0" class="space-y-3">
        <!-- 桌面端表格视图 -->
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th v-if="batchMode" class="w-10 px-4 py-3 text-left text-xs font-semibold text-slate-600">
                  <input type="checkbox" :checked="selectedIds.length === containers.length && containers.length > 0" @change="selectAll" class="rounded border-slate-300 text-emerald-500 focus:ring-emerald-500" />
                </th>
                <th class="w-auto px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">名称</th>
                <th class="w-auto px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">镜像</th>
                <th class="w-48 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">状态</th>
                <th class="w-56 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">端口</th>
                <th class="w-32 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">创建时间</th>
                <th class="w-56 px-4 py-3 text-right text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="ct in containers" :key="ct.id" :class="['hover:bg-slate-50 transition-colors', selectedIds.includes(ct.id) ? 'bg-blue-50' : '']">
                <td v-if="batchMode" class="px-4 py-3">
                  <input type="checkbox" :checked="selectedIds.includes(ct.id)" @change="toggleSelect(ct.id)" class="rounded border-slate-300 text-emerald-500 focus:ring-emerald-500" />
                </td>
                <td class="px-4 py-3">
                  <div class="flex items-center gap-2">
                    <div :class="['w-8 h-8 rounded-lg flex items-center justify-center', ct.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
                      <i class="fas fa-box text-white text-sm"></i>
                    </div>
                    <span class="font-medium text-slate-800">{{ ct.name || ct.id }}</span>
                    <span v-if="ct.isSwarm" class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-cyan-50 text-cyan-600" title="由 Docker Swarm 管理">swarm</span>
                  </div>
                </td>
                <td class="px-4 py-3">
                  <code 
                    class="text-xs bg-slate-100 px-2 py-1 rounded" 
                    :title="ct.image"
                  >{{ formatImageName(ct.image) }}</code>
                </td>
                <td class="px-4 py-3 text-sm text-slate-600">{{ ct.status }}</td>
                <td class="px-4 py-3 font-mono text-sm text-slate-600">
                  <template v-if="ct.ports && ct.ports.length > 0">
                    <div v-for="port in ct.ports" :key="port">{{ port }}</div>
                  </template>
                  <template v-else>-</template>
                </td>
                <td class="px-4 py-3 whitespace-nowrap text-sm text-slate-600">{{ formatTime(new Date(ct.created * 1000).toISOString()) }}</td>
                <td class="px-4 py-3">
                  <div class="flex justify-end items-center gap-1">
                    <button v-if="ct.name && !ct.isSwarm" @click="editContainerModal(ct)" class="btn-icon text-violet-600 hover:bg-violet-50" title="编辑配置">
                      <i class="fas fa-cog text-xs"></i>
                    </button>
                    <button v-if="ct.state === 'running'" @click="router.push({ path: '/docker/container/' + ct.id + '/stats' })" class="btn-icon text-indigo-600 hover:bg-indigo-50" title="统计">
                      <i class="fas fa-chart-bar text-xs"></i>
                    </button>
                    <button v-if="ct.state === 'running'" @click="router.push({ path: '/docker/container/' + ct.id + '/logs' })" class="btn-icon text-slate-600 hover:bg-slate-50" title="日志">
                      <i class="fas fa-file-alt text-xs"></i>
                    </button>
                    <button v-if="ct.state === 'running'" @click="router.push({ path: '/docker/container/' + ct.id + '/terminal' })" class="btn-icon text-teal-600 hover:bg-teal-50" title="登录终端">
                      <i class="fas fa-terminal text-xs"></i>
                    </button>
                    <button v-if="ct.state !== 'running'" @click="handleContainerAction(ct, 'start')" class="btn-icon text-emerald-600 hover:bg-emerald-50" title="启动">
                      <i class="fas fa-play text-xs"></i>
                    </button>
                    <button v-if="ct.state === 'running'" @click="handleContainerAction(ct, 'stop')" class="btn-icon text-amber-600 hover:bg-amber-50" title="停止">
                      <i class="fas fa-stop text-xs"></i>
                    </button>
                    <button v-if="ct.state === 'running'" @click="handleContainerAction(ct, 'restart')" class="btn-icon text-blue-600 hover:bg-blue-50" title="重启">
                      <i class="fas fa-redo text-xs"></i>
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

        <!-- 移动端卡片视图 -->
        <div class="md:hidden space-y-3">
          <div 
            v-for="ct in containers" 
            :key="ct.id"
            :class="['rounded-xl border border-slate-200 bg-white p-4 transition-all', selectedIds.includes(ct.id) ? 'border-blue-300 bg-blue-50' : '']"
          >
            <!-- 顶部：名称和状态 -->
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-2">
                <div :class="['w-8 h-8 rounded-lg flex items-center justify-center', ct.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
                  <i class="fas fa-box text-white text-sm"></i>
                </div>
                <div>
                  <div class="flex items-center gap-2">
                    <span class="font-medium text-slate-800 text-sm">{{ ct.name || ct.id }}</span>
                    <span v-if="ct.isSwarm" class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-cyan-50 text-cyan-600">swarm</span>
                  </div>
                  <div class="flex items-center gap-2 mt-1">
                    <span :class="['text-xs px-2 py-0.5 rounded-full', ct.state === 'running' ? 'bg-emerald-100 text-emerald-700' : 'bg-slate-100 text-slate-600']">
                      {{ ct.status }}
                    </span>
                    <span class="text-xs text-slate-400">{{ formatTime(new Date(ct.created * 1000).toISOString()) }}</span>
                  </div>
                </div>
              </div>
              <div v-if="batchMode" class="ml-2">
                <input type="checkbox" :checked="selectedIds.includes(ct.id)" @change="toggleSelect(ct.id)" class="rounded border-slate-300 text-emerald-500 focus:ring-emerald-500" />
              </div>
            </div>
            
            <!-- 中间：镜像信息 -->
            <div class="mb-3">
              <p class="text-xs text-slate-500 mb-1">镜像</p>
              <code class="text-xs bg-slate-100 px-2 py-1 rounded break-all" :title="ct.image">
                {{ formatImageName(ct.image) }}
              </code>
            </div>
            
            <!-- 端口信息 -->
            <div class="mb-3">
              <p class="text-xs text-slate-500 mb-1">端口映射</p>
              <div class="font-mono text-xs text-slate-600">
                <template v-if="ct.ports && ct.ports.length > 0">
                  <div v-for="port in ct.ports" :key="port" class="truncate">{{ port }}</div>
                </template>
                <template v-else>-</template>
              </div>
            </div>
            
            <!-- 底部：操作按钮 -->
            <div class="flex flex-wrap gap-1 pt-2 border-t border-slate-100">
              <button v-if="ct.name && !ct.isSwarm" @click="editContainerModal(ct)" class="btn-icon text-violet-600 hover:bg-violet-50" title="编辑配置">
                <i class="fas fa-cog text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">配置</span>
              </button>
              <button v-if="ct.state === 'running'" @click="router.push({ path: '/docker/container/' + ct.id + '/stats' })" class="btn-icon text-indigo-600 hover:bg-indigo-50" title="统计">
                <i class="fas fa-chart-bar text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">统计</span>
              </button>
              <button v-if="ct.state === 'running'" @click="router.push({ path: '/docker/container/' + ct.id + '/logs' })" class="btn-icon text-slate-600 hover:bg-slate-50" title="日志">
                <i class="fas fa-file-alt text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">日志</span>
              </button>
              <button v-if="ct.state === 'running'" @click="router.push({ path: '/docker/container/' + ct.id + '/terminal' })" class="btn-icon text-teal-600 hover:bg-teal-50" title="终端">
                <i class="fas fa-terminal text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">终端</span>
              </button>
              <button v-if="ct.state !== 'running'" @click="handleContainerAction(ct, 'start')" class="btn-icon text-emerald-600 hover:bg-emerald-50" title="启动">
                <i class="fas fa-play text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">启动</span>
              </button>
              <button v-if="ct.state === 'running'" @click="handleContainerAction(ct, 'stop')" class="btn-icon text-amber-600 hover:bg-amber-50" title="停止">
                <i class="fas fa-stop text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">停止</span>
              </button>
              <button v-if="ct.state === 'running'" @click="handleContainerAction(ct, 'restart')" class="btn-icon text-blue-600 hover:bg-blue-50" title="重启">
                <i class="fas fa-redo text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">重启</span>
              </button>
              <button @click="handleContainerAction(ct, 'remove')" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
                <i class="fas fa-trash text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">删除</span>
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Empty State -->
      <div v-else class="flex flex-col items-center justify-center py-20">
        <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
          <i class="fab fa-docker text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">暂无容器</p>
        <p class="text-sm text-slate-400">点击「创建容器」开始使用 Docker</p>
      </div>
    </div>

    <!-- 创建容器模态框 -->
    <BaseModal
      v-model="modalOpen"
      :title="modalTitle"
      :loading="modalLoading"
      :show-footer="modalTitle === '创建容器' || modalTitle === '编辑容器配置'"
      @confirm="isEditMode ? handleUpdateContainer() : handleCreateContainer()"
    >
      <template #confirm-text>{{ isEditMode ? '更新并重建' : '创建' }}</template>
      <!-- 创建/编辑容器表单 -->
      <template v-if="modalTitle === '创建容器' || modalTitle === '编辑容器配置'">
        <form @submit.prevent="isEditMode ? handleUpdateContainer() : handleCreateContainer()" class="space-y-4">
          <!-- 编辑模式提示 -->
          <div v-if="isEditMode" class="bg-amber-50 border border-amber-200 rounded-lg p-3 mb-4">
            <p class="text-sm text-amber-700">
              <i class="fas fa-exclamation-triangle mr-1"></i>
              更新配置后将会重建容器，旧容器将被停止并删除
            </p>
          </div>

          <!-- 基础设置 -->
          <div class="grid grid-cols-2 gap-3">
            <div class="col-span-2">
              <label class="block text-sm font-medium text-slate-700 mb-2">镜像 <span class="text-red-500">*</span></label>
              <ImageSelect v-model="formData.image" :images="images" placeholder="选择或输入镜像名称" />
            </div>
            <div>
              <label class="block text-sm font-medium text-slate-700 mb-2">容器名称</label>
              <input type="text" v-model="formData.name" placeholder="my-container" class="input" :disabled="isEditMode" />
            </div>
            <div>
              <label class="block text-sm font-medium text-slate-700 mb-2">网络模式</label>
              <select v-model="formData.network" class="input">
                <option v-for="opt in networkOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
              </select>
            </div>
          </div>

          <!-- 端口映射 -->
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">端口映射</label>
            <textarea v-model="formData.portsStr" rows="2" placeholder="8080:80" class="input font-mono text-sm"></textarea>
            <p class="mt-1 text-xs text-slate-400">主机端口:容器端口</p>
          </div>

          <!-- 目录映射 -->
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">目录映射</label>
            <textarea v-model="formData.volumesStr" rows="2" placeholder="data:/app/data:ro" class="input font-mono text-sm"></textarea>
            <p class="mt-1 text-xs text-slate-400">主机路径:容器路径[:ro]，相对路径自动补全</p>
          </div>

          <!-- 环境变量 -->
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">环境变量</label>
            <textarea v-model="formData.envStr" rows="2" placeholder="KEY=value" class="input font-mono text-sm"></textarea>
          </div>

          <!-- 高级选项 -->
          <div class="border-t border-slate-200 pt-4">
            <button type="button" @click="showAdvanced = !showAdvanced" class="flex items-center gap-2 text-sm text-slate-600 hover:text-slate-800">
              <i :class="['fas fa-chevron-down text-xs transition-transform', showAdvanced ? 'rotate-180' : '']"></i>
              高级选项
            </button>
            <div v-if="showAdvanced" class="mt-4 space-y-4">
              <div>
                <label class="block text-sm font-medium text-slate-700 mb-2">启动命令</label>
                <input type="text" v-model="formData.cmd" placeholder="覆盖默认命令" class="input font-mono text-sm" />
              </div>
              <div class="grid grid-cols-2 gap-3">
                <div>
                  <label class="block text-sm font-medium text-slate-700 mb-2">重启策略</label>
                  <select v-model="formData.restart" class="input">
                    <option v-for="opt in restartOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
                  </select>
                </div>
                <div>
                  <label class="block text-sm font-medium text-slate-700 mb-2">主机名</label>
                  <input type="text" v-model="formData.hostname" placeholder="容器主机名" class="input" />
                </div>
              </div>
              <div class="grid grid-cols-2 gap-3">
                <div>
                  <label class="block text-sm font-medium text-slate-700 mb-2">内存限制 (MB)</label>
                  <input type="number" v-model="formData.memory" placeholder="512" class="input" />
                </div>
                <div>
                  <label class="block text-sm font-medium text-slate-700 mb-2">CPU 限制 (核心)</label>
                  <input type="number" step="0.1" v-model="formData.cpus" placeholder="1.5" class="input" />
                </div>
              </div>
              <div class="grid grid-cols-2 gap-3">
                <div>
                  <label class="block text-sm font-medium text-slate-700 mb-2">工作目录</label>
                  <input type="text" v-model="formData.workdir" placeholder="/app" class="input" />
                </div>
                <div>
                  <label class="block text-sm font-medium text-slate-700 mb-2">运行用户</label>
                  <input type="text" v-model="formData.user" placeholder="root" class="input" />
                </div>
              </div>
            </div>
          </div>

          <!-- 安全配置 -->
          <div class="border-t border-slate-200 pt-4">
            <button type="button" @click="showSecurity = !showSecurity" class="flex items-center gap-2 text-sm text-slate-600 hover:text-slate-800">
              <i :class="['fas fa-chevron-down text-xs transition-transform', showSecurity ? 'rotate-180' : '']"></i>
              安全配置
              <span v-if="formData.privileged || formData.capAdd?.length || formData.capDrop?.length" class="inline-flex items-center px-1.5 py-0.5 rounded-full text-xs font-medium bg-amber-100 text-amber-700">
                {{ [formData.privileged ? '特权' : '', formData.capAdd?.length ? `+${formData.capAdd.length}` : '', formData.capDrop?.length ? `-${formData.capDrop.length}` : ''].filter(Boolean).join(' ') }}
              </span>
            </button>
            <div v-if="showSecurity" class="mt-4 space-y-4">
              <div class="flex items-center gap-3">
                <label class="relative inline-flex items-center cursor-pointer">
                  <input type="checkbox" v-model="formData.privileged" class="sr-only peer" />
                  <div class="w-10 h-5 bg-slate-200 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-amber-300 rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-amber-500"></div>
                  <span class="ml-2 text-sm text-slate-700">特权模式</span>
                </label>
                <span class="text-xs text-slate-400">⚠️ 赋予容器所有主机权限，谨慎使用</span>
              </div>
              <div>
                <label class="block text-sm font-medium text-slate-700 mb-2">添加权限 (CapAdd)</label>
                <CapSelect v-model="formData.capAdd" />
              </div>
              <div>
                <label class="block text-sm font-medium text-slate-700 mb-2">移除权限 (CapDrop)</label>
                <CapSelect v-model="formData.capDrop" type="drop" />
              </div>
            </div>
          </div>
        </form>
      </template>
    </BaseModal>

  </div>
</template>
