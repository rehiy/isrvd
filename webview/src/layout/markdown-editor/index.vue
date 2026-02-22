<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'

import 'cherry-markdown/dist/cherry-markdown.css'
import Cherry from 'cherry-markdown'

const cherryInstance = ref(null)

onMounted(() => {
  nextTick(() => {
    cherryInstance.value = new Cherry({
      id: 'markdown-editor',
      value: '# welcome to cherry editor!\n\n开始编写你的 Markdown 内容...',
      theme: 'light',
      editor: {
        theme: 'default',
        height: '100%',
        defaultModel: 'edit&preview',
        convertLineHeight: '1.6em'
      },
      previewer: {
        theme: 'default',
        externals: {
          katex: {},
          mermaid: {}
        }
      }
    })
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
  height: calc(100vh - 88px);
}
</style>
