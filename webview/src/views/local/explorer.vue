<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { ExplorerPanel } from '@/component/explorer'
import { createFilerAdapter } from './explorer-driver'

@Component({ components: { ExplorerPanel } })
class FileExplorer extends Vue {
    adapter = createFilerAdapter()
}

export default toNative(FileExplorer)
</script>

<template>
  <ExplorerPanel
    :adapter="adapter"
    :show-search="true"
    :show-batch-ops="true"
    :show-mobile-view="true"
  />
</template>
