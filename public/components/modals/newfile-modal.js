// 新建文件模态框组件

const { defineComponent, inject, reactive, ref } = Vue;

import { APP_STATE_KEY, APP_ACTIONS_KEY } from '../../helpers/state.js';
import { BaseModal } from '../modal-base.js';

export const NewFileModal = defineComponent({
    name: 'NewFileModal',
    components: { BaseModal },
    setup() {
        const state = inject(APP_STATE_KEY);
        const actions = inject(APP_ACTIONS_KEY);

        const formData = reactive({
            name: '',
            content: '',
            loading: false,
            error: ''
        });

        const modalRef = ref(null);

        const show = () => {
            formData.name = '';
            formData.content = '';
            formData.error = '';
            formData.loading = false;
            modalRef.value.show();
        };

        const handleConfirm = async () => {
            if (!formData.name.trim()) return;

            formData.loading = true;
            formData.error = '';

            try {
                await axios.post('/api/newfile', {
                    path: state.currentPath,
                    name: formData.name,
                    content: formData.content
                });

                actions.showSuccess('文件创建成功');
                actions.loadFiles();
                modalRef.value.hide();

            } catch (error) {
                formData.error = error.response?.data?.error || '创建文件失败';
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
            id="newFileModal"
            title="新建文件"
            size="modal-lg"
            :loading="formData.loading"
            :confirm-disabled="!formData.name.trim()"
            @confirm="handleConfirm"
        >
            <form @submit.prevent="handleConfirm">
                <div class="mb-3">
                    <label for="fileName" class="form-label">文件名称</label>
                    <input type="text" class="form-control" id="fileName" v-model="formData.name" :disabled="formData.loading" required>
                </div>
                <div class="mb-3">
                    <label for="fileContent" class="form-label">文件内容</label>
                    <textarea class="form-control" id="fileContent" rows="10" v-model="formData.content" :disabled="formData.loading"></textarea>
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
