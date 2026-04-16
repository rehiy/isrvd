<script setup>
import { inject, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'

import { formatTime } from '@/helper/utils.js'
import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

import VolumeCreateModal from '@/views/docker/widget/volume-create-modal.vue'

const router = useRouter()

const actions = inject(APP_ACTIONS_KEY)

const volumes = ref([])
const loading = ref(false)

const createModalRef = ref(null)


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

// 删除卷
const handleVolumeAction = (vol, action) => {
  actions.showConfirm({
    title: '删除数据卷',
    message: `确定要删除数据卷 <strong class="text-slate-900">${vol.name}</strong> 吗？`,
    icon: 'fa-trash',
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
const viewVolumeDetail = (vol) => {
  router.push({ name: 'docker-volume', params: { name: vol.name } })
}

defineExpose({
  loadVolumes,
  createVolumeModal: () => createModalRef.value?.show()
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
            <button @click="createModalRef?.show()" class="px-3 py-1.5 rounded-lg bg-amber-500 hover:bg-amber-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
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

      <!-- Volume List -->
      <div v-else-if="volumes.length > 0" class="space-y-3">
        <!-- 桌面端表格视图 -->
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="w-1/4 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">名称</th>
                <th class="w-24 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">驱动</th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">挂载点</th>
                <th class="w-36 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">创建时间</th>
                <th class="w-32 px-4 py-3 text-right text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
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
                  <div class="flex justify-end items-center gap-0.5">
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

        <!-- 移动端卡片视图 -->
        <div class="md:hidden space-y-3">
          <div 
            v-for="vol in volumes" 
            :key="vol.name"
            class="rounded-xl border border-slate-200 bg-white p-4 transition-all hover:shadow-sm"
          >
            <!-- 顶部：卷信息和图标 -->
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-amber-400 flex items-center justify-center">
                  <i class="fas fa-database text-white text-base"></i>
                </div>
                <div class="min-w-0">
                  <div class="flex items-center gap-2">
                    <span class="font-medium text-slate-800 text-sm">{{ vol.name }}</span>
                  </div>
                  <div class="flex items-center gap-3 mt-1">
                    <span class="text-xs text-slate-500">{{ vol.driver }}</span>
                    <span class="text-xs text-slate-500">{{ formatTime(vol.createdAt) }}</span>
                  </div>
                </div>
              </div>
            </div>
            
            <!-- 中间：挂载点信息 -->
            <div class="mb-3">
              <p class="text-xs text-slate-500 mb-1">挂载点</p>
              <code class="font-mono text-xs text-slate-500 break-all" :title="vol.mountpoint">
                {{ vol.mountpoint }}
              </code>
            </div>
            
            <!-- 底部：操作按钮 -->
            <div class="flex flex-wrap gap-1 pt-2 border-t border-slate-100">
              <button @click="viewVolumeDetail(vol)" class="btn-icon text-amber-600 hover:bg-amber-50" title="详情">
                <i class="fas fa-info-circle text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">详情</span>
              </button>
              <button @click="handleVolumeAction(vol, 'remove')" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
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
          <i class="fas fa-database text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">暂无数据卷</p>
        <p class="text-sm text-slate-400">点击「创建数据卷」添加数据卷</p>
      </div>
    </div>

    <VolumeCreateModal ref="createModalRef" @success="loadVolumes" />


  </div>
</template>
