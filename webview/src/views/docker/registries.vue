<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { DockerRegistryInfo } from '@/service/types'

import PageSearch from '@/component/page-search.vue'

import RegistryEditModal from './widget/registry-edit-modal.vue'

@Component({
    components: { PageSearch, RegistryEditModal }
})
class Registries extends Vue {
    portal = usePortal()
    @Ref readonly editModalRef!: InstanceType<typeof RegistryEditModal>

    // ─── 数据属性 ───
    daemonMirrors: string[] = []
    indexServerAddress = ''
    registries: DockerRegistryInfo[] = []
    loading = false
    searchText = ''

    get filteredRegistries() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.registries
        return this.registries.filter((registry: DockerRegistryInfo) =>
            (registry.name || '').toLowerCase().includes(keyword) ||
            (registry.url || '').toLowerCase().includes(keyword) ||
            (registry.username || '').toLowerCase().includes(keyword) ||
            (registry.description || '').toLowerCase().includes(keyword)
        )
    }

    get showDockerHub() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return true
        const candidates = [
            'docker hub',
            'docker.io',
            'index.docker.io',
            'anonymous',
            '匿名',
            this.indexServerAddress,
            ...this.daemonMirrors
        ]
        return candidates.some((value: string) => (value || '').toLowerCase().includes(keyword))
    }

    // ─── 方法 ───
    async loadDaemonInfo() {
        try {
            const res = await api.dockerInfo()
            const info = res.payload
            this.daemonMirrors = info?.registryMirrors || []
            this.indexServerAddress = info?.indexServerAddress || ''
        } catch {}
    }

    async loadRegistries() {
        this.loading = true
        try {
            const res = await api.dockerRegistryList()
            this.registries = res.payload || []
        } catch {
            this.portal.showNotification('error', '加载仓库列表失败')
        } finally {
            this.loading = false
        }
    }

    openAdd() {
        this.editModalRef?.show(null)
    }

    openEdit(reg: DockerRegistryInfo) {
        this.editModalRef?.show(reg)
    }

    handleDelete(reg: DockerRegistryInfo) {
        this.portal.showConfirm({
            title: '删除镜像仓库',
            message: `确定要删除仓库 <strong class="text-slate-900">${reg.name}</strong> (${reg.url}) 吗？`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                try {
                    await api.dockerRegistryDelete(reg.url)
                    this.portal.showNotification('success', '仓库删除成功')
                    this.loadRegistries()
                } catch {}
            }
        })
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadDaemonInfo()
        this.loadRegistries()
    }
}

export default toNative(Registries)
</script>

