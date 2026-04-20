import { createRouter, createWebHistory, RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'home',
    redirect: '/overview'
  },
  {
    path: '/overview',
    name: 'overview',
    component: () => import('@/views/overview.vue')
  },
  {
    path: '/explorer',
    name: 'explorer',
    component: () => import('@/views/filer/explorer.vue')
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
    path: '/docker',
    name: 'docker',
    redirect: '/docker/containers'
  },
  {
    path: '/docker/containers',
    name: 'docker-containers',
    component: () => import('@/views/docker/containers.vue')
  },
  {
    path: '/docker/container/:id',
    redirect: to => ({ name: 'docker-container-stats', params: { id: to.params.id } })
  },
  {
    path: '/docker/container/:id/stats',
    name: 'docker-container-stats',
    component: () => import('@/views/docker/container-stats.vue')
  },
  {
    path: '/docker/container/:id/logs',
    name: 'docker-container-logs',
    component: () => import('@/views/docker/container-logs.vue')
  },
  {
    path: '/docker/container/:id/terminal',
    name: 'docker-container-terminal',
    component: () => import('@/views/docker/container-terminal.vue')
  },
  {
    path: '/docker/images',
    name: 'docker-images',
    component: () => import('@/views/docker/images.vue')
  },
  {
    path: '/docker/image/:id',
    name: 'docker-image',
    component: () => import('@/views/docker/image.vue')
  },
  {
    path: '/docker/networks',
    name: 'docker-networks',
    component: () => import('@/views/docker/networks.vue')
  },
  {
    path: '/docker/network/:id',
    name: 'docker-network',
    component: () => import('@/views/docker/network.vue')
  },
  {
    path: '/docker/volumes',
    name: 'docker-volumes',
    component: () => import('@/views/docker/volumes.vue')
  },
  {
    path: '/docker/volume/:name',
    name: 'docker-volume',
    component: () => import('@/views/docker/volume.vue')
  },
  {
    path: '/docker/registries',
    name: 'docker-registries',
    component: () => import('@/views/docker/registries.vue')
  },
  {
    path: '/swarm',
    name: 'swarm',
    redirect: '/swarm/nodes'
  },
  {
    path: '/swarm/nodes',
    name: 'swarm-nodes',
    component: () => import('@/views/swarm/nodes.vue')
  },
  {
    path: '/swarm/node/:id',
    name: 'swarm-node',
    component: () => import('@/views/swarm/node.vue')
  },
  {
    path: '/swarm/services',
    name: 'swarm-services',
    component: () => import('@/views/swarm/services.vue')
  },
  {
    path: '/swarm/service/:id',
    redirect: to => ({ name: 'swarm-service-info', params: { id: to.params.id } })
  },
  {
    path: '/swarm/service/:id/info',
    name: 'swarm-service-info',
    component: () => import('@/views/swarm/service-info.vue')
  },
  {
    path: '/swarm/service/:id/logs',
    name: 'swarm-service-logs',
    component: () => import('@/views/swarm/service-logs.vue')
  },
  {
    path: '/swarm/tasks',
    name: 'swarm-tasks',
    component: () => import('@/views/swarm/tasks.vue')
  },
  {
    path: '/shell',
    name: 'shell',
    component: () => import('@/views/shell.vue')
  },
  {
    path: '/compose/marketplace',
    name: 'compose-marketplace',
    component: () => import('@/views/compose/marketplace.vue')
  },
  {
    path: '/system/members',
    name: 'system-members',
    component: () => import('@/views/system/members.vue')
  },
  {
    path: '/system/settings',
    name: 'system-settings',
    component: () => import('@/views/system/settings.vue')
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
