<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { SwarmNode } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

@Component
class Nodes extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    nodes: SwarmNode[] = []
    nodesLoading = false

    // ─── 方法 ───
    nodeStateClass(state: string) {
        if (state === 'ready') return 'bg-emerald-100 text-emerald-700'
        if (state === 'down') return 'bg-red-100 text-red-700'
        return 'bg-slate-100 text-slate-600'
    }

    availabilityClass(avail: string) {
        if (avail === 'active') return 'bg-emerald-100 text-emerald-700'
        if (avail === 'drain') return 'bg-amber-100 text-amber-700'
        if (avail === 'pause') return 'bg-slate-100 text-slate-600'
        return 'bg-slate-100 text-slate-500'
    }

    async loadNodes() {
        this.nodesLoading = true
        try {
            const res = await api.swarmListNodes()
            const list = res.payload || []
            // leader 节点排最前
            this.nodes = list.sort((a: SwarmNode, b: SwarmNode) => (b.leader ? 1 : 0) - (a.leader ? 1 : 0))
        } catch (e) {
            this.actions.showNotification('error', '获取节点列表失败')
        }
        this.nodesLoading = false
    }

    handleNodeAction(node: SwarmNode, action: string) {
        const labels: Record<string, string> = { drain: '排空', active: '激活', pause: '暂停', remove: '移除' }
        const label = labels[action] || action
        this.actions.showConfirm({
            title: `${label}节点`,
            message: `确定要${label}节点 <strong class="text-slate-900">${node.hostname}</strong> 吗？`,
            icon: action === 'remove' ? 'fa-trash' : 'fa-server',
            iconColor: action === 'remove' ? 'red' : 'amber',
            confirmText: `确认${label}`,
            danger: action === 'remove',
            onConfirm: async () => {
                await api.swarmNodeAction(node.id, action)
                this.actions.showNotification('success', `节点${label}成功`)
                this.loadNodes()
            }
        })
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadNodes()
    }
}

export default toNative(Nodes)
</script>

