const { createApp, defineComponent, reactive, provide, inject, ref, computed, onMounted, onUnmounted } = Vue;

// ==================== 配置信息 ====================

const TEXT_EXTENSIONS = [
    'txt', 'md', 'js', 'css', 'html', 'htm', 'json', 'xml', 'csv',
    'log', 'conf', 'ini', 'cfg', 'yaml', 'yml', 'php', 'py', 'go',
    'java', 'cpp', 'c', 'h', 'sql', 'sh', 'bat', 'env'
];

const FILE_ICON_MAP = {
    'txt': 'fas fa-file-alt text-secondary',
    'md': 'fab fa-markdown text-dark',
    'pdf': 'fas fa-file-pdf text-danger',
    'doc': 'fas fa-file-word text-primary',
    'docx': 'fas fa-file-word text-primary',
    'xls': 'fas fa-file-excel text-success',
    'xlsx': 'fas fa-file-excel text-success',
    'ppt': 'fas fa-file-powerpoint text-warning',
    'pptx': 'fas fa-file-powerpoint text-warning',
    'zip': 'fas fa-file-archive text-warning',
    'rar': 'fas fa-file-archive text-warning',
    '7z': 'fas fa-file-archive text-warning',
    'tar': 'fas fa-file-archive text-warning',
    'gz': 'fas fa-file-archive text-warning',
    'jpg': 'fas fa-file-image text-info',
    'jpeg': 'fas fa-file-image text-info',
    'png': 'fas fa-file-image text-info',
    'gif': 'fas fa-file-image text-info',
    'bmp': 'fas fa-file-image text-info',
    'svg': 'fas fa-file-image text-info',
    'mp3': 'fas fa-file-audio text-success',
    'wav': 'fas fa-file-audio text-success',
    'mp4': 'fas fa-file-video text-danger',
    'avi': 'fas fa-file-video text-danger',
    'mov': 'fas fa-file-video text-danger',
    'js': 'fab fa-js-square text-warning',
    'html': 'fab fa-html5 text-danger',
    'css': 'fab fa-css3-alt text-primary',
    'php': 'fab fa-php text-purple',
    'py': 'fab fa-python text-info',
    'java': 'fab fa-java text-danger',
    'cpp': 'fas fa-file-code text-info',
    'c': 'fas fa-file-code text-info',
    'go': 'fas fa-file-code text-primary',
    'sql': 'fas fa-database text-secondary'
};

// ==================== 状态管理 ====================

// Provide/Inject keys
const APP_STATE_KEY = Symbol('appState');
const APP_ACTIONS_KEY = Symbol('appActions');

// 全局状态管理
const createAppState = () => {
    return reactive({
        // 用户认证状态
        user: null,
        token: null,

        // 文件管理状态
        currentPath: '/',
        files: [],
        loading: false,

        // 通知状态
        notification: {
            type: null, // 'success' | 'error' | null
            message: '',
            timer: null
        }
    });
};

