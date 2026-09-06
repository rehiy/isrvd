<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import type { SwarmServiceInfo } from '@/service/types'

import RedeployModal from '@/views/compose/widget/redeploy-modal.vue'

@Component({
    expose: ['show'],
    components: { RedeployModal },
    emits: ['success']
})
class ServiceEditModal extends Vue {
    @Ref readonly modalRef!: InstanceType<typeof RedeployModal>

    serviceName = ''

    async show(svc: SwarmServiceInfo) {
        this.serviceName = svc.name
        await this.$nextTick()
        await this.modalRef.show()
    }
}

export default toNative(ServiceEditModal)
</script>

<template>
  <RedeployModal
    ref="modalRef"
    target="swarm"
    :resource-name="serviceName"
    :title="`编辑服务：${serviceName}`"
    warning="更新配置后将会删除旧服务并重新创建，期间服务短暂不可用"
    refresh-title="跳过 compose.yml，按当前服务运行态重新反推 Compose"
    success-message="Swarm 服务配置更新成功，已重建服务"
    @success="$emit('success')"
  />
</template>
