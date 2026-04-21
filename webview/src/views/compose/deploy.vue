<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { ComposeDeployTarget } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import { Codemirror } from 'vue-codemirror'
import { yaml } from '@codemirror/lang-yaml'

import MarketplaceModal, { type MarketplacePick } from './widget/marketplace-modal.vue'

@Component({
    components: { Codemirror, MarketplaceModal }
})
class ComposeDeploy extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    loading = false

    // 部署参数
    target: ComposeDeployTarget = 'docker'
    projectName = ''
    initURL = ''
    content = ''

    // swarm 可用性（通过探测接口得出，不可用则禁用 swarm 选项）
    swarmAvailable = false

    // 应用市场 modal 开关
    marketplaceVisible = false

    // 预填态（来自应用市场一键选择）：仅用于头部提示徽章，不锁定输入
    fromMarketplace = false

    readonly extensions = [yaml()]

    // 实例名合法性校验（与后端 safeName 保持一致）
    readonly nameRegex = /^[a-zA-Z0-9][a-zA-Z0-9_.-]*$/

    // ─── 计算属性 ───
    get projectNameValid(): boolean {
        const name = this.projectName.trim()
        return !!name && this.nameRegex.test(name)
    }

    get canSubmit(): boolean {
        return !this.loading && !!this.content.trim() && this.projectNameValid
    }

    get targetLabel(): string {
        return this.target === 'swarm' ? 'Swarm 服务' : '单机容器'
    }

    // ─── 生命周期 ───
    async mounted() {
        // 探测 swarm 能力
        try {
            const res = await api.swarmInfo()
            this.swarmAvailable = !!res.payload
        } catch {
            this.swarmAvailable = false
        }
    }

    // ─── 方法 ───
    selectTarget(t: ComposeDeployTarget) {
        if (t === 'swarm' && !this.swarmAvailable) return
        this.target = t
    }

    openMarketplace() {
        this.marketplaceVisible = true
    }

    onMarketplacePick(payload: MarketplacePick) {
        this.content = payload.compose
        this.projectName = payload.name
        this.initURL = payload.initURL || ''
        this.target = 'docker'
        this.fromMarketplace = true
    }

    async handleDeploy() {
        if (!this.canSubmit) return

        const projectName = this.projectName.trim()
        // initURL 仅在 docker 目标下生效（swarm 集群自管，不落盘）
        const initURL = this.target === 'docker' ? this.initURL.trim() : ''

        this.loading = true
        try {
            const res = await api.composeDeploy({
                target: this.target,
                content: this.content,
                projectName,
                initURL: initURL || undefined,
            })
            const created = res.payload?.items || []
            const label = this.target === 'swarm' ? '服务' : '容器'
            this.actions.showNotification('success', `${projectName} 部署成功，已创建 ${created.length} 个${label}`)

            // 成功后跳转到对应列表页
            if (this.target === 'swarm') {
                this.$router.push('/swarm/services')
            } else {
                this.$router.push('/docker/containers')
            }
        } catch {
            // 错误由 axios 拦截器统一弹窗
        }
        this.loading = false
    }

    resetForm() {
        this.content = ''
        this.projectName = ''
        this.initURL = ''
        this.target = 'docker'
        this.fromMarketplace = false
    }
}

export default toNative(ComposeDeploy)
</script>

