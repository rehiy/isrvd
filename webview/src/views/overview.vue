<script setup>
import { onMounted, ref } from 'vue'

import DockerOverview from '@/views/docker/overview.vue'
import SwarmOverview from '@/views/swarm/overview.vue'

const dockerRef = ref(null)
const swarmRef = ref(null)

const refreshAll = () => {
  dockerRef.value?.load()
  swarmRef.value?.load()
}

onMounted(() => refreshAll())
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
            <h1 class="text-lg font-semibold text-slate-800">整体概览</h1>
            <p class="text-xs text-slate-500">Docker 与 Swarm 集群状态总览</p>
          </div>
        </div>
        <button @click="refreshAll" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
          <i class="fas fa-rotate"></i>刷新
        </button>
      </div>

      <!-- Docker 概览区块 -->
      <div class="p-6 border-b border-slate-100">
        <div class="flex items-center gap-2 mb-4">
          <i class="fab fa-docker text-blue-500 text-lg"></i>
          <h2 class="text-base font-semibold text-slate-700">Docker</h2>
        </div>
        <DockerOverview ref="dockerRef" />
      </div>

      <!-- Swarm 概览区块 -->
      <div class="p-6">
        <div class="flex items-center gap-2 mb-4">
          <i class="fas fa-circle-nodes text-cyan-600 text-lg"></i>
          <h2 class="text-base font-semibold text-slate-700">Swarm 集群</h2>
        </div>
        <SwarmOverview ref="swarmRef" />
      </div>
    </div>
  </div>
</template>
