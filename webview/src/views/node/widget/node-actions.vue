<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { NodeInfo } from '@/service/types'

@Component({ emits: ['edit', 'success'] })
class NodeActions extends Vue {
    @Prop({ type: Object, required: true }) readonly node!: NodeInfo
    @Prop({ type: Boolean, default: false }) readonly compact!: boolean

    portal = usePortal()

    revoke() {
        const node = this.node
        this.portal.showConfirm({
            title: this.$t('吊销节点'),
            message: this.$t('确定要吊销节点 <strong class="text-slate-900">{0}</strong> 吗？吊销后令牌立即失效并断开连接，需要删除后重新注册才能再次接入。', node.name),
            icon: 'fa-ban',
            iconColor: 'amber',
            confirmText: this.$t('确认吊销'),
            danger: true,
            onConfirm: async () => {
                try {
                    await api.nodeRevoke(node.id)
                    this.portal.showNotification('success', this.$t('节点已吊销'))
                    this.$emit('success')
                } catch {}
            }
        })
    }

    remove() {
        const node = this.node
        this.portal.showConfirm({
            title: this.$t('删除节点'),
            message: this.$t('确定要删除节点 <strong class="text-slate-900">{0}</strong> 吗？在线的节点会被立即断开。', node.name),
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: this.$t('确认删除'),
            danger: true,
            onConfirm: async () => {
                try {
                    await api.nodeDelete(node.id)
                    this.portal.showNotification('success', this.$t('节点已删除'))
                    this.$emit('success')
                } catch {}
            }
        })
    }
}

export default toNative(NodeActions)
</script>

<template>
  <div :class="compact ? 'table-actions' : 'card-actions'">
    <button v-if="portal.hasPerm('PUT /api/node/item/:id')" class="btn-icon btn-icon-blue" :title="$t('重命名')" @click="$emit('edit', node)">
      <i class="fas fa-pen text-xs"></i><span v-if="!compact" class="text-xs ml-1">{{ $t('重命名') }}</span>
    </button>
    <button v-if="node.status !== 'revoked' && portal.hasPerm('POST /api/node/item/:id/revoke')" class="btn-icon btn-icon-amber" :title="$t('吊销')" @click="revoke">
      <i class="fas fa-ban text-xs"></i><span v-if="!compact" class="text-xs ml-1">{{ $t('吊销') }}</span>
    </button>
    <button v-if="portal.hasPerm('DELETE /api/node/item/:id')" class="btn-icon btn-icon-red" :title="$t('删除')" @click="remove">
      <i class="fas fa-trash text-xs"></i><span v-if="!compact" class="text-xs ml-1">{{ $t('删除') }}</span>
    </button>
  </div>
</template>
