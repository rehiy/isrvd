<script setup>
import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'
import { computed, inject, onMounted, ref } from 'vue'

const actions = inject(APP_ACTIONS_KEY)
const whitelist = ref([])
const loading = ref(false)
const searchText = ref('')

const filteredWhitelist = computed(() => {
  if (!searchText.value) return whitelist.value
  const s = searchText.value.toLowerCase()
  return whitelist.value.filter(r =>
    (r.name || '').toLowerCase().includes(s) ||
    (r.id || '').toLowerCase().includes(s) ||
    (r.consumers || []).some(c => c.toLowerCase().includes(s))
  )
})

const loadWhitelist = async () => {
  loading.value = true
  try { whitelist.value = (await api.apisixGetWhitelist()).payload || [] }
  catch { actions.showNotification('error', '加载白名单失败') }
  loading.value = false
}

const getRouteUri = (r) => r.uris?.length ? r.uris.join(', ') : (r.uri || '-')
const getRouteHost = (r) => r.hosts?.length ? r.hosts.join(', ') : (r.host || '*')

const revokeConsumer = (route, consumer) => {
  actions.showConfirm({
    title: '撤销白名单',
    message: `确定要将用户 <strong class="text-slate-900">${consumer}</strong> 从路由 <strong class="text-slate-900">${route.name || route.id}</strong> 的白名单中移除吗？`,
    icon: 'fa-user-minus', iconColor: 'red', confirmText: '确认撤销', danger: true,
    onConfirm: async () => {
      await api.apisixRevokeWhitelist(route.id, consumer)
      actions.showNotification('success', '撤销成功')
      loadWhitelist()
    }
  })
}

onMounted(() => { loadWhitelist() })
</script>

<template>
  <div>
    <div class="card mb-4">
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3">
        <div class="flex flex-col md:flex-row md:items-center justify-between gap-3">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-amber-500 flex items-center justify-center"><i class="fas fa-shield-halved text-white"></i></div>
            <div><h1 class="text-lg font-semibold text-slate-800">白名单管理</h1><p class="text-xs text-slate-500">管理路由的 Consumer 白名单</p></div>
          </div>
          <div class="flex flex-col sm:flex-row items-start sm:items-center gap-2">
            <div class="relative">
              <input v-model="searchText" type="text" placeholder="搜索路由或用户..." class="pl-8 pr-3 py-1.5 text-xs border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-amber-500 focus:border-transparent w-full sm:w-48" />
              <i class="fas fa-search absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400 text-xs"></i>
            </div>
            <button @click="loadWhitelist()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors"><i class="fas fa-rotate"></i>刷新</button>
          </div>
        </div>
      </div>
      <div v-if="loading" class="flex flex-col items-center justify-center py-20"><div class="w-12 h-12 spinner mb-3"></div><p class="text-slate-500">加载中...</p></div>
      <div v-else-if="filteredWhitelist.length === 0" class="flex flex-col items-center justify-center py-20">
        <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
          <i class="fas fa-shield-halved text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">暂无白名单数据</p>
        <p class="text-sm text-slate-400">配置路由的 Consumer 白名单后将在此显示</p>
      </div>
      <div v-else class="divide-y divide-slate-100">
        <div v-for="route in filteredWhitelist" :key="route.id" class="px-4 md:px-6 py-4 hover:bg-slate-50/50 transition-colors">
          <div class="flex flex-col md:flex-row md:items-start justify-between gap-2 mb-2">
            <div class="flex-1">
              <div class="font-medium text-sm text-slate-800">{{ route.name || route.id }}</div>
              <div class="flex flex-col sm:flex-row sm:items-center gap-2 mt-1">
                <code class="text-xs bg-slate-100 px-1.5 py-0.5 rounded text-slate-600 break-all">{{ getRouteUri(route) }}</code>
                <span class="text-xs text-slate-400">{{ getRouteHost(route) }}</span>
              </div>
            </div>
            <span class="text-xs text-slate-400">{{ (route.consumers || []).length }} 个用户</span>
          </div>
          <div class="flex flex-wrap gap-1.5">
            <span v-for="consumer in (route.consumers || [])" :key="consumer" class="inline-flex items-center gap-1.5 px-2.5 py-1 bg-amber-50 text-amber-800 rounded-lg text-xs group">
              <i class="fas fa-user text-amber-500 text-[10px]"></i>
              <span class="break-all">{{ consumer }}</span>
              <button @click="revokeConsumer(route, consumer)" class="opacity-0 group-hover:opacity-100 hover:text-red-500 transition-all" title="撤销"><i class="fas fa-xmark text-[10px]"></i></button>
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
