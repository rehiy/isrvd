<script lang="ts">
import { Component, Inject, Ref, Vue, toNative } from 'vue-facing-decorator'

import { APP_STATE_KEY } from '@/store/state'

import ApisixOverview from '@/views/apisix/overview.vue'
import DockerOverview from '@/views/docker/overview.vue'
import SwarmOverview from '@/views/swarm/overview.vue'
import SystemOverview from '@/views/system/overview.vue'

@Component({
    components: { ApisixOverview, DockerOverview, SwarmOverview, SystemOverview }
})
class Overview extends Vue {
    @Inject({ from: APP_STATE_KEY }) readonly state!: any

    // ─── Refs ───
    @Ref readonly dockerRef!: InstanceType<typeof DockerOverview>
    @Ref readonly swarmRef!: InstanceType<typeof SwarmOverview>
    @Ref readonly apisixRef!: InstanceType<typeof ApisixOverview>
    @Ref readonly systemRef!: InstanceType<typeof SystemOverview>

    // ─── 方法 ───
    refreshAll() {
        this.systemRef?.load()
        if (this.state.serviceAvailability.docker) {
            this.dockerRef?.load()
        }
        if (this.state.serviceAvailability.swarm) {
            this.swarmRef?.load()
        }
        if (this.state.serviceAvailability.apisix) {
            this.apisixRef?.load()
        }
    }

    // ─── 生命周期 ───
    mounted() {
        this.refreshAll()
    }

    unmounted() {
        this.systemRef?.stopPoll?.()
    }
}

export default toNative(Overview)
</script>

<template>
  <div>
    <div class="card mb-4">
      <!-- 页面标题栏 -->
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-6 py-3 flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="w-9 h-9 rounded-lg bg-primary-500 flex items-center justify-center">
            <i class="fas fa-gauge-high text-white"></i>
          </div>
          <div>
          <h1 class="text-lg font-semibold text-slate-800">概览</h1>
          <p class="text-xs text-slate-500">服务状态总览</p>
          </div>
        </div>
        <button @click="refreshAll" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
          <i class="fas fa-rotate"></i>刷新
        </button>
      </div>

      <!-- 系统信息区块 -->
      <div class="p-6 border-b border-slate-100">
        <div class="flex items-center gap-2 mb-4">
          <i class="fas fa-server text-slate-500 text-lg"></i>
          <h2 class="text-base font-semibold text-slate-700">系统信息</h2>
        </div>
        <SystemOverview ref="systemRef" />
      </div>

      <!-- Docker 概览区块 -->
      <div v-if="state.serviceAvailability.docker" class="p-6 border-b border-slate-100">
        <div class="flex items-center gap-2 mb-4">
          <i class="fab fa-docker text-blue-500 text-lg"></i>
          <h2 class="text-base font-semibold text-slate-700">Docker 服务</h2>
        </div>
        <DockerOverview ref="dockerRef" />
      </div>

      <!-- Swarm 概览区块 -->
      <div v-if="state.serviceAvailability.swarm" class="p-6 border-b border-slate-100">
        <div class="flex items-center gap-2 mb-4">
          <i class="fas fa-circle-nodes text-cyan-600 text-lg"></i>
          <h2 class="text-base font-semibold text-slate-700">Swarm 集群</h2>
        </div>
        <SwarmOverview ref="swarmRef" />
      </div>

      <!-- APISIX 概览区块 -->
      <div v-if="state.serviceAvailability.apisix" class="p-6">
        <div class="flex items-center gap-2 mb-4">
          <i class="fas fa-route text-orange-500 text-lg"></i>
          <h2 class="text-base font-semibold text-slate-700">APISIX 网关</h2>
        </div>
        <ApisixOverview ref="apisixRef" />
      </div>
    </div>
  </div>
</template>
