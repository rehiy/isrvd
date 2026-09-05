<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'

// Inspector 仅在开发构建加载。CopilotKit 未提供关闭浮动启动器的配置，
// 将其隐藏并由聊天标题栏的同一入口替代。
const enabled = import.meta.env.DEV
const inspectorSelector = 'cpk-web-inspector'
const launcherStyleID = 'isrvd-copilot-inspector-launcher'

type InspectorElement = HTMLElement & {
    openInspector?: (source: 'floating_button') => void
}

let documentObserver: MutationObserver | undefined
let shadowObserver: MutationObserver | undefined
let observedRoot: ShadowRoot | undefined

function inspector(): InspectorElement | null {
    return document.querySelector<InspectorElement>(inspectorSelector)
}

function hideFloatingLauncher() {
    const root = inspector()?.shadowRoot
    if (!root || root.getElementById(launcherStyleID)) return

    const style = document.createElement('style')
    style.id = launcherStyleID
    style.textContent = '.console-button-wrapper { display: none !important; }'
    root.append(style)
}

function observeInspector() {
    const root = inspector()?.shadowRoot
    if (!root || root === observedRoot) return

    shadowObserver?.disconnect()
    observedRoot = root
    shadowObserver = new MutationObserver(hideFloatingLauncher)
    shadowObserver.observe(root, { childList: true })
    hideFloatingLauncher()
}

function openInspector() {
    inspector()?.openInspector?.('floating_button')
}

onMounted(() => {
    if (!enabled) return

    observeInspector()
    documentObserver = new MutationObserver(observeInspector)
    documentObserver.observe(document.body, { childList: true, subtree: true })
})

onUnmounted(() => {
    documentObserver?.disconnect()
    shadowObserver?.disconnect()
})
</script>

<template>
  <button
    v-if="enabled"
    type="button"
    title="打开 Copilot 调试面板"
    aria-label="打开 Copilot 调试面板"
    class="btn btn-icon btn-icon-slate"
    @click="openInspector"
  >
    <i class="fas fa-bug"></i>
  </button>
</template>
