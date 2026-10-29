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
import { EditorView } from 'codemirror'
import { Codemirror } from 'vue-codemirror'
import { Component, Prop, Vue, toNative } from 'vue-facing-decorator'

const LANGUAGE_EXTENSIONS = {
    css: css(),
    go: go(),
    html: html(),
    javascript: javascript(),
    jsx: javascript({ jsx: true }),
    json: json(),
    markdown: markdown(),
    python: python(),
    sql: sql(),
    typescript: javascript({ typescript: true }),
    tsx: javascript({ typescript: true, jsx: true }),
    xml: xml(),
    yaml: yaml(),
}

const FILE_LANGUAGES: Record<string, keyof typeof LANGUAGE_EXTENSIONS> = {
    css: 'css',
    go: 'go',
    htm: 'html',
    html: 'html',
    vue: 'html',
    js: 'javascript',
    mjs: 'javascript',
    cjs: 'javascript',
    jsx: 'jsx',
    json: 'json',
    md: 'markdown',
    markdown: 'markdown',
    py: 'python',
    sql: 'sql',
    ts: 'typescript',
    mts: 'typescript',
    cts: 'typescript',
    tsx: 'tsx',
    xml: 'xml',
    svg: 'xml',
    yaml: 'yaml',
    yml: 'yaml',
}

@Component({
    components: { Codemirror },
    emits: ['update:modelValue', 'focus', 'blur']
})
class CodeEditor extends Vue {
    @Prop({ type: String, default: '' }) readonly modelValue!: string
    @Prop({ type: String, default: '' }) readonly language!: string
    @Prop({ type: String, default: '' }) readonly filename!: string
    @Prop({ type: String, default: '' }) readonly placeholder!: string
    @Prop({ type: String, default: '' }) readonly label!: string
    @Prop({ type: String, default: '15rem' }) readonly height!: string
    @Prop({ type: Boolean, default: false }) readonly disabled!: boolean

    get content() {
        return this.modelValue
    }

    set content(value: string) {
        this.$emit('update:modelValue', value)
    }

    get extensions() {
        const suffix = this.filename.toLowerCase().split('.').pop() || ''
        const language = this.language || (Object.prototype.hasOwnProperty.call(FILE_LANGUAGES, suffix) ? FILE_LANGUAGES[suffix] : '')
        const syntax = Object.prototype.hasOwnProperty.call(LANGUAGE_EXTENSIONS, language)
            ? LANGUAGE_EXTENSIONS[language as keyof typeof LANGUAGE_EXTENSIONS]
            : []
        return [
            syntax,
            EditorView.lineWrapping,
            EditorView.contentAttributes.of({
                'aria-label': this.label || this.filename || this.$t('文件内容'),
                spellcheck: 'false',
                autocapitalize: 'off',
                autocomplete: 'off',
                autocorrect: 'off',
            }),
        ]
    }
}

export default toNative(CodeEditor)
</script>

<template>
  <div class="editor-container min-w-0">
    <Codemirror
      v-model="content"
      :extensions="extensions"
      :style="{ height }"
      :disabled="disabled"
      :placeholder="placeholder"
      :tab-size="2"
      :indent-with-tab="false"
      @focus="$emit('focus')"
      @blur="$emit('blur')"
    />
  </div>
</template>
