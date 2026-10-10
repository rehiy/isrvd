<script lang="ts">
import { Component, Vue, Watch, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import type { NodeInfo } from '@/service/types'

import { switchNode } from '@/helper/node'

import Dropdown from '@/component/dropdown.vue'

@Component({
    components: { Dropdown }
})
class NodeSwitcher extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    menuOpen = false

    // ─── 计算属性 ───

    // 仅中控模式且有节点查看权限（创始人）时显示
    get visible() {
        return this.portal.hasPerm('GET /api/node/nodes')
    }

    get approvedNodes() {
        return this.portal.nodes.filter(n => n.status === 'approved')
    }

    get viewingNode() {
        return this.portal.currentNodeId !== ''
    }

    get label() {
        if (!this.viewingNode) return this.$t('本机')
        return this.portal.nodes.find(n => n.id === this.portal.currentNodeId)?.name || this.$t('节点')
    }

    // ─── 监听器 ───
    @Watch('visible', { immediate: true })
    onVisibleChange(visible: boolean) {
        if (visible) this.portal.nodeLoad()
    }

    @Watch('menuOpen')
    onMenuOpenChange(open: boolean) {
        if (open) this.portal.nodeLoad()
    }

    // ─── 方法 ───
    usable(node: NodeInfo) {
        return node.online && node.compatible
    }

    select(id: string) {
        this.menuOpen = false
        if (id === this.portal.currentNodeId) return
        // 详情页（带路由参数）的资源只存在于原来的节点，切换后回到概览；列表页保持原位
        const hash = Object.keys(this.$route.params).length === 0 ? window.location.hash : '#/overview'
        switchNode(id, hash)
    }
}

export default toNative(NodeSwitcher)
</script>

<template>
  <Dropdown v-if="visible" v-model:open="menuOpen" placement="bottom" align="right" max-height="360px">
    <template #trigger="{ toggle }">
      <button
        class="btn btn-ghost gap-2"
        :class="viewingNode
          ? 'text-primary-600 bg-primary-50 hover:bg-primary-100'
          : 'text-slate-600 hover:text-primary-600 hover:bg-primary-50'"
        :title="viewingNode ? $t(`当前节点：${label}`) : $t('切换节点')"
        @click="toggle"
      >
        <i class="fas fa-sitemap"></i>
        <span class="hidden sm:inline max-w-40 truncate">{{ label }}</span>
        <i class="fas fa-chevron-down text-xs text-slate-400 hidden sm:inline transition-transform duration-200" :class="{ 'rotate-180': menuOpen }"></i>
      </button>
    </template>

    <!-- 本机（中控） -->
    <button class="dropdown-item" :class="{ 'dropdown-item-active': !viewingNode }" @click="select('')">
      <i class="fas fa-circle text-[8px] text-emerald-500"></i>
      <span>{{ $t('本机（中控）') }}</span>
      <i v-if="!viewingNode" class="fas fa-check text-xs ml-auto"></i>
    </button>

    <!-- 已审批的受管机 -->
    <button
      v-for="node in approvedNodes"
      :key="node.id"
      class="dropdown-item"
      :class="{ 'dropdown-item-active': node.id === portal.currentNodeId }"
      :disabled="!usable(node) && node.id !== portal.currentNodeId"
      @click="select(node.id)"
    >
      <i class="fas fa-circle text-[8px]" :class="usable(node) ? 'text-emerald-500' : 'text-slate-300'"></i>
      <span class="truncate max-w-48">{{ node.name }}</span>
      <span v-if="!node.online" class="text-xs ml-auto">{{ $t('离线') }}</span>
      <span v-else-if="!node.compatible" class="text-xs text-amber-600 ml-auto">{{ $t('版本不兼容') }}</span>
      <i v-else-if="node.id === portal.currentNodeId" class="fas fa-check text-xs ml-auto"></i>
    </button>

    <div v-if="approvedNodes.length === 0" class="px-4 py-3 text-xs text-slate-400">{{ $t('暂无已接入的节点') }}</div>
  </Dropdown>
</template>
