// ==================== 模态框基础组件 ====================

const { defineComponent, onMounted } = Vue;

// 模态框基础组件
export const BaseModal = defineComponent({
    name: 'BaseModal',
    props: {
        id: { type: String, required: true },
        title: { type: String, required: true },
        size: { type: String, default: '' }, // '' | 'modal-lg' | 'modal-xl'
        loading: { type: Boolean, default: false },
        confirmDisabled: { type: Boolean, default: false }
    },
    emits: ['confirm', 'cancel', 'shown', 'hidden'],
    setup(props, { emit, expose }) {
        let modalInstance = null;

        const show = () => {
            const modalEl = document.getElementById(props.id);
            if (modalEl) {
                modalInstance = new bootstrap.Modal(modalEl);
                modalInstance.show();
                emit('shown');
            }
        };

        const hide = () => {
            if (modalInstance) {
                modalInstance.hide();
            }
        };

        const handleConfirm = () => {
            emit('confirm');
        };

        const handleCancel = () => {
            emit('cancel');
            hide();
        };

        onMounted(() => {
            const modalEl = document.getElementById(props.id);
            if (modalEl) {
                modalEl.addEventListener('hidden.bs.modal', () => {
                    emit('hidden');
                });
            }
        });

        expose({ show, hide });

        return {
            handleConfirm,
            handleCancel
        };
    },
    template: `
        <div class="modal fade" :id="id" tabindex="-1">
            <div class="modal-dialog" :class="size">
                <div class="modal-content">
                    <div class="modal-header">
                        <h5 class="modal-title">{{ title }}</h5>
                        <button type="button" class="btn-close" data-bs-dismiss="modal" :disabled="loading"></button>
                    </div>
                    <div class="modal-body">
                        <slot></slot>
                    </div>
                    <div class="modal-footer">
                        <button type="button" class="btn btn-secondary" @click="handleCancel" :disabled="loading">
                            <slot name="cancel-text">取消</slot>
                        </button>
                        <button type="button" class="btn btn-primary" @click="handleConfirm" :disabled="loading || confirmDisabled">
                            <i class="fas fa-spinner fa-spin" v-if="loading"></i>
                            <slot name="confirm-text">确认</slot>
                        </button>
                    </div>
                </div>
            </div>
        </div>
    `
});
