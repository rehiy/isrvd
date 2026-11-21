import type { FileInfo, ListResult } from './types'

type RemoteListPayload = {
    path: string
    files?: Omit<FileInfo, 'path' | 'modeOctal'>[]
}

// 子项路径与当前目录均以服务端解析后的目录为准。
export function remoteListResult(payload?: RemoteListPayload | null): ListResult {
    const path = payload?.path ?? ''
    const basePath = path.replace(/\/+$/, '')
    return {
        path,
        files: (payload?.files ?? []).map(file => ({
            name: file.name,
            path: basePath + '/' + file.name,
            size: file.size,
            mode: file.mode,
            modeOctal: modeToOctal(file.mode),
            modTime: file.modTime,
            isDir: file.isDir,
            isLink: file.isLink || !!file.linkTarget,
            linkTarget: file.linkTarget,
        })),
    }
}

function modeToOctal(mode: string): string {
    if (mode.length < 10) {
        return ''
    }
    const parseRwx = (s: string): number => {
        let v = 0
        if (s[0] === 'r') {
            v += 4
        }
        if (s[1] === 'w') {
            v += 2
        }
        if (s[2] === 'x' || s[2] === 's' || s[2] === 't') {
            v += 1
        }
        return v
    }
    return `${parseRwx(mode.slice(1, 4))}${parseRwx(mode.slice(4, 7))}${parseRwx(mode.slice(7, 10))}`
}
