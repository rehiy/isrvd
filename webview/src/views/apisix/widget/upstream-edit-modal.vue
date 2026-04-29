<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import api from '@/service/api'
import type {
    ApisixCreateUpstreamRequest,
    ApisixRouteUpstreamFormNode,
    ApisixUpstream,
    ApisixUpstreamHashOn,
    ApisixUpstreamType,
    ApisixUpdateUpstreamRequest,
    DockerContainerInfo
} from '@/service/types'

import { normalizeUpstreamFormNodes, normalizeUpstreamType } from '@/helper/apisix'

import BaseModal from '@/component/modal.vue'
import HostSelect from './host-select.vue'
import PortSelect from './port-select.vue'

const UPSTREAM_TYPE_OPTIONS: Array<{ value: ApisixUpstreamType; label: string; desc: string }> = [
    { value: 'roundrobin', label: 'roundrobin', desc: '加权轮询' },
    { value: 'chash', label: 'chash', desc: '一致性哈希' },
    { value: 'ewma', label: 'ewma', desc: '最小延迟' },
    { value: 'least_conn', label: 'least_conn', desc: '最少连接' }
]

const HASH_ON_OPTIONS: Array<{ value: ApisixUpstreamHashOn; label: string; keyPlaceholder: string; keyHint: string }> = [
    { value: 'vars', label: '变量', keyPlaceholder: 'remote_addr', keyHint: '按 NGINX 变量取值，例如 remote_addr' },
    { value: 'header', label: '请求头', keyPlaceholder: 'X-User-ID', keyHint: '按请求头名称取值' },
    { value: 'cookie', label: 'Cookie', keyPlaceholder: 'session_id', keyHint: '按 Cookie 名称取值' },
    { value: 'consumer', label: 'Consumer', keyPlaceholder: '', keyHint: '按 APISIX Consumer 取值，通常无需 key' },
    { value: 'vars_combinations', label: '变量组合', keyPlaceholder: 'remote_addr,uri', keyHint: '按多个变量组合取值' }
]

const createNode = (): ApisixRouteUpstreamFormNode => ({ host: '', port: '', weight: 1 })

const defaultFormData = () => ({
    id: '',
    name: '',
    desc: '',
    type: 'roundrobin' as ApisixUpstreamType,
    hash_on: 'vars' as ApisixUpstreamHashOn,
    key: 'remote_addr',
    nodes: [createNode()] as ApisixRouteUpstreamFormNode[],
    timeout_connect: '' as string | number,
    timeout_send: '' as string | number,
    timeout_read: '' as string | number
})

@Component({
    expose: ['show'],
    components: { BaseModal, HostSelect, PortSelect },
    emits: ['success']
})
class UpstreamEditModal extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    isOpen = false
    modalLoading = false
    isEditMode = false
    formData = defaultFormData()
    containers: DockerContainerInfo[] = []

    readonly upstreamTypeOptions = UPSTREAM_TYPE_OPTIONS
    readonly hashOnOptions = HASH_ON_OPTIONS

    get selectedHashOnOption() {
        return this.hashOnOptions.find(item => item.value === this.formData.hash_on) || this.hashOnOptions[0]
    }

    resetForm() {
        this.formData = defaultFormData()
    }

    async show(upstream: ApisixUpstream | null = null) {
        await this.loadContainers()
        if (upstream) {
            const upstreamType = normalizeUpstreamType(upstream.type)
            this.isEditMode = true
            this.formData = {
                id: upstream.id || '',
                name: upstream.name || '',
                desc: upstream.desc || '',
                type: upstreamType,
                hash_on: upstream.hash_on || 'vars',
                key: upstream.key || (upstreamType === 'chash' ? 'remote_addr' : ''),
                nodes: normalizeUpstreamFormNodes(upstream),
                timeout_connect: upstream.timeout?.connect ?? '',
                timeout_send: upstream.timeout?.send ?? '',
                timeout_read: upstream.timeout?.read ?? ''
            }
        } else {
            this.isEditMode = false
            this.resetForm()
        }
        this.isOpen = true
    }

    addNode() {
        this.formData.nodes.push(createNode())
    }

    removeNode(index: number) {
        if (this.formData.nodes.length <= 1) return
        this.formData.nodes.splice(index, 1)
    }

    getPortsByHost(host: string): string[] {
        return this.containers.find(c => c.name === host.trim())?.ports || []
    }

    updateNode(index: number, field: 'host' | 'port', value: string) {
        const node = this.formData.nodes[index]
        if (!node) return
        node[field] = value
        if (field === 'host' && value.trim()) {
            const port = (this.getPortsByHost(value)[0] || '').split('/')[0].split(':').pop() || ''
            if (port) node.port = port
        }
    }

    async loadContainers() {
        try {
            const res = await api.listContainers()
            this.containers = (res.payload || []).filter(c => c.state === 'running')
        } catch {
            this.containers = []
        }
    }

    buildPayload(): ApisixCreateUpstreamRequest | ApisixUpdateUpstreamRequest {
        const nodes = this.formData.nodes
            .map(node => ({
                host: node.host.trim(),
                port: Number(node.port),
                weight: Number(node.weight) >= 0 ? Number(node.weight) : 1
            }))
            .filter(node => node.host && node.port > 0)

        const payload: ApisixCreateUpstreamRequest = {
            name: this.formData.name.trim(),
            desc: this.formData.desc.trim(),
            type: this.formData.type,
            nodes
        }

        const connect = Number(this.formData.timeout_connect) || 0
        const send = Number(this.formData.timeout_send) || 0
        const read = Number(this.formData.timeout_read) || 0
        if (connect > 0 || send > 0 || read > 0) {
            payload.timeout = { connect: connect || undefined, send: send || undefined, read: read || undefined }
        }

        if (this.formData.type === 'chash') {
            payload.hash_on = this.formData.hash_on
            if (this.formData.key.trim()) payload.key = this.formData.key.trim()
        }

        return payload
    }

    async handleConfirm() {
        if (!this.formData.name.trim()) {
            this.actions.showNotification('error', '上游名称不能为空')
            return
        }
        const payload = this.buildPayload()
        const nodes = Array.isArray(payload.nodes) ? payload.nodes : []
        if (nodes.length === 0) {
            this.actions.showNotification('error', '至少需要配置一个有效节点')
            return
        }

        this.modalLoading = true
        try {
            if (this.isEditMode) {
                await api.apisixUpdateUpstream(this.formData.id, payload)
            } else {
                await api.apisixCreateUpstream(payload)
            }
            this.isOpen = false
            this.$emit('success')
        } catch (e: unknown) {
            this.actions.showNotification('error', (e instanceof Error ? e.message : '') || '操作失败')
        }
        this.modalLoading = false
    }
}