<template>
  <div>
    <div class="card mb-4">
      <!-- Toolbar -->
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3">
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-violet-500 flex items-center justify-center">
              <i class="fas fa-file-code text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">Compose 部署</h1>
              <p class="text-xs text-slate-500">直接粘贴 docker-compose.yml 或从应用市场选择模板</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <button type="button" @click="resetForm()" :disabled="loading" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors disabled:opacity-50">
              <i class="fas fa-rotate-left"></i>清空
            </button>
            <button type="button" @click="openMarketplace()" class="px-3 py-1.5 rounded-lg bg-amber-500 hover:bg-amber-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-store"></i>从应用市场选择
            </button>
          </div>
        </div>
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="w-9 h-9 rounded-lg bg-violet-500 flex items-center justify-center flex-shrink-0">
              <i class="fas fa-file-code text-white"></i>
            </div>
            <div class="min-w-0 flex-1">
              <h1 class="text-base font-semibold text-slate-800 truncate">Compose 部署</h1>
            </div>
          </div>
          <div class="flex items-center gap-2 flex-shrink-0">
            <button type="button" @click="openMarketplace()" class="w-9 h-9 rounded-lg bg-amber-500 hover:bg-amber-600 flex items-center justify-center text-white transition-colors" title="从应用市场选择">
              <i class="fas fa-store"></i>
            </button>
            <button type="button" @click="resetForm()" :disabled="loading" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 flex items-center justify-center transition-colors disabled:opacity-50" title="清空">
              <i class="fas fa-rotate-left"></i>
            </button>
          </div>
        </div>
      </div>

      <!-- 表单 -->
      <div class="p-4 md:p-6 space-y-4 max-w-5xl">
        <!-- 应用市场预填提示 -->
        <div v-if="fromMarketplace" class="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 flex items-start gap-2 text-xs">
          <i class="fas fa-circle-info text-amber-500 mt-0.5"></i>
          <div class="text-amber-800">
            已从应用市场预填模板，可在此基础上直接部署或调整后再部署。
          </div>
        </div>

        <!-- 部署目标 -->
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

        <!-- 实例名（必填；docker 落盘到 数据目录/实例名，swarm 仅作 compose project 名） -->
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">
            实例名 <span class="text-red-500">*</span>
          </label>
          <input
            type="text"
            v-model="projectName"
            placeholder="例如 my-app"
            class="input"
            :disabled="loading"
          />
          <p class="mt-1 text-xs text-slate-400">
            同时作为 compose project 名<span v-if="target === 'docker'">，将在同名目录下保存 compose.yaml</span>
          </p>
          <p v-if="projectName.trim() && !projectNameValid" class="mt-1 text-xs text-red-500">
            实例名不符合命名规则 <code class="px-1 bg-slate-100 rounded">[a-zA-Z0-9][a-zA-Z0-9_.-]*</code>
          </p>
        </div>

        <!-- initURL（仅 docker） -->
        <div v-if="target === 'docker'">
          <label class="block text-sm font-medium text-slate-700 mb-2">
            附加文件 URL
            <span class="text-xs font-normal text-slate-400">（选填，指向 init.zip 下载地址）</span>
          </label>
          <input
            type="text"
            v-model="initURL"
            placeholder="例如 http://market.example.com/apps/foo/1.0.0/init.zip"
            class="input"
            :disabled="loading"
          />
          <p class="mt-1 text-xs text-slate-400">部署前下载 zip 并解压到实例目录，随后写入 compose.yaml 并启动</p>
        </div>

        <!-- Compose 内容 -->
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">
            Compose 内容 <span class="text-red-500">*</span>
          </label>
          <div class="rounded-xl overflow-hidden border border-slate-200">
            <Codemirror v-model="content" :style="{ height: '50vh' }" :extensions="extensions" :disabled="loading" />
          </div>
          <p class="mt-1 text-xs text-slate-400">
            变量插值需在客户端完成，后端仅按原文落盘与加载
          </p>
        </div>

        <!-- 操作按钮 -->
        <div class="flex flex-col sm:flex-row sm:items-center gap-3 pt-2">
          <button
            type="button"
            @click="handleDeploy()"
            :disabled="!canSubmit"
            class="whitespace-nowrap flex-shrink-0 self-start px-5 py-2 rounded-lg bg-blue-500 hover:bg-blue-600 disabled:bg-slate-300 disabled:cursor-not-allowed text-white text-sm font-medium flex items-center gap-2 transition-colors"
          >
            <i v-if="loading" class="fas fa-spinner fa-spin"></i>
            <i v-else class="fas fa-rocket"></i>
            <span>{{ loading ? '部署中...' : '部署' }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- 应用市场 Modal -->
    <MarketplaceModal v-model="marketplaceVisible" @pick="onMarketplacePick" />
  </div>
</template>
