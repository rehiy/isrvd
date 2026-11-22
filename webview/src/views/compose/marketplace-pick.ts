import type { ComposeDeployTarget, ComposeMarketplacePick } from '@/service/types'

let pending: { pick: ComposeMarketplacePick; target: ComposeDeployTarget; session: number } | undefined

export function holdMarketplacePick(pick: ComposeMarketplacePick, target: ComposeDeployTarget, session: number) {
    const selection = { pick, target, session }
    pending = selection
    return () => {
        if (pending === selection) {
            pending = undefined
        }
    }
}

export function takeMarketplacePick(target: ComposeDeployTarget, session: number): ComposeMarketplacePick | undefined {
    if (pending?.target !== target || pending.session !== session) {
        return undefined
    }
    const { pick } = pending
    pending = undefined
    return pick
}
