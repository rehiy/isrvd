<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { DockerVolumeInfo } from '@/service/types'

import { formatTime } from '@/helper/utils'

import PageSearch from '@/component/page-search.vue'

import { usePortal } from '@/stores'

import VolumeCreateModal from './widget/volume-create-modal.vue'

@Component({
    expose: ['load', 'show'],
    components: { PageSearch, VolumeCreateModal }
})
class Volumes extends Vue {
    portal = usePortal()

    // ─── Refs ───
    @Ref readonly createModalRef!: InstanceType<typeof VolumeCreateModal>

    // ─── 数据属性 ───
    volumes: DockerVolumeInfo[] = []
    loading = false
    searchText = ''
    formatTime = formatTime

    get filteredVolumes() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.volumes
        return this.volumes.filter((volume: DockerVolumeInfo) =>
            volume.name.toLowerCase().includes(keyword) ||
            (volume.driver || '').toLowerCase().includes(keyword) ||
            (volume.mountpoint || '').toLowerCase().includes(keyword)
        )
    }

    // ─── 方法 ───
    async loadVolumes() {
        this.loading = true
        try {
            const res = await api.dockerVolumeList()
            this.volumes = res.payload || []
        } catch {
            this.portal.showNotification('error', '加载数据卷列表失败')
        }
        this.loading = false
    }
    handleVolumeAction(vol: DockerVolumeInfo, action: string) {
        this.portal.showConfirm({
            title: '删除数据卷',
            message: `确定要删除数据卷 <strong class="text-slate-900">${vol.name}</strong> 吗？`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                await api.dockerVolumeAction(vol.name, action)
                this.portal.showNotification('success', '数据卷删除成功')
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
              <p class="text-xs text-slate-500">管理 Docker 数据卷，持久化容器数据</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <PageSearch v-model="searchText" search-key="docker-volumes" placeholder="搜索卷名称、驱动或挂载点..." width-class="w-64" focus-color="amber" type-to-search />
            <button class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors" @click="loadVolumes()">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button v-if="portal.hasPerm('POST /api/docker/volume')" class="px-3 py-1.5 rounded-lg bg-amber-500 hover:bg-amber-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors" @click="createModalRef?.show()">
              <i class="fas fa-plus"></i>新建卷
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
              <p class="text-xs text-slate-500 truncate">管理容器数据卷</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <button class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 flex items-center justify-center text-slate-600 transition-colors" title="刷新" @click="loadVolumes()">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button v-if="portal.hasPerm('POST /api/docker/volume')" class="w-9 h-9 rounded-lg bg-amber-500 hover:bg-amber-600 flex items-center justify-center text-white transition-colors" title="新建卷" @click="createModalRef?.show()">
              <i class="fas fa-plus text-sm"></i>
            </button>
          </div>
        </div>
      </div>

      <div class="md:hidden px-4 py-2 border-b border-slate-100">
        <PageSearch v-model="searchText" search-key="docker-volumes" placeholder="搜索卷名称、驱动或挂载点..." width-class="w-full" focus-color="amber" />
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <!-- Volume List -->
      <div v-else-if="filteredVolumes.length > 0" class="space-y-3">
        <!-- 桌面端表格视图 -->
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">名称</th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">挂载点</th>
                <th class="w-36 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">创建时间</th>
                <th class="w-32 px-4 py-3 text-right text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="vol in filteredVolumes" :key="vol.name" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3 max-w-[280px]">
                  <div class="flex items-center gap-2 min-w-0">
                    <div class="w-8 h-8 rounded-lg bg-amber-400 flex items-center justify-center flex-shrink-0">
                      <i class="fas fa-database text-white text-sm"></i>
                    </div>
                    <div class="min-w-0">
                      <router-link v-if="portal.hasPerm('GET /api/docker/volume/:name')" :to="'/docker/volume/' + encodeURIComponent(vol.name)" class="font-medium text-slate-800 hover:text-amber-600 transition-colors truncate block" :title="vol.name">{{ vol.name }}</router-link>
                      <span v-else class="font-medium text-slate-800 truncate block" :title="vol.name">{{ vol.name }}</span>
                      <span class="text-xs text-slate-400 truncate block mt-0.5">{{ vol.driver }}</span>
                    </div>
                  </div>
                </td>
                <td class="px-4 py-3 font-mono text-xs text-slate-500 truncate max-w-xs" :title="vol.mountpoint">{{ vol.mountpoint }}</td>
                <td class="px-4 py-3 whitespace-nowrap text-sm text-slate-600">{{ formatTime(vol.createdAt) }}</td>
                <td class="px-4 py-3">
                  <div class="flex justify-end items-center gap-1">
                    <button v-if="portal.hasPerm('GET /api/docker/volume/:name')" class="btn-icon text-slate-600 hover:bg-slate-50" title="详情" @click="viewVolumeDetail(vol)">
                      <i class="fas fa-circle-info text-xs"></i>
                    </button>
                    <button v-if="portal.hasPerm('POST /api/docker/volume/:name/action')" class="btn-icon text-red-600 hover:bg-red-50" title="删除" @click="handleVolumeAction(vol, 'remove')">
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
            v-for="vol in filteredVolumes" 
            :key="vol.name"
            class="rounded-xl border border-slate-200 bg-white p-4 transition-all hover:shadow-sm"
          >
            <!-- 顶部：卷信息和图标 -->
            <div class="flex items-center gap-3 min-w-0 flex-1 mb-3">
              <div class="w-10 h-10 rounded-lg bg-amber-400 flex items-center justify-center flex-shrink-0">
                <i class="fas fa-database text-white text-base"></i>
              </div>
              <div class="min-w-0">
                <router-link v-if="portal.hasPerm('GET /api/docker/volume/:name')" :to="'/docker/volume/' + encodeURIComponent(vol.name)" class="font-medium text-slate-800 hover:text-amber-600 transition-colors text-sm truncate block" :title="vol.name">{{ vol.name }}</router-link>
                <span v-else class="font-medium text-slate-800 text-sm truncate block" :title="vol.name">{{ vol.name }}</span>
                <span class="text-xs text-slate-400 truncate block mt-0.5">{{ vol.driver }}</span>
              </div>
            </div>

            <!-- 挂载点 -->
            <div class="flex items-start gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">挂载点</span>
              <code class="font-mono text-xs text-slate-500 break-all" :title="vol.mountpoint">{{ vol.mountpoint }}</code>
            </div>

            <!-- 创建时间 -->
            <div class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">创建时间</span>
              <span class="text-xs text-slate-500">{{ formatTime(vol.createdAt) }}</span>
            </div>
            
            <!-- 底部：操作按钮 -->
            <div class="flex flex-wrap gap-1.5 pt-2 border-t border-slate-100">
              <button v-if="portal.hasPerm('GET /api/docker/volume/:name')" class="btn-icon text-slate-600 hover:bg-slate-50" title="详情" @click="viewVolumeDetail(vol)">
                <i class="fas fa-circle-info text-xs"></i><span class="text-xs ml-1">详情</span>
              </button>
              <button v-if="portal.hasPerm('POST /api/docker/volume/:name/action')" class="btn-icon text-red-600 hover:bg-red-50" title="删除" @click="handleVolumeAction(vol, 'remove')">
                <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Empty State -->
      <div v-else class="flex flex-col items-center justify-center py-20">
        <div class="w-16 h-16 rounded-lg bg-slate-100 flex items-center justify-center mb-4">
          <i class="fas fa-database text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">{{ volumes.length === 0 ? '暂无数据卷' : '未找到匹配数据卷' }}</p>
        <p class="text-sm text-slate-400">{{ volumes.length === 0 ? '点击「新建卷」创建数据卷' : '尝试更换关键词或清空搜索条件' }}</p>
      </div>
    </div>

    <VolumeCreateModal ref="createModalRef" @success="loadVolumes" />
  </div>
</template>
