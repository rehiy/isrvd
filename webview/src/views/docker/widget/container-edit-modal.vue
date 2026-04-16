<script setup>
import { inject, reactive, ref } from 'vue'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY, APP_STATE_KEY } from '@/store/state.js'

import CapSelect from '@/views/docker/widget/cap-select.vue'
import ImageSelect from '@/views/docker/widget/image-select.vue'
import BaseModal from '@/component/modal.vue'

const actions = inject(APP_ACTIONS_KEY)
const state = inject(APP_STATE_KEY)

const emit = defineEmits(['success'])

const modalRef = ref(null)
const isOpen = ref(false)
const modalLoading = ref(false)
const isEditMode = ref(false)
const showAdvanced = ref(false)
const showSecurity = ref(false)

const formData = reactive({
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
})

const images = ref([])
const networks = ref([])

const restartOptions = [
  { value: 'always', label: '总是重启' },
  { value: 'unless-stopped', label: '除非手动停止' },
  { value: 'on-failure', label: '失败时重启' },
  { value: 'no', label: '不重启' }
]

const networkOptions = computed(() => {
  const options = [{ value: '', label: '不指定' }]
  networks.value.forEach(net => {
    options.push({ value: net.name, label: `${net.name} (${net.driver})` })
  })
  return options
})

import { computed } from 'vue'

const loadImages = async () => {
  try {
    const res = await api.listImages(false)
    images.value = res.payload || []
  } catch (e) {}
}

const loadNetworks = async () => {
  try {
    const res = await api.listNetworks()
    networks.value = res.payload || []
  } catch (e) {}
}

const show = async (container) => {
  if (container) {
    // 编辑模式
    isEditMode.value = true
    modalLoading.value = true
    showAdvanced.value = true
    try {
      const res = await api.getContainerConfig(container.name)
      const config = res.payload
      Object.assign(formData, {
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
      })
    } catch (e) {
      actions.showNotification('error', '加载容器配置失败: ' + (e.response?.data?.message || e.message))
      return
    } finally {
      modalLoading.value = false
    }
  } else {
    // 创建模式
    isEditMode.value = false
    Object.assign(formData, {
      image: '', name: '', envStr: '', portsStr: '', cmd: '',
      volumesStr: '', restart: 'always', network: '', memory: '',
      cpus: '', workdir: '', user: '', hostname: '',
      privileged: false, capAdd: [], capDrop: [],
    })
  }
  isOpen.value = true
  loadImages()
  loadNetworks()
}

const buildRequestData = () => {
  const data = {
    image: formData.image,
    name: formData.name || undefined,
    env: formData.envStr ? formData.envStr.split('\n').filter(e => e.trim()) : [],
    ports: formData.portsStr ? Object.fromEntries(
      formData.portsStr.split('\n').filter(p => p.trim()).map(p => {
        const [hostPort, containerPort] = p.split(':').map(s => s.trim())
        return [hostPort, containerPort]
      })
    ) : {},
    volumes: formData.volumesStr ? formData.volumesStr.split('\n').filter(v => v.trim()).map(v => {
      const parts = v.split(':').map(s => s.trim())
      const hostPath = parts[0]
      const containerPath = parts[1]
      const readOnly = parts[2] === 'ro'
      return { hostPath, containerPath, readOnly }
    }) : [],
    restart: formData.restart || 'always',
    network: formData.network || undefined,
    memory: formData.memory ? parseInt(formData.memory) : undefined,
    cpus: formData.cpus ? parseFloat(formData.cpus) : undefined,
    workdir: formData.workdir || undefined,
    user: formData.user || undefined,
    hostname: formData.hostname || undefined,
    privileged: formData.privileged || undefined,
    capAdd: formData.capAdd.length > 0 ? formData.capAdd : undefined,
    capDrop: formData.capDrop.length > 0 ? formData.capDrop : undefined,
  }
  if (formData.cmd && formData.cmd.trim()) {
    data.cmd = formData.cmd.trim().split(/\s+/)
  }
  return data
}

const handleConfirm = async () => {
  if (!formData.image.trim()) return
  modalLoading.value = true
  try {
    if (isEditMode.value) {
      const data = buildRequestData()
      data.name = formData.name
      await api.updateContainerConfig(data)
      actions.showNotification('success', '容器配置更新成功，已重建容器')
    } else {
      await api.createContainer(buildRequestData())
      actions.showNotification('success', '容器创建成功')
    }
    isOpen.value = false
    emit('success')
  } catch (e) {}
  modalLoading.value = false
}

defineExpose({ show })
</script>

<template>
  <BaseModal
    ref="modalRef"
    v-model="isOpen"
    :title="isEditMode ? '编辑容器配置' : '创建容器'"
    :loading="modalLoading"
    show-footer
    @confirm="handleConfirm"
  >
    <template #confirm-text>{{ isEditMode ? '更新并重建' : '创建' }}</template>
    <form @submit.prevent="handleConfirm" class="space-y-4">
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
  </BaseModal>
</template>
