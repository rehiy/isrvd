// ==================== 文件列表组件 ====================

const { defineComponent, inject, ref } = Vue;

import { APP_STATE_KEY, APP_ACTIONS_KEY } from '../helpers/state.js';
import { isEditableFile, getFileIcon, formatFileSize, formatTime } from '../helpers/utils.js';
import { EditModal, RenameModal, ChmodModal, ZipModal, DeleteModal, UnzipModal } from './modal-index.js';

// 文件列表组件
export const FileIndex = defineComponent({
    name: 'FileIndex',
    components: {
        EditModal,
        RenameModal,
        ChmodModal,
        ZipModal,
        DeleteModal,
        UnzipModal
    },
    setup() {
        const state = inject(APP_STATE_KEY);
        const actions = inject(APP_ACTIONS_KEY);

        const editModalRef = ref(null);
        const renameModalRef = ref(null);
        const chmodModalRef = ref(null);
        const zipModalRef = ref(null);
        const deleteModalRef = ref(null);
        const unzipModalRef = ref(null);

        const navigateTo = (path) => {
            actions.loadFiles(path);
        };

        const downloadFile = (file) => {
            const url = `/api/download?file=${encodeURIComponent(file.path)}&token=${state.token}`;
            window.open(url, '_blank');
        };

        const deleteFile = (file) => {
            deleteModalRef.value.show(file);
        };

        const unzipFile = (file) => {
            unzipModalRef.value.show(file);
        };

        const editFile = (file) => editModalRef.value.show(file);
        const showRenameModal = (file) => renameModalRef.value.show(file);
        const showChmodModal = (file) => chmodModalRef.value.show(file);
        const showZipModal = (file) => zipModalRef.value.show(file);
        const showDeleteModal = (file) => deleteModalRef.value.show(file);
        const showUnzipModal = (file) => unzipModalRef.value.show(file);

        return {
            state,
            editModalRef,
            renameModalRef,
            chmodModalRef,
            zipModalRef,
            deleteModalRef,
            unzipModalRef,
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
            showZipModal,
            showDeleteModal,
            showUnzipModal
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
            <DeleteModal ref="deleteModalRef" />
            <UnzipModal ref="unzipModalRef" />
        </div>
    `
});
