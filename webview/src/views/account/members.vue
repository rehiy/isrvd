<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { MemberInfo } from '@/service/types'

import PageSearch from '@/component/page-search.vue'

import MemberEditModal from './widget/member-edit-modal.vue'

@Component({
    components: { PageSearch, MemberEditModal }
})
class Members extends Vue {
    portal = usePortal()

    // ─── Refs ───
    @Ref readonly memberEditModalRef!: InstanceType<typeof MemberEditModal>

    // ─── 数据属性 ───
    members: MemberInfo[] = []
    loading = false
    searchText = ''

    get filteredMembers() {
        const keyword = this.searchText.trim().toLowerCase()
        if (!keyword) return this.members
        return this.members.filter((m: MemberInfo) =>
            (m.username || '').toLowerCase().includes(keyword) ||
            (m.description || '').toLowerCase().includes(keyword) ||
            (m.homeDirectory || '').toLowerCase().includes(keyword) ||
            (m.permissions || []).join(' ').toLowerCase().includes(keyword)
        )
    }

    // ─── 方法 ───
    async loadMembers() {
        this.loading = true
        try {
            const res = await api.accountMemberList()
            this.members = res.payload || []
        } finally {
            this.loading = false
        }
    }

    openAddMember() {
        this.memberEditModalRef?.show(null)
    }

    openEditMember(m: MemberInfo) {
        this.memberEditModalRef?.show(m)
    }

    handleDeleteMember(m: MemberInfo) {
        this.portal.showConfirm({
            title: '删除成员',
            message: `确定要删除成员 <strong class="text-slate-900">${m.username}</strong> 吗？此操作仅从配置文件移除，不删除家目录。`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                try {
                    await api.accountMemberDelete(m.username)
                    this.portal.showNotification('success', '成员删除成功')
                    this.loadMembers()
                } catch {}
            }
        })
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadMembers()
    }
}

export default toNative(Members)
</script>

