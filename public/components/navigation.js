// ==================== 导航组件 ====================

const { defineComponent, inject, computed, ref } = Vue;

import { APP_STATE_KEY, APP_ACTIONS_KEY } from '../helpers/state.js';
import { MkdirModal, NewFileModal, UploadModal, ShellModal } from './modal-index.js';
import { LogoutButton } from './auth.js';

// 面包屑导航组件
export const BreadcrumbNav = defineComponent({
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
                    <a href="#" class="text-decoration-none" @click="navigateTo('/')">
                        <i class="fas fa-home me-1"></i> 首页
                    </a>
                </li>
                <li v-for="(part, index) in pathParts" :key="index" class="breadcrumb-item" :class="{ active: index === pathParts.length - 1 }">
                    <a class="text-decoration-none" v-if="index < pathParts.length - 1" href="#" @click="navigateTo('/' + pathParts.slice(0, index + 1).join('/'))">
                        {{ part }}
                    </a>
                    <span v-else>{{ part }}</span>
                </li>
            </ol>
        </nav>
    `
});

// 导航栏组件
export const NavigationBar = defineComponent({
    name: 'NavigationBar',
    components: {
        LogoutButton
    },
    setup() {
        const state = inject(APP_STATE_KEY);
        const actions = inject(APP_ACTIONS_KEY);

        const goHome = () => {
            actions.loadFiles('/');
        };

        return {
            state,
            goHome
        };
    },
    template: `
        <nav class="navbar navbar-expand-lg navbar-dark bg-dark mb-3">
            <div class="container-fluid">
                <a class="navbar-brand fw-semibold" href="#" @click="goHome">
                    <i class="fas fa-folder-open me-2"></i>服务器管理
                </a>
                <div v-if="state.user" class="d-flex align-items-center">
                    <span class="navbar-text me-3">欢迎, {{ state.user }}</span>
                    <LogoutButton />
                </div>
            </div>
        </nav>
    `
});

// 操作按钮组件
export const ActionButtons = defineComponent({
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
                    <i class="fas fa-folder me-1"></i>新建目录
                </button>
                <button class="btn btn-primary btn-sm" @click="showNewFileModal">
                    <i class="fas fa-file me-1"></i>新建文件
                </button>
                <button class="btn btn-info btn-sm" @click="showUploadModal">
                    <i class="fas fa-upload me-1"></i>上传文件
                </button>
                <button class="btn btn-secondary btn-sm" @click="refreshFiles">
                    <i class="fas fa-sync-alt me-1"></i>刷新
                </button>
                <button class="btn btn-dark btn-sm ms-auto" @click="showShellModal">
                    <i class="fas fa-terminal me-1"></i>终端
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
