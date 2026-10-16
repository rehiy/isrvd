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
    get connectionConfig() {
        if (!this.created) return ''
        return `export ISRVD_MODE=agent\nexport ISRVD_CENTER_URL=${this.shellQuote(centerUrl())}\nexport ISRVD_NODE_TOKEN=${this.shellQuote(this.created.code)}`
    }

    get dockerCommand() {
        return `${this.connectionConfig}\ndocker run -d --name isrvd-agent --restart unless-stopped -e ISRVD_MODE -e ISRVD_CENTER_URL -e ISRVD_NODE_TOKEN -v /srv/data:/data -v /var/run/docker.sock:/var/run/docker.sock rehiy/isrvd:slim`
    }

    get expiresText() {
        return formatTime(this.created?.expiresAt)
    }

    shellQuote(value: string) {
        return "'" + value.replace(/'/g, "'\\''") + "'"
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
      <p class="text-sm text-slate-600">{{ $t('token 首次接入前将于 {0} 过期；接入后沿用同一 token，仅允许一个客户端在线。', expiresText) }}</p>
      <div>
        <div class="flex items-center justify-between mb-2">
          <span class="text-sm font-medium text-slate-700">{{ $t('受控机配置') }}</span>
          <button type="button" class="btn btn-secondary" @click="copyCommand(connectionConfig)">
            <i class="fas fa-copy"></i>{{ $t('复制配置') }}
          </button>
        </div>
        <pre class="detail-value-mono whitespace-pre-wrap">{{ connectionConfig }}</pre>
        <p class="mt-2 text-xs text-slate-400">{{ $t('在受控机启动环境中设置这些环境变量，再运行 isrvd；修改后需重启服务。') }}</p>
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
        <p class="text-xs text-slate-400 mt-1">{{ $t('有效期仅限制首次接入；连接后同一 token 可用于重启和重连') }}</p>
      </div>
    </form>

    <template v-if="created" #footer>
      <button type="button" class="btn btn-secondary" @click="isOpen = false">{{ $t('关闭') }}</button>
    </template>

    <template #confirm-text>{{ $t('生成注册码') }}</template>
  </BaseModal>
</template>
