<script setup>
import { computed, ref } from 'vue'

import Dropdown from '@/component/dropdown.vue'

const props = defineProps({
  modelValue: { type: Array, default: () => [] },
  type: { type: String, default: 'add' }, // 'add' or 'drop'
})

const emit = defineEmits(['update:modelValue'])

const dropdownOpen = ref(false)
const searchQuery = ref('')
const inputRef = ref(null)

// 常用 Linux 能力分类
const capabilityCategories = [
  {
    name: '网络',
    icon: 'fa-network-wired',
    color: '#3b82f6',
    caps: [
      { name: 'NET_ADMIN', desc: '网络管理（配置接口、路由等）' },
      { name: 'NET_BIND_SERVICE', desc: '绑定 1024 以下端口' },
      { name: 'NET_BROADCAST', desc: '广播监听' },
      { name: 'NET_RAW', desc: '原始套接字（ping 等）' },
    ],
  },
  {
    name: '文件系统',
    icon: 'fa-folder-open',
    color: '#10b981',
    caps: [
      { name: 'DAC_OVERRIDE', desc: '绕过文件读写执行权限检查' },
      { name: 'DAC_READ_SEARCH', desc: '绕过文件读和目录搜索权限' },
      { name: 'FOWNER', desc: '绕过文件属主权限检查' },
      { name: 'FSETID', desc: '修改文件 set-ID 位' },
      { name: 'CHOWN', desc: '修改文件属主' },
      { name: 'SETFCAP', desc: '设置文件能力' },
    ],
  },
  {
    name: '用户与进程',
    icon: 'fa-user-shield',
    color: '#8b5cf6',
    caps: [
      { name: 'SETUID', desc: '修改进程 UID' },
      { name: 'SETGID', desc: '修改进程 GID' },
      { name: 'SETPCAP', desc: '修改进程能力' },
      { name: 'SYS_ADMIN', desc: '系统管理（万能权限，谨慎使用）' },
      { name: 'KILL', desc: '向任意进程发送信号' },
      { name: 'AUDIT_WRITE', desc: '写入审计日志' },
    ],
  },
  {
    name: '系统控制',
    icon: 'fa-cogs',
    color: '#f59e0b',
    caps: [
      { name: 'SYS_CHROOT', desc: '使用 chroot' },
      { name: 'SYS_PTRACE', desc: '进程跟踪调试' },
      { name: 'SYS_RESOURCE', desc: '修改系统资源限制' },
      { name: 'SYS_TIME', desc: '修改系统时钟' },
      { name: 'SYS_NICE', desc: '修改进程优先级' },
      { name: 'MKNOD', desc: '创建设备文件' },
      { name: 'LINUX_IMMUTABLE', desc: '修改文件不可变属性' },
    ],
  },
  {
    name: '设备与内核',
    icon: 'fa-microchip',
    color: '#f43f5e',
    caps: [
      { name: 'SYS_MODULE', desc: '加载/卸载内核模块' },
      { name: 'SYS_RAWIO', desc: '原始 I/O 端口访问' },
      { name: 'SYS_BOOT', desc: '重启系统' },
      { name: 'SYSLOG', desc: '内核日志访问' },
      { name: 'LEASE', desc: '获取文件租约' },
      { name: 'BLOCK_SUSPEND', desc: '阻止系统挂起' },
    ],
  },
]

// 已选中的能力
const selectedCaps = computed({
  get: () => props.modelValue || [],
  set: (val) => emit('update:modelValue', val),
})

// 判断能力是否已选中
const isSelected = (capName) => selectedCaps.value.includes(capName)

// 切换能力选择
const toggleCap = (capName) => {
  if (isSelected(capName)) {
    selectedCaps.value = selectedCaps.value.filter(c => c !== capName)
  } else {
    selectedCaps.value = [...selectedCaps.value, capName]
  }
}

// 移除已选能力
const removeCap = (capName) => {
  selectedCaps.value = selectedCaps.value.filter(c => c !== capName)
}

// 添加自定义能力
const addCustomCap = () => {
  const cap = searchQuery.value.trim().toUpperCase()
  if (!cap) return
  if (cap.startsWith('CAP_')) {
    // 去掉 CAP_ 前缀，Docker compose 会自动加
  }
  if (!isSelected(cap) && cap.length > 0) {
    selectedCaps.value = [...selectedCaps.value, cap]
  }
  searchQuery.value = ''
}

// 过滤后的分类（搜索用）
const filteredCategories = computed(() => {
  if (!searchQuery.value.trim()) return capabilityCategories
  const query = searchQuery.value.toUpperCase()
  return capabilityCategories.map(cat => ({
    ...cat,
    caps: cat.caps.filter(cap =>
      cap.name.includes(query) || cap.desc.includes(searchQuery.value.trim())
    ),
  })).filter(cat => cat.caps.length > 0)
})
</script>

