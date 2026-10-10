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

@Component({ components: { ToggleCard } })
class ConfigAlert extends Vue {
    config = useConfigStore()
    selectedWebhookTemplates = new WeakMap<WebhookConfig, string>()

    // Go text/template 变量说明；定义为变量后插值输出，避免与 Vue 的 {{ }} 语法冲突
    templateHint = '支持 Go 模板变量；在 JSON 字段中使用 {{.Title | json}} 可安全处理引号和换行。'
    get standardWebhookPreview() {
        return JSON.stringify(
            {
                source: 'isrvd',
                event: this.$t('<事件类型>'),
                level: this.$t('<告警级别>'),
                title: this.$t('<告警标题>'),
                message: this.$t('<告警内容>'),
                timestamp: 0,
                data: {},
            },
            null,
            2,
        )
    }

    webhookTemplates: WebhookTemplatePreset[] = [
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
            label: 'Telegram Bot',
            template: '{"chat_id":"<CHAT_ID>","text":{{printf "isrvd 告警：%s\\n%s" .Title .Message | json}}}',
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
        if (selected === 'generic' && !hook.template.trim()) return selected
        if (selected && this.webhookTemplates.some(item => item.value === selected && item.template === hook.template)) return selected
        this.selectedWebhookTemplates.delete(hook)
        if (!hook.template.trim()) return 'generic'
        const matches = this.webhookTemplates.filter(item => item.template === hook.template)
        return matches.length === 1 ? matches[0].value : 'custom'
    }

