// 编辑文件模态框组件

const { defineComponent, inject, reactive, ref } = Vue;

import { APP_ACTIONS_KEY } from '../../helpers/state.js';
import { BaseModal } from '../modal-base.js';

export const EditModal = defineComponent({
    name: 'EditModal',
    components: { BaseModal },
    setup() {
        const actions = inject(APP_ACTIONS_KEY);

        const formData = reactive({
            filename: '',
            content: '',
            filePath: '',
            loading: false
        });

        const modalRef = ref(null);

        const show = async (file) => {
            formData.loading = true;

            try {
                const response = await axios.get('/api/edit', {
                    params: { file: file.path }
                });

                formData.filePath = file.path;
                formData.filename = file.name;
                formData.content = response.data.content;

                modalRef.value.show();

            } catch (error) {
                actions.showError(error.response?.data?.error || '无法打开文件');
            } finally {
                formData.loading = false;
            }
        };

        const handleConfirm = async () => {
            formData.loading = true;

            try {
                await axios.post('/api/edit', {
                    content: formData.content
                }, {
                    params: { file: formData.filePath }
                });

                actions.showSuccess('文件保存成功');
                modalRef.value.hide();

            } catch (error) {
                actions.showError(error.response?.data?.error || '保存文件失败');
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
            id="editModal"
            :title="'编辑文件: ' + formData.filename"
            size="modal-xl"
            :loading="formData.loading"
            @confirm="handleConfirm"
        >
            <textarea class="form-control" rows="20" v-model="formData.content" :disabled="formData.loading"></textarea>
            <template #confirm-text>
                {{ formData.loading ? '保存中...' : '保存' }}
            </template>
        </BaseModal>
    `
});