<template>
  <div class="card">
    <!-- Toolbar Bar -->
    <div class="card-toolbar">
      <!-- 桌面端 -->
      <div class="hidden md:flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-blue-500">
            <i class="fas fa-users text-white"></i>
          </div>
          <div>
            <h1 class="text-lg font-semibold text-slate-800 truncate">用户管理</h1>
            <p class="text-xs text-slate-500">管理可登录系统的成员与权限</p>
          </div>
        </div>
        <div class="flex items-center gap-2 flex-shrink-0">
          <PageSearch v-model="searchText" search-key="account-members" placeholder="请输入搜索关键词..." focus-color="blue" type-to-search />
          <button type="button" class="btn btn-secondary" @click="loadMembers">
            <i class="fas fa-rotate"></i>刷新
          </button>
          <button v-if="portal.hasPerm('POST /api/account/member')" type="button" class="btn btn-blue" @click="openAddMember">
            <i class="fas fa-plus"></i>新建用户
          </button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="flex md:hidden items-center justify-between">
        <div class="flex items-center gap-3 min-w-0 flex-1">
          <div class="page-icon bg-blue-500">
            <i class="fas fa-users text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">用户管理</h1>
            <p class="text-xs text-slate-500 truncate">管理可登录系统的成员与权限</p>
          </div>
        </div>
        <div class="flex items-center gap-1 flex-shrink-0">
          <button type="button" class="btn-icon-sm" title="刷新" @click="loadMembers">
            <i class="fas fa-rotate text-sm"></i>
          </button>
          <button v-if="portal.hasPerm('POST /api/account/member')" type="button" class="btn-icon-sm" title="新建用户" @click="openAddMember">
            <i class="fas fa-plus text-sm"></i>
          </button>
        </div>
      </div>
    </div>
    <div class="mobile-search">
      <PageSearch v-model="searchText" search-key="account-members" placeholder="请输入搜索关键词..." width-class="w-full" focus-color="blue" />
    </div>

    <!-- Loading -->
    <div v-if="loading" class="card-body">
      <div class="empty-state">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
    </div>

    <!-- Empty -->
    <div v-else-if="filteredMembers.length === 0" class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-users text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">{{ members.length === 0 ? '暂无成员' : '未找到匹配成员' }}</p>
        <p class="text-sm text-slate-400">{{ members.length === 0 ? '点击「新建用户」创建成员' : '尝试更换关键词或清空搜索条件' }}</p>
      </div>
    </div>

    <!-- Table -->
    <template v-else>
      <!-- 桌面端表格 -->
      <div class="card-table hidden md:block">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-slate-50 border-b border-slate-200">
              <th class="th">用户名</th>
              <th class="th">家目录</th>
              <th class="th">权限</th>
              <th class="w-28 th-right">操作</th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-slate-100">
            <tr v-for="m in filteredMembers" :key="m.username" class="hover:bg-slate-50 transition-colors">
              <td class="px-4 py-3 max-w-[280px]">
                <div class="flex items-center gap-2 min-w-0">
                  <div class="row-icon bg-blue-500">
                    <i class="fas fa-user text-white text-sm"></i>
                  </div>
                  <div class="min-w-0">
                    <span class="font-medium text-slate-800 truncate block">{{ m.username }}</span>
                    <span v-if="m.description" class="text-xs text-slate-400 truncate block mt-0.5">{{ m.description }}</span>
                  </div>
                </div>
              </td>
              <td class="px-4 py-3">
                <code class="text-xs text-slate-600 font-mono">{{ m.homeDirectory }}</code>
              </td>
              <td class="px-4 py-3 text-sm text-slate-600">
                <template v-if="m.founder"><i class="fas fa-crown text-violet-400 mr-1"></i>创始人</template>
                <template v-else-if="m.permissions && m.permissions.length > 0"><i class="fas fa-key text-amber-400 mr-1"></i>{{ m.permissions.length }} 条</template>
                <span v-else class="text-slate-400">-</span>
              </td>
              <td class="px-4 py-3">
                <div class="flex justify-end items-center gap-1">
                  <button v-if="!m.founder && portal.hasPerm('PUT /api/account/member/:username')" class="btn-icon btn-icon-blue" title="编辑" @click="openEditMember(m)">
                    <i class="fas fa-pen text-xs"></i>
                  </button>
                  <button v-if="!m.founder && portal.hasPerm('DELETE /api/account/member/:username')" class="btn-icon btn-icon-red" title="删除" @click="handleDeleteMember(m)">
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
        <div v-for="m in filteredMembers" :key="m.username" class="card-interactive">
          <!-- 顶部：用户信息 -->
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-3 min-w-0 flex-1">
              <div class="list-icon bg-blue-500">
                <i class="fas fa-user text-white text-base"></i>
              </div>
              <div class="min-w-0">
                <span class="font-medium text-slate-800 text-sm truncate block">{{ m.username }}</span>
                <span v-if="m.description" class="text-xs text-slate-400 truncate block mt-0.5">{{ m.description }}</span>
              </div>
            </div>
          </div>
          <!-- 创始人标识 -->
          <div v-if="m.founder" class="card-prop-row">
            <span class="text-xs text-slate-400 flex-shrink-0">身份</span>
            <span class="text-xs text-slate-500"><i class="fas fa-crown text-violet-400 mr-1"></i>创始人</span>
          </div>
          <!-- 家目录 -->
          <div class="card-prop-row-start">
            <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">家目录</span>
            <code class="text-xs bg-slate-100 px-2 py-0.5 rounded break-all">{{ m.homeDirectory }}</code>
          </div>
          <!-- 路由权限 -->
          <div class="card-prop-row">
            <span class="text-xs text-slate-400 flex-shrink-0">权限</span>
            <span v-if="m.permissions && m.permissions.length > 0" class="text-xs text-slate-500"><i class="fas fa-key text-amber-400 mr-1"></i>{{ m.permissions.length }} 条</span>
            <span v-else class="text-xs text-slate-400">-</span>
          </div>
          <!-- 底部：操作按鈕 -->
          <div class="card-actions">
            <button v-if="!m.founder && portal.hasPerm('PUT /api/account/member/:username')" class="btn-icon btn-icon-blue" title="编辑" @click="openEditMember(m)">
              <i class="fas fa-pen text-xs"></i><span class="text-xs ml-1">编辑</span>
            </button>
            <button v-if="!m.founder && portal.hasPerm('DELETE /api/account/member/:username')" class="btn-icon btn-icon-red" title="删除" @click="handleDeleteMember(m)">
              <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
            </button>
          </div>
        </div>
      </div>
    </template>
  </div>

  <MemberEditModal ref="memberEditModalRef" @success="loadMembers" />
</template>