    applyWebhookTemplate(hook: WebhookConfig, event: Event) {
        const select = event.target as HTMLSelectElement
        if (select.value === 'custom') return
        if (select.value === 'generic') {
            this.selectedWebhookTemplates.set(hook, 'generic')
            hook.template = ''
            if (!hook.name.trim()) hook.name = this.$t('通用 Webhook')
            return
        }
        const preset = this.webhookTemplates.find(item => item.value === select.value)
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
          <h2 class="config-section-title">{{ $t('监控日志') }}</h2>
          <p class="config-section-description">{{ $t('系统与容器监控采集') }}</p>
        </div>
      </div>
      <div>
        <label class="form-label">{{ $t('监控采集间隔') }}</label>
        <select v-model.number="config.draft.monitor.interval" class="input">
          <option :value="0">{{ $t('禁用') }}</option>
          <option :value="5">{{ $t('5 秒') }}</option>
          <option :value="15">{{ $t('15 秒') }}</option>
          <option :value="30">{{ $t('30 秒') }}</option>
          <option :value="60">{{ $t('60 秒') }}</option>
        </select>
        <p class="mt-1 text-xs text-slate-400">{{ $t('系统与容器监控数据的采集频率，禁用后停止历史采集与资源阈值告警；下方应用故障告警仍可独立运行') }}</p>
      </div>
    </section>

    <!-- Webhook 通道 -->
    <section class="space-y-4">
      <div class="config-section-heading">
        <div>
          <h2 class="config-section-title">{{ $t('通知通道') }}</h2>
          <p class="config-section-description">{{ $t('配置 Webhook 地址与消息格式，资源和应用故障告警均推送到全部通道') }}</p>
        </div>
      </div>
      <div v-if="config.draft.notify.webhooks.length === 0" class="empty-note">{{ $t('暂无通知通道，点击下方按钮开始配置') }}</div>
      <div v-else class="space-y-4">
        <div v-for="(hook, index) in config.draft.notify.webhooks" :key="index" class="panel-frame">
          <div class="card-body space-y-4">
            <div class="flex items-center justify-between">
              <span class="text-sm font-semibold text-slate-700">{{ $t('通道') }} {{ index + 1 }}</span>
              <button type="button" class="btn-icon btn-icon-red" :title="$t('删除通道')" @click="removeWebhook(index)">
                <i class="fas fa-trash-can text-xs"></i>
              </button>
            </div>
            <div>
              <label class="form-label">{{ $t('通道名称') }}</label>
              <input v-model="hook.name" type="text" required :placeholder="$t('如：钉钉告警')" class="input" />
            </div>
            <div>
              <label class="form-label">{{ $t('Webhook 地址') }}</label>
              <input v-model="hook.url" type="url" required pattern="https?://.*" :placeholder="$t('粘贴渠道提供的 HTTP(S) Webhook 地址')" class="input" />
            </div>
            <div>
              <label class="form-label">{{ $t('消息渠道模板') }}</label>
              <select class="input" :value="webhookTemplateValue(hook)" @change="applyWebhookTemplate(hook, $event)">
                <option v-if="webhookTemplateValue(hook) === 'custom'" value="custom">{{ $t('自定义请求体模板') }}</option>
                <option value="generic">{{ $t('使用标准 JSON（无需模板）') }}</option>
                <option v-for="preset in webhookTemplates" :key="preset.value" :value="preset.value">{{ $t(preset.label) }}</option>
              </select>
              <p class="mt-1 text-xs text-slate-400">{{ $t('选择后会覆盖下方请求体；带占位符的模板需按提示补充对应参数。') }}</p>
            </div>
            <div v-if="webhookTemplateValue(hook) === 'generic'">
              <label class="form-label">{{ $t('标准请求体结构预览') }}</label>
              <textarea :value="standardWebhookPreview" rows="9" readonly class="input font-mono text-xs"></textarea>
              <p class="mt-1 text-xs text-slate-400">{{ $t('发送时由后端根据实际告警自动生成，配置中无需保存模板内容。') }}</p>
            </div>
            <div v-else>
              <label class="form-label">{{ $t('请求体模板（JSON）') }}</label>
              <textarea v-model="hook.template" rows="4" class="input font-mono text-xs" :placeholder="$t('留空则发送 isrvd 标准事件 JSON')"></textarea>
              <p class="mt-1 text-xs text-slate-400">{{ $t(templateHint) }}</p>
            </div>
          </div>
        </div>
      </div>
      <button type="button" class="btn-add-row" @click="addWebhook">
        <i class="fas fa-plus text-xs"></i>{{ $t('添加通道') }}
      </button>
    </section>

    <!-- 应用故障告警 -->
    <section class="space-y-4">
      <div class="config-section-heading">
        <div>
          <h2 class="config-section-title">{{ $t('应用故障告警') }}</h2>
          <p class="config-section-description">{{ $t('独立检测容器与网关证书，计划任务失败时立即通知') }}</p>
        </div>
      </div>
      <ToggleCard v-model="config.draft.notify.events.containerEnabled" :label="$t('容器故障')" :desc="$t('异常退出、内存不足、健康检查失败、持续重启及频繁重启；恢复后通知')">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label class="form-label">{{ $t('频繁重启次数') }}</label>
            <input v-model.number="config.draft.notify.events.restartThreshold" type="number" min="1" max="10000" required class="input" />
          </div>
          <div>
            <label class="form-label">{{ $t('重启统计窗口（秒）') }}</label>
            <input v-model.number="config.draft.notify.events.restartWindow" type="number" min="30" max="86400" required class="input" />
          </div>
        </div>
        <p class="mt-3 text-xs text-slate-400">{{ $t('每 30 秒检测一次，同一故障不重复发送；正常退出及常见停止信号不告警。为容器添加 isrvd.notify.ignore=true 标签可排除检测。') }}</p>
      </ToggleCard>
      <ToggleCard v-model="config.draft.notify.events.cronEnabled" :label="$t('计划任务失败')" :desc="$t('每次失败执行发送一次通知，包括超时；详细原因在执行历史中查看，通知不含脚本或输出')" />
      <ToggleCard v-model="config.draft.notify.events.certificateEnabled" :label="$t('证书到期')" :desc="$t('检查 APISIX 与 Caddy 的可读取证书，到期前提醒，续期恢复后通知')">
        <div>
          <label class="form-label">{{ $t('提前提醒天数') }}</label>
          <input v-model.number="config.draft.notify.events.certificateDays" type="number" min="1" max="3650" required class="input" />
        </div>
        <p class="mt-3 text-xs text-slate-400">{{ $t('每小时检查一次，同一证书每天最多提醒一次；检测到过期时升级告警级别。Caddy 文件与缓存证书需要面板进程能够读取。') }}</p>
      </ToggleCard>
    </section>

    <!-- 资源告警规则 -->
    <section class="space-y-4">
      <div class="config-section-heading">
        <div>
          <h2 class="config-section-title">{{ $t('资源告警规则') }}</h2>
          <p class="config-section-description">{{ $t('资源阈值与持续次数，命中后触发推送') }}</p>
        </div>
      </div>
      <div v-if="config.draft.notify.rules.length === 0" class="empty-note">{{ $t('暂无规则，点击下方按钮添加') }}</div>
      <div v-else class="space-y-4">
        <div v-for="(rule, index) in config.draft.notify.rules" :key="index" class="panel-frame">
          <div class="card-body space-y-4">
            <div class="flex items-center justify-between">
              <span class="text-sm font-semibold text-slate-700">{{ $t('规则') }} {{ index + 1 }}</span>
              <button type="button" class="btn-icon btn-icon-red" :title="$t('删除规则')" @click="removeRule(index)">
                <i class="fas fa-trash-can text-xs"></i>
              </button>
            </div>
            <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <div>
                <label class="form-label">{{ $t('指标') }}</label>
                <select v-model="rule.metric" class="input">
                  <option value="cpu">{{ $t('CPU 使用率') }}</option>
                  <option value="memory">{{ $t('内存使用率') }}</option>
                  <option value="disk">{{ $t('磁盘使用率') }}</option>
                </select>
              </div>
              <div>
                <label class="form-label">{{ $t('阈值(%)') }}</label>
                <input v-model.number="rule.threshold" type="number" min="1" max="100" required class="input" />
              </div>
              <div>
                <label class="form-label">{{ $t('持续次数') }}</label>
                <input v-model.number="rule.duration" type="number" min="1" required class="input" />
              </div>
            </div>
          </div>
        </div>
      </div>
      <button type="button" class="btn-add-row" @click="addRule">
        <i class="fas fa-plus text-xs"></i>{{ $t('添加规则') }}
      </button>
      <p class="text-xs text-slate-400">{{ $t('持续次数指连续多少个采集周期超阈值才告警，用于抑制瞬时抖动。') }}</p>
    </section>
  </div>
</template>
