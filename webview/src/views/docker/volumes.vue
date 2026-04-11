<script setup>
import { inject, onMounted, ref } from 'vue'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const actions = inject(APP_ACTIONS_KEY)

// 卷列表数据
const volumes = ref([])

// 模态框状态
const modalOpen = ref(false)
const modalTitle = ref('')
const modalLoading = ref(false)
const formData = ref({})

// 加载卷列表
const loadVolumes = async () => {
  try {
const res = await api.listVolumes()
    volumes.value = res.payload || []
  } catch (e) {
    actions.showNotification('error', '加载卷列表失败')
  }
}

// 创建卷弹窗
const createVolumeModal = () => {
  formData.value = { name: '', driver: 'local' }
  modalTitle.value = '创建卷'
  modalOpen.value = true
}

// 创建卷
const handleCreateVolume = async () => {
  if (!formData.value.name.trim()) return
  modalLoading.value = true
  try {
    await api.createVolume(formData.value)
    actions.showNotification('success', '卷创建成功')
    modalOpen.value = false
    loadVolumes()
  } catch (e) {}
  modalLoading.value = false
}

// 删除卷
const handleVolumeAction = async (vol, action) => {
  if (!confirm(`确定要删除卷 "${vol.name}" 吗？`)) return
  try {
    await api.volumeAction(vol.name, action)
    actions.showNotification('success', '卷删除成功')
    loadVolumes()
  } catch (e) {}
}

// 暴露方法给 toolbar
defineExpose({
  refresh: loadVolumes,
  createAction: createVolumeModal
})

onMounted(() => {
  loadVolumes()
})
</script>

<template>
  <div class="p-4 space-y-4">
    <!-- 页面标题 -->
    <div class="flex items-center justify-between mb-4">
      <div class="flex items-center gap-3">
        <div class="w-9 h-9 rounded-lg bg-indigo-500 flex items-center justify-center">
          <i class="fas fa-database text-white"></i>
        </div>
        <div>
          <h1 class="text-lg font-semibold text-slate-800">数据卷</h1>
          <p class="text-xs text-slate-500">管理 Docker 持久化存储卷</p>
        </div>
      </div>
<div class="flex items-center gap-2">
        <button @click="loadVolumes()" class="px-3 py-1.5 rounded-lg bg-slate-100 hover:bg-slate-200 text-slate-600 text-xs font-medium flex items-center gap-1.5 transition-colors">
          <i class="fas fa-rotate"></i>刷新
        </button>
        <button @click="createVolumeModal()" class="px-3 py-1.5 rounded-lg bg-indigo-500 hover:bg-indigo-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
          <i class="fas fa-plus"></i>创建
        </button>
      </div>
    </div>

    <!-- Volume Table -->
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
      <p class="text-sm text-slate-400">点击顶部工具栏的「创建」按钮添加持久化存储</p>
    </div>

    <!-- 创建卷模态框 -->
    <BaseModal 
      v-model="modalOpen" 
      :title="modalTitle" 
      :loading="modalLoading"
      :show-footer="modalTitle === '创建卷'"
      @confirm="handleCreateVolume"
    >
      <template v-if="modalTitle === '创建卷'">
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
  </div>
</template>
