<script lang="ts">
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

import Combobox from '@/component/combobox.vue'

interface CapItem { name: string; desc: string }
interface CapCategory { name: string; icon: string; color: string; tone: string; caps: CapItem[] }

@Component({
    components: { Combobox },
    emits: ['update:modelValue']
})
class CapSelect extends Vue {
    @Prop({ type: Array, default: () => [] }) readonly modelValue!: string[]

    readonly capabilityCategories: CapCategory[] = [
        {
            name: '网络', icon: 'fa-network-wired', color: '#3b82f6', tone: 'blue',
            caps: [
                { name: 'NET_ADMIN', desc: '网络管理（配置接口、路由等）' },
                { name: 'NET_BIND_SERVICE', desc: '绑定 1024 以下端口' },
                { name: 'NET_BROADCAST', desc: '广播监听' },
                { name: 'NET_RAW', desc: '原始套接字（ping 等）' }
            ]
        },
        {
            name: '文件系统', icon: 'fa-folder-open', color: '#10b981', tone: 'emerald',
            caps: [
                { name: 'DAC_OVERRIDE', desc: '绕过文件读写执行权限检查' },
                { name: 'DAC_READ_SEARCH', desc: '绕过文件读和目录搜索权限' },
                { name: 'FOWNER', desc: '绕过文件属主权限检查' },
                { name: 'FSETID', desc: '修改文件 set-ID 位' },
                { name: 'CHOWN', desc: '修改文件属主' },
                { name: 'SETFCAP', desc: '设置文件能力' }
            ]
        },
        {
            name: '用户与进程', icon: 'fa-user-shield', color: '#8b5cf6', tone: 'violet',
            caps: [
                { name: 'SETUID', desc: '修改进程 UID' },
                { name: 'SETGID', desc: '修改进程 GID' },
                { name: 'SETPCAP', desc: '修改进程能力' },
                { name: 'SYS_ADMIN', desc: '系统管理（万能权限，谨慎使用）' },
                { name: 'KILL', desc: '向任意进程发送信号' },
                { name: 'AUDIT_WRITE', desc: '写入审计日志' }
            ]
        },
        {
            name: '系统控制', icon: 'fa-cogs', color: '#f59e0b', tone: 'amber',
            caps: [
                { name: 'SYS_CHROOT', desc: '使用 chroot' },
                { name: 'SYS_PTRACE', desc: '进程跟踪调试' },
                { name: 'SYS_RESOURCE', desc: '修改系统资源限制' },
                { name: 'SYS_TIME', desc: '修改系统时钟' },
                { name: 'SYS_NICE', desc: '修改进程优先级' },
                { name: 'MKNOD', desc: '创建设备文件' },
                { name: 'LINUX_IMMUTABLE', desc: '修改文件不可变属性' }
            ]
        },
        {
            name: '设备与内核', icon: 'fa-microchip', color: '#f43f5e', tone: 'rose',
            caps: [
                { name: 'SYS_MODULE', desc: '加载/卸载内核模块' },
                { name: 'SYS_RAWIO', desc: '原始 I/O 端口访问' },
                { name: 'SYS_BOOT', desc: '重启系统' },
                { name: 'SYSLOG', desc: '内核日志访问' },
                { name: 'LEASE', desc: '获取文件租约' },
                { name: 'BLOCK_SUSPEND', desc: '阻止系统挂起' }
            ]
        }
    ]

    // cap 名称 → 所属分类 的索引
    get capIndex(): Record<string, CapCategory> {
        const map: Record<string, CapCategory> = {}
        for (const cat of this.capabilityCategories) {
            for (const c of cat.caps) map[c.name] = cat
        }
        return map
    }

    // tone → 选中态样式（背景/文字/边框）
    readonly toneMap: Record<string, { tag: string; item: string; box: string }> = {
        blue:    { tag: 'bg-blue-50 text-blue-700 border border-blue-200',
                   item: 'bg-blue-50 border border-blue-200',
                   box: 'bg-blue-500 border-blue-500' },
        emerald: { tag: 'bg-emerald-50 text-emerald-700 border border-emerald-200',
                   item: 'bg-emerald-50 border border-emerald-200',
                   box: 'bg-emerald-500 border-emerald-500' },
        violet:  { tag: 'bg-violet-50 text-violet-700 border border-violet-200',
                   item: 'bg-violet-50 border border-violet-200',
                   box: 'bg-violet-500 border-violet-500' },
        amber:   { tag: 'bg-amber-50 text-amber-700 border border-amber-200',
                   item: 'bg-amber-50 border border-amber-200',
                   box: 'bg-amber-500 border-amber-500' },
        rose:    { tag: 'bg-rose-50 text-rose-700 border border-rose-200',
                   item: 'bg-rose-50 border border-rose-200',
                   box: 'bg-rose-500 border-rose-500' }
    }

