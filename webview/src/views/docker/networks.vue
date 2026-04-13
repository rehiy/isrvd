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

// 详情状态
const detailOpen = ref(false)
const detailData = ref(null)
const detailLoading = ref(false)

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
const handleNetworkAction = (net, action) => {
  actions.showConfirm({
    title: '删除网络',
    message: `确定要删除网络 <strong class="text-slate-900">${net.name}</strong> 吗？`,
    icon: 'fas fa-trash',
    iconColor: 'red',
    confirmText: '确认删除',
    danger: true,
    onConfirm: async () => {
      await api.networkAction(net.id, action)
      actions.showNotification('success', '网络删除成功')
      loadNetworks()
    }
  })
}

// 查看网络详情
const viewNetworkDetail = async (net) => {
  detailOpen.value = true
  detailData.value = null
  detailLoading.value = true
  try {
    const res = await api.networkInspect(net.id)
    detailData.value = res.payload
  } catch (e) {
    actions.showNotification('error', '获取网络详情失败')
  }
  detailLoading.value = false
}

// 判断网络是否可删除（仅 Docker 默认预置网络不可删除）
const canDeleteNetwork = (net) => {
  const undeletableNames = ['bridge', 'host', 'none']
  return !undeletableNames.includes(net.name)
}

// 获取不可删除原因
const getDeleteDisabledReason = (net) => {
  const networkNames = {
    bridge: '默认桥接网络',
    host: '主机网络',
    none: '空网络'
  }
  return `${networkNames[net.name] || '系统网络'}不可删除`
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
  <div>
    <!-- Toolbar Bar -->
    <div class="card mb-4">
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-6 py-3">
        <div class="flex items-center justify-between">
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
            <button @click="loadNetworks()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button @click="createNetworkModal()" class="px-3 py-1.5 rounded-lg bg-purple-500 hover:bg-purple-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
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

      <!-- Network Table -->
      <div v-else-if="networks.length > 0" class="overflow-x-auto">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-50 border-b border-slate-200">
              <th class="w-1/4 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">名称</th>
              <th class="w-24 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">驱动</th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">子网</th>
              <th class="w-24 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">范围</th>
              <th class="w-32 px-4 py-3 text-center text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-slate-100">
            <tr v-for="net in networks" :key="net.id" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3">
                <div class="flex items-center gap-2">
                  <div class="w-8 h-8 rounded-lg bg-purple-400 flex items-center justify-center">
                    <i class="fas fa-network-wired text-white text-sm"></i>
                  </div>
                  <span class="font-medium text-slate-800">{{ net.name }}</span>
                </div>
              </td>
              <td class="px-4 py-3"><code class="text-xs bg-slate-100 px-2 py-1 rounded">{{ net.driver }}</code></td>
              <td class="px-4 py-3 font-mono text-sm text-slate-600">{{ net.subnet || '-' }}</td>
              <td class="px-4 py-3 text-sm text-slate-600">{{ net.scope }}</td>
              <td class="px-4 py-3">
                <div class="flex justify-center items-center gap-0.5">
                  <button @click="viewNetworkDetail(net)" class="btn-icon text-purple-600 hover:bg-purple-50" title="详情">
                    <i class="fas fa-info-circle text-xs"></i>
                  </button>
                  <button
                    v-if="canDeleteNetwork(net)"
                    @click="handleNetworkAction(net, 'remove')"
                    class="btn-icon text-red-600 hover:bg-red-50"
                    title="删除"
                  >
                    <i class="fas fa-trash text-xs"></i>
                  </button>
                  <button
                    v-else
                    disabled
                    class="btn-icon text-slate-300 cursor-not-allowed"
                    :title="getDeleteDisabledReason(net)"
                  >
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
          <i class="fas fa-network-wired text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">暂无自定义网络</p>
        <p class="text-sm text-slate-400">点击「创建网络」添加自定义网络</p>
      </div>
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

    <!-- 网络详情模态框 -->
    <BaseModal
      v-model="detailOpen"
      title="网络详情"
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
            <div class="overflow-hidden">
              <span class="text-xs text-slate-500">ID</span>
              <p class="text-xs font-mono text-slate-600 mt-0.5 truncate" :title="detailData.id">{{ detailData.id }}</p>
            </div>
            <div>
              <span class="text-xs text-slate-500">驱动</span>
              <p class="text-sm font-medium text-slate-800 mt-0.5"><code class="bg-slate-100 px-2 py-0.5 rounded">{{ detailData.driver }}</code></p>
            </div>
            <div>
              <span class="text-xs text-slate-500">范围</span>
              <p class="text-sm font-medium text-slate-800 mt-0.5">{{ detailData.scope }}</p>
            </div>
            <div>
              <span class="text-xs text-slate-500">子网</span>
              <p class="text-sm font-mono text-slate-800 mt-0.5">{{ detailData.subnet || '-' }}</p>
            </div>
            <div>
              <span class="text-xs text-slate-500">网关</span>
              <p class="text-sm font-mono text-slate-800 mt-0.5">{{ detailData.gateway || '-' }}</p>
            </div>
            <div>
              <span class="text-xs text-slate-500">内部网络</span>
              <p class="text-sm font-medium text-slate-800 mt-0.5">{{ detailData.internal ? '是' : '否' }}</p>
            </div>
            <div>
              <span class="text-xs text-slate-500">IPv6</span>
              <p class="text-sm font-medium text-slate-800 mt-0.5">{{ detailData.enableIPv6 ? '已启用' : '未启用' }}</p>
            </div>
          </div>
        </div>

        <!-- 已连接的容器 -->
        <div>
          <h3 class="text-sm font-medium text-slate-700 mb-2">
            已连接容器
            <span v-if="detailData.containers" class="text-xs text-slate-400 ml-1">({{ detailData.containers.length }})</span>
          </h3>
          <div v-if="detailData.containers && detailData.containers.length > 0" class="border border-slate-200 rounded-xl overflow-hidden">
            <table class="w-full">
              <thead>
                <tr class="bg-slate-50 border-b border-slate-200">
                  <th class="px-3 py-2 text-left text-xs font-semibold text-slate-600">名称</th>
                  <th class="px-3 py-2 text-left text-xs font-semibold text-slate-600">IPv4</th>
                  <th class="px-3 py-2 text-left text-xs font-semibold text-slate-600">MAC 地址</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-100">
                <tr v-for="ct in detailData.containers" :key="ct.id" class="hover:bg-slate-50">
                  <td class="px-3 py-2">
                    <div class="flex items-center gap-1.5">
                      <div class="w-6 h-6 rounded bg-purple-100 flex items-center justify-center">
                        <i class="fas fa-box text-purple-500 text-xs"></i>
                      </div>
                      <span class="text-sm text-slate-800">{{ ct.name || ct.id }}</span>
                    </div>
                  </td>
                  <td class="px-3 py-2 font-mono text-xs text-slate-600">{{ ct.ipv4 || '-' }}</td>
                  <td class="px-3 py-2 font-mono text-xs text-slate-600">{{ ct.macAddress || '-' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div v-else class="text-sm text-slate-400 py-4 text-center bg-slate-50 rounded-xl">
            暂无容器连接到此网络
          </div>
        </div>
      </div>
    </BaseModal>
  </div>
</template>
