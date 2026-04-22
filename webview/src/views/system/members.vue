<script lang="ts">
import { Component, Inject, Ref, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { SystemMemberInfo } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

import MemberEditModal from '@/views/system/widget/member-edit-modal.vue'

@Component({
    components: { MemberEditModal }
})
class Members extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    @Ref readonly memberEditModalRef!: InstanceType<typeof MemberEditModal>

    // ─── 数据属性 ───
    members: SystemMemberInfo[] = []
    membersLoading = false

    // ─── 方法 ───
    async loadMembers() {
        this.membersLoading = true
        try {
            const res = await api.listMembers()
            this.members = res.payload || []
        } catch (e) {
            this.actions.showNotification('error', '加载成员列表失败')
        }
        this.membersLoading = false
    }

    openAddMember() {
        this.memberEditModalRef?.show(null)
    }

    openEditMember(m: SystemMemberInfo) {
        this.memberEditModalRef?.show(m)
    }

    handleDeleteMember(m: SystemMemberInfo) {
        this.actions.showConfirm({
            title: '删除成员',
            message: `确定要删除成员 <strong class="text-slate-900">${m.username}</strong> 吗？此操作仅从配置文件移除，不删除 home 目录。`,
            icon: 'fa-trash',
            iconColor: 'red',
            confirmText: '确认删除',
            danger: true,
            onConfirm: async () => {
                try {
                    await api.deleteMember(m.username)
                    this.actions.showNotification('success', '成员删除成功')
                    this.loadMembers()
                } catch (e) {}
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
  <div>
    <div class="card mb-4">
      <!-- Toolbar Bar -->
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3">
        <!-- 桌面端 -->
        <div class="hidden md:flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-blue-500 flex items-center justify-center">
              <i class="fas fa-users text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">用户管理</h1>
              <p class="text-xs text-slate-500">管理可登录系统的成员与权限</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <button type="button" @click="loadMembers" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-rotate"></i>刷新
            </button>
            <button type="button" @click="openAddMember" class="px-3 py-1.5 rounded-lg bg-blue-500 hover:bg-blue-600 text-white text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-plus"></i>添加
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="flex md:hidden items-center justify-between">
          <div class="flex items-center gap-3 min-w-0 flex-1">
            <div class="w-9 h-9 rounded-lg bg-blue-500 flex items-center justify-center flex-shrink-0">
              <i class="fas fa-users text-white"></i>
            </div>
            <div class="min-w-0">
              <h1 class="text-lg font-semibold text-slate-800 truncate">用户管理</h1>
              <p class="text-xs text-slate-500 truncate">管理可登录系统的成员与权限</p>
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <button type="button" @click="loadMembers" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 flex items-center justify-center text-slate-600 transition-colors" title="刷新">
              <i class="fas fa-rotate text-sm"></i>
            </button>
            <button type="button" @click="openAddMember" class="w-9 h-9 rounded-lg bg-blue-500 hover:bg-blue-600 flex items-center justify-center text-white transition-colors" title="添加">
              <i class="fas fa-plus text-sm"></i>
            </button>
          </div>
        </div>
      </div>

      <!-- Loading -->
      <div v-if="membersLoading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <!-- Empty -->
      <div v-else-if="members.length === 0" class="flex flex-col items-center justify-center py-20">
        <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
          <i class="fas fa-users text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">暂无成员</p>
        <p class="text-sm text-slate-400">点击右上角「添加」创建成员</p>
      </div>

      <!-- Table -->
      <div v-else class="space-y-3">
        <!-- 桌面端表格 -->
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
              <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                  <th class="w-1/4 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">用户名</th>
                  <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">Home 目录</th>
                  <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">模块权限</th>
                  <th class="w-28 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">密码</th>
                  <th class="w-28 px-4 py-3 text-right text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
                </tr>
              </thead>
              <tbody class="bg-white divide-y divide-slate-100">
                <tr v-for="m in members" :key="m.username" class="hover:bg-slate-50 transition-colors">
                  <td class="px-4 py-3">
                    <div class="flex items-center gap-2">
                      <div class="w-8 h-8 rounded-lg bg-blue-500 flex items-center justify-center">
                        <i class="fas fa-user text-white text-sm"></i>
                      </div>
                      <span class="font-medium text-slate-800">{{ m.username }}</span>
                    </div>
                  </td>
                  <td class="px-4 py-3"><code class="text-xs bg-slate-100 px-2 py-1 rounded">{{ m.homeDirectory }}</code></td>
                  <td class="px-4 py-3">
                    <div class="flex flex-wrap gap-1">
                      <template v-for="(perm, mod) in m.permissions" :key="mod">
                        <span v-if="perm === 'rw'" class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-green-50 text-green-700">{{ mod }}<span class="ml-0.5 opacity-60">rw</span></span>
                        <span v-else-if="perm === 'r'" class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-blue-50 text-blue-600">{{ mod }}<span class="ml-0.5 opacity-60">r</span></span>
                      </template>
                      <span v-if="!Object.values(m.permissions || {}).some(p => p)" class="text-xs text-slate-400">无权限</span>
                    </div>
                  </td>
                  <td class="px-4 py-3">
                    <span v-if="m.passwordSet" class="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-green-50 text-green-700">
                      <i class="fas fa-check mr-1"></i>已设置
                    </span>
                    <span v-else class="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-amber-50 text-amber-700">
                      <i class="fas fa-exclamation mr-1"></i>未设置
                    </span>
                  </td>
                  <td class="px-4 py-3">
                    <div class="flex justify-end items-center gap-0.5">
                      <button @click="openEditMember(m)" class="btn-icon text-blue-600 hover:bg-blue-50" title="编辑">
                        <i class="fas fa-pen text-xs"></i>
                      </button>
                      <button @click="handleDeleteMember(m)" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
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
          <div v-for="m in members" :key="m.username" class="rounded-xl border border-slate-200 bg-white p-4 transition-all hover:shadow-sm">
            <!-- 顶部：用户信息 -->
            <div class="flex items-center justify-between mb-3">
                <div class="flex items-center gap-3 min-w-0 flex-1">
                <div class="w-10 h-10 rounded-lg bg-blue-500 flex items-center justify-center flex-shrink-0">
                  <i class="fas fa-user text-white text-base"></i>
                </div>
                <div class="min-w-0">
                  <div class="flex items-center gap-2">
                    <span class="font-medium text-slate-800 text-sm truncate">{{ m.username }}</span>
                  </div>
                  <div class="flex flex-wrap items-center gap-2 mt-1">
                    <span v-if="m.passwordSet" class="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-green-50 text-green-700">
                      <i class="fas fa-check mr-1"></i>密码已设
                    </span>
                    <span v-else class="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-amber-50 text-amber-700">
                      <i class="fas fa-exclamation mr-1"></i>无密码
                    </span>
                  </div>
                </div>
              </div>
            </div>
            <!-- 中间：Home 目录 -->
            <div class="flex items-start gap-2 mb-3">
              <span class="text-xs text-slate-400 flex-shrink-0 mt-0.5">Home</span>
              <code class="text-xs bg-slate-100 px-2 py-1 rounded break-all">{{ m.homeDirectory }}</code>
            </div>
            <!-- 模块权限 -->
            <div class="mb-3">
              <span class="text-xs text-slate-400 mr-2">权限</span>
              <span class="inline-flex flex-wrap gap-1">
                <template v-for="(perm, mod) in m.permissions" :key="mod">
                  <span v-if="perm === 'rw'" class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-green-50 text-green-700">{{ mod }}<span class="ml-0.5 opacity-60">rw</span></span>
                  <span v-else-if="perm === 'r'" class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-blue-50 text-blue-600">{{ mod }}<span class="ml-0.5 opacity-60">r</span></span>
                </template>
                <span v-if="!Object.values(m.permissions || {}).some(p => p)" class="text-xs text-slate-400">无权限</span>
              </span>
            </div>
            <!-- 底部：操作按鈕 -->
            <div class="flex flex-wrap gap-1 pt-2 border-t border-slate-100">
              <button @click="openEditMember(m)" class="btn-icon text-blue-600 hover:bg-blue-50" title="编辑">
                <i class="fas fa-pen text-xs"></i><span class="text-xs ml-1">编辑</span>
              </button>
              <button @click="handleDeleteMember(m)" class="btn-icon text-red-600 hover:bg-red-50" title="删除">
                <i class="fas fa-trash text-xs"></i><span class="text-xs ml-1">删除</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <MemberEditModal ref="memberEditModalRef" @success="loadMembers" />
  </div>
</template>
