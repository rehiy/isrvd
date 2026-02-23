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
  <div class="shell-container">
    <div class="card h-100">
      <div class="card-header d-flex align-items-center">
        <i class="fas fa-terminal me-2"></i> Shell 终端
        <select v-model="shellType" :disabled="connected" class="form-select form-select-sm ms-3" style="width: auto;">
          <option value="bash">bash</option>
          <option value="sh">sh</option>
          <option value="zsh">zsh</option>
          <option value="fish">fish</option>
        </select>
        <button @click="connected ? handleDisconnect() : handleConnect()" :class="connected ? 'btn-outline-danger' : 'btn-primary'" class="btn btn-sm ms-2">
          <i :class="connected ? 'fa-plug-circle-xmark' : 'fa-plug'" class="fas"></i>
          {{ connected ? '断开' : '连接' }}
        </button>
      </div>
      <div class="card-body bg-dark p-0">
        <div ref="xtermRef" class="shell-terminal"></div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.shell-container {
  height: calc(100vh - 84px);
}

.shell-terminal {
  height: 100%;
  padding: 10px;
  padding-right: 0;
}
</style>