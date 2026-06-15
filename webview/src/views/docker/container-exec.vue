<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'
import api from '@/service/api'
import type { DockerContainerInfo } from '@/service/types'
import { wsUrl } from '@/service/axios'
import { TerminalPanel, WsTerminal } from '@/component/terminal'
import type { TerminalAdapter } from '@/component/terminal'

@Component({ components: { TerminalPanel } })
class ContainerTerminal extends Vue {
    portal = usePortal()

    container: DockerContainerInfo | null = null
    adapter: TerminalAdapter | null = null
    shell = '/bin/sh'

    get containerId() { return this.$route.params.id as string }
    get connected() { return this.adapter?.connected ?? false }

    async mounted() {
        await this.loadContainer()
    }

    unmounted() {
        this.adapter?.disconnect()
        this.adapter = null
    }

    async loadContainer() {
        try {
            const res = await api.dockerContainerList(true)
            this.container = (res.payload || []).find((c: DockerContainerInfo) => c.id === this.containerId) ?? null
            if (!this.container) {
                this.portal.showNotification('error', '容器不存在')
                this.$router.push('/docker/containers')
                return
            }
            this.handleConnect()
        } catch {}
    }

    handleConnect() {
        if (this.connected || !this.container) return
        this.adapter = new WsTerminal(wsUrl(`docker/container/${encodeURIComponent(this.containerId)}/exec?token=${this.portal.token ?? ''}&shell=${encodeURIComponent(this.shell)}`))
    }

    handleDisconnect() {
        this.adapter?.disconnect()
        this.adapter = null
    }

    handleShellChange() {
        if (this.connected) {
            this.handleDisconnect()
            this.$nextTick(() => this.handleConnect())
        }
    }
}

export default toNative(ContainerTerminal)
</script>

<template>
  <div class="h-[calc(100vh-100px)]">
    <div class="h-full card flex flex-col overflow-hidden">
      <!-- Toolbar -->
      <div class="card-toolbar">
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
            <select v-model="shell" :disabled="connected" class="w-28 select-sm" @change="handleShellChange">
              <option value="/bin/sh">/bin/sh</option>
              <option value="/bin/bash">/bin/bash</option>
              <option value="/bin/ash">/bin/ash</option>
            </select>
            <button v-if="!connected" type="button" class="btn btn-emerald" @click="handleConnect()">
              <i class="fas fa-plug"></i>连接
            </button>
            <button v-else type="button" class="btn btn-secondary" @click="handleDisconnect()">
              <i class="fas fa-plug-circle-xmark"></i>断开
            </button>
          </div>
        </div>
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
            <select v-model="shell" :disabled="connected" class="w-24 select-sm" @change="handleShellChange">
              <option value="/bin/sh">/bin/sh</option>
              <option value="/bin/bash">/bin/bash</option>
              <option value="/bin/ash">/bin/ash</option>
            </select>
            <button v-if="!connected" type="button" class="btn btn-emerald w-9 h-9 !px-0" title="连接" @click="handleConnect()">
              <i class="fas fa-plug text-sm"></i>
            </button>
            <button v-else type="button" class="btn btn-secondary w-9 h-9 !px-0" title="断开" @click="handleDisconnect()">
              <i class="fas fa-plug-circle-xmark text-sm"></i>
            </button>
          </div>
        </div>
      </div>

      <!-- 终端 -->
      <div class="terminal-pane">
        <TerminalPanel v-if="adapter" :adapter="adapter" />
      </div>
    </div>
  </div>
</template>
