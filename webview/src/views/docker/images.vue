<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'

import { formatFileSize, formatTime } from '@/helper/utils.js'
import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'
import { inject } from 'vue'

import BaseModal from '@/component/modal.vue'

const actions = inject(APP_ACTIONS_KEY)
const router = useRouter()

// 自己管理数据
const images = ref([])
const loading = ref(false)
const showAllImages = ref(false)

// 模态框状态
const modalOpen = ref(false)
const modalTitle = ref('')
const modalLoading = ref(false)
const formData = ref({})

// 搜索状态
const searchResults = ref([])
const searchLoading = ref(false)
const searchKeyword = ref('')

// 镜像加速器（来自 Docker daemon）
const daemonMirrors = ref([])
const indexServerAddress = ref('')

const loadDaemonInfo = async () => {
  try {
    const res = await api.dockerInfo()
    const info = res.payload || {}
    daemonMirrors.value = info.registryMirrors || []
    indexServerAddress.value = info.indexServerAddress || ''
  } catch (e) {}
}

// 打标签状态
const tagOpen = ref(false)
const tagImage = ref(null)
const tagRepoTag = ref('')
const tagLoading = ref(false)

// 构建状态
const buildOpen = ref(false)
const buildTag = ref('')
const buildDockerfile = ref('FROM alpine:latest\nCMD ["echo", "Hello World"]')
const buildLoading = ref(false)

// 加载镜像列表
const loadImages = async () => {
  loading.value = true
  try {
const res = await api.listImages(showAllImages.value)
    images.value = res.payload || []
  } catch (e) {
    actions.showNotification('error', '加载镜像列表失败')
  }
  loading.value = false
}

// 拉取镜像弹窗
const pullImageModal = () => {
  formData.value = { image: '', tag: '' }
  searchResults.value = []
  searchKeyword.value = ''
  loadDaemonInfo()
  modalTitle.value = '拉取镜像'
  modalOpen.value = true
}

const handlePullImage = async () => {
  if (!formData.value.image.trim()) return
  modalLoading.value = true
  try {
    await api.pullImage(formData.value.image, formData.value.tag)
    actions.showNotification('success', '镜像拉取成功')
    modalOpen.value = false
    loadImages()
  } catch (e) {}
  modalLoading.value = false
}

// 搜索镜像
const handleSearchImage = async () => {
  if (!searchKeyword.value.trim()) return
  searchLoading.value = true
  try {
    const res = await api.imageSearch(searchKeyword.value.trim())
    searchResults.value = res.payload || []
  } catch (e) {
    actions.showNotification('error', '搜索镜像失败')
  }
  searchLoading.value = false
}

// 选择搜索结果
const selectSearchResult = (item) => {
  formData.value.image = item.name
  formData.value.tag = 'latest'
  searchResults.value = []
  searchKeyword.value = ''
}

// 删除镜像
const handleImageAction = (image, action) => {
  actions.showConfirm({
    title: '删除镜像',
    message: `确定要删除镜像 <strong class="text-slate-900">${image.repoTags[0] || image.id}</strong> 吗？`,
    icon: 'fa-trash',
    iconColor: 'red',
    confirmText: '确认删除',
    danger: true,
    onConfirm: async () => {
      await api.imageAction(image.id, action)
      actions.showNotification('success', '镜像删除成功')
      loadImages()
    }
  })
}

// 打标签
const openTagModal = (image) => {
  tagImage.value = image
  tagRepoTag.value = ''
  tagOpen.value = true
}

const handleTag = async () => {
  if (!tagRepoTag.value.trim() || !tagImage.value) return
  tagLoading.value = true
  try {
    await api.imageTag(tagImage.value.id, tagRepoTag.value.trim())
    actions.showNotification('success', '镜像标签添加成功')
    tagOpen.value = false
    loadImages()
  } catch (e) {}
  tagLoading.value = false
}

// 构建镜像
const openBuildModal = () => {
  buildTag.value = ''
  buildDockerfile.value = 'FROM alpine:latest\nCMD ["echo", "Hello World"]'
  buildOpen.value = true
}

const handleBuild = async () => {
  if (!buildDockerfile.value.trim()) return
  buildLoading.value = true
  try {
    await api.imageBuild(buildDockerfile.value, buildTag.value)
    actions.showNotification('success', '镜像构建成功')
    buildOpen.value = false
    loadImages()
  } catch (e) {}
  buildLoading.value = false
}

