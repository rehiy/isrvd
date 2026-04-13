<script setup>
import { computed, onMounted, ref } from 'vue'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'
import { inject } from 'vue'

import BaseModal from '@/component/modal.vue'

const actions = inject(APP_ACTIONS_KEY)

// 仓库列表
const registries = ref([])
const loading = ref(false)

// 本地镜像列表（用于选择推送/同步的镜像）
const localImages = ref([])

// 推送镜像模态框
const pushOpen = ref(false)
const pushLoading = ref(false)
const pushForm = ref({ image: '', registryUrl: '', namespace: '' })

// 拉取镜像模态框（从仓库拉取到本地）
const pullOpen = ref(false)
const pullLoading = ref(false)
const pullForm = ref({ image: '', registryUrl: '' })

// 加载仓库列表
const loadRegistries = async () => {
  loading.value = true
  try {
    const res = await api.listRegistries()
    registries.value = res.payload || []
  } catch (e) {
    actions.showNotification('error', '加载仓库列表失败')
  }
  loading.value = false
}

// 加载本地镜像列表
const loadLocalImages = async () => {
  try {
    const res = await api.listImages(false)
    localImages.value = (res.payload || []).filter(img => img.repoTags && img.repoTags.length > 0)
  } catch (e) {}
}

// 可选的镜像标签列表（扁平化）
const imageTagOptions = computed(() => {
  const tags = []
  for (const img of localImages.value) {
    for (const tag of img.repoTags) {
      if (tag !== '<none>:<none>') {
        tags.push(tag)
      }
    }
  }
  return tags
})

// 打开推送模态框
const openPushModal = (registry = null) => {
  pushForm.value = {
    image: '',
    registryUrl: registry ? registry.url : '',
    namespace: ''
  }
  loadLocalImages()
  pushOpen.value = true
}

// 计算推送目标引用预览
const pushTargetPreview = computed(() => {
  const registry = pushForm.value.registryUrl || 'registry'
  const ns = pushForm.value.namespace ? pushForm.value.namespace + '/' : ''
  // 从镜像名中提取短名称（去掉仓库前缀）
  let imageName = pushForm.value.image || 'image:tag'
  const lastSlash = imageName.lastIndexOf('/')
  if (lastSlash >= 0) {
    imageName = imageName.substring(lastSlash + 1)
  }
  return registry + '/' + ns + imageName
})

// 执行推送
const handlePush = async () => {
  if (!pushForm.value.image.trim() || !pushForm.value.registryUrl.trim()) return
  pushLoading.value = true
  try {
    await api.pushImage(pushForm.value.image, pushForm.value.registryUrl, pushForm.value.namespace.trim() || undefined)
    actions.showNotification('success', '镜像推送成功')
    pushOpen.value = false
  } catch (e) {}
  pushLoading.value = false
}

// 打开拉取镜像模态框（从仓库拉取到本地）
const openPullModal = (registry = null) => {
  pullForm.value = {
    image: '',
    registryUrl: registry ? registry.url : ''
  }
  pullOpen.value = true
}

// 执行拉取（从仓库拉取到本地）
const handlePull = async () => {
  if (!pullForm.value.image.trim() || !pullForm.value.registryUrl.trim()) return
  pullLoading.value = true
  try {
    await api.pullFromRegistry(pullForm.value.image, pullForm.value.registryUrl)
    actions.showNotification('success', '镜像拉取成功')
    pullOpen.value = false
  } catch (e) {}
  pullLoading.value = false
}

onMounted(() => {
  loadRegistries()
})
</script>

