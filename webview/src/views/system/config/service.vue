<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { useConfigStore } from '@/stores'

import ToggleCard from '@/component/toggle-card.vue'

@Component({ components: { ToggleCard } })
class ConfigService extends Vue {
    config = useConfigStore()
}

export default toNative(ConfigService)
</script>

<template>
  <section class="max-w-4xl space-y-4">
    <div class="config-section-heading">
      <div>
        <h2 class="config-section-title">{{ $t('基础服务') }}</h2>
        <p class="config-section-description">{{ $t('端口、目录、上传、跨域与 JWT') }}</p>
      </div>
    </div>
    <div>
      <label for="server-mode" class="form-label">{{ $t('运行模式') }}</label>
      <select id="server-mode" v-model="config.draft.server.mode" class="input">
        <option value="server">{{ $t('单机模式') }}</option>
        <option value="center">{{ $t('中控模式') }}</option>
        <option value="agent">{{ $t('受控模式') }}</option>
      </select>
      <p class="mt-1 text-xs text-slate-400">{{ $t('运行模式与连接参数保存到 config.yml，重启服务后生效') }}</p>
    </div>

    <div v-if="config.draft.server.mode === 'agent'" class="space-y-4">
      <div>
        <label for="server-center-url" class="form-label">{{ $t('服务器地址') }}</label>
        <input id="server-center-url" v-model="config.draft.server.centerUrl" type="url" required placeholder="https://center.example.com" class="input" autocomplete="off" />
        <p class="mt-1 text-xs text-slate-400">{{ $t('中控服务器的 HTTP(S) 地址，不含路径或查询参数') }}</p>
      </div>
      <div>
        <label for="server-node-token" class="form-label">{{ $t('连接 token') }}</label>
        <input id="server-node-token" v-model="config.draft.server.nodeToken" type="password" :placeholder="$t('留空则保持不变')" class="input" autocomplete="new-password" spellcheck="false" />
        <p class="mt-1 text-xs text-slate-400">{{ $t('直接填写中控生成的 token；同一 token 仅允许一个客户端在线') }}</p>
      </div>
      <p class="text-sm text-amber-600">{{ $t('受控模式不开放对外端口；请确认地址与 token 正确，重启后只能通过中控访问') }}</p>
    </div>

    <div>
      <label class="form-label">{{ $t('监听地址') }}</label>
      <input v-model="config.draft.server.listenAddr" type="text" :placeholder="$t('请输入监听地址')" class="input" />
      <p class="mt-1 text-xs text-slate-400">{{ $t('HTTP 服务监听地址，如 :8080 或 127.0.0.1:8080（重启生效）') }}</p>
    </div>

    <div>
      <label class="form-label">{{ $t('基础目录') }}</label>
      <input v-model="config.draft.server.rootDirectory" type="text" :placeholder="$t('请输入基础目录')" class="input" />
      <p class="mt-1 text-xs text-slate-400">{{ $t('成员家目录及容器数据的基础目录，默认当前目录（.）') }}</p>
    </div>

    <div>
      <label class="form-label">{{ $t('文件上传大小限制（字节）') }}</label>
      <input v-model.number="config.draft.server.maxUploadSize" type="number" min="0" :placeholder="$t('请输入文件上传大小限制')" class="input" />
      <p class="mt-1 text-xs text-slate-400">{{ $t('单次上传的最大文件大小，默认 104857600（100 MB）') }}</p>
    </div>

    <div>
      <label class="form-label">{{ $t('允许的跨域 Origin') }}</label>
      <textarea v-model="config.allowedOriginsText" rows="3" :placeholder="$t('请输入，每行一个')" class="input font-mono text-xs"></textarea>
      <p class="mt-1 text-xs text-slate-400">{{ $t('示例：https://example.com、https://*.example.com；支持通配符 *；留空则不限制') }}</p>
    </div>

    <div>
      <label class="form-label">{{ $t('JWT 认证密钥') }}</label>
      <input v-model="config.draft.server.jwtSecret" type="password" :placeholder="$t('留空则保持不变')" class="input" autocomplete="new-password" />
      <p class="mt-1 text-xs text-slate-400">{{ $t('用于签名登录令牌，修改后所有用户需要重新登录') }}</p>
    </div>

    <div>
      <label class="form-label">{{ $t('JWT 有效期（秒）') }}</label>
      <input v-model.number="config.draft.server.jwtExpiration" type="number" min="60" :placeholder="$t('请输入 JWT 有效期')" class="input" />
      <p class="mt-1 text-xs text-slate-400">{{ $t('登录令牌的有效期，默认 86400（24 小时）') }}</p>
    </div>

    <ToggleCard v-model="config.draft.server.openapi" :label="$t('API 文档')" :desc="$t('开启后对外提供 /openapi/ 接口文档页（含全部接口结构，建议生产环境关闭）')" />
    <ToggleCard v-model="config.draft.server.debug" :label="$t('Debug 模式')" :desc="$t('开启后输出详细调试日志')" />
  </section>
</template>
