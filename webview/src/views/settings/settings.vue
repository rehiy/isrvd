<script lang="ts">
import { Component, Inject, Vue, toNative } from 'vue-facing-decorator'

import api from '@/service/api'
import type { AllSettings, ServerSettings, ApisixSettings, DockerSettings } from '@/service/types'
import { APP_ACTIONS_KEY } from '@/store/state'
import type { AppActions } from '@/store/state'

type TabKey = 'server' | 'apisix' | 'docker'

@Component({})
class Settings extends Vue {
    @Inject({ from: APP_ACTIONS_KEY }) readonly actions!: AppActions

    // ─── 数据属性 ───
    loading = false
    saving = false
    activeTab: TabKey = 'server'

    server: ServerSettings = { debug: false, listenAddr: '', jwtSecret: '', proxyHeaderName: '', rootDirectory: '' }
    apisix: ApisixSettings = { adminUrl: '', adminKey: '' }
    docker: DockerSettings = { host: '', containerRoot: '' }

    // 敏感字段当前是否已设置（后端返回）
    jwtSecretSet = false
    adminKeySet = false

    // 敏感字段 placeholder
    get jwtSecretPlaceholder() {
        return this.jwtSecretSet ? '已设置（留空保持不变）' : '尚未设置'
    }

    get adminKeyPlaceholder() {
        return this.adminKeySet ? '已设置（留空保持不变）' : '尚未设置'
    }

    tabs: { key: TabKey; label: string; icon: string; desc: string }[] = [
        { key: 'server', label: '服务器', icon: 'fa-server', desc: '修改服务器监听地址、JWT 密钥及代理配置' },
        { key: 'docker', label: 'Docker', icon: 'fa-cube', desc: '修改 Docker 守护进程及容器数据目录' },
        { key: 'apisix', label: 'APISIX', icon: 'fa-cloud', desc: '修改 APISIX Admin API 连接信息' }
    ]

    get currentTabDesc() {
        return this.tabs.find(t => t.key === this.activeTab)?.desc || '修改服务器、Docker 及 APISIX 配置'
    }

    // ─── 方法 ───
    async loadSettings() {
        this.loading = true
        try {
            const res = await api.getSettings()
            const payload = res.payload as AllSettings
            // 敏感字段统一置空，仅用标志位驱动 placeholder
            this.server = { ...payload.server, jwtSecret: '' }
            this.apisix = { ...payload.apisix, adminKey: '' }
            this.docker = { ...payload.docker }
            this.jwtSecretSet = !!payload.server.jwtSecretSet
            this.adminKeySet = !!payload.apisix.adminKeySet
        } catch (e) {
            this.actions.showNotification('error', '加载配置失败')
        }
        this.loading = false
    }

    async saveServer() {
        this.saving = true
        try {
            await api.updateServerSettings(this.server)
            this.actions.showNotification('success', '服务器配置已保存，部分项需重启生效')
            this.loadSettings()
        } catch (e) {}
        this.saving = false
    }

    async saveApisix() {
        this.saving = true
        try {
            await api.updateApisixSettings(this.apisix)
            this.actions.showNotification('success', 'APISIX 配置已保存，重启后生效')
            this.loadSettings()
        } catch (e) {}
        this.saving = false
    }

    async saveDocker() {
        this.saving = true
        try {
            await api.updateDockerSettings(this.docker)
            this.actions.showNotification('success', 'Docker 配置已保存，重启后生效')
            this.loadSettings()
        } catch (e) {}
        this.saving = false
    }

    goToRegistries() {
        this.$router.push('/docker/registries')
    }

    // ─── 生命周期 ───
    mounted() {
        this.loadSettings()
    }
}

export default toNative(Settings)
</script>

