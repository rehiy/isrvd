<script setup>
import { onMounted, ref } from 'vue'

import api from '@/service/api.js'

const props = defineProps({
  containerId: { type: String, required: true },
  container: { type: Object, required: true }
})

const logLoading = ref(false)
const logContent = ref('')
const logTail = ref('100')

const loadLogs = async () => {
  if (!props.container) return
  logLoading.value = true
  logContent.value = ''
  try {
    const data = await api.containerLogs(props.containerId, logTail.value)
    logContent.value = (data.payload.logs || []).join('\n')
  } catch (e) {
    logContent.value = '加载日志失败'
  }
  logLoading.value = false
}

const refreshLogs = () => loadLogs()

onMounted(() => {
  loadLogs()
})
</script>

<template>
  <div class="p-6">
    <div class="flex items-center justify-between mb-4">
      <div class="flex items-center gap-3">
        <label class="text-sm text-slate-600">显示行数</label>
        <select v-model="logTail" class="w-28 px-3 py-1.5 bg-white border border-slate-200 rounded-lg text-xs text-slate-700 hover:border-slate-300 focus:outline-none focus:border-slate-400">
          <option value="50">50 行</option>
          <option value="100">100 行</option>
          <option value="200">200 行</option>
          <option value="500">500 行</option>
          <option value="1000">1000 行</option>
        </select>
      </div>
      <button @click="refreshLogs" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
        <i class="fas fa-rotate"></i>刷新
      </button>
    </div>

    <div v-if="logLoading" class="flex flex-col items-center justify-center py-12 gap-3 text-slate-400 text-sm">
      <div class="w-8 h-8 spinner"></div>
      <span>加载日志中...</span>
    </div>
    <pre v-else class="bg-slate-900 text-green-400 p-4 rounded-xl overflow-auto max-h-[70vh] text-sm font-mono whitespace-pre-wrap">{{ logContent || '暂无日志' }}</pre>
  </div>
</template>
