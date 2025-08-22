# Front-end 源码结构说明

## 目录结构

src/
├── app.vue              # 根组件
├── main.js              # 应用入口文件
├── assets/              # 静态资源文件
│   └── style.css
├── components/          # 按功能分类的可复用组件
│   ├── auth/            # 认证相关组件
│   │   └── auth.vue
│   ├── base/            # 基础 UI 组件
│   │   ├── base-modal.vue
│   │   └── notification.vue
│   ├── common/          # 通用组件
│   ├── file-manager/    # 文件管理组件
│   │   ├── file-actions.vue
│   │   └── file-explorer.vue
│   ├── forms/           # 表单组件
│   └── modals/          # 模态框组件
│       ├── chmod.vue
│       ├── delete.vue
│       ├── edit.vue
│       ├── mkdir.vue
│       ├── new-file.vue
│       ├── rename.vue
│       ├── shell.vue
│       ├── unzip.vue
│       ├── upload.vue
│       └── zip.vue
├── composables/         # Vue 3 组合式函数
├── layouts/             # 布局组件
│   ├── breadcrumb.vue
│   ├── logout-button.vue
│   └── navigation.vue
├── pages/               # 页面级组件
├── services/            # API 服务层
├── stores/              # 模块化状态管理
│   └── state.js
└── utils/               # 工具函数库
    ├── shell.js
    └── utils.js


## 导入路径配置

### 路径别名设置

项目配置了 `@` 路径别名，指向 `src` 目录：

```javascript
// vite.config.js
resolve: {
  alias: {
    '@': resolve(__dirname, 'src')
  }
}
```

### 导入示例

所有模块导入都使用 `@/` 开头的绝对路径：

```javascript
// 导入组件
import LoginForm from '@/components/auth/auth.vue'
import BaseModal from '@/components/base/base-modal.vue'
import NavigationBar from '@/layouts/navigation.vue'

// 导入状态管理
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/stores/state.js'

// 导入工具函数
import { isEditableFile, getFileIcon } from '@/utils/utils.js'
```

## 重构说明

### 主要变更

1. **组件分类优化**：
   - 将原 `ui/` 目录下的组件按功能重新分类到 `components/` 下
   - 认证组件移至 `components/auth/`
   - 基础 UI 组件移至 `components/base/`

2. **文件管理组件整合**：
   - 将独立的 `file-manager/` 目录整合到 `components/file-manager/`
   - 保持文件管理相关组件的集中管理

3. **布局组件统一**：
   - 将 `layout/` 目录内容合并到 `layouts/`
   - 统一布局组件的组织方式

4. **模态框组件整合**：
   - 将独立的 `modals/` 目录移至 `components/modals/`
   - 便于组件的统一管理和维护

### 导入路径更新

所有相关文件的导入路径已经更新，包括：
- 模态框组件中的 `BaseModal` 导入
- 主应用组件中的各种组件导入
- 组件间的相互引用

### 目录结构优势

- **清晰的功能分组**：每个目录都有明确的职责
- **易于维护**：相关功能的组件集中管理
- **便于扩展**：新组件可以轻松按功能分类添加
- **符合 Vue 最佳实践**：遵循 Vue 3 项目的标准目录结构