export default toNative(UpstreamEditModal)
</script>

<template>
  <BaseModal
    v-model="isOpen"
    :title="isEditMode ? '编辑上游' : '创建上游'"
    :loading="modalLoading"
  >
    <div class="max-w-3xl space-y-4 p-1">
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">名称 <span class="text-red-500">*</span></label>
          <input v-model="formData.name" type="text" class="input" placeholder="上游名称" />
        </div>
        <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">负载均衡策略</label>
          <select v-model="formData.type" class="input">
            <option v-for="item in upstreamTypeOptions" :key="item.value" :value="item.value">{{ item.label }} - {{ item.desc }}</option>
          </select>
        </div>
      </div>

      <div>
        <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">描述</label>
        <textarea v-model="formData.desc" rows="2" class="input" placeholder="上游描述"></textarea>
      </div>

      <div>
        <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">超时时间（秒）</label>
        <div class="grid grid-cols-1 md:grid-cols-3 gap-2">
          <input v-model.number="formData.timeout_connect" type="number" min="0" class="input" placeholder="连接 connect" />
          <input v-model.number="formData.timeout_send" type="number" min="0" class="input" placeholder="发送 send" />
          <input v-model.number="formData.timeout_read" type="number" min="0" class="input" placeholder="读取 read" />
        </div>
        <p class="text-xs text-slate-400 mt-1">留空或 0 表示使用 APISIX 默认超时。</p>
      </div>

      <div v-if="formData.type === 'chash'" class="grid grid-cols-1 md:grid-cols-2 gap-4 rounded-xl border border-emerald-100 bg-emerald-50/40 p-4">
        <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">Hash On</label>
          <select v-model="formData.hash_on" class="input bg-white">
            <option v-for="item in hashOnOptions" :key="item.value" :value="item.value">{{ item.label }}</option>
          </select>
        </div>
        <div>
          <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">Key</label>
          <input v-model="formData.key" type="text" class="input bg-white" :placeholder="selectedHashOnOption.keyPlaceholder" />
          <p class="text-xs text-slate-400 mt-1">{{ selectedHashOnOption.keyHint }}</p>
        </div>
      </div>

      <div class="rounded-xl border border-slate-200 overflow-hidden">
        <div class="flex items-center justify-between bg-slate-50 px-4 py-3 border-b border-slate-200">
          <div>
            <h3 class="text-sm font-semibold text-slate-700">节点配置</h3>
            <p class="text-xs text-slate-400 mt-0.5">填写后端服务地址和权重</p>
          </div>
          <button type="button" @click="addNode()" class="px-2.5 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-emerald-600 text-xs font-medium flex items-center gap-1">
            <i class="fas fa-plus"></i>添加节点
          </button>
        </div>
        <div class="divide-y divide-slate-100">
          <div v-for="(node, index) in formData.nodes" :key="index" class="grid grid-cols-12 gap-2 p-3 items-end">
            <div class="col-span-12 md:col-span-5">
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">Host</label>
              <HostSelect :model-value="node.host" :containers="containers" placeholder="127.0.0.1 或 容器名" @update:modelValue="updateNode(index, 'host', $event)" />
            </div>
            <div class="col-span-5 md:col-span-3">
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">Port</label>
              <PortSelect :model-value="String(node.port || '')" :ports="getPortsByHost(node.host)" placeholder="8080" @update:modelValue="updateNode(index, 'port', $event)" />
            </div>
            <div class="col-span-5 md:col-span-3">
              <label class="block text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">权重</label>
              <input v-model.number="node.weight" type="number" min="0" class="input" />
            </div>
            <div class="col-span-2 md:col-span-1 flex h-12 items-center justify-end">
              <button type="button" @click="removeNode(index)" :disabled="formData.nodes.length <= 1" class="btn-icon text-red-600 hover:bg-red-50 disabled:text-slate-300 disabled:cursor-not-allowed" title="删除节点">
                <i class="fas fa-trash text-xs"></i>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-2">
        <button @click="isOpen = false" class="px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-200 rounded-lg hover:bg-slate-50">取消</button>
        <button @click="handleConfirm()" :disabled="modalLoading" class="px-4 py-2 text-sm font-medium text-white bg-emerald-500 rounded-lg hover:bg-emerald-600 disabled:opacity-50 shadow-sm">
          <i v-if="modalLoading" class="fas fa-spinner fa-spin mr-1"></i>{{ isEditMode ? '保存' : '创建' }}
        </button>
      </div>
    </template>
  </BaseModal>
</template>