// 全局操作方法
const createAppActions = (state) => {
    return {
        // 用户认证操作
        setAuth(userData) {
            state.user = userData.user;
            state.token = userData.token;
            localStorage.setItem('fileManagerToken', userData.token);
            localStorage.setItem('fileManagerUser', userData.user);
            this.setupAxios();
        },

        clearAuth() {
            state.user = null;
            state.token = null;
            localStorage.removeItem('fileManagerToken');
            localStorage.removeItem('fileManagerUser');
            delete axios.defaults.headers.common['Authorization'];
        },

        setupAxios() {
            if (state.token) {
                axios.defaults.headers.common['Authorization'] = state.token;
            }
        },

        // 路径导航操作
        setPath(path) {
            state.currentPath = path;
        },

        setFiles(files) {
            state.files = files || [];
        },

        setLoading(loading) {
            state.loading = loading;
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

        // 文件操作
        async loadFiles(path = state.currentPath) {
            this.setLoading(true);
            try {
                const response = await axios.get('/api/files', {
                    params: { path }
                });
                this.setFiles(response.data.files);
                this.setPath(response.data.path);
            } catch (error) {
                this.showError(error.response?.data?.error || '加载文件列表失败');
                if (error.response?.status === 401) {
                    this.clearAuth();
                }
            } finally {
                this.setLoading(false);
            }
        },

        async deleteFile(file) {
            try {
                await axios.delete('/api/delete', {
                    params: { file: file.path }
                });
                this.showSuccess('删除成功');
                this.loadFiles();
            } catch (error) {
                this.showError(error.response?.data?.error || '删除失败');
            }
        },

        async unzipFile(file) {
            try {
                await axios.post('/api/unzip', {
                    path: state.currentPath,
                    zipName: file.name
                });
                this.showSuccess('解压成功');
                this.loadFiles();
            } catch (error) {
                this.showError(error.response?.data?.error || '解压失败');
            }
        }
    };
};

// ==================== 通用组件 ====================

// 通知管理器组件
const NotificationManager = defineComponent({
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

// 模态框基础组件
const BaseModal = defineComponent({
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

// ==================== 模态组件 ====================

// 新建目录模态框组件
const MkdirModal = defineComponent({
    name: 'MkdirModal',
    components: { BaseModal },
    setup() {
        const state = inject(APP_STATE_KEY);
        const actions = inject(APP_ACTIONS_KEY);

        const formData = reactive({
            name: '',
            loading: false,
            error: ''
        });

        const modalRef = ref(null);

        const show = () => {
            formData.name = '';
            formData.error = '';
            formData.loading = false;
            modalRef.value.show();
        };

        const handleConfirm = async () => {
            if (!formData.name.trim()) return;

            formData.loading = true;
            formData.error = '';

            try {
                await axios.post('/api/mkdir', {
                    path: state.currentPath,
                    name: formData.name
                });

                actions.showSuccess('目录创建成功');
                actions.loadFiles();
                modalRef.value.hide();

            } catch (error) {
                formData.error = error.response?.data?.error || '创建目录失败';
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
            id="mkdirModal"
            title="新建目录"
            :loading="formData.loading"
            :confirm-disabled="!formData.name.trim()"
            @confirm="handleConfirm"
        >
            <form @submit.prevent="handleConfirm">
                <div class="mb-3">
                    <label for="dirName" class="form-label">目录名称</label>
                    <input type="text" class="form-control" id="dirName" v-model="formData.name" :disabled="formData.loading" required>
                </div>
                <div v-if="formData.error" class="alert alert-danger">
                    {{ formData.error }}
                </div>
            </form>
            <template #confirm-text>
                {{ formData.loading ? '创建中...' : '创建' }}
            </template>
        </BaseModal>
    `
});

// 新建文件模态框组件
const NewFileModal = defineComponent({
    name: 'NewFileModal',
    components: { BaseModal },
    setup() {
        const state = inject(APP_STATE_KEY);
        const actions = inject(APP_ACTIONS_KEY);

        const formData = reactive({
            name: '',
            content: '',
            loading: false,
            error: ''
        });

        const modalRef = ref(null);

        const show = () => {
            formData.name = '';
            formData.content = '';
            formData.error = '';
            formData.loading = false;
            modalRef.value.show();
        };

        const handleConfirm = async () => {
            if (!formData.name.trim()) return;

            formData.loading = true;
            formData.error = '';

            try {
                await axios.post('/api/newfile', {
                    path: state.currentPath,
                    name: formData.name,
                    content: formData.content
                });

                actions.showSuccess('文件创建成功');
                actions.loadFiles();
                modalRef.value.hide();

            } catch (error) {
                formData.error = error.response?.data?.error || '创建文件失败';
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
            id="newFileModal"
            title="新建文件"
            size="modal-lg"
            :loading="formData.loading"
            :confirm-disabled="!formData.name.trim()"
            @confirm="handleConfirm"
        >
            <form @submit.prevent="handleConfirm">
                <div class="mb-3">
                    <label for="fileName" class="form-label">文件名称</label>
                    <input type="text" class="form-control" id="fileName" v-model="formData.name" :disabled="formData.loading" required>
                </div>
                <div class="mb-3">
                    <label for="fileContent" class="form-label">文件内容</label>
                    <textarea class="form-control" id="fileContent" rows="10" v-model="formData.content" :disabled="formData.loading"></textarea>
                </div>
                <div v-if="formData.error" class="alert alert-danger">
                    {{ formData.error }}
                </div>
            </form>
            <template #confirm-text>
                {{ formData.loading ? '创建中...' : '创建' }}
            </template>
        </BaseModal>
    `
});

// 上传文件模态框组件
const UploadModal = defineComponent({
    name: 'UploadModal',
    components: { BaseModal },
    setup() {
        const state = inject(APP_STATE_KEY);
        const actions = inject(APP_ACTIONS_KEY);

        const formData = reactive({
            loading: false,
            error: '',
            selectedFile: null
        });

        const fileInput = ref(null);
        const modalRef = ref(null);

        const show = () => {
            formData.error = '';
            formData.loading = false;
            formData.selectedFile = null;
            modalRef.value.show();
        };

        const handleFileChange = (event) => {
            formData.selectedFile = event.target.files[0] || null;
        };

        const handleConfirm = async () => {
            if (!formData.selectedFile) return;

            formData.loading = true;
            formData.error = '';

            const formDataToSend = new FormData();
            formDataToSend.append('file', formData.selectedFile);
            formDataToSend.append('path', state.currentPath);

            try {
                await axios.post('/api/upload', formDataToSend, {
                    headers: {
                        'Content-Type': 'multipart/form-data'
                    }
                });

                actions.showSuccess('文件上传成功');
                actions.loadFiles();
                formData.selectedFile = null;
                if (fileInput.value) {
                    fileInput.value.value = '';
                }
                modalRef.value.hide();

            } catch (error) {
                formData.error = error.response?.data?.error || '上传文件失败';
            } finally {
                formData.loading = false;
            }
        };

        const hasFile = computed(() => {
            return formData.selectedFile !== null;
        });

        return {
            formData,
            fileInput,
            show,
            handleConfirm,
            handleFileChange,
            modalRef,
            hasFile
        };
    },
    template: `
        <BaseModal
            ref="modalRef"
            id="uploadModal"
            title="上传文件"
            :loading="formData.loading"
            :confirm-disabled="!hasFile"
            @confirm="handleConfirm"
        >
            <form @submit.prevent="handleConfirm">
                <div class="mb-3">
                    <label for="uploadFile" class="form-label">选择文件</label>
                    <input type="file" class="form-control" id="uploadFile" ref="fileInput" @change="handleFileChange" :disabled="formData.loading" required>
                </div>
                <div v-if="formData.error" class="alert alert-danger">
                    {{ formData.error }}
                </div>
            </form>
            <template #confirm-text>
                {{ formData.loading ? '上传中...' : '上传' }}
            </template>
        </BaseModal>
    `
});

// Shell终端模态框组件
const ShellModal = defineComponent({
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
                    if (mountPoint && window.ShellTerminalManager) {
                        ShellTerminalManager.create(mountPoint);
                    }
                }, 200);

                // 监听弹窗关闭，自动卸载 terminal
                if (!modalEl.__shellModalListener) {
                    modalEl.addEventListener('hidden.bs.modal', () => {
                        if (window.ShellTerminalManager) {
                            ShellTerminalManager.destroy();
                        }
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

// 编辑文件模态框组件
const EditModal = defineComponent({
    name: 'EditModal',
    components: { BaseModal },
    setup() {
        const actions = inject(APP_ACTIONS_KEY);

        const formData = reactive({
            filename: '',
            content: '',
            filePath: '',
            loading: false,
            error: ''
        });

        const modalRef = ref(null);

        const show = async (file) => {
            formData.error = '';
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
            formData.error = '';

            try {
                await axios.post('/api/edit', {
                    content: formData.content
                }, {
                    params: { file: formData.filePath }
                });

                actions.showSuccess('文件保存成功');
                modalRef.value.hide();

            } catch (error) {
                formData.error = error.response?.data?.error || '保存文件失败';
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
            <textarea class="form-control" rows="20" v-model="formData.content" :disabled="formData.loading" style="font-family: 'Courier New', monospace;"></textarea>
            <div v-if="formData.error" class="alert alert-danger mt-3">
                {{ formData.error }}
            </div>
            <template #confirm-text>
                {{ formData.loading ? '保存中...' : '保存' }}
            </template>
        </BaseModal>
    `
});

// 重命名模态框组件
const RenameModal = defineComponent({
    name: 'RenameModal',
    components: { BaseModal },
    setup() {
        const actions = inject(APP_ACTIONS_KEY);

        const formData = reactive({
            oldPath: '',
            newName: '',
            loading: false,
            error: ''
        });

        const modalRef = ref(null);

        const show = (file) => {
            formData.oldPath = file.path;
            formData.newName = file.name;
            formData.error = '';
            formData.loading = false;
            modalRef.value.show();
        };

        const handleConfirm = async () => {
            if (!formData.newName.trim()) return;

            formData.loading = true;
            formData.error = '';

            try {
                await axios.post('/api/rename', {
                    oldPath: formData.oldPath,
                    newName: formData.newName
                });

                actions.showSuccess('重命名成功');
                actions.loadFiles();
                modalRef.value.hide();

            } catch (error) {
                formData.error = error.response?.data?.error || '重命名失败';
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
            id="renameModal"
            title="重命名"
            :loading="formData.loading"
            :confirm-disabled="!formData.newName.trim()"
            @confirm="handleConfirm"
        >
            <form @submit.prevent="handleConfirm">
                <div class="mb-3">
                    <label for="newName" class="form-label">新名称</label>
                    <input type="text" class="form-control" id="newName" v-model="formData.newName" :disabled="formData.loading" required>
                </div>
                <div v-if="formData.error" class="alert alert-danger">
                    {{ formData.error }}
                </div>
            </form>
            <template #confirm-text>
                {{ formData.loading ? '重命名中...' : '重命名' }}
            </template>
        </BaseModal>
    `
});

// 权限修改模态框组件
const ChmodModal = defineComponent({
    name: 'ChmodModal',
    components: { BaseModal },
    setup() {
        const actions = inject(APP_ACTIONS_KEY);

        const formData = reactive({
            path: '',
            mode: '',
            loading: false,
            error: ''
        });

        const modalRef = ref(null);

        const show = async (file) => {
            formData.error = '';
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
            formData.error = '';

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
                formData.error = error.response?.data?.error || '修改权限失败';
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
                <div v-if="formData.error" class="alert alert-danger">
                    {{ formData.error }}
                </div>
            </form>
            <template #confirm-text>
                {{ formData.loading ? '修改中...' : '修改' }}
            </template>
        </BaseModal>
    `
});

// 压缩目录模态框组件
const ZipModal = defineComponent({
    name: 'ZipModal',
    components: { BaseModal },
    setup() {
        const actions = inject(APP_ACTIONS_KEY);

        const formData = reactive({
            path: '',
            name: '',
            zipName: '',
            loading: false,
            error: ''
        });

        const modalRef = ref(null);

        const show = (file) => {
            formData.path = file.path;
            formData.name = file.name;
            formData.zipName = file.name + '.zip';
            formData.error = '';
            formData.loading = false;
            modalRef.value.show();
        };

        const handleConfirm = async () => {
            if (!formData.zipName.trim()) return;

            formData.loading = true;
            formData.error = '';

            try {
                await axios.post('/api/zip', {
                    path: formData.path,
                    zipName: formData.zipName
                });

                actions.showSuccess('压缩成功');
                actions.loadFiles();
                modalRef.value.hide();

            } catch (error) {
                formData.error = error.response?.data?.error || '压缩失败';
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
                <div v-if="formData.error" class="alert alert-danger">
                    {{ formData.error }}
                </div>
            </form>
            <template #confirm-text>
                {{ formData.loading ? '压缩中...' : '压缩' }}
            </template>
        </BaseModal>
    `
});

// ==================== 功能组件 ====================

// 登录组件
const LoginForm = defineComponent({
    name: 'LoginForm',
    setup() {
        const actions = inject(APP_ACTIONS_KEY);

        const loginForm = reactive({
            username: '',
            password: ''
        });

        const loading = ref(false);
        const error = ref('');

        const handleLogin = async () => {
            loading.value = true;
            error.value = '';

            try {
                const response = await axios.post('/api/login', loginForm);

                const token = response.data.token;
                const user = response.data.user;

                actions.setAuth({ token, user });

                // 登录成功后加载文件
                actions.loadFiles();

                // 清空表单
                loginForm.username = '';
                loginForm.password = '';

            } catch (err) {
                error.value = err.response?.data?.error || '登录失败';
            } finally {
                loading.value = false;
            }
        };

        return {
            loginForm,
            loading,
            error,
            handleLogin
        };
    },
    template: `
        <div class="container">
            <div class="row justify-content-center">
                <div class="col-md-6">
                    <div class="card">
                        <div class="card-header">
                            <h5 class="mb-0">
                                <i class="fas fa-sign-in-alt"></i> 用户登录
                            </h5>
                        </div>
                        <div class="card-body">
                            <form @submit.prevent="handleLogin">
                                <div class="mb-3">
                                    <label for="username" class="form-label">用户名</label>
                                    <input type="text" class="form-control" id="username" v-model="loginForm.username" :disabled="loading" required>
                                </div>
                                <div class="mb-3">
                                    <label for="password" class="form-label">密码</label>
                                    <input type="password" class="form-control" id="password" v-model="loginForm.password" :disabled="loading" required>
                                </div>
                                <button type="submit" class="btn btn-primary w-100" :disabled="loading">
                                    <i class="fas fa-spinner fa-spin" v-if="loading"></i>
                                    {{ loading ? '登录中...' : '登录' }}
                                </button>
                            </form>
                            <div v-if="error" class="alert alert-danger mt-3">
                                {{ error }}
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    `
});

// 面包屑导航组件
const BreadcrumbNav = defineComponent({
    name: 'BreadcrumbNav',
    setup() {
        const state = inject(APP_STATE_KEY);
        const actions = inject(APP_ACTIONS_KEY);

        const pathParts = computed(() => {
            if (state.currentPath === '/') return [];
            return state.currentPath.split('/').filter(part => part);
        });

        const navigateTo = (path) => {
            actions.loadFiles(path);
        };

        return {
            state,
            pathParts,
            navigateTo
        };
    },
    template: `
        <nav aria-label="breadcrumb">
            <ol class="breadcrumb">
                <li class="breadcrumb-item">
                    <a href="#" @click="navigateTo('/')">
                        <i class="fas fa-home"></i> 首页
                    </a>
                </li>
                <li v-for="(part, index) in pathParts" :key="index" class="breadcrumb-item" :class="{ active: index === pathParts.length - 1 }">
                    <a v-if="index < pathParts.length - 1" href="#" @click="navigateTo('/' + pathParts.slice(0, index + 1).join('/'))">
                        {{ part }}
                    </a>
                    <span v-else>{{ part }}</span>
                </li>
            </ol>
        </nav>
    `
});

// 导航栏组件
const NavigationBar = defineComponent({
    name: 'NavigationBar',
    setup() {
        const state = inject(APP_STATE_KEY);
        const actions = inject(APP_ACTIONS_KEY);

        const loading = ref(false);

        const goHome = () => {
            actions.loadFiles('/');
        };

        const handleLogout = async () => {
            loading.value = true;

            try {
                await axios.post('/api/logout');
            } catch (error) {
                console.warn('Logout request failed:', error);
            }

            actions.clearAuth();
            loading.value = false;
        };

        return {
            state,
            loading,
            goHome,
            handleLogout
        };
    },
    template: `
        <nav class="navbar navbar-expand-lg navbar-dark bg-dark mb-3">
            <div class="container-fluid">
                <a class="navbar-brand" href="#" @click="goHome">
                    <i class="fas fa-folder-open"></i> 文件管理器
                </a>
                <div v-if="state.user">
                    <span class="navbar-text me-3">欢迎, {{ state.user }}</span>
                    <button class="btn btn-outline-light btn-sm" @click="handleLogout" :disabled="loading">
                        <i class="fas fa-spinner fa-spin" v-if="loading"></i>
                        <i class="fas fa-sign-out-alt" v-else></i>
                        {{ loading ? '注销中...' : '注销' }}
                    </button>
                </div>
            </div>
        </nav>
    `
});

// 操作按钮组件
const ActionButtons = defineComponent({
    name: 'ActionButtons',
    components: {
        MkdirModal,
        NewFileModal,
        UploadModal,
        ShellModal
    },
    setup() {
        const actions = inject(APP_ACTIONS_KEY);

        const mkdirModalRef = ref(null);
        const newFileModalRef = ref(null);
        const uploadModalRef = ref(null);
        const shellModalRef = ref(null);

        const showMkdirModal = () => mkdirModalRef.value.show();
        const showNewFileModal = () => newFileModalRef.value.show();
        const showUploadModal = () => uploadModalRef.value.show();
        const showShellModal = () => shellModalRef.value.show();
        const refreshFiles = () => actions.loadFiles();

        return {
            mkdirModalRef,
            newFileModalRef,
            uploadModalRef,
            shellModalRef,
            showMkdirModal,
            showNewFileModal,
            showUploadModal,
            showShellModal,
            refreshFiles
        };
    },
    template: `
        <div>
            <div class="mb-3 d-flex flex-wrap align-items-center gap-2">
                <button class="btn btn-success btn-sm" @click="showMkdirModal">
                    <i class="fas fa-folder"></i> 新建目录
                </button>
                <button class="btn btn-primary btn-sm" @click="showNewFileModal">
                    <i class="fas fa-file"></i> 新建文件
                </button>
                <button class="btn btn-info btn-sm" @click="showUploadModal">
                    <i class="fas fa-upload"></i> 上传文件
                </button>
                <button class="btn btn-secondary btn-sm" @click="refreshFiles">
                    <i class="fas fa-sync-alt"></i> 刷新
                </button>
                <button class="btn btn-dark btn-sm ms-auto" @click="showShellModal">
                    <i class="fas fa-terminal"></i> 终端
                </button>
            </div>

            <!-- 模态框组件 -->
            <MkdirModal ref="mkdirModalRef" />
            <NewFileModal ref="newFileModalRef" />
            <UploadModal ref="uploadModalRef" />
            <ShellModal ref="shellModalRef" />
        </div>
    `
});

// 文件列表组件
const FileList = defineComponent({
    name: 'FileList',
    components: {
        EditModal,
        RenameModal,
        ChmodModal,
        ZipModal
    },
    setup() {
        const state = inject(APP_STATE_KEY);
        const actions = inject(APP_ACTIONS_KEY);

        const editModalRef = ref(null);
        const renameModalRef = ref(null);
        const chmodModalRef = ref(null);
        const zipModalRef = ref(null);

        const isEditableFile = (file) => {
            const ext = file.name.split('.').pop().toLowerCase();
            return TEXT_EXTENSIONS.includes(ext);
        };

        const getFileIcon = (file) => {
            if (file.isDir) {
                return 'fas fa-folder text-warning';
            }
            const ext = file.name.split('.').pop().toLowerCase();
            return FILE_ICON_MAP[ext] || 'fas fa-file text-secondary';
        };

        const formatFileSize = (bytes) => {
            if (bytes === 0) return '0 B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        };

        const formatTime = (timeString) => {
            const date = new Date(timeString);
            return date.toLocaleDateString('zh-CN') + ' ' + date.toLocaleTimeString('zh-CN', { hour12: false });
        };

        const navigateTo = (path) => {
            actions.loadFiles(path);
        };

        const downloadFile = (file) => {
            const url = `/api/download?file=${encodeURIComponent(file.path)}&token=${state.token}`;
            window.open(url, '_blank');
        };

        const deleteFile = (file) => {
            if (confirm(`确定要删除 "${file.name}" 吗？`)) {
                actions.deleteFile(file);
            }
        };

        const unzipFile = (file) => {
            if (confirm(`确定要解压 "${file.name}" 吗？`)) {
                actions.unzipFile(file);
            }
        };

        const editFile = (file) => editModalRef.value.show(file);
        const showRenameModal = (file) => renameModalRef.value.show(file);
        const showChmodModal = (file) => chmodModalRef.value.show(file);
        const showZipModal = (file) => zipModalRef.value.show(file);

        return {
            state,
            editModalRef,
            renameModalRef,
            chmodModalRef,
            zipModalRef,
            isEditableFile,
            getFileIcon,
            formatFileSize,
            formatTime,
            navigateTo,
            downloadFile,
            deleteFile,
            unzipFile,
            editFile,
            showRenameModal,
            showChmodModal,
            showZipModal
        };
    },
    template: `
        <div>
            <div v-if="state.loading" class="loading">
                <i class="fas fa-spinner fa-spin fa-2x"></i>
                <p>加载中...</p>
            </div>

            <div v-else class="table-responsive">
                <table class="table table-hover">
                    <thead class="table-light">
                        <tr>
                            <th>名称</th>
                            <th>大小</th>
                            <th>权限</th>
                            <th>修改时间</th>
                            <th width="300">操作</th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr v-for="file in state.files" :key="file.name">
                            <td>
                                <i :class="getFileIcon(file)" class="file-icon"></i>
                                <a v-if="file.isDir" href="#" @click="navigateTo(file.path)">
                                    {{ file.name }}
                                </a>
                                <span v-else>{{ file.name }}</span>
                            </td>
                            <td>
                                <span v-if="!file.isDir" class="file-size">
                                    {{ formatFileSize(file.size) }}
                                </span>
                                <span v-else>-</span>
                            </td>
                            <td>{{ file.mode }}</td>
                            <td class="file-time">{{ formatTime(file.modTime) }}</td>
                            <td class="file-action">
                                <!-- 目录操作 -->
                                <template v-if="file.isDir">
                                    <button class="btn btn-outline-primary" @click="navigateTo(file.path)" title="进入目录">
                                        <i class="fas fa-folder-open"></i>
                                    </button>
                                    <button class="btn btn-outline-secondary" @click="showZipModal(file)" title="打包目录">
                                        <i class="fas fa-file-archive"></i>
                                    </button>
                                </template>
                                <!-- 文件操作 -->
                                <template v-else>
                                    <button class="btn btn-outline-success" @click="downloadFile(file)" title="下载">
                                        <i class="fas fa-download"></i>
                                    </button>
                                    <button v-if="isEditableFile(file)" class="btn btn-outline-info" @click="editFile(file)" title="编辑">
                                        <i class="fas fa-edit"></i>
                                    </button>
                                    <button v-if="file.name.endsWith('.zip')" class="btn btn-outline-warning" @click="unzipFile(file)" title="解压">
                                        <i class="fas fa-expand-arrows-alt"></i>
                                    </button>
                                </template>
                                <!-- 通用操作 -->
                                <button class="btn btn-outline-dark" @click="showRenameModal(file)" title="重命名">
                                    <i class="fas fa-pen"></i>
                                </button>
                                <button class="btn btn-outline-secondary" @click="showChmodModal(file)" title="权限">
                                    <i class="fas fa-key"></i>
                                </button>
                                <button class="btn btn-outline-danger" @click="deleteFile(file)" title="删除">
                                    <i class="fas fa-trash"></i>
                                </button>
                            </td>
                        </tr>
                    </tbody>
                </table>

                <div v-if="state.files.length === 0" class="text-center text-muted py-4">
                    <i class="fas fa-folder-open fa-3x mb-3"></i>
                    <p>此目录为空</p>
                </div>
            </div>

            <!-- 模态框组件 -->
            <EditModal ref="editModalRef" />
            <RenameModal ref="renameModalRef" />
            <ChmodModal ref="chmodModalRef" />
            <ZipModal ref="zipModalRef" />
        </div>
    `
});

// ==================== 应用组件 ====================

// 主应用组件
const FilerApp = defineComponent({
    name: 'FilerApp',
    components: {
        NavigationBar,
        LoginForm,
        BreadcrumbNav,
        ActionButtons,
        FileList,
        NotificationManager
    },
    setup() {
        const state = createAppState();
        const actions = createAppActions(state);

        provide(APP_STATE_KEY, state);
        provide(APP_ACTIONS_KEY, actions);

        onMounted(() => {
            // 检查本地存储的认证信息
            const savedToken = localStorage.getItem('fileManagerToken');
            const savedUser = localStorage.getItem('fileManagerUser');

            if (savedToken && savedUser) {
                actions.setAuth({ token: savedToken, user: savedUser });
                // 认证状态恢复后立即加载文件
                actions.loadFiles();
            }
        });

        return { state, actions };
    },
    template: `
        <NavigationBar />
        <LoginForm v-if="!state.user"  />
        <div v-else class="container-fluid">
            <BreadcrumbNav />
            <ActionButtons />
            <FileList />
        </div>
        <NotificationManager />
    `
});

// 创建 Vue 应用并挂载
createApp(FilerApp).mount('#app');
