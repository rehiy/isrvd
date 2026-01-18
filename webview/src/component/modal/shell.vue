<script setup>
import { inject, ref } from 'vue'
import * as ShellTerminal from '@/helper/shell.js'
import { APP_STATE_KEY } from '@/store/state.js'
import BaseModal from '@/component/modal.vue'

const state = inject(APP_STATE_KEY)

const modalRef = ref(null)
const xtermRef = ref(null)
const shellType = ref('bash')
const connectionStatus = ref(0) // 0: 未连接, 1: 连接中, 2: 已连接

const handleConnect = () => {
  if (connectionStatus.value > 0) return
  connectionStatus.value = 1
  setTimeout(() => { connectionStatus.value = 2 }, 500)
  xtermRef.value && ShellTerminal.create(xtermRef.value, state.token, shellType.value)
}

const handleHidden = () => {
  ShellTerminal.destroy()
  if (xtermRef.value) xtermRef.value.innerHTML = ''
  connectionStatus.value = 0
}

const show = () => modalRef.value.show()
const hide = () => { handleHidden(); modalRef.value.hide() }
defineExpose({ show, hide })
</script>

<template>
  <BaseModal ref="modalRef" id="shellModal" title="实时 Shell 终端" size="modal-xl" :show-footer="false" body-class="bg-dark pe-0" @hidden="handleHidden">
    <template #title>
      <i class="fas fa-terminal"></i> 实时 Shell 终端
      <select v-model="shellType" :disabled="connectionStatus === 2" class="form-select form-select-sm ms-3">
        <option value="bash">bash</option>
        <option value="sh">sh</option>
        <option value="zsh">zsh</option>
        <option value="fish">fish</option>
      </select>
      <button @click="handleConnect" :disabled="connectionStatus > 0" class="btn btn-primary btn-sm ms-2">
        <i class="fas fa-plug"></i> {{ connectionStatus === 1 ? '连接中...' : '连接' }}
      </button>
    </template>
    <div ref="xtermRef"></div>
  </BaseModal>
</template>

<style scoped>
.form-select {
  width: auto;
  display: inline-block;
}
</style>