<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { SSHCredentialInfo } from '@/service/types'

import PageSearch from '@/component/page-search.vue'

import CredentialEditModal from './widget/credential-edit-modal.vue'

@Component({
    components: { PageSearch, CredentialEditModal }
})
class SSHCredentials extends Vue {
    portal = usePortal()
    @Ref readonly editModalRef!: InstanceType<typeof CredentialEditModal>

    // ─── 数据属性 ───
    credentials: SSHCredentialInfo[] = []
    loading = false
    searchText = ''

    // ─── 计算属性 ───
    get filteredCredentials() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.credentials
        return this.credentials.filter((c: SSHCredentialInfo) =>
            (c.name || '').toLowerCase().includes(keyword) ||
            (c.user || '').toLowerCase().includes(keyword) ||
            (c.description || '').toLowerCase().includes(keyword)
        )
    }

    // ─── 方法 ───
    async loadCredentials() {
        this.loading = true
        try {
            const res = await api.sshCredentialList()
            this.credentials = res.payload || []
        } catch {
            this.portal.showNotification('error', '加载凭据列表失败')
        } finally {
            this.loading = false
        }
    }

    openAdd() {
        this.editModalRef?.show(null)
    }

    openEdit(cred: SSHCredentialInfo) {
        this.editModalRef?.show(cred)
    }

    handleDelete(cred: SSHCredentialInfo) {
        this.portal.showConfirm({
            title: '删除 SSH 凭据',
            message: `确定要删除凭据 <strong class="text-slate-900">${cred.name}</strong> (${cred.user}) 吗？`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                try {
                    await api.sshCredentialDelete(cred.id)
                    this.portal.showNotification('success', '凭据删除成功')
                    this.loadCredentials()
                } catch {}
            }
        })
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadCredentials()
    }
}

export default toNative(SSHCredentials)
</script>

