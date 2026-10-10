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
        this.$router.push(`/ssh/host/${host.id}`)
    }

    handleDelete(host: SSHHostInfo) {
        this.portal.showConfirm({
            title: this.$t('删除 SSH 主机'),
            message: this.$t('确定要删除主机 <strong class="text-slate-900">{0}</strong> ({1}) 吗？', host.name, host.addr),
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: this.$t('确认删除'),
            danger: true,
            onConfirm: async () => {
                try {
                    await api.sshHostDelete(host.id)
                    this.portal.showNotification('success', this.$t('主机删除成功'))
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
  <!-- Toolbar -->
  <div class="page">
    <div class="page-toolbar">
      <!-- 桌面端 -->
      <div class="toolbar-desktop">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-teal-500">
            <i class="fas fa-server text-white"></i>
          </div>
          <div>
            <h1 class="title-text">{{ $t('主机连接') }}</h1>
            <p class="text-xs text-slate-500">{{ $t('通过 SSH 协议管理主机，在浏览器中连接远程服务器') }}</p>
          </div>
        </div>
        <div class="action-group">
          <PageSearch v-model="searchText" search-key="ssh-hosts" :placeholder="$t('搜索主机名、地址或用户名...')" focus-color="teal" type-to-search />
          <button class="btn btn-secondary" @click="loadHosts()">
            <i class="fas fa-rotate"></i>{{ $t('刷新') }}
          </button>
          <button v-if="portal.hasPerm('POST /api/ssh/host')" class="btn btn-emerald" @click="openAdd">
            <i class="fas fa-plus"></i>{{ $t('添加主机') }}
          </button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="toolbar-mobile">
        <div class="title-group">
          <div class="page-icon bg-teal-500">
            <i class="fas fa-server text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="title-text">{{ $t('主机连接') }}</h1>
            <p class="text-xs text-slate-500 truncate">{{ $t('通过 SSH 协议管理主机') }}</p>
          </div>
        </div>
        <div class="action-group-sm">
          <button class="btn btn-secondary btn-square" :title="$t('刷新')" @click="loadHosts()">
            <i class="fas fa-rotate text-sm"></i>
          </button>
          <button v-if="portal.hasPerm('POST /api/ssh/host')" class="btn btn-emerald btn-square" :title="$t('添加主机')" @click="openAdd">
            <i class="fas fa-plus text-sm"></i>
          </button>
        </div>
      </div>
    </div>

    <!-- 移动端搜索 -->
    <div class="mobile-search">
      <PageSearch v-model="searchText" search-key="ssh-hosts" :placeholder="$t('搜索主机名、地址或用户名...')" width-class="w-full" focus-color="teal" />
    </div>

    <!-- Loading -->
    <div v-if="loading" class="card-body">
      <div class="empty-state">
        <div class="spinner-lg"></div>
        <p class="text-slate-500">{{ $t('加载中...') }}</p>
      </div>
    </div>

    <div v-else-if="filteredHosts.length === 0" class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-server text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">{{ hosts.length === 0 ? $t('暂无主机连接') : $t('未找到匹配主机') }}</p>
        <p class="text-sm text-slate-400">{{ hosts.length === 0 ? $t('点击右上角「添加主机」开始配置') : $t('尝试更换关键词或清空搜索条件') }}</p>
      </div>
    </div>

    <template v-else>
      <!-- 桌面端表格 -->
      <div class="card-table hidden md:block">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-100 border-b border-slate-200">
              <th class="th">{{ $t('主机') }}</th>
              <th class="th">{{ $t('地址') }}</th>
              <th class="w-36 th">{{ $t('用户名') }}</th>
              <th class="th">{{ $t('认证方式') }}</th>
              <th class="w-32 th-right">{{ $t('操作') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-for="host in filteredHosts" :key="host.id" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3 max-w-[280px]">
                <div class="inline-info">
                  <div class="row-icon bg-teal-400">
                    <i class="fas fa-server text-white text-sm"></i>
                  </div>
                  <div class="min-w-0">
                    <span class="item-title">{{ host.name }}</span>
                    <span v-if="host.description" class="item-subtitle">{{ host.description }}</span>
                  </div>
                </div>
              </td>
              <td class="px-4 py-3">
                <code class="text-xs bg-slate-100 px-2 py-0.5 rounded-lg text-slate-600">{{ host.addr }}</code>
              </td>
              <td class="td-text">{{ host.user }}</td>
              <td class="td-text">
                <span v-if="host.credentialId" class="inline-flex items-center gap-1 text-xs text-purple-600 font-medium">
                  <i class="fas fa-id-card text-purple-400"></i>{{ host.credentialName || $t('已保存凭据') }}
                </span>
                <span v-else class="inline-flex items-center gap-1 text-xs text-slate-500">
                  <i class="fas fa-key text-slate-400"></i>{{ $t('手动认证') }}
                </span>
              </td>
              <td class="px-4 py-3">
                <div class="table-actions">
                  <button v-if="portal.hasPerm('GET /api/ssh/host/:id')" class="btn-icon btn-icon-teal" :title="$t('打开 SSH/SFTP')" @click="openTerminal(host)">
                    <i class="fas fa-external-link-alt text-xs"></i>
                  </button>
                  <button v-if="portal.hasPerm('PUT /api/ssh/host/:id')" class="btn-icon btn-icon-blue" :title="$t('编辑')" @click="openEdit(host)">
                    <i class="fas fa-pen text-xs"></i>
                  </button>
                  <button v-if="portal.hasPerm('DELETE /api/ssh/host/:id')" class="btn-icon btn-icon-red" :title="$t('删除')" @click="handleDelete(host)">
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
        <div v-for="host in filteredHosts" :key="host.id" class="card-interactive">
          <div class="card-info-row">
            <div class="list-icon bg-teal-400 flex-shrink-0">
              <i class="fas fa-server text-white text-base"></i>
            </div>
            <div class="min-w-0">
              <span class="item-title-sm">{{ host.name }}</span>
              <span v-if="host.description" class="item-subtitle">{{ host.description }}</span>
            </div>
          </div>

          <div class="card-prop-row-start">
            <span class="prop-label-start">{{ $t('地址') }}</span>
            <code class="text-xs bg-slate-100 text-slate-600 px-1.5 py-0.5 rounded-lg break-all">{{ host.addr }}</code>
          </div>
          <div class="card-prop-row">
            <span class="text-xs text-slate-400 flex-shrink-0">{{ $t('用户') }}</span>
            <span class="text-xs text-slate-500">{{ host.user }}</span>
          </div>
          <div class="card-prop-row">
            <span class="text-xs text-slate-400 flex-shrink-0">{{ $t('认证') }}</span>
            <span class="text-xs text-slate-500">
              <span v-if="host.credentialId" class="text-purple-600 font-medium"><i class="fas fa-id-card text-purple-400 mr-1"></i>{{ host.credentialName || $t('已保存凭据') }}</span>
              <span v-else><i class="fas fa-key text-slate-400 mr-1"></i>{{ $t('手动认证') }}</span>
            </span>
          </div>

          <div class="card-actions">
            <button v-if="portal.hasPerm('GET /api/ssh/host/:id')" class="btn-icon btn-icon-teal" :title="$t('连接终端')" @click="openTerminal(host)">
              <i class="fas fa-terminal text-xs"></i><span class="text-xs ml-1">{{ $t('连接') }}</span>
            </button>
            <button v-if="portal.hasPerm('PUT /api/ssh/host/:id')" class="btn-icon btn-icon-blue" :title="$t('编辑')" @click="openEdit(host)">
              <i class="fas fa-pen text-xs"></i><span class="text-xs ml-1">{{ $t('编辑') }}</span>
            </button>
            <button v-if="portal.hasPerm('DELETE /api/ssh/host/:id')" class="btn-icon btn-icon-red" :title="$t('删除')" @click="handleDelete(host)">
              <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">{{ $t('删除') }}</span>
            </button>
          </div>
        </div>
      </div>
    </template>
  </div>

  <HostEditModal ref="editModalRef" @success="loadHosts" />
</template>
