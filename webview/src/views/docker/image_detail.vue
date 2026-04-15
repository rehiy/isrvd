<script setup>
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { formatFileSize, formatTime } from '@/helper/utils.js'
import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'
import { inject } from 'vue'

const route = useRoute()
const router = useRouter()
const actions = inject(APP_ACTIONS_KEY)

const imageId = route.params.id
const inspectData = ref(null)
const loading = ref(false)

const loadDetail = async () => {
  loading.value = true
  try {
    const res = await api.imageInspect(imageId)
    inspectData.value = res.payload
  } catch (e) {
    actions.showNotification('error', '获取镜像详情失败')
  }
  loading.value = false
}

onMounted(() => {
  loadDetail()
})
</script>

<template>
  <div>
    <div class="card mb-4">
      <!-- Toolbar -->
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-6 py-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-blue-500 flex items-center justify-center">
              <i class="fas fa-compact-disc text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">镜像详情</h1>
              <p class="text-xs text-slate-500 font-mono truncate max-w-xs">{{ imageId }}</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <button @click="loadDetail()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button @click="router.back()" class="px-3 py-1.5 rounded-lg bg-blue-500 hover:bg-blue-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-arrow-left"></i>返回
            </button>
          </div>
        </div>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <!-- Detail Content -->
      <div v-else-if="inspectData" class="p-6 space-y-6 text-sm">
        <!-- 基本信息 -->
        <div>
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">基本信息</h2>
          <div class="grid grid-cols-2 gap-3">
            <div class="col-span-2">
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">镜像 ID</label>
              <code class="block px-3 py-2 bg-slate-50 rounded-lg text-xs font-mono text-slate-700 break-all">{{ inspectData.id }}</code>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">架构</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ inspectData.architecture || '-' }}</div>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">操作系统</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ inspectData.os || '-' }}</div>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">大小</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ formatFileSize(inspectData.size) }}</div>
            </div>
            <div>
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">层数</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ inspectData.layers }}</div>
            </div>
            <div class="col-span-2">
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">创建时间</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ formatTime(inspectData.created) }}</div>
            </div>
            <div v-if="inspectData.author" class="col-span-2">
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">作者</label>
              <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ inspectData.author }}</div>
            </div>
          </div>
        </div>

        <!-- 标签 -->
        <div v-if="inspectData.repoTags && inspectData.repoTags.length > 0">
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">标签</h2>
          <div class="flex flex-wrap gap-1.5">
            <span v-for="tag in inspectData.repoTags" :key="tag" class="inline-flex items-center px-2 py-1 rounded-lg text-xs font-medium bg-blue-50 text-blue-700">{{ tag }}</span>
          </div>
        </div>

        <!-- Digest -->
        <div v-if="inspectData.repoDigests && inspectData.repoDigests.length > 0">
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Digest</h2>
          <div class="space-y-1">
            <code v-for="d in inspectData.repoDigests" :key="d" class="block px-3 py-2 bg-slate-50 rounded-lg text-xs font-mono text-slate-600 break-all">{{ d }}</code>
          </div>
        </div>

        <!-- 运行配置 -->
        <div>
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">运行配置</h2>
          <div class="space-y-3">
            <div v-if="inspectData.workingDir">
              <label class="block text-xs text-slate-500 mb-1">工作目录</label>
              <code class="block px-3 py-2 bg-slate-50 rounded-lg text-xs font-mono text-slate-700">{{ inspectData.workingDir }}</code>
            </div>
            <div v-if="inspectData.entrypoint && inspectData.entrypoint.length > 0">
              <label class="block text-xs text-slate-500 mb-1">Entrypoint</label>
              <code class="block px-3 py-2 bg-slate-50 rounded-lg text-xs font-mono text-slate-700">{{ inspectData.entrypoint.join(' ') }}</code>
            </div>
            <div v-if="inspectData.cmd && inspectData.cmd.length > 0">
              <label class="block text-xs text-slate-500 mb-1">CMD</label>
              <code class="block px-3 py-2 bg-slate-50 rounded-lg text-xs font-mono text-slate-700">{{ inspectData.cmd.join(' ') }}</code>
            </div>
            <div v-if="inspectData.exposedPorts && inspectData.exposedPorts.length > 0">
              <label class="block text-xs text-slate-500 mb-1">暴露端口</label>
              <div class="flex flex-wrap gap-1.5">
                <span v-for="port in inspectData.exposedPorts" :key="port" class="inline-flex items-center px-2 py-1 rounded-lg text-xs font-medium bg-emerald-50 text-emerald-700">{{ port }}</span>
              </div>
            </div>
            <div v-if="!inspectData.workingDir && !inspectData.entrypoint?.length && !inspectData.cmd?.length && !inspectData.exposedPorts?.length" class="text-sm text-slate-400">
              无运行配置
            </div>
          </div>
        </div>

        <!-- 环境变量 -->
        <div v-if="inspectData.env && inspectData.env.length > 0">
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">环境变量</h2>
          <div class="border border-slate-200 rounded-lg divide-y divide-slate-100">
            <div v-for="(env, idx) in inspectData.env" :key="idx" class="px-3 py-1.5">
              <code class="text-xs font-mono text-slate-600">{{ env }}</code>
            </div>
          </div>
        </div>

        <!-- Labels -->
        <div v-if="inspectData.labels && Object.keys(inspectData.labels).length > 0">
          <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Labels</h2>
          <div class="border border-slate-200 rounded-lg divide-y divide-slate-100">
            <div v-for="(val, key) in inspectData.labels" :key="key" class="px-3 py-1.5 flex gap-2">
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
          <i class="fas fa-compact-disc text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium">未找到镜像详情</p>
      </div>
    </div>
  </div>
</template>
