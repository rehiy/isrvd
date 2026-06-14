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

import BaseModal from '@/component/modal.vue'

import type { FileInfo, ExplorerAdapter } from '../types'

@Component({
    expose: ['show'],
    emits: ['success'],
    components: { BaseModal, Codemirror },
})
class ModifyModal extends Vue {
    portal = usePortal()

    isOpen = false
    loading = false
    adapter: ExplorerAdapter | null = null
    formData = { filename: '', content: '', path: '' }
    readonly extensions = [css(), go(), html(), javascript(), json(), markdown(), python(), sql(), xml(), yaml()]

    async show(adapter: ExplorerAdapter, file: FileInfo) {
        this.adapter = adapter
        this.formData.path = file.path
        this.formData.filename = file.name
        this.formData.content = ''
        this.isOpen = true
        try {
            this.loading = true
            this.formData.content = await adapter.readFile(file.path)
        } catch (e: unknown) {
            this.portal.showNotification('error', '读取文件失败: ' + (e instanceof Error ? e.message : ''))
            this.isOpen = false
        } finally {
            this.loading = false
        }
    }

    async handleConfirm() {
        if (!this.adapter) return
        this.loading = true
        try {
            await this.adapter.writeFile(this.formData.path, this.formData.content)
            this.portal.showNotification('success', '文件保存成功')
            this.$emit('success')
            this.isOpen = false
        } catch (e: unknown) {
            this.portal.showNotification('error', '文件保存失败: ' + (e instanceof Error ? e.message : ''))
        } finally {
            this.loading = false
        }
    }
}

export default toNative(ModifyModal)
</script>

<template>
  <BaseModal v-model="isOpen" :title="'编辑: ' + formData.filename" :loading="loading" @confirm="handleConfirm">
    <div class="editor-container">
      <Codemirror v-model="formData.content" :style="{ height: '60vh' }" :extensions="extensions" :disabled="loading" />
    </div>
    <template #confirm-text>{{ loading ? '保存中...' : '保存文件' }}</template>
  </BaseModal>
</template>
