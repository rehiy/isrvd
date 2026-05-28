<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { SSHHostInfo } from '@/service/types'

import * as SSHTerminal from '@/helper/ssh'

@Component
class SSHTerminalPage extends Vue {
    portal = usePortal()

    @Ref readonly xtermRef!: HTMLDivElement

    // ─── 数据属性 ───
    host: SSHHostInfo | null = null
    connected = false

    // ─── 计算属性 ───
    get hostId() {
        return this.$route.params.id as string
    }

    // ─── 方法 ───
    async loadHost() {
        try {
            const res = await api.sshHost(this.hostId)
            this.host = res.payload || null
        } catch {
            this.portal.showNotification('error', '加载主机信息失败')
        }
    }

    handleConnect() {
        if (this.connected) return
        this.connected = true
        SSHTerminal.create(this.xtermRef, this.portal.token || '', this.hostId)
    }

    handleDisconnect() {
        SSHTerminal.destroy()
        this.xtermRef.innerHTML = ''
        this.connected = false
    }

    // ─── 生命周期 ───
    async mounted() {
        await this.loadHost()
        this.handleConnect()
    }

    unmounted() {
        SSHTerminal.destroy()
        this.connected = false
    }
}

export default toNative(SSHTerminalPage)
</script>

<template>
  <div class="terminal-page">
    <div class="h-full card flex flex-col overflow-hidden">
      <!-- Toolbar -->
      <div class="card-toolbar">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="page-icon bg-teal-600">
              <i class="fas fa-terminal text-white text-sm"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">{{ host?.name || 'SSH 终端' }}</h1>
              <p class="text-xs text-slate-500">{{ host ? `${host.user}@${host.addr}` : '加载中...' }}</p>
            </div>
          </div>
          <div class="flex items-center gap-2 flex-shrink-0">
            <button v-if="!connected" class="btn btn-emerald" @click="handleConnect()">
              <i class="fas fa-plug"></i>连接
            </button>
            <button v-else class="btn btn-secondary" @click="handleDisconnect()">
              <i class="fas fa-plug-circle-xmark"></i>断开
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="page-icon bg-teal-600 flex-shrink-0">
              <i class="fas fa-terminal text-white text-sm"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">{{ host?.name || 'SSH 终端' }}</h1>
              <p class="text-xs text-slate-500 truncate">{{ host ? `${host.user}@${host.addr}` : '加载中...' }}</p>
            </div>
          </div>
          <div class="flex items-center gap-2 flex-shrink-0">
            <button v-if="!connected" class="btn btn-emerald w-9 h-9 !px-0" title="连接" @click="handleConnect()">
              <i class="fas fa-plug text-sm"></i>
            </button>
            <button v-else class="btn btn-secondary w-9 h-9 !px-0" title="断开连接" @click="handleDisconnect()">
              <i class="fas fa-plug-circle-xmark text-sm"></i>
            </button>
          </div>
        </div>
      </div>

      <!-- 终端区域 -->
      <div class="terminal-body">
        <div class="terminal-pane">
          <div ref="xtermRef" class="h-full rounded-lg overflow-hidden"></div>
        </div>
      </div>
    </div>
  </div>
</template>
