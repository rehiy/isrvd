<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { useConfigStore } from '@/stores'

import ToggleCard from '@/component/toggle-card.vue'

@Component({ components: { ToggleCard } })
class ConfigAuth extends Vue {
    config = useConfigStore()
}

export default toNative(ConfigAuth)
</script>

<template>
  <div class="max-w-4xl space-y-6">
    <!-- 密码登录 -->
    <section class="space-y-4">
      <div class="flex items-center gap-2">
        <span class="card-icon bg-indigo-100 text-indigo-600"><i class="fas fa-key"></i></span>
        <div>
          <h2 class="text-sm font-semibold text-slate-700">密码登录</h2>
          <p class="text-xs text-slate-400 mt-0.5">密码登录开关</p>
        </div>
      </div>
      <ToggleCard v-model="config.draft.password.disabled" label="禁用密码登录" desc="禁用后仅允许 Passkey、OIDC 或代理 Header 登录；请确保已配置至少一种可用的替代方式" />
      <div>
        <label class="form-label">密码最小长度</label>
        <input v-model.number="config.draft.password.minLength" type="number" min="1" max="128" class="input" placeholder="请输入密码最小长度" />
        <p class="mt-1 text-xs text-slate-400">创建成员和修改密码时的最小字符数，默认 6</p>
      </div>
    </section>

    <!-- Passkey -->
    <section class="border-t border-slate-200 pt-6 space-y-4">
      <div class="flex items-center gap-2">
        <span class="card-icon bg-indigo-100 text-indigo-600"><i class="fas fa-fingerprint"></i></span>
        <div>
          <h2 class="text-sm font-semibold text-slate-700">Passkey</h2>
          <p class="text-xs text-slate-400 mt-0.5">WebAuthn/FIDO2 登录</p>
        </div>
      </div>
      <ToggleCard v-model="config.draft.passkey.enabled" label="启用 Passkey 登录" desc="使用 WebAuthn/FIDO2 进行无密码登录" />
      <div>
        <label class="form-label">Relying Party 名称</label>
        <input v-model="config.draft.passkey.rpName" type="text" placeholder="请输入 RP 名称" class="input" />
        <p class="mt-1 text-xs text-slate-400">显示在 Passkey 注册/登录界面上的名称，如 iSrvd</p>
      </div>
      <div>
        <label class="form-label">Relying Party ID</label>
        <input v-model="config.draft.passkey.rpId" type="text" placeholder="纯域名，如 example.com" class="input" />
        <p class="mt-1 text-xs text-slate-400">必须是纯域名，不含 https:// 前缀，如 example.com；填写带 scheme 的地址将自动提取域名部分</p>
      </div>
      <div>
        <label class="form-label">允许的 Origin</label>
        <textarea v-model="config.passkeyOriginsText" rows="3" placeholder="请输入允许的 Origin，每行一个" class="input font-mono text-xs"></textarea>
        <p class="mt-1 text-xs text-slate-400">示例：https://example.com、https://*.example.com；必须与访问地址一致</p>
      </div>
      <div>
        <label class="form-label">超时时间（毫秒）</label>
        <input v-model.number="config.draft.passkey.timeout" type="number" min="1000" placeholder="请输入超时时间" class="input" />
        <p class="mt-1 text-xs text-slate-400">Passkey 操作的超时时间，默认 60000（60 秒）</p>
      </div>
    </section>

    <!-- OIDC -->
    <section class="border-t border-slate-200 pt-6 space-y-4">
      <div class="flex items-center gap-2">
        <span class="card-icon bg-indigo-100 text-indigo-600"><i class="fas fa-circle-nodes"></i></span>
        <div>
          <h2 class="text-sm font-semibold text-slate-700">OIDC</h2>
          <p class="text-xs text-slate-400 mt-0.5">单点登录 Provider 参数</p>
        </div>
      </div>
      <ToggleCard v-model="config.draft.oidc.enabled" label="启用 OIDC 登录" desc="使用 OpenID Connect 进行单点登录" />
      <div>
        <label class="form-label">颁发者地址</label>
        <input v-model="config.draft.oidc.issuerUrl" type="text" placeholder="请输入颁发者地址" class="input" />
        <p class="mt-1 text-xs text-slate-400">示例：https://idp.example.com；用于自动发现 authorization_endpoint、token_endpoint、jwks_uri 等元数据；保存后立即生效</p>
      </div>
      <div>
        <label class="form-label">客户端 ID</label>
        <input v-model="config.draft.oidc.clientId" type="text" placeholder="请输入客户端 ID" class="input" />
        <p class="mt-1 text-xs text-slate-400">在 OIDC Provider 处注册应用时获得</p>
      </div>
      <div>
        <label class="form-label">客户端密钥</label>
        <input v-model="config.draft.oidc.clientSecret" type="password" placeholder="留空则保持不变" class="input" autocomplete="new-password" />
        <p class="mt-1 text-xs text-slate-400">在 OIDC Provider 处注册应用时获得</p>
      </div>
      <div>
        <label class="form-label">回调地址</label>
        <input v-model="config.draft.oidc.redirectUrl" type="text" placeholder="请输入回调地址" class="input" />
        <p class="mt-1 text-xs text-slate-400">示例：https://isrvd.example.com/api/account/oidc/callback；开发环境可留空自动生成，生产环境建议填写固定 HTTPS 回调地址</p>
      </div>
      <div>
        <label class="form-label">用户名字段</label>
        <input v-model="config.draft.oidc.usernameClaim" type="text" placeholder="请输入用户名字段" class="input" />
        <p class="mt-1 text-xs text-slate-400">OIDC 用户信息中作为用户名的字段，默认 sub；该字段的值必须与 members.username 完全一致，用户不存在时登录失败</p>
      </div>
      <div>
        <label class="form-label">授权范围</label>
        <input v-model="config.oidcScopesText" type="text" placeholder="请输入授权范围" class="input" />
        <p class="mt-1 text-xs text-slate-400">示例：openid profile email；以空格分隔，系统会自动确保包含 openid</p>
      </div>
      <div>
        <label class="form-label">登录按钮名称</label>
        <input v-model="config.draft.oidc.loginLabel" type="text" placeholder="请输入登录按钮名称" class="input" />
        <p class="mt-1 text-xs text-slate-400">自定义 OIDC 登录按钮显示名称；留空则使用默认文案「使用 OIDC 登录」</p>
      </div>
    </section>

    <!-- 代理 Header 登录 -->
    <section class="border-t border-slate-200 pt-6 space-y-4">
      <div class="flex items-center gap-2">
        <span class="card-icon bg-indigo-100 text-indigo-600"><i class="fas fa-user-shield"></i></span>
        <div>
          <h2 class="text-sm font-semibold text-slate-700">代理 Header 登录</h2>
          <p class="text-xs text-slate-400 mt-0.5">从上游代理 Header 读取用户名</p>
        </div>
      </div>
      <ToggleCard v-model="config.draft.tha.enabled" label="启用代理 Header 登录" desc="开启后使用上游代理传入的用户名 Header" />
      <div>
        <label class="form-label">用户名 Header</label>
        <input v-model="config.draft.tha.headerName" type="text" placeholder="请输入 Header 名称" class="input" />
        <p class="mt-1 text-xs text-slate-400">启用时，将使用该 Header 值作为登录用户名；留空则禁用</p>
      </div>
      <div>
        <label class="form-label">可信代理 CIDR</label>
        <textarea v-model="config.thaTrustedCIDRsText" rows="3" placeholder="请输入代理来源 CIDR，每行一个" class="input font-mono text-xs"></textarea>
        <p class="mt-1 text-xs text-slate-400">示例：127.0.0.1/32、10.0.0.0/8；仅列出的代理来源 IP 允许传入用户名 Header；留空则不限制来源</p>
      </div>
    </section>
  </div>
</template>
