<script setup>
import { inject, onMounted, onUnmounted, ref } from 'vue'

import * as ContainerExec from '@/helper/container-exec.js'
import { APP_STATE_KEY } from '@/store/state.js'

const props = defineProps({
  containerId: { type: String, required: true },
  container: { type: Object, required: true }
})

const state = inject(APP_STATE_KEY)

const terminalConnected = ref(false)
const terminalShell = ref('/bin/sh')
const xtermRef = ref(null)

const handleTerminalConnect = () => {
  if (terminalConnected.value || !props.container) return
  terminalConnected.value = true
  ContainerExec.create(xtermRef.value, state.token, props.containerId, terminalShell.value)
}

const handleTerminalDisconnect = () => {
  ContainerExec.destroy()
  if (xtermRef.value) xtermRef.value.innerHTML = ''
  terminalConnected.value = false
}

const handleShellChange = () => {
  if (terminalConnected.value) {
    handleTerminalDisconnect()
    setTimeout(() => handleTerminalConnect(), 100)
  }
}

onMounted(() => {
  setTimeout(() => handleTerminalConnect(), 200)
})

onUnmounted(() => {
  handleTerminalDisconnect()
})
</script>

<template>
  <div class="p-4 md:p-6">
    <!-- 桌面端工具栏 -->
    <div class="hidden md:flex md:items-center justify-between mb-4">
      <div class="flex items-center gap-3">
        <select
          v-model="terminalShell"
          @change="handleShellChange"
          :disabled="terminalConnected"
          class="w-28 px-3 py-1.5 bg-white border border-slate-200 rounded-lg text-xs text-slate-700 hover:border-slate-300 focus:outline-none disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <option value="/bin/sh">/bin/sh</option>
          <option value="/bin/bash">/bin/bash</option>
          <option value="/bin/ash">/bin/ash</option>
        </select>
      </div>
      <button
        @click="terminalConnected ? handleTerminalDisconnect() : handleTerminalConnect()"
        :class="[
          'px-3 py-1.5 rounded-lg text-xs font-medium flex items-center gap-1.5 transition-colors',
          terminalConnected
            ? 'bg-white border border-slate-200 hover:bg-slate-50 text-slate-700'
            : 'bg-emerald-500 hover:bg-emerald-600 text-white'
        ]"
      >
        <i :class="['fas', terminalConnected ? 'fa-plug-circle-xmark' : 'fa-plug']"></i>
        {{ terminalConnected ? '断开连接' : '连接' }}
      </button>
    </div>

    <!-- 移动端工具栏 -->
    <div class="block md:hidden space-y-3 mb-4">
      <div class="flex items-center gap-2">
        <select
          v-model="terminalShell"
          @change="handleShellChange"
          :disabled="terminalConnected"
          class="flex-1 px-3 py-1.5 bg-white border border-slate-200 rounded-lg text-xs text-slate-700 hover:border-slate-300 focus:outline-none disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <option value="/bin/sh">/bin/sh</option>
          <option value="/bin/bash">/bin/bash</option>
          <option value="/bin/ash">/bin/ash</option>
        </select>
        <button
          @click="terminalConnected ? handleTerminalDisconnect() : handleTerminalConnect()"
          :class="[
            'px-3 py-1.5 rounded-lg text-xs font-medium flex items-center gap-1.5 transition-colors min-w-[80px]',
            terminalConnected
              ? 'bg-white border border-slate-200 hover:bg-slate-50 text-slate-700'
              : 'bg-emerald-500 hover:bg-emerald-600 text-white'
          ]"
        >
          <i :class="['fas', terminalConnected ? 'fa-plug-circle-xmark' : 'fa-plug']"></i>
          <span class="ml-1">{{ terminalConnected ? '断开' : '连接' }}</span>
        </button>
      </div>
    </div>

    <div class="bg-slate-900 rounded-xl p-3 md:p-4 min-h-[400px] md:min-h-[500px]">
      <div ref="xtermRef" class="h-full rounded-lg overflow-hidden min-h-[360px] md:min-h-[480px]"></div>
    </div>
  </div>
</template>
