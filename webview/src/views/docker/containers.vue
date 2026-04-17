<script lang="ts">
import { Component, Inject, Ref, Vue, toNative } from 'vue-facing-decorator'

import { formatTime } from '@/helper/utils'
import api from '@/service/api'
import type { ContainerInfo } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import ContainerEditModal from '@/views/docker/widget/container-edit-modal.vue'
import ComposeModal from '@/views/docker/widget/compose-modal.vue'

@Component({
    expose: ['load', 'show'],
    components: { ContainerEditModal, ComposeModal }
})
class Containers extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── Refs ───
    @Ref readonly containerModalRef!: InstanceType<typeof ContainerEditModal>
    @Ref readonly composeModalRef!: InstanceType<typeof ComposeModal>

    // ─── 数据属性 ───
    containers: ContainerInfo[] = []
    loading = false
    showAll = false
    selectedIds: string[] = []
    batchMode = false

    readonly actionConfigs: Record<string, { icon: string; iconColor: string; title: string; confirmText: string; danger?: boolean }> = {
        start: { icon: 'fa-play', iconColor: 'emerald', title: '启动容器', confirmText: '启动' },
        stop: { icon: 'fa-stop', iconColor: 'amber', title: '停止容器', confirmText: '停止' },
        restart: { icon: 'fa-redo', iconColor: 'blue', title: '重启容器', confirmText: '重启' },
        remove: { icon: 'fa-trash', iconColor: 'red', title: '删除容器', confirmText: '删除', danger: true },
        pause: { icon: 'fa-pause', iconColor: 'amber', title: '暂停容器', confirmText: '暂停' },
        unpause: { icon: 'fa-play', iconColor: 'emerald', title: '恢复容器', confirmText: '恢复' }
    }

    // ─── 方法 ───
    async loadContainers() {
        this.loading = true
        try {
            const res = await api.listContainers(this.showAll)
            this.containers = res.payload || []
        } catch (e) {
            this.actions.showNotification('error', '加载容器列表失败')
        }
        this.loading = false
    }

    handleContainerAction(container: ContainerInfo, action: string) {
        const config = this.actionConfigs[action]
        if (!config) return
        this.actions.showConfirm({
            title: config.title,
            message: `确定要${config.confirmText}容器 <strong class="text-slate-900">${container.name || container.id}</strong> 吗？`,
            icon: config.icon,
            iconColor: config.iconColor,
            confirmText: `确认${config.confirmText}`,
            danger: config.danger,
            onConfirm: async () => {
                await api.containerAction(container.id, action)
                this.actions.showNotification('success', `容器 ${config.confirmText} 成功`)
                this.loadContainers()
            }
        })
    }

    createContainerModal() {
        this.containerModalRef?.show()
    }

    openComposeModal() {
        this.composeModalRef?.show()
    }

    toggleBatchMode() {
        this.batchMode = !this.batchMode
        if (!this.batchMode) {
            this.selectedIds = []
        }
    }

    toggleSelect(id: string) {
        if (this.selectedIds.includes(id)) {
            this.selectedIds = this.selectedIds.filter(i => i !== id)
        } else {
            this.selectedIds.push(id)
        }
    }

    selectAll() {
        this.selectedIds = this.selectedIds.length === this.containers.length
            ? []
            : this.containers.map(ct => ct.id)
    }

    batchAction(action: string) {
        if (this.selectedIds.length === 0) return
        const config = this.actionConfigs[action]
        if (!config) return
        this.actions.showConfirm({
            title: `批量${config.confirmText}`,
            message: `确定要批量${config.confirmText} <strong class="text-slate-900">${this.selectedIds.length}</strong> 个容器吗？`,
            icon: config.icon,
            iconColor: config.iconColor,
            confirmText: `确认批量${config.confirmText}`,
            danger: config.danger,
            onConfirm: async () => {
                const promises = this.selectedIds.map(id => api.containerAction(id, action))
                await Promise.allSettled(promises)
                this.actions.showNotification('success', `批量${config.confirmText}操作完成`)
                this.selectedIds = []
                this.loadContainers()
            }
        })
    }

    formatImageName(image: string) {
        if (!image) return ''
        // 去掉 registry host（含端口）前缀，只保留 name:tag 部分
        return image.replace(/^[^/]+\.[^/]+\//, '').replace(/^[^/]+:[0-9]+\//, '')
    }

    formatTime = formatTime

    // ─── 生命周期 ───
    mounted() {
        this.loadContainers()
    }
}

export default toNative(Containers)
</script>

