<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import { ExplorerPanel } from '@/component/explorer'
import type { ExplorerAdapter } from '@/component/explorer'
import { createSftpAdapter } from '../explorer-driver'

@Component({ components: { ExplorerPanel } })
class SftpPanel extends Vue {
    @Prop({ required: true }) readonly hostId!: string

    get adapter(): ExplorerAdapter {
        return createSftpAdapter(this.hostId)
    }
}

export default toNative(SftpPanel)
</script>

<template>
  <ExplorerPanel :adapter="adapter" :show-card="false" :show-batch-ops="true" />
</template>
