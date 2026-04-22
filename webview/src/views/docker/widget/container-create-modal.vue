<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'
import * as yaml from 'js-yaml'

import api from '@/service/api'
import type { DockerImageInfo, DockerNetworkInfo } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import CapSelect from '@/views/docker/widget/cap-select.vue'
import ImageSelect from '@/views/docker/widget/image-select.vue'
import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal, CapSelect, ImageSelect },
    emits: ['success']
})
class ContainerCreateModal extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    isOpen = false
    modalLoading = false
    showAdvanced = false
    showSecurity = false

    formData = {
        image: '', name: '', envStr: '', portsStr: '', cmd: '',
        volumesStr: '', restart: 'always', network: '', memory: '',
        cpus: '', workdir: '', user: '', hostname: '',
        privileged: false, capAdd: [] as string[], capDrop: [] as string[]
    }

    images: DockerImageInfo[] = []
    networks: DockerNetworkInfo[] = []

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

    show() {
        this.resetForm()
        this.isOpen = true
        this.loadImages()
        this.loadNetworks()
    }

    resetForm() {
        Object.assign(this.formData, {
            image: '', name: '', envStr: '', portsStr: '', cmd: '',
            volumesStr: '', restart: 'always', network: '', memory: '',
            cpus: '', workdir: '', user: '', hostname: '',
            privileged: false, capAdd: [], capDrop: []
        })
        this.showAdvanced = false
        this.showSecurity = false
    }

    buildYamlFromForm(): string {
        const svcName = this.formData.name || 'app'
        const svc: Record<string, unknown> = {
            image: this.formData.image,
            restart: this.formData.restart || 'always',
        }

        if (this.formData.cmd.trim()) {
            svc.command = this.formData.cmd.trim().split(/\s+/)
        }
        if (this.formData.envStr.trim()) {
            svc.environment = this.formData.envStr.split('\n').filter(e => e.trim())
        }
        if (this.formData.portsStr.trim()) {
            svc.ports = this.formData.portsStr.split('\n').filter(p => p.trim())
        }
        if (this.formData.volumesStr.trim()) {
            svc.volumes = this.formData.volumesStr.split('\n').filter(v => v.trim())
        }
        if (this.formData.network) {
            svc.networks = [this.formData.network]
        }
        if (this.formData.workdir) svc.working_dir = this.formData.workdir
        if (this.formData.user) svc.user = this.formData.user
        if (this.formData.hostname) svc.hostname = this.formData.hostname
        if (this.formData.privileged) svc.privileged = true
        if (this.formData.capAdd.length > 0) svc.cap_add = this.formData.capAdd
        if (this.formData.capDrop.length > 0) svc.cap_drop = this.formData.capDrop

        if (this.formData.memory || this.formData.cpus) {
            const limits: Record<string, unknown> = {}
            if (this.formData.memory) limits.memory = `${this.formData.memory}m`
            if (this.formData.cpus) limits.cpus = parseFloat(this.formData.cpus)
            svc.deploy = { resources: { limits } }
        }

        const doc: Record<string, unknown> = {
            services: { [svcName]: svc }
        }
        return yaml.dump(doc, { lineWidth: -1 })
    }

    async handleConfirm() {
        const projectName = this.formData.name.trim()
        if (!projectName) {
            this.actions.showNotification('error', '请填写容器名称')
            return
        }

        const content = this.buildYamlFromForm()
        if (!content) return

        this.modalLoading = true
        try {
            await api.composeDeployDocker({ content, projectName })
            this.actions.showNotification('success', '容器创建成功')
            this.isOpen = false
            this.$emit('success')
        } catch (e) {}
        this.modalLoading = false
    }
}

export default toNative(ContainerCreateModal)
</script>

<template>
  <BaseModal
    ref="modalRef"
    v-model="isOpen"
    title="创建容器"
    :loading="modalLoading"
    show-footer
    @confirm="handleConfirm"
  >
    <template #confirm-text>创建</template>

    <form @submit.prevent="handleConfirm" class="space-y-4">
      <!-- 基础设置 -->
      <div class="grid grid-cols-2 gap-3">
        <div class="col-span-2">
          <label class="block text-sm font-medium text-slate-700 mb-2">镜像 <span class="text-red-500">*</span></label>
          <ImageSelect v-model="formData.image" :images="images" placeholder="选择或输入镜像名称" />
        </div>
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">容器名称 <span class="text-red-500">*</span></label>
          <input type="text" v-model="formData.name" placeholder="my-container" required class="input" />
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
        <p class="mt-1 text-xs text-slate-400">主机端口:容器端口，每行一条</p>
      </div>

      <!-- 目录映射 -->
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">目录映射</label>
        <textarea v-model="formData.volumesStr" rows="2" placeholder="data:/app/data:ro" class="input font-mono text-sm"></textarea>
        <p class="mt-1 text-xs text-slate-400">主机路径:容器路径[:ro]，每行一条</p>
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
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-1.5">特权模式</label>
            <select v-model="formData.privileged" class="input">
              <option :value="false">禁用</option>
              <option :value="true">启用（⚠️ 赋予容器所有主机权限，谨慎使用）</option>
            </select>
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