<template>
  <Dropdown v-model:open="dropdownOpen" max-height="320px">
    <!-- 触发区域：已选标签 + 搜索输入 -->
    <template #trigger="{ open }">
      <div class="input min-h-[42px] !px-3 !py-2 cursor-text flex flex-wrap gap-1.5 items-center" :class="open ? '!border-primary-400' : ''" @click="inputRef?.focus(); dropdownOpen = true">
        <span v-for="cap in selectedCaps" :key="cap" :class="[
          'inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-medium transition-all',
          type === 'add'
            ? 'bg-emerald-50 text-emerald-700 border border-emerald-200'
            : 'bg-red-50 text-red-700 border border-red-200'
        ]">
          {{ cap }}
          <button type="button" @click.stop="removeCap(cap)" :class="[
            'w-3.5 h-3.5 flex items-center justify-center rounded-full transition-colors',
            type === 'add'
              ? 'hover:bg-emerald-200 text-emerald-500'
              : 'hover:bg-red-200 text-red-500'
          ]">
            <i class="fas fa-times" style="font-size:8px"></i>
          </button>
        </span>
        <input ref="inputRef" v-model="searchQuery" type="text" class="flex-1 min-w-[80px] border-0 outline-none bg-transparent text-sm text-slate-700 placeholder:text-slate-400 p-0 focus:ring-0 focus:border-0 focus:shadow-none" :placeholder="selectedCaps.length === 0 ? '点击选择或输入权限名称...' : '搜索...'" @focus="dropdownOpen = true" @keydown.enter.prevent="addCustomCap" />
      </div>
    </template>

    <!-- 搜索提示 -->
    <template #search-hint>
      <div v-if="searchQuery.trim()" class="px-3 py-2 bg-slate-50 border-b border-slate-100 flex items-center justify-between">
        <span class="text-xs text-slate-500">按 Enter 添加自定义权限: <code class="bg-slate-200 px-1.5 py-0.5 rounded text-slate-700">{{ searchQuery.trim().toUpperCase() }}</code></span>
        <button type="button" class="text-xs text-primary-600 hover:text-primary-700 font-medium" @click="addCustomCap">
          <i class="fas fa-plus mr-1"></i>添加
        </button>
      </div>
    </template>

    <!-- 分类列表 -->
    <template #default>
      <div v-for="cat in filteredCategories" :key="cat.name" class="border-b border-slate-100 last:border-0">
        <div class="px-3 py-2 bg-slate-50/80 flex items-center gap-2 sticky top-0 z-10">
          <i :class="['fas text-xs', cat.icon]" :style="{ color: cat.color }"></i>
          <span class="text-xs font-semibold text-slate-600">{{ cat.name }}</span>
          <span class="text-xs text-slate-400">{{cat.caps.filter(c => isSelected(c.name)).length}}/{{ cat.caps.length }}</span>
        </div>
        <div class="px-2 py-1.5 grid grid-cols-1 gap-0.5">
          <button v-for="cap in cat.caps" :key="cap.name" type="button" @click="toggleCap(cap.name)" :class="[
            'w-full flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-left transition-all duration-150 group',
            isSelected(cap.name)
              ? type === 'add'
                ? 'bg-emerald-50 border border-emerald-200'
                : 'bg-red-50 border border-red-200'
              : 'hover:bg-slate-50 border border-transparent'
          ]">
            <span :class="[
              'w-4 h-4 rounded flex items-center justify-center flex-shrink-0 transition-all border',
              isSelected(cap.name)
                ? type === 'add'
                  ? 'bg-emerald-500 border-emerald-500'
                  : 'bg-red-500 border-red-500'
                : 'border-slate-300 group-hover:border-slate-400'
            ]">
              <i v-if="isSelected(cap.name)" class="fas fa-check text-white" style="font-size:9px"></i>
            </span>
            <div class="flex-1 min-w-0">
              <div class="text-sm font-medium" :class="isSelected(cap.name) ? 'text-slate-800' : 'text-slate-700'">{{ cap.name }}</div>
              <div class="text-xs text-slate-400 truncate">{{ cap.desc }}</div>
            </div>
          </button>
        </div>
      </div>
    </template>

    <!-- 空状态 -->
    <template #empty>
      <div v-if="filteredCategories.length === 0" class="py-8 text-center">
        <i class="fas fa-search text-slate-300 text-2xl mb-2"></i>
        <p class="text-sm text-slate-400">未找到匹配的权限</p>
        <p class="text-xs text-slate-400 mt-1">按 Enter 可添加自定义权限</p>
      </div>
    </template>

    <!-- 底部统计 -->
    <template #footer>
      <div class="px-3 py-2 bg-slate-50 border-t border-slate-100 flex items-center justify-between">
        <span class="text-xs text-slate-400">
          已选 <strong :class="selectedCaps.length > 0 ? 'text-slate-700' : 'text-slate-400'">{{ selectedCaps.length }}</strong> 项权限
        </span>
        <button v-if="selectedCaps.length > 0" type="button" class="text-xs text-red-500 hover:text-red-600 font-medium" @click="selectedCaps = []">
          <i class="fas fa-trash-alt mr-1"></i>清空
        </button>
      </div>
    </template>
  </Dropdown>
</template>
