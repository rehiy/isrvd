// 解压确认模态框组件

const { defineComponent, inject, reactive, ref } = Vue;

import { APP_STATE_KEY, APP_ACTIONS_KEY } from '../../helpers/state.js';
import { BaseModal } from '../modal-base.js';

export const UnzipModal = defineComponent({
    name: 'UnzipModal',
    components: { BaseModal },
    setup() {
        const state = inject(APP_STATE_KEY);
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
                await axios.post('/api/unzip', {
                    path: state.currentPath,
                    zipName: formData.file.name
                });

                actions.showSuccess('解压成功');
                actions.loadFiles();
                modalRef.value.hide();
            } catch (error) {
                actions.showError(error.response?.data?.error || '解压失败');
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
            id="unzipModal"
            title="解压确认"
            headerClass="bg-warning text-dark"
            :loading="formData.loading"
            :confirm-disabled="!formData.file"
            @confirm="handleConfirm"
            @cancel="handleCancel"
        >
            <div v-if="formData.file" class="text-center">
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
    `
});
