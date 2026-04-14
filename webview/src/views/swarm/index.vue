<script setup>
import { ref } from 'vue'

import SwarmNodes from './nodes.vue'
import SwarmOverview from './overview.vue'
import SwarmServices from './services.vue'
import SwarmTasks from './tasks.vue'

const activeTab = ref('overview')

const overviewRef = ref(null)
const nodesRef    = ref(null)
const servicesRef = ref(null)
const tasksRef    = ref(null)

// 供 tasks.vue 过滤服务下拉框使用
const serviceList = ref([])

const switchTab = (tab) => {
  activeTab.value = tab
  if (tab === 'overview') overviewRef.value?.loadInfo()
  else if (tab === 'nodes') nodesRef.value?.loadNodes()
  else if (tab === 'services') {
    servicesRef.value?.loadServices()
  } else if (tab === 'tasks') {
    // 切换到任务时同步一次服务列表，供过滤下拉框使用
    servicesRef.value?.loadServices().then?.(() => {
      serviceList.value = servicesRef.value?.services ?? []
    })
    tasksRef.value?.loadTasks()
  }
}
</script>

<template>
  <div>
    <div class="card mb-4">
      <!-- Toolbar -->
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-6 py-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-cyan-600 flex items-center justify-center">
              <i class="fas fa-circle-nodes text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">Docker Swarm</h1>
              <p class="text-xs text-slate-500">集群节点、服务与任务管理</p>
            </div>
          </div>
          <!-- Tab 切换 -->
          <div class="flex items-center gap-1 bg-slate-100 p-1 rounded-lg">
            <button
              v-for="tab in [
                { key: 'overview',  label: '概览',  icon: 'fa-tachometer-alt' },
                { key: 'nodes',     label: '节点',  icon: 'fa-server' },
                { key: 'services',  label: '服务',  icon: 'fa-cubes' },
                { key: 'tasks',     label: '任务',  icon: 'fa-tasks' },
              ]"
              :key="tab.key"
              @click="switchTab(tab.key)"
              :class="[
                'px-3 py-1.5 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5',
                activeTab === tab.key ? 'bg-white text-cyan-600 shadow-sm' : 'text-slate-500 hover:text-slate-700'
              ]"
            >
              <i :class="['fas', tab.icon]"></i>{{ tab.label }}
            </button>
          </div>
        </div>
      </div>

      <!-- 概览 -->
      <SwarmOverview
        v-show="activeTab === 'overview'"
        ref="overviewRef"
        @switch-tab="switchTab"
      />

      <!-- 节点 -->
      <SwarmNodes
        v-show="activeTab === 'nodes'"
        ref="nodesRef"
      />

      <!-- 服务 -->
      <SwarmServices
        v-show="activeTab === 'services'"
        ref="servicesRef"
      />

      <!-- 任务 -->
      <SwarmTasks
        v-show="activeTab === 'tasks'"
        ref="tasksRef"
        :services="serviceList"
      />
    </div>
  </div>
</template>
