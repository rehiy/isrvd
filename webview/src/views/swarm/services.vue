<script lang="ts">
import { Component, Inject, Ref, Vue, toNative } from 'vue-facing-decorator'

import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import api from '@/service/api'
import type { SwarmServiceInfo } from '@/service/types'

import CreateServiceModal from './widget/service-create-modal.vue'
import ServiceEditModal from './widget/service-edit-modal.vue'
import ScaleModal from './widget/service-scale-modal.vue'

@Component({
    components: { ScaleModal, CreateServiceModal, ServiceEditModal }
})
class Services extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── Refs ───
    @Ref readonly scaleModalRef!: InstanceType<typeof ScaleModal>
    @Ref readonly createServiceModalRef!: InstanceType<typeof CreateServiceModal>
    @Ref readonly editServiceModalRef!: InstanceType<typeof ServiceEditModal>

    // ─── 数据属性 ───
    services: SwarmServiceInfo[] = []
    servicesLoading = false

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
        this.servicesLoading = true
        try {
            const res = await api.swarmListServices()
            this.services = res.payload || []
        } catch (e) {
            this.actions.showNotification('error', '获取服务列表失败')
        }
        this.servicesLoading = false
    }

    handleScaleSuccess() {
        this.actions.showNotification('success', '服务扩缩容成功')
        this.loadServices()
    }

    handleCreateSuccess() {
        this.actions.showNotification('success', '服务创建成功')
        this.loadServices()
    }

    handleEditSuccess() {
        this.loadServices()
    }

    handleServiceRemove(svc: SwarmServiceInfo) {
        this.actions.showConfirm({
            title: '删除服务',
            message: `确定要删除服务 <strong class="text-slate-900">${svc.name}</strong> 吗？`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                await api.swarmServiceAction(svc.id, 'remove')
                this.actions.showNotification('success', '服务删除成功')
                this.loadServices()
            }
        })
    }

    handleRedeploy(svc: SwarmServiceInfo) {
        this.actions.showConfirm({
            title: '强制重部署',
            message: `重新拉取并部署服务 <strong class="text-slate-900">${svc.name}</strong>，正在运行的副本会滚动更新。`,
            icon: 'fa-arrows-rotate',
            iconColor: 'blue',
            confirmText: '确认重部署',
            onConfirm: async () => {
                await api.swarmRedeployService(svc.id)
                this.actions.showNotification('success', '已触发强制重部署')
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
    <div class="card mb-4">
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-emerald-500 flex items-center justify-center">
              <i class="fas fa-cubes text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">服务管理</h1>
              <p class="text-xs text-slate-500">管理 Swarm 服务</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <button @click="loadServices" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button v-if="actions.hasPerm('swarm', true)" @click="openCreateModal" class="px-3 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-plus"></i>创建
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="w-9 h-9 rounded-lg bg-emerald-500 flex items-center justify-center flex-shrink-0">
              <i class="fas fa-cubes text-white"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">服务管理</h1>
              <p class="text-xs text-slate-500 truncate">管理 Swarm 服务</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <button @click="loadServices" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 flex items-center justify-center text-slate-600 transition-colors" title="刷新">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button v-if="actions.hasPerm('swarm', true)" @click="openCreateModal" class="w-9 h-9 rounded-lg bg-emerald-500 hover:bg-emerald-600 flex items-center justify-center text-white transition-colors" title="创建">
              <i class="fas fa-plus text-sm"></i>
            </button>
          </div>
        </div>
      </div>

      <div v-if="servicesLoading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
      <div v-else-if="services.length > 0" class="space-y-3">
        <!-- 桌面端表格视图 -->
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">服务名</th>
                <th class="w-24 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">模式</th>
                <th class="w-24 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">副本</th>
                <th class="w-36 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">端口</th>
                <th class="w-36 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">更新时间</th>
                <th class="w-44 px-4 py-3 text-right text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="svc in services" :key="svc.id" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3 max-w-[280px]">
                  <div class="flex items-center gap-2 min-w-0">
                    <div class="w-8 h-8 rounded-lg bg-emerald-400 flex items-center justify-center flex-shrink-0">
                      <i class="fas fa-cubes text-white text-sm"></i>
                    </div>
                    <div class="min-w-0">
                      <span class="font-medium text-slate-800 truncate block">{{ svc.name }}</span>
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
                <td class="px-4 py-3 text-sm text-slate-500">{{ svc.updatedAt?.slice(0, 16).replace('T', ' ') }}</td>
                <td class="px-4 py-3">
                  <div class="flex justify-end items-center gap-0.5">
                    <button @click="$router.push({ name: 'swarm-service-info', params: { id: svc.id } })" class="btn-icon text-slate-600 hover:bg-slate-50" title="详情"><i class="fas fa-circle-info text-xs"></i></button>
                    <button @click="$router.push({ name: 'swarm-service-logs', params: { id: svc.id } })" class="btn-icon text-slate-600 hover:bg-slate-50" title="日志"><i class="fas fa-file-lines text-xs"></i></button>
                    <button v-if="svc.mode === 'replicated' && actions.hasPerm('swarm', true)" @click="openScaleModal(svc)" class="btn-icon text-indigo-600 hover:bg-indigo-50" title="扩缩容"><i class="fas fa-up-right-and-down-left-from-center text-xs"></i></button>
                    <button v-if="actions.hasPerm('swarm', true)" @click="handleRedeploy(svc)" class="btn-icon text-blue-600 hover:bg-blue-50" title="强制重部署"><i class="fas fa-arrows-rotate text-xs"></i></button>
                    <button v-if="actions.hasPerm('swarm', true)" @click="openEditModal(svc)" class="btn-icon text-blue-600 hover:bg-blue-50" title="编辑"><i class="fas fa-pen text-xs"></i></button>
                    <button v-if="actions.hasPerm('swarm', true)" @click="handleServiceRemove(svc)" class="btn-icon text-red-600 hover:bg-red-50" title="删除"><i class="fas fa-trash text-xs"></i></button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 移动端卡片视图 -->
        <div class="md:hidden space-y-3 p-4">
          <div 
            v-for="svc in services" 
            :key="svc.id"
            class="rounded-xl border border-slate-200 bg-white p-4 transition-all hover:shadow-sm"
          >
            <!-- 顶部：服务信息和图标 -->
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-3 min-w-0 flex-1">
                <div class="w-10 h-10 rounded-lg bg-emerald-400 flex items-center justify-center flex-shrink-0">
                  <i class="fas fa-cubes text-white text-base"></i>
                </div>
                <div class="min-w-0">
                  <span class="font-medium text-slate-800 text-sm truncate block">{{ svc.name }}</span>
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
            <div class="flex flex-wrap gap-1 pt-2 border-t border-slate-100">
              <button @click="$router.push({ name: 'swarm-service-info', params: { id: svc.id } })" class="btn-icon text-slate-600 hover:bg-slate-50" title="详情">
                <i class="fas fa-circle-info text-xs"></i><span class="text-xs ml-1">详情</span>
              </button>
              <button @click="$router.push({ name: 'swarm-service-logs', params: { id: svc.id } })" class="btn-icon text-slate-600 hover:bg-slate-50" title="日志">
                <i class="fas fa-file-lines text-xs"></i><span class="text-xs ml-1">日志</span>
              </button>
              <button v-if="svc.mode === 'replicated' && actions.hasPerm('swarm', true)" @click="openScaleModal(svc)" class="btn-icon text-indigo-600 hover:bg-indigo-50" title="扩缩容">
                <i class="fas fa-up-right-and-down-left-from-center text-xs"></i><span class="text-xs ml-1">扩缩容</span>
              </button>
              <button v-if="actions.hasPerm('swarm', true)" @click="handleRedeploy(svc)" class="btn-icon text-blue-600 hover:bg-blue-50" title="强制重部署">
                <i class="fas fa-arrows-rotate text-xs"></i><span class="text-xs ml-1">重部署</span>
              </button>
              <button v-if="actions.hasPerm('swarm', true)" @click="openEditModal(svc)" class="btn-icon text-blue-600 hover:bg-blue-50" title="编辑">
                <i class="fas fa-pen text-xs"></i><span class="text-xs ml-1">编辑</span>
              </button>
              <button v-if="actions.hasPerm('swarm', true)" @click="handleServiceRemove(svc)" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
                <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
              </button>
            </div>
          </div>
        </div>
      </div>
      <div v-else class="flex flex-col items-center justify-center py-20">
        <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
          <i class="fas fa-cubes text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">暂无服务</p>
      </div>
    </div>

    <ScaleModal ref="scaleModalRef" @success="handleScaleSuccess" />
    <CreateServiceModal ref="createServiceModalRef" @success="handleCreateSuccess" />
    <ServiceEditModal ref="editServiceModalRef" @success="handleEditSuccess" />
  </div>
</template>
