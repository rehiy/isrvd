<script setup>
import { ref, watch } from 'vue'
import api from '@/service/api.js'
import BaseModal from '@/component/modal.vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  mode: { type: String, default: 'open' },
  filter: { type: String, default: '' },
})

const emit = defineEmits(['update:modelValue', 'select', 'save'])

const currentDir = ref('')
const files = ref([])
const loading = ref(false)
const fileName = ref('')

async function loadDir(path) {
  loading.value = true
  try {
    const res = await api.list(path)
    currentDir.value = res.payload?.path ?? path
    files.value = res.payload?.files || []
  } catch (e) {
    files.value = []
  } finally {
    loading.value = false
  }
}

async function enterDir(file) {
  if (!file.isDir) return
  await loadDir(file.path)
}

async function goUp() {
  if (!currentDir.value || currentDir.value === '/') return
  const parts = currentDir.value.replace(/\/$/, '').split('/')
  parts.pop()
  await loadDir(parts.join('/') || '/')
}

function selectFile(file) {
  if (file.isDir) {
    enterDir(file)
    return
  }
  emit('select', file)
  emit('update:modelValue', false)
}

function confirmSave() {
  if (!fileName.value.trim()) return
  const dir = currentDir.value
  const fullPath = dir === '/' 
    ? '/' + fileName.value.trim()
    : dir + '/' + fileName.value.trim()
  emit('save', fullPath)
}

function isSelectable(file) {
  if (file.isDir) return true
  if (!props.filter) return true
  return file.name.endsWith(props.filter)
}

watch(() => props.modelValue, (val) => {
  if (val) {
    fileName.value = ''
    loadDir('/')
  }
})

defineExpose({ loadDir })
</script>

<template>
  <BaseModal
    :model-value="modelValue"
    @update:model-value="$emit('update:modelValue', $event)"
    :title="mode === 'open' ? '打开文件' : '保存文件'"
    size="lg"
    :loading="loading"
    :confirm-disabled="!fileName.trim()"
    @confirm="confirmSave"
  >
    <!-- Path Bar -->
    <div class="flex items-center space-x-2 text-sm text-slate-500 mb-4 p-3 bg-slate-50 rounded-xl">
      <button @click="loadDir('/')" class="hover:text-slate-800 transition-colors">
        <i class="fas fa-home"></i>
      </button>
      <i class="fas fa-chevron-right text-xs text-slate-300"></i>
      <span class="truncate">{{ currentDir || '/' }}</span>
      <button v-if="currentDir && currentDir !== '/'" @click="goUp" class="ml-auto hover:text-slate-800 transition-colors flex-shrink-0">
        <i class="fas fa-level-up-alt mr-1"></i>上级
      </button>
    </div>

    <!-- File List -->
    <div class="overflow-y-auto max-h-[40vh] border border-slate-200 rounded-xl">
      <div v-if="loading" class="flex items-center justify-center py-10 text-slate-400">
        <i class="fas fa-spinner fa-spin mr-2"></i>加载中...
      </div>
      <div v-else-if="!files || files.length === 0" class="flex items-center justify-center py-10 text-slate-400 text-sm">
        目录为空
      </div>
      <div v-else>
        <div
          v-for="file in files"
          :key="file.path"
          @click="selectFile(file)"
          class="flex items-center space-x-3 px-4 py-3 cursor-pointer hover:bg-slate-50 transition-colors group border-b border-slate-100 last:border-b-0"
          :class="{
            'opacity-40 cursor-not-allowed': !isSelectable(file),
            'hidden': mode === 'save' && !file.isDir
          }"
        >
          <i 
            :class="file.isDir ? 'fas fa-folder text-amber-400' : 'fas fa-file-alt text-emerald-500'"
            class="w-4 text-center"
          ></i>
          <span class="text-sm text-slate-700 flex-1 truncate">{{ file.name }}</span>
          <span v-if="!file.isDir" class="text-xs text-slate-400 flex-shrink-0">{{ (file.size / 1024).toFixed(1) }} KB</span>
          <i v-if="file.isDir" class="fas fa-chevron-right text-xs text-slate-300 group-hover:text-slate-500 transition-colors flex-shrink-0"></i>
        </div>
      </div>
    </div>

    <!-- File Name Input (Save Mode Only) -->
    <div v-if="mode === 'save'" class="mt-4">
      <label class="block text-sm font-medium text-slate-700 mb-2">文件名</label>
      <div class="relative">
        <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
          <i class="fas fa-file text-slate-400"></i>
        </div>
        <input 
          v-model="fileName" 
          type="text" 
          placeholder="输入文件名（如：readme.md）"
          class="input pl-11"
          @keyup.enter="confirmSave"
        />
      </div>
    </div>

    <template #footer>
      <button 
        type="button" 
        class="btn-secondary"
        @click="$emit('update:modelValue', false)"
        :disabled="loading"
      >
        取消
      </button>
      <button 
        v-if="mode === 'save'"
        type="button" 
        class="btn-success"
        @click="confirmSave"
        :disabled="loading || !fileName.trim()"
      >
        <i class="fas fa-save mr-2"></i>保存
      </button>
    </template>
  </BaseModal>
</template>
