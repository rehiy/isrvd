<script lang="ts">
import { Component, Inject, Ref, Vue, toNative } from 'vue-facing-decorator'

import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import api from '@/service/api'
import type { DockerRegistryInfo } from '@/service/types'

import RegistryEditModal from '@/views/docker/widget/registry-edit-modal.vue'

@Component({
    components: { RegistryEditModal }
})
class Registries extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    @Ref readonly editModalRef!: InstanceType<typeof RegistryEditModal>

    // ─── 数据属性 ───
    daemonMirrors: string[] = []
    indexServerAddress = ''
    registries: DockerRegistryInfo[] = []
    loading = false

    // ─── 方法 ───
    async loadDaemonInfo() {
        try {
            const res = await api.dockerInfo()
            const info = res.payload
            this.daemonMirrors = info?.registryMirrors || []
            this.indexServerAddress = info?.indexServerAddress || ''
        } catch (e) {}
    }

    async loadRegistries() {
        this.loading = true
        try {
            const res = await api.listRegistries()
            this.registries = res.payload || []
        } catch (e) {
            this.actions.showNotification('error', '加载仓库列表失败')
        }
        this.loading = false
    }

    openAdd() {
        this.editModalRef?.show(null)
    }

    openEdit(reg: DockerRegistryInfo) {
        this.editModalRef?.show(reg)
    }

    handleDelete(reg: DockerRegistryInfo) {
        this.actions.showConfirm({
            title: '删除镜像仓库',
            message: `确定要删除仓库 <strong class="text-slate-900">${reg.name}</strong> (${reg.url}) 吗？`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                try {
                    await api.deleteRegistry(reg.url)
                    this.actions.showNotification('success', '仓库删除成功')
                    this.loadRegistries()
                } catch (e) {}
            }
        })
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadDaemonInfo()
        this.loadRegistries()
    }
}

export default toNative(Registries)
</script>

