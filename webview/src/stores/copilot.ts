import { defineStore } from 'pinia'
import { ref } from 'vue'

// 侧栏开合控制由 CopilotSidebar 的 toggle-button 插槽提供，
// 经此 store 转交给顶栏按钮，避免侧栏(z-1200)覆盖 header(z-30)。
interface SidebarCtl {
    isOpen: boolean
    toggle: () => void
}

export const useCopilotStore = defineStore('copilot', () => {
    const sidebarOpen = ref(false)
    let ctl: SidebarCtl | null = null

    function bindSidebar(c: SidebarCtl) {
        ctl = c
        sidebarOpen.value = c.isOpen
    }

    function unbindSidebar() {
        ctl = null
        sidebarOpen.value = false
    }

    function toggleSidebar() {
        ctl?.toggle()
    }

    return { sidebarOpen, bindSidebar, unbindSidebar, toggleSidebar }
})
