<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { useConfigStore } from '@/stores'

import type { WebhookConfig } from '@/service/types'

import ToggleCard from '@/component/toggle-card.vue'

interface WebhookTemplatePreset {
    value: string
    label: string
    template: string
}

interface WebhookTemplateGroup {
    label: string
    presets: WebhookTemplatePreset[]
}

@Component({ components: { ToggleCard } })
class ConfigAlert extends Vue {
    config = useConfigStore()
    selectedWebhookTemplates = new WeakMap<WebhookConfig, string>()

    // Go text/template 变量说明；定义为变量后插值输出，避免与 Vue 的 {{ }} 语法冲突
    templateHint = '支持 Go 模板变量；在 JSON 字段中使用 {{.Title | json}} 可安全处理引号和换行。'

    webhookTemplateGroups: WebhookTemplateGroup[] = [
        {
            label: '国内协作平台',
            presets: [
                {
                    value: 'dingtalk',
                    label: '钉钉机器人',
                    template: '{"msgtype":"text","text":{"content":{{printf "isrvd 告警\\n%s\\n%s" .Title .Message | json}}}}',
                },
                {
                    value: 'feishu',
                    label: '飞书机器人',
                    template: '{"msg_type":"text","content":{"text":{{printf "isrvd 告警\\n%s\\n%s" .Title .Message | json}}}}',
                },
                {
                    value: 'wecom',
                    label: '企业微信机器人',
                    template: '{"msgtype":"text","text":{"content":{{printf "isrvd 告警\\n%s\\n%s" .Title .Message | json}}}}',
                },
            ],
        },
        {
            label: '海外协作与聊天',
            presets: [
                {
                    value: 'slack',
                    label: 'Slack',
                    template: '{"text":{{printf "isrvd 告警：%s\\n%s" .Title .Message | json}}}',
                },
                {
                    value: 'discord',
                    label: 'Discord',
                    template: '{"content":{{printf "isrvd 告警：%s\\n%s" .Title .Message | json}}}',
                },
                {
                    value: 'teams',
                    label: 'Microsoft Teams',
                    template: '{"text":{{printf "isrvd 告警：%s\\n%s" .Title .Message | json}}}',
                },
                {
                    value: 'google-chat',
                    label: 'Google Chat',
                    template: '{"text":{{printf "isrvd 告警：%s\\n%s" .Title .Message | json}}}',
                },
                {
                    value: 'telegram',
                    label: 'Telegram Bot（sendMessage + CHAT_ID）',
                    template: '{"chat_id":"<CHAT_ID>","text":{{printf "isrvd 告警：%s\\n%s" .Title .Message | json}}}',
                },
            ],
        },
    ]

    addWebhook() {
        this.config.draft.notify.webhooks.push({ name: '', url: '', template: '' })
    }

    removeWebhook(index: number) {
        this.config.draft.notify.webhooks.splice(index, 1)
    }

    addRule() {
        this.config.draft.notify.rules.push({ metric: 'cpu', threshold: 80, duration: 3 })
    }

    removeRule(index: number) {
        this.config.draft.notify.rules.splice(index, 1)
    }

    webhookTemplateValue(hook: WebhookConfig) {
        const selected = this.selectedWebhookTemplates.get(hook)
        if (selected) return selected
        if (!hook.template.trim()) return 'generic'
        const matches = this.webhookTemplateGroups.flatMap(group => group.presets).filter(item => item.template === hook.template)
        return matches.length === 1 ? matches[0].value : 'custom'
    }

    applyWebhookTemplate(hook: WebhookConfig, event: Event) {
        const select = event.target as HTMLSelectElement
        if (select.value === 'generic') {
            this.selectedWebhookTemplates.set(hook, 'generic')
            hook.template = ''
            if (!hook.name.trim()) hook.name = '通用 Webhook'
            return
        }
        const preset = this.webhookTemplateGroups.flatMap(group => group.presets).find(item => item.value === select.value)
        if (preset) {
            this.selectedWebhookTemplates.set(hook, preset.value)
            hook.template = preset.template
            if (!hook.name.trim()) hook.name = preset.label.replace(/（.*）$/, '')
        }
    }
}

export default toNative(ConfigAlert)
</script>