<template>
  <!-- Toolbar -->
  <div class="card">
    <div class="card-toolbar">
      <!-- 桌面端 -->
      <div class="hidden md:flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-purple-500">
            <i class="fas fa-id-card text-white"></i>
          </div>
          <div>
            <h1 class="text-lg font-semibold text-slate-800 truncate">认证凭据</h1>
            <p class="text-xs text-slate-500">管理 SSH 认证凭据，可被多台主机复用</p>
          </div>
        </div>
        <div class="flex items-center gap-2 flex-shrink-0">
          <PageSearch v-model="searchText" search-key="ssh-credentials" placeholder="搜索凭据名或用户名..." focus-color="purple" type-to-search />
          <button class="btn btn-secondary" @click="loadCredentials()">
            <i class="fas fa-rotate"></i>刷新
          </button>
          <button v-if="portal.hasPerm('POST /api/ssh/credential')" class="btn btn-purple" @click="openAdd">
            <i class="fas fa-plus"></i>添加凭据
          </button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="flex md:hidden items-center justify-between">
        <div class="flex items-center gap-3 min-w-0 flex-1">
          <div class="page-icon bg-purple-500">
            <i class="fas fa-id-card text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">认证凭据</h1>
            <p class="text-xs text-slate-500 truncate">可被多台主机复用</p>
          </div>
        </div>
        <div class="flex items-center gap-1 flex-shrink-0">
          <button class="btn btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadCredentials()">
            <i class="fas fa-rotate text-sm"></i>
          </button>
          <button v-if="portal.hasPerm('POST /api/ssh/credential')" class="btn btn-purple w-9 h-9 !px-0" title="添加凭据" @click="openAdd">
            <i class="fas fa-plus text-sm"></i>
          </button>
        </div>
      </div>
    </div>

    <!-- 移动端搜索 -->
    <div class="mobile-search">
      <PageSearch v-model="searchText" search-key="ssh-credentials" placeholder="搜索凭据名或用户名..." width-class="w-full" focus-color="purple" />
    </div>

    <!-- Loading -->
    <div v-if="loading" class="card-body">
      <div class="empty-state">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
    </div>

    <div v-else-if="filteredCredentials.length === 0" class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-id-card text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">{{ credentials.length === 0 ? '暂无认证凭据' : '未找到匹配凭据' }}</p>
        <p class="text-sm text-slate-400">{{ credentials.length === 0 ? '点击右上角「添加凭据」创建可复用的认证凭据' : '尝试更换关键词或清空搜索条件' }}</p>
      </div>
    </div>

    <template v-else>
      <!-- 桌面端表格 -->
      <div class="card-table hidden md:block">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-50 border-b border-slate-200">
              <th class="th">凭据名称</th>
              <th class="th">用户名</th>
              <th class="w-28 th">认证方式</th>
              <th class="w-32 th">描述</th>
              <th class="w-32 th-right">操作</th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-slate-100">
            <tr v-for="cred in filteredCredentials" :key="cred.id" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3">
                <div class="flex items-center gap-2 min-w-0">
                  <div class="row-icon bg-purple-400">
                    <i class="fas fa-id-card text-white text-sm"></i>
                  </div>
                  <span class="font-medium text-slate-800 truncate">{{ cred.name }}</span>
                </div>
              </td>
              <td class="px-4 py-3 text-sm text-slate-600"><code class="text-xs bg-slate-100 px-2 py-0.5 rounded-lg text-slate-600">{{ cred.user }}</code></td>
              <td class="px-4 py-3 text-sm text-slate-600">
                <span v-if="cred.authType === 'privateKey'" class="inline-flex items-center gap-1 text-xs">
                  <i class="fas fa-key text-amber-400"></i>私钥
                </span>
                <span v-else-if="cred.authType === 'password'" class="inline-flex items-center gap-1 text-xs">
                  <i class="fas fa-lock text-slate-400"></i>密码
                </span>
                <span v-else class="text-xs text-slate-400">未设置</span>
              </td>
              <td class="px-4 py-3 text-sm text-slate-600 truncate max-w-[200px]">{{ cred.description || '-' }}</td>
              <td class="px-4 py-3">
                <div class="flex justify-end items-center gap-1">
                  <button v-if="portal.hasPerm('PUT /api/ssh/credential/:id')" class="btn-icon btn-icon-blue" title="编辑" @click="openEdit(cred)">
                    <i class="fas fa-pen text-xs"></i>
                  </button>
                  <button v-if="portal.hasPerm('DELETE /api/ssh/credential/:id')" class="btn-icon btn-icon-red" title="删除" @click="handleDelete(cred)">
                    <i class="fas fa-trash text-xs"></i>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 移动端卡片 -->
      <div class="card-body md:hidden space-y-3">
        <div v-for="cred in filteredCredentials" :key="cred.id" class="card-interactive">
          <div class="card-info-row">
            <div class="list-icon bg-purple-400 flex-shrink-0">
              <i class="fas fa-id-card text-white text-base"></i>
            </div>
            <div class="min-w-0">
              <span class="font-medium text-slate-800 text-sm truncate block">{{ cred.name }}</span>
              <span v-if="cred.description" class="text-xs text-slate-400 truncate block mt-0.5">{{ cred.description }}</span>
            </div>
          </div>

          <div class="flex items-start gap-2 mb-3">
            <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">用户</span>
            <code class="text-xs bg-slate-100 text-slate-600 px-1.5 py-0.5 rounded-lg break-all">{{ cred.user }}</code>
          </div>
          <div class="flex items-center gap-2 mb-3">
            <span class="text-xs text-slate-400 flex-shrink-0">认证</span>
            <span class="text-xs text-slate-500">
              <span v-if="cred.authType === 'privateKey'"><i class="fas fa-key text-amber-400 mr-1"></i>私钥</span>
              <span v-else-if="cred.authType === 'password'"><i class="fas fa-lock text-slate-400 mr-1"></i>密码</span>
              <span v-else class="text-slate-400">未设置</span>
            </span>
          </div>

          <div class="card-actions">
            <button v-if="portal.hasPerm('PUT /api/ssh/credential/:id')" class="btn-icon btn-icon-blue" title="编辑" @click="openEdit(cred)">
              <i class="fas fa-pen text-xs"></i><span class="text-xs ml-1">编辑</span>
            </button>
            <button v-if="portal.hasPerm('DELETE /api/ssh/credential/:id')" class="btn-icon btn-icon-red" title="删除" @click="handleDelete(cred)">
              <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
            </button>
          </div>
        </div>
      </div>
    </template>
  </div>

  <CredentialEditModal ref="editModalRef" @success="loadCredentials" />
</template>
