// ==================== 状态管理 ====================

const { reactive } = Vue;

// Provide/Inject keys
export const APP_STATE_KEY = Symbol('appState');
export const APP_ACTIONS_KEY = Symbol('appActions');

// 全局状态管理
export const createAppState = () => {
    return reactive({
        // 用户认证状态
        user: null,
        token: null,

        // 文件管理状态
        loading: false,
        currentPath: '/',
        files: [],

        // 通知状态
        notification: {
            type: null, // 'success' | 'error' | null
            message: '',
            timer: null
        }
    });
};

// 全局操作方法
export const createAppActions = (state) => {
    return {
        // 认证操作
        setAuth(userData) {
            state.user = userData.user;
            state.token = userData.token;
            localStorage.setItem('file-manager-token', userData.token);
            localStorage.setItem('file-manager-user', userData.user);
            if (state.token) {
                axios.defaults.headers.common['Authorization'] = state.token;
            }
        },

        clearAuth() {
            state.user = null;
            state.token = null;
            localStorage.removeItem('file-manager-token');
            localStorage.removeItem('file-manager-user');
            delete axios.defaults.headers.common['Authorization'];
        },

        // 文件操作
        async loadFiles(path = state.currentPath) {
            state.loading = true;
            try {
                const response = await axios.get('/api/files', {
                    params: { path }
                });
                state.files = response.data.files || [];
                state.currentPath = response.data.path;
            } catch (error) {
                this.showError(error.response?.data?.error || '加载文件列表失败');
                if (error.response?.status === 401) {
                    this.clearAuth();
                }
            } finally {
               state.loading = false;
            }
        },

        // 通知操作
        showNotification(type, message) {
            if (state.notification.timer) {
                clearTimeout(state.notification.timer);
            }

            state.notification.type = type;
            state.notification.message = message;

            const duration = type === 'error' ? 5000 : 3000;
            state.notification.timer = setTimeout(() => {
                this.clearNotification();
            }, duration);
        },

        clearNotification() {
            if (state.notification.timer) {
                clearTimeout(state.notification.timer);
                state.notification.timer = null;
            }
            state.notification.type = null;
            state.notification.message = '';
        },

        showSuccess(message) {
            this.showNotification('success', message);
        },

        showError(message) {
            this.showNotification('error', message);
        },
    };
};
