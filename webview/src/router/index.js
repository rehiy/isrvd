import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    name: 'home',
    redirect: '/explorer'
  },
  {
    path: '/explorer',
    name: 'explorer',
    component: () => import('@/layout/file-manager/index.vue')
  },
  {
    path: '/markdown',
    name: 'markdown',
    component: () => import('@/layout/markdown-editor/index.vue')
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
