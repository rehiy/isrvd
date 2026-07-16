<script lang="ts">
import { Codemirror } from 'vue-codemirror'
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { CronJob, CronJobCreate, CronTypeInfo, DockerContainerInfo, DockerImageInfo } from '@/service/types'

import BaseModal from '@/component/modal.vue'
import ToggleCard from '@/component/toggle-card.vue'

import ContainerSelect from '@/views/docker/widget/container-select.vue'
import ImageSelect from '@/views/docker/widget/image-select.vue'

const defaultFormData = (type = 'SHELL'): CronJobCreate => ({
    name: '',
    schedule: '',
    type: type as CronJobCreate['type'],
    content: '',
    workDir: '',
    image: '',
    container: '',
    volumes: '',
    timeout: 0,
    enabled: true,
    description: ''
})

@Component({
    expose: ['show'],
    components: { BaseModal, Codemirror, ImageSelect, ContainerSelect, ToggleCard },
    emits: ['success']
})
class JobEditModal extends Vue {
    portal = usePortal()

    isOpen = false
    modalLoading = false
    isEditMode = false
    jobID = ''
    types: CronTypeInfo[] = []
    formData = defaultFormData()

    images: DockerImageInfo[] = []
    containers: DockerContainerInfo[] = []

    get dockerAvailable() {
        return this.portal.serviceAvailability.docker
    }

    get canLoadDockerContainers() {
        return this.portal.hasPerm('GET /api/docker/containers')
    }

    get canLoadDockerImages() {
        return this.portal.hasPerm('GET /api/docker/images')
    }

    show(job: CronJob | null = null, types: CronTypeInfo[] = []) {
        this.types = this.dockerAvailable
            ? types
            : types.filter(type => type.value !== 'DOCKER_TMP' && type.value !== 'DOCKER_CTR')
        this.isEditMode = !!job
        if (job) {
            this.jobID = job.id
            this.formData = {
                name: job.name,
                schedule: job.schedule,
                type: job.type,
                content: job.content,
                workDir: job.workDir,
                image: job.image || '',
                container: job.container || '',
                volumes: job.volumes || '',
                timeout: job.timeout,
                enabled: job.enabled,
                description: job.description
            }
        } else {
            this.jobID = ''
            this.formData = defaultFormData(this.types[0]?.value)
        }
        this.isOpen = true
        void this.loadDockerData()
    }

    get isDockerType() {
        return this.formData.type === 'DOCKER_TMP' || this.formData.type === 'DOCKER_CTR'
    }

    async loadDockerData() {
        this.images = []
        this.containers = []
        const requests: Promise<void>[] = []
        if (this.canLoadDockerImages) {
            requests.push(api.dockerImageList(false).then(res => { this.images = res.payload || [] }))
        }
        if (this.canLoadDockerContainers) {
            requests.push(api.dockerContainerList(true).then(res => { this.containers = res.payload || [] }))
        }
        await Promise.allSettled(requests)
    }

    async handleConfirm() {
        if (!this.formData.name || !this.formData.schedule || !this.formData.content) {
            this.portal.showNotification('error', '请填写必填项：名称、执行计划、脚本内容')
            return
        }
        if (this.formData.type === 'DOCKER_TMP' && !this.formData.image) {
            this.portal.showNotification('error', 'DOCKER 镜像类型请填写镜像名')
            return
        }
        if (this.formData.type === 'DOCKER_CTR' && !this.formData.container) {
            this.portal.showNotification('error', 'DOCKER 容器类型请填写目标容器名')
            return
        }
        this.modalLoading = true
        try {
            if (this.isEditMode) {
                await api.cronJobUpdate(this.jobID, this.formData)
                this.portal.showNotification('success', '任务已更新')
            } else {
                await api.cronJobCreate(this.formData)
                this.portal.showNotification('success', '任务已创建')
            }
            this.isOpen = false
            this.$emit('success')
        } catch {
            this.portal.showNotification('error', this.isEditMode ? '更新任务失败' : '创建任务失败')
        } finally {
            this.modalLoading = false
        }
    }
}

export default toNative(JobEditModal)
</script>

