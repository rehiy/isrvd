<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'
import { PageAgent } from 'page-agent'

@Component
class PageAgentModal extends Vue {
    agent = {} as PageAgent

    initAgent() {
        this.agent = new PageAgent({
            baseURL: '/api/agent/proxy',
            apiKey: 'proxy',
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
