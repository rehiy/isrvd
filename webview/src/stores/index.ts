/**
 * Store 统一入口
 * 
 * 所有功能都通过 Portal 访问，确保：
 * 1. 统一的初始化流程
 * 2. 统一的拦截器注册
 * 3. 一致的访问接口
 */

import { usePortalStore } from './portal'
import type { PortalStore } from './portal'
import type { ConfirmOptions } from './ui'

export { useAgentStore } from './agent'
export type { AgentStore } from './agent'

// 包装 usePortal，提供明确的类型注解
export function usePortal(): PortalStore {
    return usePortalStore()
}

export type { PortalStore as Portal }
export type { ConfirmOptions }
