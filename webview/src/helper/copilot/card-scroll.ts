import { nextTick } from 'vue'

interface ScrollCard {
    element: HTMLElement
    collapse: () => void
    canCollapse: () => boolean
    seen: boolean
}

// 同一滚动容器统一收起和补偿高度，避免多个卡片同时折叠时互相覆盖 scrollTop。
const scrollGroups = new WeakMap<HTMLElement, { cards: Set<ScrollCard>; dispose: () => void }>()

export function bindCardAutoCollapse(element: HTMLElement, canCollapse: () => boolean, collapse: () => void): () => void {
    let root = element.parentElement
    while (root && !/^(auto|scroll)$/.test(getComputedStyle(root).overflowY)) root = root.parentElement
    if (!root) return () => {}

    let group = scrollGroups.get(root)
    if (!group) {
        const cards = new Set<ScrollCard>()
        group = { cards, dispose: installScrollListener(root, cards) }
        scrollGroups.set(root, group)
    }
    const card = { element, canCollapse, collapse, seen: isVisible(element, root) }
    group.cards.add(card)
    const scrollRoot = root
    return () => {
        group.cards.delete(card)
        if (!group.cards.size) {
            group.dispose()
            scrollGroups.delete(scrollRoot)
        }
    }
}

function installScrollListener(root: HTMLElement, cards: Set<ScrollCard>) {
    let previousTop = root.scrollTop
    let frame = 0
    const onScroll = () => {
        if (!frame) frame = requestAnimationFrame(update)
    }
    const update = async () => {
        frame = 0
        const scrollTop = root.scrollTop
        const movingDown = scrollTop > previousTop
        previousTop = scrollTop
        if (!root.clientHeight) return

        const top = root.getBoundingClientRect().top + root.clientTop
        const collapsing: { card: ScrollCard; height: number }[] = []
        for (const card of cards) {
            const rect = card.element.getBoundingClientRect()
            const wasSeen = card.seen
            card.seen ||= rect.bottom > top && rect.top < top + root.clientHeight
            // 尾部只剩 48px 时收起；未曾进入视野、向上回看和待确认卡片不受影响。
            if (movingDown && wasSeen && card.canCollapse() && rect.top < top && rect.bottom <= top + 48) {
                collapsing.push({ card, height: rect.height })
            }
        }
        if (!collapsing.length) return

        for (const { card } of collapsing) card.collapse()
        await nextTick()
        const removedHeight = collapsing.reduce((total, { card, height }) => total + height - card.element.getBoundingClientRect().height, 0)
        // 显式设置原位置减去缩短的高度，也兼容浏览器已进行的滚动锚定补偿。
        root.scrollTo({ top: Math.max(0, scrollTop - removedHeight), behavior: 'instant' })
        previousTop = root.scrollTop
    }
    root.addEventListener('scroll', onScroll, { passive: true })

    return () => {
        root.removeEventListener('scroll', onScroll)
        cancelAnimationFrame(frame)
    }
}

function isVisible(element: HTMLElement, root: HTMLElement): boolean {
    const rect = element.getBoundingClientRect()
    const top = root.getBoundingClientRect().top + root.clientTop
    return root.clientHeight > 0 && rect.bottom > top && rect.top < top + root.clientHeight
}
