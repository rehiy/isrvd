<script lang="ts">
import { css } from '@codemirror/lang-css'
import { go } from '@codemirror/lang-go'
import { html } from '@codemirror/lang-html'
import { javascript } from '@codemirror/lang-javascript'
import { json } from '@codemirror/lang-json'
import { markdown } from '@codemirror/lang-markdown'
import { python } from '@codemirror/lang-python'
import { sql } from '@codemirror/lang-sql'
import { xml } from '@codemirror/lang-xml'
import { yaml } from '@codemirror/lang-yaml'
import { Codemirror } from 'vue-codemirror'
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import api from '@/service/api'
import type { SFTPFileInfo } from '@/service/types'

import BaseModal from '@/component/modal.vue'

@Component({
    expose: ['show'],
    components: { BaseModal, Codemirror }
})
class SftpModifyModal extends Vue {
    portal = usePortal()
    
    // ─── 数据属性 ───
    isOpen = false
    loading = false
    hostId = ''
    formData = { filename: '', content: '', path: '' }
    readonly extensions = [css(), go(), html(), javascript(), json(), markdown(), python(), sql(), xml(), yaml()]

    // ─── 方法 ───
    async show(hostId: string, file: SFTPFileInfo, basePath: string) {
        this.hostId = hostId
        this.formData.path = basePath === '/' ? '/' + file.name : basePath + '/' + file.name
        this.formData.filename = file.name
        this.formData.content = ''
        this.isOpen = true
        
        // 读取文件内容
        try {
            this.loading = true
            const res = await api.sftpRead(this.hostId, this.formData.path)
            this.formData.content = res.payload?.content ?? ''
        } catch (e: unknown) {
            this.portal.showNotification('error', '读取文件失败: ' + (e instanceof Error ? e.message : ''))
            this.isOpen = false
        } finally {
            this.loading = false
        }
    }

    async handleConfirm() {
        this.loading = true
        try {
            await api.sftpWrite(this.hostId, {
                path: this.formData.path,
                content: this.formData.content
            })
            this.portal.showNotification('success', '文件保存成功')
            this.isOpen = false
            // 触发父组件刷新
            this.$emit('success')
        } catch (e: unknown) {
            this.portal.showNotification('error', '文件保存失败: ' + (e instanceof Error ? e.message : ''))
        } finally {
            this.loading = false
        }
    }
}

export default toNative(SftpModifyModal)
</script>

<template>
  <BaseModal ref="modalRef" v-model="isOpen" :title="'编辑: ' + formData.filename" :loading="loading" @confirm="handleConfirm">
    <div class="editor-container">
      <Codemirror v-model="formData.content" :style="{ height: '60vh' }" :extensions="extensions" :disabled="loading" />
    </div>

    <template #confirm-text>
      {{ loading ? '保存中...' : '保存文件' }}
    </template>
  </BaseModal>
</template>
