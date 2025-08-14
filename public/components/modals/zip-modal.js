// 压缩目录模态框组件

const { defineComponent, inject, reactive, ref } = Vue;

import { APP_ACTIONS_KEY } from '../../helpers/state.js';
import { BaseModal } from '../modal-base.js';

export const ZipModal = defineComponent({
    name: 'ZipModal',
    components: { BaseModal },
    setup() {
        const actions = inject(APP_ACTIONS_KEY);

        const formData = reactive({
            path: '',
            name: '',
            zipName: '',
            loading: false
        });

        const modalRef = ref(null);

        const show = (file) => {
            formData.path = file.path;
            formData.name = file.name;
            formData.zipName = file.name + '.zip';
            formData.loading = false;
            modalRef.value.show();
        };

        const handleConfirm = async () => {
            if (!formData.zipName.trim()) return;

            formData.loading = true;

            try {
                await axios.post('/api/zip', {
                    path: formData.path,
                    zipName: formData.zipName
                });

                actions.showSuccess('压缩成功');
                actions.loadFiles();
                modalRef.value.hide();
            } catch (error) {
                actions.showError(error.response?.data?.error || '压缩失败');
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
            id="zipModal"
            title="压缩目录"
            :loading="formData.loading"
            :confirm-disabled="!formData.zipName.trim()"
            @confirm="handleConfirm"
        >
            <form @submit.prevent="handleConfirm">
                <div class="mb-3">
                    <label for="zipName" class="form-label">压缩包名称</label>
                    <input type="text" class="form-control" id="zipName" v-model="formData.zipName" :disabled="formData.loading" required>
                </div>
            </form>
            <template #confirm-text>
                {{ formData.loading ? '压缩中...' : '压缩' }}
            </template>
        </BaseModal>
    `
});
