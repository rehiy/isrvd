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
    component: () => import('@/views/file-manager/index.vue')
  },
  {
    path: '/docker',
    name: 'docker',
    component: () => import('@/views/docker.vue')
  },
  {
    path: '/markdown',
    name: 'markdown',
    component: () => import('@/views/markdown.vue')
  },
  {
    path: '/shell',
    name: 'shell',
    component: () => import('@/views/shell.vue')
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
