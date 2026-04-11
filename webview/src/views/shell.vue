<script setup>
import { inject, ref, onUnmounted } from 'vue'

import * as ShellTerminal from '@/helper/shell.js'
import { APP_STATE_KEY } from '@/store/state.js'

const state = inject(APP_STATE_KEY)

const xtermRef = ref(null)
const shellType = ref('bash')
const connected = ref(false)

const handleConnect = () => {
  if (connected.value) return
  connected.value = true
  ShellTerminal.create(xtermRef.value, state.token, shellType.value)
}

const handleDisconnect = () => {
  ShellTerminal.destroy()
  xtermRef.value.innerHTML = ''
  connected.value = false
}

onUnmounted(() => ShellTerminal.destroy())
</script>

<template>
  <div class="h-[calc(100vh-100px)]">
    <div class="h-full card flex flex-col overflow-hidden">
      <!-- Toolbar Bar -->
      <div class="bg-slate-50 border-b border-slate-200 px-6 py-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center space-x-3">
            <div class="w-10 h-10 rounded-xl bg-slate-700 flex items-center justify-center">
              <i class="fas fa-terminal text-white"></i>
            </div>
            <div>
              <h3 class="font-semibold text-slate-800">Shell 终端</h3>
              <p class="text-xs text-slate-500">连接到远程服务器</p>
            </div>
          </div>

          <div class="flex items-center gap-2">
            <select 
              v-model="shellType" 
              :disabled="connected" 
              class="input w-28 text-sm"
            >
              <option value="bash">bash</option>
              <option value="sh">sh</option>
              <option value="zsh">zsh</option>
              <option value="fish">fish</option>
            </select>

            <button 
              @click="connected ? handleDisconnect() : handleConnect()" 
              :class="[
                'btn text-sm',
                connected 
                  ? 'bg-white border border-red-200 text-red-600 hover:bg-red-50' 
                  : 'btn-primary'
              ]"
            >
              <i :class="['fas mr-1.5', connected ? 'fa-plug-circle-xmark' : 'fa-plug']"></i>
              {{ connected ? '断开连接' : '连接' }}
            </button>
          </div>
        </div>
      </div>

      <!-- Terminal -->
      <div class="flex-1 bg-slate-900 p-4">
        <div ref="xtermRef" class="h-full rounded-lg overflow-hidden"></div>
      </div>
    </div>
  </div>
</template>
