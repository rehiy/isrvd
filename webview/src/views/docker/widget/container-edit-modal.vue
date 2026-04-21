<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { ContainerInfo, ImageInfo, NetworkInfo, ContainerCreateRequest, ContainerUpdateRequest, VolumeMapping } from '@/service/types'
import { APP_ACTIONS_KEY, APP_STATE_KEY } from '@/store/state'
import type { AppActions, AppState } from '@/store/state'

import CapSelect from '@/views/docker/widget/cap-select.vue'
import ImageSelect from '@/views/docker/widget/image-select.vue'
import BaseModal from '@/component/modal.vue'
import ComposeEditor from '@/views/compose/widget/compose-editor.vue'

@Component({
    expose: ['show'],
    components: { BaseModal, CapSelect, ImageSelect, ComposeEditor },
    emits: ['success']
})
class ContainerEditModal extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions
    @Inject({ from: APP_STATE_KEY }) readonly state!: AppState

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    isEditMode = false
    showAdvanced = false
    showSecurity = false
    editMode: 'ui' | 'compose' = 'ui'
    composeContent = ''
    composeLoading = false

    formData = {
        image: '', name: '', envStr: '', portsStr: '', cmd: '',
        volumesStr: '', restart: 'always', network: '', memory: '',
        cpus: '', workdir: '', user: '', hostname: '',
        privileged: false, capAdd: [] as string[], capDrop: [] as string[]
    }

    images: ImageInfo[] = []
    networks: NetworkInfo[] = []

    readonly restartOptions = [
        { value: 'always', label: '总是重启' },
        { value: 'unless-stopped', label: '除非手动停止' },
        { value: 'on-failure', label: '失败时重启' },
        { value: 'no', label: '不重启' }
    ]

    // ─── 计算属性 ───
    get networkOptions() {
        const options: { value: string; label: string }[] = [{ value: '', label: '不指定' }]
        this.networks.forEach(net => {
            options.push({ value: net.name, label: `${net.name} (${net.driver})` })
        })
        return options
    }

    // ─── 方法 ───
    async loadImages() {
        try {
            const res = await api.listImages(false)
            this.images = res.payload || []
        } catch (e) {}
    }

    async loadNetworks() {
        try {
            const res = await api.listNetworks()
            this.networks = res.payload || []
        } catch (e) {}
    }

    async show(container?: ContainerInfo) {
        if (container) {
            this.isEditMode = true
            this.editMode = 'ui'
            this.composeContent = ''
            this.modalLoading = true
            this.showAdvanced = true
            try {
                const res = await api.getContainerConfig(container.name)
                const config = res.payload
                if (!config) {
                    this.actions.showNotification('error', '加载容器配置失败: 未获取到配置数据')
                    return
                }
                Object.assign(this.formData, {
                    name: config.name,
                    image: config.image,
                    envStr: (config.env || []).join('\n'),
                    portsStr: Object.entries(config.ports || {}).map(([h, c]) => `${h}:${c}`).join('\n'),
                    volumesStr: (config.volumes || []).map((v: VolumeMapping) => {
                        let s = `${v.hostPath}:${v.containerPath}`
                        if (v.readOnly) s += ':ro'
                        return s
                    }).join('\n'),
                    cmd: (config.cmd || []).join(' '),
                    restart: config.restart || 'always',
                    network: config.network || '',
                    memory: config.memory || '',
                    cpus: config.cpus || '',
                    workdir: config.workdir || '',
                    user: config.user || '',
                    hostname: config.hostname || '',
                    privileged: config.privileged || false,
                    capAdd: config.capAdd || [],
                    capDrop: config.capDrop || []
                })
            } catch (e: unknown) {
                return
            } finally {
                this.modalLoading = false
            }
        } else {
            this.isEditMode = false
            Object.assign(this.formData, {
                image: '', name: '', envStr: '', portsStr: '', cmd: '',
                volumesStr: '', restart: 'always', network: '', memory: '',
                cpus: '', workdir: '', user: '', hostname: '',
                privileged: false, capAdd: [], capDrop: []
            })
        }
        this.isOpen = true
        this.loadImages()
        this.loadNetworks()
    }

    buildRequestData(): ContainerCreateRequest | ContainerUpdateRequest {
        const data: ContainerCreateRequest = {
            image: this.formData.image,
            name: this.formData.name || undefined,
            env: this.formData.envStr ? this.formData.envStr.split('\n').filter(e => e.trim()) : [],
            ports: this.formData.portsStr ? Object.fromEntries(
                this.formData.portsStr.split('\n').filter(p => p.trim()).map(p => {
                    const [hostPort, containerPort] = p.split(':').map(s => s.trim())
                    return [hostPort, containerPort]
                })
            ) : {},
            volumes: this.formData.volumesStr ? this.formData.volumesStr.split('\n').filter(v => v.trim()).map(v => {
                const parts = v.split(':').map(s => s.trim())
                return { hostPath: parts[0], containerPath: parts[1], readOnly: parts[2] === 'ro' }
            }) : [],
            restart: this.formData.restart || 'always',
            network: this.formData.network || undefined,
            memory: this.formData.memory ? parseInt(this.formData.memory) : undefined,
            cpus: this.formData.cpus ? parseFloat(this.formData.cpus) : undefined,
            workdir: this.formData.workdir || undefined,
            user: this.formData.user || undefined,
            hostname: this.formData.hostname || undefined,
            privileged: this.formData.privileged || undefined,
            capAdd: this.formData.capAdd.length > 0 ? this.formData.capAdd : undefined,
            capDrop: this.formData.capDrop.length > 0 ? this.formData.capDrop : undefined
        }
        if (this.formData.cmd && this.formData.cmd.trim()) {
            data.cmd = this.formData.cmd.trim().split(/\s+/)
        }
        return data
    }

    async switchToCompose() {
        if (this.composeContent) return
        this.composeLoading = true
        try {
            const res = await api.getContainerCompose(this.formData.name)
            this.composeContent = res.payload?.content || ''
        } catch (e) {
            this.actions.showNotification('error', '加载 Compose 文件失败')
        } finally {
            this.composeLoading = false
        }
    }

    async handleConfirm() {
        if (this.isEditMode && this.editMode === 'compose') {
            if (!this.composeContent.trim()) return
            this.modalLoading = true
            try {
                await api.composeRedeploy({
                    content: this.composeContent,
                    projectName: this.formData.name
                })
                this.actions.showNotification('success', '容器配置更新成功，已重建容器')
                this.isOpen = false
                this.$emit('success')
            } catch (e) {}
            this.modalLoading = false
            return
        }
        if (!this.formData.image.trim()) return
        this.modalLoading = true
        try {
            if (this.isEditMode) {
                const data = this.buildRequestData() as ContainerUpdateRequest
                data.name = this.formData.name
                await api.updateContainerConfig(data)
                this.actions.showNotification('success', '容器配置更新成功，已重建容器')
            } else {
                await api.createContainer(this.buildRequestData())
                this.actions.showNotification('success', '容器创建成功')
            }
            this.isOpen = false
            this.$emit('success')
        } catch (e) {}
        this.modalLoading = false
    }
}

