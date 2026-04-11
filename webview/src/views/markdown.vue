<script setup>
import { ref, onMounted, onUnmounted } from 'vue'

import 'cherry-markdown/dist/cherry-markdown.css'
import Cherry from 'cherry-markdown'

const cherryInstance = ref(null)

onMounted(() => {
  cherryInstance.value = new Cherry({
    id: 'markdown-editor',
    value: '# 欢迎使用 Markdown 编辑器\n\n开始编写你的 Markdown 内容...\n\n## 功能特性\n\n- 支持 GitHub 风格的 Markdown\n- 实时预览\n- 语法高亮\n- 表格支持\n\n```javascript\nconsole.log("Hello, World!");\n```',
  })
})

onUnmounted(() => {
  if (cherryInstance.value) {
    cherryInstance.value.destroy()
  }
})
</script>

<template>
  <div class="h-[calc(100vh-100px)]">
    <div class="h-full card overflow-hidden">
      <!-- Toolbar Bar -->
      <div class="bg-slate-50 border-b border-slate-200 px-6 py-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center space-x-3">
            <div class="w-10 h-10 rounded-xl bg-emerald-500 flex items-center justify-center">
              <i class="fas fa-edit text-white"></i>
            </div>
            <div>
              <h3 class="font-semibold text-slate-800">Markdown 编辑器</h3>
              <p class="text-xs text-slate-500">支持实时预览的 Markdown 编辑器</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Editor -->
      <div id="markdown-editor" class="h-[calc(100%-68px)]"></div>
    </div>
  </div>
</template>
