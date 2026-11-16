<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

@Component
class ContainerTabs extends Vue {
    @Prop({ required: true }) readonly containerId!: string

    portal = usePortal()

    tabs = [
        { name: 'docker-container', label: '详情', icon: 'fa-circle-info', permission: 'GET /api/docker/container/:id' },
        { name: 'docker-container-stats', label: '性能', icon: 'fa-chart-line', permission: 'GET /api/docker/container/:id/stats' },
        { name: 'docker-container-logs', label: '日志', icon: 'fa-file-lines', permission: 'GET /api/docker/container/:id/logs' }
    ]

    get visibleTabs() {
        return this.tabs.filter(tab => this.portal.hasPerm(tab.permission))
    }
}

export default toNative(ContainerTabs)
</script>

<template>
  <div v-if="visibleTabs.length" class="tab-group">
    <router-link v-for="tab in visibleTabs" :key="tab.name" :to="{ name: tab.name, params: { id: containerId } }" :class="['tab-btn', $route.name === tab.name ? 'tab-btn-active text-emerald-600' : 'tab-btn-inactive']">
      <i :class="['fas', tab.icon]"></i><span>{{ $t(tab.label) }}</span>
    </router-link>
  </div>
</template>
