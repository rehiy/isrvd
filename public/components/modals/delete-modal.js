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
            loading: false,
            error: ''
        });

        const modalRef = ref(null);

        const show = (file) => {
            formData.file = file;
            formData.error = '';
            formData.loading = false;
            modalRef.value.show();
        };

        const handleConfirm = async () => {
            if (!formData.file) return;

            formData.loading = true;
            formData.error = '';

            try {
                await axios.delete('/api/delete', {
                    params: { file: formData.file.path }
                });

                actions.showSuccess('删除成功');
                actions.loadFiles();
                modalRef.value.hide();

            } catch (error) {
                formData.error = error.response?.data?.error || '删除失败';
            } finally {
                formData.loading = false;
            }
        };

        const handleCancel = () => {
            formData.error = '';
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
            <template #confirm-text>
                <i class="fas fa-trash me-1"></i>
                删除
            </template>

            <div v-if="formData.error" class="alert alert-danger">
                <i class="fas fa-exclamation-triangle me-2"></i>
                {{ formData.error }}
            </div>

            <div v-if="formData.file" class="text-center">
                <div class="mb-3">
                    <i class="fas fa-exclamation-triangle text-warning" style="font-size: 3rem;"></i>
                </div>
                <p class="mb-3">
                    确定要删除 <strong>{{ formData.file.name }}</strong> 吗？
                </p>
            </div>
        </BaseModal>
    `
});
