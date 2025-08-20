<template>
  <BaseModal ref="modalRef" id="mkdirModal" title="新建目录" :loading="formData.loading" :confirm-disabled="!formData.name.trim()" @confirm="handleConfirm">
    <form @submit.prevent="handleConfirm">
      <div class="mb-3">
        <label for="dirName" class="form-label">目录名称</label>
        <input type="text" class="form-control" id="dirName" v-model="formData.name" :disabled="formData.loading" required>
      </div>
    </form>
    <template #confirm-text>
      {{ formData.loading ? '创建中...' : '创建' }}
    </template>
  </BaseModal>
</template>

<script>
import { defineComponent, inject, reactive, ref } from 'vue'
import axios from 'axios'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '../../helpers/state.js'
import BaseModal from '../base_modal.vue'

export default defineComponent({
  name: 'MkdirModal',
  components: { BaseModal },
  setup(props, { expose }) {
    const state = inject(APP_STATE_KEY)
    const actions = inject(APP_ACTIONS_KEY)

    const formData = reactive({
      name: '',
      loading: false
    })

    const modalRef = ref(null)

    const show = () => {
      formData.name = ''
      formData.loading = false
      modalRef.value?.show()
    }

    const handleConfirm = async () => {
      if (!formData.name.trim()) return

      formData.loading = true

      try {
        await axios.post('/api/mkdir', {
          path: state.currentPath,
          name: formData.name
        })

        actions.showSuccess('目录创建成功')
        actions.loadFiles()
        modalRef.value.hide()
      } catch (error) {
        actions.showError(error.response?.data?.error || '创建目录失败')
      } finally {
        formData.loading = false
      }
    }

    expose({ show })

    return {
      formData,
      show,
      handleConfirm,
      modalRef
    }
  }
})
</script>
