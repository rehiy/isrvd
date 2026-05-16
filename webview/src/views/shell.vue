<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import * as ShellTerminal from '@/helper/shell'

import { usePortal } from '@/stores'

@Component
class Shell extends Vue {
    portal = usePortal()

    // ─── Refs ───
    @Ref readonly xtermRef!: HTMLDivElement

    // ─── 数据属性 ───
    shellType = 'bash'
    connected = false

    // ─── 方法 ───
    handleConnect() {
        if (this.connected) return
        this.connected = true
        ShellTerminal.create(this.xtermRef, this.portal.token || '', this.shellType)
    }

    handleDisconnect() {
        ShellTerminal.destroy()
        this.xtermRef.innerHTML = ''
        this.connected = false
    }

    // ─── 生命周期 ───
    unmounted() {
        ShellTerminal.destroy()
    }
}

export default toNative(Shell)
</script>

<template>
  <div class="h-[calc(100vh-100px)]">
    <div class="h-full card flex flex-col overflow-hidden">
      <!-- Toolbar Bar -->
      <div class="bg-slate-50 border-b border-slate-200 px-4 md:px-6 py-3">
        <!-- 桌面端工具栏 -->
        <div class="hidden md:flex md:items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-slate-700 flex items-center justify-center">
              <i class="fas fa-terminal text-white text-sm"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">Shell 终端</h1>
              <p class="text-xs text-slate-500">通过 Web 终端连接到远程服务器</p>
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
              :class="[
                'px-3 py-1.5 rounded-lg text-xs font-medium flex items-center gap-1.5 transition-colors',
                connected 
                  ? 'bg-white border border-slate-200 hover:bg-slate-50 text-slate-700' 
                  : 'bg-primary-500 hover:bg-primary-600 text-white'
              ]" 
              @click="connected ? handleDisconnect() : handleConnect()"
            >
              <i :class="['fas', connected ? 'fa-plug-circle-xmark' : 'fa-plug']"></i>
              {{ connected ? '断开连接' : '连接' }}
            </button>
          </div>
        </div>

        <!-- 移动端工具栏 -->
        <div class="block md:hidden">
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-3 min-w-0 flex-1">
              <div class="w-9 h-9 rounded-lg bg-slate-700 flex items-center justify-center flex-shrink-0">
                <i class="fas fa-terminal text-white text-sm"></i>
              </div>
              <div class="min-w-0">
                <h1 class="text-lg font-semibold text-slate-800 truncate">Shell 终端</h1>
                <p class="text-xs text-slate-500 truncate">Web 终端连接</p>
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
              :class="[
                'px-3 py-1.5 rounded-lg text-xs font-medium flex items-center gap-1.5 transition-colors justify-center',
                connected 
                  ? 'bg-white border border-slate-200 hover:bg-slate-50 text-slate-700' 
                  : 'bg-primary-500 hover:bg-primary-600 text-white'
              ]" 
              @click="connected ? handleDisconnect() : handleConnect()"
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
