<script setup>
import { inject, ref } from 'vue'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

const actions = inject(APP_ACTIONS_KEY)

const info = ref(null)
const loading = ref(false)

const statCards = [
  { key: 'routes',    label: '路由总数',   icon: 'fa-route',       bgColor: 'bg-orange-500' },
  { key: 'consumers', label: '消费者总数', icon: 'fa-user-tag',    bgColor: 'bg-amber-500' },
  { key: 'whitelist', label: '白名单授权', icon: 'fa-shield-halved', bgColor: 'bg-emerald-500' },
]

const load = async () => {
  loading.value = true
  try {
    const [routesRes, consumersRes, whitelistRes] = await Promise.all([
      api.apisixListRoutes(),
      api.apisixListConsumers(),
      api.apisixGetWhitelist(),
    ])
    const routes = routesRes.payload || []
    const consumers = consumersRes.payload || []
    const whitelist = whitelistRes.payload || []
    info.value = {
      routes: routes.length,
      consumers: consumers.length,
      whitelist: whitelist.length,
    }
  } catch (e) {
    actions.showNotification('error', '获取 APISIX 信息失败')
    info.value = null
  }
  loading.value = false
}

defineExpose({ load })
</script>

<template>
  <div>
    <div v-if="loading" class="flex items-center justify-center py-10">
      <div class="w-8 h-8 spinner mr-2"></div>
      <span class="text-slate-400 text-sm">加载中...</span>
    </div>

    <div v-else-if="info" class="grid grid-cols-2 md:grid-cols-3 gap-3">
      <div
        v-for="card in statCards"
        :key="card.key"
        class="rounded-xl border border-slate-200 bg-white p-4 hover:shadow-md transition-shadow"
      >
        <div class="flex flex-col items-center gap-2 text-center">
          <div :class="['w-10 h-10 rounded-xl flex items-center justify-center', card.bgColor]">
            <i :class="['fas', card.icon, 'text-white']"></i>
          </div>
          <p class="text-2xl font-bold text-slate-800">{{ info[card.key] ?? 0 }}</p>
          <p class="text-xs text-slate-500 leading-tight">{{ card.label }}</p>
        </div>
      </div>
    </div>

    <div v-else class="flex items-center gap-3 py-6 px-4 rounded-xl bg-slate-50">
      <i class="fas fa-route text-2xl text-slate-300"></i>
      <div>
        <p class="text-sm font-medium text-slate-600">无法获取 APISIX 信息</p>
        <p class="text-xs text-slate-400">请确认 APISIX 服务是否正常运行</p>
      </div>
    </div>
  </div>
</template>
