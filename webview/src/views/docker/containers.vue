<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { DockerContainerInfo } from '@/service/types'
import { COMPOSE_PROJECT_LABEL, COMPOSE_SERVICE_LABEL } from '@/service/types/docker'

import { formatTime } from '@/helper/utils'

import PageSearch from '@/component/page-search.vue'

import ContainerCreateModal from './widget/container-create-modal.vue'
import ContainerEditModal from './widget/container-edit-modal.vue'

@Component({
    components: { PageSearch, ContainerCreateModal, ContainerEditModal }
})
class Containers extends Vue {
    portal = usePortal()

    // ─── Refs ───
    @Ref readonly containerCreateModalRef!: InstanceType<typeof ContainerCreateModal>
    @Ref readonly containerEditModalRef!: InstanceType<typeof ContainerEditModal>

    // ─── 数据属性 ───
    containers: DockerContainerInfo[] = []
    loading = false
    showAll = false
    searchText = ''

    get filteredContainers() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.containers
        return this.containers.filter((container: DockerContainerInfo) => {
            const ports = container.ports?.join(' ') || ''
            return (
                (container.name || '').toLowerCase().includes(keyword) ||
                container.id.toLowerCase().includes(keyword) ||
                (container.image || '').toLowerCase().includes(keyword) ||
                (container.labels?.[COMPOSE_PROJECT_LABEL] || '').toLowerCase().includes(keyword) ||
                (container.labels?.[COMPOSE_SERVICE_LABEL] || '').toLowerCase().includes(keyword) ||
                (container.state || '').toLowerCase().includes(keyword) ||
                (container.status || '').toLowerCase().includes(keyword) ||
                ports.toLowerCase().includes(keyword)
            )
        })
    }

    readonly actionConfigs: Record<string, { icon: string; iconColor: string; title: string; confirmText: string; danger?: boolean }> = {
        start: { icon: 'fa-play', iconColor: 'emerald', title: '启动容器', confirmText: '启动' },
        stop: { icon: 'fa-stop', iconColor: 'amber', title: '停止容器', confirmText: '停止' },
        restart: { icon: 'fa-rotate', iconColor: 'blue', title: '重启容器', confirmText: '重启' },
        remove: { icon: 'fa-trash', iconColor: 'red', title: '删除容器', confirmText: '删除', danger: true },
        pause: { icon: 'fa-pause', iconColor: 'amber', title: '暂停容器', confirmText: '暂停' },
        unpause: { icon: 'fa-play', iconColor: 'emerald', title: '恢复容器', confirmText: '恢复' }
    }

    // ─── 方法 ───
    async loadContainers() {
        this.loading = true
        try {
            const res = await api.dockerContainerList(this.showAll)
            this.containers = res.payload || []
        } catch {
            this.portal.showNotification('error', '加载容器列表失败')
        } finally {
            this.loading = false
        }
    }

    handleContainerAction(container: DockerContainerInfo, action: string) {
        const config = this.actionConfigs[action]
        if (!config) return
        // 获取显示名称：优先名称，其次短 ID
        const displayName = container.name || container.id.substring(0, 12)
        this.portal.showConfirm({
            title: config.title,
            message: `确定要${config.confirmText}容器 <strong class="text-slate-900">${displayName}</strong> 吗？`,
            icon: config.icon,
            iconColor: config.iconColor,
            confirmText: `确认${config.confirmText}`,
            danger: config.danger,
            onConfirm: async () => {
                await api.dockerContainerAction(container.id, action)
                this.portal.showNotification('success', `容器 ${config.confirmText} 成功`)
                this.loadContainers()
            }
        })
    }

    createContainerModal() {
        this.containerCreateModalRef?.show()
    }

    formatImageName(image: string) {
        if (!image) return ''
        // 去掉 registry host（含端口）前缀，只保留 name:tag 部分
        return image.replace(/^[^/]+\.[^/]+\//, '').replace(/^[^/]+:[0-9]+\//, '')
    }

    isCompose(container: DockerContainerInfo) {
        return !!container.labels?.[COMPOSE_PROJECT_LABEL]
    }

    composeBadgeTitle(container: DockerContainerInfo) {
        const project = container.labels?.[COMPOSE_PROJECT_LABEL]
        if (!project) return ''
        const service = container.labels?.[COMPOSE_SERVICE_LABEL]
        return `Compose: ${project}${service ? ` / ${service}` : ''}`
    }

    composeEditTitle(container: DockerContainerInfo) {
        return container.isSwarm
            ? '由 Swarm 管理，不支持直接编辑'
            : (this.isCompose(container) ? '编辑 Compose 项目配置' : '编辑配置')
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
    <div class="card">
      <div class="card-toolbar">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="page-icon bg-emerald-500">
              <i class="fas fa-cube text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">容器管理</h1>
              <p class="text-xs text-slate-500">管理 Docker 容器的生命周期与运行状态</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <PageSearch v-model="searchText" search-key="docker-containers" placeholder="搜索容器名称、ID、镜像或端口..." focus-color="emerald" type-to-search />
            <div class="flex gap-1 bg-slate-100 p-1 rounded-lg">
              <button :class="['tab-btn', !showAll ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="showAll = false; loadContainers()">
                <i class="fas fa-play"></i><span>运行中</span>
              </button>
              <button :class="['tab-btn', showAll ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="showAll = true; loadContainers()">
                <i class="fas fa-layer-group"></i><span>全部</span>
              </button>
            </div>
            <button class="btn btn-secondary" @click="loadContainers()">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button v-if="portal.hasPerm('POST /api/docker/container')" class="btn btn-emerald" @click="createContainerModal()">
              <i class="fas fa-plus"></i>新建容器
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="block md:hidden">
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-3 min-w-0 flex-1">
              <div class="page-icon bg-emerald-500">
                <i class="fas fa-cube text-white"></i>
              </div>
              <div class="min-w-0">
                <h1 class="text-lg font-semibold text-slate-800 truncate">容器管理</h1>
                <p class="text-xs text-slate-500 truncate">管理容器生命周期</p>
              </div>
            </div>
            <div class="flex items-center gap-1 flex-shrink-0">
              <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadContainers()">
                <i class="fas fa-rotate text-sm"></i>
              </button>
              <button v-if="portal.hasPerm('POST /api/docker/container')" class="btn btn-emerald w-9 h-9 !px-0" title="新建容器" @click="createContainerModal()">
                <i class="fas fa-plus text-sm"></i>
              </button>
            </div>
          </div>
          <div class="flex justify-center gap-1 bg-slate-100 p-1 rounded-lg">
            <button :class="['tab-btn', !showAll ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="showAll = false; loadContainers()">
              <i class="fas fa-play"></i><span>运行中</span>
            </button>
            <button :class="['tab-btn', showAll ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']" @click="showAll = true; loadContainers()">
              <i class="fas fa-layer-group"></i><span>全部</span>
            </button>
          </div>
        </div>
      </div>
      <div class="mobile-search">
        <PageSearch v-model="searchText" search-key="docker-containers" placeholder="搜索容器名称、镜像或端口..." width-class="w-full" focus-color="emerald" />
      </div>
      <!-- Loading -->
      <div v-if="loading" class="empty-state">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <!-- Container List -->
      <div v-else-if="filteredContainers.length > 0">
        <!-- 桌面端表格视图 -->
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="th">名称</th>
                <th class="w-40 th">状态</th>
                <th class="w-48 th">端口</th>
                <th class="w-28 th">创建时间</th>
                <th class="w-48 th-right">操作</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="ct in filteredContainers" :key="ct.id" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3 max-w-[280px]">
                  <div class="flex items-center gap-2 min-w-0">
                    <div :class="['row-icon', ct.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
                      <i class="fas fa-cube text-white text-sm"></i>
                    </div>
                    <div class="min-w-0">
                      <router-link v-if="portal.hasPerm('GET /api/docker/container/:id')" :to="'/docker/container/' + ct.id" class="font-medium text-slate-800 hover:text-emerald-600 transition-colors truncate block" :title="ct.name || ct.id">{{ ct.name || ct.id }}</router-link>
                      <span v-else class="font-medium text-slate-800 truncate block" :title="ct.name || ct.id">{{ ct.name || ct.id }}</span>
                      <code class="text-xs text-slate-400 truncate block min-w-0 mt-0.5" :title="ct.image">{{ formatImageName(ct.image) }}</code>
                    </div>
                  </div>
                </td>
                <td class="px-4 py-3">
                  <div class="flex flex-wrap gap-1">
                    <span :class="['inline-flex items-center px-1.5 py-0.5 rounded-lg text-[11px] font-medium', ct.state === 'running' ? 'bg-emerald-50 text-emerald-700 border border-emerald-100' : 'bg-slate-100 text-slate-500 border border-slate-200']" :title="ct.status">{{ ct.state }}</span>
                    <span v-if="isCompose(ct)" class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-lg text-[11px] font-medium bg-amber-50 text-amber-700 border border-amber-100" :title="composeBadgeTitle(ct)">
                      <i class="fas fa-layer-group text-[10px]"></i>
                      <span>Compose</span>
                    </span>
                  </div>
                </td>
                <td class="px-4 py-3 font-mono text-xs text-slate-600">
                  <template v-if="ct.ports && ct.ports.length > 0">
                    <div v-for="port in ct.ports" :key="port">{{ port }}</div>
                  </template>
                  <template v-else><span class="text-slate-400">-</span></template>
                </td>
                <td class="px-4 py-3 whitespace-nowrap text-sm text-slate-600">{{ formatTime(new Date(ct.created * 1000).toISOString()) }}</td>
                <td class="px-4 py-3">
                  <div class="flex justify-end items-center gap-1">
                    <button v-if="portal.hasPerm('GET /api/docker/container/:id')" class="btn-icon btn-icon-slate" title="详情" @click="$router.push({ path: '/docker/container/' + ct.id })">
                      <i class="fas fa-circle-info text-xs"></i>
                    </button>
                    <button v-if="ct.state === 'running' && portal.hasPerm('GET /api/docker/container/:id/stats')" class="btn-icon btn-icon-indigo" title="统计" @click="$router.push({ path: '/docker/container/' + ct.id + '/stats' })">
                      <i class="fas fa-chart-line text-xs"></i>
                    </button>
                    <button v-if="portal.hasPerm('GET /api/docker/container/:id/logs')" class="btn-icon btn-icon-slate" title="日志" @click="$router.push({ path: '/docker/container/' + ct.id + '/logs' })">
                      <i class="fas fa-file-lines text-xs"></i>
                    </button>
                    <button v-if="ct.state === 'running' && portal.hasPerm('GET /api/docker/container/:id/exec')" class="btn-icon btn-icon-teal" title="登录终端" @click="$router.push({ path: '/docker/container/' + ct.id + '/exec' })">
                      <i class="fas fa-terminal text-xs"></i>
                    </button>
                    <button v-if="ct.state !== 'running' && portal.hasPerm('POST /api/docker/container/:id/action')" class="btn-icon btn-icon-emerald" title="启动" @click="handleContainerAction(ct, 'start')">
                      <i class="fas fa-play text-xs"></i>
                    </button>
                    <button v-if="!ct.isSelf && ct.state === 'running' && portal.hasPerm('POST /api/docker/container/:id/action')" class="btn-icon btn-icon-blue" title="重启" @click="handleContainerAction(ct, 'restart')">
                      <i class="fas fa-rotate text-xs"></i>
                    </button>
                    <button v-if="!ct.isSelf && ct.state === 'running' && portal.hasPerm('POST /api/docker/container/:id/action')" class="btn-icon btn-icon-amber" title="停止" @click="handleContainerAction(ct, 'stop')">
                      <i class="fas fa-stop text-xs"></i>
                    </button>
                    <button v-if="!ct.isSelf && portal.hasPerm('GET /api/compose/docker/:name') && portal.hasPerm('POST /api/compose/docker/:name/redeploy')" :disabled="ct.isSwarm" :class="['btn-icon', ct.isSwarm ? 'text-slate-300 cursor-not-allowed' : 'btn-icon-blue']" :title="composeEditTitle(ct)" @click="!ct.isSwarm && containerEditModalRef?.show(ct)">
                      <i class="fas fa-pen text-xs"></i>
                    </button>
                    <button v-if="!ct.isSelf && portal.hasPerm('POST /api/docker/container/:id/action')" class="btn-icon btn-icon-red" title="删除" @click="handleContainerAction(ct, 'remove')">
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
          <div v-for="ct in filteredContainers" :key="ct.id" class="card-interactive">
            <!-- 顶部：名称和状态 -->
            <div class="card-info-row">
              <div :class="['list-icon', ct.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
                <i class="fas fa-cube text-white text-base"></i>
              </div>
              <div class="min-w-0">
                <router-link v-if="portal.hasPerm('GET /api/docker/container/:id')" :to="'/docker/container/' + ct.id" class="font-medium text-slate-800 hover:text-emerald-600 transition-colors text-sm truncate block" :title="ct.name || ct.id">{{ ct.name || ct.id }}</router-link>
                <span v-else class="font-medium text-slate-800 text-sm truncate block" :title="ct.name || ct.id">{{ ct.name || ct.id }}</span>
                <code class="text-xs text-slate-400 truncate block min-w-0 mt-0.5" :title="ct.image">{{ formatImageName(ct.image) }}</code>
              </div>
            </div>

            <!-- 状态 -->
            <div class="flex items-center gap-1 flex-wrap mb-3">
              <span :class="['inline-flex items-center px-1.5 py-0.5 rounded-lg text-[11px] font-medium', ct.state === 'running' ? 'bg-emerald-50 text-emerald-700 border border-emerald-100' : 'bg-slate-100 text-slate-500 border border-slate-200']" :title="ct.status">{{ ct.state }}</span>
              <span v-if="isCompose(ct)" class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-lg text-[11px] font-medium bg-amber-50 text-amber-700 border border-amber-100" :title="composeBadgeTitle(ct)">
                <i class="fas fa-layer-group text-[10px]"></i>
                <span>Compose</span>
              </span>
            </div>
            <!-- 创建时间 -->
            <div class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">创建时间</span>
              <span class="text-xs text-slate-500">{{ formatTime(new Date(ct.created * 1000).toISOString()) }}</span>
            </div>

            <!-- 端口信息 -->
            <div v-if="ct.ports && ct.ports.length > 0" class="flex items-start gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">端口</span>
              <div class="flex flex-wrap gap-1">
                <code v-for="port in ct.ports" :key="port" class="inline-flex items-center px-1.5 py-0.5 rounded-lg text-xs font-mono bg-slate-100 text-slate-600">{{ port }}</code>
              </div>
            </div>

            <!-- 底部：操作按钮 -->
            <div class="card-actions">
              <button v-if="portal.hasPerm('GET /api/docker/container/:id')" class="btn-icon btn-icon-slate" title="详情" @click="$router.push({ path: '/docker/container/' + ct.id })">
                <i class="fas fa-circle-info text-xs"></i><span class="text-xs ml-1">详情</span>
              </button>
              <button v-if="portal.hasPerm('GET /api/docker/container/:id/logs')" class="btn-icon btn-icon-slate" title="日志" @click="$router.push({ path: '/docker/container/' + ct.id + '/logs' })">
                <i class="fas fa-file-lines text-xs"></i><span class="text-xs ml-1">日志</span>
              </button>
              <button v-if="ct.state === 'running' && portal.hasPerm('GET /api/docker/container/:id/stats')" class="btn-icon btn-icon-indigo" title="统计" @click="$router.push({ path: '/docker/container/' + ct.id + '/stats' })">
                <i class="fas fa-chart-line text-xs"></i><span class="text-xs ml-1">统计</span>
              </button>
              <button v-if="ct.state === 'running' && portal.hasPerm('GET /api/docker/container/:id/exec')" class="btn-icon btn-icon-teal" title="终端" @click="$router.push({ path: '/docker/container/' + ct.id + '/exec' })">
                <i class="fas fa-terminal text-xs"></i><span class="text-xs ml-1">终端</span>
              </button>
              <button v-if="ct.state !== 'running' && portal.hasPerm('POST /api/docker/container/:id/action')" class="btn-icon btn-icon-emerald" title="启动" @click="handleContainerAction(ct, 'start')">
                <i class="fas fa-play text-xs"></i><span class="text-xs ml-1">启动</span>
              </button>
              <button v-if="!ct.isSelf && ct.state === 'running' && portal.hasPerm('POST /api/docker/container/:id/action')" class="btn-icon btn-icon-blue" title="重启" @click="handleContainerAction(ct, 'restart')">
                <i class="fas fa-rotate text-xs"></i><span class="text-xs ml-1">重启</span>
              </button>
              <button v-if="!ct.isSelf && ct.state === 'running' && portal.hasPerm('POST /api/docker/container/:id/action')" class="btn-icon btn-icon-amber" title="停止" @click="handleContainerAction(ct, 'stop')">
                <i class="fas fa-stop text-xs"></i><span class="text-xs ml-1">停止</span>
              </button>
              <button v-if="!ct.isSelf && portal.hasPerm('GET /api/compose/docker/:name') && portal.hasPerm('POST /api/compose/docker/:name/redeploy')" :disabled="ct.isSwarm" :class="['btn-icon', ct.isSwarm ? 'text-slate-300 cursor-not-allowed' : 'btn-icon-blue']" :title="composeEditTitle(ct)" @click="!ct.isSwarm && containerEditModalRef?.show(ct)">
                <i class="fas fa-pen text-xs"></i><span class="text-xs ml-1">编辑</span>
              </button>
              <button v-if="!ct.isSelf && portal.hasPerm('POST /api/docker/container/:id/action')" class="btn-icon btn-icon-red" title="删除" @click="handleContainerAction(ct, 'remove')">
                <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Empty State -->
      <div v-else class="empty-state">
        <div class="empty-state-icon">
          <i class="fab fa-docker text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">{{ containers.length === 0 ? '暂无容器' : '未找到匹配容器' }}</p>
        <p class="text-sm text-slate-400">{{ containers.length === 0 ? '点击「新建容器」开始创建' : '尝试更换关键词或清空搜索条件' }}</p>
      </div>
    </div>

    <ContainerCreateModal ref="containerCreateModalRef" @success="loadContainers" />
    <ContainerEditModal ref="containerEditModalRef" @success="loadContainers" />
  </div>
</template>
