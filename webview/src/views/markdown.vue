<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import Cherry from 'cherry-markdown'
import 'cherry-markdown/dist/cherry-markdown.css'

import api from '@/service/api.js'
import FilePicker from '@/component/markdown/picker.vue'

const cherryInstance = ref(null)

// 文件状态
const currentPath = ref('')
const isDirty = ref(false)
const saveStatus = ref('')

// 文件选择器状态
const pickerMode = ref('open')
const pickerOpen = ref(false)

// 打开文件选择器
function openFilePicker(mode = 'open') {
  pickerMode.value = mode
  pickerOpen.value = true
}

// 选择文件（打开模式）
async function onFileSelect(file) {
  try {
    const res = await api.read(file.path)
    cherryInstance.value.setValue(res.payload?.content || '')
    currentPath.value = file.path
    isDirty.value = false
    saveStatus.value = ''
  } catch (e) {
    alert('读取文件失败：' + e.message)
  }
}

// 保存文件（保存模式）
async function onFileSave(fullPath) {
  try {
    await saveToPath(fullPath)
    pickerOpen.value = false
  } catch (e) {
    alert('保存失败：' + e.message)
  }
}

// 保存
async function save() {
  if (!currentPath.value) {
    openFilePicker('save')
    return
  }
  await saveToPath(currentPath.value)
}

// 实际写文件
async function saveToPath(path) {
  const content = cherryInstance.value.getValue()
  try {
    await api.modify(path, content)
    currentPath.value = path
    isDirty.value = false
    saveStatus.value = '已保存'
    setTimeout(() => { saveStatus.value = '' }, 2000)
  } catch (e) {
    try {
      await api.create(path, content)
      currentPath.value = path
      isDirty.value = false
      saveStatus.value = '已保存'
      setTimeout(() => { saveStatus.value = '' }, 2000)
    } catch (e2) {
      throw e2
    }
  }
}

// 新建文件
function newFile() {
  if (isDirty.value) {
    if (!confirm('当前有未保存的修改，确认新建？')) return
  }
  cherryInstance.value.setValue('# 新文档\n\n')
  currentPath.value = ''
  isDirty.value = false
  saveStatus.value = ''
}

onMounted(() => {
  cherryInstance.value = new Cherry({
    id: 'markdown-editor',
    value: '# 欢迎使用 Markdown 编辑器\n\n点击左上角 **打开** 加载文件，或直接开始编写。\n\n## 功能\n\n- 打开 / 保存到 home 目录\n- 实时预览\n- 语法高亮\n\n```javascript\nconsole.log("Hello!");\n```',
    callback: {
      afterChange: () => {
        isDirty.value = true
      }
    }
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
    <div class="h-full bg-white rounded-2xl shadow-sm border border-slate-200/60 overflow-hidden flex flex-col">

      <!-- Toolbar -->
      <div class="bg-slate-50 border-b border-slate-200 px-4 py-3 flex items-center justify-between flex-shrink-0">
        <div class="flex items-center space-x-3">
          <div class="w-9 h-9 rounded-xl bg-emerald-500 flex items-center justify-center">
            <i class="fas fa-edit text-white text-sm"></i>
          </div>
          <div class="min-w-0">
            <h3 class="font-semibold text-slate-800 text-sm leading-tight">Markdown 编辑器</h3>
            <p class="text-xs text-slate-400 truncate max-w-[240px]">
              {{ currentPath || '未命名' }}
              <span v-if="isDirty" class="text-amber-500 ml-1">●</span>
              <span v-if="saveStatus" class="text-emerald-500 ml-1">{{ saveStatus }}</span>
            </p>
          </div>
        </div>
        <!-- 操作按钮 -->
        <div class="flex items-center space-x-2">
          <button @click="newFile" class="btn-secondary text-sm px-3 py-1.5">
            <i class="fas fa-file mr-1"></i>新建
          </button>
          <button @click="openFilePicker('open')" class="btn-secondary text-sm px-3 py-1.5">
            <i class="fas fa-folder-open mr-1"></i>打开
          </button>
          <button @click="save" class="btn-success text-sm px-3 py-1.5">
            <i class="fas fa-save mr-1"></i>保存
          </button>
        </div>
      </div>

      <!-- Editor -->
      <div id="markdown-editor" class="flex-1 min-h-0"></div>
    </div>

    <!-- File Picker -->
    <FilePicker 
      v-model="pickerOpen" 
      :mode="pickerMode" 
      filter=".md"
      @select="onFileSelect"
      @save="onFileSave"
    />
  </div>
</template>
