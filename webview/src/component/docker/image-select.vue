<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  images: { type: Array, default: () => [] },
  placeholder: { type: String, default: '选择或输入镜像名称' },
  disabled: { type: Boolean, default: false }
})

const emit = defineEmits(['update:modelValue'])

const isOpen = ref(false)
const inputRef = ref(null)
const dropdownRef = ref(null)
const searchQuery = ref('')

const filteredImages = computed(() => {
  if (!searchQuery.value) return props.images
  const query = searchQuery.value.toLowerCase()
  return props.images.filter(img => 
    img.repoTags && img.repoTags[0] && 
    img.repoTags[0].toLowerCase().includes(query)
  )
})

const openDropdown = () => {
  if (props.disabled) return
  isOpen.value = true
  searchQuery.value = props.modelValue
}

const closeDropdown = () => {
  isOpen.value = false
  searchQuery.value = ''
}

const selectImage = (imageName) => {
  emit('update:modelValue', imageName)
  closeDropdown()
}

const handleInput = (value) => {
  emit('update:modelValue', value)
  searchQuery.value = value
  if (!isOpen.value) isOpen.value = true
}

const handleClickOutside = (e) => {
  if (inputRef.value && !inputRef.value.contains(e.target) &&
      dropdownRef.value && !dropdownRef.value.contains(e.target)) {
    closeDropdown()
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})

watch(() => props.modelValue, (val) => {
  if (!isOpen.value) searchQuery.value = val
})
</script>

<template>
  <div class="relative" ref="inputRef">
    <input
      type="text"
      :value="modelValue"
      @input="handleInput($event.target.value)"
      @focus="openDropdown"
      :placeholder="placeholder"
      :disabled="disabled"
      class="input pr-10"
    />
    <div class="absolute inset-y-0 right-0 flex items-center pr-3 pointer-events-none">
      <i :class="['fas fa-chevron-down text-slate-400 text-sm transition-transform duration-200', { 'rotate-180': isOpen }]"></i>
    </div>

    <Teleport to="body">
      <div
        v-if="isOpen && images.length > 0"
        ref="dropdownRef"
        class="fixed bg-white border border-slate-200 rounded-xl shadow-lg max-h-64 overflow-auto"
        :style="{
          top: inputRef ? inputRef.getBoundingClientRect().bottom + 4 + 'px' : '0',
          left: inputRef ? inputRef.getBoundingClientRect().left + 'px' : '0',
          width: inputRef ? inputRef.offsetWidth + 'px' : 'auto',
          zIndex: 9999
        }"
      >
        <div class="p-1">
          <div
            v-for="img in filteredImages"
            :key="img.id"
            @click="selectImage(img.repoTags[0])"
            class="px-3 py-2.5 hover:bg-blue-50 cursor-pointer flex items-center gap-3 rounded-lg transition-colors"
          >
            <div class="w-8 h-8 rounded-lg bg-gradient-to-br from-blue-400 to-blue-500 flex items-center justify-center flex-shrink-0">
              <i class="fas fa-compact-disc text-white text-xs"></i>
            </div>
            <div class="flex-1 min-w-0">
              <div class="text-sm font-medium text-slate-700 truncate">{{ img.repoTags[0] }}</div>
              <div class="text-xs text-slate-400">{{ img.shortId }}</div>
            </div>
          </div>
          <div 
            v-if="filteredImages.length === 0" 
            class="px-3 py-8 text-center"
          >
            <div class="w-12 h-12 mx-auto mb-2 rounded-full bg-slate-100 flex items-center justify-center">
              <i class="fas fa-search text-slate-400"></i>
            </div>
            <p class="text-sm text-slate-400">无匹配镜像</p>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
