<script lang="ts">
import * as yaml from 'js-yaml'
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { ComposeDeployTarget, ComposeMarketplacePick } from '@/service/types'

import ComposeEditor from './widget/compose-editor.vue'
import MarketplaceModal from './widget/marketplace-modal.vue'

@Component({
    components: { ComposeEditor, MarketplaceModal }
})
class ComposeDeploy extends Vue {
    portal = usePortal()

    // ─── 数据属性 ───
    loading = false

    // 部署参数
    target: ComposeDeployTarget = 'docker'
    initURL = ''
    initFile: File | null = null
    content = ''

    // 应用市场 modal 开关
    marketplaceVisible = false

    // 预填态（来自应用市场一键选择）：仅用于头部提示徽章，不锁定输入
    fromMarketplace = false

    // ─── 计算属性 ───
    get swarmAvailable(): boolean {
        return this.portal.hasPerm('POST /api/compose/swarm')
    }

    get canSubmit(): boolean {
        return !this.loading && !!this.content.trim()
    }

    /** 编辑器警告/提示文案 */
    get dynamicWarning(): string {
        if (this.fromMarketplace) {
            return '已从应用市场预填模板，可在此基础上直接部署或调整后再部署'
        }
        return '项目名来自 compose 文件的 name 字段；变量插值需在客户端完成，后端仅按原文落盘与加载'
    }

    // ─── 方法 ───
    selectTarget(t: ComposeDeployTarget) {
        if (t === 'swarm' && !this.swarmAvailable) return
        this.target = t
    }

    openMarketplace() {
        this.marketplaceVisible = true
    }

    onMarketplacePick(payload: ComposeMarketplacePick) {
        let composeContent = payload.compose
        try {
            const doc = yaml.load(composeContent) as Record<string, unknown> | null
            if (doc && typeof doc === 'object' && !doc.name) {
                const nameValue = payload.name || 'compose-project'
                // 移除 BOM 和开头的 ---
                composeContent = composeContent.replace(/^\uFEFF/, '').replace(/^---\n?/, '')
                // 在文件最前面添加 name 字段
                composeContent = `name: ${nameValue}\n` + composeContent
            }
        } catch (e) {
            const msg = e instanceof Error ? e.message : '未知错误'
            this.portal.showNotification('warning', `模板格式异常，已按原文加载：${msg}`)
        }
        this.content = composeContent
        this.initURL = payload.initURL || ''
        this.target = 'docker'
        this.fromMarketplace = true
    }

    onInitFileChange(e: Event) {
        const input = e.target as HTMLInputElement
        const file = input.files?.[0] ?? null
        this.initFile = file
        if (file) this.initURL = ''
    }

    clearInitFile() {
        this.initFile = null
        const input = this.$refs.fileInput as HTMLInputElement | undefined
        if (input) input.value = ''
    }

    async handleDeploy() {
        if (!this.canSubmit) return

        this.loading = true
        try {
            const res = this.target === 'swarm'
                ? await api.composeSwarmDeploy({
                    content: this.content,
                    initURL: this.initURL.trim() || undefined,
                    initFile: this.initFile ?? undefined,
                })
                : await api.composeDockerDeploy({
                    content: this.content,
                    initURL: this.initURL.trim() || undefined,
                    initFile: this.initFile ?? undefined,
                })
            const projectName = res.payload?.projectName || ''
            const created = res.payload?.items || []
            const label = this.target === 'swarm' ? '服务' : '容器'
            this.portal.showNotification('success', `${projectName} 部署成功，已创建 ${created.length} 个${label}`)

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
        this.initURL = ''
        this.initFile = null
        this.target = 'docker'
        const input = this.$refs.fileInput as HTMLInputElement | undefined
        if (input) input.value = ''
        this.fromMarketplace = false
    }
}

export default toNative(ComposeDeploy)
</script>

<template>
  <div>
    <div class="card">
      <!-- Toolbar -->
      <div class="card-toolbar">
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="page-icon bg-amber-500">
              <i class="fas fa-file-code text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800 truncate">Compose 部署</h1>
              <p class="text-xs text-slate-500">直接粘贴 compose.yml 或从应用市场选择模板</p>
            </div>
          </div>
          <div class="flex items-center gap-2 flex-shrink-0">
            <button type="button" :disabled="loading" class="btn btn-secondary" @click="resetForm()">
              <i class="fas fa-rotate-left"></i>清空
            </button>
            <button v-if="portal.hasPerm('POST /api/compose/docker')" type="button" class="btn btn-amber" @click="openMarketplace()">
              <i class="fas fa-store"></i>应用市场
            </button>
          </div>
        </div>
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="page-icon bg-amber-500">
              <i class="fas fa-file-code text-white"></i>
            </div>
            <div class="min-w-0 flex-1">
              <h1 class="text-lg font-semibold text-slate-800 truncate">Compose 部署</h1>
              <p class="text-xs text-slate-500 truncate">粘贴 compose.yml 或从应用市场选择</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <button type="button" :disabled="loading" class="btn btn-secondary w-9 h-9 !px-0" title="清空" @click="resetForm()">
              <i class="fas fa-rotate-left"></i>
            </button>
            <button v-if="portal.hasPerm('POST /api/compose/docker')" type="button" class="btn btn-amber w-9 h-9 !px-0" title="从应用市场选择" @click="openMarketplace()">
              <i class="fas fa-store"></i>
            </button>
          </div>
        </div>
      </div>

