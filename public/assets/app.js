
const { createApp } = Vue;

createApp({
    data() {
        return {
            user: null,
            token: null,
            currentPath: '/',
            files: [],
            loading: false,
            error: '',
            success: '',

            // 登录表单
            loginForm: {
                username: '',
                password: ''
            },

            // 新建目录表单
            mkdirForm: {
                name: ''
            },

            // 新建文件表单
            newFileForm: {
                name: '',
                content: ''
            },

            // 编辑文件表单
            editForm: {
                path: '',
                filename: '',
                content: ''
            },

            // 重命名表单
            renameForm: {
                oldPath: '',
                newName: ''
            },

            // 权限修改表单
            chmodForm: {
                path: '',
                mode: ''
            },

            // 压缩目录表单
            zipForm: {
                path: '',
                name: '',
                zipName: ''
            }
        };
    },

    computed: {
        pathParts() {
            if (this.currentPath === '/') return [];
            return this.currentPath.split('/').filter(part => part);
        }
    },

    mounted() {
        // 检查本地存储的token
        const savedToken = localStorage.getItem('fileManagerToken');
        const savedUser = localStorage.getItem('fileManagerUser');

        if (savedToken && savedUser) {
            this.token = savedToken;
            this.user = savedUser;
            this.setupAxiosAuth();
            this.loadFiles();
        }
    },

    methods: {
        // 设置Axios认证
        setupAxiosAuth() {
            axios.defaults.headers.common['Authorization'] = this.token;
        },

        // 登录
        async login() {
            this.loading = true;
            this.error = '';

            try {
                const response = await axios.post('/api/login', this.loginForm);

                this.token = response.data.token;
                this.user = response.data.user;

                // 保存到本地存储
                localStorage.setItem('fileManagerToken', this.token);
                localStorage.setItem('fileManagerUser', this.user);

                this.setupAxiosAuth();
                this.loadFiles();

            } catch (error) {
                this.error = error.response?.data?.error || '登录失败';
            } finally {
                this.loading = false;
            }
        },

        // 登出
        async logout() {
            try {
                await axios.post('/api/logout');
            } catch (error) {
                console.warn('Logout request failed:', error);
            }

            this.user = null;
            this.token = null;
            this.files = [];

            // 清除本地存储
            localStorage.removeItem('fileManagerToken');
            localStorage.removeItem('fileManagerUser');

            delete axios.defaults.headers.common['Authorization'];
        },

        // 加载文件列表
        async loadFiles(path = this.currentPath) {
            this.loading = true;
            this.error = '';

            try {
                const response = await axios.get('/api/files', {
                    params: { path }
                });

                this.files = response.data.files || [];
                this.currentPath = response.data.path;

            } catch (error) {
                this.error = error.response?.data?.error || '加载文件列表失败';

                // 如果是401错误，可能是token过期
                if (error.response?.status === 401) {
                    this.logout();
                }
            } finally {
                this.loading = false;
            }
        },

        // 导航到指定路径
        navigateTo(path) {
            this.loadFiles(path);
        },

        // 刷新文件列表
        refreshFiles() {
            this.loadFiles();
        },

        // 显示新建目录模态框
        showMkdirModal() {
            this.mkdirForm.name = '';
            const modal = new bootstrap.Modal(document.getElementById('mkdirModal'));
            modal.show();
        },

        // 创建目录
        async createDirectory() {
            if (!this.mkdirForm.name.trim()) {
                this.error = '目录名称不能为空';
                return;
            }

            try {
                await axios.post('/api/mkdir', {
                    path: this.currentPath,
                    name: this.mkdirForm.name
                });

                this.success = '目录创建成功';
                this.mkdirForm.name = '';

                const modal = bootstrap.Modal.getInstance(document.getElementById('mkdirModal'));
                modal.hide();

                this.loadFiles();

            } catch (error) {
                this.error = error.response?.data?.error || '创建目录失败';
            }
        },

        // 显示新建文件模态框
        showNewFileModal() {
            this.newFileForm.name = '';
            this.newFileForm.content = '';
            const modal = new bootstrap.Modal(document.getElementById('newFileModal'));
            modal.show();
        },

        // 创建文件
        async createFile() {
            if (!this.newFileForm.name.trim()) {
                this.error = '文件名称不能为空';
                return;
            }

            try {
                await axios.post('/api/newfile', {
                    path: this.currentPath,
                    name: this.newFileForm.name,
                    content: this.newFileForm.content
                });

                this.success = '文件创建成功';
                this.newFileForm.name = '';
                this.newFileForm.content = '';

                const modal = bootstrap.Modal.getInstance(document.getElementById('newFileModal'));
                modal.hide();

                this.loadFiles();

            } catch (error) {
                this.error = error.response?.data?.error || '创建文件失败';
            }
        },

        // 显示上传文件模态框
        showUploadModal() {
            const modal = new bootstrap.Modal(document.getElementById('uploadModal'));
            modal.show();
        },

        // 上传文件
        async uploadFile() {
            const fileInput = this.$refs.fileInput;
            if (!fileInput.files.length) {
                this.error = '请选择要上传的文件';
                return;
            }

            const formData = new FormData();
            formData.append('file', fileInput.files[0]);
            formData.append('path', this.currentPath);

            try {
                await axios.post('/api/upload', formData, {
                    headers: {
                        'Content-Type': 'multipart/form-data'
                    }
                });

                this.success = '文件上传成功';
                fileInput.value = '';

                const modal = bootstrap.Modal.getInstance(document.getElementById('uploadModal'));
                modal.hide();

                this.loadFiles();

            } catch (error) {
                this.error = error.response?.data?.error || '上传文件失败';
            }
        },

        // 下载文件
        downloadFile(file) {
            const url = `/api/download?file=${encodeURIComponent(file.path)}&token=${this.token}`;
            window.open(url, '_blank');
        },

        // 判断文件是否可编辑
        isEditableFile(file) {
            const textExtensions = ['txt', 'md', 'js', 'css', 'html', 'htm', 'json', 'xml', 'csv',
                'log', 'conf', 'ini', 'cfg', 'yaml', 'yml', 'php', 'py', 'go',
                'java', 'cpp', 'c', 'h', 'sql', 'sh', 'bat', 'env'];
            const ext = file.name.split('.').pop().toLowerCase();
            return textExtensions.includes(ext);
        },

        // 编辑文件
        async editFile(file) {
            try {
                const response = await axios.get('/api/edit', {
                    params: { file: file.path }
                });

                this.editForm.path = file.path;
                this.editForm.filename = file.name;
                this.editForm.content = response.data.content;

                const modal = new bootstrap.Modal(document.getElementById('editModal'));
                modal.show();

            } catch (error) {
                this.error = error.response?.data?.error || '无法打开文件';
            }
        },

        // 保存文件
        async saveFile() {
            try {
                await axios.post('/api/edit', {
                    content: this.editForm.content
                }, {
                    params: { file: this.editForm.path }
                });

                this.success = '文件保存成功';

                const modal = bootstrap.Modal.getInstance(document.getElementById('editModal'));
                modal.hide();

                this.loadFiles();

            } catch (error) {
                this.error = error.response?.data?.error || '保存文件失败';
            }
        },

        // 显示重命名模态框
        showRenameModal(file) {
            this.renameForm.oldPath = file.path;
            this.renameForm.newName = file.name;
            const modal = new bootstrap.Modal(document.getElementById('renameModal'));
            modal.show();
        },

        // 重命名文件
        async renameFile() {
            if (!this.renameForm.newName.trim()) {
                this.error = '新名称不能为空';
                return;
            }

            try {
                await axios.post('/api/rename', {
                    oldPath: this.renameForm.oldPath,
                    newName: this.renameForm.newName
                });

                this.success = '重命名成功';

                const modal = bootstrap.Modal.getInstance(document.getElementById('renameModal'));
                modal.hide();

                this.loadFiles();

            } catch (error) {
                this.error = error.response?.data?.error || '重命名失败';
            }
        },

        // 显示权限修改模态框
        async showChmodModal(file) {
            try {
                const response = await axios.get('/api/chmod', {
                    params: { file: file.path }
                });

                this.chmodForm.path = file.path;
                this.chmodForm.mode = response.data.mode;

                const modal = new bootstrap.Modal(document.getElementById('chmodModal'));
                modal.show();

            } catch (error) {
                this.error = error.response?.data?.error || '无法获取文件权限';
            }
        },

        // 修改权限
        async changePermissions() {
            if (!this.chmodForm.mode.trim()) {
                this.error = '权限不能为空';
                return;
            }

            try {
                await axios.post('/api/chmod', {
                    mode: this.chmodForm.mode
                }, {
                    params: { file: this.chmodForm.path }
                });

                this.success = '权限修改成功';

                const modal = bootstrap.Modal.getInstance(document.getElementById('chmodModal'));
                modal.hide();

                this.loadFiles();

            } catch (error) {
                this.error = error.response?.data?.error || '修改权限失败';
            }
        },

        // 删除文件
        async deleteFile(file) {
            if (!confirm(`确定要删除 "${file.name}" 吗？`)) {
                return;
            }

            try {
                await axios.delete('/api/delete', {
                    params: { file: file.path }
                });

                this.success = '删除成功';
                this.loadFiles();

            } catch (error) {
                this.error = error.response?.data?.error || '删除失败';
            }
        },

        // 显示压缩目录模态框
        showZipModal(file) {
            this.zipForm = {
                path: file.path,
                name: file.name,
                zipName: file.name + '.zip'
            };
            const modal = new bootstrap.Modal(document.getElementById('zipModal'));
            modal.show();
        },

        // 压缩目录
        async zipDirectory() {
            if (!this.zipForm.zipName.trim()) {
                this.error = '压缩包名称不能为空';
                return;
            }
            try {
                await axios.post('/api/zip', {
                    path: this.zipForm.path,
                    zipName: this.zipForm.zipName
                });
                this.success = '压缩成功';
                const modal = bootstrap.Modal.getInstance(document.getElementById('zipModal'));
                modal.hide();
                this.loadFiles();
            } catch (error) {
                this.error = error.response?.data?.error || '压缩失败';
            }
        },

        // 解压文件
        async unzipFile(file) {
            if (!confirm(`确定要解压 "${file.name}" 吗？`)) {
                return;
            }

            try {
                await axios.post('/api/unzip', {
                    path: this.currentPath,
                    zipName: file.name
                });

                this.success = '解压成功';
                this.loadFiles();

            } catch (error) {
                this.error = error.response?.data?.error || '解压失败';
            }
        },

        // 获取文件图标
        getFileIcon(file) {
            if (file.isDir) {
                return 'fas fa-folder text-warning';
            }

            const ext = file.name.split('.').pop().toLowerCase();
            const iconMap = {
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

            return iconMap[ext] || 'fas fa-file text-secondary';
        },

        // 格式化文件大小
        formatFileSize(bytes) {
            if (bytes === 0) return '0 B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        },

        // 格式化时间
        formatTime(timeString) {
            const date = new Date(timeString);
            return date.toLocaleDateString('zh-CN') + ' ' + date.toLocaleTimeString('zh-CN', { hour12: false });
        }
    },

    watch: {
        // 自动清除提示信息
        error(newVal) {
            if (newVal) {
                setTimeout(() => {
                    this.error = '';
                }, 5000);
            }
        },

        success(newVal) {
            if (newVal) {
                setTimeout(() => {
                    this.success = '';
                }, 3000);
            }
        }
    }
}).mount('app');
