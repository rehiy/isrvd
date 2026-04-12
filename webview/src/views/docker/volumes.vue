<script setup>
import { inject, onMounted, ref } from 'vue'

import { formatTime } from '@/helper/utils.js'
import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const actions = inject(APP_ACTIONS_KEY)

// 卷数据
const volumes = ref([])
const loading = ref(false)

// 模态框状态
const modalOpen = ref(false)
const modalTitle = ref('')
const modalLoading = ref(false)
const formData = ref({})

// 详情状态
const detailOpen = ref(false)
const detailData = ref(null)
const detailLoading = ref(false)

// 加载卷列表
const loadVolumes = async () => {
  loading.value = true
  try {
const res = await api.listVolumes()
    volumes.value = res.payload || []
  } catch (e) {
    actions.showNotification('error', '加载卷列表失败')
  }
  loading.value = false
}

// 创建卷弹窗
const createVolumeModal = () => {
  formData.value = { name: '', driver: 'local' }
  modalTitle.value = '创建数据卷'
  modalOpen.value = true
}

// 创建卷
const handleCreateVolume = async () => {
  if (!formData.value.name.trim()) return
  modalLoading.value = true
  try {
    await api.createVolume(formData.value)
    actions.showNotification('success', '数据卷创建成功')
    modalOpen.value = false
    loadVolumes()
  } catch (e) {}
  modalLoading.value = false
}

// 删除卷
const handleVolumeAction = (vol, action) => {
  actions.showConfirm({
    title: '删除数据卷',
    message: `确定要删除数据卷 <strong class="text-slate-900">${vol.name}</strong> 吗？`,
    icon: 'fas fa-trash',
    iconColor: 'red',
    confirmText: '确认删除',
    danger: true,
    onConfirm: async () => {
      await api.volumeAction(vol.name, action)
      actions.showNotification('success', '数据卷删除成功')
      loadVolumes()
    }
  })
}

// 查看卷详情
const viewVolumeDetail = async (vol) => {
  detailOpen.value = true
  detailData.value = null
  detailLoading.value = true
  try {
    const res = await api.volumeInspect(vol.name)
    detailData.value = res.payload
  } catch (e) {
    actions.showNotification('error', '获取卷详情失败')
  }
  detailLoading.value = false
}

// 暴露方法给 toolbar 使用
defineExpose({
  loadVolumes,
  createVolumeModal
})

onMounted(() => {
  loadVolumes()
})
</script>