<template>
  <div>
    <div class="card mb-4">
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-6 py-3 flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="w-9 h-9 rounded-lg bg-blue-500 flex items-center justify-center">
            <i class="fas fa-server text-white"></i>
          </div>
          <div>
            <h1 class="text-lg font-semibold text-slate-800">节点管理</h1>
            <p class="text-xs text-slate-500">管理 Swarm 集群节点</p>
          </div>
        </div>
        <button @click="loadNodes" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
          <i class="fas fa-rotate"></i>刷新
        </button>
      </div>

      <div v-if="nodesLoading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
      <div v-else-if="nodes.length > 0" class="space-y-3">
        <!-- 桌面端表格视图 -->
        <div class="hidden md:block overflow-x-auto">
          <table class="w-full border-collapse">
            <thead>
              <tr class="bg-slate-50 border-b border-slate-200">
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">主机名</th>
                <th class="w-24 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">角色</th>
                <th class="w-28 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">状态</th>
                <th class="w-28 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">可用性</th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">地址</th>
                <th class="w-28 px-4 py-3 text-left text-xs font-semibold text-slate-600 uppercase tracking-wider">引擎版本</th>
                <th class="w-44 px-4 py-3 text-right text-xs font-semibold text-slate-600 uppercase tracking-wider">操作</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-slate-100">
              <tr v-for="n in nodes" :key="n.id" class="hover:bg-slate-50 transition-colors">
                <td class="px-4 py-3">
                  <div class="flex items-center gap-2">
                    <div class="w-8 h-8 rounded-lg bg-blue-400 flex items-center justify-center">
                      <i class="fas fa-server text-white text-sm"></i>
                    </div>
                    <div>
                      <span class="font-medium text-slate-800">{{ n.hostname }}</span>
                      <span v-if="n.leader" class="ml-2 inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-indigo-100 text-indigo-700">
                        <i class="fas fa-crown mr-1 text-[10px]"></i>Leader
                      </span>
                    </div>
                  </div>
                </td>
                <td class="px-4 py-3">
                  <span :class="n.role === 'manager' ? 'bg-indigo-100 text-indigo-700' : 'bg-slate-100 text-slate-600'" class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize">{{ n.role }}</span>
                </td>
                <td class="px-4 py-3">
                  <span :class="nodeStateClass(n.state)" class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize">{{ n.state }}</span>
                </td>
                <td class="px-4 py-3">
                  <span :class="availabilityClass(n.availability)" class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize">{{ n.availability }}</span>
                </td>
                <td class="px-4 py-3 font-mono text-xs text-slate-500">{{ n.addr || '-' }}</td>
                <td class="px-4 py-3 text-xs text-slate-500">{{ n.engineVersion || '-' }}</td>
                <td class="px-4 py-3">
                  <div class="flex justify-end items-center gap-0.5">
                    <button @click="$router.push(`/swarm/node/${n.id}`)" class="btn-icon text-slate-600 hover:bg-slate-100" title="查看详情"><i class="fas fa-circle-info text-xs"></i></button>
                    <button v-if="n.availability !== 'active'" @click="handleNodeAction(n, 'active')" class="btn-icon text-emerald-600 hover:bg-emerald-50" title="激活"><i class="fas fa-play text-xs"></i></button>
                    <button v-if="n.availability !== 'drain'"  @click="handleNodeAction(n, 'drain')"  class="btn-icon text-amber-600 hover:bg-amber-50"   title="排空"><i class="fas fa-arrow-down text-xs"></i></button>
                    <button v-if="n.availability !== 'pause'"  @click="handleNodeAction(n, 'pause')"  class="btn-icon text-slate-600 hover:bg-slate-100"   title="暂停"><i class="fas fa-pause text-xs"></i></button>
                    <button @click="n.leader ? null : handleNodeAction(n, 'remove')" :disabled="n.leader" :class="n.leader ? 'btn-icon text-slate-300 cursor-not-allowed' : 'btn-icon text-red-600 hover:bg-red-50'" :title="n.leader ? '不能移除 Leader 节点' : '移除'"><i class="fas fa-trash text-xs"></i></button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 移动端卡片视图 -->
        <div class="md:hidden space-y-3">
          <div 
            v-for="n in nodes" 
            :key="n.id"
            class="rounded-xl border border-slate-200 bg-white p-4 transition-all hover:shadow-sm"
          >
            <!-- 顶部：主机名和图标 -->
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-blue-400 flex items-center justify-center">
                  <i class="fas fa-server text-white text-base"></i>
                </div>
                <div class="min-w-0">
                  <div class="flex items-center gap-2">
                    <span class="font-medium text-slate-800 text-sm">{{ n.hostname }}</span>
                    <span v-if="n.leader" class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-indigo-100 text-indigo-700">
                      <i class="fas fa-crown mr-1 text-[10px]"></i>Leader
                    </span>
                  </div>
                  <div class="text-xs text-slate-500 mt-1">{{ n.addr || '-' }}</div>
                </div>
              </div>
            </div>
            
            <!-- 中间：角色、状态、可用性信息 -->
            <div class="grid grid-cols-2 gap-3 mb-3">
              <div>
                <p class="text-xs text-slate-500 mb-1">角色</p>
                <span :class="n.role === 'manager' ? 'bg-indigo-100 text-indigo-700' : 'bg-slate-100 text-slate-600'" class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize">{{ n.role }}</span>
              </div>
              <div>
                <p class="text-xs text-slate-500 mb-1">状态</p>
                <span :class="nodeStateClass(n.state)" class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize">{{ n.state }}</span>
              </div>
              <div>
                <p class="text-xs text-slate-500 mb-1">可用性</p>
                <span :class="availabilityClass(n.availability)" class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize">{{ n.availability }}</span>
              </div>
              <div>
                <p class="text-xs text-slate-500 mb-1">引擎版本</p>
                <span class="text-xs text-slate-600">{{ n.engineVersion || '-' }}</span>
              </div>
            </div>
            
            <!-- 底部：操作按钮 -->
            <div class="flex flex-wrap gap-1 pt-2 border-t border-slate-100">
              <button @click="$router.push(`/swarm/node/${n.id}`)" class="btn-icon text-slate-600 hover:bg-slate-100" title="查看详情">
                <i class="fas fa-circle-info text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">详情</span>
              </button>
              <button v-if="n.availability !== 'active'" @click="handleNodeAction(n, 'active')" class="btn-icon text-emerald-600 hover:bg-emerald-50" title="激活">
                <i class="fas fa-play text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">激活</span>
              </button>
              <button v-if="n.availability !== 'drain'"  @click="handleNodeAction(n, 'drain')"  class="btn-icon text-amber-600 hover:bg-amber-50"   title="排空">
                <i class="fas fa-arrow-down text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">排空</span>
              </button>
              <button v-if="n.availability !== 'pause'"  @click="handleNodeAction(n, 'pause')"  class="btn-icon text-slate-600 hover:bg-slate-100"   title="暂停">
                <i class="fas fa-pause text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">暂停</span>
              </button>
              <button @click="n.leader ? null : handleNodeAction(n, 'remove')" :disabled="n.leader" :class="n.leader ? 'btn-icon text-slate-300 cursor-not-allowed' : 'btn-icon text-red-600 hover:bg-red-50'" :title="n.leader ? '不能移除 Leader 节点' : '移除'">
                <i class="fas fa-trash text-xs"></i>
                <span class="text-xs ml-1 hidden xs:inline">移除</span>
              </button>
            </div>
          </div>
        </div>
      </div>
      <div v-else class="flex flex-col items-center justify-center py-20">
        <div class="w-20 h-20 rounded-full bg-slate-100 flex items-center justify-center mb-4">
          <i class="fas fa-server text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">暂无节点</p>
      </div>
    </div>
  </div>
</template>
