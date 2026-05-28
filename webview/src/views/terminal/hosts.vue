<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { SSHHostInfo } from '@/service/types'

import PageSearch from '@/component/page-search.vue'

import HostEditModal from './widget/host-edit-modal.vue'

@Component({
    components: { PageSearch, HostEditModal }
})
class SSHHosts extends Vue {
    portal = usePortal()
    @Ref readonly editModalRef!: InstanceType<typeof HostEditModal>

    // ─── 数据属性 ───
    hosts: SSHHostInfo[] = []
    loading = false
    searchText = ''

    // ─── 计算属性 ───
    get filteredHosts() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.hosts
        return this.hosts.filter((h: SSHHostInfo) =>
            (h.name || '').toLowerCase().includes(keyword) ||
            (h.addr || '').toLowerCase().includes(keyword) ||
            (h.user || '').toLowerCase().includes(keyword) ||
            (h.description || '').toLowerCase().includes(keyword)
        )
    }

    // ─── 方法 ───
    async loadHosts() {
        this.loading = true
        try {
            const res = await api.sshHostList()
            this.hosts = res.payload || []
        } catch {
            this.portal.showNotification('error', '加载主机列表失败')
        } finally {
            this.loading = false
        }
    }

    openAdd() {
        this.editModalRef?.show(null)
    }

    openEdit(host: SSHHostInfo) {
        this.editModalRef?.show(host)
    }

    openTerminal(host: SSHHostInfo) {
        this.$router.push(`/ssh/${host.id}`)
    }

    handleDelete(host: SSHHostInfo) {
        this.portal.showConfirm({
            title: '删除 SSH 主机',
            message: `确定要删除主机 <strong class="text-slate-900">${host.name}</strong> (${host.addr}) 吗？`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                try {
                    await api.sshHostDelete(host.id)
                    this.portal.showNotification('success', '主机删除成功')
                    this.loadHosts()
                } catch {}
            }
        })
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadHosts()
    }
}

export default toNative(SSHHosts)
</script>

