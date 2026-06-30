type SearchInputRef = HTMLInputElement | HTMLInputElement[] | null | undefined

const isEditableElement = (el: Element | null): boolean => {
    if (!el) return false
    if (el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement || el instanceof HTMLSelectElement) {
        return true
    }
    return el instanceof HTMLElement && el.isContentEditable
}

const resolveSearchInput = (inputRef: SearchInputRef): HTMLInputElement | null => {
    if (!inputRef) return null
    if (Array.isArray(inputRef)) {
        return inputRef.find((el: HTMLInputElement) => el && el.offsetParent !== null) || null
    }
    return inputRef
}

// 在页面空白区域直接键入时，将输入重定向到搜索框。
export const bindTypeToSearchFocus = (getInput: () => SearchInputRef): (() => void) => {
    const handleKeydown = (event: KeyboardEvent) => {
        if (event.defaultPrevented || event.isComposing) return
        if (event.ctrlKey || event.metaKey || event.altKey) return
        const isPrintable = event.key.length === 1 && event.key.trim() !== ''
        const isDeleteKey = event.key === 'Backspace' || event.key === 'Delete'
        if (!isPrintable && !isDeleteKey) return
        if (document.querySelector('.modal-card')) return
        if (isEditableElement(document.activeElement)) return

        const input = resolveSearchInput(getInput())
        if (!input || input.disabled || input.readOnly) return

        event.preventDefault()
        let nextValue = input.value
        if (isPrintable) {
            nextValue = `${input.value}${event.key}`
        } else if (event.key === 'Backspace') {
            nextValue = input.value.slice(0, -1)
        } else if (event.key === 'Delete') {
            nextValue = ''
        }

        input.focus()
        input.value = nextValue
        input.dispatchEvent(new Event('input', { bubbles: true }))
        const cursor = nextValue.length
        input.setSelectionRange(cursor, cursor)
        requestAnimationFrame(() => input.focus())
    }

    window.addEventListener('keydown', handleKeydown)
    return () => window.removeEventListener('keydown', handleKeydown)
}

/**
 * 复制文本到剪贴板
 * 优先使用 Clipboard API，降级到 execCommand
 * @returns 是否复制成功
 */
export const copyToClipboard = async (text: string): Promise<boolean> => {
    try {
        if (navigator.clipboard?.writeText) {
            await navigator.clipboard.writeText(text)
            return true
        }
        const el = document.createElement('textarea')
        el.value = text
        el.style.cssText = 'position:fixed;top:-9999px;left:-9999px;opacity:0'
        document.body.appendChild(el)
        el.select()
        const ok = document.execCommand('copy')
        document.body.removeChild(el)
        return ok
    } catch {
        return false
    }
}