<template>
  <div class="max-w-4xl space-y-6">
    <!-- 监控采集 -->
    <section class="space-y-4">
      <div class="config-section-heading">
        <div>
          <h2 class="config-section-title">监控日志</h2>
          <p class="config-section-description">系统与容器监控采集</p>
        </div>
      </div>
      <div>
        <label class="form-label">监控采集间隔</label>
        <select v-model.number="config.draft.monitor.interval" class="input">
          <option :value="0">禁用</option>
          <option :value="5">5 秒</option>
          <option :value="15">15 秒</option>
          <option :value="30">30 秒</option>
          <option :value="60">60 秒</option>
        </select>
        <p class="mt-1 text-xs text-slate-400">系统与容器监控数据的采集频率，禁用后停止历史采集与资源阈值告警；下方应用故障告警仍可独立运行</p>
      </div>
    </section>

    <!-- Webhook 通道 -->
    <section class="space-y-4">
      <div class="config-section-heading">
        <div>
          <h2 class="config-section-title">通知通道</h2>
          <p class="config-section-description">配置 Webhook 地址与消息格式，资源和应用故障告警均推送到全部通道</p>
        </div>
      </div>
      <div v-if="config.draft.notify.webhooks.length === 0" class="empty-note">暂无通知通道，点击下方按钮开始配置</div>
      <div v-else class="space-y-4">
        <div v-for="(hook, index) in config.draft.notify.webhooks" :key="index" class="panel-frame">
          <div class="card-body space-y-4">
            <div class="flex items-center justify-between">
              <span class="text-sm font-semibold text-slate-700">通道 {{ index + 1 }}</span>
              <button type="button" class="btn-icon btn-icon-red" title="删除通道" @click="removeWebhook(index)">
                <i class="fas fa-trash-can text-xs"></i>
              </button>
            </div>
            <div>
              <label class="form-label">通道名称</label>
              <input v-model="hook.name" type="text" required placeholder="如：钉钉告警" class="input" />
            </div>
            <div>
              <label class="form-label">Webhook 地址</label>
              <input v-model="hook.url" type="url" required pattern="https?://.*" placeholder="粘贴渠道提供的 HTTP(S) Webhook 地址" class="input" />
            </div>
            <div>
              <label class="form-label">消息渠道模板</label>
              <select class="input" :value="webhookTemplateValue(hook)" @change="applyWebhookTemplate(hook, $event)">
                <option value="custom" disabled>已配置请求体模板</option>
                <option value="generic">使用标准 JSON（无需模板）</option>
                <optgroup v-for="group in webhookTemplateGroups" :key="group.label" :label="group.label">
                  <option v-for="preset in group.presets" :key="preset.value" :value="preset.value">{{ preset.label }}</option>
                </optgroup>
              </select>
              <p class="mt-1 text-xs text-slate-400">选择后会覆盖下方请求体；带占位符的模板需按提示补充对应参数。</p>
            </div>
            <div>
              <label class="form-label">请求体模板（JSON）</label>
              <textarea v-model="hook.template" rows="4" class="input font-mono text-xs" placeholder="留空则发送 isrvd 标准事件 JSON"></textarea>
              <p class="mt-1 text-xs text-slate-400">{{ templateHint }}</p>
            </div>
          </div>
        </div>
      </div>
      <button type="button" class="btn-add-row" @click="addWebhook">
        <i class="fas fa-plus text-xs"></i>添加通道
      </button>
    </section>

    <!-- 应用故障告警 -->
    <section class="space-y-4">
      <div class="config-section-heading">
        <div>
          <h2 class="config-section-title">应用故障告警</h2>
          <p class="config-section-description">独立检测容器与网关证书，计划任务失败时立即通知</p>
        </div>
      </div>
      <ToggleCard v-model="config.draft.notify.events.containerEnabled" label="容器故障" desc="异常退出、内存不足、健康检查失败、持续重启及频繁重启；恢复后通知">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label class="form-label">频繁重启次数</label>
            <input v-model.number="config.draft.notify.events.restartThreshold" type="number" min="1" max="10000" required class="input" />
          </div>
          <div>
            <label class="form-label">重启统计窗口（秒）</label>
            <input v-model.number="config.draft.notify.events.restartWindow" type="number" min="30" max="86400" required class="input" />
          </div>
        </div>
        <p class="mt-3 text-xs text-slate-400">每 30 秒检测一次，同一故障不重复发送；正常退出及常见停止信号不告警。为容器添加 isrvd.notify.ignore=true 标签可排除检测。</p>
      </ToggleCard>
      <ToggleCard v-model="config.draft.notify.events.cronEnabled" label="计划任务失败" desc="每次失败执行发送一次通知，包括超时；详细原因在执行历史中查看，通知不含脚本或输出" />
      <ToggleCard v-model="config.draft.notify.events.certificateEnabled" label="证书到期" desc="检查 APISIX 与 Caddy 的可读取证书，到期前提醒，续期恢复后通知">
        <div>
          <label class="form-label">提前提醒天数</label>
          <input v-model.number="config.draft.notify.events.certificateDays" type="number" min="1" max="3650" required class="input" />
        </div>
        <p class="mt-3 text-xs text-slate-400">每小时检查一次，同一证书每天最多提醒一次；检测到过期时升级告警级别。Caddy 文件与缓存证书需要面板进程能够读取。</p>
      </ToggleCard>
    </section>

    <!-- 资源告警规则 -->
    <section class="space-y-4">
      <div class="config-section-heading">
        <div>
          <h2 class="config-section-title">资源告警规则</h2>
          <p class="config-section-description">资源阈值与持续次数，命中后触发推送</p>
        </div>
      </div>
      <div v-if="config.draft.notify.rules.length === 0" class="empty-note">暂无规则，点击下方按钮添加</div>
      <div v-else class="space-y-4">
        <div v-for="(rule, index) in config.draft.notify.rules" :key="index" class="panel-frame">
          <div class="card-body space-y-4">
            <div class="flex items-center justify-between">
              <span class="text-sm font-semibold text-slate-700">规则 {{ index + 1 }}</span>
              <button type="button" class="btn-icon btn-icon-red" title="删除规则" @click="removeRule(index)">
                <i class="fas fa-trash-can text-xs"></i>
              </button>
            </div>
            <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <div>
                <label class="form-label">指标</label>
                <select v-model="rule.metric" class="input">
                  <option value="cpu">CPU 使用率</option>
                  <option value="memory">内存使用率</option>
                  <option value="disk">磁盘使用率</option>
                </select>
              </div>
              <div>
                <label class="form-label">阈值(%)</label>
                <input v-model.number="rule.threshold" type="number" min="1" max="100" required class="input" />
              </div>
              <div>
                <label class="form-label">持续次数</label>
                <input v-model.number="rule.duration" type="number" min="1" required class="input" />
              </div>
            </div>
          </div>
        </div>
      </div>
      <button type="button" class="btn-add-row" @click="addRule">
        <i class="fas fa-plus text-xs"></i>添加规则
      </button>
      <p class="text-xs text-slate-400">持续次数指连续多少个采集周期超阈值才告警，用于抑制瞬时抖动。</p>
    </section>
  </div>
</template>
