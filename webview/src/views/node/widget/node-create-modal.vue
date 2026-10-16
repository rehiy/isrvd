<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'

import { copyToClipboard } from '@/helper/dom'
import { centerUrl } from '@/helper/node'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal },
    emits: ['success']
})
class NodeCreateModal extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    name = ''
    // token 仅在本次创建成功后展示。
    token = ''

    // ─── 计算属性 ───
    get installCommand() {
        if (!this.token) return ''
        return `export ISRVD_MODE=agent\nexport ISRVD_CENTER_URL=${this.shellQuote(centerUrl())}\nexport ISRVD_NODE_TOKEN=${this.shellQuote(this.token)}\nbash <(curl -sL https://isrvd.rehiy.com/install/) install`
    }

    get dockerCommand() {
        return this.token ? `${this.installCommand} --docker` : ''
    }

    shellQuote(value: string) {
        return "'" + value.replace(/'/g, "'\\''") + "'"
    }

    // ─── 方法 ───
    show() {
        this.name = ''
        this.token = ''
        this.isOpen = true
    }

    async handleConfirm() {
        if (!this.name.trim() || this.modalLoading) return
        this.modalLoading = true
        try {
            const res = await api.nodeCreate({ name: this.name.trim() })
            this.token = res.payload?.token || ''
            this.$emit('success')
        } catch {}
        this.modalLoading = false
    }

    async copyCommand(command: string) {
        const ok = await copyToClipboard(command)
        this.portal.showNotification(ok ? 'success' : 'error', ok ? this.$t('命令已复制到剪贴板') : this.$t('复制失败，请手动复制'))
    }
}

export default toNative(NodeCreateModal)
</script>

<template>
  <BaseModal v-model="isOpen" :title="$t('接入节点')" :loading="modalLoading" :show-confirm="!token" :confirm-disabled="!name.trim()" confirm-class="btn-orange" show-footer @confirm="handleConfirm">
    <!-- 接入命令：仅在创建成功后展示 -->
    <div v-if="token" class="space-y-4">
      <p class="text-sm text-slate-600">{{ $t('token 仅在创建时显示，请妥善保存；长期有效，仅允许一个客户端在线。') }}</p>
      <div>
        <div class="flex items-center justify-between mb-2">
          <span class="text-sm font-medium text-slate-700">{{ $t('裸机安装') }}</span>
          <button type="button" class="btn btn-secondary" @click="copyCommand(installCommand)">
            <i class="fas fa-copy"></i>{{ $t('复制命令') }}
          </button>
        </div>
        <pre class="detail-value-mono whitespace-pre-wrap">{{ installCommand }}</pre>
      </div>
      <div>
        <div class="flex items-center justify-between mb-2">
          <span class="text-sm font-medium text-slate-700">{{ $t('Docker 安装') }}</span>
          <button type="button" class="btn btn-secondary" @click="copyCommand(dockerCommand)">
            <i class="fas fa-copy"></i>{{ $t('复制命令') }}
          </button>
        </div>
        <pre class="detail-value-mono whitespace-pre-wrap">{{ dockerCommand }}</pre>
      </div>
    </div>

    <!-- 创建表单 -->
    <form v-else class="space-y-4" @submit.prevent="handleConfirm">
      <div>
        <label class="form-label">{{ $t('备注') }} <span class="text-red-500">*</span></label>
        <input v-model="name" type="text" maxlength="64" required :placeholder="$t('如：杭州机房 web-01')" class="input" />
      </div>
    </form>

    <template v-if="token" #footer>
      <button type="button" class="btn btn-secondary" @click="isOpen = false">{{ $t('关闭') }}</button>
    </template>

    <template #confirm-text>{{ $t('生成 token') }}</template>
  </BaseModal>
</template>
