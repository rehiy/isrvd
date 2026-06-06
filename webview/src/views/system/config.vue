<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { AllConfig, ServerConfig, THAConfig, OIDCConfig, PasskeyConfig, AgentConfig, ApisixConfig, CaddyConfig, DockerConfig, MonitorConfig, MarketplaceConfig, LinkConfig } from '@/service/types'

import IconSelect from '@/component/icon-select.vue'

type ConfigTab = 'server' | 'tha' | 'oidc' | 'passkey' | 'agent' | 'apisix' | 'caddy' | 'docker' | 'monitor' | 'marketplace' | 'links'

@Component({ components: { IconSelect } })
class Config extends Vue {
  portal = usePortal()

  // ─── 数据属性 ───
  loading = false
  saving = false
  activeTab: ConfigTab = 'server'

  server: ServerConfig = { listenAddr: '', rootDirectory: '', maxUploadSize: 104857600, allowedOrigins: [], jwtExpiration: 86400, debug: false }
  allowedOriginsText = ''
  tha: THAConfig = { enabled: false, headerName: '', trustedCIDRs: [] }
  thaTrustedCIDRsText = ''
  oidc: OIDCConfig = { enabled: false, issuerUrl: '', clientId: '', redirectUrl: '', usernameClaim: 'sub', scopes: ['openid', 'profile', 'email'], loginLabel: '' }
  oidcScopes = 'openid profile email'
  passkey: PasskeyConfig = { enabled: false, rpName: '', rpId: '', rpOrigins: [], timeout: 60000 }
  passkeyOriginsText = ''
  agent: AgentConfig = { model: '', baseUrl: '' }
  apisix: ApisixConfig = { adminUrl: '' }
  caddy: CaddyConfig = { adminUrl: '' }
  docker: DockerConfig = { host: '', containerRoot: '' }
  monitor: MonitorConfig = { interval: 0 }
  marketplace: MarketplaceConfig = { url: '' }
  links: LinkConfig[] = []

  get configSections(): Array<{ id: ConfigTab; label: string; description: string; icon: string }> {
    return [
      { id: 'server', label: 'Server', description: '端口、目录、上传、跨域与 JWT', icon: 'fa-server' },
      { id: 'tha', label: '代理 Header 登录', description: '从上游代理 Header 读取用户名', icon: 'fa-user-shield' },
      { id: 'oidc', label: 'OIDC 登录', description: '单点登录 Provider 参数', icon: 'fa-circle-nodes' },
      { id: 'passkey', label: 'Passkey 登录', description: 'WebAuthn/FIDO2 登录', icon: 'fa-fingerprint' },
      { id: 'agent', label: 'AI 助手', description: 'LLM 代理与模型改写', icon: 'fa-robot' },
      { id: 'apisix', label: 'APISIX', description: 'Admin API 连接参数', icon: 'fa-route' },
      { id: 'caddy', label: 'Caddy', description: 'Admin API 连接参数', icon: 'fa-globe' },
      { id: 'docker', label: 'Docker', description: '引擎连接与容器根目录', icon: 'fa-boxes-stacked' },
      { id: 'monitor', label: '监控日志', description: '系统与容器监控采集', icon: 'fa-chart-line' },
      { id: 'marketplace', label: '应用市场', description: '市场 iframe 站点地址', icon: 'fa-store' },
      { id: 'links', label: '导航链接', description: '顶部工具栏外部链接', icon: 'fa-link' }
    ]
  }

  // ─── 方法 ───
  async loadConfig(reload = false) {
    this.loading = true
    try {
      const res = await api.systemConfig(reload ? { reload: 'true' } : undefined)
      const payload = res.payload as AllConfig
      this.server = { ...payload.server }
      this.allowedOriginsText = (this.server.allowedOrigins || []).join('\n')
      this.tha = { ...payload.tha }
      this.thaTrustedCIDRsText = (this.tha.trustedCIDRs || []).join('\n')
      this.oidc = { ...(payload.oidc || { enabled: false, issuerUrl: '', clientId: '', redirectUrl: '', usernameClaim: 'sub', scopes: ['openid', 'profile', 'email'], loginLabel: '' }) }
      this.oidcScopes = (this.oidc.scopes || []).join(' ')
      this.passkey = { ...(payload.passkey || { enabled: false, rpName: '', rpId: '', rpOrigins: [], timeout: 60000 }) }
      this.passkeyOriginsText = (this.passkey.rpOrigins || []).join('\n')
      this.agent = { ...payload.agent }
      this.apisix = { ...payload.apisix }
      this.caddy = { ...(payload.caddy || { adminUrl: '' }) }
      this.docker = { ...payload.docker }
      this.monitor = { ...(payload.monitor || { interval: 0 }) }
      this.marketplace = { ...(payload.marketplace || { url: '' }) }
      this.links = payload.links ? payload.links.map(l => ({ ...l })) : []
      if (reload) {
        this.portal.showNotification('success', '配置已重载')
      }
    } catch {
      this.portal.showNotification('error', reload ? '重载配置失败' : '加载配置失败')
    } finally {
      this.loading = false
    }
  }

