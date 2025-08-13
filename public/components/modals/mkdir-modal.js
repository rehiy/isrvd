// 新建目录模态框组件

const { defineComponent, inject, reactive, ref } = Vue;

import { APP_STATE_KEY, APP_ACTIONS_KEY } from '../../helpers/state.js';
import { BaseModal } from '../modal-base.js';

export const MkdirModal = defineComponent({
    name: 'MkdirModal',
    components: { BaseModal },
    setup() {
        const state = inject(APP_STATE_KEY);
        const actions = inject(APP_ACTIONS_KEY);

        const formData = reactive({
            name: '',
            loading: false,
            error: ''
        });

        const modalRef = ref(null);

        const show = () => {
            formData.name = '';
            formData.error = '';
            formData.loading = false;
            modalRef.value.show();
        };

        const handleConfirm = async () => {
            if (!formData.name.trim()) return;

            formData.loading = true;
            formData.error = '';

            try {
                await axios.post('/api/mkdir', {
                    path: state.currentPath,
                    name: formData.name
                });

                actions.showSuccess('目录创建成功');
                actions.loadFiles();
                modalRef.value.hide();

            } catch (error) {
                formData.error = error.response?.data?.error || '创建目录失败';
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
            id="mkdirModal"
            title="新建目录"
            :loading="formData.loading"
            :confirm-disabled="!formData.name.trim()"
            @confirm="handleConfirm"
        >
            <form @submit.prevent="handleConfirm">
                <div class="mb-3">
                    <label for="dirName" class="form-label">目录名称</label>
                    <input type="text" class="form-control" id="dirName" v-model="formData.name" :disabled="formData.loading" required>
                </div>
                <div v-if="formData.error" class="alert alert-danger">
                    {{ formData.error }}
                </div>
            </form>
            <template #confirm-text>
                {{ formData.loading ? '创建中...' : '创建' }}
            </template>
        </BaseModal>
    `
});
