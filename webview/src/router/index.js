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
    redirect: '/docker/overview'
  },
  {
    path: '/docker/overview',
    name: 'docker-overview',
    component: () => import('@/views/docker/overview.vue')
  },
  {
    path: '/docker/containers',
    name: 'docker-containers',
    component: () => import('@/views/docker/containers.vue')
  },
  {
    path: '/docker/container/:id',
    component: () => import('@/views/docker/container.vue'),
    children: [
      {
        path: '',
        redirect: to => ({ name: 'docker-container-stats', params: { id: to.params.id } })
      },
      {
        path: 'stats',
        name: 'docker-container-stats',
        component: () => import('@/views/docker/container_stats.vue')
      },
      {
        path: 'logs',
        name: 'docker-container-logs',
        component: () => import('@/views/docker/container_logs.vue')
      },
      {
        path: 'terminal',
        name: 'docker-container-terminal',
        component: () => import('@/views/docker/container_terminal.vue')
      }
    ]
  },
  {
    path: '/docker/images',
    name: 'docker-images',
    component: () => import('@/views/docker/images.vue')
  },
  {
    path: '/docker/networks',
    name: 'docker-networks',
    component: () => import('@/views/docker/networks.vue')
  },
  {
    path: '/docker/volumes',
    name: 'docker-volumes',
    component: () => import('@/views/docker/volumes.vue')
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
