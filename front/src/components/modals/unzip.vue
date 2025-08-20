<template>
  <BaseModal ref="modalRef" id="unzipModal" title="解压确认" header-class="bg-warning text-dark" :loading="formData.loading" :confirm-disabled="!formData.file" @confirm="handleConfirm">
    <div v-if="formData.file" class="text-center">s
      <div class="mb-3">
        <i class="fas fa-file-archive text-warning display-1"></i>
      </div>
      <p class="mb-3">
        确定要解压 <strong>{{ formData.file.name }}</strong> 到当前目录吗？
      </p>
    </div>
    <template #confirm-text>
      {{ formData.loading ? '解压中...' : '解压' }}
    </template>
  </BaseModal>
</template>

<script>
import { defineComponent, inject, reactive, ref } from 'vue'
import axios from 'axios'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '../../helpers/state.js'
import BaseModal from '../base_modal.vue'

export default defineComponent({
  name: 'UnzipModal',
  components: { BaseModal },
  setup(props, { expose }) {
    const state = inject(APP_STATE_KEY)
    const actions = inject(APP_ACTIONS_KEY)

    const formData = reactive({
      file: null,
      loading: false
    })

    const modalRef = ref(null)

    const show = (file) => {
      formData.file = file
      formData.loading = false
      modalRef.value.show()
    }

    const handleConfirm = async () => {
      if (!formData.file) return

      formData.loading = true

      try {
        await axios.post('/api/unzip', {
          path: state.currentPath,
          zipName: formData.file.name
        })

        actions.showSuccess('解压成功')
        actions.loadFiles()
        modalRef.value.hide()
      } catch (error) {
        actions.showError(error.response?.data?.error || '解压失败')
      } finally {
        formData.loading = false
      }
    }

    expose({ show })

    return {
      formData,
      modalRef,
      show,
      handleConfirm
    }
  }
})
</script>