<template>
  <div>
    <!-- Toolbar Bar -->
    <div class="card mb-4">
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-6 py-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-amber-500 flex items-center justify-center">
              <i class="fas fa-database text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">数据卷管理</h1>
              <p class="text-xs text-slate-500">管理 Docker 数据卷</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <button @click="loadVolumes()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button @click="createVolumeModal()" class="px-3 py-1.5 rounded-lg bg-amber-500 hover:bg-amber-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-plus"></i>创建
            </button>
          </div>
        </div>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <!-- Volume Table -->
      <div v-else-if="volumes.length > 0" class="overflow-x-auto">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-50 border-b border-slate-200">
              <th class="w-1/4 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">名称</th>
              <th class="w-24 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">驱动</th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">挂载点</th>
              <th class="w-36 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">创建时间</th>
              <th class="w-32 px-4 py-3 text-center text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-slate-100">
            <tr v-for="vol in volumes" :key="vol.name" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3">
                <div class="flex items-center gap-2">
                  <div class="w-8 h-8 rounded-lg bg-amber-400 flex items-center justify-center">
                    <i class="fas fa-database text-white text-sm"></i>
                  </div>
                  <span class="font-medium text-slate-800">{{ vol.name }}</span>
                </div>
              </td>
              <td class="px-4 py-3"><code class="text-xs bg-slate-100 px-2 py-1 rounded">{{ vol.driver }}</code></td>
              <td class="px-4 py-3 font-mono text-xs text-slate-500 truncate max-w-xs" :title="vol.mountpoint">{{ vol.mountpoint }}</td>
              <td class="px-4 py-3 whitespace-nowrap text-sm text-slate-600">{{ formatTime(vol.createdAt) }}</td>
              <td class="px-4 py-3">
                <div class="flex justify-center items-center gap-0.5">
                  <button @click="viewVolumeDetail(vol)" class="btn-icon text-amber-600 hover:bg-amber-50" title="详情">
                    <i class="fas fa-info-circle text-xs"></i>
                  </button>
                  <button @click="handleVolumeAction(vol, 'remove')" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
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
          <i class="fas fa-database text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">暂无数据卷</p>
        <p class="text-sm text-slate-400">点击「创建数据卷」添加数据卷</p>
      </div>
    </div>

    <!-- 创建数据卷模态框 -->
    <BaseModal 
      v-model="modalOpen" 
      :title="modalTitle" 
      :loading="modalLoading"
      :show-footer="modalTitle === '创建数据卷'"
      @confirm="handleCreateVolume"
    >
      <template v-if="modalTitle === '创建数据卷'">
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
    </BaseModal>

    <!-- 数据卷详情模态框 -->
    <BaseModal
      v-model="detailOpen"
      title="数据卷详情"
      size="lg"
      :show-footer="false"
    >
      <div v-if="detailLoading" class="flex items-center justify-center py-10">
        <div class="w-8 h-8 spinner"></div>
      </div>

      <div v-else-if="detailData" class="space-y-4">
        <!-- 基本信息 -->
        <div class="bg-slate-50 rounded-xl p-4">
          <div class="grid grid-cols-2 gap-4">
            <div>
              <span class="text-xs text-slate-500">名称</span>
              <p class="text-sm font-medium text-slate-800 mt-0.5">{{ detailData.name }}</p>
            </div>
            <div>
              <span class="text-xs text-slate-500">驱动</span>
              <p class="text-sm font-medium text-slate-800 mt-0.5"><code class="bg-slate-100 px-2 py-0.5 rounded">{{ detailData.driver }}</code></p>
            </div>
            <div>
              <span class="text-xs text-slate-500">挂载点</span>
              <p class="text-xs font-mono text-slate-600 mt-0.5 break-all">{{ detailData.mountpoint }}</p>
            </div>
            <div>
              <span class="text-xs text-slate-500">创建时间</span>
              <p class="text-sm text-slate-800 mt-0.5">{{ formatTime(detailData.createdAt) }}</p>
            </div>
            <div>
              <span class="text-xs text-slate-500">范围</span>
              <p class="text-sm text-slate-800 mt-0.5">{{ detailData.scope }}</p>
            </div>
            <div>
              <span class="text-xs text-slate-500">占用空间</span>
              <p class="text-sm text-slate-800 mt-0.5">{{ detailData.size > 0 ? formatFileSize(detailData.size) : '-' }}</p>
            </div>
            <div>
              <span class="text-xs text-slate-500">引用数</span>
              <p class="text-sm text-slate-800 mt-0.5">{{ detailData.refCount || 0 }}</p>
            </div>
          </div>
        </div>

        <!-- 使用此卷的容器 -->
        <div>
          <h3 class="text-sm font-medium text-slate-700 mb-2">
            使用此卷的容器
            <span v-if="detailData.usedBy" class="text-xs text-slate-400 ml-1">({{ detailData.usedBy.length }})</span>
          </h3>
          <div v-if="detailData.usedBy && detailData.usedBy.length > 0" class="border border-slate-200 rounded-xl overflow-hidden">
            <table class="w-full">
              <thead>
                <tr class="bg-slate-50 border-b border-slate-200">
                  <th class="px-3 py-2 text-left text-xs font-semibold text-slate-600">名称</th>
                  <th class="px-3 py-2 text-left text-xs font-semibold text-slate-600">挂载路径</th>
                  <th class="px-3 py-2 text-left text-xs font-semibold text-slate-600">权限</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-100">
                <tr v-for="ct in detailData.usedBy" :key="ct.id" class="hover:bg-slate-50">
                  <td class="px-3 py-2">
                    <div class="flex items-center gap-1.5">
                      <div class="w-6 h-6 rounded bg-amber-100 flex items-center justify-center">
                        <i class="fas fa-box text-amber-500 text-xs"></i>
                      </div>
                      <span class="text-sm text-slate-800">{{ ct.name || ct.id }}</span>
                    </div>
                  </td>
                  <td class="px-3 py-2 font-mono text-xs text-slate-600">{{ ct.mountPath }}</td>
                  <td class="px-3 py-2">
                    <span :class="ct.readOnly ? 'text-orange-600 bg-orange-50' : 'text-green-600 bg-green-50'" class="text-xs px-2 py-0.5 rounded-full font-medium">
                      {{ ct.readOnly ? '只读' : '读写' }}
                    </span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <div v-else class="text-sm text-slate-400 py-4 text-center bg-slate-50 rounded-xl">
            暂无容器使用此数据卷
          </div>
        </div>
      </div>
    </BaseModal>
  </div>
</template>
