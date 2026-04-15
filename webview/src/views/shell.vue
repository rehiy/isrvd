<script setup>
import { inject, onUnmounted, ref } from 'vue'

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
    <div class="h-full bg-white rounded-2xl shadow-lg border border-slate-200/60 flex flex-col overflow-hidden">
      <!-- Toolbar Bar -->
      <div class="bg-slate-50 border-b border-slate-200 px-4 md:px-6 py-3">
        <!-- 桌面端工具栏 -->
        <div class="hidden md:flex md:items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 rounded-xl bg-slate-700 flex items-center justify-center">
              <i class="fas fa-terminal text-white text-base"></i>
            </div>
            <div>
              <h3 class="font-semibold text-slate-800 text-base">Shell 终端</h3>
              <p class="text-xs text-slate-500">连接到远程服务器</p>
            </div>
          </div>

          <div class="flex items-center gap-2">
            <select 
              v-model="shellType" 
              :disabled="connected" 
              class="w-28 px-3 py-1.5 bg-white border border-slate-200 rounded-lg text-xs text-slate-700 hover:border-slate-300 focus:outline-none disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <option value="bash">bash</option>
              <option value="sh">sh</option>
              <option value="zsh">zsh</option>
              <option value="fish">fish</option>
              <option value="powershell">powershell</option>
              <option value="cmd">cmd</option>
            </select>

            <button 
              @click="connected ? handleDisconnect() : handleConnect()" 
              :class="[
                'px-3 py-1.5 rounded-lg text-xs font-medium flex items-center gap-1.5 transition-colors',
                connected 
                  ? 'bg-white border border-slate-200 hover:bg-slate-50 text-slate-700' 
                  : 'bg-primary-500 hover:bg-primary-600 text-white'
              ]"
            >
              <i :class="['fas', connected ? 'fa-plug-circle-xmark' : 'fa-plug']"></i>
              {{ connected ? '断开连接' : '连接' }}
            </button>
          </div>
        </div>

        <!-- 移动端工具栏 -->
        <div class="block md:hidden">
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-3">
              <div class="w-8 h-8 rounded-xl bg-slate-700 flex items-center justify-center">
                <i class="fas fa-terminal text-white text-sm"></i>
              </div>
              <div>
                <h3 class="font-semibold text-slate-800 text-sm">Shell 终端</h3>
                <p class="text-xs text-slate-500">连接到远程服务器</p>
              </div>
            </div>
          </div>
          
          <div class="flex flex-col gap-2">
            <select 
              v-model="shellType" 
              :disabled="connected" 
              class="px-3 py-1.5 bg-white border border-slate-200 rounded-lg text-xs text-slate-700 hover:border-slate-300 focus:outline-none disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <option value="bash">bash</option>
              <option value="sh">sh</option>
              <option value="zsh">zsh</option>
              <option value="fish">fish</option>
              <option value="powershell">powershell</option>
              <option value="cmd">cmd</option>
            </select>

            <button 
              @click="connected ? handleDisconnect() : handleConnect()" 
              :class="[
                'px-3 py-1.5 rounded-lg text-xs font-medium flex items-center gap-1.5 transition-colors justify-center',
                connected 
                  ? 'bg-white border border-slate-200 hover:bg-slate-50 text-slate-700' 
                  : 'bg-primary-500 hover:bg-primary-600 text-white'
              ]"
            >
              <i :class="['fas', connected ? 'fa-plug-circle-xmark' : 'fa-plug']"></i>
              <span class="ml-1">{{ connected ? '断开' : '连接' }}</span>
            </button>
          </div>
        </div>
      </div>

      <!-- Terminal -->
      <div class="flex-1 bg-slate-900 p-2 md:p-4">
        <div ref="xtermRef" class="h-full rounded-lg overflow-hidden"></div>
      </div>
    </div>
  </div>
</template>
