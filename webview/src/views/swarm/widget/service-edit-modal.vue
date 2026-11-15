<script lang="ts">
import { Component, Ref, Vue, toNative } from 'vue-facing-decorator'

import type { SwarmServiceInfo } from '@/service/types'

import RedeployModal from '@/component/compose/redeploy-modal.vue'

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
    :title="$t(`编辑服务：${serviceName}`)"
    :warning="$t('更新将按 Swarm 策略执行，移除的服务会被删除')"
    :refresh-title="$t('跳过 compose.yml，按项目运行态重新反推 Compose')"
    :success-message="$t('Swarm 更新请求已提交')"
    @success="$emit('success')"
  />
</template>