<template>
  <div>
    <div class="card mb-4">
      <!-- Toolbar Bar -->
      <div class="bg-slate-50 border-b border-slate-200 rounded-t-2xl px-4 md:px-6 py-3">
        <!-- 桌面端 -->
        <div class="hidden md:flex md:items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-slate-700 flex items-center justify-center">
              <i class="fas fa-gear text-white"></i>
            </div>
            <div>
              <h1 class="text-lg font-semibold text-slate-800">系统设置</h1>
              <p class="text-xs text-slate-500">{{ currentTabDesc }}</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <div class="flex gap-1 bg-slate-100 p-1 rounded-lg">
              <button
                v-for="tab in tabs"
                :key="tab.key"
                @click="activeTab = tab.key"
                :class="[
                  'px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5',
                  activeTab === tab.key ? 'bg-white text-blue-600 shadow-sm' : 'text-slate-500 hover:text-slate-700'
                ]"
              >
                <i :class="['fas', tab.icon]"></i><span>{{ tab.label }}</span>
              </button>
            </div>
            <button @click="loadSettings()" class="px-3 py-1.5 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 text-xs font-medium flex items-center gap-1.5 transition-colors">
              <i class="fas fa-rotate"></i>刷新
            </button>
          </div>
        </div>
        <!-- 移动端 -->
        <div class="block md:hidden">
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-3 min-w-0">
              <div class="w-9 h-9 rounded-lg bg-slate-700 flex items-center justify-center flex-shrink-0">
                <i class="fas fa-gear text-white"></i>
              </div>
              <div class="min-w-0">
                <h1 class="text-lg font-semibold text-slate-800">系统设置</h1>
                <p class="text-xs text-slate-500 truncate">{{ currentTabDesc }}</p>
              </div>
            </div>
            <button @click="loadSettings()" class="w-9 h-9 rounded-lg bg-white border border-slate-200 hover:bg-slate-50 flex items-center justify-center text-slate-600 transition-colors flex-shrink-0">
              <i class="fas fa-rotate text-sm"></i>
            </button>
          </div>
          <div class="flex justify-center gap-1 bg-slate-100 p-1 rounded-lg">
            <button
              v-for="tab in tabs"
              :key="tab.key"
              @click="activeTab = tab.key"
              :class="[
                'px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 flex items-center gap-1.5',
                activeTab === tab.key ? 'bg-white text-blue-600 shadow-sm' : 'text-slate-500 hover:text-slate-700'
              ]"
            >
              <i :class="['fas', tab.icon]"></i><span>{{ tab.label }}</span>
            </button>
          </div>
        </div>
      </div>
      <!-- Loading -->
      <div v-if="loading" class="flex flex-col items-center justify-center py-20">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>

      <div v-else class="p-6">
        <!-- 服务器配置 -->
        <form v-if="activeTab === 'server'" @submit.prevent="saveServer" class="max-w-3xl space-y-4">
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">监听地址</label>
            <input type="text" v-model="server.listenAddr" placeholder=":8080" class="input" />
            <p class="mt-1 text-xs text-slate-400">HTTP 服务监听端口，例如 :8080 或 127.0.0.1:8080（重启生效）</p>
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">基础目录</label>
            <input type="text" v-model="server.rootDirectory" placeholder="." class="input" />
            <p class="mt-1 text-xs text-slate-400">成员 home 目录及容器数据的基础目录</p>
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">
              JWT 密钥
              <span v-if="jwtSecretSet" class="ml-1 inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-green-50 text-green-700"><i class="fas fa-check mr-0.5"></i>已设置</span>
              <span v-else class="ml-1 inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-slate-100 text-slate-500">未设置</span>
            </label>
            <input type="password" v-model="server.jwtSecret" :placeholder="jwtSecretPlaceholder" class="input" autocomplete="new-password" />
            <p class="mt-1 text-xs text-slate-400">用于签名登录令牌，修改后所有用户需要重新登录</p>
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">代理用户名 Header</label>
            <input type="text" v-model="server.proxyHeaderName" placeholder="例如 X-Auth-User（留空禁用）" class="input" />
            <p class="mt-1 text-xs text-slate-400">启用内网代理认证时使用，网关会注入该 Header 作为登录用户</p>
          </div>
          <div class="flex items-center gap-2">
            <input id="debugSwitch" type="checkbox" v-model="server.debug" class="w-4 h-4" />
            <label for="debugSwitch" class="text-sm text-slate-700">启用 Debug 模式</label>
          </div>
          <div class="pt-2 flex flex-col sm:flex-row sm:items-center gap-3">
            <button type="submit" :disabled="saving" class="self-start px-4 py-2 rounded-lg bg-blue-500 hover:bg-blue-600 disabled:opacity-50 text-white text-xs font-medium flex items-center gap-1.5 transition-colors whitespace-nowrap flex-shrink-0">
              <i :class="['fas', saving ? 'fa-spinner fa-spin' : 'fa-save']"></i>{{ saving ? '保存中...' : '保存服务器配置' }}
            </button>
            <p class="text-xs text-slate-400 flex items-start gap-1">
              <i class="fas fa-circle-info mt-0.5"></i>
              <span>保存后立即写入配置文件，监听地址及 JWT 密钥修改需重启服务生效；密钥留空即保留原值。</span>
            </p>
          </div>
        </form>

        <!-- Docker 配置 -->
        <form v-else-if="activeTab === 'docker'" @submit.prevent="saveDocker" class="max-w-3xl space-y-4">
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">Docker Host</label>
            <input type="text" v-model="docker.host" placeholder="unix:///var/run/docker.sock 或 tcp://host:2375" class="input" />
            <p class="mt-1 text-xs text-slate-400">Docker 守护进程地址，留空则使用环境变量</p>
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">容器数据根目录</label>
            <input type="text" v-model="docker.containerRoot" placeholder="containers" class="input" />
            <p class="mt-1 text-xs text-slate-400">用于存放容器数据卷的基础目录（相对于基础目录）</p>
          </div>
          <div class="pt-2 flex flex-col sm:flex-row sm:items-center gap-3">
            <button type="submit" :disabled="saving" class="self-start px-4 py-2 rounded-lg bg-blue-500 hover:bg-blue-600 disabled:opacity-50 text-white text-xs font-medium flex items-center gap-1.5 transition-colors whitespace-nowrap flex-shrink-0">
              <i :class="['fas', saving ? 'fa-spinner fa-spin' : 'fa-save']"></i>{{ saving ? '保存中...' : '保存 Docker 配置' }}
            </button>
            <p class="text-xs text-slate-400 flex items-start gap-1">
              <i class="fas fa-circle-info mt-0.5"></i>
              <span>保存后立即写入配置文件，Docker Host 变更需重启服务后才能连接新地址。</span>
            </p>
          </div>

          <!-- 镜像仓库跳转卡片 -->
          <div class="mt-6 pt-4 border-t border-slate-200">
            <button type="button" @click="goToRegistries" class="w-full flex items-center justify-between gap-4 p-4 rounded-xl border border-slate-200 hover:border-purple-300 hover:bg-purple-50/40 transition-colors text-left group">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-purple-500 flex items-center justify-center flex-shrink-0">
                  <i class="fas fa-warehouse text-white"></i>
                </div>
                <div>
                  <div class="text-sm font-semibold text-slate-800">镜像仓库管理</div>
                  <div class="text-xs text-slate-500">管理私有镜像仓库账号与加速器，用于镜像推送与拉取</div>
                </div>
              </div>
              <i class="fas fa-chevron-right text-slate-400 group-hover:text-purple-500 transition-colors"></i>
            </button>
          </div>
        </form>

        <!-- APISIX 配置 -->
        <form v-else-if="activeTab === 'apisix'" @submit.prevent="saveApisix" class="max-w-3xl space-y-4">
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">Admin URL</label>
            <input type="text" v-model="apisix.adminUrl" placeholder="http://127.0.0.1:9180" class="input" />
            <p class="mt-1 text-xs text-slate-400">APISIX Admin API 地址</p>
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-700 mb-2">
              Admin Key
              <span v-if="adminKeySet" class="ml-1 inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-green-50 text-green-700"><i class="fas fa-check mr-0.5"></i>已设置</span>
              <span v-else class="ml-1 inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-slate-100 text-slate-500">未设置</span>
            </label>
            <input type="password" v-model="apisix.adminKey" :placeholder="adminKeyPlaceholder" class="input" autocomplete="new-password" />
            <p class="mt-1 text-xs text-slate-400">访问 APISIX Admin API 的密钥</p>
          </div>
          <div class="pt-2 flex flex-col sm:flex-row sm:items-center gap-3">
            <button type="submit" :disabled="saving" class="self-start px-4 py-2 rounded-lg bg-blue-500 hover:bg-blue-600 disabled:opacity-50 text-white text-xs font-medium flex items-center gap-1.5 transition-colors whitespace-nowrap flex-shrink-0">
              <i :class="['fas', saving ? 'fa-spinner fa-spin' : 'fa-save']"></i>{{ saving ? '保存中...' : '保存 APISIX 配置' }}
            </button>
            <p class="text-xs text-slate-400 flex items-start gap-1">
              <i class="fas fa-circle-info mt-0.5"></i>
              <span>保存后立即写入配置文件，Admin URL 及 Admin Key 变更需重启服务生效；密钥留空即保留原值。</span>
            </p>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
