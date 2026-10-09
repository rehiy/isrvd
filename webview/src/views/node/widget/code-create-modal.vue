<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { NodeCodeCreate, NodeCodeInfo } from '@/service/types'

import { copyToClipboard } from '@/helper/dom'
import { formatTime } from '@/helper/format'
import { centerUrl } from '@/helper/node'

import BaseModal from '@/component/modal.vue'
import ToggleCard from '@/component/toggle-card.vue'

@Component({
    expose: ['show', 'view'],
    components: { BaseModal, ToggleCard },
    emits: ['success']
})
class CodeCreateModal extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    formData: NodeCodeCreate = { name: '', ttlMinutes: 60, autoApprove: false }
    // 要展示的注册码：创建成功后，或从列表中查看已有的待接入注册码
    created: NodeCodeInfo | null = null

    ttlOptions = [
        { label: '10 分钟', value: 10 },
        { label: '1 小时', value: 60 },
        { label: '1 天', value: 1440 },
        { label: '7 天', value: 10080 }
    ]

    // ─── 计算属性 ───
    get command() {
        if (!this.created) return ''
        return `isrvd --mode agent --center-url ${centerUrl()} --enroll-code ${this.created.code}`
    }

    get expiresText() {
        return formatTime(this.created?.expiresAt)
    }

    // ─── 方法 ───
    show() {
        this.formData = { name: '', ttlMinutes: 60, autoApprove: false }
        this.created = null
        this.isOpen = true
    }

    // 查看列表中尚未使用的注册码及其接入命令
    view(code: NodeCodeInfo) {
        this.created = code
        this.isOpen = true
    }

    async handleConfirm() {
        this.modalLoading = true
        try {
            const res = await api.nodeCodeCreate({ ...this.formData, name: this.formData.name.trim() })
            this.created = res.payload ?? null
            this.$emit('success')
        } catch {}
        this.modalLoading = false
    }

    async copyCommand() {
        const ok = await copyToClipboard(this.command)
        this.portal.showNotification(ok ? 'success' : 'error', ok ? '命令已复制到剪贴板' : '复制失败，请手动复制')
    }
}

export default toNative(CodeCreateModal)
</script>

<template>
  <BaseModal v-model="isOpen" title="接入节点" :loading="modalLoading" :show-confirm="!created" confirm-class="btn-orange" show-footer @confirm="handleConfirm">
    <!-- 接入命令：生成后或从列表查看 -->
    <div v-if="created" class="space-y-4">
      <p class="text-sm text-slate-600">注册码只能使用一次，{{ expiresText }} 过期；未使用前可在节点管理列表中再次查看。</p>
      <div>
        <label class="form-label">在受管机上执行</label>
        <code class="detail-value-mono">{{ command }}</code>
        <p class="text-xs text-slate-400 mt-1">
          {{ created.autoApprove ? '该注册码已设置自动审批，节点启动后会直接上线。' : '节点启动后会出现在「节点管理」列表中，审批通过后上线。' }}
          首次注册成功后，重启受管机无需再带注册码。
        </p>
      </div>
    </div>

    <!-- 创建表单 -->
    <form v-else class="space-y-4" @submit.prevent="handleConfirm">
      <div>
        <label class="form-label">备注 <span class="text-slate-400 font-normal">(可选)</span></label>
        <input v-model="formData.name" type="text" maxlength="64" placeholder="如：杭州机房 web-01" class="input" />
      </div>
      <div>
        <label class="form-label">有效期</label>
        <select v-model="formData.ttlMinutes" class="input">
          <option v-for="opt in ttlOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
        </select>
        <p class="text-xs text-slate-400 mt-1">注册码只能使用一次，过期或使用后自动失效</p>
      </div>
      <ToggleCard v-model="formData.autoApprove" label="自动通过审批" desc="使用该注册码接入的节点无需人工审批，直接上线。仅建议在可信网络内使用" />
    </form>

    <!-- 生成后：底部改为「关闭」与「复制命令」 -->
    <template v-if="created" #footer>
      <button type="button" class="btn btn-secondary" @click="isOpen = false">关闭</button>
      <button type="button" class="btn btn-orange" @click="copyCommand">
        <i class="fas fa-copy"></i>复制命令
      </button>
    </template>

    <template #confirm-text>生成注册码</template>
  </BaseModal>
</template>
