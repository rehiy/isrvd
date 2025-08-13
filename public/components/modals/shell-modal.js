// Shell终端模态框组件

const { defineComponent, ref } = Vue;

import * as ShellTerminal from '../../helpers/shell.js';

export const ShellModal = defineComponent({
    name: 'ShellModal',
    setup() {
        const modalRef = ref(null);

        const show = () => {
            const modalEl = document.getElementById('shellModal');
            if (modalEl) {
                const modal = new bootstrap.Modal(modalEl);
                modal.show();

                // 延迟挂载，确保 DOM 已渲染
                setTimeout(() => {
                    const mountPoint = document.getElementById('xterm-container');
                    if (mountPoint) {
                        ShellTerminal.create(mountPoint);
                    }
                }, 200);

                // 监听弹窗关闭，自动卸载 terminal
                if (!modalEl.__shellModalListener) {
                    modalEl.addEventListener('hidden.bs.modal', () => {
                        ShellTerminal.destroy();
                        const mountPoint = document.getElementById('xterm-container');
                        if (mountPoint) {
                            mountPoint.innerHTML = '';
                        }
                    });
                    modalEl.__shellModalListener = true;
                }
            }
        };

        return { show };
    },
    template: `
        <div class="modal fade" id="shellModal" tabindex="-1" aria-labelledby="shellModalLabel" aria-hidden="true">
            <div class="modal-dialog modal-xl modal-dialog-centered">
                <div class="modal-content">
                    <div class="modal-header border-0">
                        <h5 class="modal-title" id="shellModalLabel">
                            <i class="fas fa-terminal"></i> 实时 Shell 终端
                        </h5>
                        <button type="button" class="btn-close btn-close-white" data-bs-dismiss="modal"></button>
                    </div>
                    <div class="modal-body p-0">
                        <div id="xterm-container" style="height:400px;width:100%;background:#222;"></div>
                    </div>
                </div>
            </div>
        </div>
    `
});
