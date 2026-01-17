<script setup>
import { ref } from 'vue'

import * as ShellTerminal from '@/helper/shell.js'

import BaseModal from '@/component/modal.vue'

const modalRef = ref(null)

const show = () => {
  modalRef.value.show()
}

const hide = () => {
  modalRef.value.hide()
}

const handleShown = () => {
  const mountPoint = document.getElementById('xterm-container')
  if (mountPoint) {
    ShellTerminal.create(mountPoint)
  }
}

const handleHidden = () => {
  ShellTerminal.destroy()
  const mountPoint = document.getElementById('xterm-container')
  if (mountPoint) {
    mountPoint.innerHTML = ''
  }
}

defineExpose({ show, hide })
</script>

<template>
  <BaseModal ref="modalRef" id="shellModal" title="实时 Shell 终端" size="modal-xl" :show-footer="false" body-class="bg-dark pe-0" @shown="handleShown" @hidden="handleHidden">
    <template #title>
      <i class="fas fa-terminal"></i> 实时 Shell 终端
    </template>
    <div id="xterm-container"></div>
  </BaseModal>
</template>
