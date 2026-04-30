import api from './api'

interface ServiceAvailabilityItem {
    available?: boolean
}

interface ServiceProbeResult {
    docker?: ServiceAvailabilityItem
    swarm?: ServiceAvailabilityItem
    apisix?: ServiceAvailabilityItem
}

/**
 * 获取服务探测数据
 */
export const fetchServiceProbe = async (): Promise<ServiceProbeResult> => {
    const response = await api.serviceProbe()
    return response.payload || {}
}