  async saveAll() {
    this.saving = true
    try {
      const payload = {
        server: { ...this.server, allowedOrigins: this.allowedOriginsText.split(/\s+/).filter(Boolean) },
        tha: { ...this.tha, trustedCIDRs: this.thaTrustedCIDRsText.split(/\s+/).filter(Boolean) },
        oidc: { ...this.oidc, scopes: this.oidcScopes.split(/\s+/).filter(Boolean) },
        passkey: { ...this.passkey, rpOrigins: this.passkeyOriginsText.split(/\s+/).filter(Boolean) },
        agent: this.agent,
        apisix: this.apisix,
        caddy: this.caddy,
        docker: this.docker,
        monitor: this.monitor,
        marketplace: this.marketplace,
        links: this.links,
      }
      await api.systemConfigUpdate(payload)
      await this.portal.loadSystemData()
      this.portal.showNotification('success', '全部配置已保存，监听地址变更需重启生效')
      await this.loadConfig()
    } catch (e) {
      console.error('保存配置失败:', e)
      this.portal.showNotification('error', '保存配置失败')
    } finally {
      this.saving = false
    }
  }

  addLink() {
    this.links.push({ label: '', url: '', icon: 'fas fa-link' })
  }

  removeLink(index: number) {
    this.links.splice(index, 1)
  }

  scrollToConfigSection(id: ConfigTab) {
    this.activeTab = id
    this.$nextTick(() => {
      const el = document.getElementById(`config-${id}`)
      if (!el) return
      const offset = window.innerWidth < 1024 ? 152 : 88
      const top = el.getBoundingClientRect().top + window.scrollY - offset
      window.scrollTo({ top, behavior: 'smooth' })
    })
  }

  // ─── 生命周期 ───
  mounted() {
    this.loadConfig()
  }
}

export default toNative(Config)
</script>