<template>
  <div>
    <!-- Toolbar -->
    <div class="card mb-4">
      <div class="card-toolbar">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="page-icon bg-teal-500">
              <i class="fas fa-server text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">SSH 连接</h1>
              <p class="text-xs text-slate-500">通过 SSH 管理主机，通过浏览器直接连接远程终端</p>
            </div>
          </div>
          <div class="flex items-center gap-2 flex-shrink-0">
            <PageSearch v-model="searchText" search-key="ssh-hosts" placeholder="搜索主机名、地址或用户名..." width-class="w-64" focus-color="teal" type-to-search />
            <button class="btn btn-secondary" @click="loadHosts()">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button v-if="portal.hasPerm('POST /api/ssh/host')" class="btn btn-emerald" @click="openAdd">
              <i class="fas fa-plus"></i>添加主机
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="page-icon bg-teal-500">
              <i class="fas fa-server text-white"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">SSH 连接</h1>
              <p class="text-xs text-slate-500 truncate">通过 SSH 管理主机</p>
            </div>
          </div>
          <div class="flex items-center gap-2 flex-shrink-0">
            <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadHosts()">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button v-if="portal.hasPerm('POST /api/ssh/host')" class="btn btn-emerald w-9 h-9 !px-0" title="添加主机" @click="openAdd">
              <i class="fas fa-plus text-sm"></i>
            </button>
          </div>
        </div>
      </div>

      <!-- 移动端搜索 -->
      <div class="mobile-search">
        <PageSearch v-model="searchText" search-key="ssh-hosts" placeholder="搜索主机名、地址或用户名..." width-class="w-full" focus-color="teal" />
      </div>

      <!-- Loading -->
      <div v-if="loading" class="empty-state">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <div v-else>
        <!-- 桌面端表格 -->
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="th">主机</th>
                <th class="th">地址</th>
                <th class="th">用户名</th>
                <th class="w-28 th">认证方式</th>
                <th class="w-32 th-right">操作</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="host in filteredHosts" :key="host.id" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3 max-w-[280px]">
                  <div class="flex items-center gap-2 min-w-0">
                    <div class="row-icon bg-teal-400">
                      <i class="fas fa-server text-white text-sm"></i>
                    </div>
                    <div class="min-w-0">
                      <span class="font-medium text-slate-800 truncate block">{{ host.name }}</span>
                      <span v-if="host.description" class="text-xs text-slate-400 truncate block mt-0.5">{{ host.description }}</span>
                    </div>
                  </div>
                </td>
                <td class="px-4 py-3">
                  <code class="text-xs bg-slate-100 px-2 py-0.5 rounded-lg text-slate-600">{{ host.addr }}</code>
                </td>
                <td class="px-4 py-3 text-sm text-slate-600">{{ host.user }}</td>
                <td class="px-4 py-3 text-sm text-slate-600">
                  <span v-if="host.privateKey" class="inline-flex items-center gap-1 text-xs">
                    <i class="fas fa-key text-amber-400"></i>私钥
                  </span>
                  <span v-else-if="host.passwordSet" class="inline-flex items-center gap-1 text-xs">
                    <i class="fas fa-lock text-slate-400"></i>密码
                  </span>
                  <span v-else class="text-xs text-slate-400">未设置</span>
                </td>
                <td class="px-4 py-3">
                  <div class="flex justify-end items-center gap-1">
                    <button v-if="portal.hasPerm('GET /api/ssh/:id')" class="btn-icon btn-icon-teal" title="连接终端" @click="openTerminal(host)">
                      <i class="fas fa-terminal text-xs"></i>
                    </button>
                    <button v-if="portal.hasPerm('PUT /api/ssh/host/:id')" class="btn-icon btn-icon-blue" title="编辑" @click="openEdit(host)">
                      <i class="fas fa-pen text-xs"></i>
                    </button>
                    <button v-if="portal.hasPerm('DELETE /api/ssh/host/:id')" class="btn-icon btn-icon-red" title="删除" @click="handleDelete(host)">
                      <i class="fas fa-trash text-xs"></i>
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 移动端卡片 -->
        <div class="md:hidden space-y-3 p-4">
          <div v-for="host in filteredHosts" :key="host.id" class="card-interactive">
            <div class="card-info-row">
              <div class="list-icon bg-teal-400 flex-shrink-0">
                <i class="fas fa-server text-white text-base"></i>
              </div>
              <div class="min-w-0">
                <span class="font-medium text-slate-800 text-sm truncate block">{{ host.name }}</span>
                <span v-if="host.description" class="text-xs text-slate-400 truncate block mt-0.5">{{ host.description }}</span>
              </div>
            </div>

            <div class="flex items-start gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">地址</span>
              <code class="text-xs bg-slate-100 text-slate-600 px-1.5 py-0.5 rounded-lg break-all">{{ host.addr }}</code>
            </div>
            <div class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">用户</span>
              <span class="text-xs text-slate-500">{{ host.user }}</span>
            </div>
            <div class="flex items-center gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0">认证</span>
              <span class="text-xs text-slate-500">
                <span v-if="host.privateKey"><i class="fas fa-key text-amber-400 mr-1"></i>私钥</span>
                <span v-else-if="host.passwordSet"><i class="fas fa-lock text-slate-400 mr-1"></i>密码</span>
                <span v-else class="text-slate-400">未设置</span>
              </span>
            </div>

            <div class="card-actions">
              <button v-if="portal.hasPerm('GET /api/ssh/:id')" class="btn-icon btn-icon-teal" title="连接终端" @click="openTerminal(host)">
                <i class="fas fa-terminal text-xs"></i><span class="text-xs ml-1">连接</span>
              </button>
              <button v-if="portal.hasPerm('PUT /api/ssh/host/:id')" class="btn-icon btn-icon-blue" title="编辑" @click="openEdit(host)">
                <i class="fas fa-pen text-xs"></i><span class="text-xs ml-1">编辑</span>
              </button>
              <button v-if="portal.hasPerm('DELETE /api/ssh/host/:id')" class="btn-icon btn-icon-red" title="删除" @click="handleDelete(host)">
                <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
              </button>
            </div>
          </div>

          <div v-if="filteredHosts.length === 0" class="rounded-xl border border-slate-200 bg-white py-10 px-4 text-center">
            <p class="text-sm text-slate-500">{{ hosts.length === 0 ? '暂无 SSH 主机' : '未找到匹配主机' }}</p>
          </div>
        </div>

        <!-- 桌面端空状态 -->
        <div v-if="filteredHosts.length === 0" class="hidden md:flex flex-col items-center justify-center py-20">
          <div class="empty-state-icon">
            <i class="fas fa-server text-4xl text-slate-300"></i>
          </div>
          <p class="text-slate-600 font-medium mb-1">{{ hosts.length === 0 ? '暂无 SSH 主机' : '未找到匹配主机' }}</p>
          <p class="text-sm text-slate-400">{{ hosts.length === 0 ? '点击「添加主机」配置 SSH 连接信息' : '尝试更换关键词或清空搜索条件' }}</p>
        </div>
      </div>
    </div>

    <HostEditModal ref="editModalRef" @success="loadHosts" />
  </div>
</template>
