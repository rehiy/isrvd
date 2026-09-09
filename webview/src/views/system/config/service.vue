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
    <div class="flex items-center gap-2">
      <span class="card-icon bg-indigo-100 text-indigo-600"><i class="fas fa-server"></i></span>
      <div>
        <h2 class="text-sm font-semibold text-slate-700">基础服务</h2>
        <p class="text-xs text-slate-400 mt-0.5">端口、目录、上传、跨域与 JWT</p>
      </div>
    </div>

    <div>
      <label class="form-label">监听地址</label>
      <input v-model="config.draft.server.listenAddr" type="text" placeholder="请输入监听地址" class="input" />
      <p class="mt-1 text-xs text-slate-400">HTTP 服务监听地址，如 :8080 或 127.0.0.1:8080（重启生效）</p>
    </div>

    <div>
      <label class="form-label">基础目录</label>
      <input v-model="config.draft.server.rootDirectory" type="text" placeholder="请输入基础目录" class="input" />
      <p class="mt-1 text-xs text-slate-400">成员家目录及容器数据的基础目录，默认当前目录（.）</p>
    </div>

    <div>
      <label class="form-label">文件上传大小限制（字节）</label>
      <input v-model.number="config.draft.server.maxUploadSize" type="number" min="0" placeholder="请输入文件上传大小限制" class="input" />
      <p class="mt-1 text-xs text-slate-400">单次上传的最大文件大小，默认 104857600（100 MB）</p>
    </div>

    <div>
      <label class="form-label">允许的跨域 Origin</label>
      <textarea v-model="config.allowedOriginsText" rows="3" placeholder="请输入，每行一个" class="input font-mono text-xs"></textarea>
      <p class="mt-1 text-xs text-slate-400">示例：https://example.com、https://*.example.com；支持通配符 *；留空则不限制</p>
    </div>

    <div>
      <label class="form-label">JWT 认证密钥</label>
      <input v-model="config.draft.server.jwtSecret" type="password" placeholder="留空则保持不变" class="input" autocomplete="new-password" />
      <p class="mt-1 text-xs text-slate-400">用于签名登录令牌，修改后所有用户需要重新登录</p>
    </div>

    <div>
      <label class="form-label">JWT 有效期（秒）</label>
      <input v-model.number="config.draft.server.jwtExpiration" type="number" min="60" placeholder="请输入 JWT 有效期" class="input" />
      <p class="mt-1 text-xs text-slate-400">登录令牌的有效期，默认 86400（24 小时）</p>
    </div>

    <ToggleCard v-model="config.draft.server.openapi" label="API 文档" desc="开启后对外提供 /openapi/ 接口文档页（含全部接口结构，建议生产环境关闭）" />
    <ToggleCard v-model="config.draft.server.debug" label="Debug 模式" desc="开启后输出详细调试日志" />
  </section>
</template>
