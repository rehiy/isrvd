<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { SwarmServiceInfo } from '@/service/types'

import PageSearch from '@/component/page-search.vue'

import CreateServiceModal from './widget/service-create-modal.vue'
import ServiceEditModal from './widget/service-edit-modal.vue'
import ScaleModal from './widget/service-scale-modal.vue'

@Component({
    components: { PageSearch, ScaleModal, CreateServiceModal, ServiceEditModal }
})
class Services extends Vue {
    portal = usePortal()

    // ─── Refs ───
    @Ref readonly scaleModalRef!: InstanceType<typeof ScaleModal>
    @Ref readonly createServiceModalRef!: InstanceType<typeof CreateServiceModal>
    @Ref readonly editServiceModalRef!: InstanceType<typeof ServiceEditModal>

    // ─── 数据属性 ───
    services: SwarmServiceInfo[] = []
    loading = false
    searchText = ''

    get filteredServices() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.services
        return this.services.filter((svc: SwarmServiceInfo) =>
            (svc.name || '').toLowerCase().includes(keyword) ||
            (svc.id || '').toLowerCase().includes(keyword) ||
            (svc.image || '').toLowerCase().includes(keyword) ||
            (svc.mode || '').toLowerCase().includes(keyword) ||
            (svc.ports || []).some(p => `${p.publishedPort}:${p.targetPort}/${p.protocol}`.toLowerCase().includes(keyword))
        )
    }

    // ─── 方法 ───
    openScaleModal(svc: SwarmServiceInfo) {
        this.scaleModalRef?.show(svc)
    }

    openCreateModal() {
        this.createServiceModalRef?.show()
    }

    openEditModal(svc: SwarmServiceInfo) {
        this.editServiceModalRef?.show(svc)
    }

    async loadServices() {
        this.loading = true
        try {
            const res = await api.swarmServiceList()
            this.services = res.payload || []
        } finally {
            this.loading = false
        }
    }

    handleScaleSuccess() {
        this.portal.showNotification('success', '服务扩缩容成功')
        this.loadServices()
    }

    handleCreateSuccess() {
        this.portal.showNotification('success', '服务创建成功')
        this.loadServices()
    }

    handleEditSuccess() {
        this.loadServices()
    }

    handleServiceRemove(svc: SwarmServiceInfo) {
        this.portal.showConfirm({
            title: '删除服务',
            message: `确定要删除服务 <strong class="text-slate-900">${svc.name}</strong> 吗？`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                await api.swarmServiceAction(svc.id, 'remove')
                this.portal.showNotification('success', '服务删除成功')
                this.loadServices()
            }
        })
    }

    handleRedeploy(svc: SwarmServiceInfo) {
        this.portal.showConfirm({
            title: '强制重部署',
            message: `重新拉取并部署服务 <strong class="text-slate-900">${svc.name}</strong>，正在运行的副本会滚动更新。`,
            icon: 'fa-arrows-rotate',
            iconColor: 'blue',
            confirmText: '确认重部署',
            onConfirm: async () => {
                await api.swarmServiceForceUpdate(svc.id)
                this.portal.showNotification('success', '已触发强制重部署')
                this.loadServices()
            }
        })
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadServices()
    }
}

export default toNative(Services)
</script>

