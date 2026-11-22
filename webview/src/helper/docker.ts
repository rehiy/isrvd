/** Docker 容器状态颜色，列表和详情页共用。 */
export function containerStateTextClass(state: string): string {
    switch (state) {
        case 'running':
            return 'text-emerald-600 font-medium'
        case 'paused':
        case 'restarting':
            return 'text-amber-600 font-medium'
        case 'stopped':
        case 'exited':
        case 'dead':
            return 'text-red-500 font-medium'
        default:
            return 'text-slate-500'
    }
}