<template>
  <div class="card overflow-visible">
    <!-- Toolbar -->
    <div class="card-toolbar">
      <!-- 桌面端 -->
      <div class="hidden md:flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="page-icon bg-indigo-500">
            <i class="fas fa-gear text-white"></i>
          </div>
          <div>
            <h1 class="text-lg font-semibold text-slate-800 truncate">系统配置</h1>
            <p class="text-xs text-slate-500">管理服务器、认证、网关与容器引擎参数</p>
          </div>
        </div>
        <div class="flex items-center gap-2 flex-shrink-0">
          <button type="button" class="btn btn-secondary" @click="loadConfig(true)">
            <i class="fas fa-rotate"></i>重载
          </button>
          <button v-if="portal.hasPerm('PATCH /api/system/config')" type="button" class="btn btn-indigo rounded-xl whitespace-nowrap" :disabled="saving" @click="saveAll">
            <i v-if="saving" class="fas fa-spinner fa-spin"></i>
            <i v-else class="fas fa-save"></i>
            <span>{{ saving ? '保存中...' : '保存配置' }}</span>
          </button>
        </div>
      </div>
      <!-- 移动端 -->
      <div class="flex md:hidden items-center justify-between">
        <div class="flex items-center gap-3 min-w-0 flex-1">
          <div class="page-icon bg-indigo-500">
            <i class="fas fa-gear text-white"></i>
          </div>
          <div class="min-w-0">
            <h1 class="text-lg font-semibold text-slate-800 truncate">系统配置</h1>
            <p class="text-xs text-slate-500 truncate">服务器、认证、网关与容器参数</p>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <button type="button" class="btn btn-secondary w-9 h-9 !px-0" title="刷新" @click="loadConfig(true)">
            <i class="fas fa-rotate text-sm"></i>
          </button>
          <button v-if="portal.hasPerm('PATCH /api/system/config')" type="button" class="btn btn-indigo w-9 h-9 !px-0" title="保存配置" :disabled="saving" @click="saveAll">
            <i v-if="saving" class="fas fa-spinner fa-spin text-sm"></i>
            <i v-else class="fas fa-save text-sm"></i>
          </button>
        </div>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="card-body">
      <div class="empty-state">
        <div class="w-12 h-12 spinner mb-3"></div>
        <p class="text-slate-500">加载中...</p>
      </div>
    </div>

    <form v-else-if="portal.hasPerm('PATCH /api/system/config')" class="card-body" @submit.prevent="saveAll">
      <div class="lg:hidden sticky top-16 z-30 w-full min-w-0 overflow-hidden bg-white pt-4 pb-3 border-b border-slate-100">
        <div class="block w-full max-w-full min-w-0 overflow-x-auto overflow-y-hidden pb-1">
          <div class="tab-group inline-flex w-max max-w-none">
            <button
              v-for="item in configSections"
              :key="item.id"
              type="button"
              class="tab-btn whitespace-nowrap flex-shrink-0"
              :class="activeTab === item.id ? 'tab-btn-active text-indigo-600' : 'tab-btn-inactive'"
              @click="scrollToConfigSection(item.id)"
            >
              <i class="fas" :class="item.icon"></i>
              {{ item.label }}
            </button>
          </div>
        </div>
      </div>

      <div class="grid min-w-0 gap-6 lg:grid-cols-[minmax(0,1fr)_320px] lg:items-start">
        <div class="min-w-0 order-2 lg:order-1 space-y-8">
          <!-- 服务器配置 -->
          <section id="config-server" class="max-w-3xl space-y-4">
            <div class="flex items-center gap-2">
              <span class="card-icon bg-indigo-100 text-indigo-600"><i class="fas fa-server"></i></span>
              <div>
                <h2 class="text-sm font-semibold text-slate-700">Server</h2>
                <p class="text-xs text-slate-400 mt-0.5">端口、目录、上传、跨域与 JWT</p>
              </div>
            </div>
            <div>
              <label class="form-label">监听地址</label>
              <input v-model="server.listenAddr" type="text" placeholder="请输入监听地址" class="input" />
              <p class="mt-1 text-xs text-slate-400">HTTP 服务监听地址，如 :8080 或 127.0.0.1:8080（重启生效）</p>
            </div>
            <div>
              <label class="form-label">基础目录</label>
              <input v-model="server.rootDirectory" type="text" placeholder="请输入基础目录" class="input" />
              <p class="mt-1 text-xs text-slate-400">成员家目录及容器数据的基础目录，默认当前目录（.）</p>
            </div>
            <div>
              <label class="form-label">文件上传大小限制（字节）</label>
              <input v-model.number="server.maxUploadSize" type="number" min="0" placeholder="请输入文件上传大小限制" class="input" />
              <p class="mt-1 text-xs text-slate-400">单次上传的最大文件大小，默认 104857600（100 MB）</p>
            </div>
            <div>
              <label class="form-label">允许的跨域 Origin</label>
              <textarea v-model="allowedOriginsText" rows="3" placeholder="请输入，每行一个" class="input font-mono text-xs"></textarea>
              <p class="mt-1 text-xs text-slate-400">示例：https://example.com、https://*.example.com；支持通配符 *；留空则不限制</p>
            </div>
            <div>
              <label class="form-label">JWT 认证密钥</label>
              <input v-model="server.jwtSecret" type="password" placeholder="留空则保持不变" class="input" autocomplete="new-password" />
              <p class="mt-1 text-xs text-slate-400">用于签名登录令牌，修改后所有用户需要重新登录</p>
            </div>
            <div>
              <label class="form-label">JWT 有效期（秒）</label>
              <input v-model.number="server.jwtExpiration" type="number" min="60" placeholder="请输入 JWT 有效期" class="input" />
              <p class="mt-1 text-xs text-slate-400">登录令牌的有效期，默认 86400（24 小时）</p>
            </div>
            <div class="toggle-row">
              <div>
                <span class="text-sm text-slate-600">Debug 模式</span>
                <p class="text-xs text-slate-400 mt-0.5">开启后输出详细调试日志</p>
              </div>
              <button type="button" class="toggle" :class="{ 'toggle-on': server.debug }" role="switch" :aria-checked="server.debug" @click="server.debug = !server.debug">
                <span class="toggle-thumb" />
              </button>
            </div>
          </section>

          <!-- 代理 Header 登录配置 -->
          <section id="config-tha" class="max-w-3xl space-y-4">
            <div class="flex items-center gap-2">
              <span class="card-icon bg-indigo-100 text-indigo-600"><i class="fas fa-user-shield"></i></span>
              <div>
                <h2 class="text-sm font-semibold text-slate-700">代理 Header 登录</h2>
                <p class="text-xs text-slate-400 mt-0.5">从上游代理 Header 读取用户名</p>
              </div>
            </div>
            <div class="toggle-row">
              <div>
                <span class="text-sm text-slate-600">启用代理 Header 登录</span>
                <p class="text-xs text-slate-400 mt-0.5">开启后使用上游代理传入的用户名 Header</p>
              </div>
              <button type="button" class="toggle" :class="{ 'toggle-on': tha.enabled }" role="switch" :aria-checked="tha.enabled" @click="tha.enabled = !tha.enabled">
                <span class="toggle-thumb" />
              </button>
            </div>
            <div>
              <label class="form-label">用户名 Header</label>
              <input v-model="tha.headerName" type="text" placeholder="请输入 Header 名称" class="input" />
              <p class="mt-1 text-xs text-slate-400">启用时，将使用该 Header 值作为登录用户名；留空则禁用</p>
            </div>
            <div>
              <label class="form-label">可信代理 CIDR</label>
              <textarea v-model="thaTrustedCIDRsText" rows="3" placeholder="请输入代理来源 CIDR，每行一个" class="input font-mono text-xs"></textarea>
              <p class="mt-1 text-xs text-slate-400">示例：127.0.0.1/32、10.0.0.0/8；仅列出的代理来源 IP 允许传入用户名 Header；留空则不限制来源</p>
            </div>
          </section>

          <!-- OIDC 配置 -->
          <section id="config-oidc" class="max-w-3xl space-y-4">
            <div class="flex items-center gap-2">
              <span class="card-icon bg-indigo-100 text-indigo-600"><i class="fas fa-circle-nodes"></i></span>
              <div>
                <h2 class="text-sm font-semibold text-slate-700">OIDC</h2>
                <p class="text-xs text-slate-400 mt-0.5">单点登录 Provider 参数</p>
              </div>
            </div>
            <div class="toggle-row">
              <div>
                <span class="text-sm text-slate-600">启用 OIDC 登录</span>
                <p class="text-xs text-slate-400 mt-0.5">使用 OpenID Connect 进行单点登录</p>
              </div>
              <button type="button" class="toggle" :class="{ 'toggle-on': oidc.enabled }" role="switch" :aria-checked="oidc.enabled" @click="oidc.enabled = !oidc.enabled">
                <span class="toggle-thumb" />
              </button>
            </div>
            <div>
              <label class="form-label">颁发者地址</label>
              <input v-model="oidc.issuerUrl" type="text" placeholder="请输入颁发者地址" class="input" />
              <p class="mt-1 text-xs text-slate-400">示例：https://idp.example.com；用于自动发现 authorization_endpoint、token_endpoint、jwks_uri 等元数据；保存后立即生效</p>
            </div>
            <div>
              <label class="form-label">客户端 ID</label>
              <input v-model="oidc.clientId" type="text" placeholder="请输入客户端 ID" class="input" />
              <p class="mt-1 text-xs text-slate-400">在 OIDC Provider 处注册应用时获得</p>
            </div>
            <div>
              <label class="form-label">客户端密钥</label>
              <input v-model="oidc.clientSecret" type="password" placeholder="留空则保持不变" class="input" autocomplete="new-password" />
              <p class="mt-1 text-xs text-slate-400">在 OIDC Provider 处注册应用时获得</p>
            </div>
            <div>
              <label class="form-label">回调地址</label>
              <input v-model="oidc.redirectUrl" type="text" placeholder="请输入回调地址" class="input" />
              <p class="mt-1 text-xs text-slate-400">示例：https://isrvd.example.com/api/account/oidc/callback；开发环境可留空自动生成，生产环境建议填写固定 HTTPS 回调地址</p>
            </div>
            <div>
              <label class="form-label">用户名字段</label>
              <input v-model="oidc.usernameClaim" type="text" placeholder="请输入用户名字段" class="input" />
              <p class="mt-1 text-xs text-slate-400">OIDC 用户信息中作为用户名的字段，默认 sub；该字段的值必须与 members.username 完全一致，用户不存在时登录失败</p>
            </div>
            <div>
              <label class="form-label">授权范围</label>
              <input v-model="oidcScopes" type="text" placeholder="请输入授权范围" class="input" />
              <p class="mt-1 text-xs text-slate-400">示例：openid profile email；以空格分隔，系统会自动确保包含 openid</p>
            </div>
            <div>
              <label class="form-label">登录按钮名称</label>
              <input v-model="oidc.loginLabel" type="text" placeholder="请输入登录按钮名称" class="input" />
              <p class="mt-1 text-xs text-slate-400">自定义 OIDC 登录按钮显示名称；留空则使用默认文案「使用 OIDC 登录」</p>
            </div>
          </section>

          <!-- Passkey 配置 -->
          <section id="config-passkey" class="max-w-3xl space-y-4">
            <div class="flex items-center gap-2">
              <span class="card-icon bg-indigo-100 text-indigo-600"><i class="fas fa-fingerprint"></i></span>
              <div>
                <h2 class="text-sm font-semibold text-slate-700">Passkey</h2>
                <p class="text-xs text-slate-400 mt-0.5">WebAuthn/FIDO2 登录</p>
              </div>
            </div>
            <div class="toggle-row">
              <div>
                <span class="text-sm text-slate-600">启用 Passkey 登录</span>
                <p class="text-xs text-slate-400 mt-0.5">使用 WebAuthn/FIDO2 进行无密码登录</p>
              </div>
              <button type="button" class="toggle" :class="{ 'toggle-on': passkey.enabled }" role="switch" :aria-checked="passkey.enabled" @click="passkey.enabled = !passkey.enabled">
                <span class="toggle-thumb" />
              </button>
            </div>
            <div>
              <label class="form-label">Relying Party 名称</label>
              <input v-model="passkey.rpName" type="text" placeholder="请输入 RP 名称" class="input" />
              <p class="mt-1 text-xs text-slate-400">显示在 Passkey 注册/登录界面上的名称，如 iSrvd</p>
            </div>
            <div>
              <label class="form-label">Relying Party ID</label>
              <input v-model="passkey.rpId" type="text" placeholder="纯域名，如 example.com" class="input" />
              <p class="mt-1 text-xs text-slate-400">必须是纯域名，不含 https:// 前缀，如 example.com；填写带 scheme 的地址将自动提取域名部分</p>
            </div>
            <div>
              <label class="form-label">允许的 Origin</label>
              <textarea v-model="passkeyOriginsText" rows="3" placeholder="请输入允许的 Origin，每行一个" class="input font-mono text-xs"></textarea>
              <p class="mt-1 text-xs text-slate-400">示例：https://example.com、https://*.example.com；必须与访问地址一致</p>
            </div>
            <div>
              <label class="form-label">超时时间（毫秒）</label>
              <input v-model.number="passkey.timeout" type="number" min="1000" placeholder="请输入超时时间" class="input" />
              <p class="mt-1 text-xs text-slate-400">Passkey 操作的超时时间，默认 60000（60 秒）</p>
            </div>
          </section>


          <!-- Agent 配置 -->
          <section id="config-agent" class="max-w-3xl space-y-4">
            <div class="flex items-center gap-2">
              <span class="card-icon bg-indigo-100 text-indigo-600"><i class="fas fa-robot"></i></span>
              <div>
                <h2 class="text-sm font-semibold text-slate-700">Agent</h2>
                <p class="text-xs text-slate-400 mt-0.5">LLM 代理与模型改写</p>
              </div>
            </div>
            <div>
              <label class="form-label">模型名称</label>
              <input v-model="agent.model" type="text" placeholder="请输入模型名称" class="input" />
              <p class="mt-1 text-xs text-slate-400">代理转发时强制改写请求体中的 model 字段，留空则不改写</p>
            </div>
            <div>
              <label class="form-label">基础地址</label>
              <input v-model="agent.baseUrl" type="text" placeholder="请输入基础地址" class="input" />
              <p class="mt-1 text-xs text-slate-400">示例：https://api.openai.com/v1；OpenAI 兼容的 LLM API 基础地址，留空则禁用代理</p>
            </div>
            <div>
              <label class="form-label">API 密钥</label>
              <input v-model="agent.apiKey" type="password" placeholder="留空则保持不变" class="input" autocomplete="new-password" />
              <p class="mt-1 text-xs text-slate-400">代理转发时以 Bearer 形式注入 Authorization 请求头</p>
            </div>
          </section>


          <!-- APISIX 配置 -->
          <section id="config-apisix" class="max-w-3xl space-y-4">
            <div class="flex items-center gap-2">
              <span class="card-icon bg-indigo-100 text-indigo-600"><i class="fas fa-route"></i></span>
              <div>
                <h2 class="text-sm font-semibold text-slate-700">APISIX</h2>
                <p class="text-xs text-slate-400 mt-0.5">Admin API 连接参数</p>
              </div>
            </div>
            <div>
              <label class="form-label">Admin URL</label>
              <input v-model="apisix.adminUrl" type="text" placeholder="请输入 Admin URL" class="input" />
              <p class="mt-1 text-xs text-slate-400">APISIX Admin API 地址，默认 http://127.0.0.1:9180</p>
            </div>
            <div>
              <label class="form-label">Admin Key</label>
              <input v-model="apisix.adminKey" type="password" placeholder="留空则保持不变" class="input" autocomplete="new-password" />
              <p class="mt-1 text-xs text-slate-400">访问 APISIX Admin API 的密钥</p>
            </div>
          </section>

          <!-- Caddy 配置 -->
          <section id="config-caddy" class="max-w-3xl space-y-4">
            <div class="flex items-center gap-2">
              <span class="card-icon bg-indigo-100 text-indigo-600"><i class="fas fa-globe"></i></span>
              <div>
                <h2 class="text-sm font-semibold text-slate-700">Caddy</h2>
                <p class="text-xs text-slate-400 mt-0.5">Admin API 连接参数</p>
              </div>
            </div>
            <div>
              <label class="form-label">Admin URL</label>
              <input v-model="caddy.adminUrl" type="text" placeholder="请输入 Admin URL" class="input" />
              <p class="text-xs text-slate-400 mt-1">Caddy Admin API 地址，默认 http://127.0.0.1:2019</p>
            </div>
          </section>

          <!-- Docker 配置 -->
          <section id="config-docker" class="max-w-3xl space-y-4">
            <div class="flex items-center gap-2">
              <span class="card-icon bg-indigo-100 text-indigo-600"><i class="fas fa-boxes-stacked"></i></span>
              <div>
                <h2 class="text-sm font-semibold text-slate-700">Docker</h2>
                <p class="text-xs text-slate-400 mt-0.5">引擎连接与容器根目录</p>
              </div>
            </div>
            <div>
              <label class="form-label">Docker Host</label>
              <input v-model="docker.host" type="text" placeholder="请输入 Docker Host" class="input" />
              <p class="mt-1 text-xs text-slate-400">示例：unix:///var/run/docker.sock 或 tcp://host:2375；留空则使用环境变量 DOCKER_HOST</p>
            </div>
            <div>
              <label class="form-label">容器数据根目录</label>
              <input v-model="docker.containerRoot" type="text" placeholder="请输入容器数据根目录" class="input" />
              <p class="mt-1 text-xs text-slate-400">用于存放容器数据卷的基础目录（相对于基础目录），默认 containers</p>
            </div>
          </section>

          <!-- Monitor 配置 -->
          <section id="config-monitor" class="max-w-3xl space-y-4">
            <div class="flex items-center gap-2">
              <span class="card-icon bg-indigo-100 text-indigo-600"><i class="fas fa-chart-line"></i></span>
              <div>
                <h2 class="text-sm font-semibold text-slate-700">Monitor</h2>
                <p class="text-xs text-slate-400 mt-0.5">系统与容器监控采集</p>
              </div>
            </div>
            <div>
              <label class="form-label">监控采集间隔</label>
              <select v-model.number="monitor.interval" class="input">
                <option :value="0">禁用</option>
                <option :value="5">5 秒</option>
                <option :value="15">15 秒</option>
                <option :value="30">30 秒</option>
                <option :value="60">60 秒</option>
              </select>
              <p class="mt-1 text-xs text-slate-400">系统与容器监控数据的采集频率，禁用后不再写入监控文件</p>
            </div>
          </section>

          <!-- 应用市场配置 -->
          <section id="config-marketplace" class="max-w-3xl space-y-4">
            <div class="flex items-center gap-2">
              <span class="card-icon bg-indigo-100 text-indigo-600"><i class="fas fa-store"></i></span>
              <div>
                <h2 class="text-sm font-semibold text-slate-700">应用市场</h2>
                <p class="text-xs text-slate-400 mt-0.5">市场 iframe 站点地址</p>
              </div>
            </div>
            <div>
              <label class="form-label">站点 URL</label>
              <input v-model="marketplace.url" type="text" placeholder="请输入应用市场 URL" class="input" />
              <p class="mt-1 text-xs text-slate-400">应用市场页面以 iframe 方式嵌入，并通过 postMessage 协议接收安装事件</p>
            </div>
          </section>


          <!-- Links 配置 -->
          <section id="config-links" class="max-w-3xl space-y-4">
            <div class="flex items-center gap-2">
              <span class="card-icon bg-indigo-100 text-indigo-600"><i class="fas fa-link"></i></span>
              <div>
                <h2 class="text-sm font-semibold text-slate-700">导航链接</h2>
                <p class="text-xs text-slate-400 mt-0.5">顶部工具栏外部链接</p>
              </div>
            </div>
            <!-- 列标题（仅在有数据时显示） -->
            <div v-if="links.length" class="hidden sm:grid sm:grid-cols-[1fr_2fr_1.2fr_auto] gap-3 px-0.5">
              <span class="text-xs font-medium text-slate-500">名称</span>
              <span class="text-xs font-medium text-slate-500">URL</span>
              <span class="text-xs font-medium text-slate-500">图标</span>
              <span></span>
            </div>
            <div v-for="(link, index) in links" :key="index" class="grid grid-cols-1 sm:grid-cols-[1fr_2fr_1.2fr_auto] gap-3 items-center">
              <input v-model="link.label" type="text" placeholder="请输入名称" class="input" />
              <input v-model="link.url" type="text" placeholder="请输入链接 URL" class="input" />
              <!-- 图标选择器 -->
              <IconSelect v-model="link.icon" />
              <button v-if="portal.hasPerm('PATCH /api/system/config')" type="button" class="btn-icon btn-icon-red w-11 h-11" @click="removeLink(index)">
                <i class="fas fa-trash-can text-sm"></i>
              </button>
            </div>
            <button type="button" class="btn-add-row" @click="addLink()">
              <i class="fas fa-plus text-xs"></i>添加链接
            </button>
          </section>

          <!-- 提示信息 -->
          <div class="max-w-3xl mt-6 pt-4 border-t border-slate-200">
            <p class="text-xs text-slate-400 flex items-start gap-1">
              <i class="fas fa-circle-info mt-0.5 flex-shrink-0"></i>
              <span>保存后立即写入配置文件，监听地址变更需重启服务生效。</span>
            </p>
          </div>
        </div>

        <aside class="order-1 min-w-0 overflow-hidden lg:order-2 lg:border-l lg:border-slate-200 lg:pl-6 lg:sticky lg:top-24 lg:self-start lg:max-h-[calc(100vh-7rem)] lg:overflow-y-auto">
          <div class="hidden lg:block space-y-3">
            <h2 class="section-title">配置块</h2>
            <button
              v-for="item in configSections"
              :key="item.id"
              type="button"
              class="option-card w-full"
              :class="activeTab === item.id ? 'border-indigo-200 bg-indigo-50 text-indigo-700' : 'option-card-inactive'"
              @click="scrollToConfigSection(item.id)"
            >
              <div class="flex items-center gap-3 min-w-0">
                <span
                  class="option-card-icon flex-shrink-0"
                  :class="activeTab === item.id ? 'bg-white text-indigo-600' : 'bg-slate-100 text-slate-500'"
                >
                  <i class="fas" :class="item.icon"></i>
                </span>
                <span class="min-w-0">
                  <span class="font-medium text-sm truncate block">{{ item.label }}</span>
                  <span class="text-xs opacity-75 truncate block mt-0.5">{{ item.description }}</span>
                </span>
              </div>
            </button>
          </div>
        </aside>
      </div>
    </form>

    <div v-else class="card-body">
      <div class="empty-state">
        <div class="empty-state-icon">
          <i class="fas fa-lock text-4xl text-slate-300"></i>
        </div>
        <p class="text-slate-600 font-medium mb-1">无权限修改系统配置</p>
        <p class="text-sm text-slate-400">请联系管理员调整账号权限</p>
      </div>
    </div>
  </div>
</template>