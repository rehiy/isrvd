import type { FileInfo } from './types'

// SSH 和容器文件列表共用路径、权限与符号链接字段转换。
export function remoteFileInfo(basePath: string, file: Omit<FileInfo, 'path' | 'modeOctal'>): FileInfo {
    return {
        name: file.name,
        path: basePath.replace(/\/+$/, '') + '/' + file.name,
        size: file.size,
        mode: file.mode,
        modeOctal: modeToOctal(file.mode),
        modTime: file.modTime,
        isDir: file.isDir,
        isLink: file.isLink || !!file.linkTarget,
        linkTarget: file.linkTarget,
    }
}

function modeToOctal(mode: string): string {
    if (mode.length < 10) return ''
    const parseRwx = (s: string): number => {
        let v = 0
        if (s[0] === 'r') v += 4
        if (s[1] === 'w') v += 2
        if (s[2] === 'x' || s[2] === 's' || s[2] === 't') v += 1
        return v
    }
    return `${parseRwx(mode.slice(1, 4))}${parseRwx(mode.slice(4, 7))}${parseRwx(mode.slice(7, 10))}`
}
