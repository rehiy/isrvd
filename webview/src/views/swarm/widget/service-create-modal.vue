<script setup>
import { ref } from 'vue'

import api from '@/service/api.js'
import ImageSelect from '@/views/docker/widget/image-select.vue'
import BaseModal from '@/component/modal.vue'

const emit = defineEmits(['success'])

const isOpen = ref(false)
const loading = ref(false)
const showAdvanced = ref(false)

const form = ref({
  name: '', image: '', mode: 'replicated', replicas: 1,
  env: '', args: '', network: '', ports: '', mounts: ''
})

const images = ref([])
const networks = ref([])

const loadResources = async () => {
  try {
    const [imgRes, netRes] = await Promise.all([
      api.listImages(false),
      api.listNetworks()
    ])
    images.value = imgRes.payload || []
    networks.value = (netRes.payload || []).filter(n =>
      n.driver === 'overlay' || n.driver === 'host' || n.driver === 'bridge'
    )
  } catch (e) {}
}

const show = () => {
  form.value = { name: '', image: '', mode: 'replicated', replicas: 1, env: '', args: '', network: '', ports: '', mounts: '' }
  showAdvanced.value = false
  isOpen.value = true
  loadResources()
}

const handleConfirm = async () => {
  loading.value = true
  try {
    const parseLines = (s) => s.split('\n').map(l => l.trim()).filter(Boolean)
    const parsePorts = (s) => parseLines(s).map(l => {
      const [pub, rest] = l.split(':')
      const [tgt, proto] = (rest || pub).split('/')
      return { published: parseInt(pub) || 0, target: parseInt(tgt), protocol: proto || 'tcp' }
    })
    const parseMounts = (s) => parseLines(s).map(l => {
      const parts = l.split(':')
      return { type: 'bind', source: parts[0], target: parts[1] || parts[0] }
    })

    await api.swarmCreateService({
      name: form.value.name,
      image: form.value.image,
      mode: form.value.mode,
      replicas: form.value.replicas,
      env: parseLines(form.value.env),
      args: parseLines(form.value.args),
      networks: form.value.network ? [form.value.network] : [],
      ports: parsePorts(form.value.ports),
      mounts: parseMounts(form.value.mounts),
    })
    isOpen.value = false
    emit('success')
  } catch (e) {}
  loading.value = false
}

defineExpose({ show })
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
