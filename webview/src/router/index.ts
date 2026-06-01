import { createRouter, createWebHashHistory, RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'home',
    redirect: '/overview'
  },
  {
    path: '/overview',
    name: 'overview',
    component: () => import('@/views/overview/index.vue')
  },
  {
    path: '/local',
    name: 'local',
    redirect: '/local/monitor'
  },
  {
    path: '/local/monitor',
    name: 'local-monitor',
    component: () => import('@/views/local/monitor.vue'),
    meta: { title: '系统监控' }
  },
  {
    path: '/local/explorer',
    name: 'local-explorer',
    component: () => import('@/views/local/explorer.vue')
  },
  {
    path: '/local/shell',
    name: 'local-shell',
    component: () => import('@/views/local/shell.vue')
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
    path: '/apisix/upstreams',
    name: 'apisix-upstreams',
    component: () => import('@/views/apisix/upstreams.vue')
  },
  {
    path: '/apisix/plugin-configs',
    name: 'apisix-plugin-configs',
    component: () => import('@/views/apisix/plugin-configs.vue')
  },
  {
    path: '/apisix/ssls',
    name: 'apisix-ssls',
    component: () => import('@/views/apisix/ssls.vue')
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
    path: '/caddy',
    name: 'caddy',
    redirect: '/caddy/routes'
  },
  {
    path: '/caddy/routes',
    name: 'caddy-routes',
    component: () => import('@/views/caddy/routes.vue')
  },
  {
    path: '/caddy/certs',
    name: 'caddy-certs',
    component: () => import('@/views/caddy/certs.vue')
  },
  {
    path: '/caddy/global',
    name: 'caddy-global',
    component: () => import('@/views/caddy/global.vue')
  },
  {
    path: '/caddy/raw',
    name: 'caddy-raw',
    component: () => import('@/views/caddy/raw.vue')
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
    name: 'docker-container',
    component: () => import('@/views/docker/container.vue')
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
    path: '/docker/container/:id/exec',
    name: 'docker-container-exec',
    component: () => import('@/views/docker/container-exec.vue')
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
    name: 'swarm-service',
    component: () => import('@/views/swarm/service.vue')
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
    path: '/compose/deploy',
    name: 'compose-deploy',
    component: () => import('@/views/compose/deploy.vue')
  },
  {
    path: '/cron',
    name: 'cron',
    redirect: '/cron/jobs'
  },
  {
    path: '/cron/jobs',
    name: 'cron-jobs',
    component: () => import('@/views/cron/jobs.vue')
  },
  {
    path: '/ssh',
    name: 'ssh',
    redirect: '/ssh/hosts'
  },
  {
    path: '/ssh/credentials',
    name: 'ssh-credentials',
    component: () => import('@/views/ssh/credentials.vue')
  },
  {
    path: '/ssh/hosts',
    name: 'ssh-hosts',
    component: () => import('@/views/ssh/hosts.vue')
  },
  {
    path: '/ssh/host/:id',
    name: 'ssh-client',
    component: () => import('@/views/ssh/client.vue'),
    meta: { title: 'SSH 客户端' }
  },
  {
    path: '/account/members',
    name: 'account-members',
    component: () => import('@/views/account/members.vue')
  },
  {
    path: '/account/profile',
    name: 'account-profile',
    component: () => import('@/views/account/profile.vue')
  },
  {
    path: '/system/config',
    name: 'system-config',
    component: () => import('@/views/system/config.vue')
  },
  {
    path: '/system/audit/logs',
    name: 'system-audit-logs',
    component: () => import('@/views/system/audit-logs.vue')
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

// 由 main.ts 在 Pinia 初始化后注入，避免循环依赖
let _hasPerm: ((module: string) => boolean) | null = null
let _permsLoaded: (() => boolean) | null = null
let _isAuthenticated: (() => boolean) | null = null

export const setRouterGuard = (
    hasPerm: (module: string) => boolean,
    permsLoaded: () => boolean,
    isAuthenticated: () => boolean
) => {
    _hasPerm = hasPerm
    _permsLoaded = permsLoaded
    _isAuthenticated = isAuthenticated
}

// 路由守卫：根据权限控制页面访问
router.beforeEach((to) => {
  // 未登录时放行（由 app.vue 的 v-if 控制显示登录页）
  if (!_isAuthenticated?.()) return true
  // 权限尚未加载完成时放行（刷新页面场景，等 loadMe 完成后由导航菜单 v-if 控制）
  if (!_permsLoaded?.()) return true

  // 从路由路径提取模块名（/api/<module>/...）
  const module = to.path.match(/^\/([^/]+)/)?.[1]
  if (module && !['overview', 'account', 'local'].includes(module)) {
    if (!_hasPerm?.(module)) return { path: '/overview' }
  }
  return true
})

export default router
