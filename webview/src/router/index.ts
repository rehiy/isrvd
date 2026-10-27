import { createRouter, createWebHashHistory, RouteRecordRaw } from 'vue-router'

import { isCenterPage, nodeLocation } from '@/helper/node'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'home',
    redirect: '/overview'
  },
  {
    path: '/overview',
    name: 'overview',
    component: () => import('@/views/overview/index.vue'),
    meta: { title: '概览' }
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
    component: () => import('@/views/local/explorer.vue'),
    meta: { title: '文件管理' }
  },
  {
    path: '/local/shell',
    name: 'local-shell',
    component: () => import('@/views/local/shell.vue'),
    meta: { title: 'Shell 终端' }
  },
  {
    path: '/local/process',
    name: 'local-process',
    component: () => import('@/views/local/process.vue'),
    meta: { title: '进程管理' }
  },
  {
    path: '/ssh',
    name: 'ssh',
    redirect: '/ssh/hosts'
  },
  {
    path: '/ssh/hosts',
    name: 'ssh-hosts',
    component: () => import('@/views/ssh/hosts.vue'),
    meta: { title: '主机连接' }
  },
  {
    path: '/ssh/host/:id',
    name: 'ssh-client',
    component: () => import('@/views/ssh/client.vue'),
    meta: { title: 'SSH 客户端' }
  },
  {
    path: '/ssh/credentials',
    name: 'ssh-credentials',
    component: () => import('@/views/ssh/credentials.vue'),
    meta: { title: '认证凭据' }
  },
  {
    path: '/node',
    name: 'node',
    redirect: '/node/access'
  },
  {
    path: '/node/access',
    name: 'node-access',
    component: () => import('@/views/node/nodes.vue'),
    meta: { title: '节点管理' }
  },
  {
    path: '/node/portmaps',
    name: 'node-portmaps',
    component: () => import('@/views/node/portmaps.vue'),
    meta: { title: '端口转发' }
  },
  {
    path: '/apisix',
    name: 'apisix',
    redirect: '/apisix/routes'
  },
  {
    path: '/apisix/routes',
    name: 'apisix-routes',
    component: () => import('@/views/apisix/routes.vue'),
    meta: { title: 'APISIX 路由' }
  },
  {
    path: '/apisix/upstreams',
    name: 'apisix-upstreams',
    component: () => import('@/views/apisix/upstreams.vue'),
    meta: { title: 'APISIX 上游' }
  },
  {
    path: '/apisix/plugin-configs',
    name: 'apisix-plugin-configs',
    component: () => import('@/views/apisix/plugin-configs.vue'),
    meta: { title: 'APISIX 插件配置' }
  },
  {
    path: '/apisix/ssls',
    name: 'apisix-ssls',
    component: () => import('@/views/apisix/ssls.vue'),
    meta: { title: 'APISIX SSL 证书' }
  },
  {
    path: '/apisix/consumers',
    name: 'apisix-consumers',
    component: () => import('@/views/apisix/consumers.vue'),
    meta: { title: 'APISIX 消费者' }
  },
  {
    path: '/apisix/whitelist',
    name: 'apisix-whitelist',
    component: () => import('@/views/apisix/whitelist.vue'),
    meta: { title: '访问授权' }
  },
  {
    path: '/caddy',
    name: 'caddy',
    redirect: '/caddy/routes'
  },
  {
    path: '/caddy/servers',
    name: 'caddy-servers',
    component: () => import('@/views/caddy/servers.vue'),
    meta: { title: 'Caddy 服务' }
  },
  {
    path: '/caddy/routes',
    name: 'caddy-routes',
    component: () => import('@/views/caddy/routes.vue'),
    meta: { title: 'Caddy 路由' }
  },
  {
    path: '/caddy/certs',
    name: 'caddy-certs',
    component: () => import('@/views/caddy/certs.vue'),
    meta: { title: 'Caddy SSL 证书' }
  },
  {
    path: '/caddy/global',
    name: 'caddy-global',
    component: () => import('@/views/caddy/global.vue'),
    meta: { title: '全局选项' }
  },
  {
    path: '/caddy/basic-auth',
    name: 'caddy-basic-auth',
    component: () => import('@/views/caddy/basic-auth.vue'),
    meta: { title: '基础认证' }
  },
  {
    path: '/caddy/raw',
    name: 'caddy-raw',
    component: () => import('@/views/caddy/raw.vue'),
    meta: { title: '原始配置' }
  },
  {
    path: '/docker',
    name: 'docker',
    redirect: '/docker/containers'
  },
  {
    path: '/docker/containers',
    name: 'docker-containers',
    component: () => import('@/views/docker/containers.vue'),
    meta: { title: '容器' }
  },
  {
    path: '/docker/container/:id',
    name: 'docker-container',
    component: () => import('@/views/docker/container.vue'),
    meta: { title: '容器详情' }
  },
  {
    path: '/docker/container/:id/stats',
    name: 'docker-container-stats',
    component: () => import('@/views/docker/container-stats.vue'),
    meta: { title: '容器监控' }
  },
  {
    path: '/docker/container/:id/logs',
    name: 'docker-container-logs',
    component: () => import('@/views/docker/container-logs.vue'),
    meta: { title: '容器日志' }
  },
  {
    path: '/docker/container/:id/exec',
    name: 'docker-container-exec',
    component: () => import('@/views/docker/container-exec.vue'),
    meta: { title: '容器终端' }
  },
  {
    path: '/docker/images',
    name: 'docker-images',
    component: () => import('@/views/docker/images.vue'),
    meta: { title: '镜像' }
  },
  {
    path: '/docker/image/:id',
    name: 'docker-image',
    component: () => import('@/views/docker/image.vue'),
    meta: { title: '镜像详情' }
  },
  {
    path: '/docker/networks',
    name: 'docker-networks',
    component: () => import('@/views/docker/networks.vue'),
    meta: { title: '网络' }
  },
  {
    path: '/docker/network/:id',
    name: 'docker-network',
    component: () => import('@/views/docker/network.vue'),
    meta: { title: '网络详情' }
  },
  {
    path: '/docker/volumes',
    name: 'docker-volumes',
    component: () => import('@/views/docker/volumes.vue'),
    meta: { title: '数据卷' }
  },
  {
    path: '/docker/volume/:name',
    name: 'docker-volume',
    component: () => import('@/views/docker/volume.vue'),
    meta: { title: '数据卷详情' }
  },
  {
    path: '/docker/registries',
    name: 'docker-registries',
    component: () => import('@/views/docker/registries.vue'),
    meta: { title: '镜像仓库' }
  },
  {
    path: '/swarm',
    name: 'swarm',
    redirect: '/swarm/nodes'
  },
  {
    path: '/swarm/nodes',
    name: 'swarm-nodes',
    component: () => import('@/views/swarm/nodes.vue'),
    meta: { title: 'Swarm 节点' }
  },
  {
    path: '/swarm/node/:id',
    name: 'swarm-node',
    component: () => import('@/views/swarm/node.vue'),
    meta: { title: 'Swarm 节点详情' }
  },
  {
    path: '/swarm/services',
    name: 'swarm-services',
    component: () => import('@/views/swarm/services.vue'),
    meta: { title: 'Swarm 服务' }
  },
  {
    path: '/swarm/service/:id',
    name: 'swarm-service',
    component: () => import('@/views/swarm/service.vue'),
    meta: { title: 'Swarm 服务详情' }
  },
  {
    path: '/swarm/service/:id/logs',
    name: 'swarm-service-logs',
    component: () => import('@/views/swarm/service-logs.vue'),
    meta: { title: '服务日志' }
  },
  {
    path: '/swarm/tasks',
    name: 'swarm-tasks',
    component: () => import('@/views/swarm/tasks.vue'),
    meta: { title: 'Swarm 任务' }
  },
  {
    path: '/compose/marketplace',
    name: 'compose-marketplace',
    component: () => import('@/views/compose/marketplace.vue'),
    meta: { title: '应用市场' }
  },
  {
    path: '/compose/deploy',
    name: 'compose-deploy',
    component: () => import('@/views/compose/deploy.vue'),
    meta: { title: 'Compose 部署' }
  },
  {
    path: '/cron',
    name: 'cron',
    redirect: '/cron/jobs'
  },
  {
    path: '/cron/jobs',
    name: 'cron-jobs',
    component: () => import('@/views/cron/jobs.vue'),
    meta: { title: '计划任务' }
  },
  {
    path: '/account/members',
    name: 'account-members',
    component: () => import('@/views/account/members.vue'),
    meta: { title: '用户管理' }
  },
  {
    path: '/account/password',
    name: 'account-password',
    component: () => import('@/views/account/password.vue'),
    meta: { title: '账号安全' }
  },
  {
    path: '/account/passkeys',
    name: 'account-passkeys',
    component: () => import('@/views/account/passkeys.vue'),
    meta: { title: 'Passkey' }
  },
  {
    path: '/account/apikey',
    name: 'account-apikey',
    component: () => import('@/views/account/apikey.vue'),
    meta: { title: 'API Key' }
  },
  {
    path: '/system/config',
    component: () => import('@/views/system/config.vue'),
    redirect: '/system/config/service',
    children: [
      {
        path: 'service',
        name: 'config-service',
        component: () => import('@/views/system/config/service.vue'),
        meta: { title: '基础服务' }
      },
      {
        path: 'auth',
        name: 'config-auth',
        component: () => import('@/views/system/config/auth.vue'),
        meta: { title: '登录认证' }
      },
      {
        path: 'gateway',
        name: 'config-gateway',
        component: () => import('@/views/system/config/gateway.vue'),
        meta: { title: '网关容器' }
      },
      {
        path: 'alert',
        name: 'config-alert',
        component: () => import('@/views/system/config/alert.vue'),
        meta: { title: '监控告警' }
      },
      {
        path: 'integrations',
        name: 'config-integrations',
        component: () => import('@/views/system/config/integrations.vue'),
        meta: { title: '扩展集成' }
      }
    ]
  },
  {
    path: '/system/audit/logs',
    name: 'system-audit-logs',
    component: () => import('@/views/system/audit-logs.vue'),
    meta: { title: '操作审计' }
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
  // 视角由页面路径决定，无需等待登录或 bootstrap，冷启动也不能挂载中控管理页面
  if (nodeLocation().id && isCenterPage(to.path)) return { path: '/overview' }

  // 未登录时放行（由 app.vue 的 v-if 控制显示登录页）
  if (!_isAuthenticated?.()) return true
  // 权限尚未加载完成时放行（刷新页面场景，等 loadMe 完成后由导航菜单 v-if 控制）
  if (!_permsLoaded?.()) return true

  if (to.path === '/compose/deploy' || to.path === '/compose/marketplace') {
    const dockerAvailable = _hasPerm?.('POST /api/compose/docker')
    const swarmAvailable = _hasPerm?.('POST /api/compose/swarm')
    if (!dockerAvailable && !swarmAvailable) return { path: '/overview' }
    if (to.path === '/compose/deploy' && to.query.target !== undefined) {
      if (to.query.target === 'docker' && dockerAvailable) return true
      if (to.query.target === 'swarm' && swarmAvailable) return true
      return { path: '/overview' }
    }
  }

  // 从前端路由路径提取模块名（/<module>/...）
  const module = to.path.match(/^\/([^/]+)/)?.[1]
  if (module && !['overview', 'account', 'local'].includes(module)) {
    if (!_hasPerm?.(module)) return { path: '/overview' }
  }
  return true
})

export default router