    readonly fallbackTone = {
        tag: 'bg-slate-100 text-slate-700 border border-slate-200',
        item: 'bg-slate-50 border border-slate-200',
        box: 'bg-slate-500 border-slate-500'
    }

    toneFor(capName: string) {
        const cat = this.capIndex[capName]
        return cat ? this.toneMap[cat.tone] : this.fallbackTone
    }

    get placeholder() {
        return (this.modelValue?.length ?? 0) === 0 ? '点击选择或输入权限名称...' : ''
    }

    get tagClass() {
        return (val: string) => this.toneFor(val).tag
    }

    filteredCategories(query: string): CapCategory[] {
        if (!query) return this.capabilityCategories
        const upper = query.toUpperCase()
        return this.capabilityCategories
            .map(cat => ({ ...cat, caps: cat.caps.filter(c => c.name.includes(upper) || c.desc.includes(query)) }))
            .filter(cat => cat.caps.length > 0)
    }

    selectedCountIn(cat: CapCategory, selected: string[]) {
        return cat.caps.filter(c => selected.includes(c.name)).length
    }
}

export default toNative(CapSelect)
</script>

<template>
  <Combobox
    multiple
    :model-value="modelValue"
    :placeholder="placeholder"
    search-placeholder="搜索..."
    :tag-class="tagClass"
    max-height="320px"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <template #default="{ query, select, selected, isSelected }">
      <div v-for="cat in filteredCategories(query)" :key="cat.name" class="border-b border-slate-100 last:border-0">
        <div class="px-3 py-2 bg-slate-50/80 flex items-center gap-2 sticky top-0 z-10">
          <i :class="['fas text-xs', cat.icon]" :style="{ color: cat.color }"></i>
          <span class="text-xs font-semibold text-slate-600">{{ cat.name }}</span>
          <span class="text-xs text-slate-400">{{ selectedCountIn(cat, selected) }}/{{ cat.caps.length }}</span>
        </div>
        <div class="px-2 py-1.5 grid grid-cols-1 gap-0.5">
          <button
            v-for="cap in cat.caps"
            :key="cap.name"
            type="button"
            @click="select(cap.name)"
            :class="[
              'w-full flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-left transition-all duration-150 group',
              isSelected(cap.name)
                ? toneFor(cap.name).item
                : 'hover:bg-slate-50 border border-transparent'
            ]"
          >
            <span :class="[
              'w-4 h-4 rounded flex items-center justify-center flex-shrink-0 transition-all border',
              isSelected(cap.name)
                ? toneFor(cap.name).box
                : 'border-slate-300 group-hover:border-slate-400'
            ]">
              <i v-if="isSelected(cap.name)" class="fas fa-check text-white text-[9px]"></i>
            </span>
            <div class="flex-1 min-w-0">
              <div class="text-sm font-medium" :class="isSelected(cap.name) ? 'text-slate-800' : 'text-slate-700'">{{ cap.name }}</div>
              <div class="text-xs text-slate-400 truncate">{{ cap.desc }}</div>
            </div>
          </button>
        </div>
      </div>
    </template>

    <template #empty="{ query }">
      <div v-if="filteredCategories(query.toLowerCase()).length === 0" class="py-8 text-center">
        <i class="fas fa-search text-slate-300 text-2xl mb-2"></i>
        <p class="text-sm text-slate-400">未找到匹配的权限</p>
        <p class="text-xs text-slate-400 mt-1">按 Enter 可添加自定义权限</p>
      </div>
    </template>

    <template #footer="{ selected, clearAll }">
      <div class="px-3 py-2 bg-slate-50 border-t border-slate-100 flex items-center justify-between">
        <span class="text-xs text-slate-400">
          已选 <strong :class="selected.length > 0 ? 'text-slate-700' : 'text-slate-400'">{{ selected.length }}</strong> 项权限
        </span>
        <button
          v-if="selected.length > 0"
          type="button"
          class="text-xs text-red-500 hover:text-red-600 font-medium"
          @click="clearAll"
        >
          <i class="fas fa-trash-alt mr-1"></i>清空
        </button>
      </div>
    </template>
  </Combobox>
</template>