<template>
  <div>
    <!-- Toolbar Bar -->
    <div class="card mb-4">
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-emerald-500 flex items-center justify-center">
              <i class="fab fa-docker text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">容器管理</h1>
              <p class="text-xs text-slate-500">管理 Docker 容器</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <div class="flex gap-1 bg-slate-100 p-1 rounded-lg">
              <button @click="showAll = false; loadContainers()" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', !showAll ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
                <i class="fas fa-play"></i><span>运行中</span>
              </button>
              <button @click="showAll = true; loadContainers()" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', showAll ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
                <i class="fas fa-layer-group"></i><span>全部</span>
              </button>
            </div>
            <button v-if="batchMode && selectedIds.length > 0" @click="batchAction('start')" class="px-3 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors" title="批量启动">
              <i class="fas fa-play"></i>
            </button>
            <button v-if="batchMode && selectedIds.length > 0" @click="batchAction('stop')" class="px-3 py-1.5 rounded-lg bg-amber-500 hover:bg-amber-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors" title="批量停止">
              <i class="fas fa-stop"></i>
            </button>
            <button v-if="batchMode && selectedIds.length > 0" @click="batchAction('remove')" class="px-3 py-1.5 rounded-lg bg-red-500 hover:bg-red-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors" title="批量删除">
              <i class="fas fa-trash"></i>
            </button>
            <button @click="toggleBatchMode()" :class="['px-3 py-1.5 rounded-lg border text-xs font-medium flex items-center gap-1.5 transition-colors', batchMode ? 'bg-blue-50 border-blue-200 text-blue-600' : 'bg-white border-slate-200 hover:bg-slate-50 text-slate-700']">
              <i class="fas fa-check-double"></i><span>{{ batchMode ? '取消多选' : '多选' }}</span>
            </button>
            <button @click="loadContainers()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button @click="createContainerModal()" class="px-3 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-plus"></i>创建
            </button>
            <button @click="openComposeModal" class="px-3 py-1.5 rounded-lg bg-violet-500 hover:bg-violet-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-file-code"></i>Compose
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="block md:hidden">
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-3 min-w-0">
              <div class="w-9 h-9 rounded-lg bg-emerald-500 flex items-center justify-center flex-shrink-0">
                <i class="fab fa-docker text-white"></i>
              </div>
              <div class="min-w-0">
                <h1 class="text-lg font-semibold text-slate-800 truncate">容器管理</h1>
                <p class="text-xs text-slate-500 truncate">管理 Docker 容器</p>
              </div>
            </div>
            <div class="flex items-center gap-1 flex-shrink-0">
              <button @click="toggleBatchMode()" :class="['w-9 h-9 rounded-lg border flex items-center justify-center transition-colors', batchMode ? 'bg-blue-50 border-blue-200 text-blue-600' : 'bg-white border-slate-200 hover:bg-slate-50 text-slate-700']" :title="batchMode ? '取消多选' : '多选'">
                <i class="fas fa-check-double text-sm"></i>
              </button>
              <button @click="loadContainers()" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 flex items-center justify-center text-slate-600 transition-colors" title="刷新">
                <i class="fas fa-rotate text-sm"></i>
              </button>
              <button @click="createContainerModal()" class="w-9 h-9 rounded-lg bg-emerald-500 hover:bg-emerald-600 flex items-center justify-center text-white transition-colors" title="创建">
                <i class="fas fa-plus text-sm"></i>
              </button>
              <button @click="openComposeModal" class="w-9 h-9 rounded-lg bg-violet-500 hover:bg-violet-600 flex items-center justify-center text-white transition-colors" title="Compose">
                <i class="fas fa-file-code text-sm"></i>
              </button>
            </div>
          </div>
          <!-- 批量操作（移动端） -->
          <div v-if="batchMode && selectedIds.length > 0" class="flex items-center gap-1 mb-2">
            <button @click="batchAction('start')" class="flex-1 px-3 py-1.5 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-white text-xs font-medium flex items-center justify-center gap-1.5 transition-colors">
              <i class="fas fa-play"></i>批量启动
            </button>
            <button @click="batchAction('stop')" class="flex-1 px-3 py-1.5 rounded-lg bg-amber-500 hover:bg-amber-600 text-white text-xs font-medium flex items-center justify-center gap-1.5 transition-colors">
              <i class="fas fa-stop"></i>批量停止
            </button>
            <button @click="batchAction('remove')" class="flex-1 px-3 py-1.5 rounded-lg bg-red-500 hover:bg-red-600 text-white text-xs font-medium flex items-center justify-center gap-1.5 transition-colors">
              <i class="fas fa-trash"></i>批量删除
            </button>
          </div>
          <div class="flex justify-center gap-1 bg-slate-100 p-1 rounded-lg">
            <button @click="showAll = false; loadContainers()" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', !showAll ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
              <i class="fas fa-play"></i><span>运行中</span>
            </button>
            <button @click="showAll = true; loadContainers()" :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5', showAll ? 'bg-white text-emerald-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']">
              <i class="fas fa-layer-group"></i><span>全部</span>
            </button>
          </div>
        </div>
      </div>
      <!-- Loading -->
      <div v-if="loading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <!-- Container List -->
      <div v-if="containers.length > 0" class="space-y-3">
        <!-- 桌面端表格视图 -->
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th v-if="batchMode" class="w-10 px-4 py-3 text-left text-xs font-semibold text-slate-600">
                  <input type="checkbox" :checked="selectedIds.length === containers.length && containers.length > 0" @change="selectAll" class="rounded border-slate-300 text-emerald-500 focus:ring-emerald-500" />
                </th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">名称</th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">镜像</th>
                <th class="w-48 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">状态</th>
                <th class="w-56 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">端口</th>
                <th class="w-32 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">创建时间</th>
                <th class="w-56 px-4 py-3 text-right text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="ct in containers" :key="ct.id" :class="['hover:bg-slate-50 transition-colors', selectedIds.includes(ct.id) ? 'bg-blue-50' : '']">
                <td v-if="batchMode" class="px-4 py-3">
                  <input type="checkbox" :checked="selectedIds.includes(ct.id)" @change="toggleSelect(ct.id)" class="rounded border-slate-300 text-emerald-500 focus:ring-emerald-500" />
                </td>
                <td class="px-4 py-3">
                  <div class="flex items-center gap-2">
                    <div :class="['w-8 h-8 rounded-lg flex items-center justify-center', ct.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
                      <i class="fas fa-box text-white text-sm"></i>
                    </div>
                    <span class="font-medium text-slate-800 truncate max-w-[160px]">{{ ct.name || ct.id }}</span>
                  </div>
                </td>
                <td class="px-4 py-3">
                  <code 
                    class="text-xs bg-slate-100 px-2 py-1 rounded" 
                    :title="ct.image"
                  >{{ formatImageName(ct.image) }}</code>
                </td>
                <td class="px-4 py-3 text-sm text-slate-600">{{ ct.status }}</td>
                <td class="px-4 py-3 font-mono text-sm text-slate-600">
                  <template v-if="ct.ports && ct.ports.length > 0">
                    <div v-for="port in ct.ports" :key="port">{{ port }}</div>
                  </template>
                  <template v-else>-</template>
                </td>
                <td class="px-4 py-3 whitespace-nowrap text-sm text-slate-600">{{ formatTime(new Date(ct.created * 1000).toISOString()) }}</td>
                <td class="px-4 py-3">
                  <div class="flex justify-end items-center gap-1">
                    <button @click="!ct.isSwarm && containerModalRef?.show(ct)" :disabled="ct.isSwarm" :class="['btn-icon', ct.isSwarm ? 'text-slate-300 cursor-not-allowed' : 'text-violet-600 hover:bg-violet-50']" :title="ct.isSwarm ? '由 Swarm 管理，不支持直接编辑' : '编辑配置'">
                      <i class="fas fa-cog text-xs"></i>
                    </button>
                    <button v-if="ct.state === 'running'" @click="$router.push({ path: '/docker/container/' + ct.id + '/stats' })" class="btn-icon text-indigo-600 hover:bg-indigo-50" title="统计">
                      <i class="fas fa-chart-bar text-xs"></i>
                    </button>
                    <button v-if="ct.state === 'running'" @click="$router.push({ path: '/docker/container/' + ct.id + '/logs' })" class="btn-icon text-slate-600 hover:bg-slate-50" title="日志">
                      <i class="fas fa-file-alt text-xs"></i>
                    </button>
                    <button v-if="ct.state === 'running'" @click="$router.push({ path: '/docker/container/' + ct.id + '/terminal' })" class="btn-icon text-teal-600 hover:bg-teal-50" title="登录终端">
                      <i class="fas fa-terminal text-xs"></i>
                    </button>
                    <button v-if="ct.state !== 'running'" @click="handleContainerAction(ct, 'start')" class="btn-icon text-emerald-600 hover:bg-emerald-50" title="启动">
                      <i class="fas fa-play text-xs"></i>
                    </button>
                    <button v-if="ct.state === 'running'" @click="handleContainerAction(ct, 'restart')" class="btn-icon text-blue-600 hover:bg-blue-50" title="重启">
                      <i class="fas fa-redo text-xs"></i>
                    </button>
                    <button v-if="ct.state === 'running'" @click="handleContainerAction(ct, 'stop')" class="btn-icon text-amber-600 hover:bg-amber-50" title="停止">
                      <i class="fas fa-stop text-xs"></i>
                    </button>
                    <button @click="handleContainerAction(ct, 'remove')" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
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
            v-for="ct in containers" 
            :key="ct.id"
            :class="['rounded-xl border border-slate-200 bg-white p-4 transition-all', selectedIds.includes(ct.id) ? 'border-blue-300 bg-blue-50' : '']"
          >
            <!-- 顶部：名称和状态 -->
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-3">
                <div :class="['w-10 h-10 rounded-lg flex items-center justify-center', ct.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
                  <i class="fas fa-box text-white text-base"></i>
                </div>
                <div class="min-w-0">
                  <div class="flex items-center gap-2">
                    <span class="font-medium text-slate-800 text-sm">{{ ct.name || ct.id }}</span>
                  </div>
                  <div class="flex items-center gap-3 mt-1">
                    <span :class="['text-xs px-2 py-0.5 rounded-full', ct.state === 'running' ? 'bg-emerald-100 text-emerald-700' : 'bg-slate-100 text-slate-600']">
                      {{ ct.status }}
                    </span>
                    <span class="text-xs text-slate-500">{{ formatTime(new Date(ct.created * 1000).toISOString()) }}</span>
                  </div>
                </div>
              </div>
              <div v-if="batchMode" class="ml-2">
                <input type="checkbox" :checked="selectedIds.includes(ct.id)" @change="toggleSelect(ct.id)" class="rounded border-slate-300 text-emerald-500 focus:ring-emerald-500" />
              </div>
            </div>
            
            <!-- 中间：镜像信息 -->
            <div class="mb-3">
              <p class="text-xs text-slate-500 mb-1">镜像</p>
              <code class="text-xs bg-slate-100 px-2 py-1 rounded break-all" :title="ct.image">
                {{ formatImageName(ct.image) }}
              </code>
            </div>
            
            <!-- 端口信息 -->
            <div class="mb-3">
              <p class="text-xs text-slate-500 mb-1">端口映射</p>
              <div class="font-mono text-xs text-slate-600">
                <template v-if="ct.ports && ct.ports.length > 0">
                  <div v-for="port in ct.ports" :key="port" class="truncate">{{ port }}</div>
                </template>
                <template v-else>-</template>
              </div>
            </div>
            
            <!-- 底部：操作按钮 -->
            <div class="flex flex-wrap gap-1 pt-2 border-t border-slate-100">
              <button @click="!ct.isSwarm && containerModalRef?.show(ct)" :disabled="ct.isSwarm" :class="['btn-icon', ct.isSwarm ? 'text-slate-300 cursor-not-allowed' : 'text-violet-600 hover:bg-violet-50']" :title="ct.isSwarm ? '由 Swarm 管理，不支持直接编辑' : '编辑配置'">
                <i class="fas fa-cog text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">配置</span>
              </button>
              <button v-if="ct.state === 'running'" @click="$router.push({ path: '/docker/container/' + ct.id + '/stats' })" class="btn-icon text-indigo-600 hover:bg-indigo-50" title="统计">
                <i class="fas fa-chart-bar text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">统计</span>
              </button>
              <button v-if="ct.state === 'running'" @click="$router.push({ path: '/docker/container/' + ct.id + '/logs' })" class="btn-icon text-slate-600 hover:bg-slate-50" title="日志">
                <i class="fas fa-file-alt text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">日志</span>
              </button>
              <button v-if="ct.state === 'running'" @click="$router.push({ path: '/docker/container/' + ct.id + '/terminal' })" class="btn-icon text-teal-600 hover:bg-teal-50" title="终端">
                <i class="fas fa-terminal text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">终端</span>
              </button>
              <button v-if="ct.state !== 'running'" @click="handleContainerAction(ct, 'start')" class="btn-icon text-emerald-600 hover:bg-emerald-50" title="启动">
                <i class="fas fa-play text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">启动</span>
              </button>
              <button v-if="ct.state === 'running'" @click="handleContainerAction(ct, 'restart')" class="btn-icon text-blue-600 hover:bg-blue-50" title="重启">
                <i class="fas fa-redo text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">重启</span>
              </button>
              <button v-if="ct.state === 'running'" @click="handleContainerAction(ct, 'stop')" class="btn-icon text-amber-600 hover:bg-amber-50" title="停止">
                <i class="fas fa-stop text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">停止</span>
              </button>
              <button @click="handleContainerAction(ct, 'remove')" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
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
          <i class="fab fa-docker text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">暂无容器</p>
        <p class="text-sm text-slate-400">点击「创建容器」开始使用 Docker</p>
      </div>
    </div>

    <ContainerEditModal ref="containerModalRef" @success="loadContainers" />
    <ComposeModal ref="composeModalRef" @success="loadContainers" />
  </div>
</template>
