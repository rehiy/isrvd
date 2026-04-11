<script setup>
import { inject, onMounted, ref } from 'vue'

import { formatTime } from '@/helper/utils.js'
import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const actions = inject(APP_ACTIONS_KEY)

// 自己管理数据
const containers = ref([])
const images = ref([])
const loading = ref(false)
const showAll = ref(false)

// 模态框状态
const modalOpen = ref(false)
const modalTitle = ref('')
const modalLoading = ref(false)
const formData = ref({})
const logContent = ref('')
const selectedContainer = ref(null)
const showImageDropdown = ref(false)
const imageInputRef = ref(null)
const dropdownPosition = ref({ top: 0, left: 0, width: 0 })

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
  loadImages() // 加载镜像列表供选择
}

const handleCreateContainer = async () => {
  if (!formData.value.image.trim()) return
  modalLoading.value = true
  try {
    const data = {
      image: formData.value.image,
      name: formData.value.name || undefined,
      env: formData.value.envStr ? formData.value.envStr.split('\n').filter(e => e.trim()) : [],
      ports: {},
      remove: formData.value.remove || false,
    }
    if (formData.value.hostPort && formData.value.containerPort) {
      data.ports[formData.value.hostPort] = formData.value.containerPort
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
              <i class="fas fa-play mr-1"></i>运行中
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
              <i class="fas fa-layer-group mr-1"></i>全部
            </button>
          </div>
            <div class="flex items-center gap-2 ml-2">
              <button @click="loadContainers()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
                <i class="fas fa-rotate"></i>刷新
              </button>
              <button @click="createContainerModal()" class="px-3 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
                <i class="fas fa-plus"></i>创建
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <!-- Container Table -->
      <div v-if="containers.length > 0" class="overflow-x-auto">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-50 border-b border-slate-200">
              <th class="w-1/4 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">名称</th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">镜像</th>
              <th class="w-24 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">状态</th>
              <th class="w-40 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">端口</th>
              <th class="w-32 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">创建时间</th>
              <th class="w-56 px-4 py-3 text-center text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-slate-100">
            <tr v-for="ct in containers" :key="ct.id" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3">
                <div class="flex items-center gap-2">
                  <div :class="['w-8 h-8 rounded-lg flex items-center justify-center', ct.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
                    <i class="fas fa-box text-white text-sm"></i>
                  </div>
                  <span class="font-medium text-slate-800">{{ ct.name || ct.id }}</span>
                </div>
              </td>
              <td class="px-4 py-3"><code class="text-xs bg-slate-100 px-2 py-1 rounded">{{ ct.image }}</code></td>
              <td class="px-4 py-3 text-sm text-slate-600">{{ ct.status }}</td>
              <td class="px-4 py-3 font-mono text-sm text-slate-600">{{ ct.ports || '-' }}</td>
              <td class="px-4 py-3 whitespace-nowrap text-sm text-slate-600">{{ formatTime(new Date(ct.created * 1000).toISOString()) }}</td>
              <td class="px-4 py-3">
                <div class="flex justify-center items-center gap-1">
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
      :size="logContent ? 'xl' : ''"
      :loading="modalLoading"
      :show-footer="modalTitle === '创建容器' || !!logContent"
      @confirm="handleCreateContainer"
    >
      <!-- 日志查看 -->
      <template v-if="logContent && modalTitle !== '创建容器'">
        <pre class="bg-slate-900 text-green-400 p-4 rounded-xl overflow-auto max-h-[60vh] text-sm font-mono whitespace-pre-wrap">{{ logContent }}</pre>
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

    <!-- 镜像下拉列表 -->
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
        </div>
        <div v-if="!images.filter(i => i.repoTags && i.repoTags[0] && (!formData.image || i.repoTags[0].toLowerCase().includes(formData.image.toLowerCase()))).length" class="px-3 py-2 text-sm text-slate-400 text-center">
          无匹配镜像
        </div>
      </div>
    </Teleport>
  </div>
</template>
