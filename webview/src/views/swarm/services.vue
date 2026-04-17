<script lang="ts">
import { Component, Inject, Ref, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { SwarmService } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import ScaleModal from '@/views/swarm/widget/service-scale-modal.vue'
import CreateServiceModal from '@/views/swarm/widget/service-create-modal.vue'
import ComposeModal from '@/views/swarm/widget/compose-modal.vue'

@Component({
    components: { ScaleModal, CreateServiceModal, ComposeModal }
})
class Services extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── Refs ───
    @Ref readonly scaleModalRef!: InstanceType<typeof ScaleModal>
    @Ref readonly createServiceModalRef!: InstanceType<typeof CreateServiceModal>
    @Ref readonly composeModalRef!: InstanceType<typeof ComposeModal>

    // ─── 数据属性 ───
    services: SwarmService[] = []
    servicesLoading = false

    // ─── 方法 ───
    openScaleModal(svc: SwarmService) {
        this.scaleModalRef?.show(svc)
    }

    openCreateModal() {
        this.createServiceModalRef?.show()
    }

    openComposeModal() {
        this.composeModalRef?.show()
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

    handleComposeSuccess(count: number) {
        this.actions.showNotification('success', `Compose 部署成功，已创建 ${count} 个服务`)
        this.loadServices()
    }

    handleServiceRemove(svc: SwarmService) {
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

    handleRedeploy(svc: SwarmService) {
        this.actions.showConfirm({
            title: '强制重部署',
            message: `重新拉取并部署服务 <strong class="text-slate-900">${svc.name}</strong>，正在运行的副本会滚动更新。`,
            icon: 'fa-rotate',
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
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3 flex items-center justify-between">
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
          <button @click="openCreateModal" class="px-3 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
            <i class="fas fa-plus"></i>创建
          </button>
          <button @click="openComposeModal" class="px-3 py-1.5 rounded-lg bg-violet-500 hover:bg-violet-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
            <i class="fas fa-file-code"></i>Compose
          </button>
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
                <th class="w-1/4 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">服务名</th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">镜像</th>
                <th class="w-24 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">模式</th>
                <th class="w-24 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">副本</th>
                <th class="w-36 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">端口</th>
                <th class="w-44 px-4 py-3 text-right text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="svc in services" :key="svc.id" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3">
                  <div class="flex items-center gap-2">
                    <div class="w-8 h-8 rounded-lg bg-emerald-400 flex items-center justify-center">
                      <i class="fas fa-cubes text-white text-sm"></i>
                    </div>
                    <span class="font-medium text-slate-800">{{ svc.name }}</span>
                  </div>
                </td>
                <td class="px-4 py-3"><code class="text-xs bg-slate-100 px-2 py-1 rounded">{{ svc.image }}</code></td>
                <td class="px-4 py-3"><span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-600 capitalize">{{ svc.mode }}</span></td>
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
                <td class="px-4 py-3">
                  <div class="flex justify-end items-center gap-0.5">
                    <button @click="$router.push({ name: 'swarm-service-info', params: { id: svc.id } })" class="btn-icon text-slate-600 hover:bg-slate-100" title="详情"><i class="fas fa-circle-info text-xs"></i></button>
                    <button @click="$router.push({ name: 'swarm-service-logs', params: { id: svc.id } })" class="btn-icon text-slate-600 hover:bg-slate-100" title="日志"><i class="fas fa-file-lines text-xs"></i></button>

                    <button @click="handleRedeploy(svc)" class="btn-icon text-blue-600 hover:bg-blue-50" title="强制重部署"><i class="fas fa-rotate text-xs"></i></button>
                    <button v-if="svc.mode === 'replicated'" @click="openScaleModal(svc)" class="btn-icon text-indigo-600 hover:bg-indigo-50" title="扩缩容"><i class="fas fa-up-right-and-down-left-from-center text-xs"></i></button>
                    <button @click="handleServiceRemove(svc)" class="btn-icon text-red-600 hover:bg-red-50" title="删除"><i class="fas fa-trash text-xs"></i></button>
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
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-emerald-400 flex items-center justify-center">
                  <i class="fas fa-cubes text-white text-base"></i>
                </div>
                <div class="min-w-0">
                  <div class="flex items-center gap-2">
                    <span class="font-medium text-slate-800 text-sm">{{ svc.name }}</span>
                  </div>
                  <div class="flex items-center gap-3 mt-1">
                    <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-600 capitalize">{{ svc.mode }}</span>
                    <span class="text-xs text-slate-500">
                      <span class="text-emerald-600 font-medium">{{ svc.runningTasks }}</span>
                      <span v-if="svc.mode === 'replicated'" class="text-slate-400"> / {{ svc.replicas ?? '?' }}</span>
                    </span>
                  </div>
                </div>
              </div>
            </div>
            
            <!-- 中间：镜像和端口信息 -->
            <div class="mb-3">
              <p class="text-xs text-slate-500 mb-1">镜像</p>
              <code class="text-xs bg-slate-100 px-2 py-1 rounded break-all">{{ svc.image }}</code>
            </div>
            
            <!-- 端口信息 -->
            <div class="mb-3">
              <p class="text-xs text-slate-500 mb-1">端口映射</p>
              <div class="font-mono text-xs text-slate-500">
                <template v-if="svc.ports && svc.ports.length">
                  <div v-for="p in svc.ports" :key="p.publishedPort" class="truncate">{{ p.publishedPort }}:{{ p.targetPort }}/{{ p.protocol }}</div>
                </template>
                <template v-else>-</template>
              </div>
            </div>
            
            <!-- 底部：操作按钮 -->
            <div class="flex flex-wrap gap-1 pt-2 border-t border-slate-100">
              <button @click="$router.push({ name: 'swarm-service-info', params: { id: svc.id } })" class="btn-icon text-slate-600 hover:bg-slate-50" title="详情">
                <i class="fas fa-circle-info text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">详情</span>
              </button>
              <button @click="$router.push({ name: 'swarm-service-logs', params: { id: svc.id } })" class="btn-icon text-slate-600 hover:bg-slate-50" title="日志">
                <i class="fas fa-file-lines text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">日志</span>
              </button>

              <button @click="handleRedeploy(svc)" class="btn-icon text-blue-600 hover:bg-blue-50" title="强制重部署">
                <i class="fas fa-rotate text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">重部署</span>
              </button>
              <button v-if="svc.mode === 'replicated'" @click="openScaleModal(svc)" class="btn-icon text-indigo-600 hover:bg-indigo-50" title="扩缩容">
                <i class="fas fa-up-right-and-down-left-from-center text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">扩缩容</span>
              </button>
              <button @click="handleServiceRemove(svc)" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
                <i class="fas fa-trash text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">删除</span>
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

    <ScaleModal ref="scaleModalRef" @success="handleScaleSuccess" />
    <CreateServiceModal ref="createServiceModalRef" @success="handleCreateSuccess" />
    <ComposeModal ref="composeModalRef" @success="handleComposeSuccess" />
    </div>
  </div>
</template>
