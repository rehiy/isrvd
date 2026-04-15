<script setup>
import { computed, inject, onMounted, ref } from 'vue'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

import BaseModal from '@/component/modal.vue'

const actions = inject(APP_ACTIONS_KEY)

// 数据
const consumers = ref([])
const whitelist = ref([])
const loading = ref(false)
const searchText = ref('')

// 编辑弹窗
const modalOpen = ref(false)
const modalTitle = ref('')
const modalLoading = ref(false)
const isEditMode = ref(false)

// 表单数据
const formData = ref({
  username: '',
  desc: '',
})

// 过滤后的用户列表
const filteredConsumers = computed(() => {
  if (!searchText.value) return consumers.value
  const s = searchText.value.toLowerCase()
  return consumers.value.filter(c =>
    (c.username || '').toLowerCase().includes(s) ||
    (c.desc || '').toLowerCase().includes(s)
  )
})

// 加载用户列表
const loadConsumers = async () => {
  loading.value = true
  try {
    const [consRes, wlRes] = await Promise.all([api.apisixListConsumers(), api.apisixGetWhitelist()])
    consumers.value = consRes.payload || []
    whitelist.value = wlRes.payload || []
  } catch (e) {
    actions.showNotification('error', '加载用户列表失败')
  }
  loading.value = false
}

// 获取用户关联的路由列表
const getConsumerRoutes = (username) => {
  return whitelist.value.filter(r => (r.consumers || []).includes(username)).map(r => r.name || r.id)
}

// 格式化时间
const formatTs = (ts) => {
  if (!ts) return '-'
  return new Date(ts * 1000).toLocaleString()
}

// 打开创建弹窗
const openCreateModal = () => {
  isEditMode.value = false
  modalTitle.value = '创建用户'
  formData.value = { username: '', desc: '' }
  modalOpen.value = true
}

// 打开编辑弹窗
const openEditModal = (consumer) => {
  isEditMode.value = true
  modalTitle.value = '编辑用户'
  formData.value = {
    username: consumer.username,
    desc: consumer.desc || '',
  }
  modalOpen.value = true
}

// 提交表单
const submitForm = async () => {
  if (!formData.value.username) {
    actions.showNotification('error', '用户名不能为空')
    return
  }
  modalLoading.value = true
  try {
    if (isEditMode.value) {
      await api.apisixUpdateConsumer(formData.value.username, { desc: formData.value.desc })
    } else {
      await api.apisixCreateConsumer(formData.value)
    }
    modalOpen.value = false
    loadConsumers()
  } catch (e) {
    actions.showNotification('error', e.message || '操作失败')
  }
  modalLoading.value = false
}

// 删除用户
const deleteConsumer = (consumer) => {
  actions.showConfirm({
    title: '删除用户',
    message: `确定要删除用户 <strong class="text-slate-900">${consumer.username}</strong> 吗？此操作不可恢复。`,
    icon: 'fa-trash',
    iconColor: 'red',
    confirmText: '确认删除',
    danger: true,
    onConfirm: async () => {
      await api.apisixDeleteConsumer(consumer.username)
      actions.showNotification('success', '删除成功')
      loadConsumers()
    }
  })
}

onMounted(() => {
  loadConsumers()
})
</script>

<template>
  <div>
    <!-- Toolbar -->
    <div class="card mb-4">
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-6 py-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-violet-500 flex items-center justify-center">
              <i class="fas fa-users text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">用户管理</h1>
              <p class="text-xs text-slate-500">管理 Apisix Consumer</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <div class="relative">
              <input
                v-model="searchText"
                type="text"
                placeholder="搜索用户..."
                class="pl-8 pr-3 py-1.5 text-xs border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent w-48"
              />
              <i class="fas fa-search absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400 text-xs"></i>
            </div>
            <button @click="loadConsumers()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button @click="openCreateModal()" class="px-3 py-1.5 rounded-lg bg-violet-500 hover:bg-violet-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
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

      <!-- 空状态 -->
      <div v-else-if="filteredConsumers.length === 0" class="flex flex-col items-center justify-center py-20">
        <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
          <i class="fas fa-users text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">暂无用户</p>
        <p class="text-sm text-slate-400">点击「创建」添加 Consumer 用户</p>
      </div>

      <!-- 用户列表 -->
      <div v-else class="overflow-x-auto">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-50 border-b border-slate-200">
              <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">用户名</th>
              <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">描述</th>
              <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">API Key</th>
              <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">关联路由</th>
              <th class="text-left px-4 py-3 text-xs font-semibold text-slate-600 uppercase tracking-wider">创建时间</th>
<th class="w-32 px-4 py-3 text-right text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-slate-100">
            <tr v-for="consumer in filteredConsumers" :key="consumer.username" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3">
                <div class="flex items-center gap-2">
                  <div class="w-8 h-8 rounded-lg bg-violet-400 flex items-center justify-center flex-shrink-0">
                    <i class="fas fa-user text-white text-sm"></i>
                  </div>
                  <span class="font-medium text-slate-800">{{ consumer.username }}</span>
                </div>
              </td>
              <td class="px-4 py-3">
                <span class="text-sm text-slate-600">{{ consumer.desc || '-' }}</span>
              </td>
              <td class="px-4 py-3">
                <code class="text-xs bg-slate-100 px-2 py-1 rounded text-slate-600">{{ consumer.plugins?.['key-auth']?.key || '-' }}</code>
              </td>
              <td class="px-4 py-3">
                <div v-if="getConsumerRoutes(consumer.username).length > 0" class="flex flex-wrap gap-1">
                  <span v-for="name in getConsumerRoutes(consumer.username)" :key="name" class="inline-flex items-center px-1.5 py-0.5 bg-violet-50 text-violet-700 rounded text-xs">{{ name }}</span>
                </div>
                <span v-else class="text-xs text-slate-400">-</span>
              </td>
              <td class="px-4 py-3 whitespace-nowrap text-sm text-slate-600">
                {{ formatTs(consumer.create_time) }}
              </td>
              <td class="px-4 py-3">
<div class="flex justify-end items-center gap-0.5">
                  <button @click="openEditModal(consumer)" class="btn-icon text-violet-600 hover:bg-violet-50" title="编辑">
                    <i class="fas fa-pen-to-square text-xs"></i>
                  </button>
                  <button @click="deleteConsumer(consumer)" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
                    <i class="fas fa-trash text-xs"></i>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- 创建/编辑弹窗 -->
    <BaseModal v-model="modalOpen" :title="modalTitle" :loading="modalLoading" @confirm="submitForm">
      <form @submit.prevent="submitForm" class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">用户名 <span class="text-red-500">*</span></label>
          <input
            v-model="formData.username"
            type="text"
            :disabled="isEditMode"
            class="input"
            :class="{ 'disabled:bg-slate-50 disabled:text-slate-500': isEditMode }"
            placeholder="输入用户名"
          />
        </div>
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">描述</label>
          <input
            v-model="formData.desc"
            type="text"
            class="input"
            placeholder="用户描述"
          />
        </div>
      </form>
      <template #confirm-text>{{ isEditMode ? '保存' : '创建' }}</template>
    </BaseModal>
  </div>
</template>
