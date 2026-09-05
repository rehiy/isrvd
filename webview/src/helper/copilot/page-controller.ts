import { PageController } from '@page-agent/page-controller'

let controller: PageController | null = null

/**
 * 获取共享的 PageController 实例。
 *
 * 复用同一实例是必须的：元素序号由实例内部维护，多次实例化会导致序号错乱。
 * enableMask 在执行期间遮挡页面，避免用户操作与自动化互相干扰。
 */
export function getPageController(): PageController {
    if (!controller) {
        controller = new PageController({ enableMask: true })
    }
    return controller
}

export function disposePageController(): void {
    controller?.dispose()
    controller = null
}
