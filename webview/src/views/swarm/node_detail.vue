<script setup>
import { inject, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { formatFileSize, formatTime } from '@/helper/utils.js'
import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

const route = useRoute()
const router = useRouter()
const actions = inject(APP_ACTIONS_KEY)

const nodeId = route.params.id
const nodeData = ref(null)
const loading = ref(false)

const nodeStateClass = (state) => {
  if (state === 'ready') return 'bg-emerald-100 text-emerald-700'
  if (state === 'down') return 'bg-red-100 text-red-700'
  return 'bg-slate-100 text-slate-600'
}

const availabilityClass = (avail) => {
  if (avail === 'active') return 'bg-emerald-100 text-emerald-700'
  if (avail === 'drain') return 'bg-amber-100 text-amber-700'
  if (avail === 'pause') return 'bg-slate-100 text-slate-600'
  return 'bg-slate-100 text-slate-500'
}

const loadDetail = async () => {
  loading.value = true
  try {
    const res = await api.swarmInspectNode(nodeId)
    nodeData.value = res.payload
  } catch (e) {
    actions.showNotification('error', '获取节点详情失败')
  }
  loading.value = false
}

onMounted(() => loadDetail())
</script>

<template>
  <div>
    <div class="card mb-4">
      <!-- Toolbar -->
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-6 py-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <button @click="router.back()" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-100 flex items-center justify-center text-slate-600 transition-colors" title="返回节点列表">
              <i class="fas fa-arrow-left text-sm"></i>
            </button>
            <div class="w-9 h-9 rounded-lg bg-blue-500 flex items-center justify-center">
              <i class="fas fa-server text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">节点详情</h1>
              <p class="text-xs text-slate-500 font-mono truncate max-w-xs">{{ nodeId }}</p>
            </div>
          </div>
          <button @click="loadDetail()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
            <i class="fas fa-rotate"></i>刷新
          </button>
        </div>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <!-- Detail -->
      <div v-else-if="nodeData" class="p-6 space-y-6 text-sm">
        <!-- 基本信息 -->
        <div>
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">基本信息</h2>
          <div class="grid grid-cols-2 gap-3">
            <div class="col-span-2">
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">节点 ID</label>
              <code class="block px-3 py-2 bg-slate-50 rounded-lg text-xs font-mono text-slate-700 break-all">{{ nodeData.id }}</code>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">主机名</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700 flex items-center gap-2">
                {{ nodeData.hostname }}
                <span v-if="nodeData.leader" class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-indigo-100 text-indigo-700">
                  <i class="fas fa-crown mr-1 text-[10px]"></i>Leader
                </span>
              </div>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">地址</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg font-mono text-slate-700">{{ nodeData.addr || '-' }}</div>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">角色</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg">
                <span :class="nodeData.role === 'manager' ? 'bg-indigo-100 text-indigo-700' : 'bg-slate-100 text-slate-600'" class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize">{{ nodeData.role }}</span>
              </div>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">状态</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg">
                <span :class="nodeStateClass(nodeData.state)" class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize">{{ nodeData.state }}</span>
              </div>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">可用性</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg">
                <span :class="availabilityClass(nodeData.availability)" class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize">{{ nodeData.availability }}</span>
              </div>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">引擎版本</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ nodeData.engineVersion || '-' }}</div>
            </div>
          </div>
        </div>

        <!-- 硬件资源 -->
        <div>
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">硬件资源</h2>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">操作系统</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700 capitalize">{{ nodeData.os || '-' }}</div>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">架构</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ nodeData.architecture || '-' }}</div>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">CPU 核数</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ nodeData.cpus || '-' }}</div>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">内存</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ nodeData.memoryBytes ? formatFileSize(nodeData.memoryBytes) : '-' }}</div>
            </div>
          </div>
        </div>

        <!-- 时间信息 -->
        <div>
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">时间信息</h2>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">创建时间</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ formatTime(nodeData.createdAt) }}</div>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">更新时间</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ formatTime(nodeData.updatedAt) }}</div>
            </div>
          </div>
        </div>

        <!-- Labels -->
        <div v-if="nodeData.labels && Object.keys(nodeData.labels).length > 0">
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Labels</h2>
          <div class="border border-slate-200 rounded-lg divide-y divide-slate-100">
            <div v-for="(val, key) in nodeData.labels" :key="key" class="px-3 py-1.5 flex gap-2">
              <code class="text-xs font-mono text-blue-600 shrink-0">{{ key }}</code>
              <span class="text-slate-400">=</span>
              <code class="text-xs font-mono text-slate-600 break-all">{{ val }}</code>
            </div>
          </div>
        </div>
      </div>

      <!-- Empty -->
      <div v-else class="flex flex-col items-center justify-center py-20">
        <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
          <i class="fas fa-server text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium">未找到节点详情</p>
      </div>
    </div>
  </div>
</template>
