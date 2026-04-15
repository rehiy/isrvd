<script setup>
import { inject, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

const route = useRoute()
const actions = inject(APP_ACTIONS_KEY)

const serviceId = route.params.id
const logsContent = ref([])
const logsLoading = ref(false)
const logsTail = ref('200')

const loadLogs = async () => {
  logsLoading.value = true
  try {
    const res = await api.swarmServiceLogs(serviceId, logsTail.value)
    logsContent.value = res.payload?.logs || []
  } catch (e) {
    logsContent.value = []
    actions.showNotification('error', '获取日志失败')
  }
  logsLoading.value = false
}

onMounted(() => loadLogs())
</script>

<template>
  <div class="p-6 space-y-3">
    <div class="flex items-center gap-2">
      <label class="text-xs text-slate-500 flex-shrink-0">最近行数</label>
      <select v-model="logsTail" @change="loadLogs()" class="w-24 px-2 py-1 bg-white border border-slate-200 rounded-lg text-xs text-slate-700">
        <option value="50">50</option>
        <option value="100">100</option>
        <option value="200">200</option>
        <option value="500">500</option>
        <option value="1000">1000</option>
      </select>
    </div>
    <div v-if="logsLoading" class="flex flex-col items-center justify-center py-20">
      <div class="w-12 h-12 spinner mb-3"></div>
      <p class="text-slate-500">加载中...</p>
    </div>
    <pre v-else-if="logsContent.length > 0" class="bg-slate-900 text-slate-100 rounded-xl p-4 text-xs font-mono overflow-auto max-h-[600px] whitespace-pre-wrap break-all">{{ logsContent.join('') }}</pre>
    <div v-else class="flex flex-col items-center justify-center py-16">
      <div class="w-16 h-16 rounded-full bg-slate-100 flex items-center justify-center mb-3">
        <i class="fas fa-file-lines text-2xl text-slate-300"></i>
      </div>
      <p class="text-slate-500 text-sm">暂无日志</p>
    </div>
  </div>
</template>
