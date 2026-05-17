<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import type { DockerContainerInfo } from '@/service/types'

import * as ContainerExec from '@/helper/container-exec'

import { usePortal } from '@/stores'

import ContainerNav from './widget/container-nav.vue'

@Component({
    components: { ContainerNav }
})
class ContainerTerminal extends Vue {
    portal = usePortal()

    // ─── Refs ───
    @Ref readonly xtermRef!: HTMLDivElement

    // ─── 数据属性 ───
    container: DockerContainerInfo | null = null
    terminalConnected = false
    terminalShell = '/bin/sh'

    get containerId() {
        return this.$route.params.id as string
    }

    // ─── 方法 ───
    onContainerLoaded(ct: DockerContainerInfo) {
        this.container = ct
        setTimeout(() => this.handleTerminalConnect(), 200)
    }

    handleTerminalConnect() {
        if (this.terminalConnected || !this.container) return
        this.terminalConnected = true
        ContainerExec.create(this.xtermRef, this.portal.token ?? '', this.containerId, this.terminalShell)
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
            <select v-model="terminalShell" :disabled="terminalConnected" class="w-28 select-sm" @change="handleShellChange">
              <option value="/bin/sh">/bin/sh</option>
              <option value="/bin/bash">/bin/bash</option>
              <option value="/bin/ash">/bin/ash</option>
            </select>
          </div>
          <div class="flex items-center gap-2">
            <button v-if="!terminalConnected" type="button" class="btn btn-sm btn-success" @click="handleTerminalConnect()">
              <i class="fas fa-plug"></i>连接
            </button>
            <button v-else type="button" class="btn btn-sm btn-secondary" @click="handleTerminalDisconnect()">
              <i class="fas fa-plug-circle-xmark"></i>断开连接
            </button>
          </div>
        </div>
        <!-- 移动端工具栏 -->
        <div class="block md:hidden space-y-3 mb-4">
          <div class="flex items-center gap-2">
            <select v-model="terminalShell" :disabled="terminalConnected" class="flex-1 select-sm" @change="handleShellChange">
              <option value="/bin/sh">/bin/sh</option>
              <option value="/bin/bash">/bin/bash</option>
              <option value="/bin/ash">/bin/ash</option>
            </select>
            <div class="flex items-center gap-1">
              <button v-if="!terminalConnected" type="button" class="btn btn-sm btn-success" @click="handleTerminalConnect()">
                <i class="fas fa-plug"></i>连接
              </button>
              <button v-else type="button" class="btn btn-sm btn-secondary" @click="handleTerminalDisconnect()">
                <i class="fas fa-plug-circle-xmark"></i>断开
              </button>
            </div>
          </div>
        </div>
        <div class="bg-slate-900 rounded-xl p-3 md:p-4 min-h-[400px] md:min-h-[500px]">
          <div ref="xtermRef" class="h-full rounded-lg overflow-hidden min-h-[360px] md:min-h-[480px]"></div>
        </div>
      </div>
    </div>
  </div>
</template>
