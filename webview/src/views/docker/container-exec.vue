<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import type { DockerContainerInfo } from '@/service/types'

import api from '@/service/api'
import * as ContainerExec from '@/helper/container-exec'

import { usePortal } from '@/stores'

@Component
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
    async loadContainer() {
        try {
            const res = await api.dockerContainerList(true)
            this.container = (res.payload || []).find((c: DockerContainerInfo) => c.id === this.containerId) ?? null
            if (!this.container) {
                this.portal.showNotification('error', '容器不存在')
                this.$router.push('/docker/containers')
                return
            }
            setTimeout(() => this.handleTerminalConnect(), 200)
        } catch {
            this.portal.showNotification('error', '加载容器信息失败')
        }
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
    mounted() {
        this.loadContainer()
    }

    unmounted() {
        this.handleTerminalDisconnect()
    }
}

export default toNative(ContainerTerminal)
</script>

<template>
  <div>
    <div class="card mb-4">
      <!-- Toolbar -->
      <div class="card-toolbar">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div :class="['page-icon', container?.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
              <i class="fas fa-terminal text-white text-sm"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">容器终端</h1>
              <p class="text-xs text-slate-500 font-mono truncate max-w-xs">{{ container ? `${container.name || container.id} · ${container.image}` : '加载中...' }}</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <select v-model="terminalShell" :disabled="terminalConnected" class="w-28 select-sm" @change="handleShellChange">
              <option value="/bin/sh">/bin/sh</option>
              <option value="/bin/bash">/bin/bash</option>
              <option value="/bin/ash">/bin/ash</option>
            </select>
            <button v-if="!terminalConnected" type="button" class="btn btn-success" @click="handleTerminalConnect()">
              <i class="fas fa-plug"></i>连接
            </button>
            <button v-else type="button" class="btn btn-secondary" @click="handleTerminalDisconnect()">
              <i class="fas fa-plug-circle-xmark"></i>断开
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div :class="['page-icon', container?.state === 'running' ? 'bg-emerald-400' : 'bg-slate-400']">
              <i class="fas fa-terminal text-white text-sm"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">容器终端</h1>
              <p class="text-xs text-slate-500 font-mono truncate">{{ container ? `${container.name || container.id} · ${container.image}` : '加载中...' }}</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <select v-model="terminalShell" :disabled="terminalConnected" class="w-24 select-sm" @change="handleShellChange">
              <option value="/bin/sh">/bin/sh</option>
              <option value="/bin/bash">/bin/bash</option>
              <option value="/bin/ash">/bin/ash</option>
            </select>
            <button v-if="!terminalConnected" type="button" class="btn btn-success w-9 h-9 !px-0" title="连接" @click="handleTerminalConnect()">
              <i class="fas fa-plug text-sm"></i>
            </button>
            <button v-else type="button" class="btn btn-secondary w-9 h-9 !px-0" title="断开" @click="handleTerminalDisconnect()">
              <i class="fas fa-plug-circle-xmark text-sm"></i>
            </button>
          </div>
        </div>
      </div>

      <!-- 内容区域 -->
      <div class="p-4 md:p-6">
        <div class="bg-slate-900 rounded-xl p-3 md:p-4 min-h-[400px] md:min-h-[500px]">
          <div ref="xtermRef" class="h-full rounded-lg overflow-hidden min-h-[360px] md:min-h-[480px]"></div>
        </div>
      </div>
    </div>
  </div>
</template>