<template>
  <div>
    <div class="card mb-4">
      <!-- Toolbar Bar -->
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-purple-500 flex items-center justify-center">
              <i class="fas fa-warehouse text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">镜像仓库</h1>
              <p class="text-xs text-slate-500">管理镜像仓库账号与加速器</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <button @click="loadRegistries()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button v-if="actions.hasPerm('docker', true)" @click="openAdd" class="px-3 py-1.5 rounded-lg bg-purple-500 hover:bg-purple-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-plus"></i>添加
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="w-9 h-9 rounded-lg bg-purple-500 flex items-center justify-center flex-shrink-0">
              <i class="fas fa-warehouse text-white"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">镜像仓库</h1>
              <p class="text-xs text-slate-500 truncate">管理镜像仓库账号与加速器</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <button @click="loadRegistries()" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 flex items-center justify-center text-slate-600 transition-colors" title="刷新">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button v-if="actions.hasPerm('docker', true)" @click="openAdd" class="w-9 h-9 rounded-lg bg-purple-500 hover:bg-purple-600 flex items-center justify-center text-white transition-colors" title="添加">
              <i class="fas fa-plus text-sm"></i>
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
                <th class="w-28 px-4 py-3 text-right text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
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
                  <div class="flex flex-col gap-1">
                    <code class="text-xs bg-slate-100 px-2 py-1 rounded self-start">{{ indexServerAddress || 'https://index.docker.io/v1/' }}</code>
                    <template v-if="daemonMirrors.length > 0">
                      <code v-for="mirror in daemonMirrors" :key="mirror" class="text-xs bg-sky-50 text-sky-700 px-2 py-1 rounded self-start inline-flex items-center gap-1">
                        <i class="fas fa-bolt text-sky-400 text-xs"></i>{{ mirror }}
                      </code>
                    </template>
                  </div>
                </td>
                <td class="px-4 py-3">
                  <span class="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-slate-100 text-slate-500">
                    <i class="fas fa-lock-open mr-1"></i>匿名
                  </span>
                </td>
                <td class="px-4 py-3 text-right text-xs text-slate-400">—</td>
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
                    <button v-if="actions.hasPerm('docker', true)" @click="openEdit(reg)" class="btn-icon text-blue-600 hover:bg-blue-50" title="编辑">
                      <i class="fas fa-pen text-xs"></i>
                    </button>
                    <button v-if="actions.hasPerm('docker', true)" @click="handleDelete(reg)" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
                      <i class="fas fa-trash text-xs"></i>
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 移动端卡片视图 -->
        <div class="md:hidden space-y-3 p-4">
          <!-- Docker Hub 卡片 -->
          <div class="rounded-xl border border-slate-200 bg-white p-4 transition-all hover:shadow-sm">
            <div class="flex items-center gap-3 mb-3">
              <div class="w-10 h-10 rounded-lg bg-blue-500 flex items-center justify-center flex-shrink-0">
                <i class="fab fa-docker text-white text-base"></i>
              </div>
              <div class="min-w-0">
                <h3 class="font-medium text-slate-800 text-sm">Docker Hub</h3>
                <span class="text-xs text-slate-400">默认公共仓库</span>
              </div>
            </div>

            <div class="space-y-2">
              <div class="flex items-start gap-2">
                <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">地址</span>
                <code class="text-xs bg-slate-100 text-slate-600 px-1.5 py-0.5 rounded break-all">{{ indexServerAddress || 'https://index.docker.io/v1/' }}</code>
              </div>
              <template v-if="daemonMirrors.length > 0">
                <div v-for="mirror in daemonMirrors" :key="mirror" class="flex items-center gap-2">
                  <span class="text-xs text-slate-400 flex-shrink-0">加速</span>
                  <code class="text-xs bg-sky-50 text-sky-700 px-1.5 py-0.5 rounded truncate flex items-center gap-1">
                    <i class="fas fa-bolt text-sky-400 text-xs"></i>{{ mirror }}
                  </code>
                </div>
              </template>
              <div class="flex items-center gap-2">
                <span class="text-xs text-slate-400 flex-shrink-0">认证</span>
                <span class="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-slate-100 text-slate-500">
                  <i class="fas fa-lock-open mr-1"></i>匿名
                </span>
              </div>
            </div>
          </div>

          <!-- 私有仓库卡片 -->
          <div v-for="reg in registries" :key="reg.url" class="rounded-xl border border-slate-200 bg-white p-4 transition-all hover:shadow-sm">
            <div class="flex items-center gap-3 min-w-0 mb-3">
              <div class="w-10 h-10 rounded-lg bg-purple-400 flex items-center justify-center flex-shrink-0">
                <i class="fas fa-warehouse text-white text-base"></i>
              </div>
              <div class="min-w-0">
                <h3 class="font-medium text-slate-800 text-sm truncate">{{ reg.name }}</h3>
                <span v-if="reg.description" class="text-xs text-slate-400 truncate block">{{ reg.description }}</span>
              </div>
            </div>

            <div class="space-y-2 mb-3">
              <div class="flex items-center gap-2">
                <span class="text-xs text-slate-400 flex-shrink-0">地址</span>
                <code class="text-xs bg-slate-100 text-slate-600 px-1.5 py-0.5 rounded truncate">{{ reg.url }}</code>
              </div>
              <div class="flex items-center gap-2">
                <span class="text-xs text-slate-400 flex-shrink-0">认证</span>
                <span v-if="reg.username" class="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-green-50 text-green-700">
                  <i class="fas fa-user mr-1"></i>{{ reg.username }}
                </span>
                <span v-else class="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-slate-100 text-slate-500">
                  <i class="fas fa-lock-open mr-1"></i>匿名
                </span>
              </div>
            </div>

            <!-- 底部：操作按钮 -->
            <div class="flex flex-wrap gap-1.5 pt-2 border-t border-slate-100">
              <button v-if="actions.hasPerm('docker', true)" @click="openEdit(reg)" class="btn-icon text-blue-600 hover:bg-blue-50" title="编辑">
                <i class="fas fa-pen text-xs"></i><span class="text-xs ml-1">编辑</span>
              </button>
              <button v-if="actions.hasPerm('docker', true)" @click="handleDelete(reg)" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
                <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <RegistryEditModal ref="editModalRef" @success="loadRegistries" />
  </div>
</template>
