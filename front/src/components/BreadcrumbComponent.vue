<template>
  <nav aria-label="breadcrumb">
    <ol class="breadcrumb">
      <li class="breadcrumb-item">
        <a href="#" class="text-decoration-none" @click="navigateTo('/')">
          <i class="fas fa-home me-1"></i> 首页
        </a>
      </li>
      <li 
        v-for="(part, index) in pathParts" 
        :key="index" 
        class="breadcrumb-item" 
        :class="{ active: index === pathParts.length - 1 }"
      >
        <a 
          class="text-decoration-none" 
          v-if="index < pathParts.length - 1" 
          href="#" 
          @click="navigateTo('/' + pathParts.slice(0, index + 1).join('/'))"
        >
          {{ part }}
        </a>
        <span v-else>{{ part }}</span>
      </li>
    </ol>
  </nav>
</template>

<script>
import { defineComponent, inject, computed } from 'vue'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '../helpers/state.js'

export default defineComponent({
  name: 'BreadcrumbNav',
  setup() {
    const state = inject(APP_STATE_KEY)
    const actions = inject(APP_ACTIONS_KEY)

    const pathParts = computed(() => {
      if (state.currentPath === '/') return []
      return state.currentPath.split('/').filter(part => part)
    })

    const navigateTo = (path) => {
      actions.loadFiles(path)
    }

    return {
      state,
      pathParts,
      navigateTo
    }
  }
})
</script>
