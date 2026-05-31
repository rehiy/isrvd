<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import * as ShellTerminal from '@/helper/shell'

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
      <div class="card-toolbar">
        <!-- 桌面端工具栏 -->
        <div class="hidden md:flex md:items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="page-icon bg-slate-700">
              <i class="fas fa-terminal text-white text-sm"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">Shell 终端</h1>
              <p class="text-xs text-slate-500">通过 Web 终端连接到远程服务器</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <select v-model="shellType" :disabled="connected" class="w-28 select-sm">
              <option value="bash">bash</option>
              <option value="sh">sh</option>
              <option value="zsh">zsh</option>
              <option value="fish">fish</option>
              <option value="powershell">powershell</option>
              <option value="cmd">cmd</option>
            </select>
            <button v-if="!connected" class="btn btn-primary" @click="handleConnect()">
              <i class="fas fa-plug"></i>连接
            </button>
            <button v-else class="btn btn-secondary" @click="handleDisconnect()">
              <i class="fas fa-plug-circle-xmark"></i>断开
            </button>
          </div>
        </div>
        <!-- 移动端工具栏 -->
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="page-icon bg-slate-700">
              <i class="fas fa-terminal text-white text-sm"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">Shell 终端</h1>
              <p class="text-xs text-slate-500 truncate">Web 终端连接</p>
            </div>
          </div>
          <div class="flex items-center gap-2 flex-shrink-0">
            <select v-model="shellType" :disabled="connected" class="w-24 select-sm">
              <option value="bash">bash</option>
              <option value="sh">sh</option>
              <option value="zsh">zsh</option>
              <option value="fish">fish</option>
              <option value="powershell">powershell</option>
              <option value="cmd">cmd</option>
            </select>
            <button v-if="!connected" class="btn btn-primary w-9 h-9 !px-0" title="连接" @click="handleConnect()">
              <i class="fas fa-plug text-sm"></i>
            </button>
            <button v-else class="btn btn-secondary w-9 h-9 !px-0" title="断开连接" @click="handleDisconnect()">
              <i class="fas fa-plug-circle-xmark text-sm"></i>
            </button>
          </div>
        </div>
      </div>

      <!-- 内容区域 -->
      <div class="terminal-pane">
        <div ref="xtermRef" class="h-full rounded-lg overflow-hidden"></div>
      </div>
    </div>
  </div>
</template>
