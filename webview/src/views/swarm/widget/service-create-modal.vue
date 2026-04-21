<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { DockerImageInfo, DockerNetworkInfo } from '@/service/types'
import ImageSelect from '@/views/docker/widget/image-select.vue'
import BaseModal from '@/component/modal.vue'

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
    form = { name: '', image: '', mode: 'replicated', replicas: 1, env: '', args: '', network: '', ports: '', mounts: '' }
    images: DockerImageInfo[] = []
    networks: DockerNetworkInfo[] = []

    // ─── 方法 ───
    async loadResources() {
        try {
            const [imgRes, netRes] = await Promise.all([api.listImages(false), api.listNetworks()])
            this.images = imgRes.payload || []
            this.networks = (netRes.payload || []).filter((n: DockerNetworkInfo) =>
                n.driver === 'overlay' || n.driver === 'host' || n.driver === 'bridge'
            )
        } catch (e) {}
    }

    show() {
        this.form = { name: '', image: '', mode: 'replicated', replicas: 1, env: '', args: '', network: '', ports: '', mounts: '' }
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

            await api.swarmCreateService({
                name: this.form.name,
                image: this.form.image,
                mode: this.form.mode,
                replicas: this.form.replicas,
                env: parseLines(this.form.env),
                args: parseLines(this.form.args),
                networks: this.form.network ? [this.form.network] : [],
                ports: parsePorts(this.form.ports),
                mounts: parseMounts(this.form.mounts)
            })
            this.isOpen = false
            this.$emit('success')
        } catch (e) {}
        this.loading = false
    }
}

export default toNative(ServiceCreateModal)
</script>

<template>
  <BaseModal v-model="isOpen" title="创建服务" :loading="loading" show-footer @confirm="handleConfirm">
    <template #confirm-text>创建</template>
    <form @submit.prevent="handleConfirm" class="space-y-4">
      <!-- 基础设置 -->
      <div class="grid grid-cols-2 gap-3">
        <div class="col-span-2">
          <label class="block text-sm font-medium text-slate-700 mb-2">镜像 <span class="text-red-500">*</span></label>
          <ImageSelect v-model="form.image" :images="images" placeholder="选择或输入镜像名称" />
        </div>
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">服务名 <span class="text-red-500">*</span></label>
          <input v-model="form.name" type="text" placeholder="my-service" class="input" />
        </div>
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">网络</label>
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
        <label class="block text-sm font-medium text-slate-700 mb-2">端口映射</label>
        <textarea v-model="form.ports" rows="2" placeholder="8080:80/tcp" class="input font-mono text-sm"></textarea>
        <p class="mt-1 text-xs text-slate-400">每行一条，格式：宿主端口:容器端口/协议</p>
      </div>

      <!-- 目录挂载 -->
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">目录挂载</label>
        <textarea v-model="form.mounts" rows="2" placeholder="/data:/app/data" class="input font-mono text-sm"></textarea>
        <p class="mt-1 text-xs text-slate-400">每行一条，格式：宿主路径:容器路径</p>
      </div>

      <!-- 环境变量 -->
      <div>
        <label class="block text-sm font-medium text-slate-700 mb-2">环境变量</label>
        <textarea v-model="form.env" rows="2" placeholder="KEY=value" class="input font-mono text-sm"></textarea>
      </div>

      <!-- 高级选项 -->
      <div class="border-t border-slate-200 pt-4">
        <button type="button" @click="showAdvanced = !showAdvanced" class="flex items-center gap-2 text-sm text-slate-600 hover:text-slate-800">
          <i :class="['fas fa-chevron-down text-xs transition-transform', showAdvanced ? 'rotate-180' : '']"></i>
          高级选项
        </button>
        <div v-if="showAdvanced" class="mt-4 space-y-4">
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-sm font-medium text-slate-700 mb-2">模式</label>
              <select v-model="form.mode" class="input">
                <option value="replicated">Replicated</option>
                <option value="global">Global</option>
              </select>
            </div>
            <div v-if="form.mode === 'replicated'">
              <label class="block text-sm font-medium text-slate-700 mb-2">副本数</label>
              <input v-model.number="form.replicas" type="number" min="1" class="input" />
            </div>
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">启动参数</label>
            <input v-model="form.args" type="text" placeholder="覆盖默认启动参数" class="input font-mono text-sm" />
          </div>
        </div>
      </div>
    </form>
  </BaseModal>
</template>
