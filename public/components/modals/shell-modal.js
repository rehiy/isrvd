// Shell终端模态框组件

const { defineComponent, ref } = Vue;

import * as ShellTerminal from '../../helpers/shell.js';
import { BaseModal } from '../modal-base.js';

export const ShellModal = defineComponent({
    name: 'ShellModal',
    components: { BaseModal },
    setup() {
        const modalRef = ref(null);

        const show = () => {
            modalRef.value.show();
        };

        const hide = () => {
            modalRef.value.hide();
        };

        const handleShown = () => {
            const mountPoint = document.getElementById('xterm-container');
            if (mountPoint) {
                ShellTerminal.create(mountPoint);
            }
        };

        const handleHidden = () => {
            ShellTerminal.destroy();
            const mountPoint = document.getElementById('xterm-container');
            if (mountPoint) {
                mountPoint.innerHTML = '';
            }
        };

        return {
            show,
            hide,
            modalRef,
            handleShown,
            handleHidden
        };
    },
    template: `
        <BaseModal
            ref="modalRef"
            id="shellModal"
            title="实时 Shell 终端"
            size="modal-xl"
            :show-footer="false"
            body-class="bg-dark pe-0"
            @shown="handleShown"
            @hidden="handleHidden"
        >
            <template #title>
                <i class="fas fa-terminal"></i> 实时 Shell 终端
            </template>
            <div id="xterm-container"></div>
        </BaseModal>
    `
});
