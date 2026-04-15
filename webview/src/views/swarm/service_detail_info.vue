<script setup>
import { defineProps } from 'vue'
import { formatTime } from '@/helper/utils.js'

const props = defineProps({
  serviceData: { type: Object, required: true }
})
</script>

<template>
  <div class="p-6 space-y-6 text-sm">
    <!-- 基本信息 -->
    <div>
      <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">基本信息</h2>
      <div class="grid grid-cols-2 gap-3">
        <div class="col-span-2">
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">服务 ID</label>
          <code class="block px-3 py-2 bg-slate-50 rounded-lg text-xs font-mono text-slate-700 break-all">{{ serviceData.id }}</code>
        </div>
        <div class="col-span-2">
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">镜像</label>
          <code class="block px-3 py-2 bg-slate-50 rounded-lg text-xs font-mono text-slate-700 break-all">{{ serviceData.image }}</code>
        </div>
        <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">模式</label>
          <div class="px-3 py-2 bg-slate-50 rounded-lg">
            <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-600 capitalize">{{ serviceData.mode }}</span>
          </div>
        </div>
        <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">副本</label>
          <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">
            <span class="text-emerald-600 font-medium">{{ serviceData.runningTasks }}</span>
            <span v-if="serviceData.mode === 'replicated'" class="text-slate-400"> / {{ serviceData.replicas ?? '?' }}</span>
            <span v-else class="text-slate-400"> 运行中</span>
          </div>
        </div>
        <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">创建时间</label>
          <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ formatTime(serviceData.createdAt) }}</div>
        </div>
        <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">更新时间</label>
          <div class="px-3 py-2 bg-slate-50 rounded-lg text-slate-700">{{ formatTime(serviceData.updatedAt) }}</div>
        </div>
      </div>
    </div>

    <!-- 端口 -->
    <div v-if="serviceData.ports && serviceData.ports.length > 0">
      <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">端口映射</h2>
      <div class="border border-slate-200 rounded-lg divide-y divide-slate-100">
        <div v-for="(p, idx) in serviceData.ports" :key="idx" class="px-3 py-2 flex items-center gap-3">
          <code class="text-xs font-mono text-emerald-700 font-medium">{{ p.publishedPort }}</code>
          <i class="fas fa-arrow-right text-slate-300 text-xs"></i>
          <code class="text-xs font-mono text-slate-600">{{ p.targetPort }}/{{ p.protocol }}</code>
          <span class="ml-auto text-xs text-slate-400 capitalize">{{ p.publishMode }}</span>
        </div>
      </div>
    </div>

    <!-- 网络 -->
    <div v-if="serviceData.networks && serviceData.networks.length > 0">
      <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">网络</h2>
      <div class="flex flex-wrap gap-1.5">
        <span v-for="n in serviceData.networks" :key="n" class="inline-flex items-center px-2 py-1 rounded-lg text-xs font-medium bg-blue-50 text-blue-700">{{ n }}</span>
      </div>
    </div>

    <!-- 挂载 -->
    <div v-if="serviceData.mounts && serviceData.mounts.length > 0">
      <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">挂载</h2>
      <div class="border border-slate-200 rounded-lg divide-y divide-slate-100">
        <div v-for="(mt, idx) in serviceData.mounts" :key="idx" class="px-3 py-2">
          <div class="flex items-center gap-2 text-xs font-mono">
            <span class="text-slate-400 capitalize">{{ mt.type }}</span>
            <code class="text-slate-700">{{ mt.source || '(匿名)' }}</code>
            <i class="fas fa-arrow-right text-slate-300"></i>
            <code class="text-slate-700">{{ mt.target }}</code>
            <span v-if="mt.readOnly" class="ml-auto text-amber-600">只读</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 环境变量 -->
    <div v-if="serviceData.env && serviceData.env.length > 0">
      <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">环境变量</h2>
      <div class="border border-slate-200 rounded-lg divide-y divide-slate-100">
        <div v-for="(env, idx) in serviceData.env" :key="idx" class="px-3 py-1.5">
          <code class="text-xs font-mono text-slate-600">{{ env }}</code>
        </div>
      </div>
    </div>

    <!-- 启动参数 -->
    <div v-if="serviceData.args && serviceData.args.length > 0">
      <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">启动参数</h2>
      <code class="block px-3 py-2 bg-slate-50 rounded-lg text-xs font-mono text-slate-700">{{ serviceData.args.join(' ') }}</code>
    </div>

    <!-- 约束 -->
    <div v-if="serviceData.constraints && serviceData.constraints.length > 0">
      <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">调度约束</h2>
      <div class="border border-slate-200 rounded-lg divide-y divide-slate-100">
        <div v-for="(c, idx) in serviceData.constraints" :key="idx" class="px-3 py-1.5">
          <code class="text-xs font-mono text-slate-600">{{ c }}</code>
        </div>
      </div>
    </div>

    <!-- Labels -->
    <div v-if="serviceData.labels && Object.keys(serviceData.labels).length > 0">
      <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Labels</h2>
      <div class="border border-slate-200 rounded-lg divide-y divide-slate-100">
        <div v-for="(val, key) in serviceData.labels" :key="key" class="px-3 py-1.5 flex gap-2">
          <code class="text-xs font-mono text-blue-600 shrink-0">{{ key }}</code>
          <span class="text-slate-400">=</span>
          <code class="text-xs font-mono text-slate-600 break-all">{{ val }}</code>
        </div>
      </div>
    </div>
  </div>
</template>
