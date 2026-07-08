import api from '@/service/api'
import type { DockerContainerInfo, DockerImageInfo } from '@/service/types'

export interface LoadDockerContainersOptions {
    all?: boolean
    runningOnly?: boolean
}

export async function loadDockerContainers(options: LoadDockerContainersOptions = {}): Promise<DockerContainerInfo[]> {
	try {
		const res = await api.dockerContainerList(options.all ?? false, { silentError: true })
		const containers = res.payload || []
		return options.runningOnly ? containers.filter(c => c.state === 'running') : containers
	} catch {
		return []
	}
}

export async function loadDockerImages(all = false): Promise<DockerImageInfo[]> {
	try {
		const res = await api.dockerImageList(all, { silentError: true })
		return res.payload || []
	} catch {
		return []
	}
}
