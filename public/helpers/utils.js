// ==================== 工具函数 ====================

export const TEXT_EXTENSIONS = [
    'txt', 'md', 'js', 'css', 'html', 'htm', 'json', 'xml', 'csv',
    'log', 'conf', 'ini', 'cfg', 'yaml', 'yml', 'php', 'py', 'go',
    'java', 'cpp', 'c', 'h', 'sql', 'sh', 'bat', 'env'
];

export const FILE_ICON_MAP = {
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

export const isEditableFile = (file) => {
    const ext = file.name.split('.').pop().toLowerCase();
    return TEXT_EXTENSIONS.includes(ext);
};

export const getFileIcon = (file) => {
    if (file.isDir) {
        return 'fas fa-folder text-warning';
    }
    const ext = file.name.split('.').pop().toLowerCase();
    return FILE_ICON_MAP[ext] || 'fas fa-file text-secondary';
};

export const formatFileSize = (bytes) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};

export const formatTime = (timeString) => {
    const date = new Date(timeString);
    return date.toLocaleDateString('zh-CN') + ' ' + date.toLocaleTimeString('zh-CN', { hour12: false });
};
