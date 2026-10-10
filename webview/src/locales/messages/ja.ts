import type { LanguageModule } from '../types'

/**
 * 日本語訳：キーは中国語原文（簡体字）
 *
 * よく使われる UI 文言を中心に登録済み（737/1897）。未登録の文案は中国語原文に
 * フォールバックするため、空白やキーの漏洩は発生しない。
 *
 * 訳文を追加するときは、未訳リストを書き出して埋めるのが早い：
 *   node scripts/check-locales.mjs --worklist ja > tmp/ja-ui.jsonl
 * 登録後は node scripts/check-locales.mjs で取りこぼしを確認する。
 * このファイルは手書きでもよいが、scripts/apply-translations.mjs で取り込むと
 * 英文コメントと並び順が保たれる。
 */

const messages: Record<string, string> = {
    // Loading...
    '加载中...': '読み込み中...',
    // AI Assistant
    'AI 助手': 'AI アシスタント',
    // Not signed in
    '未登录': '未ログイン',
    // Account Security
    '账号安全': 'アカウントセキュリティ',
    // Sign out
    '退出': 'ログアウト',
    // Current: 
    '当前：': '現在：',
    // , click to switch
    '，点击切换': '（クリックで切り替え）',
    // Light mode
    '浅色模式': 'ライトモード',
    // Dark mode
    '深色模式': 'ダークモード',
    // Follow system
    '跟随系统': 'システムに合わせる',
    // Overview
    '概览': '概要',
    // Local
    '本机管理': 'ローカル管理',
    // Monitoring
    '系统监控': 'システム監視',
    // Files
    '文件管理': 'ファイル管理',
    // Processes
    '进程管理': 'プロセス管理',
    // Shell
    'Shell 终端': 'Shell ターミナル',
    // APISIX Gateway
    'APISIX 网关': 'APISIX ゲートウェイ',
    // Routes
    '路由': 'ルート',
    // Upstreams
    '上游': 'アップストリーム',
    // Plugin Configs
    '插件配置': 'プラグイン設定',
    // Consumers
    '消费者': 'コンシューマー',
    // SSL Certificates
    'SSL 证书': 'SSL 証明書',
    // Access Control
    '访问授权': 'アクセス権限',
    // Caddy Gateway
    'Caddy 网关': 'Caddy ゲートウェイ',
    // Services
    '服务': 'サービス',
    // Basic Auth
    '基础认证': 'Basic 認証',
    // Global Options
    '全局选项': 'グローバル設定',
    // Raw Config
    '原始配置': '生の設定',
    // Docker
    'Docker 服务': 'Docker',
    // Containers
    '容器': 'コンテナ',
    // Images
    '镜像': 'イメージ',
    // Networks
    '网络': 'ネットワーク',
    // Volumes
    '数据卷': 'ボリューム',
    // Registries
    '镜像仓库': 'レジストリ',
    // Swarm Cluster
    'Swarm 集群': 'Swarm クラスタ',
    // Nodes
    '节点': 'ノード',
    // Tasks
    '任务': 'タスク',
    // Hosts
    '主机连接': 'ホスト接続',
    // Credentials
    '认证凭据': '認証情報',
    // Marketplace
    '应用市场': 'マーケット',
    // Compose Deploy
    'Compose 部署': 'Compose デプロイ',
    // Cron Jobs
    '计划任务': 'スケジュールタスク',
    // System Settings
    '系统配置': 'システム設定',
    // Audit Logs
    '操作审计': '操作監査',
    // Members
    '用户管理': 'メンバー管理',
    // Node Management
    '节点管理': 'ノード管理',
    // Expand menu
    '展开菜单': 'メニューを展開',
    // Collapse menu
    '收起菜单': 'メニューを折りたたむ',
    // API Docs
    'API 文档': 'API ドキュメント',
    // GitHub
    'GitHub 仓库': 'GitHub',
    // Quick links
    '快捷链接': 'クイックリンク',
    // No quick links
    '无快捷链接': 'クイックリンクはありません',
    // Services
    '基础服务': 'サービス',
    // Authentication
    '登录认证': '認証',
    // Gateway
    '网关容器': 'ゲートウェイ',
    // Alerts
    '监控告警': '監視とアラート',
    // Integrations
    '扩展集成': '連携',
    // Ports, directories, upload, CORS and JWT
    '端口、目录、上传、跨域与 JWT': 'ポート、ディレクトリ、アップロード、CORS、JWT',
    // Password, Passkey, OIDC and proxy header login
    '密码、Passkey、OIDC 与代理 Header 登录': 'パスワード、Passkey、OIDC、プロキシヘッダー認証',
    // APISIX, Caddy and Docker connection options
    'APISIX、Caddy 与 Docker 连接参数': 'APISIX、Caddy、Docker の接続設定',
    // Interval, webhook channels, thresholds and app faults
    '采集间隔、Webhook 通道、资源阈值与应用故障': '収集間隔、Webhook チャネル、リソース閾値、アプリ障害',
    // AI assistant, marketplace and navigation links
    'AI 助手、应用市场与导航链接': 'AI アシスタント、マーケット、ナビゲーションリンク',
    // Welcome back
    '欢迎回来': 'おかえりなさい',
    // Sign in to the iSrvd admin panel
    '登录到 iSrvd 管理面板': 'iSrvd 管理パネルにサインイン',
    // Username
    '用户名': 'ユーザー名',
    // Password
    '密码': 'パスワード',
    // Enter your username
    '请输入用户名': 'ユーザー名を入力',
    // Enter your password
    '请输入密码': 'パスワードを入力',
    // Two-factor code
    '二次验证码': '二段階認証コード',
    // Enter the 6-digit code
    '请输入 6 位验证码': '6 桁の認証コードを入力',
    // TOTP two-factor auth is enabled for this account, enter the code from your authenticator app.
    '该账户已启用 TOTP 二次验证，请输入认证器 App 中的动态验证码。': 'このアカウントでは TOTP 二段階認証が有効です。認証アプリのワンタイムコードを入力してください。',
    // Sign in
    '登录': 'ログイン',
    // Signing in...
    '登录中...': 'サインイン中...',
    // Verify and sign in
    '验证并登录': '検証してサインイン',
    // Sign in with Passkey
    '使用 Passkey 登录': 'Passkey でサインイン',
    // Authenticating with Passkey...
    'Passkey 认证中...': 'Passkey で認証中...',
    // Sign in with OIDC
    '使用 OIDC 登录': 'OIDC でサインイン',
    // Password login is disabled, contact the administrator to configure another method.
    '密码登录已禁用，请联系管理员配置其他登录方式。': 'パスワードログインは無効です。他のログイン方法を設定するよう管理者にお問い合わせください。',
    // Language
    '语言': '言語',
    // Confirm
    '确认操作': '確認',
    // Confirm
    '确认': '確認',
    // Cancel
    '取消': 'キャンセル',
    // Processing...
    '处理中...': '処理中...',
    // This action cannot be undone!
    '此操作不可恢复！': 'この操作は取り消せません！',
    // Search...
    '搜索...': '検索...',
    // Unknown error
    '发生未知错误': '不明なエラーが発生しました',
    // Session expired, please sign in again
    '登录已过期，请重新登录': 'セッションの有効期限が切れました。再度サインインしてください',
    // Request failed
    '请求失败': 'リクエストに失敗しました',
    // Download failed
    '下载失败': 'ダウンロードに失敗しました',
    // Network unreachable, please check your connection
    '网络连接失败，请检查网络': 'ネットワークに接続できません。接続を確認してください',
    // BAT batch script
    'BAT 批处理脚本': 'BAT バッチスクリプト',
    // PowerShell script
    'PowerShell 脚本': 'PowerShell スクリプト',
    // Shell script
    'Shell 脚本': 'Shell スクリプト',
    // Docker temporary container
    'Docker 临时容器': 'Docker 一時コンテナ',
    // Docker existing container
    'Docker 现有容器': 'Docker 既存コンテナ',
    // Executable file
    '可执行文件': '実行ファイル',
    // The call reference does not exist or has expired, call lookup_api again.
    '调用引用不存在或已过期，请重新调用 lookup_api。': '呼び出し参照が存在しないか期限切れです。lookup_api を再度呼び出してください。',
    // The same call failed twice in a row, look up the API again or adjust the parameters.
    '相同调用已连续失败两次，请重新查询接口或调整参数后再试。': '同じ呼び出しが 2 回連続で失敗しました。API を再検索するか、パラメータを調整して再試行してください。',
    // The HTTP method in the OpenAPI document is not supported.
    'OpenAPI 中的 HTTP 方法不受支持。': 'OpenAPI の HTTP メソッドはサポートされていません。',
    // The API did not return a standard JSON response, use the page instead.
    '接口未返回标准 JSON 响应，请改用页面操作。': 'API が標準的な JSON 応答を返しませんでした。画面から操作してください。',
    // The API definition changed, call lookup_api again.
    '接口定义已变化，请重新调用 lookup_api。': 'API 定義が変更されました。lookup_api を再度呼び出してください。',
    // The resolved API path is not a valid internal path.
    '解析后的 API 路径不是合法的站内路径。': '解決された API パスは不正な内部パスです。',
    // arguments is not valid JSON.
    'arguments 不是合法 JSON。': 'arguments は正しい JSON ではありません。',
    // arguments must be a JSON object.
    'arguments 必须是 JSON 对象。': 'arguments は JSON オブジェクトでなければなりません。',
    // arguments.path must be an object.
    'arguments.path 必须是对象。': 'arguments.path はオブジェクトでなければなりません。',
    // arguments.query must be an object.
    'arguments.query 必须是对象。': 'arguments.query はオブジェクトでなければなりません。',
    // This operation must use {0}.
    '该操作必须使用 {0}。': 'この操作は {0} を使用する必要があります。',
    // Enter a message...
    '请输入消息...': 'メッセージを入力...',
    // Start voice input
    '开始语音输入': '音声入力を開始',
    // Cancel voice input
    '取消语音输入': '音声入力をキャンセル',
    // Finish voice input
    '完成语音输入': '音声入力を完了',
    // Add image or file
    '添加图片或文件': '画像またはファイルを追加',
    // Tools
    '工具': 'ツール',
    // Copy code
    '复制代码': 'コードをコピー',
    // Copy message
    '复制消息': 'メッセージをコピー',
    // Helpful
    '有帮助': '役に立った',
    // Not helpful
    '没帮助': '役に立たなかった',
    // Read aloud
    '朗读': '読み上げ',
    // Regenerate
    '重新生成': '再生成',
    // Edit message
    '编辑消息': 'メッセージを編集',
    // AI can make mistakes, please verify important information.
    'AI 可能出错，请核实重要信息。': 'AI は誤る可能性があります。重要な情報は確認してください。',
    // Open chat
    '打开聊天': 'チャットを開く',
    // Close chat
    '关闭聊天': 'チャットを閉じる',
    // Chat iSrvd
    'Chat iSrvd': 'Chat iSrvd',
    // Hi, how can I help you?
    '你好，我能帮你做什么？': 'こんにちは。何をお手伝いしましょうか？',
    // Failed to look up the API docs
    '查阅 API 文档失败': 'API ドキュメントの参照に失敗しました',
    // Waiting for confirmation
    '等待确认': '確認を待っています',
    // Preparing operation
    '正在准备操作': '操作を準備しています',
    // Querying resource
    '正在查询资源': 'リソースを照会しています',
    // Applying change
    '正在执行变更': '変更を適用しています',
    // Operation cancelled
    '操作已取消': '操作はキャンセルされました',
    // Operation completed
    '操作完成': '操作が完了しました',
    // Operation failed
    '操作失败': '操作に失敗しました',
    // Execution failed
    '执行失败': '実行に失敗しました',
    // Failed to cancel
    '取消失败': 'キャンセルに失敗しました',
    // Executing…
    '执行中…': '実行中…',
    // Confirm Execution
    '确认执行': '実行の確認',
    // Reading resource…
    '正在读取资源…': 'リソースを読み込み中…',
    // Executing operation…
    '正在执行操作…': '操作を実行中…',
    // The user cancelled the operation
    '用户取消了操作': 'ユーザーが操作をキャンセルしました',
    // Waiting for call reference
    '等待调用引用': '呼び出し参照を待っています',
    // [connecting...]
    '[连接中...]': '[接続中...]',
    // [connection closed]
    '[连接已关闭]': '[接続が閉じられました]',
    // [connection error: {0}]
    '[连接错误: {0}]': '[接続エラー: {0}]',
    // Enter the plugin config JSON...
    '请输入插件配置 JSON...': 'プラグイン設定の JSON を入力...',
    // Search options
    '搜索选项': '項目を検索',
    // Invalid JSON: 
    'JSON 格式错误: ': 'JSON の形式が不正です: ',
    // {0} · {1} and {2} nodes
    '{0} · {1} 等 {2} 个节点': '{0} · {1} ほか {2} ノード',
    // {0} failed
    '{0} 失败': '{0} に失敗しました',
    // {0} cancelled
    '{0} 已取消': '{0} はキャンセルされました',
    // Usage: 
    '使用率: ': '使用率: ',
    // Failed to save the system config
    '保存系统配置失败': 'システム設定の保存に失敗しました',
    // Failed to load logs
    '加载日志失败': 'ログの読み込みに失敗しました',
    // Cancel Upload
    '取消上传': 'アップロードのキャンセル',
    // Failed to connect to the live log stream
    '实时日志连接失败': 'リアルタイムログへの接続に失敗しました',
    // Cancelled
    '已取消': 'キャンセルしました',
    // References upstream #{0}
    '引用上游 #{0}': 'アップストリーム #{0} を参照',
    // Cannot start Passkey registration
    '无法开始 Passkey 注册': 'Passkey の登録を開始できません',
    // Cannot start Passkey sign-in
    '无法开始 Passkey 登录': 'Passkey のサインインを開始できません',
    // Clear Cancelled
    '清理已取消': 'クリアのキャンセル',
    // The user cancelled Passkey registration
    '用户取消了 Passkey 注册': 'ユーザーが Passkey の登録をキャンセルしました',
    // The user cancelled Passkey authentication
    '用户取消了 Passkey 认证': 'ユーザーが Passkey 認証をキャンセルしました',
    // Sign-in failed
    '登录失败': 'サインインに失敗しました',
    // Editing: 
    '编辑: ': '編集中: ',
    // Retry
    '重试': '再試行',
    // Retry Failed Items
    '重试失败项': '失敗した項目を再試行',
    // Preview: 
    '预览: ': 'プレビュー: ',
    //  · no snapshot
    ' · 无快照': ' · スナップショットなし',
    // Compose config updated, related containers rebuilt
    'Compose 配置更新成功，已重建关联容器': 'Compose 設定を更新し、関連コンテナを再作成しました',
    // Invalid JSON:
    'JSON 格式错误:': 'JSON の形式が不正です:',
    // PEM private key starting with "-----BEGIN", key authentication takes precedence when set
    'PEM 格式私钥，以 "-----BEGIN" 开头，设置后优先使用私钥认证': '"-----BEGIN" で始まる PEM 形式の秘密鍵。設定すると秘密鍵認証が優先されます',
    // Upload zip
    '上传 zip': 'zip をアップロード',
    // Host
    '主机': 'ホスト',
    // Usage: 
    '使用率:': '使用率:',
    // A recognizable name, e.g. "production key"
    '便于识别的名称，如 "生产环境密钥"': '識別しやすい名前（例: "本番用キー"）',
    // Disables automatic certificates and HTTPS config for this service.
    '关闭该服务的自动证书和 HTTPS 配置。': 'このサービスの自動証明書と HTTPS 設定を無効にします。',
    // Write
    '写': '書き込み',
    // Create Service
    '创建服务': 'サービスを作成',
    // Delete Authorization
    '删除授权': '権限を削除',
    // Power
    '功耗': '消費電力',
    // Failed to load the system config
    '加载系统配置失败': 'システム設定の読み込みに失敗しました',
    // Header Match
    '匹配请求头': 'リクエストヘッダーの一致条件',
    // Enable TLS
    '启用 TLS': 'TLS を有効化',
    // Response Header
    '响应头': 'レスポンスヘッダー',
    // Heap Objects
    '堆对象': 'ヒープオブジェクト数',
    // Client Private Key
    '客户端私钥': 'クライアント秘密鍵',
    // Client Certificate
    '客户端证书': 'クライアント証明書',
    // Container Name
    '容器名称': 'コンテナ名',
    // Resume
    '恢复': '再開する',
    // Manual Input
    '手动输入': '手動入力',
    // Search
    '搜索': '検索',
    // File System
    '文件系统': 'ファイルシステム',
    // Related containers are rebuilt from the Compose project after updating, old containers are stopped and deleted
    '更新配置后将会按 Compose 项目重建关联容器，旧容器将被停止并删除': '設定を更新すると Compose プロジェクトに従って関連コンテナが再作成され、古いコンテナは停止して削除されます',
    // Add
    '添加': '追加する',
    // Add Authentication Account
    '添加认证账号': '認証アカウントを追加',
    // Prune
    '清理': '整理',
    // Target Directory
    '目标目录': '対象ディレクトリ',
    // Relative to the "base directory", empty creates base directory/username automatically
    '相对路径基于"基础目录"，留空则自动创建为 基础目录/用户名': '相対パスは「ベースディレクトリ」基準です。空の場合は ベースディレクトリ/ユーザー名 を自動作成します',
    // Kill
    '终止': 'kill',
    // Editing:
    '编辑:': '編集中:',
    // Network Name
    '网络名称': 'ネットワーク名',
    // Node Name
    '节点名称': 'ノード名',
    // Certificate Source
    '证书来源': '証明書の取得元',
    // Request Header
    '请求头': 'リクエストヘッダー',
    // Enter the hostname
    '请输入主机名': 'ホスト名を入力',
    // Enter the private key PEM content
    '请输入私钥 PEM 内容': '秘密鍵の PEM を入力',
    // Enter the port (optional)
    '请输入端口（可选）': 'ポートを入力（省略可）',
    // Enter the certificate PEM content
    '请输入证书 PEM 内容': '証明書の PEM を入力',
    // Read Timeout
    '读取超时': '読み取りタイムアウト',
    // Skips compose.yml, re-derive Compose from the running container state
    '跳过 compose.yml，按当前容器运行态重新反推 Compose': 'compose.yml を読み飛ばし、現在のコンテナの状態から Compose を再構成します',
    // Process
    '进程': 'プロセス',
    // Connect Timeout
    '连接超时': '接続タイムアウト',
    // Select Container and Port
    '选择容器与端口': 'コンテナとポートを選択',
    // Failed to load the config
    '配置加载失败': '設定の読み込みに失敗しました',
    // Failed to load the config, the previous content is kept
    '配置加载失败，已保留原内容': '設定の読み込みに失敗したため、元の内容を保持しています',
    // Preview:
    '预览:': 'プレビュー:',
    // Refresh
    '刷新': '更新',
    // Delete
    '删除': '削除する',
    // Edit
    '编辑': '編集する',
    // Try another keyword or clear the filters
    '尝试更换关键词或清空搜索条件': '別のキーワードを試すか、検索条件をクリアしてください',
    // Details
    '详情': '詳細',
    // Actions
    '操作': '操作',
    // Confirm delete
    '确认删除': '削除の確認',
    // Name
    '名称': '名前',
    // Leave empty to keep unchanged
    '留空则保持不变': '空の場合は変更しません',
    // Created
    '创建时间': '作成日時',
    // Enter keywords to search...
    '请输入搜索关键词...': '検索キーワードを入力...',
    // Status
    '状态': 'ステータス',
    // Logs
    '日志': 'ログ',
    // Description
    '描述': '説明',
    // Disable
    '禁用': '無効にする',
    // (optional)
    '(可选)': '(省略可)',
    // Update
    '更新': '更新する',
    // Tags
    '标签': 'タグ',
    // Enable
    '启用': '有効にする',
    // Stop
    '停止': '停止する',
    // Permissions
    '权限': '権限',
    // Total
    '共': '全',
    // Deleted
    '删除成功': '削除しました',
    // Basic info
    '基本信息': '基本情報',
    // Address
    '地址': 'アドレス',
    // Create
    '创建': '作成する',
    // New Route
    '新建路由': 'ルートを新規作成',
    // New Certificate
    '新建证书': '証明書を新規作成',
    // User
    '用户': 'ユーザー',
    // New
    '新建': '新規作成',
    // Default
    '默认': 'デフォルト',
    // Hostname
    '主机名': 'ホスト名',
    // Start
    '启动': '起動する',
    // Memory
    '内存': 'メモリ',
    // items
    '项': '件',
    // Clear
    '清空': 'クリア',
    // All
    '全部': 'すべて',
    // Size
    '大小': 'サイズ',
    // (optional)
    '（可选）': '(省略可)',
    // New User
    '新建用户': 'ユーザーを新規作成',
    // records
    '条': '件',
    // Expires
    '有效期': '有効期限',
    // Read-only
    '只读': '読み取り専用',
    // Updated
    '更新时间': '更新日時',
    // Timeout (seconds)
    '超时时间（秒）': 'タイムアウト（秒）',
    // Save Config
    '保存配置': '設定を保存',
    // New Service
    '新建服务': 'サービスを新規作成',
    // Environment Variables
    '环境变量': '環境変数',
    // Pause
    '暂停': '一時停止する',
    // Running
    '运行中': '実行中',
    // Anonymous
    '匿名': '匿名',
    // Authentication
    '认证': '認証',
    // Connect
    '连接': '接続',
    // Confirm create
    '确认新建': '作成の確認',
    // New File
    '新建文件': 'ファイルを新規作成',
    // Rename
    '重命名': '名前を変更',
    // Copy failed, please copy manually
    '复制失败，请手动复制': 'コピーに失敗しました。手動でコピーしてください',
    // Enter the port
    '请输入端口': 'ポートを入力',
    // Edit Config
    '编辑配置': '設定を編集',
    // Protocol
    '协议': 'プロトコル',
    // Confirm add
    '确认添加': '追加の確認',
    // Save changes
    '保存修改': '変更を保存',
    // Schedule
    '执行计划': 'スケジュール',
    // Working Directory
    '工作目录': '作業ディレクトリ',
    // Container not found
    '容器不存在': 'コンテナが見つかりません',
    // Port Mapping
    '端口映射': 'ポートマッピング',
    // Driver
    '驱动': 'ドライバ',
    // Scope
    '范围': 'スコープ',
    // Restart
    '重启': '再起動する',
    // Port
    '端口': 'ポート',
    // View details
    '查看详情': '詳細を見る',
    // Live
    '实时': 'リアルタイム',
    // Force kill
    '强制终止': '強制終了（SIGKILL）',
    // Mode
    '模式': 'モード',
    // Drain
    '排空': 'ドレイン',
    // Activate
    '激活': 'アクティブ化',
    // Remove
    '移除': '削除',
    // Saving...
    '保存中...': '保存中...',
    // Home Directory
    '家目录': 'ホームディレクトリ',
    // Enabled
    '已启用': '有効',
    // Enter your login password
    '请输入登录密码': 'ログインパスワードを入力',
    // New Consumer
    '新建消费者': 'コンシューマーを新規作成',
    // Plugin
    '插件': 'プラグイン',
    // Not configured
    '未配置': '未設定',
    // Confirm {0}
    '确认{0}': '{0} の確認',
    // No matching route
    '未找到匹配路由': '一致するルートが見つかりません',
    // {0} and {1} nodes
    '{0} 等 {1} 个节点': '{0} ほか {1} ノード',
    // Edit Authorization
    '编辑授权': '権限を編集',
    // Header Name
    '请求头名称': 'ヘッダー名',
    // Type
    '类型': '種別',
    // Route #{0}
    '路由 #{0}': 'ルート #{0}',
    // Add Account
    '添加账号': 'アカウントを追加',
    // Issuer
    '签发机构': '発行者',
    // System
    '系统': 'システム',
    // e.g. 10s, 5m
    '如 10s、5m': '例: 10s、5m',
    // Write
    '写入': '書き込み',
    // Private Key
    '私钥': '秘密鍵',
    // Deploy
    '部署': 'デプロイ',
    // Success
    '成功': '成功',
    // Failed
    '失败': '失敗',
    // Enter timeout (optional)
    '请输入超时时间（可选）': 'タイムアウトを入力（省略可）',
    // 0 or empty means no limit
    '0 或留空表示不限制': '0 または空欄は無制限',
    // Enter or select an image name
    '请输入或选择镜像名': 'イメージ名を入力または選択',
    // No tags
    '无标签': 'タグなし',
    // OS
    '操作系统': 'OS',
    // Subnet
    '子网': 'サブネット',
    // Mount Point
    '挂载点': 'マウント先',
    // New Container
    '新建容器': 'コンテナを新規作成',
    // Stats
    '统计': '統計',
    // Pull
    '拉取': '取得',
    // New Network
    '新建网络': 'ネットワークを新規作成',
    // Disconnect
    '断开': '切断',
    // Not specified
    '不指定': '指定しない',
    // Kill Process
    '终止进程': 'プロセスを強制終了',
    // Waiting for data...
    '等待数据...': 'データを待っています...',
    // Never
    '从未': 'なし',
    // Upgrading...
    '升级中...': 'アップグレード中...',
    // Waiting for restart...
    '等待重启...': '再起動を待っています...',
    // Replicas
    '副本': 'レプリカ',
    // Role
    '角色': 'ロール',
    // Availability
    '可用性': '可用性',
    // Engine Version
    '引擎版本': 'エンジンバージョン',
    // Join Cluster
    '加入集群': 'クラスタに参加',
    // Force Redeploy
    '强制重部署': '強制再デプロイ',
    // Scale
    '扩缩容': 'スケール',
    // Add Node
    '接入节点': 'ノードを追加',
    // Revoke
    '吊销': '取り消し',
    // Revoke
    '撤销': '取り消し',
    // Selected
    '已选': '選択済み',
    // items
    '个': '件',
    // Offline
    '离线': 'オフライン',
    // Version incompatible
    '版本不兼容': 'バージョン非対応',
    // Failed to load
    '加载失败': '読み込みに失敗しました',
    // Search files...
    '搜索文件...': 'ファイルを検索...',
    // New Folder
    '新建目录': 'フォルダを新規作成',
    // Upload Files
    '上传文件': 'ファイルをアップロード',
    // Move
    '移动': '移動',
    // Modified
    '修改时间': '更新日時',
    // Compress
    '压缩': '圧縮',
    // Download
    '下载': 'ダウンロード',
    // Preview
    '预览': 'プレビュー',
    // Extract
    '解压': '展開',
    // Rename / Move
    '重命名 / 移动': '名前の変更 / 移動',
    // Open
    '进入': '開く',
    // Confirm changes
    '确认修改': '変更の確認',
    // Target Path
    '目标路径': '対象パス',
    // Close Assistant
    '关闭助手': 'アシスタントを閉じる',
    // Open Copilot debug panel
    '打开 Copilot 调试面板': 'Copilot デバッグパネルを開く',
    // Add New Passkey
    '绑定新 Passkey': '新しい Passkey を登録',
    // Manage members who can sign in and their permissions
    '管理可登录系统的成员与权限': 'サインインできるメンバーと権限を管理します',
    // Founder
    '创始人': '創設者',
    // Disabled
    '未启用': '無効',
    // 1 hour
    '1 小时': '1 時間',
    // 7 days
    '7 天': '7 日間',
    // Enter a token name
    '请输入令牌名称': 'トークン名を入力',
    // Copy Token
    '复制令牌': 'トークンをコピー',
    // Create Token
    '创建令牌': 'トークンを作成',
    // Close
    '关闭': '閉じる',
    // Contact the administrator to change account permissions
    '请联系管理员调整账号权限': 'アカウント権限の変更は管理者にお問い合わせください',
    // Search consumers...
    '搜索消费者...': 'コンシューマーを検索...',
    // Delete Route
    '删除路由': 'ルートを削除',
    // No routes yet
    '暂无路由': 'ルートはまだありません',
    // Click "New Route" to create one
    '点击「新建路由」开始创建': '「ルートを新規作成」をクリックして作成してください',
    // Delete Certificate
    '删除证书': '証明書を削除',
    // No certificates yet
    '暂无证书': '証明書はまだありません',
    // No matching certificate
    '未找到匹配证书': '一致する証明書が見つかりません',
    // Click "New Certificate" to create one
    '点击「新建证书」开始创建': '「証明書を新規作成」をクリックして作成してください',
    // New Upstream
    '新建上游': 'アップストリームを新規作成',
    // Policy
    '策略': 'ポリシー',
    // Search route or user...
    '搜索路由或用户...': 'ルートまたはユーザーを検索...',
    // Configure Authorization
    '配置授权': '権限を設定',
    // Authorized User
    '授权用户': '権限ユーザー',
    // Please fix the Plugin JSON syntax error
    '请修正 Plugin JSON 格式错误': 'Plugin の JSON 形式エラーを修正してください',
    // Route updated
    '路由更新成功': 'ルートを更新しました',
    // Route created
    '路由创建成功': 'ルートを作成しました',
    // Edit Route
    '编辑路由': 'ルートを編集',
    // Enter Host (optional)
    '请输入 Host（可选）': 'Host を入力（省略可）',
    // Enter Host (IP or container name)
    '请输入 Host（IP 或容器名）': 'Host を入力（IP またはコンテナ名）',
    // Enter Host (IP or domain)
    '请输入 Host（IP 或域名）': 'Host を入力（IP またはドメイン）',
    // Enter connect timeout (optional)
    '请输入连接超时（可选）': '接続タイムアウトを入力（省略可）',
    // Enter send timeout (optional)
    '请输入发送超时（可选）': '送信タイムアウトを入力（省略可）',
    // Enter read timeout (optional)
    '请输入读取超时（可选）': '読み取りタイムアウトを入力（省略可）',
    // Leave empty or 0 to use the APISIX default timeout (seconds).
    '留空或 0 表示使用 APISIX 默认超时（单位：秒）。': '空欄または 0 の場合は APISIX の既定タイムアウト（秒）を使用します。',
    // Edit Certificate
    '编辑证书': '証明書を編集',
    // Delete Node
    '删除节点': 'ノードを削除',
    // Confirm Config
    '确认配置': '設定を確認',
    // Submit Config
    '提交配置': '設定を送信',
    // Handler
    '处理器': 'ハンドラー',
    // Search route or username...
    '搜索路由或用户名...': 'ルートまたはユーザー名を検索...',
    // Configure Auth
    '配置认证': '認証を設定',
    // Forward Headers
    '转发头': '転送ヘッダー',
    // Account
    '账号': 'アカウント',
    // File on disk
    '磁盘文件': 'ディスク上のファイル',
    // Inline PEM
    '内联 PEM': 'インライン PEM',
    // Auto issue
    '自动签发': '自動発行',
    // (empty)
    '(空)': '(空)',
    // Expired {0} days ago
    '已过期 {0} 天': '期限切れから {0} 日',
    // Expires today
    '今日过期': '本日期限切れ',
    // Expires in {0} days
    '{0} 天后到期': '{0} 日後に期限切れ',
    // Source
    '来源': '取得元',
    // Subject
    '主体': 'サブジェクト',
    // Disabled
    '已禁用': '無効',
    // Delete Service
    '删除服务': 'サービスを削除',
    // Service deleted
    '服务删除成功': 'サービスを削除しました',
    // No services yet
    '暂无服务': 'サービスはまだありません',
    // No matching service
    '未找到匹配服务': '一致するサービスが見つかりません',
    // Automatic HTTPS
    '自动 HTTPS': '自動 HTTPS',
    // No listen address
    '无监听地址': 'リッスンアドレスがありません',
    // Save
    '保存': '保存する',
    // Format
    '格式': '形式',
    // Enter tags (optional)
    '请输入标签（可选）': 'タグを入力（省略可）',
    // Separate multiple tags with commas, e.g. example,prod
    '多个标签用逗号分隔，例如：example,prod': '複数のタグはカンマ区切りで入力します（例: example,prod）',
    // Service created
    '服务创建成功': 'サービスを作成しました',
    // Listen Address
    '监听地址': 'リッスンアドレス',
    // e.g. 30s
    '例如 30s': '例: 30s',
    // Write Timeout
    '写入超时': '書き込みタイムアウト',
    // Overwrite
    '覆盖': '上書き',
    // Append
    '追加': '追記',
    // Deploying...
    '部署中...': 'デプロイ中...',
    // Redeploy
    '重部署': '再デプロイ',
    // New Job
    '新建任务': 'ジョブを新規作成',
    // No matching job
    '未找到匹配任务': '一致するジョブが見つかりません',
    // Job Name
    '任务名称': 'ジョブ名',
    // Next Run
    '下次执行': '次回実行',
    // Last Run
    '上次执行': '最終実行',
    // Run Now
    '立即执行': '今すぐ実行',
    // Execution Logs
    '执行日志': '実行ログ',
    // Restart Policy
    '重启策略': '再起動ポリシー',
    // Runtime Config
    '运行配置': '実行時設定',
    // Privileged Mode
    '特权模式': '特権モード',
    // Yes
    '是': 'はい',
    // No
    '否': 'いいえ',
    // Mounts
    '挂载': 'マウント',
    // Start Command
    '启动命令': '起動コマンド',
    // No environment variables
    '无环境变量': '環境変数はありません',
    // Architecture
    '架构': 'アーキテクチャ',
    // Network Details
    '网络详情': 'ネットワーク詳細',
    // Volume Details
    '数据卷详情': 'ボリューム詳細',
    // Terminal
    '终端': 'ターミナル',
    // Pull Image
    '拉取镜像': 'イメージを取得',
    // Image pulled
    '镜像拉取成功': 'イメージを取得しました',
    // Top Level
    '顶层': 'トップレベル',
    // Build
    '构建': 'ビルド',
    // Prune Images
    '清理镜像': 'イメージを整理',
    // Tag
    '打标签': 'タグ付け',
    // Pull (update)
    '拉取（更新）': '取得（更新）',
    // No private registry available
    '暂无可用私有仓库': '利用できるプライベートレジストリはありません',
    // Push to Registry
    '推送到仓库': 'レジストリにプッシュ',
    // Search registry name, address or account...
    '搜索仓库名称、地址或账号...': 'レジストリ名、アドレス、アカウントで検索...',
    // No registries yet
    '暂无镜像仓库': 'レジストリはまだありません',
    // No matching registry
    '未找到匹配仓库': '一致するレジストリが見つかりません',
    // Search volume name, driver or mount point...
    '搜索卷名称、驱动或挂载点...': 'ボリューム名、ドライバ、マウント先で検索...',
    // New Volume
    '新建卷': 'ボリュームを新規作成',
    // Container Terminal
    '容器终端': 'コンテナターミナル',
    // Container Logs
    '容器日志': 'コンテナログ',
    // Show 50 lines
    '显示 50 行': '50 行を表示',
    // Show 100 lines
    '显示 100 行': '100 行を表示',
    // Show 200 lines
    '显示 200 行': '200 行を表示',
    // Show 500 lines
    '显示 500 行': '500 行を表示',
    // Show 1000 lines
    '显示 1000 行': '1000 行を表示',
    // 50 lines
    '50 行': '50 行',
    // 100 lines
    '100 行': '100 行',
    // 200 lines
    '200 行': '200 行',
    // 500 lines
    '500 行': '500 行',
    // 1000 lines
    '1000 行': '1000 行',
    // Waiting for log output...
    '等待日志输出...': 'ログ出力を待っています...',
    // No logs yet
    '暂无日志': 'ログはまだありません',
    // Read
    '读取': '読み取り',
    // Enter the container name
    '请输入容器名称': 'コンテナ名を入力',
    // Advanced Options
    '高级选项': '詳細設定',
    // matches
    '个匹配': '件一致',
    // Enter the image tag
    '请输入镜像标签': 'イメージタグを入力',
    // Image Source
    '镜像源': 'イメージの取得元',
    // Local Image
    '本地镜像': 'ローカルイメージ',
    // Driver Type
    '驱动类型': 'ドライバ種別',
    // Realtime system resource monitoring
    '实时系统资源监控': 'システムリソースのリアルタイム監視',
    // Failed to get system info
    '获取系统信息失败': 'システム情報の取得に失敗しました',
    // Disconnect
    '断开连接': '切断',
    // Search local processes
    '搜索本机进程': 'ローカルプロセスを検索',
    // Display Mode
    '展示方式': '表示形式',
    // Parent/child tree
    '父子树': '親子ツリー',
    // Sort by
    '排序字段': '並べ替え',
    // Start Time
    '启动时间': '起動時刻',
    // Descending
    '降序排列': '降順',
    // Ascending
    '升序排列': '昇順',
    // Read
    '读': '読み取り',
    // CPU Usage
    'CPU 使用率': 'CPU 使用率',
    // Memory Usage
    '内存使用': 'メモリ使用量',
    // Usage
    '使用率': '使用率',
    // Service Status Overview
    '服务状态总览': 'サービス状態の概要',
    // Total Routes
    '路由总数': 'ルート総数',
    // Search credential name or username...
    '搜索凭据名或用户名...': '認証情報名またはユーザー名で検索...',
    // Add Credential
    '添加凭据': '認証情報を追加',
    // Credential Name
    '凭据名称': '認証情報名',
    // Auth Method
    '认证方式': '認証方式',
    // Not set
    '未设置': '未設定',
    // Search hostname, address or username...
    '搜索主机名、地址或用户名...': 'ホスト名、アドレス、ユーザー名で検索...',
    // Add Host
    '添加主机': 'ホストを追加',
    // Saved Credentials
    '已保存凭据': '保存済みの認証情報',
    // Manual Authentication
    '手动认证': '手動認証',
    // Connect Terminal
    '连接终端': 'ターミナルに接続',
    // SSH Private Key
    'SSH 私钥': 'SSH 秘密鍵',
    // Enter the SSH private key
    '请输入 SSH 私钥': 'SSH 秘密鍵を入力',
    // Service Details
    '服务详情': 'サービス詳細',
    // Startup Parameters
    '启动参数': '起動パラメータ',
    // View Swarm cluster task status
    '查看 Swarm 集群任务状态': 'Swarm クラスタのタスク状態を表示',
    // All Services
    '全部服务': 'すべてのサービス',
    // Message
    '消息': 'メッセージ',
    // Node Details
    '节点详情': 'ノード詳細',
    // Manage Swarm cluster nodes
    '管理 Swarm 集群节点': 'Swarm クラスタのノードを管理',
    // IP Address
    'IP 地址': 'IP アドレス',
    // The leader node cannot be removed
    '不能移除 Leader 节点': 'Leader ノードは削除できません',
    // No matching node
    '未找到匹配节点': '一致するノードが見つかりません',
    // Copy Command
    '复制命令': 'コマンドをコピー',
    // Manage Swarm services
    '管理 Swarm 服务': 'Swarm サービスを管理',
    // Service Name
    '服务名': 'サービス名',
    // Service Logs
    '服务日志': 'サービスログ',
    // Reload
    '重载': '再読み込み',
    // Search user, method, URI, IP or status...
    '搜索用户、方法、URI、IP 或状态...': 'ユーザー、メソッド、URI、IP、状態で検索...',
    // All Users
    '所有用户': 'すべてのユーザー',
    // Obtained when registering the app with the OIDC provider
    '在 OIDC Provider 处注册应用时获得': 'OIDC プロバイダーでアプリを登録する際に取得します',
    // Admin API connection options
    'Admin API 连接参数': 'Admin API の接続設定',
    // Enter the Admin URL
    '请输入 Admin URL': 'Admin URL を入力',
    // Search node name, hostname, OS or creator...
    '搜索节点名、主机名、系统或创建人...': 'ノード名、ホスト名、OS、作成者で検索...',
    // Version
    '版本': 'バージョン',
    // Latency
    '延迟': '遅延',
    // Expired
    '过期': '期限切れ',
    // Enter Node
    '进入节点': 'ノードを開く',
    // Approve
    '审批通过': '承認',
    // View join command
    '查看接入命令': '参加コマンドを表示',
    // Review
    '审批': '審査',
    // Press Enter to add:
    '按 Enter 添加:': 'Enter で追加:',
    // Usage:
    '使用:': '使い方:',
    // Clear selected
    '清空已选项': '選択をクリア',
    // Select Icon
    '选择图标': 'アイコンを選択',
    // Search icon name...
    '搜索图标名称...': 'アイコン名を検索...',
    // Collapse
    '收起': '折りたたむ',
    // More
    '更多': 'その他',
    // No matching icon
    '未找到匹配的图标': '一致するアイコンが見つかりません',
    // Local
    '本机': 'ローカル',
    // Switch Node
    '切换节点': 'ノードを切り替え',
    // Local (controller)
    '本机（中控）': 'ローカル（コントローラ）',
    // No connected nodes yet
    '暂无已接入的节点': '接続済みのノードはまだありません',
    // Created
    '创建成功': '作成しました',
    // Drop to upload into the current directory
    '松手上传到当前目录': 'ドロップして現在のディレクトリにアップロード',
    // Folders supported
    '支持文件夹': 'フォルダに対応',
    // Check the network or permissions and refresh
    '请检查网络或权限后刷新重试': 'ネットワークや権限を確認してから再読み込みしてください',
    // This directory is empty
    '此目录为空': 'このディレクトリは空です',
    // No matching files
    '未找到匹配文件': '一致するファイルが見つかりません',
    // Upload files or create a folder to get started
    '上传文件或创建新目录开始使用': 'ファイルをアップロードするかフォルダを作成してください',
    // Select all visible files
    '选择全部可见文件': '表示中のファイルをすべて選択',
    // Calculating...
    '计算中...': '計算中...',
    // Calculate size
    '计算大小': 'サイズを計算',
    // Open Directory
    '进入目录': 'ディレクトリを開く',
    // Change Permissions
    '修改权限': '権限を変更',
    // Permissions (octal)
    '权限 (八进制)': '権限（8 進数）',
    // Enter file permissions
    '请输入文件权限': 'ファイルの権限を入力',
    // Three octal digits, e.g. 755, 644
    '三位八进制数，例如：755、644': '8 進数 3 桁（例: 755、644）',
    // Common permissions:
    '常用权限:': 'よく使う権限:',
    // Saving...
    '修改中...': '権限を変更中...',
    // File Name
    '文件名称': 'ファイル名',
    // Enter the file name
    '请输入文件名称': 'ファイル名を入力',
    // File Content
    '文件内容': 'ファイルの内容',
    // Enter file content...
    '请输入文件内容...': 'ファイルの内容を入力...',
    // Creating...
    '新建中...': '作成中...',
    // {0} selected
    '选中的 {0} 项': '選択中 {0} 件',
    // Delete
    '确定要删除': '削除の確認',
    // Contains directories; the directories and their contents will be deleted;
    '包含目录，目录及其内容将被删除；': 'ディレクトリが含まれます。その内容ごと削除されます。',
    // Deleting...
    '删除中...': '削除中...',
    // Enter the folder name
    '请输入目录名称': 'ディレクトリ名を入力',
    // Failed to load the file
    '文件加载失败': 'ファイルの読み込みに失敗しました',
    // The browser cannot preview PDF files
    '浏览器不支持 PDF 预览': 'このブラウザは PDF のプレビューに対応していません',
    // Open in a new tab
    '在新标签页打开': '新しいタブで開く',
    // Rename / Move {0} items
    '重命名 / 移动 {0} 项': '名前の変更 / 移動（{0} 件）',
    // Target directory, e.g. /backup/
    '目标目录，如：/backup/': '移動先ディレクトリ（例: /backup/）',
    // e.g. new.txt, /backup/new.txt, /backup/
    '如：new.txt、/backup/new.txt、/backup/': '例: new.txt、/backup/new.txt、/backup/',
    // ending with
    '以': '末尾が',
    //  → move into that directory keeping the name; otherwise → full target path (single file only)
    '结尾 → 移入该目录并保持原名；否则 → 完整目标路径（仅单文件）': 'の場合 → そのディレクトリへ名前を保持したまま移動します。それ以外 → 完全な移動先パス（単一ファイルのみ）',
    // Operation preview (total
    '操作预览（共': '操作のプレビュー（全',
    // items)
    '项）': '件）',
    // Original Path
    '原路径': '元のパス',
    // Enter the target path
    '请输入目标路径': '移動先のパスを入力',
    // Confirm Extract
    '解压确认': '展開の確認',
    // Extract
    '确定要解压': '展開の確認',
    // If the target directory is empty, files are extracted into the current directory
    '目标目录留空时，文件将解压到当前目录': '移動先を空にすると、現在のディレクトリに展開されます',
    // Enter a directory name, e.g. output
    '请输入目录名，如：output': 'ディレクトリ名を入力（例: output）',
    // Only a directory name is allowed, without / or other path separators
    '只能输入目录名，不允许包含 / 等路径分隔符': 'ディレクトリ名のみ入力できます。/ などの区切り文字は使えません',
    // Extracting...
    '解压中...': '展開中...',
    // Start Extracting
    '开始解压': '展開を開始',
    // "{0}" is already in the upload queue
    '「{0}」已在上传队列中': '「{0}」はすでにアップロード待ちです',
    // Upload failed, leftover files have been cleaned up
    '上传失败，已尝试清理残留文件': 'アップロードに失敗しました。残ったファイルの削除を試みました',
    // Upload failed, incomplete files may remain
    '上传失败，可能残留不完整文件': 'アップロードに失敗しました。不完全なファイルが残っている可能性があります',
    // Uploaded ({0} files)
    '上传成功（{0} 个文件）': '{0} 件のファイルをアップロードしました',
    // {0} succeeded, {1} failed{2}
    '{0} 个成功，{1} 个失败{2}': '{0} 件成功、{1} 件失敗{2}',
    // Upload Queue
    '上传队列': 'アップロード待ち',
    // Confirm Compress
    '压缩确认': '圧縮の確認',
    // Compress
    '确定要压缩': '圧縮の確認',
    // The archive will be saved in the current directory
    '压缩后的文件将保存在当前目录': '圧縮したファイルは現在のディレクトリに保存されます',
    // Compressing...
    '压缩中...': '圧縮中...',
    // Start Compressing
    '开始压缩': '圧縮を開始',
    // Save File
    '保存文件': 'ファイルを保存',
    // Generating request parameters…
    '正在生成请求参数…': 'リクエストパラメータを生成中…',
    // This request changes server state, please confirm before executing.
    '此请求会修改服务器状态，请确认后执行。': 'このリクエストはサーバーの状態を変更します。確認のうえ実行してください。',
    // View request parameters
    '查看请求参数': 'リクエストパラメータを見る',
    // Resource List
    '资源列表': 'リソース一覧',
    // Only the first
    '仅展示前': '先頭',
    // items are shown, the full result is still provided to the AI assistant.
    '项，完整结果仍会提供给 AI 助手。': '件のみ表示します。完全な結果は AI アシスタントに渡されます。',
    // Request executed
    '请求执行成功': 'リクエストを実行しました',
    // The browser or environment does not support Passkey (HTTPS and WebAuthn required)
    '当前浏览器或环境不支持 Passkey（需要 HTTPS 且浏览器支持 WebAuthn）': '現在のブラウザまたは環境は Passkey に対応していません（HTTPS と WebAuthn 対応ブラウザが必要です）',
    // Passkey added!
    'Passkey 绑定成功！': 'Passkey を登録しました！',
    // Credential renamed
    '凭证已重命名': '認証情報を名前変更しました',
    // Delete Passkey
    '删除 Passkey': 'Passkey を削除',
    // Delete this Passkey credential?
    '确定要删除这个 Passkey 凭证吗？': 'この Passkey 認証情報を削除しますか？',
    // Passkey credential deleted
    'Passkey 凭证已删除': 'Passkey 認証情報を削除しました',
    // Add or remove Passkeys used for passwordless sign-in
    '绑定或移除用于免密登录的 Passkey': 'パスワードレスサインイン用の Passkey を登録・削除します',
    // No Passkey yet
    '暂无 Passkey': 'Passkey はまだありません',
    // Click "Add New Passkey" above to set it up
    '点击上方「绑定新 Passkey」开始设置': '上部の「新しい Passkey を登録」をクリックして設定してください',
    // Added at
    '添加于': '登録日時',
    // Credential Name
    '凭证名称': '認証情報名',
    // e.g. MacBook Touch ID
    '如：MacBook Touch ID': '例: MacBook Touch ID',
    // Make sure your device supports Passkey (Touch ID, Face ID or a security key).
    '请确保您的设备支持 Passkey（如 Touch ID、Face ID 或安全密钥）。': 'お使いのデバイスが Passkey に対応していることを確認してください（Touch ID、Face ID、セキュリティキーなど）。',
    // Registering...
    '绑定中...': '登録中...',
    // Delete Member
    '删除成员': 'メンバーを削除',
    // Delete member <strong class="text-slate-900">{0}</strong>? This only removes it from the config file, the home directory is kept.
    '确定要删除成员 <strong class="text-slate-900">{0}</strong> 吗？此操作仅从配置文件移除，不删除家目录。': 'メンバー <strong class="text-slate-900">{0}</strong> を削除しますか？この操作は設定ファイルから削除するだけで、ホームディレクトリは削除されません。',
    // No members yet
    '暂无成员': 'メンバーはまだいません',
    // No matching member
    '未找到匹配成员': '一致するメンバーが見つかりません',
    // Click "New User" to create a member
    '点击「新建用户」创建成员': '「ユーザーを新規作成」をクリックしてメンバーを作成してください',
    // Identity
    '身份': '種別',
    // Enter the code from your authenticator
    '请输入认证器中的验证码': '認証アプリのコードを入力',
    // TOTP two-factor auth is enabled
    'TOTP 二次验证已启用': 'TOTP 二段階認証が有効です',
    // Enter the current code
    '请输入当前验证码': '現在のコードを入力',
    // TOTP two-factor auth is disabled
    'TOTP 二次验证已禁用': 'TOTP 二段階認証が無効です',
    // Secret copied to clipboard
    '密钥已复制到剪贴板': 'シークレットをクリップボードにコピーしました',
    // Enter the new password
    '请输入新密码': '新しいパスワードを入力',
    // The two passwords do not match
    '两次输入的密码不一致': '2 つのパスワードが一致しません',
    // The password must be at least {0} characters
    '密码长度至少 {0} 位': 'パスワードは {0} 文字以上で入力してください',
    // Update the login password and manage two-factor auth for password sign-in
    '更新登录密码，并管理密码登录的二次验证': 'ログインパスワードの変更と、パスワードサインインの二段階認証を管理します',
    // Password & 2FA
    '密码与二次验证': 'パスワードと二段階認証',
    // Change
    '修改': '変更',
    // Current Password
    '原密码': '現在のパスワード',
    // Enter the current password
    '请输入原密码': '現在のパスワードを入力',
    // New Password
    '新密码': '新しいパスワード',
    // Confirm Password
    '确认密码': 'パスワード（確認）',
    // Enter the new password again
    '请再次输入新密码': '新しいパスワードを再入力',
    // After changing the password, issued API Keys become invalid, please recreate them as needed.
    '修改密码后，已签发的 API Key 将自动失效，请按需重新创建。': 'パスワードを変更すると、発行済みの API Key は無効になります。必要に応じて再作成してください。',
    // Two-factor auth (TOTP)
    '二次验证（TOTP）': '二段階認証（TOTP）',
    // Only triggered by password sign-in, Passkey, OIDC and API Token are unaffected.
    '仅账号密码登录触发，不影响 Passkey、OIDC 和 API Token。': 'パスワードサインイン時のみ動作し、Passkey、OIDC、API Token には影響しません。',
    // Cannot scan? Enter the secret manually
    '无法扫码？手动输入密钥': 'スキャンできませんか？シークレットを手動で入力',
    // Copy
    '复制': 'コピー',
    // Verification Code
    '验证码': '認証コード',
    // Enter the 6-digit code from your app
    '请输入 App 中的 6 位验证码': 'アプリの 6 桁のコードを入力',
    // Bind
    '绑定': '登録',
    // Enter the current code to disable
    '输入当前验证码以禁用': '無効にするには現在のコードを入力',
    // Enter the 6-digit code from your authenticator
    '请输入认证器中的 6 位验证码': '認証アプリの 6 桁のコードを入力',
    // 24 hours
    '24 小时': '24 時間',
    // 30 days
    '30 天': '30 日間',
    // 90 days
    '90 天': '90 日間',
    // 365 days
    '365 天': '365 日間',
    // Token created
    '令牌创建成功': 'トークンを作成しました',
    // Token copied to clipboard
    '令牌已复制到剪贴板': 'トークンをクリップボードにコピーしました',
    // Create an account token for automation
    '创建用于自动化调用的账号令牌': '自動化のためのアカウントトークンを作成します',
    // Automation Token
    '自动化调用令牌': '自動化トークン',
    // Token Name
    '令牌名称': 'トークン名',
    // Generated Result
    '生成结果': '生成結果',
    // The token is shown only once after creation.
    '令牌只在创建后显示一次。': 'トークンは作成後に 1 回だけ表示されます。',
    // Enter a name and click create, the generated token appears here.
    '填写名称后点击创建，生成的令牌会显示在这里。': '名前を入力して作成をクリックすると、生成されたトークンがここに表示されます。',
    // How to call
    '调用方式': '呼び出し方法',
    // Security Policy
    '安全策略': 'セキュリティポリシー',
    // Shown only once after creation, copy and save it now.
    '创建后仅显示一次，请立即复制保存。': '作成後に 1 回だけ表示されます。すぐにコピーして保存してください。',
    // After changing the password, issued API Keys become invalid.
    '修改密码后，已签发的 API Key 自动失效。': 'パスワードを変更すると、発行済みの API Key は自動的に無効になります。',
    // Pick the shortest validity that fits the task.
    '按任务选择最短可用有效期。': '用途に応じて最短の有効期限を選んでください。',
    // No permission to create API Keys
    '无权限创建 API Key': 'API Key を作成する権限がありません',
    // Edit Member
    '编辑成员': 'メンバーを編集',
    // New Member
    '新建成员': 'メンバーを新規作成',
    // Please enter the login password
    '请填写登录密码': 'ログインパスワードを入力してください',
    // The username cannot be changed
    '用户名不可修改': 'ユーザー名は変更できません',
    // (leave empty to keep unchanged)
    '(留空则保持不变)': '(空欄の場合は変更しません)',
    // Enter the home directory (optional)
    '请输入家目录（可选）': 'ホームディレクトリを入力（省略可）',
    // Enter a member description (optional)
    '请输入成员描述（可选）': 'メンバーの説明を入力（省略可）',
    // Describes the member purpose, up to 64 characters
    '用于标识成员用途，最长 64 字符': 'メンバーの用途を示すメモ（最大 64 文字）',
    // Route Permissions
    '路由权限': 'ルート権限',
    // Select all
    '全选': 'すべて選択',
    // Auto
    '自动': '自動',
    // Delete Consumer
    '删除消费者': 'コンシューマーを削除',
    // Delete consumer <strong class="text-slate-900">{0}</strong>? This action cannot be undone.
    '确定要删除消费者 <strong class="text-slate-900">{0}</strong> 吗？此操作不可恢复。': 'コンシューマー <strong class="text-slate-900">{0}</strong> を削除しますか？この操作は取り消せません。',
    // Manage APISIX consumers and their credentials
    '管理 APISIX Consumer 及其认证凭据': 'APISIX のコンシューマーと認証情報を管理します',
    // Manage consumers and credentials
    '管理 Consumer 与凭据': 'コンシューマーと認証情報を管理',
    // No consumers yet
    '暂无消费者': 'コンシューマーはまだありません',
    // No matching consumer
    '未找到匹配消费者': '一致するコンシューマーが見つかりません',
    // Click "New Consumer" to create one
    '点击「新建消费者」开始创建': '「コンシューマーを新規作成」をクリックして作成してください',
    // Authorized Routes
    '授权路由': '権限ルート',
    // Delete Plugin Config
    '删除插件配置': 'プラグイン設定を削除',
    // Delete plugin config <strong class="text-slate-900">{0}</strong>? APISIX may refuse while it is referenced by routes.
    '确定要删除插件配置 <strong class="text-slate-900">{0}</strong> 吗？仍被路由引用时 APISIX 可能拒绝删除。': 'プラグイン設定 <strong class="text-slate-900">{0}</strong> を削除しますか？ルートから参照されている場合、APISIX が削除を拒否することがあります。',
    // Manage reusable plugin sets referenced by routes
    '管理可复用的插件集合，供路由引用': 'ルートから参照できる、再利用可能なプラグインセットを管理します',
    // New Config
    '新建配置': '設定を新規作成',
    // Manage reusable plugin sets
    '管理可复用插件集合': '再利用可能なプラグインセットを管理',
    // No plugin configs yet
    '暂无插件配置': 'プラグイン設定はまだありません',
    // No matching plugin config
    '未找到匹配插件配置': '一致するプラグイン設定が見つかりません',
    // Click "New Config" to add a reusable Plugin Config
    '点击「新建配置」添加可复用 Plugin Config': '「設定を新規作成」をクリックして再利用可能な Plugin Config を追加してください',
    // Configure
    '配置': '設定',
    // {0} Route
    '{0}路由': 'ルート{0}',
    // {0} route <strong class="text-slate-900">{1}</strong>?
    '确定要{0}路由 <strong class="text-slate-900">{1}</strong> 吗？': '確定: ルート{0} <strong class="text-slate-900">{1}</strong>',
    // Route {0}
    '路由已{0}': 'ルート{0}完了',
    // Delete route <strong class="text-slate-900">{0}</strong>? This action cannot be undone.
    '确定要删除路由 <strong class="text-slate-900">{0}</strong> 吗？此操作不可恢复。': 'ルート <strong class="text-slate-900">{0}</strong> を削除しますか？この操作は取り消せません。',
    // Manage APISIX routes, matching rules, upstream forwarding and plugins
    '管理 APISIX 路由，配置匹配规则、上游转发与插件': 'APISIX のルートを管理し、一致条件、アップストリーム転送、プラグインを設定します',
    // Search route, URI, description or upstream...
    '搜索路由、URI、描述或上游...': 'ルート、URI、説明、アップストリームで検索...',
    // Configure matching rules, upstream and plugins
    '配置匹配规则、上游与插件': '一致条件、アップストリーム、プラグインを設定',
    // Search route, URI, upstream...
    '搜索路由、URI、上游...': 'ルート、URI、アップストリームで検索...',
    // {0} and {1} domains
    '{0} 等 {1} 个域名': '{0} ほか {1} ドメイン',
    // Delete certificate <strong class="text-slate-900">{0}</strong>? HTTPS access may be affected while it is used by SNI.
    '确定要删除证书 <strong class="text-slate-900">{0}</strong> 吗？正在被 SNI 使用时可能影响 HTTPS 访问。': '証明書 <strong class="text-slate-900">{0}</strong> を削除しますか？SNI で使用中の場合、HTTPS アクセスに影響する可能性があります。',
    // Manage APISIX SSL certificate bindings and SNI config
    '管理 APISIX 的 SSL 证书绑定与 SNI 配置': 'APISIX の SSL 証明書バインドと SNI 設定を管理します',
    // Search certificate, SNI or ID...
    '搜索证书、SNI 或 ID...': '証明書、SNI、ID で検索...',
    // Manage certificates and SNI bindings
    '管理证书与 SNI 绑定': '証明書と SNI バインドを管理',
    // Search certificate or SNI...
    '搜索证书或 SNI...': '証明書または SNI で検索...',
    // Delete Upstream
    '删除上游': 'アップストリームを削除',
    // Delete upstream <strong class="text-slate-900">{0}</strong>? APISIX may refuse while it is referenced by routes.
    '确定要删除上游 <strong class="text-slate-900">{0}</strong> 吗？仍被路由引用时 APISIX 可能拒绝删除。': 'アップストリーム <strong class="text-slate-900">{0}</strong> を削除しますか？ルートから参照されている場合、APISIX が削除を拒否することがあります。',
    // Manage reusable backend upstreams and load balancing policies
    '管理可复用的后端上游对象与负载均衡策略': '再利用可能なバックエンドのアップストリームと負荷分散ポリシーを管理します',
    // Submit
    '提交': '送信する',
    // Settings
    '设置': '設定',
    // Local Processes
    '本机进程': 'ローカルプロセス',
    // Note
    '备注': '備考',
}

const language: LanguageModule = {
    code: 'ja',
    label: '日本語',
    short: 'JA',
    htmlLang: 'ja-JP',
    tags: ['ja'],
    messages
}

export default language
