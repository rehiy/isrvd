<script lang="ts">
import { Component, Inject, Ref, Vue, toNative } from 'vue-facing-decorator'

import * as ContainerExec from '@/helper/container-exec'
import type { ContainerInfo } from '@/service/types'
import { APP_ACTIONS_KEY, APP_STATE_KEY } from '@/store/state'
import type { AppActions, AppState } from '@/store/state'

import ContainerNav from '@/views/docker/widget/container-nav.vue'

@Component({
    components: { ContainerNav }
})
class ContainerTerminal extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions
    @Inject({ from: APP_STATE_KEY }) readonly state!: AppState

    // ─── Refs ───
    @Ref readonly xtermRef!: HTMLDivElement

    // ─── 数据属性 ───
    container: ContainerInfo | null = null
    terminalConnected = false
    terminalShell = '/bin/sh'

    get containerId() {
        return this.$route.params.id as string
    }

    // ─── 方法 ───
    onContainerLoaded(ct: ContainerInfo) {
        this.container = ct
        setTimeout(() => this.handleTerminalConnect(), 200)
    }

    handleTerminalConnect() {
        if (this.terminalConnected || !this.container) return
        this.terminalConnected = true
        ContainerExec.create(this.xtermRef, this.state.token ?? '', this.containerId, this.terminalShell)
    }

    handleTerminalDisconnect() {
        ContainerExec.destroy()
        if (this.xtermRef) this.xtermRef.innerHTML = ''
        this.terminalConnected = false
    }

    handleShellChange() {
        if (this.terminalConnected) {
            this.handleTerminalDisconnect()
            setTimeout(() => this.handleTerminalConnect(), 100)
        }
    }

    // ─── 生命周期 ───
    unmounted() {
        this.handleTerminalDisconnect()
    }
}

export default toNative(ContainerTerminal)
</script>

<template>
  <div>
    <div class="card mb-4">
      <ContainerNav :container-id="containerId" @loaded="onContainerLoaded" />
      <!-- 内容区域 -->
      <div class="p-4 md:p-6">
        <!-- 桌面端工具栏 -->
        <div class="hidden md:flex md:items-center justify-between mb-4">
          <div class="flex items-center gap-3">
            <select v-model="terminalShell" @change="handleShellChange" :disabled="terminalConnected" class="w-28 px-3 py-1.5 bg-white border border-slate-200 rounded-lg text-xs text-slate-700 hover:border-slate-300 focus:outline-none disabled:opacity-50 disabled:cursor-not-allowed">
              <option value="/bin/sh">/bin/sh</option>
              <option value="/bin/bash">/bin/bash</option>
              <option value="/bin/ash">/bin/ash</option>
            </select>
          </div>
          <button @click="terminalConnected ? handleTerminalDisconnect() : handleTerminalConnect()" :class="['px-3 py-1.5 rounded-lg text-xs font-medium flex items-center gap-1.5 transition-colors', terminalConnected ? 'bg-white border border-slate-200 hover:bg-slate-50 text-slate-700' : 'bg-emerald-500 hover:bg-emerald-600 text-white']">
            <i :class="['fas', terminalConnected ? 'fa-plug-circle-xmark' : 'fa-plug']"></i>
            {{ terminalConnected ? '断开连接' : '连接' }}
          </button>
        </div>
        <!-- 移动端工具栏 -->
        <div class="block md:hidden space-y-3 mb-4">
          <div class="flex items-center gap-2">
            <select v-model="terminalShell" @change="handleShellChange" :disabled="terminalConnected" class="flex-1 px-3 py-1.5 bg-white border border-slate-200 rounded-lg text-xs text-slate-700 hover:border-slate-300 focus:outline-none disabled:opacity-50 disabled:cursor-not-allowed">
              <option value="/bin/sh">/bin/sh</option>
              <option value="/bin/bash">/bin/bash</option>
              <option value="/bin/ash">/bin/ash</option>
            </select>
            <button @click="terminalConnected ? handleTerminalDisconnect() : handleTerminalConnect()" :class="['px-3 py-1.5 rounded-lg text-xs font-medium flex items-center gap-1.5 transition-colors min-w-[80px]', terminalConnected ? 'bg-white border border-slate-200 hover:bg-slate-50 text-slate-700' : 'bg-emerald-500 hover:bg-emerald-600 text-white']">
              <i :class="['fas', terminalConnected ? 'fa-plug-circle-xmark' : 'fa-plug']"></i>
              <span class="ml-1">{{ terminalConnected ? '断开' : '连接' }}</span>
            </button>
          </div>
        </div>
        <div class="bg-slate-900 rounded-xl p-3 md:p-4 min-h-[400px] md:min-h-[500px]">
          <div ref="xtermRef" class="h-full rounded-lg overflow-hidden min-h-[360px] md:min-h-[480px]"></div>
        </div>
      </div>
    </div>
  </div>
</template>
