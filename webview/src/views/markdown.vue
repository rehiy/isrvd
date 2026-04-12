<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import Cherry from 'cherry-markdown'
import 'cherry-markdown/dist/cherry-markdown.css'
import api from '@/service/api.js'

const cherryInstance = ref(null)

// 文件状态
const currentPath = ref('')      // 当前打开的文件路径（相对 home）
const isDirty = ref(false)       // 是否有未保存的修改
const saveStatus = ref('')       // 保存状态提示

// 文件浏览器弹窗
const showFilePicker = ref(false)
const browseDir = ref('')        // 当前浏览目录
const dirFiles = ref([])         // 当前目录文件列表
const pickerLoading = ref(false)

// 打开文件浏览器
async function openPicker() {
  browseDir.value = '/'
  showFilePicker.value = true
  await loadDir('/')
}

// 加载目录
async function loadDir(path) {
  pickerLoading.value = true
  try {
    const res = await api.list(path)
    browseDir.value = path
    dirFiles.value = res.payload?.files || []
    browseDir.value = res.payload?.path ?? path
  } catch (e) {
    dirFiles.value = []
  } finally {
    pickerLoading.value = false
  }
}

// 进入子目录
async function enterDir(file) {
  if (!file.isDir) return
  await loadDir(file.path)
}

// 返回上级目录
async function goUp() {
  if (!browseDir.value || browseDir.value === '/') return
  const parts = browseDir.value.replace(/\/$/, '').split('/')
  parts.pop()
  await loadDir(parts.join('/') || '/')
}

// 选择文件打开
async function selectFileToOpen(file) {
  if (file.isDir) {
    await enterDir(file)
    return
  }
  try {
    const res = await api.read(file.path)
    cherryInstance.value.setValue(res.payload?.content || '')
    currentPath.value = file.path
    isDirty.value = false
    saveStatus.value = ''
    showFilePicker.value = false
  } catch (e) {
    alert('读取文件失败：' + e.message)
  }
}

// 保存到当前路径
async function save() {
  if (!currentPath.value) return
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
    // 文件不存在时尝试 create
    try {
      await api.create(path, content)
      currentPath.value = path
      isDirty.value = false
      saveStatus.value = '已保存'
      setTimeout(() => { saveStatus.value = '' }, 2000)
    } catch (e2) {
      alert('保存失败：' + e2.message)
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
          <button @click="openPicker()" class="btn-secondary text-sm px-3 py-1.5">
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
  </div>

  <!-- File Picker Modal -->
  <Teleport to="body">
    <div v-if="showFilePicker" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm">
      <div class="bg-white rounded-2xl shadow-2xl w-[520px] max-h-[70vh] flex flex-col">
        <!-- Modal Header -->
        <div class="px-5 py-4 border-b border-slate-100 flex items-center justify-between">
          <div class="flex items-center space-x-2">
            <i class="fas fa-folder-open text-amber-500"></i>
            <span class="font-semibold text-slate-800">打开文件</span>
          </div>
          <button @click="showFilePicker = false" class="text-slate-400 hover:text-slate-600 transition-colors">
            <i class="fas fa-times"></i>
          </button>
        </div>

        <!-- Path Bar -->
        <div class="px-5 py-2 bg-slate-50 border-b border-slate-100 flex items-center space-x-2 text-sm text-slate-500">
          <button @click="loadDir('/')" class="hover:text-slate-800 transition-colors">
            <i class="fas fa-home"></i>
          </button>
          <i class="fas fa-chevron-right text-xs text-slate-300"></i>
          <span class="truncate">{{ browseDir || '/' }}</span>
          <button v-if="browseDir" @click="goUp" class="ml-auto hover:text-slate-800 transition-colors flex-shrink-0">
            <i class="fas fa-level-up-alt mr-1"></i>上级
          </button>
        </div>

        <!-- File List -->
        <div class="flex-1 overflow-y-auto px-3 py-2">
          <div v-if="pickerLoading" class="flex items-center justify-center py-10 text-slate-400">
            <i class="fas fa-spinner fa-spin mr-2"></i>加载中...
          </div>
          <div v-else-if="!dirFiles || dirFiles.length === 0" class="flex items-center justify-center py-10 text-slate-400 text-sm">
            目录为空
          </div>
          <div v-else>
            <div
              v-for="file in dirFiles"
              :key="file.path"
              @click="selectFileToOpen(file)"
              class="flex items-center space-x-3 px-3 py-2 rounded-xl cursor-pointer hover:bg-slate-50 transition-colors group"
              :class="{'opacity-40': !file.isDir && !file.name.endsWith('.md')}"
            >
              <i 
                :class="file.isDir ? 'fas fa-folder text-amber-400' : (file.name.endsWith('.md') ? 'fas fa-file-alt text-emerald-500' : 'fas fa-file text-slate-300')"
                class="w-4 text-center"
              ></i>
              <span class="text-sm text-slate-700 flex-1 truncate">{{ file.name }}</span>
              <span v-if="!file.isDir" class="text-xs text-slate-400 flex-shrink-0">{{ (file.size / 1024).toFixed(1) }} KB</span>
              <i v-if="file.isDir" class="fas fa-chevron-right text-xs text-slate-300 group-hover:text-slate-500 transition-colors flex-shrink-0"></i>
            </div>
          </div>
        </div>

        <!-- Modal Footer -->
        <div class="px-5 py-3 border-t border-slate-100 flex justify-end">
          <button @click="showFilePicker = false" class="btn-secondary text-sm px-3 py-1.5">取消</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
