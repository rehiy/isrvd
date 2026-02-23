<script setup>
import { inject, ref, onMounted, onUnmounted } from 'vue'

import * as ShellTerminal from '@/helper/shell.js'
import { APP_STATE_KEY } from '@/store/state.js'

const state = inject(APP_STATE_KEY)

const xtermRef = ref(null)
const shellType = ref('bash')
const connectionStatus = ref(0) // 0: 未连接, 1: 连接中, 2: 已连接

const handleConnect = () => {
  if (connectionStatus.value > 0) return
  connectionStatus.value = 1
  setTimeout(() => { connectionStatus.value = 2 }, 500)
  xtermRef.value && ShellTerminal.create(xtermRef.value, state.token, shellType.value)
}

onMounted(() => {
  handleConnect()
})

onUnmounted(() => {
  ShellTerminal.destroy()
})
</script>

<template>
  <div class="card">
    <div class="card-header d-flex align-items-center">
      <i class="fas fa-terminal me-2"></i> 实时 Shell 终端
      <select v-model="shellType" :disabled="connectionStatus === 2" @change="ShellTerminal.destroy(); connectionStatus = 0; handleConnect()" class="form-select form-select-sm ms-3" style="width: auto;">
        <option value="bash">bash</option>
        <option value="sh">sh</option>
        <option value="zsh">zsh</option>
        <option value="fish">fish</option>
      </select>
      <span class="ms-3 badge" :class="connectionStatus === 2 ? 'bg-success' : 'bg-secondary'">
        {{ connectionStatus === 0 ? '未连接' : connectionStatus === 1 ? '连接中...' : '已连接' }}
      </span>
    </div>
    <div class="card-body bg-dark p-0">
      <div ref="xtermRef" style="min-height: 400px;"></div>
    </div>
  </div>
</template>