<template>
  <div>
    <div class="card mb-4">
      <!-- Toolbar Bar -->
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-6 py-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-purple-500 flex items-center justify-center">
              <i class="fas fa-warehouse text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">镜像仓库</h1>
              <p class="text-xs text-slate-500">管理镜像仓库，同步和推送镜像</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <button @click="loadRegistries()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button @click="openPullModal()" class="px-3 py-1.5 rounded-lg bg-teal-500 hover:bg-teal-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-download"></i>拉取
            </button>
            <button @click="openPushModal()" class="px-3 py-1.5 rounded-lg bg-blue-500 hover:bg-blue-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-upload"></i>推送
            </button>
          </div>
        </div>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <!-- Registry Table -->
      <div v-else-if="registries.length > 0" class="overflow-x-auto">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-50 border-b border-slate-200">
              <th class="w-1/4 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">名称</th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">地址</th>
              <th class="w-28 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">认证</th>
              <th class="w-32 px-4 py-3 text-center text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-slate-100">
            <tr v-for="reg in registries" :key="reg.url" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3">
                <div class="flex items-center gap-2">
                  <div class="w-8 h-8 rounded-lg bg-purple-400 flex items-center justify-center">
                    <i class="fas fa-warehouse text-white text-sm"></i>
                  </div>
                  <span class="font-medium text-slate-800">{{ reg.name }}</span>
                </div>
              </td>
              <td class="px-4 py-3"><code class="text-xs bg-slate-100 px-2 py-1 rounded">{{ reg.url }}</code></td>
              <td class="px-4 py-3">
                <span v-if="reg.username" class="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-green-50 text-green-700">
                  <i class="fas fa-user mr-1"></i>{{ reg.username }}
                </span>
                <span v-else class="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-slate-100 text-slate-500">
                  <i class="fas fa-lock-open mr-1"></i>匿名
                </span>
              </td>
              <td class="px-4 py-3">
                <div class="flex justify-center items-center gap-0.5">
                  <button @click="openPullModal(reg)" class="btn-icon text-teal-600 hover:bg-teal-50" title="拉取镜像">
                    <i class="fas fa-download text-xs"></i>
                  </button>
                  <button @click="openPushModal(reg)" class="btn-icon text-blue-600 hover:bg-blue-50" title="推送镜像">
                    <i class="fas fa-upload text-xs"></i>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Empty State -->
      <div v-else class="flex flex-col items-center justify-center py-20">
        <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
          <i class="fas fa-warehouse text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">暂无镜像仓库</p>
        <p class="text-sm text-slate-400">请在 config.yml 中配置 docker.registries</p>
      </div>
    </div>

    <!-- 推送镜像模态框 -->
    <BaseModal
      v-model="pushOpen"
      title="推送镜像到仓库"
      :loading="pushLoading"
      show-footer
      confirm-text="开始推送"
      @confirm="handlePush"
    >
      <form @submit.prevent="handlePush" class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">本地镜像</label>
          <select v-model="pushForm.image" class="input" required>
            <option value="" disabled>请选择镜像</option>
            <option v-for="tag in imageTagOptions" :key="tag" :value="tag">{{ tag }}</option>
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">目标仓库地址</label>
          <select v-model="pushForm.registryUrl" class="input" required>
            <option value="" disabled>请选择仓库</option>
            <option v-for="reg in registries" :key="reg.url" :value="reg.url">{{ reg.name }} ({{ reg.url }})</option>
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">命名空间 <span class="text-slate-400 font-normal">(可选)</span></label>
          <input type="text" v-model="pushForm.namespace" placeholder="例如: myteam" class="input" />
          <p class="mt-1 text-xs text-slate-400">镜像将被推送为: {{ pushTargetPreview }}</p>
        </div>
      </form>
    </BaseModal>

    <!-- 拉取镜像模态框（从仓库拉取到本地） -->
    <BaseModal
      v-model="pullOpen"
      title="从仓库拉取镜像"
      :loading="pullLoading"
      show-footer
      confirm-text="开始拉取"
      @confirm="handlePull"
    >
      <form @submit.prevent="handlePull" class="space-y-4">
        <div class="bg-blue-50 border border-blue-200 rounded-lg p-3 text-xs text-blue-700">
          <i class="fas fa-info-circle mr-1"></i>
          从指定镜像仓库拉取镜像到本地（自动使用仓库认证信息）
        </div>
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">源仓库地址</label>
          <select v-model="pullForm.registryUrl" class="input" required>
            <option value="" disabled>请选择仓库</option>
            <option v-for="reg in registries" :key="reg.url" :value="reg.url">{{ reg.name }} ({{ reg.url }})</option>
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">镜像名称</label>
          <input type="text" v-model="pullForm.image" placeholder="输入镜像名称，如 myapp:latest" class="input" required />
          <p class="mt-1 text-xs text-slate-400">
            将拉取: {{ pullForm.registryUrl || 'registry' }}/{{ pullForm.image || 'image:tag' }}
          </p>
        </div>
      </form>
    </BaseModal>
  </div>
</template>
