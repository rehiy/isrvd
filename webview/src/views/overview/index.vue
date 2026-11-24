<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import type { DockerContainerInfo, MonitorHostRecord } from '@/service/types'

import { NODE_LIST_PERM } from '@/helper/node'

import ApisixOverview from './widget/apisix.vue'
import CaddyOverview from './widget/caddy.vue'
import DockerOverview from './widget/docker.vue'
import NodeResources from './widget/nodes.vue'
import SwarmOverview from './widget/swarm.vue'
import SystemOverview from './widget/system.vue'
import SystemUpdater from './widget/updater.vue'
import WorkloadsOverview from './widget/workloads.vue'

@Component({
    components: { SystemOverview, ApisixOverview, CaddyOverview, DockerOverview, SwarmOverview, SystemUpdater, WorkloadsOverview, NodeResources }
})
class Overview extends Vue {
    portal = usePortal()
    currentReading: MonitorHostRecord | null | undefined = undefined
    selfContainer: DockerContainerInfo | null = null

    // ─── Refs ───
    @Ref readonly systemRef!: InstanceType<typeof SystemOverview>
    @Ref readonly apisixRef!: InstanceType<typeof ApisixOverview>
    @Ref readonly caddyRef!: InstanceType<typeof CaddyOverview>
    @Ref readonly dockerRef!: InstanceType<typeof DockerOverview>
    @Ref readonly swarmRef!: InstanceType<typeof SwarmOverview>
    @Ref readonly nodesRef!: InstanceType<typeof NodeResources>
    @Ref readonly workloadsRef!: InstanceType<typeof WorkloadsOverview>

    // ─── 计算属性 ───
    get canViewApisix() {
        return [
            'GET /api/apisix/routes',
            'GET /api/apisix/consumers',
            'GET /api/apisix/upstreams',
            'GET /api/apisix/plugin-configs',
            'GET /api/apisix/ssls',
            'GET /api/apisix/whitelist',
        ].some(permission => this.portal.hasPerm(permission))
    }

    get canViewHost() {
        return this.portal.hasPerm('GET /api/overview/monitor')
    }

    get canViewWorkloads() {
        return this.portal.hasPerm('GET /api/docker/containers')
    }

    get canViewNodes() {
        return this.portal.initialized && this.canViewHost && this.portal.hasPerm(NODE_LIST_PERM)
    }

    get currentNodeName() {
        if (!this.canViewNodes) {
            return ''
        }
        if (!this.portal.currentNodeId) {
            return this.$t('中控')
        }
        return this.portal.nodes.find(node => node.id === this.portal.currentNodeId)?.name ?? ''
    }

    get hasAnyBlock() {
        return (
            this.canViewApisix ||
            this.portal.hasPerm('GET /api/caddy/info') ||
            this.canViewWorkloads ||
            this.portal.hasPerm('GET /api/docker/info') ||
            this.portal.hasPerm('GET /api/swarm/info') ||
            this.canViewHost
        )
    }

    // ─── 方法 ───
    refreshAll() {
        this.systemRef?.load()
        this.nodesRef?.load()
        this.apisixRef?.load()
        this.caddyRef?.load()
        this.workloadsRef?.load()
        this.dockerRef?.load()
        this.swarmRef?.load()
    }
}

export default toNative(Overview)
</script>

<template>
  <div class="page resource-page">
    <!-- 页面标题栏 -->
    <div class="page-toolbar">
      <!-- 桌面端 -->
      <div class="toolbar-desktop">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-primary-500">
            <i class="fas fa-gauge-high text-white"></i>
          </div>
          <div>
            <h1 class="title-text">{{ $t('资源概览') }}</h1>
            <p class="text-xs text-slate-500">{{ $t('主机指标与容器运行状态') }}</p>
          </div>
        </div>
        <button class="btn btn-secondary" @click="refreshAll">
          <i class="fas fa-rotate"></i>{{ $t('刷新') }}
        </button>
      </div>
      <!-- 移动端 -->
      <div class="toolbar-mobile">
        <div class="title-group">
          <div class="page-icon bg-primary-500">
            <i class="fas fa-gauge-high text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="title-text">{{ $t('资源概览') }}</h1>
            <p class="text-xs text-slate-500 truncate">{{ $t('主机指标与容器运行状态') }}</p>
          </div>
        </div>
        <button class="btn btn-secondary btn-square" :title="$t('刷新')" @click="refreshAll">
          <i class="fas fa-rotate text-sm"></i>
        </button>
      </div>
    </div>

    <div class="mx-auto max-w-[100rem] space-y-4 px-4 py-4 sm:px-6 lg:px-7">
      <NodeResources v-if="canViewNodes" ref="nodesRef" :current-reading="currentReading" />

      <!-- 系统信息区块 -->
      <div v-if="canViewHost">
        <SystemUpdater :self-container="canViewWorkloads ? selfContainer : null" />
        <SystemOverview ref="systemRef" :node-name="currentNodeName" @sample="currentReading = $event" />
      </div>

      <section v-if="canViewWorkloads || portal.hasPerm('GET /api/docker/info')" class="card resource-panel resource-service-docker">
        <WorkloadsOverview v-if="canViewWorkloads" ref="workloadsRef" @self-container="selfContainer = $event" />
        <div v-else class="resource-section-heading mb-4">
          <span class="row-icon resource-service-icon"><i class="fab fa-docker"></i></span>
          <h2 class="resource-section-title">{{ $t('Docker 服务') }}</h2>
        </div>
        <div v-if="portal.hasPerm('GET /api/docker/info')" :class="{ 'mt-4 border-t border-slate-200/70 pt-3': canViewWorkloads }">
          <DockerOverview ref="dockerRef" :show-container-counts="!canViewWorkloads" :compact="canViewWorkloads" />
        </div>
      </section>

      <!-- APISIX 概览区块 -->
      <section v-if="canViewApisix" class="card resource-panel resource-service-apisix">
        <div class="resource-section-heading mb-4">
          <span class="row-icon resource-service-icon"><i class="fas fa-route"></i></span>
          <h2 class="resource-section-title">{{ $t('APISIX 网关') }}</h2>
        </div>
        <ApisixOverview ref="apisixRef" />
      </section>

      <!-- Caddy 概览区块 -->
      <section v-if="portal.hasPerm('GET /api/caddy/info')" class="card resource-panel resource-service-caddy">
        <div class="resource-section-heading mb-4">
          <span class="row-icon resource-service-icon"><i class="fas fa-shield"></i></span>
          <h2 class="resource-section-title">{{ $t('Caddy 网关') }}</h2>
        </div>
        <CaddyOverview ref="caddyRef" />
      </section>

      <!-- Swarm 概览区块 -->
      <section v-if="portal.hasPerm('GET /api/swarm/info')" class="card resource-panel resource-service-swarm">
        <div class="resource-section-heading mb-4">
          <span class="row-icon resource-service-icon"><i class="fas fa-circle-nodes"></i></span>
          <h2 class="resource-section-title">{{ $t('Swarm 集群') }}</h2>
        </div>
        <SwarmOverview ref="swarmRef" />
      </section>
    </div>

    <!-- 无任何权限时的空状态 -->
    <div v-if="!hasAnyBlock" class="flex flex-col items-center justify-center py-16 text-slate-400">
      <i class="fas fa-lock text-4xl mb-4"></i>
      <p class="text-sm font-medium">{{ $t('暂无可查看的模块') }}</p>
      <p class="text-xs mt-1">{{ $t('当前账号未分配任何模块权限') }}</p>
    </div>
  </div>
</template>