export default toNative(ContainerEditModal)
</script>

<template>
  <BaseModal
    ref="modalRef"
    v-model="isOpen"
    :title="isEditMode ? '编辑容器配置' : '创建容器'"
    :loading="modalLoading"
    show-footer
    @confirm="handleConfirm"
  >
    <template #confirm-text>{{ isEditMode ? '更新并重建' : '创建' }}</template>

    <!-- 编辑模式：UI / Compose 切换 Tab -->
    <div v-if="isEditMode" class="flex items-center gap-1 bg-slate-100 p-1 rounded-lg mb-4">
      <button
        type="button"
        :class="['flex-1 text-sm px-3 py-1.5 rounded-md transition-colors', editMode === 'ui' ? 'bg-white text-blue-600 shadow-sm font-medium' : 'text-slate-500 hover:text-slate-700']"
        @click="editMode = 'ui'"
      >
        <i class="fas fa-sliders-h mr-1.5"></i>表单模式
      </button>
      <button
        type="button"
        :class="['flex-1 text-sm px-3 py-1.5 rounded-md transition-colors', editMode === 'compose' ? 'bg-white text-blue-600 shadow-sm font-medium' : 'text-slate-500 hover:text-slate-700']"
        @click="editMode = 'compose'; switchToCompose()"
      >
        <i class="fas fa-file-code mr-1.5"></i>Compose 模式
      </button>
    </div>

    <!-- Compose 编辑器 -->
    <div v-if="isEditMode && editMode === 'compose'" class="space-y-3">
      <div class="bg-amber-50 border border-amber-200 rounded-lg p-3">
        <p class="text-sm text-amber-700">
          <i class="fas fa-exclamation-triangle mr-1"></i>
          更新配置后将会重建容器，旧容器将被停止并删除
        </p>
      </div>
      <div v-if="composeLoading" class="flex items-center justify-center py-12 text-slate-400">
        <i class="fas fa-spinner fa-spin mr-2"></i>加载 Compose 文件中...
      </div>
      <div v-else>
        <label class="block text-sm font-medium text-slate-700 mb-2">
          <i class="fas fa-file-code mr-1 text-slate-400"></i>docker-compose.yml
        </label>
        <ComposeEditor v-model="composeContent" height="400px" />
        <p class="mt-1 text-xs text-slate-400">直接编辑 Compose 文件内容，保存后将重建容器</p>
      </div>
    </div>

    <!-- 表单模式 -->
    <form v-if="!isEditMode || editMode === 'ui'" @submit.prevent="handleConfirm" class="space-y-4">
      <!-- 编辑模式提示 -->
      <div v-if="isEditMode" class="bg-amber-50 border border-amber-200 rounded-lg p-3">
        <p class="text-sm text-amber-700">
          <i class="fas fa-exclamation-triangle mr-1"></i>
          更新配置后将会重建容器，旧容器将被停止并删除
        </p>
      </div>

      <!-- 基础设置 -->
      <div class="grid grid-cols-2 gap-3">
        <div class="col-span-2">
          <label class="block text-sm font-medium text-slate-700 mb-2">镜像 <span class="text-red-500">*</span></label>
          <ImageSelect v-model="formData.image" :images="images" placeholder="选择或输入镜像名称" />
        </div>
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">容器名称</label>
          <input type="text" v-model="formData.name" placeholder="my-container" class="input" :disabled="isEditMode" />
        </div>
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">网络模式</label>
          <select v-model="formData.network" class="input">
            <option v-for="opt in networkOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
          </select>
        </div>
      </div>

      <!-- 端口映射 -->
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">端口映射</label>
        <textarea v-model="formData.portsStr" rows="2" placeholder="8080:80" class="input font-mono text-sm"></textarea>
        <p class="mt-1 text-xs text-slate-400">主机端口:容器端口</p>
      </div>

      <!-- 目录映射 -->
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">目录映射</label>
        <textarea v-model="formData.volumesStr" rows="2" placeholder="data:/app/data:ro" class="input font-mono text-sm"></textarea>
        <p class="mt-1 text-xs text-slate-400">主机路径:容器路径[:ro]，相对路径自动补全</p>
      </div>

      <!-- 环境变量 -->
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">环境变量</label>
        <textarea v-model="formData.envStr" rows="2" placeholder="KEY=value" class="input font-mono text-sm"></textarea>
      </div>

      <!-- 高级选项 -->
      <div class="border-t border-slate-200 pt-4">
        <button type="button" @click="showAdvanced = !showAdvanced" class="flex items-center gap-2 text-sm text-slate-600 hover:text-slate-800">
          <i :class="['fas fa-chevron-down text-xs transition-transform', showAdvanced ? 'rotate-180' : '']"></i>
          高级选项
        </button>
        <div v-if="showAdvanced" class="mt-4 space-y-4">
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">启动命令</label>
            <input type="text" v-model="formData.cmd" placeholder="覆盖默认命令" class="input font-mono text-sm" />
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-sm font-medium text-slate-700 mb-2">重启策略</label>
              <select v-model="formData.restart" class="input">
                <option v-for="opt in restartOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
              </select>
            </div>
            <div>
              <label class="block text-sm font-medium text-slate-700 mb-2">主机名</label>
              <input type="text" v-model="formData.hostname" placeholder="容器主机名" class="input" />
            </div>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-sm font-medium text-slate-700 mb-2">内存限制 (MB)</label>
              <input type="number" v-model="formData.memory" placeholder="512" class="input" />
            </div>
            <div>
              <label class="block text-sm font-medium text-slate-700 mb-2">CPU 限制 (核心)</label>
              <input type="number" step="0.1" v-model="formData.cpus" placeholder="1.5" class="input" />
            </div>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-sm font-medium text-slate-700 mb-2">工作目录</label>
              <input type="text" v-model="formData.workdir" placeholder="/app" class="input" />
            </div>
            <div>
              <label class="block text-sm font-medium text-slate-700 mb-2">运行用户</label>
              <input type="text" v-model="formData.user" placeholder="root" class="input" />
            </div>
          </div>
        </div>
      </div>

      <!-- 安全配置 -->
      <div class="border-t border-slate-200 pt-4">
        <button type="button" @click="showSecurity = !showSecurity" class="flex items-center gap-2 text-sm text-slate-600 hover:text-slate-800">
          <i :class="['fas fa-chevron-down text-xs transition-transform', showSecurity ? 'rotate-180' : '']"></i>
          安全配置
          <span v-if="formData.privileged || formData.capAdd?.length || formData.capDrop?.length" class="inline-flex items-center px-1.5 py-0.5 rounded-full text-xs font-medium bg-amber-100 text-amber-700">
            {{ [formData.privileged ? '特权' : '', formData.capAdd?.length ? `+${formData.capAdd.length}` : '', formData.capDrop?.length ? `-${formData.capDrop.length}` : ''].filter(Boolean).join(' ') }}
          </span>
        </button>
        <div v-if="showSecurity" class="mt-4 space-y-4">
          <div class="flex items-center gap-3">
            <label class="relative inline-flex items-center cursor-pointer">
              <input type="checkbox" v-model="formData.privileged" class="sr-only peer" />
              <div class="w-10 h-5 bg-slate-200 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-amber-300 rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-amber-500"></div>
              <span class="ml-2 text-sm text-slate-700">特权模式</span>
            </label>
            <span class="text-xs text-slate-400">⚠️ 赋予容器所有主机权限，谨慎使用</span>
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">添加权限 (CapAdd)</label>
            <CapSelect v-model="formData.capAdd" />
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">移除权限 (CapDrop)</label>
            <CapSelect v-model="formData.capDrop" />
          </div>
        </div>
      </div>
    </form>
  </BaseModal>
</template>
