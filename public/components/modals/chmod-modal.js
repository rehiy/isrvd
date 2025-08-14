// 权限修改模态框组件

const { defineComponent, inject, reactive, ref } = Vue;

import { APP_ACTIONS_KEY } from '../../helpers/state.js';
import { BaseModal } from '../modal-base.js';

export const ChmodModal = defineComponent({
    name: 'ChmodModal',
    components: { BaseModal },
    setup() {
        const actions = inject(APP_ACTIONS_KEY);

        const formData = reactive({
            path: '',
            mode: '',
            loading: false
        });

        const modalRef = ref(null);

        const show = async (file) => {
            formData.loading = true;

            try {
                const response = await axios.get('/api/chmod', {
                    params: { file: file.path }
                });

                formData.path = file.path;
                formData.mode = response.data.mode;

                modalRef.value.show();
            } catch (error) {
                actions.showError(error.response?.data?.error || '无法获取文件权限');
            } finally {
                formData.loading = false;
            }
        };

        const handleConfirm = async () => {
            if (!formData.mode.trim()) return;

            formData.loading = true;

            try {
                await axios.post('/api/chmod', {
                    mode: formData.mode
                }, {
                    params: { file: formData.path }
                });

                actions.showSuccess('权限修改成功');
                actions.loadFiles();
                modalRef.value.hide();
            } catch (error) {
                actions.showError(error.response?.data?.error || '修改权限失败');
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
            id="chmodModal"
            title="修改权限"
            :loading="formData.loading"
            :confirm-disabled="!formData.mode.trim()"
            @confirm="handleConfirm"
        >
            <form @submit.prevent="handleConfirm">
                <div class="mb-3">
                    <label for="fileMode" class="form-label">权限 (八进制)</label>
                    <input type="text" class="form-control" id="fileMode" v-model="formData.mode" :disabled="formData.loading" required placeholder="755">
                    <div class="form-text">
                        常用权限: 755 (rwxr-xr-x), 644 (rw-r--r--), 777 (rwxrwxrwx)
                    </div>
                </div>
            </form>
            <template #confirm-text>
                {{ formData.loading ? '修改中...' : '修改' }}
            </template>
        </BaseModal>
    `
});
