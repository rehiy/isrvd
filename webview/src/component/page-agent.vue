<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'
import { PageAgent } from 'page-agent'

@Component
class PageAgentModal extends Vue {
    agent = {} as PageAgent

    initAgent() {
        // 使用本地 JWT token 作为 apiKey，以便请求能通过后端认证中间件
        const token = localStorage.getItem('app-token') || ''
        this.agent = new PageAgent({
            baseURL: '/api/agent/proxy',
            apiKey: token,
            model: 'proxy',
            language: 'zh-CN'
        })
    }

    mounted() {
        this.initAgent()
        this.agent.panel.show()
    }
}

export default toNative(PageAgentModal)
</script>

<template>
</template>
