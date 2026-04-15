import api from './api.js'

/**
 * 获取服务探测数据
 * @returns {Promise<{docker: {available: boolean}, swarm: {available: boolean}, apisix: {available: boolean}}>}
 */
export const fetchServiceProbe = async () => {
    const response = await api.serviceProbe()
    return response.payload
}
