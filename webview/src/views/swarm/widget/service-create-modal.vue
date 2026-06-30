<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { DockerImageInfo, DockerNetworkInfo, SwarmNodeInfo } from '@/service/types'

import BaseModal from '@/component/modal.vue'

import ImageSelect from '@/views/docker/widget/image-select.vue'

@Component({
    expose: ['show'],
    components: { BaseModal, ImageSelect },
    emits: ['success']
})
class ServiceCreateModal extends Vue {
    // ─── 数据属性 ───
    isOpen = false
    loading = false
    showAdvanced = false
    form = { name: '', image: '', mode: 'replicated', replicas: 1, env: '', args: '', network: '', node: '', ports: '', mounts: '' }
    images: DockerImageInfo[] = []
    networks: DockerNetworkInfo[] = []
    nodes: SwarmNodeInfo[] = []

    // ─── 方法 ───
    async loadResources() {
        const [imgRes, netRes, nodeRes] = await Promise.allSettled([
            api.dockerImageList(false),
            api.dockerNetworkList(),
            api.swarmNodeList(),
        ])
        if (imgRes.status === 'fulfilled') {
            this.images = imgRes.value.payload || []
        }
        if (netRes.status === 'fulfilled') {
            this.networks = (netRes.value.payload || []).filter((n: DockerNetworkInfo) =>
                n.driver === 'overlay' || n.driver === 'host' || n.driver === 'bridge'
            )
            if (!this.form.network && this.networks.some(n => n.name === 'sdnet')) {
                this.form.network = 'sdnet'
            }
        }
        if (nodeRes.status === 'fulfilled') {
            this.nodes = (nodeRes.value.payload || []).filter((node: SwarmNodeInfo) =>
                node.state === 'ready' && node.availability === 'active'
            )
        }
    }

    show() {
        this.form = { name: '', image: '', mode: 'replicated', replicas: 1, env: '', args: '', network: '', node: '', ports: '', mounts: '' }
        this.showAdvanced = false
        this.isOpen = true
        this.loadResources()
    }

    async handleConfirm() {
        this.loading = true
        try {
            const parseLines = (s: string) => s.split('\n').map(l => l.trim()).filter(Boolean)
            const parsePorts = (s: string) => parseLines(s).map(l => {
                const [pub, rest] = l.split(':')
                const [tgt, proto] = (rest || pub).split('/')
                const publishedPort = parseInt(pub) || 0
                const targetPort = parseInt(tgt)
                return { protocol: proto || 'tcp', targetPort, publishedPort, publishMode: 'ingress' }
            })
            const parseMounts = (s: string) => parseLines(s).map(l => {
                const parts = l.split(':')
                return { type: 'bind', source: parts[0], target: parts[1] || parts[0], readOnly: false }
            })

            await api.swarmServiceCreate({
                name: this.form.name,
                image: this.form.image,
                mode: this.form.mode,
                replicas: this.form.replicas,
                env: parseLines(this.form.env),
                args: parseLines(this.form.args),
                networks: this.form.network ? [this.form.network] : [],
                constraints: this.form.node ? ['node.hostname == ' + this.form.node] : [],
                ports: parsePorts(this.form.ports),
                mounts: parseMounts(this.form.mounts)
            })
            this.isOpen = false
            this.$emit('success')
        } catch {}
        this.loading = false
    }
}

export default toNative(ServiceCreateModal)
</script>

<template>
  <BaseModal v-model="isOpen" title="新建服务" :loading="loading" confirm-class="btn-emerald" show-footer @confirm="handleConfirm">
    <form class="space-y-4" @submit.prevent="handleConfirm">
      <!-- 基础设置 -->
      <div class="grid grid-cols-2 gap-3">
        <div class="col-span-2">
          <label class="form-label">镜像 <span class="text-red-500">*</span></label>
          <ImageSelect v-model="form.image" :images="images" placeholder="请输入或选择镜像名" />
        </div>
        <div>
          <label class="form-label">服务名 <span class="text-red-500">*</span></label>
          <input v-model="form.name" type="text" placeholder="请输入服务名" class="input" />
        </div>
        <div>
          <label class="form-label">运行节点</label>
          <select v-model="form.node" class="input">
            <option value="">不指定</option>
            <option v-for="node in nodes" :key="node.id" :value="node.hostname">
              {{ node.hostname }} ({{ node.role }})
            </option>
          </select>
        </div>
        <div class="col-span-2">
          <label class="form-label">网络</label>
          <select v-model="form.network" class="input">
            <option value="">不指定</option>
            <option v-for="net in networks" :key="net.id" :value="net.name">
              {{ net.name }} ({{ net.driver }})
            </option>
          </select>
        </div>
      </div>

      <!-- 端口映射 -->
      <div>
        <label class="form-label">端口映射</label>
        <textarea v-model="form.ports" rows="2" placeholder="请输入端口映射（可选）" class="input font-mono text-sm"></textarea>
        <p class="mt-1 text-xs text-slate-400">每行一条，格式：宿主端口:容器端口/协议，例如：8080:80/tcp</p>
      </div>

      <!-- 目录挂载 -->
      <div>
        <label class="form-label">目录挂载</label>
        <textarea v-model="form.mounts" rows="2" placeholder="请输入目录挂载（可选）" class="input font-mono text-sm"></textarea>
        <p class="mt-1 text-xs text-slate-400">每行一条，格式：宿主路径:容器路径，例如：/data:/app/data</p>
      </div>

      <!-- 环境变量 -->
      <div>
        <label class="form-label">环境变量</label>
        <textarea v-model="form.env" rows="2" placeholder="请输入环境变量（可选）" class="input font-mono text-sm"></textarea>
        <p class="mt-1 text-xs text-slate-400">每行一条，格式：KEY=value，例如：APP_ENV=production</p>
      </div>

      <!-- 高级选项 -->
      <div class="border-t border-slate-200 pt-4">
        <button type="button" class="flex items-center gap-2 text-sm text-slate-600 hover:text-slate-800" @click="showAdvanced = !showAdvanced">
          <i :class="['fas fa-chevron-down text-xs transition-transform', showAdvanced ? 'rotate-180' : '']"></i>
          高级选项
        </button>
        <div v-if="showAdvanced" class="mt-4 space-y-4">
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="form-label">模式</label>
              <select v-model="form.mode" class="input">
                <option value="replicated">Replicated</option>
                <option value="global">Global</option>
              </select>
            </div>
            <div v-if="form.mode === 'replicated'">
              <label class="form-label">副本数</label>
              <input v-model.number="form.replicas" type="number" min="1" class="input" />
            </div>
          </div>
          <div>
            <label class="form-label">启动参数</label>
            <input v-model="form.args" type="text" placeholder="请输入启动参数（可选）" class="input font-mono text-sm" />
          </div>
        </div>
      </div>
    </form>

    <template #confirm-text>确认新建</template>
  </BaseModal>
</template>
