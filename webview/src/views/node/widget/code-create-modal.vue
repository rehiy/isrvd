<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { NodeCodeCreate, NodeCodeInfo } from '@/service/types'

import { copyToClipboard } from '@/helper/dom'
import { formatTime } from '@/helper/format'
import { centerUrl } from '@/helper/node'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show', 'view'],
    components: { BaseModal },
    emits: ['success']
})
class CodeCreateModal extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    formData: NodeCodeCreate = { name: '', ttlMinutes: 60 }
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
        return `isrvd --mode agent --center-url ${this.shellArg(centerUrl())} --enroll-code ${this.shellArg(this.created.code)}`
    }

    get dockerCommand() {
        if (!this.created) return ''
        return `docker run -d --name isrvd-agent --restart unless-stopped -e ISRVD_MODE=agent -e ISRVD_CENTER_URL=${this.shellArg(centerUrl())} -e ISRVD_ENROLL_CODE=${this.shellArg(this.created.code)} -v /srv/data:/data -v /var/run/docker.sock:/var/run/docker.sock rehiy/isrvd:slim`
    }

    shellArg(value: string) {
        return `'${value.replace(/'/g, "'\\''")}'`
    }

    get expiresText() {
        return formatTime(this.created?.expiresAt)
    }

    // ─── 方法 ───
    show() {
        this.formData = { name: '', ttlMinutes: 60 }
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

    async copyCommand(command: string) {
        const ok = await copyToClipboard(command)
        this.portal.showNotification(ok ? 'success' : 'error', ok ? this.$t('命令已复制到剪贴板') : this.$t('复制失败，请手动复制'))
    }
}

export default toNative(CodeCreateModal)
</script>

<template>
  <BaseModal v-model="isOpen" :title="$t('接入节点')" :loading="modalLoading" :show-confirm="!created" confirm-class="btn-orange" show-footer @confirm="handleConfirm">
    <!-- 接入命令：生成后或从列表查看 -->
    <div v-if="created" class="space-y-4">
      <p class="text-sm text-slate-600">{{ $t('注册码只能使用一次，') }} {{ expiresText }} {{ $t('过期；未使用前可在节点管理列表中再次查看。') }}</p>
      <div>
        <div class="flex items-center justify-between mb-2">
          <span class="text-sm font-medium text-slate-700">{{ $t('直接运行') }}</span>
          <button type="button" class="btn btn-secondary" @click="copyCommand(command)">
            <i class="fas fa-copy"></i>{{ $t('复制命令') }}
          </button>
        </div>
        <code class="detail-value-mono">{{ command }}</code>
      </div>
      <div>
        <div class="flex items-center justify-between mb-2">
          <span class="text-sm font-medium text-slate-700">{{ $t('Docker 运行') }}</span>
          <button type="button" class="btn btn-secondary" @click="copyCommand(dockerCommand)">
            <i class="fas fa-copy"></i>{{ $t('复制命令') }}
          </button>
        </div>
        <code class="detail-value-mono">{{ dockerCommand }}</code>
      </div>
    </div>

    <!-- 创建表单 -->
    <form v-else class="space-y-4" @submit.prevent="handleConfirm">
      <div>
        <label class="form-label">{{ $t('备注') }} <span class="text-slate-400 font-normal">{{ $t('(可选)') }}</span></label>
        <input v-model="formData.name" type="text" maxlength="64" :placeholder="$t('如：杭州机房 web-01')" class="input" />
      </div>
      <div>
        <label class="form-label">{{ $t('有效期') }}</label>
        <select v-model="formData.ttlMinutes" class="input">
          <option v-for="opt in ttlOptions" :key="opt.value" :value="opt.value">{{ $t(opt.label) }}</option>
        </select>
        <p class="text-xs text-slate-400 mt-1">{{ $t('注册码只能使用一次，过期或使用后自动失效') }}</p>
      </div>
    </form>

    <template v-if="created" #footer>
      <button type="button" class="btn btn-secondary" @click="isOpen = false">{{ $t('关闭') }}</button>
    </template>

    <template #confirm-text>{{ $t('生成注册码') }}</template>
  </BaseModal>
</template>
