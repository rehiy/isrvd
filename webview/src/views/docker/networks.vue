<script setup>
import { inject, onMounted, ref } from 'vue'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const actions = inject(APP_ACTIONS_KEY)

// 网络数据
const networks = ref([])
const loading = ref(false)

// 模态框状态
const modalOpen = ref(false)
const modalTitle = ref('')
const modalLoading = ref(false)
const formData = ref({})

// 加载网络列表
const loadNetworks = async () => {
  loading.value = true
  try {
const res = await api.listNetworks()
    networks.value = res.payload || []
  } catch (e) {
    actions.showNotification('error', '加载网络列表失败')
  }
  loading.value = false
}

// 创建网络弹窗
const createNetworkModal = () => {
  formData.value = { name: '', driver: 'bridge', subnet: '' }
  modalTitle.value = '创建网络'
  modalOpen.value = true
}

// 创建网络
const handleCreateNetwork = async () => {
  if (!formData.value.name.trim()) return
  modalLoading.value = true
  try {
    await api.createNetwork(formData.value)
    actions.showNotification('success', '网络创建成功')
    modalOpen.value = false
    loadNetworks()
  } catch (e) {}
  modalLoading.value = false
}

// 删除网络
const handleNetworkAction = async (net, action) => {
  if (!confirm(`确定要删除网络 "${net.name}" 吗？`)) return
  try {
    await api.networkAction(net.id, action)
    actions.showNotification('success', '网络删除成功')
    loadNetworks()
  } catch (e) {}
}

// 暴露方法给 toolbar 使用
defineExpose({
  loadNetworks,
  createNetworkModal
})

onMounted(() => {
  loadNetworks()
})
</script>

<template>
  <div class="p-4 space-y-4">
    <!-- 页面标题 -->
    <div class="flex items-center justify-between mb-4">
      <div class="flex items-center gap-3">
        <div class="w-9 h-9 rounded-lg bg-purple-500 flex items-center justify-center">
          <i class="fas fa-network-wired text-white"></i>
        </div>
        <div>
          <h1 class="text-lg font-semibold text-slate-800">网络管理</h1>
          <p class="text-xs text-slate-500">管理 Docker 网络</p>
        </div>
      </div>
<div class="flex items-center gap-2">
        <button @click="loadNetworks()" class="px-3 py-1.5 rounded-lg bg-slate-100 hover:bg-slate-200 text-slate-600 text-xs font-medium flex items-center gap-1.5 transition-colors">
          <i class="fas fa-rotate"></i>刷新
        </button>
        <button @click="createNetworkModal()" class="px-3 py-1.5 rounded-lg bg-purple-500 hover:bg-purple-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
          <i class="fas fa-plus"></i>创建
        </button>
      </div>
    </div>

    <!-- Network Table -->
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

    <!-- 创建网络模态框 -->
    <BaseModal 
      v-model="modalOpen" 
      :title="modalTitle" 
      :loading="modalLoading"
      :show-footer="modalTitle === '创建网络'"
      @confirm="handleCreateNetwork"
    >
      <template v-if="modalTitle === '创建网络'">
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
    </BaseModal>
  </div>
</template>
