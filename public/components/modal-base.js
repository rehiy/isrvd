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
        centered: { type: Boolean, default: true },
        headerClass: { type: String, default: '' },
        bodyClass: { type: String, default: '' },
        showFooter: { type: Boolean, default: true },
        confirmDisabled: { type: Boolean, default: false },
    },
    emits: ['confirm', 'cancel', 'shown', 'hidden'],
    setup(props, { emit, expose }) {
        let modalInstance = null;

        const show = () => {
            const el = document.getElementById(props.id);
            if (el) {
                modalInstance = new bootstrap.Modal(el);
                modalInstance.show();
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
            const el = document.getElementById(props.id);
            if (el) {
                el.addEventListener('shown.bs.modal', () => {
                    emit('shown');
                });
                el.addEventListener('hidden.bs.modal', () => {
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
            <div class="modal-dialog" :class="[size, { 'modal-dialog-centered': centered }]">
                <div class="modal-content">
                    <div class="modal-header" :class="headerClass">
                        <h5 class="modal-title">
                            <slot name="title">{{ title }}</slot>
                        </h5>
                        <button type="button" class="btn-close" data-bs-dismiss="modal" :disabled="loading"></button>
                    </div>
                    <div class="modal-body" :class="bodyClass">
                        <slot></slot>
                    </div>
                    <div class="modal-footer" v-if="showFooter">
                        <slot name="footer">
                            <button type="button" class="btn btn-secondary" @click="handleCancel" :disabled="loading">
                                <slot name="cancel-text">取消</slot>
                            </button>
                            <button type="button" class="btn btn-primary" @click="handleConfirm" :disabled="loading || confirmDisabled">
                                <i class="fas fa-spinner fa-spin" v-if="loading"></i>
                                <slot name="confirm-text">确认</slot>
                            </button>
                        </slot>
                    </div>
                </div>
            </div>
        </div>
    `
});