      <!-- 表单 -->
      <div class="card-body max-w-3xl space-y-4">
        <!-- 部署目标 -->
        <div class="tab-group inline-flex">
          <button type="button" :class="['tab-btn', target === 'docker' ? 'tab-btn-active text-amber-600' : 'tab-btn-inactive']" @click="selectTarget('docker')">
            <i class="fab fa-docker"></i><span>单机容器</span>
          </button>
          <button
            type="button"
            :disabled="!swarmAvailable"
            :class="['tab-btn',
                     target === 'swarm' ? 'tab-btn-active text-amber-600'
                     : (swarmAvailable ? 'tab-btn-inactive' : 'text-slate-300 cursor-not-allowed')]"
            :title="swarmAvailable ? '' : '当前节点未启用 Swarm'"
            @click="selectTarget('swarm')"
          >
            <i class="fas fa-cubes"></i><span>Swarm 服务</span>
          </button>
        </div>

        <!-- Compose 内容 -->
        <ComposeEditor v-model="content" :disabled="loading" :warning="dynamicWarning" />

        <!-- 附加文件 -->
        <div>
          <label class="form-label">附加文件
            <span class="text-xs font-normal text-slate-400">（选填，部署前解压到项目目录）</span>
          </label>
          <div class="flex items-center gap-2">
            <input v-model="initURL" type="text" placeholder="请输入 zip 下载 URL" class="input flex-1" :disabled="loading || !!initFile" />
            <span class="text-xs text-slate-400 flex-shrink-0">或</span>
            <!-- 隐藏真实 input，用自定义按钮触发 -->
            <label
              :class="['inline-flex items-center gap-1.5 h-[46px] px-4 rounded-xl border text-sm font-medium cursor-pointer transition-colors select-none',
                       loading ? 'opacity-50 cursor-not-allowed' : 'bg-white border-slate-200 text-slate-600 hover:border-slate-300 hover:bg-slate-50',
                       initFile ? 'border-blue-300 bg-blue-50 text-blue-600' : '']"
            >
              <i class="fas fa-paperclip"></i>
              <span>{{ initFile ? initFile.name : '上传 zip' }}</span>
              <i v-if="initFile" class="fas fa-xmark ml-1 hover:text-red-500" @click.prevent="clearInitFile()"></i>
              <input ref="fileInput" type="file" accept=".zip,application/zip" class="hidden" :disabled="loading" @change="onInitFileChange" />
            </label>
          </div>
          <p class="mt-1 text-xs text-slate-400">
            URL 与上传文件二选一，仅支持 .zip 格式
            <template v-if="target === 'swarm'">
              ；
              <span class="mt-1 text-xs text-amber-600">
                Swarm 模式下，附加文件仅落盘到管理节点；如需各节点共享，请将容器数据根目录配置为 NFS 等共享存储
              </span>
            </template>
          </p>
        </div>

        <!-- 操作按钮 -->
        <div class="flex flex-col sm:flex-row sm:items-center gap-3 mt-6 pt-4 border-t border-slate-200">
          <button type="button" :disabled="!canSubmit" class="btn btn-amber rounded-xl whitespace-nowrap flex-shrink-0 self-start" @click="handleDeploy()">
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
