<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import type { NodeForwardInfo } from '@/service/types'

import { forwardIs, forwardsPatch } from '@/helper/forward'

@Component({ emits: ['edit', 'success'] })
class ForwardActions extends Vue {
    @Prop({ type: Object, required: true }) readonly item!: NodeForwardInfo
    @Prop({ type: Boolean, default: false }) readonly compact!: boolean

    portal = usePortal()

    get canUpdate() {
        return this.portal.hasPerm('PUT /api/node/:id/forwards')
    }

    remove() {
        const item = this.item
        this.portal.showConfirm({
            title: this.$t('删除转发'),
            message: this.$t('确定要删除节点 <strong class="text-slate-900">{0}</strong> 上监听端口 <strong class="text-slate-900">{1}/{2}</strong> 的转发规则吗？已建立的连接会被立即断开。', item.nodeName, item.listen, item.proto.toUpperCase()),
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: this.$t('确认删除'),
            danger: true,
            onConfirm: async () => {
                try {
                    const ok = await forwardsPatch(item.nodeId, list => {
                        const next = list.filter(f => !forwardIs(f, item))
                        return next.length === list.length ? null : next
                    })
                    if (!ok) {
                        this.portal.showNotification('error', this.$t('节点或转发规则已不存在，请刷新后重试'))
                    } else {
                        this.portal.showNotification('success', this.$t('转发规则已删除'))
                    }
                    this.$emit('success')
                } catch {}
            }
        })
    }
}

export default toNative(ForwardActions)
</script>

<template>
  <div v-if="canUpdate" :class="compact ? 'table-actions' : 'card-actions'">
    <button class="btn-icon btn-icon-blue" :title="$t('编辑')" @click="$emit('edit', item)">
      <i class="fas fa-pen text-xs"></i><span v-if="!compact" class="text-xs ml-1">{{ $t('编辑') }}</span>
    </button>
    <button class="btn-icon btn-icon-red" :title="$t('删除')" @click="remove">
      <i class="fas fa-trash text-xs"></i><span v-if="!compact" class="text-xs ml-1">{{ $t('删除') }}</span>
    </button>
  </div>
</template>
