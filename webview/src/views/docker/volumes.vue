<script lang="ts">
import { Component, Inject, Ref, Vue, toNative } from 'vue-facing-decorator'

import { formatTime } from '@/helper/utils'
import api from '@/service/api'
import type { DockerVolumeInfo } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import VolumeCreateModal from '@/views/docker/widget/volume-create-modal.vue'

@Component({
    expose: ['load', 'show'],
    components: { VolumeCreateModal }
})
class Volumes extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── Refs ───
    @Ref readonly createModalRef!: InstanceType<typeof VolumeCreateModal>

    // ─── 数据属性 ───
    volumes: DockerVolumeInfo[] = []
    loading = false
    formatTime = formatTime

    // ─── 方法 ───
    async loadVolumes() {
        this.loading = true
        try {
            const res = await api.listVolumes()
            this.volumes = res.payload || []
        } catch (e) {
            this.actions.showNotification('error', '加载卷列表失败')
        }
        this.loading = false
    }

    handleVolumeAction(vol: DockerVolumeInfo, action: string) {
        this.actions.showConfirm({
            title: '删除数据卷',
            message: `确定要删除数据卷 <strong class="text-slate-900">${vol.name}</strong> 吗？`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                await api.volumeAction(vol.name, action)
                this.actions.showNotification('success', '数据卷删除成功')
                this.loadVolumes()
            }
        })
    }

    viewVolumeDetail(vol: DockerVolumeInfo) {
        this.$router.push({ name: 'docker-volume', params: { name: vol.name } })
    }

    createVolumeModal() {
        this.createModalRef?.show()
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadVolumes()
    }
}

export default toNative(Volumes)
</script>

<template>
  <div>
    <!-- Toolbar Bar -->
    <div class="card mb-4">
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
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
        <!-- 移动端 -->
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="w-9 h-9 rounded-lg bg-amber-500 flex items-center justify-center flex-shrink-0">
              <i class="fas fa-database text-white"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">数据卷管理</h1>
              <p class="text-xs text-slate-500 truncate">管理 Docker 数据卷</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <button @click="loadVolumes()" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 flex items-center justify-center text-slate-600 transition-colors" title="刷新">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button @click="createModalRef?.show()" class="w-9 h-9 rounded-lg bg-amber-500 hover:bg-amber-600 flex items-center justify-center text-white transition-colors" title="创建">
              <i class="fas fa-plus text-sm"></i>
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
                    <span class="font-medium text-slate-800 truncate max-w-[200px] block" :title="vol.name">{{ vol.name }}</span>
                  </div>
                </td>
                <td class="px-4 py-3 text-sm text-slate-600">{{ vol.driver }}</td>
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
        <div class="md:hidden space-y-3 p-4">
          <div 
            v-for="vol in volumes" 
            :key="vol.name"
            class="rounded-xl border border-slate-200 bg-white p-4 transition-all hover:shadow-sm"
          >
            <!-- 顶部：卷信息和图标 -->
            <div class="flex items-center gap-3 min-w-0 mb-3">
              <div class="w-10 h-10 rounded-lg bg-amber-400 flex items-center justify-center flex-shrink-0">
                <i class="fas fa-database text-white text-base"></i>
              </div>
              <div class="min-w-0">
                <span class="font-medium text-slate-800 text-sm truncate block" :title="vol.name">{{ vol.name }}</span>
                <span class="text-xs text-slate-500 mt-0.5 block">{{ vol.driver }}</span>
              </div>
            </div>

            <!-- 挂载点 -->
            <div class="flex items-start gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">挂载点</span>
              <code class="font-mono text-xs text-slate-500 break-all" :title="vol.mountpoint">{{ vol.mountpoint }}</code>
            </div>

            <!-- 创建时间 -->
            <div class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">创建</span>
              <span class="text-xs text-slate-500">{{ formatTime(vol.createdAt) }}</span>
            </div>
            
            <!-- 底部：操作按钮 -->
            <div class="flex flex-wrap gap-1.5 pt-2 border-t border-slate-100">
              <button @click="viewVolumeDetail(vol)" class="btn-icon text-amber-600 hover:bg-amber-50" title="详情">
                <i class="fas fa-info-circle text-xs"></i><span class="text-xs ml-1">详情</span>
              </button>
              <button @click="handleVolumeAction(vol, 'remove')" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
                <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
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
