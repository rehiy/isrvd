<script lang="ts">
import { Codemirror } from 'vue-codemirror'
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { CronJob, CronJobCreate, CronTypeInfo, DockerContainerInfo, DockerImageInfo } from '@/service/types'

import BaseModal from '@/component/modal.vue'

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
    components: { BaseModal, Codemirror, ImageSelect, ContainerSelect },
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

    show(job: CronJob | null = null, types: CronTypeInfo[] = []) {
        this.types = types
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
            this.formData = defaultFormData(types[0]?.value)
        }
        this.isOpen = true
        this.loadDockerData()
    }

    get isDockerType() {
        return this.formData.type === 'DOCKER_TMP' || this.formData.type === 'DOCKER_CTR'
    }

    async loadDockerData() {
        try {
            const [imgRes, ctRes] = await Promise.all([
                api.dockerImageList(false),
                api.dockerContainerList(true)
            ])
            this.images = imgRes.payload || []
            this.containers = ctRes.payload || []
        } catch {}
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
        <input v-model="formData.name" type="text" class="input" placeholder="如：每日备份" />
      </div>

      <div>
        <label class="form-label">描述</label>
        <textarea v-model="formData.description" rows="2" class="input resize-none" placeholder="可选说明"></textarea>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label class="form-label">
            执行计划 <span class="text-red-500">*</span>
            <a href="https://crontab.guru" target="_blank" rel="noreferrer" class="ml-1 text-primary-500 hover:underline normal-case font-normal">格式参考</a>
          </label>
          <input v-model="formData.schedule" type="text" class="input font-mono" placeholder="如：每天 2 点可填 0 2 * * *" />
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
          <input v-model="formData.workDir" type="text" class="input font-mono" placeholder="可选，默认当前目录" />
        </div>
        <div>
          <label class="form-label">超时时间（秒）</label>
          <input v-model.number="formData.timeout" type="number" min="0" class="input" placeholder="0 表示不限制" />
        </div>
      </div>

      <!-- DOCKER_TMP：镜像选择 + 额外挂载 + 超时 -->
      <template v-if="formData.type === 'DOCKER_TMP'">
        <div>
          <label class="form-label">镜像名 <span class="text-red-500">*</span></label>
          <ImageSelect v-model="formData.image" :images="images" placeholder="选择或输入镜像名，如 python:3.12-slim" />
        </div>
        <div>
          <label class="form-label">
            额外挂载
            <span class="text-xs font-normal text-slate-400 normal-case ml-1">每行 /host:/container[:ro]</span>
          </label>
          <textarea v-model="formData.volumes" rows="2" class="input font-mono resize-none" placeholder="/data/files:/data:ro&#10;/config:/etc/app:ro" />
        </div>
        <div>
          <label class="form-label">超时时间（秒）</label>
          <input v-model.number="formData.timeout" type="number" min="0" class="input" placeholder="0 表示不限制" />
        </div>
      </template>

      <!-- DOCKER_CTR：容器选择 + 超时 -->
      <template v-if="formData.type === 'DOCKER_CTR'">
        <div>
          <label class="form-label">目标容器 <span class="text-red-500">*</span></label>
          <ContainerSelect v-model="formData.container" :containers="containers" placeholder="选择或输入容器名" />
        </div>
        <div>
          <label class="form-label">超时时间（秒）</label>
          <input v-model.number="formData.timeout" type="number" min="0" class="input" placeholder="0 表示不限制" />
        </div>
      </template>

      <div>
        <label class="form-label">脚本内容 <span class="text-red-500">*</span></label>
        <div class="editor-container">
          <Codemirror v-model="formData.content" :style="{ height: '240px' }" :disabled="modalLoading" placeholder="输入脚本内容，如：#!/bin/bash&#10;echo &quot;hello&quot;" />
        </div>
      </div>

      <div>
        <label class="form-label">状态</label>
        <select v-model="formData.enabled" class="input">
          <option :value="true">启用</option>
          <option :value="false">禁用</option>
        </select>
      </div>
    </div>
  </BaseModal>
</template>
