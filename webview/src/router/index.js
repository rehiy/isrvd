import { createRouter, createWebHashHistory } from 'vue-router'

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
    path: '/docker/registries',
    name: 'docker-registries',
    component: () => import('@/views/docker/registries.vue')
  },
  {
    path: '/docker/swarm',
    name: 'docker-swarm',
    component: () => import('@/views/docker/swarm.vue')
  },
  {
    path: '/apisix',
    name: 'apisix',
    redirect: '/apisix/routes'
  },
  {
    path: '/apisix/routes',
    name: 'apisix-routes',
    component: () => import('@/views/apisix/routes.vue')
  },
  {
    path: '/apisix/consumers',
    name: 'apisix-consumers',
    component: () => import('@/views/apisix/consumers.vue')
  },
  {
    path: '/apisix/whitelist',
    name: 'apisix-whitelist',
    component: () => import('@/views/apisix/whitelist.vue')
  },
  {
    path: '/shell',
    name: 'shell',
    component: () => import('@/views/shell.vue')
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

export default router
