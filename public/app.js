// ==================== 主应用模块化入口 ====================

const { createApp, defineComponent, provide, onMounted } = Vue;

import { createAppState, createAppActions, APP_STATE_KEY, APP_ACTIONS_KEY } from './helpers/state.js';
import { LoginForm } from './components/auth.js';
import { NotificationManager } from './components/notification.js';
import { NavigationBar, BreadcrumbNav, ActionButtons } from './components/navigation.js';
import { FileIndex } from './components/file-index.js';

// 主应用组件
const FilerApp = defineComponent({
    name: 'FilerApp',
    components: {
        NavigationBar,
        LoginForm,
        BreadcrumbNav,
        ActionButtons,
        FileIndex,
        NotificationManager
    },
    setup() {
        const state = createAppState();
        const actions = createAppActions(state);

        provide(APP_STATE_KEY, state);
        provide(APP_ACTIONS_KEY, actions);

        onMounted(() => {
            // 检查本地存储的认证信息
            const savedToken = localStorage.getItem('file-manager-token');
            const savedUser = localStorage.getItem('file-manager-user');

            if (savedToken && savedUser) {
                actions.setAuth({ token: savedToken, user: savedUser });
                // 认证状态恢复后立即加载文件
                actions.loadFiles();
            }
        });

        return { state, actions };
    },
    template: `
        <template v-if="state.user">
            <NavigationBar />
            <div class="container-fluid">
                <BreadcrumbNav />
                <ActionButtons />
                <FileIndex />
            </div>
        </template>

        <LoginForm v-else />

        <NotificationManager />
    `
});

// 创建 Vue 应用并挂载
createApp(FilerApp).mount('#app');
