// ==================== 通知管理组件 ====================

const { defineComponent, inject } = Vue;

import { APP_STATE_KEY, APP_ACTIONS_KEY } from '../helpers/state.js';

// 通知管理器组件
export const NotificationManager = defineComponent({
    name: 'NotificationManager',
    setup() {
        const state = inject(APP_STATE_KEY);
        const actions = inject(APP_ACTIONS_KEY);

        return { state, actions };
    },
    template: `
        <Teleport to="body">
            <div v-if="state.notification.type" 
                 class="position-fixed top-0 start-50 translate-middle-x mt-3"
                 style="z-index: 1060;">
                <div :class="[
                    'alert alert-dismissible fade show',
                    state.notification.type === 'error' ? 'alert-danger' : 'alert-success'
                ]">
                    {{ state.notification.message }}
                    <button type="button" class="btn-close" @click="actions.clearNotification()"></button>
                </div>
            </div>
        </Teleport>
    `
});