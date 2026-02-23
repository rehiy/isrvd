<script setup>
import { ref, onMounted, onUnmounted } from 'vue'

import 'cherry-markdown/dist/cherry-markdown.css'
import Cherry from 'cherry-markdown'

const cherryInstance = ref(null)

onMounted(() => {
  cherryInstance.value = new Cherry({
    id: 'markdown-editor',
    value: '# Welcome to Markdown Editor\n\n开始编写你的 Markdown 内容...',
  })
})

onUnmounted(() => {
  if (cherryInstance.value) {
    cherryInstance.value.destroy()
  }
})
</script>

<template>
  <div class="editor-container">
    <div id="markdown-editor" class="markdown-editor"></div>
  </div>
</template>

<style scoped>
.editor-container {
  height: calc(100vh - 84px);
}
</style>