onMounted(() => {
  loadImages()
})
</script>

<template>
  <div>
    <div class="card mb-4">
      <!-- Toolbar Bar -->
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-6 py-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-blue-500 flex items-center justify-center">
              <i class="fas fa-compact-disc text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">镜像管理</h1>
              <p class="text-xs text-slate-500">管理 Docker 镜像</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <div class="flex items-center gap-1 bg-slate-100 p-1 rounded-lg">
              <button 
                @click="showAllImages = false; loadImages()" 
                :class="[
                  'px-3 py-1 text-xs font-medium rounded-md transition-all duration-200',
                  !showAllImages 
                    ? 'bg-white text-blue-600 shadow-sm' 
                    : 'text-slate-500 hover:text-slate-700'
                ]"
              >
                <i class="fas fa-cube mr-1"></i><span class="hidden sm:inline">顶层</span>
              </button>
              <button 
                @click="showAllImages = true; loadImages()" 
                :class="[
                  'px-3 py-1 text-xs font-medium rounded-md transition-all duration-200',
                  showAllImages 
                    ? 'bg-white text-blue-600 shadow-sm' 
                    : 'text-slate-500 hover:text-slate-700'
                ]"
              >
                <i class="fas fa-layer-group mr-1"></i><span class="hidden sm:inline">全部</span>
              </button>
            </div>
            <button @click="loadImages()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button @click="openBuildModal()" class="px-3 py-1.5 rounded-lg bg-blue-500 hover:bg-blue-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-hammer"></i>构建
            </button>
            <button @click="pullImageModal()" class="px-3 py-1.5 rounded-lg bg-blue-500 hover:bg-blue-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-download"></i>拉取
            </button>
          </div>
        </div>
      </div>

    <!-- Loading -->
    <div v-if="loading" class="flex flex-col items-center justify-center py-20">
      <div class="w-12 h-12 spinner mb-3"></div>
      <p class="text-slate-500">加载中...</p>
    </div>

    <!-- Image List -->
    <div v-else-if="images.length > 0" class="space-y-3">
      <!-- 桌面端表格视图 -->
      <div class="hidden md:block overflow-x-auto">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-50 border-b border-slate-200">
              <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">镜像</th>
              <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">ID</th>
              <th class="w-32 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">大小</th>
              <th class="w-36 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">创建时间</th>
              <th class="w-40 px-4 py-3 text-right text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-slate-100">
            <tr v-for="img in images" :key="img.id" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3">
                <div class="flex flex-col gap-0.5">
                  <div class="flex items-center gap-2">
                    <div class="w-8 h-8 rounded-lg bg-blue-400 flex items-center justify-center flex-shrink-0">
                      <i class="fas fa-compact-disc text-white text-sm"></i>
                    </div>
                    <span class="font-medium text-slate-800">{{ img.repoTags[0] || '<none>' }}</span>
                  </div>
                  <div v-if="img.repoTags.length > 1" class="ml-10 flex flex-wrap gap-1">
                    <span v-for="(tag, idx) in img.repoTags.slice(1)" :key="idx" class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-blue-50 text-blue-600">
                      {{ tag }}
                    </span>
                  </div>
                </div>
              </td>
              <td class="px-4 py-3"><code class="text-xs text-slate-500 font-mono">{{ img.shortId }}</code></td>
              <td class="px-4 py-3 text-sm text-slate-600">{{ formatFileSize(img.size) }}</td>
              <td class="px-4 py-3 whitespace-nowrap text-sm text-slate-600">{{ formatTime(new Date(img.created * 1000).toISOString()) }}</td>
              <td class="px-4 py-3">
                <div class="flex justify-end items-center gap-0.5">
                  <button @click="router.push('/docker/image/' + img.id)" class="btn-icon text-slate-600 hover:bg-slate-100" title="查看详情">
                    <i class="fas fa-circle-info text-xs"></i>
                  </button>
                  <button @click="openTagModal(img)" class="btn-icon text-blue-600 hover:bg-blue-50" title="打标签">
                    <i class="fas fa-tag text-xs"></i>
                  </button>
                  <button @click="handleImageAction(img, 'remove')" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
                    <i class="fas fa-trash text-xs"></i>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 移动端卡片视图 -->
      <div class="md:hidden space-y-3">
        <div 
          v-for="img in images" 
          :key="img.id"
          class="rounded-xl border border-slate-200 bg-white p-4 transition-all hover:shadow-sm"
        >
          <!-- 顶部：镜像信息和图标 -->
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-lg bg-blue-400 flex items-center justify-center flex-shrink-0">
                <i class="fas fa-compact-disc text-white text-base"></i>
              </div>
              <div class="min-w-0">
                <div class="flex items-center gap-2">
                  <span class="font-medium text-slate-800 text-sm">{{ img.repoTags[0] || '<none>' }}</span>
                </div>
                <div class="flex items-center gap-3 mt-1">
                  <span class="text-xs text-slate-500">{{ formatFileSize(img.size) }}</span>
                  <span class="text-xs text-slate-500">{{ formatTime(new Date(img.created * 1000).toISOString()) }}</span>
                </div>
              </div>
            </div>
          </div>
          
          <!-- 中间：镜像ID和标签 -->
          <div class="mb-3">
            <p class="text-xs text-slate-500 mb-1">镜像ID</p>
            <code class="text-xs text-slate-500 font-mono break-all">{{ img.shortId }}</code>
          </div>
          
          <!-- 其他标签 -->
          <div v-if="img.repoTags.length > 1" class="mb-3">
            <p class="text-xs text-slate-500 mb-1">其他标签</p>
            <div class="flex flex-wrap gap-1">
              <span v-for="(tag, idx) in img.repoTags.slice(1)" :key="idx" class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-blue-50 text-blue-600">
                {{ tag }}
              </span>
            </div>
          </div>
          
          <!-- 底部：操作按钮 -->
          <div class="flex flex-wrap gap-1 pt-2 border-t border-slate-100">
            <button @click="router.push('/docker/image/' + img.id)" class="btn-icon text-slate-600 hover:bg-slate-50" title="查看详情">
              <i class="fas fa-circle-info text-xs"></i>
              <span class="text-xs ml-1 hidden xs:inline">详情</span>
            </button>
            <button @click="openTagModal(img)" class="btn-icon text-blue-600 hover:bg-blue-50" title="打标签">
              <i class="fas fa-tag text-xs"></i>
              <span class="text-xs ml-1 hidden xs:inline">标签</span>
            </button>
            <button @click="handleImageAction(img, 'remove')" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
              <i class="fas fa-trash text-xs"></i>
              <span class="text-xs ml-1 hidden xs:inline">删除</span>
            </button>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="flex flex-col items-center justify-center py-20">
      <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
        <i class="fas fa-compact-disc text-4xl text-slate-300"></i>
      </div>
      <p class="text-slate-600 font-medium mb-1">暂无镜像</p>
      <p class="text-sm text-slate-400">点击「拉取镜像」从 Registry 获取镜像</p>
    </div>
    </div>

    <!-- 拉取镜像模态框 -->
    <BaseModal 
      v-model="modalOpen" 
      :title="modalTitle" 
      :loading="modalLoading"
      :show-footer="modalTitle === '拉取镜像'"
      @confirm="handlePullImage"
    >
      <template v-if="modalTitle === '拉取镜像'">
        <form @submit.prevent="handlePullImage" class="space-y-4">
          <!-- 镜像搜索 -->
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">搜索镜像</label>
            <div class="flex gap-2">
              <input type="text" v-model="searchKeyword" placeholder="搜索 Docker Hub 镜像" class="input flex-1" @keyup.enter="handleSearchImage" />
              <button type="button" @click="handleSearchImage" :disabled="searchLoading" class="px-4 py-2 rounded-lg bg-slate-700 hover:bg-slate-800 text-white text-xs font-medium flex items-center gap-1.5 transition-colors disabled:opacity-50">
                <i :class="['fas', searchLoading ? 'fa-spinner fa-spin' : 'fa-search']"></i>
                {{ searchLoading ? '搜索中' : '搜索' }}
              </button>
            </div>
          </div>

          <!-- 搜索结果 -->
          <div v-if="searchResults.length > 0" class="border border-slate-200 rounded-xl max-h-48 overflow-y-auto">
            <div 
              v-for="item in searchResults" 
              :key="item.name" 
              @click="selectSearchResult(item)" 
              class="px-4 py-2.5 hover:bg-blue-50 cursor-pointer border-b border-slate-100 last:border-b-0 transition-colors"
            >
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <i v-if="item.isOfficial" class="fas fa-certificate text-blue-500 text-xs" title="官方镜像"></i>
                  <span class="text-sm font-medium text-slate-800">{{ item.name }}</span>
                </div>
                <div class="flex items-center gap-2">
                  <span v-if="item.isOfficial" class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-700">官方</span>
                  <span class="text-xs text-slate-400"><i class="fas fa-star text-amber-400 mr-0.5"></i>{{ item.starCount }}</span>
                </div>
              </div>
              <p v-if="item.description" class="text-xs text-slate-500 mt-0.5 truncate">{{ item.description }}</p>
            </div>
          </div>

          <div class="border-t border-slate-200 pt-4">
            <div class="mb-4">
              <label class="block text-sm font-medium text-slate-700 mb-2">镜像名称</label>
              <input type="text" v-model="formData.image" placeholder="例如: nginx, redis:alpine, ubuntu:22.04" required class="input" />
            </div>
            <div>
              <label class="block text-sm font-medium text-slate-700 mb-2">Tag（可选）</label>
              <input type="text" v-model="formData.tag" placeholder="默认 latest" class="input" />
            </div>
          </div>

          <!-- 镜像源提示 -->
          <div class="border-t border-slate-200 pt-4">
            <p class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2">当前镜像源</p>
            <div v-if="daemonMirrors.length > 0" class="flex flex-wrap gap-1.5">
              <code
                v-for="mirror in daemonMirrors"
                :key="mirror"
                class="inline-flex items-center gap-1 px-2 py-1 bg-sky-50 border border-sky-200 rounded-lg text-xs font-mono text-sky-700"
              >
                <i class="fas fa-bolt text-sky-400 text-xs"></i>{{ mirror }}
              </code>
            </div>
            <div v-else class="flex items-center gap-1.5 text-xs text-slate-400">
              <i class="fab fa-docker text-blue-400"></i>
              {{ indexServerAddress || 'https://index.docker.io/v1/' }}
              <span class="text-slate-300">（未配置加速器）</span>
            </div>
          </div>
        </form>
      </template>
    </BaseModal>

    <!-- 打标签模态框 -->
    <BaseModal
      v-model="tagOpen"
      title="添加镜像标签"
      :loading="tagLoading"
      show-footer
      @confirm="handleTag"
    >
      <template #confirm-text>确认添加</template>
      <div v-if="tagImage" class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">当前镜像</label>
          <div class="px-3 py-2 bg-slate-50 rounded-lg text-sm text-slate-500">{{ tagImage.repoTags[0] || tagImage.shortId }}</div>
        </div>
        <div v-if="tagImage.repoTags.length > 1">
          <label class="block text-sm font-medium text-slate-700 mb-2">已有标签</label>
          <div class="flex flex-wrap gap-1.5">
            <span v-for="tag in tagImage.repoTags" :key="tag" class="inline-flex items-center px-2 py-1 rounded-lg text-xs font-medium bg-blue-50 text-blue-700">
              {{ tag }}
            </span>
          </div>
        </div>
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">新标签</label>
          <input type="text" v-model="tagRepoTag" placeholder="例如: myimage:v1, registry.example.com/app:latest" class="input" @keyup.enter="handleTag" />
          <p class="mt-1 text-xs text-slate-400">格式: 仓库路径:标签，如 myapp:v1.0</p>
        </div>
      </div>
    </BaseModal>

    <!-- 构建镜像模态框 -->
    <BaseModal
      v-model="buildOpen"
      title="构建镜像"
      :loading="buildLoading"
      show-footer
      @confirm="handleBuild"
      size="lg"
    >
      <template #confirm-text>开始构建</template>
      <div class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">镜像标签</label>
          <input type="text" v-model="buildTag" placeholder="例如: myapp:v1, custom-image:latest" class="input" />
          <p class="mt-1 text-xs text-slate-400">留空则使用 custom:latest</p>
        </div>
        <div>
          <label class="block text-sm font-medium text-slate-700 mb-2">Dockerfile</label>
          <textarea 
            v-model="buildDockerfile" 
            rows="14" 
            class="input font-mono text-sm" 
            placeholder="FROM alpine:latest&#10;RUN echo hello"
            spellcheck="false"
          ></textarea>
        </div>
      </div>
    </BaseModal>
  </div>
</template>
