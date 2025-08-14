// 删除确认模态框组件

const { defineComponent, inject, reactive, ref } = Vue;

import { APP_ACTIONS_KEY } from '../../helpers/state.js';
import { BaseModal } from '../modal-base.js';

export const DeleteModal = defineComponent({
    name: 'DeleteModal',
    components: { BaseModal },
    setup() {
        const actions = inject(APP_ACTIONS_KEY);

        const formData = reactive({
            file: null,
            loading: false
        });

        const modalRef = ref(null);

        const show = (file) => {
            formData.file = file;
            formData.loading = false;
            modalRef.value.show();
        };

        const handleConfirm = async () => {
            if (!formData.file) return;

            formData.loading = true;

            try {
                await axios.delete('/api/delete', {
                    params: { file: formData.file.path }
                });

                actions.showSuccess('删除成功');
                actions.loadFiles();
                modalRef.value.hide();

            } catch (error) {
                actions.showError(error.response?.data?.error || '删除失败');
            } finally {
                formData.loading = false;
            }
        };

        const handleCancel = () => {
            // 取消操作
        };

        return {
            formData,
            modalRef,
            show,
            handleConfirm,
            handleCancel
        };
    },
    template: `
        <BaseModal
            ref="modalRef"
            id="deleteModal"
            title="删除确认"
            headerClass="bg-danger text-white"
            :loading="formData.loading"
            :confirm-disabled="!formData.file"
            @confirm="handleConfirm"
            @cancel="handleCancel"
        >
            <div v-if="formData.file" class="text-center">
                <div class="mb-3">
                    <i class="fas fa-exclamation-triangle text-warning display-1"></i>
                </div>
                <p class="mb-3">
                    确定要删除 <strong>{{ formData.file.name }}</strong> 吗？
                </p>
            </div>
            <template #confirm-text>
                {{ formData.loading ? '删除中...' : '删除' }}
            </template>
        </BaseModal>
    `
});
