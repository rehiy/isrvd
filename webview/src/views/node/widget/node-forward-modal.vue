<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import type { NodeForwardInfo, NodeInfo } from '@/service/types'

import { forwardIs, forwardsPatch } from '@/helper/forward'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class NodeForwardModal extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    nodes: NodeInfo[] = []
    // 编辑时保存原规则，用于在最新列表中定位；新增时为 null
    original: NodeForwardInfo | null = null
    nodeId = ''
    name = ''
    proto: 'tcp' | 'udp' = 'tcp'
    listen: number | null = null
    target = ''

    // ─── 计算属性 ───
    get editing() {
        return this.original !== null
    }

    get nodeOptions() {
        return this.nodes.filter(node => node.status !== 'revoked')
    }

    get invalid() {
        return !this.nodeId || !this.listen || !this.target.trim()
    }

    // ─── 方法 ───
    show(nodes: NodeInfo[], item?: NodeForwardInfo) {
        this.nodes = nodes
        this.original = item || null
        this.nodeId = item?.nodeId || this.nodeOptions[0]?.id || ''
        this.name = item?.name || ''
        this.proto = item?.proto || 'tcp'
        this.listen = item?.listen ?? null
        this.target = item?.target || ''
        this.isOpen = true
    }

    // 只填端口时补全为节点本机回环地址
    normalizeTarget(target: string) {
        const value = target.trim()
        return /^\d+$/.test(value) ? `127.0.0.1:${value}` : value
    }

    async handleConfirm() {
        if (this.invalid || this.modalLoading) return
        this.modalLoading = true
        try {
            const entry = {
                name: this.name.trim() || undefined,
                proto: this.proto,
                listen: Number(this.listen),
                target: this.normalizeTarget(this.target)
            }
            const original = this.original
            const ok = await forwardsPatch(this.nodeId, list => {
                if (!original) return [...list, entry]
                const index = list.findIndex(f => forwardIs(f, original))
                if (index < 0) return null
                list[index] = entry
                return list
            })
            if (!ok) {
                this.portal.showNotification('error', this.$t('节点或转发规则已不存在，请刷新后重试'))
            } else {
                this.portal.showNotification('success', this.$t('转发规则已保存'))
                this.isOpen = false
                this.$emit('success')
            }
        } catch {}
        this.modalLoading = false
    }
}

export default toNative(NodeForwardModal)
</script>

<template>
  <BaseModal v-model="isOpen" :title="editing ? $t('编辑转发') : $t('添加转发')" :loading="modalLoading" :confirm-disabled="invalid" max-width-class="max-w-lg" confirm-class="btn-orange" show-footer @confirm="handleConfirm">
    <form class="space-y-4" @submit.prevent="handleConfirm">
      <p class="text-sm text-slate-600">{{ $t('中控监听指定端口，经隧道连接到节点上的目标地址，支持 TCP 与 UDP。') }}</p>
      <p class="text-xs text-amber-600">
        <i class="fas fa-triangle-exclamation mr-1"></i>{{ $t('转发的连接不经过 isrvd 认证，鉴权由目标应用负责；请勿转发无认证的服务。') }}
      </p>

      <div>
        <label class="form-label">{{ $t('所属节点') }} <span class="text-red-500">*</span></label>
        <select v-model="nodeId" :disabled="editing" required class="input">
          <option v-for="node in nodeOptions" :key="node.id" :value="node.id">{{ node.name }}</option>
        </select>
      </div>

      <div>
        <label class="form-label">{{ $t('协议') }}</label>
        <div class="flex gap-2">
          <button type="button" class="btn-proto" :class="proto === 'tcp' ? 'btn-proto-active' : 'btn-proto-inactive'" @click="proto = 'tcp'">TCP</button>
          <button type="button" class="btn-proto" :class="proto === 'udp' ? 'btn-proto-active' : 'btn-proto-inactive'" @click="proto = 'udp'">UDP</button>
        </div>
      </div>

      <div>
        <label class="form-label">{{ $t('中控监听端口') }} <span class="text-red-500">*</span></label>
        <input v-model.number="listen" type="number" min="1" max="65535" required class="input" placeholder="20022" />
        <p class="text-xs text-slate-400 mt-1">{{ $t('须在中控配置 portmap.ports 允许的范围内') }}</p>
      </div>

      <div>
        <label class="form-label">{{ $t('节点目标地址') }} <span class="text-red-500">*</span></label>
        <input v-model="target" type="text" maxlength="200" required class="input font-mono text-sm" placeholder="127.0.0.1:22" />
        <p class="text-xs text-slate-400 mt-1">{{ $t('host:port；只填端口时默认指向节点本机 127.0.0.1') }}</p>
      </div>

      <div>
        <label class="form-label">{{ $t('备注') }}</label>
        <input v-model="name" type="text" maxlength="64" class="input" :placeholder="$t('如：SSH')" />
      </div>
    </form>

    <template #confirm-text>{{ $t('保存转发') }}</template>
  </BaseModal>
</template>
