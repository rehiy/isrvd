<script lang="ts">
import { Component, Vue, toNative } from 'vue-facing-decorator'

import { usePortal } from '@/stores'

import CodeEditor from '@/component/code-editor.vue'
import BaseModal from '@/component/modal.vue'

import type { ExplorerAdapter, FileInfo } from '../types'

@Component({
    expose: ['show'],
    emits: ['success'],
    components: { BaseModal, CodeEditor },
})
class ModifyModal extends Vue {
    portal = usePortal()

    isOpen = false
    loading = false
    saving = false
    loadGeneration = 0
    adapter: ExplorerAdapter | null = null
    formData = { filename: '', content: '', path: '' }

    async show(adapter: ExplorerAdapter, file: FileInfo) {
        if (this.saving) return
        const generation = ++this.loadGeneration
        this.adapter = adapter
        this.formData.path = file.path
        this.formData.filename = file.name
        this.formData.content = ''
        this.isOpen = true
        this.loading = true
        try {
            const content = await adapter.readFile(file.path)
            if (generation === this.loadGeneration && this.isOpen) this.formData.content = content
        } catch {
            if (generation === this.loadGeneration) this.isOpen = false
        } finally {
            if (generation === this.loadGeneration) this.loading = false
        }
    }

    handleCancel() {
        this.loadGeneration++
        this.loading = false
    }

    async handleConfirm() {
        if (this.loading || !this.isOpen || !this.adapter) return
        this.loading = true
        this.saving = true
        try {
            await this.adapter.writeFile(this.formData.path, this.formData.content)
            this.portal.showNotification('success', this.$t('文件保存成功'))
            this.$emit('success')
            this.isOpen = false
        } catch {} finally {
            this.loading = false
            this.saving = false
        }
    }
}

export default toNative(ModifyModal)
</script>

<template>
  <BaseModal v-model="isOpen" :title="$t('编辑: ') + formData.filename" :loading="loading" :allow-close-while-loading="!saving" :close-disabled="saving" @confirm="handleConfirm" @cancel="handleCancel">
    <CodeEditor v-model="formData.content" :filename="formData.filename" height="60vh" :disabled="loading" />
    <template #confirm-text>{{ $t('保存') }}</template>
  </BaseModal>
</template>