<template>
  <BaseModal v-model="isOpen" :title="isEditMode ? '编辑计划任务' : '新建计划任务'" :loading="modalLoading" @confirm="handleConfirm">
    <template #confirm-text>{{ isEditMode ? '保存' : '新建' }}</template>

    <div class="max-w-3xl space-y-4 p-1">
      <div>
        <label class="form-label">任务名称 <span class="text-red-500">*</span></label>
        <input v-model="formData.name" type="text" class="input" placeholder="请输入任务名称" />
        <p class="text-xs text-slate-400 mt-1">例如：每日备份</p>
      </div>

      <div>
        <label class="form-label">描述</label>
        <textarea v-model="formData.description" rows="2" class="input resize-none" placeholder="请输入描述（可选）" />
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label class="form-label">
            执行计划 <span class="text-red-500">*</span>
            <a href="https://crontab.guru" target="_blank" rel="noreferrer" class="ml-1 text-primary-500 hover:underline normal-case font-normal">格式参考</a>
          </label>
          <input v-model="formData.schedule" type="text" class="input font-mono" placeholder="请输入执行计划（cron 表达式）" />
          <p class="text-xs text-slate-400 mt-1">标准 cron 表达式，例如：每天 2 点可填 0 2 * * *</p>
        </div>
        <div>
          <label class="form-label">脚本类型 <span class="text-red-500">*</span></label>
          <select v-model="formData.type" class="input">
            <option v-for="t in types" :key="t.value" :value="t.value">{{ t.label }}</option>
          </select>
        </div>
      </div>

      <!-- 宿主机类型：工作目录 + 超时 -->
      <div v-if="!isDockerType" class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label class="form-label">工作目录</label>
          <input v-model="formData.workDir" type="text" class="input font-mono" placeholder="请输入工作目录（可选）" />
          <p class="text-xs text-slate-400 mt-1">留空则使用当前目录</p>
        </div>
        <div>
          <label class="form-label">超时时间（秒）</label>
          <input v-model.number="formData.timeout" type="number" min="0" class="input" placeholder="请输入超时时间（可选）" />
          <p class="text-xs text-slate-400 mt-1">0 或留空表示不限制</p>
        </div>
      </div>

      <!-- DOCKER_TMP：镜像选择 + 额外挂载 + 超时 -->
      <template v-if="formData.type === 'DOCKER_TMP'">
        <div>
          <label class="form-label">镜像名 <span class="text-red-500">*</span></label>
          <ImageSelect v-if="canLoadDockerImages" v-model="formData.image" :images="images" placeholder="请输入或选择镜像名" />
          <input v-else v-model="formData.image" type="text" class="input" placeholder="请输入镜像名" />
          <p class="text-xs text-slate-400 mt-1">例如：python:3.12-slim、nginx:alpine</p>
        </div>
        <div>
          <label class="form-label">
            额外挂载
            <span class="text-xs font-normal text-slate-400 normal-case ml-1">每行 /host:/container[:ro]</span>
          </label>
          <textarea v-model="formData.volumes" rows="2" class="input font-mono resize-none" placeholder="请输入挂载配置（可选）" />
          <p class="text-xs text-slate-400 mt-1">例如：/data/files:/data:ro</p>
        </div>
        <div>
          <label class="form-label">超时时间（秒）</label>
          <input v-model.number="formData.timeout" type="number" min="0" class="input" placeholder="请输入超时时间（可选）" />
          <p class="text-xs text-slate-400 mt-1">0 或留空表示不限制</p>
        </div>
      </template>

      <!-- DOCKER_CTR：容器选择 + 超时 -->
      <template v-if="formData.type === 'DOCKER_CTR'">
        <div>
          <label class="form-label">目标容器 <span class="text-red-500">*</span></label>
          <ContainerSelect v-if="canLoadDockerContainers" v-model="formData.container" :containers="containers" placeholder="请输入或选择容器名" />
          <input v-else v-model="formData.container" type="text" class="input" placeholder="请输入容器名" />
        </div>
        <div>
          <label class="form-label">超时时间（秒）</label>
          <input v-model.number="formData.timeout" type="number" min="0" class="input" placeholder="请输入超时时间（可选）" />
          <p class="text-xs text-slate-400 mt-1">0 或留空表示不限制</p>
        </div>
      </template>

      <div>
        <label class="form-label">脚本内容 <span class="text-red-500">*</span></label>
        <div class="editor-container">
          <Codemirror v-model="formData.content" class="h-60" :disabled="modalLoading" placeholder="请输入脚本内容" />
          <p class="text-xs text-slate-400 mt-1">例如：#!/bin/bash<br>echo "hello"</p>
        </div>
      </div>

      <ToggleCard v-model="formData.enabled" label="状态" desc="启用后任务将按计划执行" />
    </div>
  </BaseModal>
</template>
