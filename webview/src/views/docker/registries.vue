<script setup>
import { computed, onMounted, ref } from 'vue'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'
import { inject } from 'vue'

import BaseModal from '@/component/modal.vue'

const actions = inject(APP_ACTIONS_KEY)

// 镜像加速器（来自 Docker daemon）
const daemonMirrors = ref([])
const indexServerAddress = ref('')

// 加载 daemon 镜像源信息
const loadDaemonInfo = async () => {
  try {
    const res = await api.dockerInfo()
    const info = res.payload || {}
    daemonMirrors.value = info.registryMirrors || []
    indexServerAddress.value = info.indexServerAddress || ''
  } catch (e) {}
}

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
const pullForm = ref({ image: '', registryUrl: '', namespace: '' })

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
    registryUrl: registry ? registry.url : '',
    namespace: ''
  }
  pullOpen.value = true
}

// 执行拉取（从仓库拉取到本地）
const handlePull = async () => {
  if (!pullForm.value.image.trim() || !pullForm.value.registryUrl.trim()) return
  pullLoading.value = true
  try {
    await api.pullFromRegistry(pullForm.value.image, pullForm.value.registryUrl, pullForm.value.namespace.trim() || undefined)
    actions.showNotification('success', '镜像拉取成功')
    pullOpen.value = false
  } catch (e) {}
  pullLoading.value = false
}

