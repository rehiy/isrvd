<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { ComposeDeployTarget } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import { Codemirror } from 'vue-codemirror'
import { yaml } from '@codemirror/lang-yaml'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal, Codemirror },
    emits: ['success']
})
class ComposeModal extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    isOpen = false
    loading = false
    content = ''
    target: ComposeDeployTarget = 'docker'
    swarmAvailable = false
    readonly extensions = [yaml()]

    // ─── 方法 ───
    async show() {
        this.content = ''
        this.target = 'docker'
        this.isOpen = true
        // 探测 swarm 能力，失败则禁用 swarm 选项
        try {
            const res = await api.swarmInfo()
            this.swarmAvailable = !!res.payload
        } catch {
            this.swarmAvailable = false
        }
    }

    selectTarget(t: ComposeDeployTarget) {
        if (t === 'swarm' && !this.swarmAvailable) return
        this.target = t
    }

    async handleConfirm() {
        if (!this.content.trim()) return
        this.loading = true
        try {
            const res = await api.composeDeployYml({
                target: this.target,
                content: this.content,
            })
            const created = res.payload?.items || []
            const label = this.target === 'swarm' ? '服务' : '容器'
            this.actions.showNotification('success', `Compose 部署成功，已创建 ${created.length} 个${label}`)
            this.isOpen = false
            this.$emit('success')
        } catch (e) {
            this.actions.showNotification('error', 'Compose 部署失败')
        }
        this.loading = false
    }
}

export default toNative(ComposeModal)
</script>

<template>
  <BaseModal
    v-model="isOpen"
    title="通过 Compose 部署"
    :loading="loading"
    show-footer
    @confirm="handleConfirm"
  >
    <template #confirm-text>部署</template>
    <div class="space-y-4">
      <!-- 目标选择 -->
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">部署目标</label>
        <div class="inline-flex gap-1 bg-slate-100 p-1 rounded-lg">
          <button
            type="button"
            @click="selectTarget('docker')"
            :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5',
              target === 'docker' ? 'bg-white text-amber-600 shadow-sm' : 'text-slate-500 hover:text-slate-700']"
          >
            <i class="fab fa-docker"></i><span>单机容器</span>
          </button>
          <button
            type="button"
            @click="selectTarget('swarm')"
            :disabled="!swarmAvailable"
            :class="['px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5',
              target === 'swarm' ? 'bg-white text-amber-600 shadow-sm'
                : (swarmAvailable ? 'text-slate-500 hover:text-slate-700' : 'text-slate-300 cursor-not-allowed')]"
            :title="swarmAvailable ? '' : '当前节点未启用 Swarm'"
          >
            <i class="fas fa-cubes"></i><span>Swarm 服务</span>
          </button>
        </div>
      </div>

      <!-- Compose 内容 -->
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">Compose 内容 <span class="text-red-500">*</span></label>
        <div class="rounded-xl overflow-hidden border border-slate-200">
          <Codemirror v-model="content" :style="{ height: '50vh' }" :extensions="extensions" :disabled="loading" />
        </div>
        <p class="mt-1 text-xs text-slate-400">
          粘贴 docker-compose.yml 内容，按服务定义逐个创建{{ target === 'swarm' ? ' Swarm 服务' : '容器' }}
        </p>
      </div>
    </div>
  </BaseModal>
</template>
