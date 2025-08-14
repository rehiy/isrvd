// 重命名模态框组件

const { defineComponent, inject, reactive, ref } = Vue;

import { APP_ACTIONS_KEY } from '../../helpers/state.js';
import { BaseModal } from '../modal-base.js';

export const RenameModal = defineComponent({
    name: 'RenameModal',
    components: { BaseModal },
    setup() {
        const actions = inject(APP_ACTIONS_KEY);

        const formData = reactive({
            oldPath: '',
            newName: '',
            loading: false
        });

        const modalRef = ref(null);

        const show = (file) => {
            formData.oldPath = file.path;
            formData.newName = file.name;
            formData.loading = false;
            modalRef.value.show();
        };

        const handleConfirm = async () => {
            if (!formData.newName.trim()) return;

            formData.loading = true;

            try {
                await axios.post('/api/rename', {
                    oldPath: formData.oldPath,
                    newName: formData.newName
                });

                actions.showSuccess('重命名成功');
                actions.loadFiles();
                modalRef.value.hide();
            } catch (error) {
                actions.showError(error.response?.data?.error || '重命名失败');
            } finally {
                formData.loading = false;
            }
        };

        return {
            formData,
            show,
            handleConfirm,
            modalRef
        };
    },
    template: `
        <BaseModal
            ref="modalRef"
            id="renameModal"
            title="重命名"
            :loading="formData.loading"
            :confirm-disabled="!formData.newName.trim()"
            @confirm="handleConfirm"
        >
            <form @submit.prevent="handleConfirm">
                <div class="mb-3">
                    <label for="newName" class="form-label">新名称</label>
                    <input type="text" class="form-control" id="newName" v-model="formData.newName" :disabled="formData.loading" required>
                </div>
            </form>
            <template #confirm-text>
                {{ formData.loading ? '重命名中...' : '重命名' }}
            </template>
        </BaseModal>
    `
});