onMounted(() => {
  loadDaemonInfo()
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
            <button @click="openPullModal()" class="px-3 py-1.5 rounded-lg bg-purple-500 hover:bg-purple-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-download"></i>拉取
            </button>
            <button @click="openPushModal()" class="px-3 py-1.5 rounded-lg bg-purple-500 hover:bg-purple-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
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
      <div v-else>
        <!-- 桌面端表格视图 -->
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="w-1/4 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">名称</th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">地址</th>
                <th class="w-28 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">认证</th>
                <th class="w-32 px-4 py-3 text-right text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <!-- Docker Hub 行（始终显示） -->
              <tr class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3">
                  <div class="flex items-center gap-2">
                    <div class="w-8 h-8 rounded-lg bg-blue-500 flex items-center justify-center">
                      <i class="fab fa-docker text-white text-sm"></i>
                    </div>
                  <div class="flex flex-col">
                      <span class="font-medium text-slate-800">Docker Hub</span>
                      <span class="text-xs text-slate-400 font-normal">默认</span>
                  </div>
                  </div>
                </td>
                <td class="px-4 py-3">
                  <code class="text-xs bg-slate-100 px-2 py-1 rounded">{{ indexServerAddress || 'https://index.docker.io/v1/' }}</code>
                  <template v-if="daemonMirrors.length > 0">
                    <code v-for="mirror in daemonMirrors" :key="mirror" class="ml-1 text-xs bg-sky-50 text-sky-700 px-2 py-1 rounded inline-flex items-center gap-1">
                      <i class="fas fa-bolt text-sky-400 text-xs"></i>{{ mirror }}
                    </code>
                  </template>
                </td>
                <td class="px-4 py-3">
                  <span class="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-slate-100 text-slate-500">
                    <i class="fas fa-lock-open mr-1"></i>匿名
                  </span>
                </td>
                <td class="px-4 py-3"></td>
              </tr>
              <!-- 私有仓库行 -->
              <tr v-for="reg in registries" :key="reg.url" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3">
                  <div class="flex items-center gap-2">
                    <div class="w-8 h-8 rounded-lg bg-purple-400 flex items-center justify-center">
                      <i class="fas fa-warehouse text-white text-sm"></i>
                    </div>
                    <div class="flex flex-col">
                      <span class="font-medium text-slate-800">{{ reg.name }}</span>
                      <span v-if="reg.description" class="text-xs text-slate-400 font-normal">{{ reg.description }}</span>
                    </div>
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
                  <div class="flex justify-end items-center gap-0.5">
                    <button @click="openPullModal(reg)" class="btn-icon text-slate-600 hover:bg-slate-100" title="拉取镜像">
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

        <!-- 移动端卡片视图 -->
        <div class="md:hidden space-y-3">
          <!-- Docker Hub 卡片 -->
          <div class="rounded-xl border border-slate-200 bg-white p-4 transition-all hover:shadow-sm">
            <div class="flex items-center gap-3 mb-3">
              <div class="w-10 h-10 rounded-lg bg-blue-500 flex items-center justify-center">
                <i class="fab fa-docker text-white text-base"></i>
              </div>
              <div class="flex flex-col">
                  <h3 class="font-medium text-slate-800 text-sm">Docker Hub</h3>
                  <span class="text-xs text-slate-400">默认</span>
              </div>
            </div>
            
            <div class="mb-3">
              <p class="text-xs text-slate-500 mb-1">地址</p>
              <div class="flex flex-col gap-1">
                <code class="text-xs bg-slate-100 px-2 py-1 rounded break-all">{{ indexServerAddress || 'https://index.docker.io/v1/' }}</code>
                <template v-if="daemonMirrors.length > 0">
                  <code v-for="mirror in daemonMirrors" :key="mirror" class="text-xs bg-sky-50 text-sky-700 px-2 py-1 rounded inline-flex items-center gap-1">
                    <i class="fas fa-bolt text-sky-400 text-xs"></i>{{ mirror }}
                  </code>
                </template>
              </div>
            </div>
            
            <div class="pt-2 border-t border-slate-100">
              <p class="text-xs text-slate-500 mb-1">认证</p>
              <span class="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-slate-100 text-slate-500">
                <i class="fas fa-lock-open mr-1"></i>匿名
              </span>
            </div>
          </div>

          <!-- 私有仓库卡片 -->
          <div v-for="reg in registries" :key="reg.url" class="rounded-xl border border-slate-200 bg-white p-4 transition-all hover:shadow-sm">
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-purple-400 flex items-center justify-center">
                  <i class="fas fa-warehouse text-white text-base"></i>
                </div>
                <div class="flex flex-col">
                  <h3 class="font-medium text-slate-800 text-sm">{{ reg.name }}</h3>
                  <span v-if="reg.description" class="text-xs text-slate-400">{{ reg.description }}</span>
                </div>
              </div>
            </div>
            
            <div class="mb-3">
              <p class="text-xs text-slate-500 mb-1">地址</p>
              <code class="text-xs bg-slate-100 px-2 py-1 rounded break-all">{{ reg.url }}</code>
            </div>
            
            <div class="mb-3">
              <p class="text-xs text-slate-500 mb-1">认证</p>
              <span v-if="reg.username" class="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-green-50 text-green-700">
                <i class="fas fa-user mr-1"></i>{{ reg.username }}
              </span>
              <span v-else class="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-slate-100 text-slate-500">
                <i class="fas fa-lock-open mr-1"></i>匿名
              </span>
            </div>
            
            <div class="flex flex-wrap gap-1 pt-2 border-t border-slate-100">
              <button @click="openPullModal(reg)" class="btn-icon text-slate-600 hover:bg-slate-100" title="拉取镜像">
                <i class="fas fa-download text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">拉取</span>
              </button>
              <button @click="openPushModal(reg)" class="btn-icon text-blue-600 hover:bg-blue-50" title="推送镜像">
                <i class="fas fa-upload text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">推送</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 推送镜像模态框 -->
    <BaseModal
      v-model="pushOpen"
      title="推送镜像到仓库"
      :loading="pushLoading"
      show-footer
      @confirm="handlePush"
    >
      <template #confirm-text>开始推送</template>
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
            <option v-for="reg in registries" :key="reg.url" :value="reg.url">{{ reg.name }} ({{ reg.url }}){{ reg.description ? ' - ' + reg.description : '' }}</option>
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
      @confirm="handlePull"
    >
      <template #confirm-text>开始拉取</template>
      <form @submit.prevent="handlePull" class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">源仓库地址</label>
          <select v-model="pullForm.registryUrl" class="input" required>
            <option value="" disabled>请选择仓库</option>
            <option v-for="reg in registries" :key="reg.url" :value="reg.url">{{ reg.name }} ({{ reg.url }}){{ reg.description ? ' - ' + reg.description : '' }}</option>
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">命名空间 <span class="text-slate-400 font-normal">(可选)</span></label>
          <input type="text" v-model="pullForm.namespace" placeholder="例如: myteam" class="input" />
        </div>
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">镜像名称</label>
          <input type="text" v-model="pullForm.image" placeholder="输入镜像名称，如 myapp:latest" class="input" required />
          <p class="mt-1 text-xs text-slate-400">
            将拉取: {{ pullForm.registryUrl || 'registry' }}/{{ pullForm.namespace ? pullForm.namespace + '/' : '' }}{{ pullForm.image || 'image:tag' }}
          </p>
        </div>
      </form>
    </BaseModal>
  </div>
</template>