<template>
  <div>
    <div class="card">
      <div class="card-toolbar">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="page-icon bg-emerald-500">
              <i class="fas fa-cubes text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800 truncate">服务管理</h1>
              <p class="text-xs text-slate-500">管理 Swarm 服务</p>
            </div>
          </div>
          <div class="flex items-center gap-2 flex-shrink-0">
            <PageSearch v-model="searchText" search-key="swarm-services" placeholder="请输入搜索关键词..." focus-color="emerald" type-to-search />
            <button class="btn btn-secondary" @click="loadServices">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button v-if="portal.hasPerm('POST /api/swarm/service')" class="btn btn-emerald" @click="openCreateModal">
              <i class="fas fa-plus"></i>新建服务
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="page-icon bg-emerald-500">
              <i class="fas fa-cubes text-white"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">服务管理</h1>
              <p class="text-xs text-slate-500 truncate">管理 Swarm 服务</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadServices">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button v-if="portal.hasPerm('POST /api/swarm/service')" class="btn btn-emerald w-9 h-9 !px-0" title="创建" @click="openCreateModal">
              <i class="fas fa-plus text-sm"></i>
            </button>
          </div>
        </div>
      </div>
      <div class="mobile-search">
        <PageSearch v-model="searchText" search-key="swarm-services" placeholder="请输入搜索关键词..." width-class="w-full" focus-color="emerald" />
      </div>

      <div v-if="loading" class="card-body">
        <div class="empty-state">
          <div class="w-12 h-12 spinner mb-3"></div>
          <p class="text-slate-500">加载中...</p>
        </div>
      </div>
      <template v-else-if="filteredServices.length > 0">
        <!-- 桌面端表格视图 -->
        <div class="card-table hidden md:block">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="th">服务名</th>
                <th class="w-24 th">模式</th>
                <th class="w-24 th">副本</th>
                <th class="w-36 th">端口</th>
                <th class="w-36 th">更新时间</th>
                <th class="w-44 th-right">操作</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="svc in filteredServices" :key="svc.id" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3 max-w-[280px]">
                  <div class="flex items-center gap-2 min-w-0">
                    <div class="row-icon bg-emerald-400">
                      <i class="fas fa-cubes text-white text-sm"></i>
                    </div>
                    <div class="min-w-0">
                      <router-link v-if="portal.hasPerm('GET /api/swarm/service/:id')" :to="'/swarm/service/' + svc.id" class="font-medium text-slate-800 hover:text-emerald-600 transition-colors truncate block">{{ svc.name }}</router-link>
                      <span v-else class="font-medium text-slate-800 truncate block">{{ svc.name }}</span>
                      <code class="text-xs text-slate-400 font-mono truncate block mt-0.5">{{ svc.image }}</code>
                    </div>
                  </div>
                </td>
                <td class="px-4 py-3 text-sm text-slate-600 capitalize">{{ svc.mode }}</td>
                <td class="px-4 py-3 text-sm text-slate-600">
                  <span class="text-emerald-600 font-medium">{{ svc.runningTasks }}</span>
                  <span v-if="svc.mode === 'replicated'" class="text-slate-400"> / {{ svc.replicas ?? '?' }}</span>
                </td>
                <td class="px-4 py-3 font-mono text-xs text-slate-500">
                  <template v-if="svc.ports && svc.ports.length">
                    <div v-for="p in svc.ports" :key="p.publishedPort">{{ p.publishedPort }}:{{ p.targetPort }}/{{ p.protocol }}</div>
                  </template>
                  <template v-else>-</template>
                </td>
                <td class="px-4 py-3 text-sm text-slate-600">{{ svc.updatedAt?.slice(0, 16).replace('T', ' ') }}</td>
                <td class="px-4 py-3">
                  <div class="flex justify-end items-center gap-1">
                    <button v-if="portal.hasPerm('GET /api/swarm/service/:id')" class="btn-icon btn-icon-slate" title="详情" @click="$router.push({ name: 'swarm-service', params: { id: svc.id } })"><i class="fas fa-circle-info text-xs"></i></button>
                    <button v-if="portal.hasPerm('GET /api/swarm/service/:id/logs')" class="btn-icon btn-icon-slate" title="日志" @click="$router.push({ name: 'swarm-service-logs', params: { id: svc.id } })"><i class="fas fa-file-lines text-xs"></i></button>
                    <button v-if="svc.mode === 'replicated' && portal.hasPerm('POST /api/swarm/service/:id/action')" class="btn-icon btn-icon-indigo" title="扩缩容" @click="openScaleModal(svc)"><i class="fas fa-up-right-and-down-left-from-center text-xs"></i></button>
                    <button v-if="portal.hasPerm('POST /api/swarm/service/:id/force-update')" class="btn-icon btn-icon-blue" title="强制重部署" @click="handleRedeploy(svc)"><i class="fas fa-arrows-rotate text-xs"></i></button>
                    <button v-if="portal.hasPerm('GET /api/compose/swarm/:name') && portal.hasPerm('POST /api/compose/swarm/:name/redeploy')" class="btn-icon btn-icon-blue" title="编辑" @click="openEditModal(svc)"><i class="fas fa-pen text-xs"></i></button>
                    <button v-if="portal.hasPerm('POST /api/swarm/service/:id/action')" class="btn-icon btn-icon-red" title="删除" @click="handleServiceRemove(svc)"><i class="fas fa-trash text-xs"></i></button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 移动端卡片视图 -->
        <div class="card-body md:hidden space-y-3">
          <div v-for="svc in filteredServices" :key="svc.id" class="card-interactive">
            <!-- 顶部：服务信息和图标 -->
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-3 min-w-0 flex-1">
                <div class="list-icon bg-emerald-400">
                  <i class="fas fa-cubes text-white text-base"></i>
                </div>
                <div class="min-w-0">
                  <router-link v-if="portal.hasPerm('GET /api/swarm/service/:id')" :to="'/swarm/service/' + svc.id" class="font-medium text-slate-800 hover:text-emerald-600 transition-colors text-sm truncate block">{{ svc.name }}</router-link>
                  <span v-else class="font-medium text-slate-800 text-sm truncate block">{{ svc.name }}</span>
                  <code class="text-xs text-slate-400 font-mono truncate block mt-0.5">{{ svc.image }}</code>
                </div>
              </div>
            </div>

            <!-- 模式 -->
            <div class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">模式</span>
              <span class="text-xs text-slate-500 capitalize">{{ svc.mode }}</span>
            </div>

            <!-- 副本数 -->
            <div class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">副本</span>
              <span class="text-xs text-slate-500">
                <span class="text-emerald-600 font-medium">{{ svc.runningTasks }}</span>
                <span v-if="svc.mode === 'replicated'" class="text-slate-400"> / {{ svc.replicas ?? '?' }}</span>
              </span>
            </div>
            
            <!-- 端口信息 -->
            <div v-if="svc.ports && svc.ports.length" class="flex items-start gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">端口</span>
              <div class="font-mono text-xs text-slate-500">
                <div v-for="p in svc.ports" :key="p.publishedPort">{{ p.publishedPort }}:{{ p.targetPort }}/{{ p.protocol }}</div>
              </div>
            </div>

            <!-- 更新时间 -->
            <div class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">更新</span>
              <span class="text-xs text-slate-500">{{ svc.updatedAt?.slice(0, 16).replace('T', ' ') }}</span>
            </div>
            
            <!-- 底部：操作按钮 -->
            <div class="card-actions">
              <button v-if="portal.hasPerm('GET /api/swarm/service/:id')" class="btn-icon btn-icon-slate" title="详情" @click="$router.push({ name: 'swarm-service', params: { id: svc.id } })">
                <i class="fas fa-circle-info text-xs"></i><span class="text-xs ml-1">详情</span>
              </button>
              <button v-if="portal.hasPerm('GET /api/swarm/service/:id/logs')" class="btn-icon btn-icon-slate" title="日志" @click="$router.push({ name: 'swarm-service-logs', params: { id: svc.id } })">
                <i class="fas fa-file-lines text-xs"></i><span class="text-xs ml-1">日志</span>
              </button>
              <button v-if="svc.mode === 'replicated' && portal.hasPerm('POST /api/swarm/service/:id/action')" class="btn-icon btn-icon-indigo" title="扩缩容" @click="openScaleModal(svc)">
                <i class="fas fa-up-right-and-down-left-from-center text-xs"></i><span class="text-xs ml-1">扩缩容</span>
              </button>
              <button v-if="portal.hasPerm('POST /api/swarm/service/:id/force-update')" class="btn-icon btn-icon-blue" title="强制重部署" @click="handleRedeploy(svc)">
                <i class="fas fa-arrows-rotate text-xs"></i><span class="text-xs ml-1">重部署</span>
              </button>
              <button v-if="portal.hasPerm('GET /api/compose/swarm/:name') && portal.hasPerm('POST /api/compose/swarm/:name/redeploy')" class="btn-icon btn-icon-blue" title="编辑" @click="openEditModal(svc)">
                <i class="fas fa-pen text-xs"></i><span class="text-xs ml-1">编辑</span>
              </button>
              <button v-if="portal.hasPerm('POST /api/swarm/service/:id/action')" class="btn-icon btn-icon-red" title="删除" @click="handleServiceRemove(svc)">
                <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
              </button>
            </div>
          </div>
        </div>
      </template>
      <div v-else class="card-body">
        <div class="empty-state">
          <div class="empty-state-icon">
            <i class="fas fa-cubes text-4xl text-slate-300"></i>
          </div>
          <p class="text-slate-600 font-medium mb-1">{{ services.length === 0 ? '暂无服务' : '未找到匹配服务' }}</p>
          <p class="text-sm text-slate-400">{{ services.length === 0 ? '点击「新建服务」添加 Swarm 服务' : '尝试更换关键词或清空搜索条件' }}</p>
        </div>
      </div>
    </div>

    <ScaleModal ref="scaleModalRef" @success="handleScaleSuccess" />
    <CreateServiceModal ref="createServiceModalRef" @success="handleCreateSuccess" />
    <ServiceEditModal ref="editServiceModalRef" @success="handleEditSuccess" />
  </div>
</template>