<template>
  <div>
    <!-- Toolbar Bar -->
    <div class="card mb-4">
      <div class="card-toolbar">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="page-icon bg-purple-500">
              <i class="fas fa-warehouse text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">镜像仓库</h1>
              <p class="text-xs text-slate-500">管理私有镜像仓库认证信息与镜像加速器</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <PageSearch v-model="searchText" search-key="docker-registries" placeholder="搜索仓库名称、地址或账号..." width-class="w-64" focus-color="purple" type-to-search />
            <button class="btn btn-secondary" @click="loadRegistries()">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button v-if="portal.hasPerm('POST /api/docker/registry')" class="btn btn-purple" @click="openAdd">
              <i class="fas fa-plus"></i>添加仓库
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="page-icon bg-purple-500">
              <i class="fas fa-warehouse text-white"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">镜像仓库</h1>
              <p class="text-xs text-slate-500 truncate">管理仓库账号与加速器</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadRegistries()">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button v-if="portal.hasPerm('POST /api/docker/registry')" class="btn btn-purple w-9 h-9 !px-0" title="添加" @click="openAdd">
              <i class="fas fa-plus text-sm"></i>
            </button>
          </div>
        </div>
      </div>

      <div class="mobile-search">
        <PageSearch v-model="searchText" search-key="docker-registries" placeholder="搜索仓库名称、地址或账号..." width-class="w-full" focus-color="purple" />
      </div>

      <!-- Loading -->
      <div v-if="loading" class="empty-state">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <!-- Registry Table -->
      <div v-else>
        <!-- 桌面端表格视图 -->
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="th">名称</th>
                <th class="th">地址</th>
                <th class="w-28 th">认证</th>
                <th class="w-28 th-right">操作</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <!-- Docker Hub 行（始终显示） -->
              <tr v-if="showDockerHub" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3 max-w-[280px]">
                  <div class="flex items-center gap-2 min-w-0">
                    <div class="row-icon bg-purple-400">
                      <i class="fas fa-warehouse text-white text-sm"></i>
                    </div>
                    <div class="min-w-0">
                      <span class="font-medium text-slate-800 truncate block">Docker Hub</span>
                      <span class="text-xs text-slate-400 truncate block mt-0.5">默认</span>
                    </div>
                  </div>
                </td>
                <td class="px-4 py-3">
                  <div class="flex flex-col gap-1">
                    <code class="text-xs bg-slate-100 px-2 py-0.5 rounded self-start">{{ indexServerAddress || 'https://index.docker.io/v1/' }}</code>
                    <template v-if="daemonMirrors.length > 0">
                      <code v-for="mirror in daemonMirrors" :key="mirror" class="text-xs bg-slate-100 text-slate-700 px-2 py-0.5 rounded self-start inline-flex items-center gap-1">
                        <i class="fas fa-bolt text-sky-400 text-xs"></i>{{ mirror }}
                      </code>
                    </template>
                  </div>
                </td>
                <td class="px-4 py-3 text-sm text-slate-600">
                  <i class="fas fa-lock-open text-slate-400 mr-1"></i>匿名
                </td>
                <td class="px-4 py-3 text-right text-xs text-slate-400">—</td>
              </tr>
              <!-- 私有仓库行 -->
              <tr v-for="reg in filteredRegistries" :key="reg.url" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3 max-w-[280px]">
                  <div class="flex items-center gap-2 min-w-0">
                    <div class="row-icon bg-purple-400">
                      <i class="fas fa-warehouse text-white text-sm"></i>
                    </div>
                    <div class="min-w-0">
                      <span class="font-medium text-slate-800 truncate block">{{ reg.name }}</span>
                      <span v-if="reg.description" class="text-xs text-slate-400 truncate block mt-0.5">{{ reg.description }}</span>
                    </div>
                  </div>
                </td>
                <td class="px-4 py-3"><code class="text-xs bg-slate-100 px-2 py-0.5 rounded">{{ reg.url }}</code></td>
                <td class="px-4 py-3 text-sm text-slate-600">
                  <template v-if="reg.username">
                    <i class="fas fa-user text-slate-400 mr-1"></i>{{ reg.username }}
                  </template>
                  <template v-else>
                    <i class="fas fa-lock-open text-slate-400 mr-1"></i>匿名
                  </template>
                </td>
                <td class="px-4 py-3">
                  <div class="flex justify-end items-center gap-1">
                    <button v-if="portal.hasPerm('PUT /api/docker/registry')" class="btn-icon btn-icon-blue" title="编辑" @click="openEdit(reg)">
                      <i class="fas fa-pen text-xs"></i>
                    </button>
                    <button v-if="portal.hasPerm('DELETE /api/docker/registry')" class="btn-icon btn-icon-red" title="删除" @click="handleDelete(reg)">
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
          <!-- Docker Hub 卡片 -->
          <div v-if="showDockerHub" class="card-interactive">
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-3 min-w-0 flex-1">
                <div class="list-icon bg-purple-400">
                  <i class="fas fa-warehouse text-white text-base"></i>
                </div>
                <div class="min-w-0">
                  <span class="font-medium text-slate-800 text-sm truncate block">Docker Hub</span>
                  <span class="text-xs text-slate-400 truncate block mt-0.5">默认</span>
                </div>
              </div>
            </div>

            <div class="flex items-start gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">地址</span>
              <code class="text-xs bg-slate-100 text-slate-600 px-1.5 py-0.5 rounded break-all">{{ indexServerAddress || 'https://index.docker.io/v1/' }}</code>
            </div>
            <template v-if="daemonMirrors.length > 0">
              <div v-for="mirror in daemonMirrors" :key="mirror" class="flex items-center gap-2 mb-3">
                <span class="text-xs text-slate-400 flex-shrink-0">加速</span>
                <code class="text-xs bg-slate-100 text-slate-700 px-1.5 py-0.5 rounded truncate flex items-center gap-1">
                  <i class="fas fa-bolt text-sky-400 text-xs"></i>{{ mirror }}
                </code>
              </div>
            </template>
            <div class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">认证</span>
              <span class="text-xs text-slate-500"><i class="fas fa-lock-open text-slate-400 mr-1"></i>匿名</span>
            </div>
          </div>

          <!-- 私有仓库卡片 -->
          <div v-for="reg in filteredRegistries" :key="reg.url" class="card-interactive">
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-3 min-w-0 flex-1">
                <div class="list-icon bg-purple-400">
                  <i class="fas fa-warehouse text-white text-base"></i>
                </div>
                <div class="min-w-0">
                  <span class="font-medium text-slate-800 text-sm truncate block">{{ reg.name }}</span>
                  <span v-if="reg.description" class="text-xs text-slate-400 truncate block mt-0.5">{{ reg.description }}</span>
                </div>
              </div>
            </div>

            <div class="flex items-start gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">地址</span>
              <code class="text-xs bg-slate-100 text-slate-600 px-1.5 py-0.5 rounded truncate">{{ reg.url }}</code>
            </div>
            <div class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">认证</span>
              <span class="text-xs text-slate-500">
                <template v-if="reg.username"><i class="fas fa-user text-slate-400 mr-1"></i>{{ reg.username }}</template>
                <template v-else><i class="fas fa-lock-open text-slate-400 mr-1"></i>匿名</template>
              </span>
            </div>

            <!-- 底部：操作按钮 -->
            <div class="card-actions">
              <button v-if="portal.hasPerm('PUT /api/docker/registry')" class="btn-icon btn-icon-blue" title="编辑" @click="openEdit(reg)">
                <i class="fas fa-pen text-xs"></i><span class="text-xs ml-1">编辑</span>
              </button>
              <button v-if="portal.hasPerm('DELETE /api/docker/registry')" class="btn-icon btn-icon-red" title="删除" @click="handleDelete(reg)">
                <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
              </button>
            </div>
          </div>

          <div v-if="!showDockerHub && filteredRegistries.length === 0" class="rounded-xl border border-slate-200 bg-white py-10 px-4 text-center">
            <p class="text-sm text-slate-500">{{ registries.length === 0 ? '暂无镜像仓库' : '未找到匹配仓库' }}</p>
          </div>
        </div>

        <div v-if="!showDockerHub && filteredRegistries.length === 0" class="hidden md:flex flex-col items-center justify-center py-20">
          <div class="empty-state-icon">
            <i class="fas fa-warehouse text-4xl text-slate-300"></i>
          </div>
          <p class="text-slate-600 font-medium mb-1">{{ registries.length === 0 ? '暂无镜像仓库' : '未找到匹配仓库' }}</p>
          <p class="text-sm text-slate-400">{{ registries.length === 0 ? '点击「新建仓库」创建私有镜像仓库' : '尝试更换关键词或清空搜索条件' }}</p>
        </div>
      </div>
    </div>

    <RegistryEditModal ref="editModalRef" @success="loadRegistries" />
  </div>
</template>
