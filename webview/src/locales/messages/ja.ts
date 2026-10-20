import type { LanguageModule } from '../types'

/**
 * 日本語訳：キーは中国語原文（簡体字）
 *
 * 英語辞書のキーをすべて翻訳しています。新しい文言を追加したときは
 * scripts/check-locales.mjs で訳し忘れを確認してください。
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
    // Central Control
    '集中管控': '集中管理',
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
    // Switch language
    '切换语言': '言語を切り替え',
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
    '添加图片或文件': 'イメージまたはファイルを追加',
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
    '清理': 'クリーンアップ',
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
    // ?
    '吗？': '?',
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
    '清理镜像': 'イメージをクリーンアップ',
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
    // Search node name, hostname or OS...
    '搜索节点名、主机名或系统...': 'ノード名、ホスト名、OS で検索...',
    // Version
    '版本': 'バージョン',
    // Latency
    '延迟': '遅延',
    // Expired
    '过期': '期限切れ',
    // View join command
    '查看接入命令': '参加コマンドを表示',
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
    // File saved
    '文件保存成功': 'ファイルは正常に保存されました',
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
    // Start Registration
    '开始绑定': 'バインディングの開始',
    // Delete Member
    '删除成员': 'メンバーを削除',
    // Delete member <strong class="text-slate-900">{0}</strong>? This only removes it from the config file, the home directory is kept.
    '确定要删除成员 <strong class="text-slate-900">{0}</strong> 吗？此操作仅从配置文件移除，不删除家目录。': 'メンバー <strong class="text-slate-900">{0}</strong> を削除しますか？この操作は設定ファイルから削除するだけで、ホームディレクトリは削除されません。',
    // Member deleted
    '成员删除成功': 'メンバーが正常に削除されました',
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
    // Password updated
    '密码修改成功': 'パスワードが正常に変更されました',
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
    // Member updated
    '成员更新成功': 'メンバーの更新が成功しました',
    // Member added
    '成员添加成功': 'メンバーが正常に追加されました',
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
    '确定要{0}路由 <strong class="text-slate-900">{1}</strong> 吗？': '{0}操作をルート <strong class="text-slate-900">{1}</strong> に対して実行してもよろしいですか？',
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
    // Search upstream, node or policy...
    '搜索上游、节点或策略...': 'アップストリーム、ノード、またはポリシーを検索...',
    // Manage reusable upstreams
    '管理可复用上游对象': '再利用可能なアップストリームオブジェクトを管理する',
    // Search upstream or node...
    '搜索上游、节点...': 'アップストリーム、ノードを検索...',
    // No upstreams yet
    '暂无上游': 'アップストリームはまだありません',
    // No matching upstream
    '未找到匹配上游': '一致するアップストリームが見つかりません',
    // Click "New Upstream" to create one
    '点击「新建上游」开始创建': '「新規アップストリーム」をクリックして作成を開始します',
    // Hash
    '哈希': 'ハッシュ',
    // Enter the key-auth key
    '请输入 key-auth key': 'キー認証キーを入力してください',
    // User "{0}" is already in the authorization list
    '用户 "{0}" 已在授权列表中': 'ユーザー「{0}」はすでに承認リストに含まれています',
    // User "{0}" created and added to the authorization list
    '用户 "{0}" 已创建并加入授权列表': 'ユーザー「{0}」が作成され、承認リストに追加されました',
    // Configure route-level consumer access control
    '配置路由级别的 Consumer 访问授权': 'ルーティングレベルのコンシューマアクセス認可を構成する',
    // Route-level Consumer Access Control
    '路由级 Consumer 访问授权': 'ルートレベルのコンシューマアクセス許可',
    // No access control data yet
    '暂无访问授权数据': 'アクセス許可データがまだありません',
    // No matching access control entry
    '未找到匹配的访问授权': '一致するアクセス許可が見つかりませんでした',
    // Shown here after configuring route-level consumer access control
    '配置路由的 Consumer 访问授权后将在此显示': 'ルートのコンシューマ アクセス許可を設定すると、ここに表示されます。',
    // Users
    '用户数': 'ユーザー数',
    // New Authorized User
    '新建授权用户': '新しい許可されたユーザーを作成する',
    // Create a consumer and add it to the current route access control
    '新建 Consumer 并加入当前路由访问授权': '新しいコンシューマを作成し、現在のルーティング アクセス許可を追加します。',
    // Enter the consumer username
    '请输入 Consumer 用户名': '一般ユーザーのユーザー名を入力してください',
    // Enter the API Key
    '请输入 API Key': 'APIキーを入力してください',
    // On submit a consumer is created with key-auth, then added to the route access control
    '提交后会先创建 Consumer 并配置 key-auth，再加入路由访问授权': '送信後、コンシューマが作成され、キー認証が設定され、ルーティング アクセス許可が追加されます。',
    // Create and add
    '创建并加入': '作成して参加する',
    // The consumer name cannot be empty
    '消费者名称不能为空': 'コンシューマ名を空にすることはできません',
    // Edit Consumer
    '编辑消费者': 'コンシューマの編集',
    // Enter the consumer name
    '请输入消费者名称': '消費者名を入力してください',
    // Enter a consumer description (optional)
    '请输入消费者描述（可选）': '顧客の説明を入力してください (オプション)',
    // At least one plugin must be configured
    '至少需要配置一个插件': '少なくとも 1 つのプラグインを構成する必要があります',
    // Edit Plugin Config
    '编辑插件配置': 'プラグイン設定を編集する',
    // New Plugin Config
    '新建插件配置': '新しいプラグイン構成',
    // Config ID
    '配置 ID': '構成ID',
    // The ID is generated by the create API and cannot be changed when editing.
    'ID 由创建接口自动生成，编辑时不能修改。': 'ID は作成インターフェイスによって自動的に生成され、編集中に変更することはできません。',
    // Enter a plugin config description (optional)
    '请输入插件配置描述（可选）': 'プラグイン構成の説明を入力してください (オプション)',
    // Plugin {0} already exists
    '插件 {0} 已存在': 'プラグイン {0} はすでに存在します',
    // Select the route to import from first
    '请先选择要导入的路由': '最初にインポートするルートを選択してください',
    // Select at least one plugin
    '请至少勾选一个插件': '少なくとも 1 つのプラグインをチェックしてください',
    // Imported {0} plugins
    '已导入 {0} 个插件': '{0} 個のプラグインがインポートされました',
    // Edit the JSON directly, or import from an existing route
    '可直接编辑 JSON，也支持从现有路由导入': 'JSON は直接編集することも、既存のルートからインポートすることもできます。',
    // Edit the JSON directly, or add plugins from the list
    '可直接编辑 JSON，或从插件列表添加': 'JSON は直接編集することも、プラグイン リストから追加することもできます',
    // Add Plugin
    '添加插件': 'プラグインを追加',
    // Import from Route
    '从路由导入': 'ルートからインポート',
    // Enter a plugin name...
    '请输入插件名称...': 'プラグイン名を入力してください...',
    // Select the source route...
    '选择来源路由...': 'ソースルートを選択...',
    // Import selected
    '导入选中': '選択したインポート',
    // plugins
    '个插件': 'プラグイン',
    // This route has no plugin config
    '该路由没有插件配置': 'このルートにはプラグイン構成がありません',
    // Select the upstream to reference
    '请选择要引用的上游对象': '参照するアップストリームオブジェクトを選択してください',
    // Enter the upstream host and port
    '请填写上游主机和端口': 'アップストリームのホストとポートを入力してください',
    // Upstream host and port must be filled together
    '上游主机和端口需要同时填写': 'アップストリームのホストとポートを同時に入力する必要があります',
    // The upstream port must be a number
    '上游端口必须为数字': 'アップストリームポートは数値である必要があります',
    // No nodes
    '无节点': 'ノードなし',
    // Unnamed upstream
    '未命名上游': 'アップストリームの名前なし',
    // Description: {0}
    '描述: {0}': '説明: {0}',
    // Type: {0}
    '类型: {0}': 'タイプ: {0}',
    // Nodes: {0}
    '节点: {0}': 'ノード: {0}',
    // Failed to load route details
    '加载路由详情失败': 'ルートの詳細を読み込めませんでした',
    // The route name cannot be empty
    '路由名称不能为空': 'ルート名を空にすることはできません',
    // URI cannot be empty
    'URI 不能为空': 'URI を空にすることはできません',
    // Enter the route name
    '请输入路由名称': '路線名を入力してください',
    // Priority
    '优先级': '優先順位',
    // Enter the priority (optional)
    '请输入优先级（可选）': '優先順位を入力してください (オプション)',
    // Enter a route description (optional)
    '请输入路由描述（可选）': 'ルートの説明を入力してください (オプション)',
    // URI (one per line)
    'URI（每行一个）': 'URI (1 行に 1 つ)',
    // Enter a URI (wildcards supported)
    '请输入 URI（支持通配符）': 'URI を入力してください (ワイルドカードをサポート)',
    // e.g. /api/v1/* or /api/v2/*
    '例如：/api/v1/* 或 /api/v2/*': '例: /api/v1/* または /api/v2/*',
    // Host (one per line, empty matches all)
    'Host（每行一个，留空匹配所有）': 'ホスト (1 行に 1 つ、すべてに一致させるには空白のままにします)',
    // e.g. example.com or *.example.com
    '例如：example.com 或 *.example.com': '例: example.com または *.example.com',
    // WebSocket Proxy
    'WebSocket 代理': 'WebSocketプロキシ',
    // Enables WebSocket protocol upgrade (ws:// / wss://)
    '启用后支持 WebSocket 协议升级（ws:// / wss://）': '有効な場合、WebSocket プロトコルのアップグレード (ws:// / wss://) をサポートします',
    // Upstream Config
    '上游配置': 'アップストリーム構成',
    // Enter a single upstream directly, reference an existing one, or configure none
    '支持直接输入单个上游、引用已有上游或暂不配置上游': '単一のアップストリームを直接入力すること、既存のアップストリームを参照すること、またはアップストリームをまだ構成していないことをサポートします。',
    // e.g. Host 127.0.0.1 or nginx, Port 8080
    '例如：Host 填写 127.0.0.1 或 nginx，Port 填写 8080': '例: ホストに 127.0.0.1 または nginx を入力し、ポートに 8080 を入力します。',
    // Direct input submits a single upstream node; for load balancing across nodes, create it in "Upstreams" first and reference it.
    '直接输入模式仅提交一个上游节点；如需多节点负载均衡，请先在「上游管理」中创建后再引用。': '直接入力モードは 1 つのアップストリームノードのみを送信します。マルチノード負荷分散が必要な場合は、まず「アップストリーム管理」で作成してから参照してください。',
    // HTTPS Upstream
    'HTTPS 回源': 'HTTPS をオリジンに戻す',
    // Forwards to this upstream node over HTTPS (HTTP by default)
    '启用后以 HTTPS 协议转发到该上游节点（默认 HTTP）': '有効にすると、HTTPS プロトコル (デフォルト HTTP) を使用してアップストリームノードに転送されます。',
    // Select an existing upstream
    '请选择已有上游': '既存のアップストリームを選択してください',
    // Saving only submits matching rules, status and plugin config, without inline nodes or referenced upstreams.
    '当前保存只会提交路由匹配规则、状态和插件配置，不会附带内联节点，也不会引用已有上游。': '現在の保存では、ルート一致ルール、ステータス、およびプラグイン設定のみが送信されます。これにはインライン ノードは含まれず、既存のアップストリームは参照されません。',
    // Reference Plugin Config
    '引用插件配置': 'リファレンスプラグイン設定',
    // Select an existing plugin config object, merged with the standalone plugin config
    '选择已有的插件配置对象，与独立插件配置合并生效': '既存のプラグイン設定オブジェクトを選択し、それを独立したプラグイン設定とマージして有効にします。',
    // None
    '不使用': '未使用',
    // At least one SNI domain is required
    '至少需要填写一个 SNI 域名': '少なくとも 1 つの SNI ドメイン名が必要です',
    // The certificate content cannot be empty
    '证书内容不能为空': '証明書の内容を空にすることはできません',
    // The private key content cannot be empty
    '私钥内容不能为空': '秘密キーの内容を空にすることはできません',
    // SNI Domains
    'SNI 域名': 'SNI ドメイン名',
    // Enter SNI (one per line)
    '请输入 SNI（每行一个）': 'SNI を入力してください (1 行に 1 つ)',
    // One domain per line, wildcards supported, e.g. example.com or *.example.com
    '每行一个域名，支持通配符，例如：example.com 或 *.example.com': '1 行に 1 つのドメイン名、ワイルドカードがサポートされています (例: example.com または *.example.com)。',
    // One domain per line, or separated by commas.
    '每行一个域名，也支持用英文逗号分隔。': '1 行に 1 つのドメイン名をカンマで区切って入力します。',
    // Certificate Content
    '证书内容': '証明書の内容',
    // Enter the certificate content in PEM format
    '请输入 PEM 格式证书内容': 'PEM 形式の証明書の内容を入力してください',
    // PEM certificate starting with -----BEGIN CERTIFICATE-----. Leave empty when editing to keep the current certificate.
    'PEM 格式证书，以 -----BEGIN CERTIFICATE----- 开头。编辑时留空表示不修改当前证书。': '-----BEGIN CERTIFICATE----- で始まる PEM 形式の証明書。現在の証明書を変更しないように編集する場合は空白のままにします。',
    // Private Key Content
    '私钥内容': '秘密鍵の内容',
    // Enter the private key content in PEM format
    '请输入 PEM 格式私钥内容': '秘密キーの内容を PEM 形式で入力してください',
    // PEM private key starting with -----BEGIN PRIVATE KEY-----. Leave empty when editing to keep the current key.
    'PEM 格式私钥，以 -----BEGIN PRIVATE KEY----- 开头。编辑时留空表示不修改当前私钥。': '-----BEGIN PRIVATE KEY----- で始まる PEM 形式の秘密キー。現在の秘密キーを変更しないように編集する場合は空白のままにします。',
    // Enabled, the certificate is in effect
    '启用后证书生效': '証明書は有効にすると有効になります',
    // The upstream name cannot be empty
    '上游名称不能为空': 'アップストリーム名を空にすることはできません',
    // At least one valid node is required
    '至少需要配置一个有效节点': '少なくとも 1 つの有効なノードを構成する必要があります',
    // Edit Upstream
    '编辑上游': 'アップストリームで編集する',
    // Create Upstream
    '创建上游': 'アップストリームの作成',
    // Enter the upstream name
    '请输入上游名称': 'アップストリーム名を入力してください',
    // Load Balancing Policy
    '负载均衡策略': '負荷分散戦略',
    // Upstream Protocol
    '回源协议': '原点回帰契約',
    // Enter an upstream description (optional)
    '请输入上游描述（可选）': 'アップストリームの説明を入力してください (オプション)',
    // Node Config
    '节点配置': 'ノード構成',
    // Fill in the backend address and weight
    '填写后端服务地址和权重': 'バックエンドサービスのアドレスと重みを入力します',
    // Add Node
    '添加节点': 'ノードの追加',
    // e.g. 127.0.0.1 or nginx
    '例如：127.0.0.1 或 nginx': '例: 127.0.0.1 または nginx',
    // e.g. 8080
    '例如：8080': '例: 8080',
    // Weight
    '权重': '重量',
    // Edit Route Access Control
    '编辑路由访问授权': 'ルーティングアクセス許可の編集',
    // Configure Route Access Control
    '配置路由访问授权': 'ルーティングアクセス許可を構成する',
    // Select the route to configure access control for
    '请选择要配置访问授权的路由': 'アクセス認可を設定したいルートを選択してください',
    // The authorized user cannot be empty
    '授权用户不能为空': '許可されたユーザーを空にすることはできません',
    // Access control updated
    '访问授权更新成功': 'アクセス許可が正常に更新されました',
    // Access control deleted
    '访问授权删除成功': 'アクセス権限が正常に削除されました',
    // Edit the key-auth and consumer-restriction.whitelist config of the current route
    '编辑当前路由的 key-auth 和 consumer-restriction.whitelist 配置': '現在のルートの key-auth および Consumer-restriction.whitelist 構成を編集します',
    // Select a route without access control configured
    '请选择未配置访问授权的路由': 'アクセス許可が設定されていないルートを選択してください',
    // On save the key-auth and consumer-restriction.whitelist plugins are configured for the selected route
    '保存后会为所选路由配置 key-auth 和 consumer-restriction.whitelist 插件': '保存後、選択したルートに対して key-auth プラグインと Consumer-restriction.whitelist プラグインが設定されます。',
    // Only routes without access control are shown
    '仅展示尚未配置访问授权的路由': 'アクセス許可が設定されていないルートのみを表示します',
    // No route available for access control
    '暂无可配置访问授权的路由': '現在、アクセス許可を構成するルートはありません。',
    // key-auth Plugin Config
    'key-auth 插件配置': 'キー認証プラグインの設定',
    // Configure route authentication plugin parameters
    '配置路由认证插件参数': 'ルーティング認証プラグインのパラメータを構成する',
    // e.g. token
    '例如：token': '例: トークン',
    // Clients pass the Consumer API Key through this header
    '客户端通过该 Header 传递 Consumer API Key': 'クライアントはこのヘッダーを通じてコンシューマ API キーを渡します',
    // Query Parameter Name
    '查询参数名称': 'クエリパラメータ名',
    // e.g. token (optional)
    '例如：token（可选）': '例: トークン (オプション)',
    // Leave empty to disable reading the key from query parameters
    '留空表示不启用 Query 参数取 key': 'クエリパラメータキーを無効にするには、空白のままにします。',
    // Hide Credentials
    '隐藏认证凭据': '認証資格情報を非表示にする',
    // When enabled, credentials are removed from the request before forwarding upstream
    '开启后转发到上游前移除请求中的认证凭据': '有効にした後、アップストリームに転送する前に、リクエストから認証資格情報を削除します。',
    // Search and select consumers, multiple selection allowed
    '搜索并选择 Consumer，可多选': 'Consumer を検索して選択します。複数の選択が可能です',
    // Search consumer
    '搜索 Consumer': '消費者の検索',
    // available
    '个可选': 'オプション',
    // No consumer with key-auth configured
    '暂无已配置 key-auth 的 Consumer': '現在、キー認証が構成されているコンシューマはありません。',
    // All selected
    '全部已选': 'すべて選択済み',
    // Only consumers with key-auth are shown; create new users in Consumer management first
    '仅展示已配置 key-auth 插件的 Consumer；如需添加新用户，请先在 Consumer 管理中创建': 'キー認証プラグインを構成したコンシューマのみが表示されます。新しいユーザーを追加する必要がある場合は、最初に消費者管理でユーザーを作成してください',
    // Invalid JSON, check the error hint in the editor
    'JSON 格式有误，请检查编辑器中的错误提示': 'JSON 形式が正しくありません。エディタでエラー メッセージを確認してください。',
    // Submit Full Config
    '提交完整配置': '完全な構成を送信する',
    // This replaces the whole running Caddy config, please verify your edits.
    '将整体替换 Caddy 当前运行配置，请确认编辑内容无误。': 'Caddy の現在の実行構成は完全に置き換えられます。編集内容が正しいかご確認ください。',
    // Confirm Submit
    '确认提交': '送信を確認',
    // Config applied
    '配置已应用': '設定が適用されました',
    // View and edit the running Caddy JSON config directly
    '直接查看和编辑 Caddy 的 JSON 运行配置': 'Caddy の JSON 実行構成を直接表示および編集します',
    // Submitting...
    '提交中...': '送信中...',
    // View and edit the JSON config
    '查看和编辑 JSON 配置': 'JSON 構成の表示と編集',
    // The submission goes through
    '提交将通过': '送信時に',
    // This replaces the whole running Caddy config, make sure you have a backup.
    '整体替换 Caddy 运行配置，操作前请确保已备份。': 'Caddy の実行構成を全体的に置き換えます。操作前に必ずバックアップを行ってください。',
    // (custom)
    '(自定义)': '（カスタム）',
    // Delete route <strong class="text-slate-900">#{1}</strong> of service <strong class="text-slate-900">{0}</strong>? This action cannot be undone.
    '确定要删除服务 <strong class="text-slate-900">{0}</strong> 下的路由 <strong class="text-slate-900">#{1}</strong> 吗？此操作不可恢复。': 'サービス <strong class="text-slate-900">{0}</strong> のルート <strong class="text-slate-900">#{1}</strong> を削除してもよろしいですか?この操作は元に戻すことができません。',
    // Configure request matching rules and handlers, multiple forwarding modes supported
    '配置请求匹配规则与处理器，支持多种转发方式': '複数の転送方法をサポートするようにリクエスト一致ルールとプロセッサを構成する',
    // Configure matching rules and handlers
    '配置匹配规则与处理器': '一致ルールとプロセッサを構成する',
    // Backend
    '后端': 'バックエンド',
    // Route HTTP Basic authentication based on username + password
    '基于用户名 + 密码的路由 HTTP Basic 认证': 'ユーザー名とパスワードに基づく HTTP 基本認証のルーティング',
    // Route HTTP Basic Authentication
    '路由 HTTP Basic 认证': 'HTTP 基本認証のルーティング',
    // No basic auth configured
    '暂无基础认证': '基本的な認定はまだありません',
    // Click "Configure Auth" to enable HTTP Basic auth for any route
    '点击「配置认证」为任意路由启用 HTTP Basic 认证': '「認証の構成」をクリックして、任意のルートに対して HTTP 基本認証を有効にします。',
    // Count
    '数量': '件数',
    // Issued
    '已签发': '発行された',
    // Delete certificate <strong class="text-slate-900">{0}</strong>? This action cannot be undone.
    '确定要删除证书 <strong class="text-slate-900">{0}</strong> 吗？此操作不可恢复。': '証明書 <strong class="text-slate-900">{0}</strong> を削除してもよろしいですか?この操作は元に戻すことができません。',
    // Manage Caddy certificate sources: file, PEM, auto issue; issued certificates are read-only
    '管理 Caddy 证书来源：文件、PEM、自动签发；已签发证书只读': 'Caddy 証明書ソースの管理: ファイル、PEM、自動発行。発行された証明書は読み取り専用です',
    // Manage certificate sources
    '管理证书来源': '証明書ソースの管理',
    // Service details not found
    '服务详情不存在': 'サービスの詳細が存在しません',
    // srv0 is the default service of the structured API and cannot be deleted
    'srv0 是结构化 API 的默认服务，不可删除': 'srv0 は構造化 API のデフォルトのサービスであり、削除できません',
    // At least one service must be kept
    '至少保留一个服务': '少なくとも 1 つのサービスを維持する',
    // Delete this service? All routes under it are deleted as well.
    '确定删除这个服务吗？其下全部路由也会被删除。': 'このサービスを削除してもよろしいですか?その下のルートもすべて削除されます。',
    // Manage named services such as srv0, listen addresses and protocols
    '管理 srv0 等命名服务、监听地址和协议': 'ネームサービス、リスニングアドレス、srv0 などのプロトコルを管理します。',
    // Search name, listen address or protocol...
    '搜索名称、监听或协议...': '名前、リスナー、またはプロトコルを検索します...',
    // Manage listen addresses and protocols
    '管理监听地址和协议': 'リスニングアドレスとプロトコルを管理する',
    // Search services...
    '搜索服务...': '検索サービス...',
    // Click "New Service" to create one
    '点击「新建服务」开始创建': '「新しいサービス」をクリックして作成を開始します',
    // Global options saved
    '全局选项已保存': 'グローバルオプションが保存されました',
    // Configure TLS automation, log level and service options
    '配置 TLS 自动化、日志级别与服务参数': 'TLS 自動化、ログ レベル、サービス パラメータを構成する',
    // TLS, Logging and Service
    'TLS、日志与服务参数': 'TLS、ログ、サービスパラメータ',
    // Certificate Issuance
    '证书签发': '証明書の発行',
    // ACME Email
    'ACME 邮箱': 'ACMEメール',
    // Enter the ACME email
    '请输入 ACME 邮箱': 'ACME メールアドレスを入力してください',
    // ACME Directory URL
    'ACME 目录 URL': 'ACME ディレクトリの URL',
    // Enter the ACME directory URL (optional)
    '请输入 ACME 目录 URL（可选）': 'ACME ディレクトリの URL を入力してください (オプション)',
    // Use a local self-signed certificate (internal issuer)
    '使用本地自签证书（internal issuer）': 'ローカルの自己署名証明書を使用する (内部発行者)',
    // Skips ACME, Caddy issues a locally trusted certificate; ACME email and directory settings are ignored
    '不走 ACME，由 Caddy 自动签发本地信任证书；启用后 ACME 邮箱和目录设置将被忽略': 'ACME を使用せず、Caddy がローカルで信頼される証明書を自動発行します。有効にすると ACME のメールアドレスとディレクトリ設定は無視されます。',
    // Enable On-Demand TLS
    '启用 On-Demand TLS': 'オンデマンドTLSを有効にする',
    // Certificates are requested on demand, suited to multi-tenant setups with changing domains; in production combine with
    '连接时动态申请证书，适合域名数量不固定的多租户场景；生产环境需配合': '接続時に証明書を動的に適用します。ドメイン名の数が固定されていないマルチテナントのシナリオに適しています。本番環境は連携する必要がある',
    // Ask endpoint abuse protection
    '端点防滥用': 'エンドポイントの悪用防止',
    // Ask Endpoint
    'Ask 鉴权端点': '認証エンドポイントに問い合わせる',
    // Enter the Ask endpoint (optional)
    '请输入 Ask 鉴权端点（可选）': 'Ask 認証エンドポイントを入力してください（任意）',
    // Caddy sends a GET request to this URL before issuing; 2xx allows it, e.g. http://localhost:9090/tls-ask; empty means no check (testing only)
    'Caddy 在申请证书前会向此 URL 发起 GET 请求，返回 2xx 则允许，例如：http://localhost:9090/tls-ask；留空时不配置鉴权（仅测试环境使用）': 'Caddy は証明書の発行前にこの URL へ GET リクエストを送信します。2xx が返ると許可します。例：http://localhost:9090/tls-ask。空欄では認証を設定しません（テスト環境のみ）。',
    // Log Level
    '日志级别': 'ログレベル',
    // Default (INFO)
    '默认（INFO）': 'デフォルト(情報)',
    // Global default log level
    '全局默认日志级别': 'グローバルなデフォルトのログレベル',
    // Log Format
    '日志格式': 'ログ形式',
    // Default (console)
    '默认（console）': 'デフォルト(コンソール)',
    // Structured log output format
    '结构化日志输出格式': '構造化されたログの出力形式',
    // Graceful Shutdown Wait
    '优雅关闭等待': '正常なシャットダウンの待機',
    // Enter the graceful shutdown wait time (optional)
    '请输入优雅关闭等待时间（可选）': '正常なシャットダウンの待ち時間を入力してください (オプション)',
    // Maximum time to wait for existing connections on reload/shutdown
    '重载/关闭时等待现有连接结束的最长时间': 'リロード/クローズ時に既存の接続が終了するまで待機する最大時間',
    // After saving it goes through
    '保存后通过': '保存後にパスする',
    // The whole running Caddy config is replaced atomically and takes effect immediately.
    '整体原子替换 Caddy 运行配置，立即生效。': 'Caddy 実行構成の完全なアトミック置換は、すぐに有効になります。',
    // Edit Authentication Config
    '编辑认证配置': '認証構成の編集',
    // Add Authentication Config
    '添加认证配置': '認証構成を追加する',
    // Leave empty to clear the Realm
    '留空将清空 Realm': '空白のままにするとレルムがクリアされます',
    // Leave empty to not set a Realm
    '留空则不设置 Realm': 'レルムを設定しない場合は空白のままにします',
    // Select the target route
    '请选择目标路由': '目的地のルートを選択してください',
    // Authentication configured
    '认证配置成功': '認証構成が成功しました',
    // Account "{0}" added
    '账号 "{0}" 添加成功': 'アカウント「{0}」が正常に追加されました',
    // Route not found
    '路由不存在': 'ルートが存在しません',
    // Config updated
    '配置更新成功': '構成の更新が成功しました',
    // Target Route
    '目标路由': '目的地ルート',
    // Select a route
    '请选择路由': 'ルートを選択してください',
    // All routes have authentication configured
    '所有路由均已配置认证': 'すべてのルートに認証が設定されています',
    // Description shown in the HTTP Basic auth dialog
    'HTTP Basic 认证弹窗中显示的描述文字': 'HTTP 基本認証ポップアップ ウィンドウに表示される説明テキスト',
    // Pass the authenticated username upstream
    '将认证用户名传给后端': '認証ユーザー名をバックエンドに渡す',
    // After authentication, the username is injected into the given header and forwarded upstream
    '认证通过后，将登录用户名注入到指定请求头中转发给上游服务': '認証に合格した後、ログイン ユーザー名が指定されたリクエスト ヘッダーに挿入され、アップストリームのサービスに転送されます。',
    // Caddy Variable
    'Caddy 变量': 'キャディ変数',
    // Will be injected into this header
    '将被注入到此 Header': 'このヘッダーに挿入されます',
    // Existing Accounts
    '现有账号': '既存のアカウント',
    // No accounts yet
    '暂无账号': 'まだアカウントがありません',
    // Search accounts
    '搜索账号': 'アカウントを検索する',
    // recoverable
    '个可恢复': '件を復元可能',
    // All accounts selected
    '已选择全部账号': 'すべてのアカウントが選択されました',
    // Click the × on an account tag to remove it, effective after saving
    '点击账号标签上的 × 标记移除，保存后生效': 'アカウントラベルの×マークをクリックして削除すると、保存後に有効になります。',
    // Passwords are stored as bcrypt hashes in the Caddy config
    '密码将以 bcrypt 形式存储在 Caddy 配置中': 'パスワードは Caddy 設定に bcrypt として保存されます。',
    // Enter the certificate file path
    '请填写证书文件路径': '証明書ファイルのパスを入力してください',
    // Enter the private key file path
    '请填写私钥文件路径': '秘密鍵ファイルのパスを入力してください',
    // Enter the certificate PEM content
    '请填写证书 PEM 内容': '証明書の PEM コンテンツを入力してください',
    // Enter the private key PEM content
    '请填写私钥 PEM 内容': '秘密キーの PEM コンテンツを入力してください',
    // Enter the hostname to issue a certificate for
    '请填写需要自动签发的主机名': '自動発行するホスト名を入力してください',
    // Certificate updated
    '证书更新成功': '証明書が正常に更新されました',
    // Certificate created
    '证书创建成功': '証明書が正常に作成されました',
    // The certificate source cannot be changed when editing
    '编辑时不允许切换证书来源类型': '編集中に証明書ソースの種類を切り替えることはできません',
    // Certificate Path
    '证书路径': '証明書のパス',
    // Enter the certificate path
    '请输入证书路径': '証明書のパスを入力してください',
    // Absolute path of the certificate file, e.g. /etc/caddy/cert.pem
    '证书文件绝对路径，例如：/etc/caddy/cert.pem': '証明書ファイルへの絶対パス (例: /etc/caddy/cert.pem)',
    // Private Key Path
    '私钥路径': '秘密鍵のパス',
    // Enter the private key path
    '请输入私钥路径': '秘密キーのパスを入力してください',
    // Absolute path of the private key file, e.g. /etc/caddy/key.pem
    '私钥文件绝对路径，例如：/etc/caddy/key.pem': '秘密キー ファイルの絶対パス (例: /etc/caddy/key.pem)',
    // Enter the certificate format (optional)
    '请输入证书格式（可选）': '証明書の形式を入力してください (オプション)',
    // Certificate format, leave empty for the default PEM
    '证书格式，留空使用默认 PEM': '証明書の形式。デフォルトの PEM を使用する場合は空白のままにします',
    // Certificate PEM
    '证书 PEM': '証明書PEM',
    // Format: -----BEGIN CERTIFICATE----- ... -----END CERTIFICATE-----
    '格式：-----BEGIN CERTIFICATE----- ... -----END CERTIFICATE-----': '形式: -----BEGIN CERTIFICATE----- ... -----END CERTIFICATE-----',
    // Private Key PEM
    '私钥 PEM': '秘密キー PEM',
    // Format: -----BEGIN PRIVATE KEY----- ... -----END PRIVATE KEY-----
    '格式：-----BEGIN PRIVATE KEY----- ... -----END PRIVATE KEY-----': '形式: -----BEGIN PRIVATE KEY----- ... -----END PRIVATE KEY-----',
    // Caddy requests and renews the certificate via ACME, e.g. example.com
    'Caddy 将通过 ACME 自动为该主机申请并续期证书，例如：example.com': 'Caddy は、ACME を通じてホストの証明書を自動的に申請し、更新します (例: example.com)。',
    // Submit
    '提交': '送信する',
    // Enter the service name
    '请输入服务名称': 'サービス名を入力してください',
    // Enter at least one listen address
    '请至少填写一个监听地址': '少なくとも 1 つのリスニング アドレスを入力してください',
    // H1 must be enabled when H2 or H2C is enabled
    '启用 H2 或 H2C 时必须同时启用 H1': 'H2 または H2C を有効にする場合は、H1 も有効にする必要があります',
    // Service updated
    '服务更新成功': 'サービスの更新が成功しました',
    // Edit Service
    '编辑服务': '編集サービス',
    // Service Name
    '服务名称': 'サービス名',
    // e.g. srv0
    '例如 srv0': 'たとえば、srv0',
    // The name cannot be changed after creation and is kept as is.
    '名称创建后不可修改，并会原样保留。': '名前は作成後に変更できず、そのまま保持されます。',
    // One address per line, e.g. :80, :443 or 127.0.0.1:8080.
    '每行一个地址，例如 :80、:443 或 127.0.0.1:8080。': ':80、:443、または 127.0.0.1:8080 など、1 行に 1 つのアドレス。',
    // Service Protocols
    '服务协议': 'サービス契約',
    // Select the service protocols
    '请选择服务协议': 'サービス契約を選択してください',
    // Search protocols
    '搜索协议': '調査契約',
    // No matching protocol
    '未找到匹配协议': '一致するプロトコルが見つかりません',
    // Multiple selection supported; Caddy defaults apply when empty. H2 and H2C depend on H1.
    '支持多选；未选择时使用 Caddy 默认协议。H2 与 H2C 依赖 H1。': '複数の選択をサポートします。選択されていない場合は、Caddy のデフォルト プロトコルを使用します。 H2 と H2C は H1 に依存します。',
    // Idle Timeout
    '空闲超时': 'アイドルタイムアウト',
    // e.g. 5m
    '例如 5m': '例えば5m',
    // Max Header Bytes
    '最大请求头字节数': 'リクエストヘッダーの最大バイト数',
    // 0 means the default value is used
    '0 表示使用默认值': '0 はデフォルト値を使用することを意味します',
    // Strict SNI Host Matching
    '严格校验 SNI Host': '厳密な検証 SNI ホスト',
    // Follow Caddy default rules
    '跟随 Caddy 默认规则': 'Caddy のデフォルト ルールに従ってください',
    // The default rules enable strict matching automatically when TLS client auth is configured.
    '默认规则会在配置 TLS 客户端认证时自动启用严格校验。': 'デフォルトのルールでは、TLS クライアント認証を構成するときに厳密な検証が自動的に有効になります。',
    // Disable Automatic HTTPS
    '禁用自动 HTTPS': '自動HTTPSを無効にする',
    // Disable HTTPS Redirects
    '禁用 HTTPS 重定向': 'HTTPS リダイレクトを無効にする',
    // Keeps automatic HTTPS but does not redirect HTTP requests to HTTPS.
    '保留自动 HTTPS，但不将 HTTP 请求重定向到 HTTPS。': '自動 HTTPS を維持しますが、HTTP リクエストを HTTPS にリダイレクトしないでください。',
    // Enter the route JSON
    '请输入路由 JSON': 'ルート JSON を入力してください',
    // Invalid JSON
    'JSON 格式错误': 'JSON形式エラー',
    // The handle field cannot be empty
    'handle 字段不能为空': 'ハンドルフィールドを空にすることはできません',
    // Complete the handler config
    '请完善处理器配置': 'プロセッサーの構成を完了してください',
    // Failed to parse JSON
    'JSON 解析失败': 'JSONの解析に失敗しました',
    // The handler cannot be empty
    '处理器不能为空': 'プロセッサーを空にすることはできません',
    // The service of a route cannot be changed
    '路由所属服务不可修改': 'ルートが属するサービスは変更できません。',
    // Select the service the new route belongs to
    '选择新路由所属的服务': '新しいルートが属するサービスを選択します',
    // Match Conditions
    '匹配条件': '一致条件',
    // One per line, empty matches all, e.g. example.com or *.example.com
    '每行一个，留空匹配所有，例如：example.com 或 *.example.com': '1 行に 1 つ、すべてに一致させるには空白のままにします (例: example.com または *.example.com)。',
    // Enter Path (optional)
    '请输入 Path（可选）': 'パスを入力してください (オプション)',
    // One per line, * wildcard supported, e.g. /api/* or /static/*
    '每行一个，支持 * 通配符，例如：/api/* 或 /static/*': '1 行に 1 つ、* ワイルドカードをサポートします。例: /api/* または /static/*',
    // Enter Method (optional)
    '请输入 Method（可选）': '方法を入力してください (オプション)',
    // Space separated, empty matches all, e.g. GET POST
    '空格分隔，留空匹配所有，例如：GET POST': 'スペースで区切って、すべてに一致させるには空白のままにします。例: GET POST',
    // Any
    '不限': '制限なし',
    // Empty matches all
    '留空匹配所有': 'すべて一致させるには空白のままにします',
    // Only forwards requests with the given headers, empty matches all
    '仅转发包含指定请求头的请求，留空匹配所有': '指定されたリクエストヘッダーを含むリクエストのみを転送し、空のままにするとすべてに一致します',
    // Value (empty matches if present)
    '值（留空表示存在即匹配）': '値 (存在する場合は空白のままにして一致させます)',
    // Add Header Match
    '添加匹配请求头': '一致するリクエストヘッダーを追加',
    // Enter a container name (optional)
    '请输入容器名（可选）': 'コンテナ名を入力してください (オプション)',
    // Enter a container name or IP; the first upstream below is filled automatically, manual input is also allowed
    '可输入容器名或 IP 地址，选择后自动填充下方第一个上游，也可手动输入': 'コンテナ名または IP アドレスを入力できます。選択後に下の最初のアップストリームが自動的に入力されるか、手動で入力できます。',
    // Upstream Address
    '上游地址': 'アップストリームアドレス',
    // Enter upstream addresses (one per line)
    '请输入上游地址（每行一个）': 'アップストリームアドレスを入力してください (1 行に 1 つ)',
    // One host:port per line, multiple upstreams are polled, e.g. backend1:8080
    '每行一个 host:port，多个上游自动轮询，例如：backend1:8080': '1 行に 1 つのホスト:ポート、複数のアップストリームが自動的にポーリングされます (例: backend1:8080)',
    // Response Header Timeout
    '响应头超时': '応答ヘッダーのタイムアウト',
    // FastCGI Protocol
    'FastCGI 协议': 'FastCGIプロトコル',
    // For FastCGI processes such as PHP-FPM
    '适用于 PHP-FPM 等 FastCGI 进程': 'PHP-FPM などの FastCGI プロセスに適しています',
    // FastCGI Document Root
    'FastCGI 文档根目录': 'FastCGI ドキュメントのルート ディレクトリ',
    // Enter the FastCGI root directory (optional)
    '请输入 FastCGI 根目录（可选）': 'FastCGI ルート ディレクトリを入力してください (オプション)',
    // Settings
    '设置': '設定',
    // , leave empty to omit root, e.g. /var/www/html
    '，留空不传 root，例如：/var/www/html': '空白のままにし、root を渡さないでください。例: /var/www/html',
    // , only affects the request URI forwarded upstream
    '，仅影响转发给上游的请求 URI': '、アップストリームに転送されたリクエスト URI にのみ影響します。',
    // Upstream URI Rewrite
    '代理前 URI 重写': 'プロキシ URI の事前書き換え',
    // Override Request Method
    '覆盖请求方法': 'リクエストメソッドをオーバーライドする',
    // e.g. GET, POST; leave empty to keep unchanged
    '如 GET、POST，留空不修改': 'GET、POSTなど、空白のままにして変更しないでください。',
    // Replace URI
    'URI 整体替换': 'URI全体の置換',
    // e.g. /backend/{http.request.uri.path.1}
    '如 /backend/{http.request.uri.path.1}': '/backend/{http.request.uri.path.1} など',
    // Fully replaces the URI forwarded upstream, leave empty to keep unchanged
    '完整替换转发给上游的 URI，留空不修改': 'アップストリームに転送された URI を完全に置き換え、空白のままにして変更しないでください。',
    // Trim Path Prefix/Suffix
    '路径前后缀裁剪': 'パスのプレフィックスとサフィックスのクリッピング',
    // Strip prefix
    '去掉前缀': 'プレフィックスを削除する',
    // e.g. /api
    '如 /api': '/api など',
    // Strip suffix
    '去掉后缀': 'サフィックスを削除する',
    // e.g. .php
    '如 .php': '.phpなど',
    // Substring Replace
    '子串查找替换': '部分文字列の検索と置換',
    // Find
    '查找': '探す',
    // e.g. /old
    '如 /old': '/old など',
    // Replace with
    '替换为': 'と置き換えます',
    // e.g. /new
    '如 /new': '/new など',
    // Root Directory
    '根目录': 'ルートディレクトリ',
    // Enter the root directory
    '请输入根目录': 'ルートディレクトリを入力してください',
    // Root directory of static files, e.g. /var/www/html
    '静态文件根目录，例如：/var/www/html': '静的ファイルのルート ディレクトリ (例: /var/www/html)',
    // Enable Directory Browsing
    '启用目录浏览': 'ディレクトリの参照を有効にする',
    // Allows visitors to browse the file list of a directory
    '允许访客浏览目录内文件列表': '訪問者がディレクトリ内のファイルのリストを参照できるようにする',
    // Status Code
    '状态码': 'ステータスコード',
    // Enter the status code
    '请输入状态码': 'ステータスコードを入力してください',
    // HTTP status code, e.g. 200
    'HTTP 状态码，例如：200': 'HTTP ステータス コード (例: 200)',
    // Response Body
    '响应体': 'レスポンスボディ',
    // Enter the response body
    '请输入响应体': '応答本文を入力してください',
    // Leave empty for the default response, e.g. OK
    '留空则返回默认响应，例如：OK': 'デフォルトの応答を返すには、空白のままにします。例: OK',
    // URI Replace
    'URI 替换': 'URIの置換',
    // Enter the URI replace rule
    '请输入 URI 替换规则': 'URI 置換ルールを入力してください',
    // Fully replaces the request URI, Caddy placeholders supported, e.g. /new-path/{http.request.uri.path.1}
    '完整替换请求 URI，支持 Caddy 占位符，例如：/new-path/{http.request.uri.path.1}': '完全な置換リクエスト URI。Caddy プレースホルダーをサポートします。例: /new-path/{http.request.uri.path.1}',
    // Strip Path Prefix
    '去掉路径前缀': 'パスプレフィックスを削除する',
    // Enter the prefix to strip
    '请输入要去掉的前缀': '削除するプレフィックスを入力してください',
    // e.g.
    '例如：': '例：',
    // , result:
    '，效果：': '、結果：',
    // Strip Path Suffix
    '去掉路径后缀': 'パス接尾辞を削除する',
    // Enter the suffix to strip
    '请输入要去掉的后缀': '削除するサフィックスを入力してください',
    // Strips the given suffix from the end of the path, e.g. .php
    '从路径末尾去掉指定后缀，例如：.php': 'パスの末尾から指定されたサフィックスを削除します (例: .php)。',
    // Find Substring
    '子串查找': '部分文字列検索',
    // Enter the substring to find
    '请输入要查找的子串': '探している部分文字列を入力してください',
    // e.g. /old
    '例如：/old': '例: /old',
    // Replace Substring
    '子串替换': '部分文字列の置換',
    // Enter the replacement
    '请输入替换内容': '置換内容を入力してください',
    // e.g. /new
    '例如：/new': '例:/新しい',
    // Route JSON
    '路由 JSON': 'ルートJSON',
    // Edit the full Caddy route JSON directly, the match field is optional
    '直接编辑完整的 Caddy 路由 JSON，match 字段可选': '完全な Caddy ルーティング JSON を直接編集します。一致フィールドはオプションです。',
    // Configure header operations
    '配置请求头/响应头操作': 'リクエストヘッダー/レスポンスヘッダーの動作を設定する',
    // Chained middleware executed before the handler, placeholders such as
    '串联中间件，在处理器前执行，支持占位符如': 'プロセッサの前で実行される連結ミドルウェアは、次のようなプレースホルダーをサポートします。',
    // e.g. field X-Real-IP, value {http.request.remote.host}
    '例如：字段 X-Real-IP，值 {http.request.remote.host}': '例: フィールド X-Real-IP、値 {http.request.remote.host}',
    // Enter the request header name
    '请输入请求头名称': 'リクエストヘッダー名を入力してください',
    // Enter the request header value (placeholders supported)
    '请输入请求头值（支持占位符）': 'リクエストヘッダー値を入力してください (プレースホルダーをサポート)',
    // Add Request Header Operation
    '添加请求头操作': 'リクエストヘッダー追加オペレーション',
    // e.g. field X-Frame-Options, value SAMEORIGIN
    '例如：字段 X-Frame-Options，值 SAMEORIGIN': '例: フィールド X-Frame-Options、値 SAMEORIGIN',
    // Enter the response header name
    '请输入响应头名称': '応答ヘッダー名を入力してください',
    // Enter the response header value
    '请输入响应头值': '応答ヘッダー値を入力してください',
    // Add Response Header Operation
    '添加响应头操作': '応答ヘッダーの追加オペレーション',
    // Issued Cache
    '已签发缓存': '発行されたキャッシュ',
    // Loads the certificate and key from a disk path, for files already present on the host or in the container.
    '从磁盘路径加载证书和私钥，适合宿主机或容器内已有证书文件。': 'ホストまたはコンテナー内の既存の証明書ファイルに適したディスク パスから証明書と秘密キーを読み込みます。',
    // Stores the PEM certificate in the Caddy config, the private key is never returned to the frontend.
    '直接在 Caddy 配置中保存 PEM 证书内容，私钥内容不会返回前端。': 'PEM 証明書の内容を Caddy 構成に直接保存します。秘密キーの内容はフロントエンドに返されません。',
    // Issued and renewed automatically by Caddy ACME, the validity in details depends on the runtime cache.
    '由 Caddy ACME 自动签发与续期，详情中的有效期依赖运行时缓存。': 'Caddy ACME によって自動的に発行および更新され、詳細の有効期間はランタイム キャッシュに依存します。',
    // Certificate cache issued by the Caddy runtime, read-only, cannot be edited or deleted.
    'Caddy 运行时已签发证书缓存，只读展示，不支持编辑或删除。': 'Caddy が発行した証明書のキャッシュです。読み取り専用で、編集や削除はできません。',
    // SSL Certificate Details
    'SSL 证书详情': 'SSL証明書の詳細',
    // Runtime cache, read-only
    '运行时缓存，只读': 'ランタイムキャッシュ、読み取り専用',
    // Configured certificate, manageable
    '配置证书，可管理': '証明書の構成と管理',
    // Valid From
    '生效时间': '効果時間',
    // Valid Until
    '过期时间': '有効期限',
    // Remaining
    '剩余时间': '残り時間',
    // Certificate Details
    '证书解析': '証明書の解析',
    // SAN Domains
    'SAN 域名': 'SAN ドメイン名',
    // Source Config
    '来源配置': 'ソース構成',
    // Certificate Content / Path
    '证书内容 / 路径': '証明書の内容/パス',
    // Auto-issued Domain
    '自动签发域名': 'ドメイン名を自動発行する',
    // The private key is not returned; leave empty when editing to keep it
    '响应不返回私钥；编辑时留空表示保持原值': '応答では秘密キーは返されません。編集時に空白のままにすると、元の値が保持されることになります。',
    // Failed to stage the template, please retry
    '暂存模板失败，请重试': '一時テンプレートが失敗しました。もう一度お試しください。',
    // Selecting an app jumps to the deploy page with the template filled in
    '选择应用后将自动跳转到部署页并回填模板': 'アプリケーションを選択すると、自動的に展開ページにジャンプし、テンプレートをバックフィルします。',
    // Select to jump to the deploy page
    '选择后跳转部署页回填': '選択後、展開ページのバックフィルにジャンプします',
    // Marketplace is not configured yet
    '尚未配置应用市场': 'アプリマーケットはまだ設定されていません',
    // Go to "System Settings → Marketplace" to configure the site URL
    '请前往「系统设置 → 应用市场」配置站点 URL': '「システム設定→App Market」に進み、サイトURLを設定してください。',
    // Go to Settings
    '前往配置': '設定に移動',
    // The project name comes from the name field of the compose file; if the compose file references variables, fill them in .env
    '项目名来自 compose 文件的 name 字段；如 compose 中引用了环境变量，请在 .env 中填写对应变量': 'プロジェクト名は、作成ファイルの名前フィールドから取得されます。 Compose で環境変数が参照されている場合は、.env に対応する変数を入力してください。',
    // Template prefilled from the marketplace, deploy it directly or adjust it first
    '已从应用市场预填模板，可在此基础上直接部署或调整后再部署': 'テンプレートはアプリケーション マーケットから事前に入力されており、直接展開することも、展開前に調整することもできます。',
    // Unknown error
    '未知错误': '不明なエラー',
    // Unexpected template format, loaded as is: {0}
    '模板格式异常，已按原文加载：{0}': 'テンプレートの形式が異常で、元のテキストに従ってロードされました: {0}',
    // {0} deployed, {1} {2} created
    '{0} 部署成功，已创建 {1} 个{2}': '{0} が正常にデプロイされ、{1} {2} が作成されました',
    // Paste a compose.yml, or pick a template from the marketplace on the left
    '直接粘贴 compose.yml，或从左侧应用市场选择模板后回填': 'compose.yml を直接貼り付けるか、左側のアプリケーション マーケットからテンプレートを選択して埋め戻します。',
    // Paste compose.yml, or fill in from the marketplace
    '粘贴 compose.yml，或从应用市场回填': 'compose.yml を貼り付けるか、App Market からバックフィルします。',
    // Standalone Container
    '单机容器': 'スタンドアロンコンテナ',
    // Swarm is not enabled on this node
    '当前节点未启用 Swarm': '現在のノードでは Swarm が有効になっていません',
    // Swarm Service
    'Swarm 服务': 'スウォームサービス',
    // Additional Files
    '附加文件': '添付ファイル',
    // (optional, extracted into the project directory before deploy)
    '（选填，部署前解压到项目目录）': '(オプションで、展開前にプロジェクト ディレクトリに解凍します)',
    // Enter the zip download URL
    '请输入 zip 下载 URL': 'zip ダウンロード URL を入力してください',
    // or
    '或': 'または',
    // Use either a URL or an upload, only .zip is supported
    'URL 与上传文件二选一，仅支持 .zip 格式': 'URL とアップロードされたファイルのどちらかを選択します。.zip 形式のみがサポートされています',
    // In Swarm mode additional files are only written to the manager node; for sharing across nodes configure the container data root as NFS or other shared storage
    'Swarm 模式下，附加文件仅落盘到管理节点；如需各节点共享，请将容器数据根目录配置为 NFS 等共享存储': 'Swarm モードでは、追加ファイルは管理ノードにのみ配置されます。すべてのノードを共有する必要がある場合は、コンテナ データのルート ディレクトリを NFS などの共有ストレージとして構成してください。',
    // (KEY=VALUE, used for Compose variable interpolation)
    '（KEY=VALUE，用于 Compose 变量插值）': '(Compose 変数補間の場合は KEY=VALUE)',
    // Rebuilding...
    '重建中...': '再構築中...',
    // Loading...
    '读取中...': '読んでいます...',
    // Rebuild with This Config
    '使用此配置重建': 'この構成で再構築します',
    // Update and Rebuild
    '更新并重建': '更新と再構築',
    // Before Change
    '变更前': '変更前',
    // This is derived from the running state, please review before submitting
    '当前为运行态反推结果，建议核对后再提交': '現在の反転結果が実行状態になります。提出前に確認することをお勧めします。',
    // File updated at: {0}
    '文件更新时间：{0}': 'ファイル更新時間: {0}',
    // Compose config re-derived from the running state
    '已从运行态重新反推 Compose 配置': 'Compose 構成が実行状態からプッシュバックされました',
    // Config Version
    '配置版本': '構成バージョン',
    // Uncommitted changes
    '未提交修改': 'コミットされていない変更',
    // Retry loading deploy records
    '重试加载部署记录': 'デプロイメントレコードのロードを再試行します',
    // Refresh Deploy Records
    '刷新部署记录': '導入記録を更新する',
    // Current Config
    '当前配置': '現在の構成',
    // Loaded historical config
    '已加载的历史配置': 'ロードされた履歴構成',
    // Loading deploy records...
    '正在加载部署记录...': 'デプロイメントレコードをロードしています...',
    // Failed to load deploy records
    '部署记录加载失败': 'デプロイメントレコードのロードに失敗しました',
    // No deploy records yet
    '暂无部署记录': '導入実績はまだありません',
    // The historical config replaces the current one and rebuilds the instance; volumes and database contents are not restored.
    '历史配置将替换当前配置并重建实例；数据卷与数据库内容不会恢复。': '現在の構成が履歴構成に置き換えられ、インスタンスが再構築されます。データ ボリュームとデータベースの内容は復元されません。',
    // Current changes are not committed, switching config discards them.
    '当前修改尚未提交，切换配置将丢弃这些修改。': '現在の変更はコミットされていないため、構成を切り替えるとこれらの変更は破棄されます。',
    // Keep Editing
    '保留编辑': '編集内容を保持する',
    // Discard Changes and Switch
    '放弃修改并切换': '変更を破棄して切り替える',
    // Retry loading config
    '重试加载配置': '構成のロードを再試行します',
    // Rebuilding the application...
    '正在重建应用...': 'アプリケーションを再構築しています...',
    // Loading configuration...
    '正在读取配置...': '構成を読み取り中...',
    // Scheduled
    '调度中': 'スケジュール設定',
    // Unregistered
    '未注册': '未登録',
    // No script type available
    '暂无可用脚本类型': '利用可能なスクリプト タイプはまだありません',
    // Delete Cron Job
    '删除计划任务': 'スケジュールされたタスクを削除する',
    // Delete job <strong class="text-slate-900">{0}</strong>? This action cannot be undone.
    '确定要删除任务 <strong class="text-slate-900">{0}</strong> 吗？此操作不可恢复。': 'タスク<strong class="text-slate-900">{0}</strong>を削除してもよろしいですか?この操作は元に戻すことができません。',
    // Job deleted
    '任务已删除': 'タスクが削除されました',
    // Job disabled
    '任务已禁用': 'タスクが無効化されました',
    // Job enabled
    '任务已启用': 'タスクが有効になっています',
    // Job "{0}" triggered
    '任务 "{0}" 已触发': 'タスク「{0}」がトリガーされました',
    // Runs script commands on a schedule or at a fixed time
    '按设定时间或周期自动执行脚本命令': '設定した時間または周期に従ってスクリプトコマンドを自動的に実行します',
    // Search job name or schedule...
    '搜索任务名称、执行计划...': 'タスク名、実行計画を検索...',
    // Runs scripts automatically on schedule
    '定时自动执行脚本': 'スクリプトを定期的に自動的に実行する',
    // Search jobs...
    '搜索任务...': 'タスクを検索...',
    // No cron jobs yet
    '暂无计划任务': 'スケジュールされたタスクはまだありません',
    // Click "New Job" to create the first scheduled task
    '点击「新建任务」创建第一个定时任务': '「新しいタスク」をクリックして、最初のスケジュールされたタスクを作成します',
    // Click to disable
    '点击禁用': 'クリックして無効にします',
    // Click to enable
    '点击启用': 'クリックして有効にします',
    // Fill in the required fields: name, schedule, script content
    '请填写必填项：名称、执行计划、脚本内容': '必須フィールドに入力してください: 名前、実行計画、スクリプトの内容',
    // Enter the image name for the DOCKER image type
    'DOCKER 镜像类型请填写镜像名': 'DOCKER イメージの種類 イメージ名を入力してください',
    // Enter the target container name for the DOCKER container type
    'DOCKER 容器类型请填写目标容器名': 'DOCKERコンテナタイプ ターゲットコンテナ名を入力してください',
    // Job updated
    '任务已更新': 'タスクが更新されました',
    // Job created
    '任务已创建': 'タスクが作成されました',
    // Edit Cron Job
    '编辑计划任务': 'スケジュールされたタスクを編集する',
    // New Cron Job
    '新建计划任务': '新しいスケジュールされたタスクを作成する',
    // Enter the job name
    '请输入任务名称': 'タスク名を入力してください',
    // e.g. daily backup
    '例如：每日备份': '例: 毎日のバックアップ',
    // Enter a description (optional)
    '请输入描述（可选）': '説明を入力してください (オプション)',
    // Format reference
    '格式参考': 'フォーマットリファレンス',
    // Enter the schedule (cron expression)
    '请输入执行计划（cron 表达式）': '実行計画 (cron 式) を入力してください',
    // Standard cron expression, e.g. 0 2 * * * for 2 AM daily
    '标准 cron 表达式，例如：每天 2 点可填 0 2 * * *': '標準の cron 式、例: 0 2 * * * は毎日 2 時に入力できます。',
    // Script Type
    '脚本类型': 'スクリプトの種類',
    // Enter the working directory (optional)
    '请输入工作目录（可选）': '作業ディレクトリを入力してください (オプション)',
    // Leave empty to use the current directory
    '留空则使用当前目录': '現在のディレクトリを使用する場合は空白のままにします',
    // Image Name
    '镜像名': 'イメージ名',
    // Enter the image name
    '请输入镜像名': 'イメージ名を入力してください',
    // e.g. python:3.12-slim, nginx:alpine
    '例如：python:3.12-slim、nginx:alpine': '例: python:3.12-slim、nginx:alpine',
    // Extra Mounts
    '额外挂载': '追加のマウント',
    // One /host:/container[:ro] per line
    '每行 /host:/container[:ro]': '各行 /host:/container[:ro]',
    // Enter mount config (optional)
    '请输入挂载配置（可选）': 'マウント構成を入力してください (オプション)',
    // e.g. /data/files:/data:ro
    '例如：/data/files:/data:ro': '例:/data/files:/data:ro',
    // Target Container
    '目标容器': 'ターゲットコンテナ',
    // Enter or select a container name
    '请输入或选择容器名': 'コンテナ名を入力または選択してください',
    // Enter the container name
    '请输入容器名': 'コンテナ名を入力してください',
    // Script Content
    '脚本内容': 'スクリプトの内容',
    // Enter the script content
    '请输入脚本内容': 'スクリプトの内容を入力してください',
    // e.g. #!/bin/bash
    '例如：#!/bin/bash': '例: #!/bin/bash',
    // When enabled the job runs on schedule
    '启用后任务将按计划执行': '有効にすると、タスクはスケジュールどおりに実行されます',
    // Execution History
    '执行历史': '実行履歴',
    // Latest
    '最近': '直近',
    // runs
    '次': '回',
    // No execution records
    '暂无执行记录': 'まだ実行記録がありません',
    // No output
    '无输出内容': '出力内容なし',
    // Resource Limits
    '资源限制': 'リソース制限',
    // Host Port
    '主机端口': 'ホストポート',
    // Container Port
    '容器端口': 'コンテナ港',
    // No port mapping
    '无端口映射': 'ポートマッピングなし',
    // No mounts
    '无挂载': 'マウントなし',
    // Command and Environment Variables
    '命令与环境变量': 'コマンドと環境変数',
    // Container details not found
    '未找到容器详情': 'コンテナの詳細が見つかりません',
    // Image Details
    '镜像详情': 'イメージの詳細',
    // Image ID
    '镜像 ID': 'イメージID',
    // Short ID
    '短 ID': 'ショートID',
    // Layers
    '层数': 'レイヤー数',
    // Author
    '作者': '著者',
    // Platform Info
    '平台信息': 'プラットフォーム情報',
    // Tags
    '标签数': 'タグの数',
    // Digests
    'Digest 数': 'ダイジェスト番号',
    // No digest
    '无 Digest': 'ダイジェストなし',
    // Exposed Ports
    '暴露端口': '公開ポート',
    // Layer Info
    '层信息': 'レイヤー情報',
    // real layers,
    '个实际层，共': '実際のレイヤー、合計',
    // steps)
    '步）': 'ステップ)',
    // Command
    '命令': 'コマンド',
    // Empty layer
    '空层': '空のレイヤー',
    // (no command)
    '(无命令)': '(コマンドなし)',
    // No layer info
    '无层信息': 'レイヤー情報がありません',
    // Image details not found
    '未找到镜像详情': 'イメージの詳細が見つかりません',
    // Gateway
    '网关': 'ゲートウェイ',
    // Internal Network
    '内部网络': '内部ネットワーク',
    // Connected Containers
    '已连接容器': 'コンテナが接続されました',
    // MAC Address
    'MAC 地址': 'MACアドレス',
    // No container connected to this network
    '暂无容器连接到此网络': '現在このネットワークに接続されているコンテナはありません',
    // Network details not found
    '未找到网络详情': 'ネットワークの詳細が見つかりません',
    // Size on Disk
    '占用空间': 'スペースを取る',
    // References
    '引用数': '参照数',
    // Containers Using This Volume
    '使用此卷的容器': 'このボリュームを使用するコンテナ',
    // Mount Path
    '挂载路径': 'マウントパス',
    // Read/Write
    '读写': '読み書き',
    // No container uses this volume
    '暂无容器使用此数据卷': '現在このデータ ボリュームを使用しているコンテナはありません',
    // Volume details not found
    '未找到数据卷详情': 'ボリュームの詳細が見つかりません',
    // Start Container
    '启动容器': 'コンテナの起動',
    // Stop Container
    '停止容器': 'コンテナを停止する',
    // Restart Container
    '重启容器': 'コンテナを再起動します',
    // Delete Container
    '删除容器': 'コンテナの削除',
    // Pause Container
    '暂停容器': 'コンテナを一時停止する',
    // Resume Container
    '恢复容器': 'コンテナを再開',
    // {0} container <strong class="text-slate-900">{1}</strong>?
    '确定要{0}容器 <strong class="text-slate-900">{1}</strong> 吗？': '{0}操作をコンテナ <strong class="text-slate-900">{1}</strong> に対して実行してもよろしいですか？',
    // Container {0}
    '容器 {0} 成功': 'コンテナ {0} が成功しました',
    // Managed by Swarm, direct editing is not supported
    '由 Swarm 管理，不支持直接编辑': 'Swarm によって管理されており、直接編集はサポートされていません',
    // Edit Compose Project Config
    '编辑 Compose 项目配置': 'Compose プロジェクト構成の編集',
    // Manage the lifecycle and running state of Docker containers
    '管理 Docker 容器的生命周期与运行状态': 'Dockerコンテナのライフサイクルと実行ステータスを管理する',
    // Search container name, ID, image or port...
    '搜索容器名称、ID、镜像或端口...': 'コンテナ名、ID、イメージ、またはポートを検索します...',
    // Manage container lifecycle
    '管理容器生命周期': 'コンテナのライフサイクルを管理する',
    // Search container name, image or port...
    '搜索容器名称、镜像或端口...': 'コンテナ名、イメージ、またはポートを検索します...',
    // Open Terminal
    '登录终端': 'ターミナルにログインする',
    // No containers yet
    '暂无容器': 'まだコンテナがありません',
    // No matching container
    '未找到匹配容器': '一致するコンテナが見つかりませんでした',
    // Click "New Container" to create one
    '点击「新建容器」开始创建': '「新しいコンテナ」をクリックして作成を開始します',
    // The image has no tag and cannot be pulled
    '镜像无标签，无法拉取': 'イメージにはラベルがないため、取得できません。',
    // Pull image <strong class="text-slate-900">{0}</strong> again?
    '确定要重新拉取镜像 <strong class="text-slate-900">{0}</strong> 吗？': '<strong class="text-slate-900">{0}</strong> イメージを再度取得してもよろしいですか?',
    // Confirm Pull
    '确认拉取': 'プルを確認する',
    // Delete Image
    '删除镜像': 'イメージの削除',
    // Delete image <strong class="text-slate-900">{0}</strong>?
    '确定要删除镜像 <strong class="text-slate-900">{0}</strong> 吗？': 'イメージ <strong class="text-slate-900">{0}</strong> を削除してもよろしいですか?',
    // Image deleted
    '镜像删除成功': 'イメージは正常に削除されました',
    // Image prune finished, reclaimed {0}
    '镜像清理完成，回收 {0}': 'イメージのクリーンアップが完了しました。{0} の容量を解放しました',
    // Pull, import, export and delete Docker images
    '拉取、导入、导出和删除 Docker 镜像': 'Docker イメージのプル、インポート、エクスポート、削除',
    // Search image name, tag or ID...
    '搜索镜像名称、标签、ID...': 'イメージ名、タグ、IDを検索...',
    // Pull and manage images
    '拉取与管理镜像': 'イメージの取得と管理',
    // Search image name, tag or ID...
    '搜索镜像名称、标签或 ID...': 'イメージ名、タグ、またはIDを検索...',
    // Push
    '推送': '押す',
    // No images yet
    '暂无镜像': 'まだ鏡がありません',
    // No matching image
    '未找到匹配镜像': '一致するイメージが見つかりませんでした',
    // Click "Pull" to fetch from the registry
    '点击「拉取」从 Registry 获取': '「プル」をクリックしてレジストリから取得します。',
    // Confirm Prune
    '确认清理': 'クリーンアップの確認',
    // Dangling image layers unused by any container are pruned, images in use are kept.
    '将清理未被任何容器使用的悬空镜像层，不会删除正在被容器使用的镜像。': 'どのコンテナーでも使用されていない未解決のイメージ レイヤーはクリーンアップされ、コンテナーで使用されているイメージは削除されません。',
    // Also prune tagged but unused images
    '同时清理有标签但未被使用的镜像': 'タグ付けされた未使用のイメージもクリーンアップします',
    // Equivalent to
    '等同于': 'に相当',
    // , reclaims more space
    '，会删除更多空间': '、より多くのスペースが削除されます',
    // Delete Network
    '删除网络': 'ネットワークを削除する',
    // Delete network <strong class="text-slate-900">{0}</strong>?
    '确定要删除网络 <strong class="text-slate-900">{0}</strong> 吗？': 'ネットワーク <strong class="text-slate-900">{0}</strong> を削除してもよろしいですか?',
    // Network deleted
    '网络删除成功': 'ネットワークの削除が成功しました',
    // Default bridge network
    '默认桥接网络': 'デフォルトのブリッジネットワーク',
    // Host network
    '主机网络': 'ホストネットワーク',
    // Empty network
    '空网络': '空のネットワーク',
    // {0} cannot be deleted
    '{0}不可删除': '{0} は削除できません',
    // Manage Docker networks and container communication
    '管理 Docker 网络，配置容器间通信': 'Docker ネットワークを管理し、コンテナ間通信を構成する',
    // Search network name, ID, driver or subnet...
    '搜索网络名称、ID、驱动或子网...': 'ネットワーク名、ID、ドライバー、またはサブネットを検索します...',
    // Manage container networks
    '管理容器网络': 'コンテナネットワークを管理する',
    // Search network name, ID or driver...
    '搜索网络名称、ID、驱动...': 'ネットワーク名、ID、ドライバーを検索...',
    // No custom networks yet
    '暂无自定义网络': 'カスタム ネットワークはまだありません',
    // No matching network
    '未找到匹配网络': '一致するネットワークが見つかりませんでした',
    // Click "New Network" to add a custom network
    '点击「新建网络」添加自定义网络': '「新しいネットワーク」をクリックしてカスタムネットワークを追加します',
    // Delete Registry
    '删除镜像仓库': 'イメージリポジトリの削除',
    // Delete registry <strong class="text-slate-900">{0}</strong> ({1})?
    '确定要删除仓库 <strong class="text-slate-900">{0}</strong> ({1}) 吗？': 'リポジトリ <strong class="text-slate-900">{0}</strong> ({1}) を削除してもよろしいですか?',
    // Registry deleted
    '仓库删除成功': 'レジストリは正常に削除されました',
    // Manage private registry credentials and image mirrors
    '管理私有镜像仓库认证信息与镜像加速器': 'プライベートイメージレジストリの認証情報とイメージアクセラレータを管理する',
    // Add Registry
    '添加仓库': '倉庫を追加する',
    // Manage registry accounts and mirrors
    '管理仓库账号与加速器': 'レジストリアカウントとアクセラレータを管理する',
    // Mirror
    '加速': 'スピードを上げる',
    // Click "New Registry" to create a private registry
    '点击「新建仓库」创建私有镜像仓库': '「新しいレジストリ」をクリックしてプライベートイメージレジストリを作成します',
    // Delete Volume
    '删除数据卷': 'データボリュームの削除',
    // Delete volume <strong class="text-slate-900">{0}</strong>?
    '确定要删除数据卷 <strong class="text-slate-900">{0}</strong> 吗？': 'ボリューム<strong class="text-slate-900">{0}</strong>を削除してもよろしいですか?',
    // Volume deleted
    '数据卷删除成功': 'データボリュームが正常に削除されました',
    // Manage Docker volumes that persist container data
    '管理 Docker 数据卷，持久化容器数据': 'Docker データ ボリュームを管理し、コンテナ データを永続化する',
    // Manage container volumes
    '管理容器数据卷': 'コンテナーのデータ ボリュームを管理する',
    // No volumes yet
    '暂无数据卷': 'ボリュームはまだありません',
    // No matching volume
    '未找到匹配数据卷': '一致するデータ ボリュームが見つかりませんでした',
    // Click "New Volume" to create a volume
    '点击「新建卷」创建数据卷': '「新しいボリューム」をクリックしてデータボリュームを作成します',
    // Receive
    '接收': '受け取る',
    // Send
    '发送': '送信する',
    // Collecting data...
    '正在采集数据...': 'データを収集中...',
    // Cores
    '核心': 'コア',
    // cores
    '核': 'コア',
    // Frequency
    '频率': '周波数',
    // Throttled
    '节流': 'スロットル',
    // Limit
    '限制': '限界',
    // Interface
    '网卡': 'ネットワークカード',
    // Block
    '块': 'ブロック',
    // Total received
    '累计收': '受信合計',
    // Total sent
    '累计发': '送信合計',
    // Disk
    '硬盘': 'ハードドライブ',
    // Device
    '设备': '設備',
    // Total read
    '累计读': '読み取り合計',
    // Total written
    '累计写': '書き込み合計',
    // Process Info
    '进程信息': 'プロセス情報',
    // / limit
    '/ 限制': '/制限',
    // No process info
    '暂无进程信息': 'プロセス情報はまだありません',
    // The container is not running, metrics cannot be collected
    '容器未运行，无法采集监控数据': 'コンテナーが実行されていないため、監視データを収集できません。',
    // Network administration (interfaces, routing, etc.)
    '网络管理（配置接口、路由等）': 'ネットワーク管理 (インターフェイスの構成、ルーティングなど)',
    // Bind ports below 1024
    '绑定 1024 以下端口': '1024 未満のポートをバインドする',
    // Broadcast listening
    '广播监听': '放送監視',
    // Raw sockets (ping, etc.)
    '原始套接字（ping 等）': '生のソケット (ping など)',
    // Bypass file read/write/execute permission checks
    '绕过文件读写执行权限检查': 'ファイルの読み書き実行権限チェックをバイパスする',
    // Bypass file read and directory search permission checks
    '绕过文件读和目录搜索权限': 'ファイル読み取りおよびディレクトリ検索権限をバイパスします。',
    // Bypass file ownership checks
    '绕过文件属主权限检查': 'ファイル所有者の権限チェックをバイパスする',
    // Modify file set-ID bits
    '修改文件 set-ID 位': 'ファイルセットIDビットを変更する',
    // Change file ownership
    '修改文件属主': 'ファイル所有者の変更',
    // Set file capabilities
    '设置文件能力': 'ファイル機能を設定する',
    // Users and Processes
    '用户与进程': 'ユーザーとプロセス',
    // Change process UID
    '修改进程 UID': 'プロセスUIDの変更',
    // Change process GID
    '修改进程 GID': 'プロセスGIDの変更',
    // Change process capabilities
    '修改进程能力': 'プロセス機能の変更',
    // System administration (all-powerful, use with care)
    '系统管理（万能权限，谨慎使用）': 'システム管理 (ユニバーサル権限、使用には注意が必要です)',
    // Send signals to any process
    '向任意进程发送信号': '任意のプロセスにシグナルを送信する',
    // Write audit logs
    '写入审计日志': '監査ログの書き込み',
    // System Control
    '系统控制': 'システム制御',
    // Use chroot
    '使用 chroot': 'chrootを使用する',
    // Trace and debug processes
    '进程跟踪调试': 'プロセスのトレースとデバッグ',
    // Modify system resource limits
    '修改系统资源限制': 'システムリソース制限を変更する',
    // Modify the system clock
    '修改系统时钟': 'システムクロックを変更する',
    // Modify process priority
    '修改进程优先级': 'Modify process priority',
    // Create device files
    '创建设备文件': 'Create device file',
    // Modify immutable file attributes
    '修改文件不可变属性': 'ファイルの不変属性を変更する',
    // Devices and Kernel
    '设备与内核': 'Devices and Kernels',
    // Load/unload kernel modules
    '加载/卸载内核模块': 'カーネルモジュールのロード/アンロード',
    // Raw I/O port access
    '原始 I/O 端口访问': 'Raw I/O port access',
    // Reboot the system
    '重启系统': 'Restart the system',
    // Kernel log access
    '内核日志访问': 'Kernel log access',
    // Acquire file leases
    '获取文件租约': 'Get file lease',
    // Block system suspend
    '阻止系统挂起': 'システムのハングを防ぐ',
    // Click to select or enter a capability name...
    '点击选择或输入权限名称...': 'クリックして権限名を選択または入力します...',
    // No matching capability
    '未找到匹配的权限': '一致する権限が見つかりませんでした',
    // Press Enter to add a custom capability
    '按 Enter 可添加自定义权限': 'Enter キーを押してカスタム権限を追加します',
    // capabilities
    '项权限': 'アイテムの権限',
    // Always restart
    '总是重启': '常に再起動する',
    // Unless stopped manually
    '除非手动停止': '手動で止めない限り',
    // Restart on failure
    '失败时重启': '失敗時の再起動',
    // No restart
    '不重启': '再起動なし',
    // Enter the container name
    '请填写容器名称': 'コンテナ名を入力してください',
    // Container created
    '容器创建成功': 'コンテナが正常に作成されました',
    // Network Mode
    '网络模式': 'ネットワークモード',
    // Enter port mappings
    '请输入端口映射': 'ポートマッピングを入力してください',
    // Format: host:container, e.g. 8080:80, one per line
    '格式：主机端口:容器端口，如 8080:80，每行一条': '形式: ホスト ポート:コンテナ ポート (8080:80 など)、1 行に 1 つ',
    // Mount Mappings
    '挂载映射': 'マウントマッピング',
    // Enter mount mappings
    '请输入挂载映射': 'マウントマッピングを入力してください',
    // Format: source:container-path[:ro], e.g. ./data:/app/data:ro, one per line
    '格式：来源:容器路径[:ro]，如 ./data:/app/data:ro，每行一条': '形式: ソース: コンテナー パス [:ro]、./data:/app/data:ro など、1 行に 1 つ',
    // Enter environment variables
    '请输入环境变量': '環境変数を入力してください',
    // One per line, KEY=value or KEY: value
    '每行一条，支持 KEY=value 或 KEY: value 格式': '1 行に 1 つ、KEY=value または KEY: value 形式をサポートします。',
    // Enter the start command
    '请输入启动命令': '起動コマンドを入力してください',
    // Enter the container hostname
    '请输入容器主机名': 'コンテナのホスト名を入力してください',
    // Memory Limit (MB)
    '内存限制 (MB)': 'メモリ制限 (MB)',
    // Enter the memory limit (MB)
    '请输入内存限制（MB）': 'メモリ制限 (MB) を入力してください',
    // CPU Limit (cores)
    'CPU 限制 (核心)': 'CPU 制限 (コア)',
    // Enter the CPU limit (cores)
    '请输入 CPU 限制（核心数）': 'CPU制限（コア数）を入力してください',
    // Enter the working directory
    '请输入工作目录': '作業ディレクトリを入力してください',
    // Run As User
    '运行用户': 'ユーザーを実行する',
    // Enter the user to run as
    '请输入运行用户': '実行中のユーザーを入力してください',
    // Security Config
    '安全配置': 'セキュリティ設定',
    // Privileged
    '特权': '特権',
    // ⚠️ Grants the container all host privileges, use with care
    '⚠️ 赋予容器所有主机权限，谨慎使用': '⚠️コンテナにすべてのホスト権限を付与します。使用には注意してください',
    // Add Capabilities (CapAdd)
    '添加权限 (CapAdd)': '権限の追加 (CapAdd)',
    // Drop Capabilities (CapDrop)
    '移除权限 (CapDrop)': '権限の削除 (CapDrop)',
    // Select Port
    '选择端口': 'ポートの選択',
    // No container available
    '无可用容器': '利用可能なコンテナがありません',
    // No matching container
    '无匹配容器': '一致するコンテナがありません',
    // running containers
    '个运行中容器': 'コンテナの実行',
    // Image built
    '镜像构建成功': 'イメージが正常に構築されました',
    // Build Image
    '构建镜像': 'ビルドイメージ',
    // Image Tag
    '镜像标签': 'イメージタグ',
    // Leave empty to use custom:latest
    '留空则使用 custom:latest': 'Custom:latest を使用するには空白のままにします',
    // Enter the Dockerfile content
    '请输入 Dockerfile 内容': 'Dockerfile の内容を入力してください',
    // Start Build
    '开始构建': '構築を開始する',
    // Docker Hub (default)
    'Docker Hub（默认）': 'Docker ハブ (デフォルト)',
    // (no mirror configured)
    '（未配置加速器）': '(アクセラレータが設定されていません)',
    // Enter the image name to search
    '请输入要搜索的镜像名称': '検索したいイメージ名を入力してください',
    // Searching
    '搜索中': '検索中',
    // Official Image
    '官方镜像': '公式イメージ',
    // Official
    '官方': '役人',
    // Image Reference
    '镜像引用': 'イメージリファレンス',
    // Enter the image reference
    '请输入镜像引用': 'イメージ参照を入力してください',
    // Pulls latest when no tag is given; the registry address is not needed again when a private registry is selected.
    '未指定 tag 时默认拉取 latest；选择私有仓库时不需要重复填写仓库地址。': 'タグを指定しない場合は latest を取得します。プライベートレジストリを選択した場合、レジストリアドレスを再入力する必要はありません。',
    // Will pull
    '最终将拉取': '結局引っ張るだろう',
    // Start Pull
    '开始拉取': '引っ張り始める',
    // Aliyun
    '阿里云': 'アリババクラウド',
    // Tencent Cloud
    '腾讯云': 'テンセントクラウド',
    // No image available
    '无可用镜像': 'イメージがありません',
    // No matching image
    '无匹配镜像': '一致するイメージがありません',
    // images
    '个镜像': '鏡',
    // Image tag added
    '镜像标签添加成功': 'イメージタグが正常に追加されました',
    // New Image Tag
    '新建镜像标签': '新しいイメージタグを作成する',
    // Current Image
    '当前镜像': '現在のイメージ',
    // Existing Tags
    '已有标签': 'すでにタグ付けされています',
    // New Tag
    '新标签': '新しいタブ',
    // Format: repository:tag, e.g. myapp:v1.0
    '格式: 仓库路径:标签，如 myapp:v1.0': '形式: レジストリ パス:ラベル (myapp:v1.0 など)',
    // Network created
    '网络创建成功': 'ネットワークが正常に作成されました',
    // Enter the network name
    '请输入网络名称': 'ネットワーク名を入力してください',
    // bridge
    'bridge (桥接)': '橋',
    // host
    'host (主机)': 'ホスト',
    // overlay
    'overlay (覆盖)': 'オーバーレイ',
    // Subnet CIDR (optional)
    '子网 CIDR（可选）': 'サブネット CIDR (オプション)',
    // Enter the subnet CIDR
    '请输入子网 CIDR': 'サブネットCIDRを入力してください',
    // Edit Registry
    '编辑镜像仓库': 'イメージリポジトリの編集',
    // New Registry
    '新建镜像仓库': '新しいイメージ レジストリを作成する',
    // Registry updated
    '仓库更新成功': 'レジストリが正常に更新されました',
    // Registry added
    '仓库添加成功': 'レジストリが正常に追加されました',
    // Enter the registry name
    '请输入仓库名称': 'レジストリ名を入力してください',
    // Registry Address
    '仓库地址': 'レジストリアドレス',
    // Enter the registry address
    '请输入仓库地址': 'レジストリアドレスを入力してください',
    // Enter a registry description
    '请输入仓库描述': 'レジストリの説明を入力してください',
    // Enter the registry password
    '请输入仓库密码': 'レジストリのパスワードを入力してください',
    // Image pushed
    '镜像推送成功': 'イメージのプッシュが成功しました',
    // Push Image to Registry
    '推送镜像到仓库': 'イメージをレジストリにプッシュします',
    // Select an image
    '请选择镜像': 'イメージを選択してください',
    // Target Registry Address
    '目标仓库地址': 'プッシュ先のレジストリアドレス',
    // Select a registry
    '请选择仓库': 'レジストリを選択してください',
    // Namespace
    '命名空间': '名前空間',
    // Enter the namespace (optional)
    '请输入命名空间（可选）': '名前空間を入力してください (オプション)',
    // The image will be pushed as:
    '镜像将被推送为:': 'イメージは次のようにプッシュされます。',
    // Start Push
    '开始推送': '押し始める',
    // Volume created
    '数据卷创建成功': 'データボリュームが正常に作成されました',
    // New Volume
    '新建数据卷': '新しいデータボリュームを作成する',
    // Volume Name
    '卷名称': 'ボリューム名',
    // Enter the volume name
    '请输入数据卷名称': 'データボリューム名を入力してください',
    // local
    'local (本地)': '地元の',
    // Edit config: {0}
    '编辑配置：{0}': '構成の編集: {0}',
    // Edit Container Config
    '编辑容器配置': 'コンテナ構成の編集',
    // 5 minutes
    '5分钟': '5分',
    // 1 hour
    '1小时': '1時間',
    // 6 hours
    '6小时': '6時間',
    // 12 hours
    '12小时': '12時間',
    // 24 hours
    '24小时': '24時間',
    // Check the service status and refresh
    '请检查服务状态后刷新重试': 'サービスのステータスを確認し、更新して再試行してください',
    // Connect to remote servers through a web terminal
    '通过 Web 终端连接到远程服务器': 'Web端末経由でリモートサーバーに接続',
    // Web Terminal Connection
    'Web 终端连接': 'Web端末接続',
    // Failed to get the process list
    '获取进程列表失败': 'プロセスリストの取得に失敗しました',
    // Force kill gives the process no chance to clean up and may cause data loss.
    '强制终止不会给进程清理资源的机会，可能导致数据丢失。': '強制終了すると、プロセスにリソースをクリーンアップする機会が与えられず、データが失われる可能性があります。',
    // The process receives a termination signal and can clean up before exiting.
    '进程会收到终止信号，允许其自行清理后退出。': 'プロセスは終了シグナルを受信し、プロセス自体の後をクリーンアップして終了できるようになります。',
    // Force Kill Process
    '强制终止进程': 'プロセスを強制終了する',
    // Are you sure you want to force kill <strong class="text-slate-900">{0}</strong> (PID {1})? {2}
    '确定要强制终止 <strong class="text-slate-900">{0}</strong>（PID {1}）吗？{2}': '<strong class="text-slate-900">{0}</strong>（PID {1}）を強制終了してもよろしいですか？{2}',
    // Are you sure you want to terminate <strong class="text-slate-900">{0}</strong> (PID {1})? {2}
    '确定要终止 <strong class="text-slate-900">{0}</strong>（PID {1}）吗？{2}': '<strong class="text-slate-900">{0}</strong>（PID {1}）を終了してもよろしいですか？{2}',
    // Confirm Kill
    '确认终止': '終了の確認',
    // Termination signal sent
    '已发送终止信号': '終了信号が送信されました',
    // Failed to kill the process
    '终止进程失败': 'プロセスの終了に失敗しました',
    // View local processes and resource usage, kill them as needed
    '查看本机进程与资源占用，并可按需终止': 'ローカルプロセスとリソースの使用状況を表示し、必要に応じて終了します',
    // Search process name, PID, parent PID, user or command line...
    '搜索进程名、PID、父 PID、用户、命令行...': 'プロセス名、PID、親 PID、ユーザー、コマンド ラインを検索します...',
    // Flat List
    '平铺列表': 'タイルリスト',
    // Memory Usage
    '内存占用': 'メモリ使用量',
    // CPU Usage
    'CPU 占用': 'CPU使用率',
    // I/O Rate
    'I/O 速率': 'I/O速度',
    // GPU Memory
    'GPU 显存': 'GPUメモリ',
    // Process ID
    '进程 ID': 'プロセスID',
    // Process Name
    '进程名称': 'プロセス名',
    // Local Processes
    '本机进程': 'ローカルプロセス',
    // Flat
    '平铺': 'タイル',
    // Search processes...
    '搜索进程...': '検索プロセス...',
    // No processes retrieved
    '未获取到进程': 'プロセスが取得できませんでした',
    // No matching process
    '未找到匹配进程': '一致するプロセスが見つかりませんでした',
    // Please refresh and retry
    '请刷新重试': '更新してもう一度お試しください',
    // Process Tree
    '进程树': 'プロセスツリー',
    // · write
    '· 写': '· 書く',
    // Download
    '下行': '受信',
    // Upload
    '上行': '送信',
    // Network Interfaces
    '网络接口': 'ネットワークインターフェース',
    // Total received:
    '累计收:': '受信合計：',
    // Total sent:
    '累计发:': '送信合計：',
    // Disk I/O
    '硬盘 I/O': 'ハードディスクI/O',
    // Total
    '总计': '合計',
    // , used
    '，已用': '、使用中',
    // Total read:
    '累计读:': '読み取り合計：',
    // Total written:
    '累计写:': '書き込み合計：',
    // Memory
    '显存': 'ビデオメモリ',
    // GPU
    '显卡': 'グラフィックスカード',
    // Allocated
    '已分配': '割り当てられた',
    // Heap Allocated
    '堆已分配': '割り当てられたヒープ',
    // Heap In Use
    '堆使用中': '使用中のヒープ',
    // Heap Idle
    '堆空闲': 'ヒープフリー',
    // Heap Released
    '堆已释放': 'ヒープが解放されました',
    // Heap Sys
    '堆已申请': 'ヒープが適用されました',
    // GC Cycles
    'GC 次数': 'GC 時間',
    // Stack In Use
    '栈已使用': 'スタックが使用されています',
    // Stack Sys
    '栈已申请': 'スタックが申請されました',
    // Total Allocated
    '累计分配': '累積配分',
    // Sys
    '系统申请': 'システムアプリケーション',
    // Go Runtime
    'Go 运行态': '実行状態に移行',
    // System Memory
    '系统内存': 'システムメモリ',
    // Heap Memory
    '堆内存': 'ヒープメモリ',
    // Stack Memory
    '栈内存': 'スタックメモリ',
    // Counters
    '计数器': 'カウンター',
    // Last GC
    '最后 GC 时间': '前回の GC 時間',
    // No module to view
    '暂无可查看的模块': '現在表示するモジュールはありません',
    // This account has no module permissions
    '当前账号未分配任何模块权限': '現在のアカウントにはモジュール権限が割り当てられていません',
    // Enabled Routes
    '启用路由': 'ルーティングを有効にする',
    // Disabled Routes
    '停用路由': 'ルーティングを無効にする',
    // Total Upstreams
    '上游总数': 'アップストリームの総数',
    // Inline Upstreams
    '直连上游': 'アップストリームに直結',
    // SSL Enabled
    '启用 SSL': 'SSLを有効にする',
    // Total Consumers
    '消费者总数': '消費者の総数',
    // Cannot get APISIX info
    '无法获取 APISIX 信息': 'APISIX情報を取得できません',
    // Make sure the APISIX service is running
    '请确认 APISIX 服务是否正常运行': 'APISIXサービスが正常に動作しているか確認してください。',
    // Servers
    'Server 数': 'サーバー番号',
    // TLS Enabled
    'TLS 启用': 'TLS が有効になっている',
    // Cannot get Caddy info
    '无法获取 Caddy 信息': 'キャディ情報を取得できません',
    // Make sure the Caddy admin API is running
    '请确认 Caddy admin api 是否正常运行': 'Caddy 管理 API が正常に動作しているか確認してください。',
    // CPU Cores
    'CPU 核心': 'CPUコア',
    // {0} physical / {1} logical
    '{0} 物理 / {1} 逻辑': '{0} 物理 / {1} 論理',
    // System time ({0})
    '系统时间（{0}）': 'システム時刻 ({0})',
    // Uptime
    '运行时间': '走行時間',
    // Version
    '程序版本': 'プログラムバージョン',
    // Enter the name of the current Docker container
    '请填写当前 Docker 容器名称': '現在の Docker コンテナ名を入力してください',
    // Upgrade succeeded, refreshing the page...
    '升级成功，即将刷新页面...': 'アップグレードは成功しました。ページはすぐに更新されます...',
    // Upgrade timed out, please check the service status manually
    '升级超时，请手动检查服务状态': 'アップグレードがタイムアウトしました。サービスのステータスを手動で確認してください',
    // New Version Available
    '发现新版本': '新しいバージョンが見つかりました',
    // View Changelog
    '查看更新日志': '変更ログを表示する',
    // Changelog
    '更新日志': '変更ログ',
    // Upgrade the Current Container
    '升级当前容器': '現在のコンテナをアップグレードする',
    // Upgrade Container
    '升级容器': 'コンテナのアップグレード',
    // Download the latest version and restart
    '下载最新版本并重启': '最新バージョンをダウンロードして再起動します',
    // Upgrade Binary
    '升级二进制': 'バイナリのアップグレード',
    // Upgrade the Current Docker Container
    '升级当前 Docker 容器': '現在の Docker コンテナをアップグレードする',
    // Current Container Name
    '当前容器名称': '現在のコンテナ名',
    // Destroy the container automatically after the upgrade
    '升级完成后自动销毁容器': 'アップグレード完了後にコンテナを自動的に破棄する',
    // After closing, the upgrade progress is visible in the container logs
    '关闭后可通过容器日志查看升级过程': '閉じた後、コンテナー ログを通じてアップグレード プロセスを確認できます。',
    // Goes through
    '将通过': '通ります',
    // A temporary container pulls the latest image and restarts, the service is briefly interrupted.
    '临时容器拉取最新镜像并重启，升级期间服务会短暂中断。': '一時コンテナは最新のイメージを取得して再起動し、アップグレード中にサービスは一時的に中断されます。',
    // Deploy and Upgrade
    '部署并升级': '導入とアップグレード',
    // Running Containers
    '运行中容器': '実行中のコンテナ',
    // Stopped Containers
    '已停止容器': 'コンテナが停止しました',
    // Paused Containers
    '已暂停容器': 'コンテナが一時停止されました',
    // Total Images
    '镜像总数': '総イメージ数',
    // Total Networks
    '网络总数': 'ネットワークの総数',
    // Total Volumes
    '数据卷总数': 'データボリュームの総数',
    // Cannot get Docker info
    '无法获取 Docker 信息': 'Docker情報を取得できません',
    // Make sure the Docker service is running
    '请确认 Docker 服务是否正常运行': 'Dockerサービスが正常に動作しているか確認してください。',
    // Cluster ID
    '集群 ID': 'クラスターID',
    // Total Nodes
    '总节点数': 'ノードの総数',
    // Total Services
    '服务总数': 'サービスの総数',
    // Total Tasks
    '任务总数': 'タスクの総数',
    // The Swarm cluster is not initialized
    'Swarm 集群未初始化': 'Swarm cluster is not initialized',
    // Run first
    '请先执行': 'まず実行してください',
    // Delete SSH Credential
    '删除 SSH 凭据': 'SSH認証情報を削除する',
    // Delete credential <strong class="text-slate-900">{0}</strong> ({1})?
    '确定要删除凭据 <strong class="text-slate-900">{0}</strong> ({1}) 吗？': '認証情報 <strong class="text-slate-900">{0}</strong> ({1}) を削除してもよろしいですか?',
    // Credential deleted
    '凭据删除成功': 'Credentials deleted successfully',
    // Manage SSH credentials reusable by multiple hosts
    '管理 SSH 认证凭据，可被多台主机复用': 'SSH 認証資格情報を管理し、複数のホスト間で再利用する',
    // Reusable by multiple hosts
    '可被多台主机复用': '複数のホストで再利用可能',
    // No credentials yet
    '暂无认证凭据': '認証資格情報がまだありません',
    // No matching credential
    '未找到匹配凭据': '一致する資格情報が見つかりません',
    // Click "Add Credential" in the top right to create a reusable credential
    '点击右上角「添加凭据」创建可复用的认证凭据': '右上隅の「資格情報の追加」をクリックして、再利用可能な認証資格情報を作成します。',
    // Delete SSH Host
    '删除 SSH 主机': 'SSHホストの削除',
    // Delete host <strong class="text-slate-900">{0}</strong> ({1})?
    '确定要删除主机 <strong class="text-slate-900">{0}</strong> ({1}) 吗？': 'ホスト <strong class="text-slate-900">{0}</strong> ({1}) を削除してもよろしいですか?',
    // Host deleted
    '主机删除成功': 'ホストは正常に削除されました',
    // Manage hosts over SSH and connect to remote servers in the browser
    '通过 SSH 协议管理主机，在浏览器中连接远程服务器': 'SSH プロトコルを介してホストを管理し、ブラウザでリモート サーバーに接続します。',
    // Manage hosts over SSH
    '通过 SSH 协议管理主机': 'SSHプロトコル経由でホストを管理する',
    // No hosts yet
    '暂无主机连接': 'まだホスト接続がありません',
    // No matching host
    '未找到匹配主机': '一致するホストが見つかりません',
    // Click "Add Host" in the top right to start
    '点击右上角「添加主机」开始配置': '右上隅の「ホストの追加」をクリックして構成を開始します',
    // Open SSH/SFTP
    '打开 SSH/SFTP': 'SSH/SFTPを開く',
    // SSH Terminal
    'SSH 终端': 'SSH端末',
    // Loading host info...
    '正在加载主机信息...': 'ホスト情報を読み込み中...',
    // Edit SSH Credential
    '编辑 SSH 凭据': 'SSH認証情報の編集',
    // Add SSH Credential
    '添加 SSH 凭据': 'SSH認証情報の追加',
    // Credential updated
    '凭据更新成功': '資格情報が正常に更新されました',
    // Credential added
    '凭据添加成功': '資格情報が正常に追加されました',
    // Enter the credential name
    '请输入凭据名称': '資格情報名を入力してください',
    // Enter a credential description
    '请输入凭据描述': '資格情報の説明を入力してください',
    // SSH Username
    'SSH 登录用户名': 'SSHログインユーザー名',
    // (either password or private key)
    '(与私钥二选一)': '(秘密鍵があるものを選択してください)',
    // (takes precedence over the password)
    '(优先于密码)': '(パスワードを優先します)',
    // PEM private key, key authentication takes precedence when set
    'PEM 格式私钥，设置后优先使用私钥认证': 'PEM形式の秘密鍵。設定後は、最初に秘密鍵認証が使用されます。',
    // Edit SSH Host
    '编辑 SSH 主机': 'SSHホストの編集',
    // Add SSH Host
    '添加 SSH 主机': 'SSHホストの追加',
    // Host updated
    '主机更新成功': 'ホストが正常に更新されました',
    // Host added
    '主机添加成功': 'ホストが正常に追加されました',
    // Enter the host name
    '请输入主机名称': 'ホスト名を入力してください',
    // Enter the host address
    '请输入主机地址': 'ホストアドレスを入力してください',
    // Format: hostname or IP [:port], e.g. 192.168.1.100:2222, default port 22
    '格式：主机名或 IP [:端口]，如 192.168.1.100:2222，默认端口 22': '形式: ホスト名または IP [:ポート]、192.168.1.100:2222 など、デフォルトのポート 22',
    // Enter a host description
    '请输入主机描述': 'ホストの説明を入力してください',
    // Select Credential
    '选择凭据': '認証情報の選択',
    // Use the username stored in the credential
    '使用凭据中保存的用户名': '認証情報に保存されているユーザー名を使用する',
    // (optional, takes precedence over the password)
    '(可选，优先于密码)': '(オプション、パスワードより優先されます)',
    // Service details not found
    '未找到服务详情': 'サービスの詳細が見つかりません',
    // Service ID
    '服务 ID': 'サービスID',
    // (anonymous)
    '(匿名)': '（匿名）',
    // Scheduling Constraints
    '调度约束': 'スケジュールの制約',
    // Node Distribution
    '节点分布': 'ノードの分散',
    // Idle
    '空闲': '無料',
    // Task ID
    '任务 ID': 'タスクID',
    // No tasks yet
    '暂无任务': 'まだタスクはありません',
    // No running tasks
    '当前没有运行中的任务': '現在実行中のタスクはありません',
    // Node ID
    '节点 ID': 'ノードID',
    // Hardware Resources
    '硬件资源': 'ハードウェアリソース',
    // CPU Cores
    'CPU 核数': 'CPUコア数',
    // Time Info
    '时间信息': '時間情報',
    // Node details not found
    '未找到节点详情': 'ノードの詳細が見つかりません',
    // {0} Node
    '{0}节点': '{0}ノード',
    // {0} node <strong class="text-slate-900">{1}</strong>?
    '确定要{0}节点 <strong class="text-slate-900">{1}</strong> 吗？': '{0} ノード <strong class="text-slate-900">{1}</strong> が必要ですか?',
    // Node {0}
    '节点{0}成功': 'ノード {0} が成功しました',
    // No nodes yet
    '暂无节点': 'まだノードがありません',
    // No node available in this cluster
    '当前集群没有可用节点': '現在のクラスターには使用可能なノードがありません',
    // Node Role
    '节点角色': 'ノードの役割',
    // Manager Address
    'Manager 地址': 'マネージャーの住所',
    // Enter the manager address
    '请输入 Manager 地址': 'マネージャーのアドレスを入力してください',
    // Leave empty to use a placeholder, fill it in to get a ready-to-use command
    '留空则使用占位符，填写后命令可直接使用': '空白のままにすると、プレースホルダーが使用されます。入力後、コマンドを直接使用できます。',
    // Join Command
    '加入命令': '結合コマンド',
    // Copied
    '已复制': 'コピーされました',
    // Service scaled
    '服务扩缩容成功': 'サービスの拡張と縮小が成功',
    // Delete service <strong class="text-slate-900">{0}</strong>?
    '确定要删除服务 <strong class="text-slate-900">{0}</strong> 吗？': 'サービス <strong class="text-slate-900">{0}</strong> を削除してもよろしいですか?',
    // Pulls and redeploys service <strong class="text-slate-900">{0}</strong>, running replicas are updated in a rolling fashion.
    '重新拉取并部署服务 <strong class="text-slate-900">{0}</strong>，正在运行的副本会滚动更新。': 'サービス<strong class="text-slate-900">{0}</strong>を再プルしてデプロイすると、実行中のレプリカがロールオーバーされます。',
    // Confirm Redeploy
    '确认重部署': '再展開の確認',
    // Force redeploy triggered
    '已触发强制重部署': '強制再展開がトリガーされました',
    // Click "New Service" to add a Swarm service
    '点击「新建服务」添加 Swarm 服务': '「新しいサービス」をクリックしてSwarmサービスを追加します',
    // Enter the service name
    '请输入服务名': 'サービス名を入力してください',
    // Node
    '运行节点': '実行中のノード',
    // Enter port mappings (optional)
    '请输入端口映射（可选）': 'ポートマッピングを入力してください (オプション)',
    // One per line, format: host:container/protocol, e.g. 8080:80/tcp
    '每行一条，格式：宿主端口:容器端口/协议，例如：8080:80/tcp': '1 行に 1 つ、形式: ホスト ポート:コンテナ ポート/プロトコル、例: 8080:80/tcp',
    // Directory Mounts
    '目录挂载': 'ディレクトリマウント',
    // Enter directory mounts (optional)
    '请输入目录挂载（可选）': 'マウントするディレクトリを入力してください (オプション)',
    // One per line, format: host-path:container-path, e.g. /data:/app/data
    '每行一条，格式：宿主路径:容器路径，例如：/data:/app/data': '1 行に 1 つ、形式: ホスト パス:コンテナ パス、例: /data:/app/data',
    // Enter environment variables (optional)
    '请输入环境变量（可选）': '環境変数を入力してください (オプション)',
    // One per line, format: KEY=value, e.g. APP_ENV=production
    '每行一条，格式：KEY=value，例如：APP_ENV=production': '1 行に 1 つ、形式: KEY=value、例: APP_ENV=production',
    // Replicas
    '副本数': 'レプリカ数',
    // Enter startup parameters (optional)
    '请输入启动参数（可选）': '起動パラメータを入力してください (オプション)',
    // Scale Service
    '服务扩缩容': 'サービスの拡大と縮小',
    // Target Replicas
    '目标副本数': '目標レプリカ数',
    // Currently running replicas:
    '当前运行中副本：': '現在稼働中のレプリカ数：',
    // Confirm Scaling
    '确认扩缩容': '伸縮を確認する',
    // The old service is deleted and recreated after updating, it is briefly unavailable
    '更新配置后将会删除旧服务并重新创建，期间服务短暂不可用': '構成の更新後、古いサービスは削除されて再作成され、その間サービスは一時的に利用できなくなります。',
    // Skips compose.yml, re-derive Compose from the running service state
    '跳过 compose.yml，按当前服务运行态重新反推 Compose': 'compose.yml をスキップし、現在のサービスの実行状態に応じて Compose を再推論します。',
    // Swarm service config updated, service rebuilt
    'Swarm 服务配置更新成功，已重建服务': 'Swarm サービス構成が正常に更新され、サービスが再構築されました。',
    // Config reloaded
    '配置已重载': '設定がリロードされました',
    // Failed to reload the config
    '配置重载失败': '構成のリロードに失敗しました',
    // Config saved, listen address changes require a restart
    '配置已保存，监听地址变更需重启生效': '設定が保存されました。リスニング アドレスへの変更を有効にするには、再起動する必要があります。',
    // Failed to save the config
    '保存配置失败': '設定の保存に失敗しました',
    // Manage server, auth, gateway and container options by group
    '按分组管理服务器、认证、网关与容器参数': 'サーバー、認証、ゲートウェイ、コンテナのパラメータをグループごとに管理',
    // Server, Auth, Gateway and Container
    '服务器、认证、网关与容器参数': 'サーバー、認証、ゲートウェイ、コンテナーのパラメーター',
    // No permission to view the system config
    '无权限查看系统配置': 'システム構成を表示する権限がありません',
    // Check the network or permissions and retry
    '请检查网络或权限后重试': 'ネットワークまたは権限を確認して、再試行してください',
    // Reload
    '重新加载': 'リロード',
    // This account has view-only access, options cannot be modified
    '当前账号仅有查看权限，配置项不可修改': '現在のアカウントには表示権限のみがあり、構成項目は変更できません。',
    // View and search operation records of all users
    '查看和检索所有用户的操作记录': '全ユーザーの操作記録の閲覧・取得',
    // AI assistant hidden
    '已屏蔽 AI 助手': 'AIアシスタントがブロックされました',
    // Hide AI Assistant
    '不看 AI 助手': 'AIアシスタントを見ないでください',
    // View User Operation Records
    '查看用户操作记录': 'ユーザーの操作記録を閲覧する',
    // Show /api/copilot/agui requests
    '显示 /api/copilot/agui 请求': '/api/copilot/agui リクエストを表示',
    // Hide /api/copilot/agui requests
    '屏蔽 /api/copilot/agui 请求': '/api/copilot/agui リクエストをブロックする',
    // No audit logs yet
    '暂无审计日志': '監査ログはまだありません',
    // No matching log
    '未找到匹配日志': '一致するログが見つかりませんでした',
    // User operation records are shown here
    '用户操作记录将在此展示': 'ユーザーの操作記録がここに表示されます',
    // Method
    '方法': 'メソッド',
    // Duration
    '耗时': '時間がかかる',
    // Time
    '时间': '時間',
    // Enter the listen address
    '请输入监听地址': 'リスニングアドレスを入力してください',
    // HTTP listen address, e.g. :8080 or 127.0.0.1:8080 (restart required)
    'HTTP 服务监听地址，如 :8080 或 127.0.0.1:8080（重启生效）': 'HTTP サービスのリスニング アドレス (:8080 または 127.0.0.1:8080 など) (再起動すると有効になります)',
    // Base Directory
    '基础目录': 'ベースディレクトリ',
    // Enter the base directory
    '请输入基础目录': 'ベースディレクトリを入力してください',
    // Base directory for member homes and container data, defaults to the current directory (.)
    '成员家目录及容器数据的基础目录，默认当前目录（.）': 'メンバーのホーム ディレクトリとコンテナ データのベース ディレクトリ。デフォルトは現在のディレクトリ (.)',
    // Upload Size Limit (bytes)
    '文件上传大小限制（字节）': 'ファイルのアップロード サイズ制限 (バイト)',
    // Enter the upload size limit
    '请输入文件上传大小限制': 'ファイルのアップロード サイズ制限を入力してください',
    // Maximum size of a single upload, defaults to 104857600 (100 MB)
    '单次上传的最大文件大小，默认 104857600（100 MB）': '1 回のアップロードの最大ファイル サイズ、デフォルトは 104857600 (100 MB)',
    // Allowed CORS Origins
    '允许的跨域 Origin': '許可されたクロスドメインオリジン',
    // Enter one per line
    '请输入，每行一个': '1 行に 1 つずつ入力してください',
    // e.g. https://example.com, https://*.example.com; the * wildcard is supported; empty means no restriction
    '示例：https://example.com、https://*.example.com；支持通配符 *；留空则不限制': '例：https://example.com、https://*.example.com。ワイルドカード * に対応しています。空欄の場合は制限しません。',
    // JWT Signing Key
    'JWT 认证密钥': 'JWT認証キー',
    // Used to sign login tokens, all users must sign in again after changing it
    '用于签名登录令牌，修改后所有用户需要重新登录': 'ログイン トークンに署名するために使用されます。変更後は、すべてのユーザーが再度ログインする必要があります。',
    // JWT Validity (seconds)
    'JWT 有效期（秒）': 'JWTの有効期間(秒)',
    // Enter the JWT validity
    '请输入 JWT 有效期': 'JWTの有効期間を入力してください',
    // Validity of the login token, defaults to 86400 (24 hours)
    '登录令牌的有效期，默认 86400（24 小时）': 'ログイントークンの有効期間、デフォルトは86400（24時間）',
    // Serves the /openapi/ docs page (contains all API structures, keep it off in production)
    '开启后对外提供 /openapi/ 接口文档页（含全部接口结构，建议生产环境关闭）': '開いた後、/openapi/ インターフェイス ドキュメント ページが外部に提供されます (すべてのインターフェイス構造を含むため、運用環境では閉じることをお勧めします)。',
    // Debug Mode
    'Debug 模式': 'デバッグモード',
    // Outputs verbose debug logs when enabled
    '开启后输出详细调试日志': '有効にした後に詳細なデバッグログを出力する',
    // LLM Proxy and Model Rewrite
    'LLM 代理与模型改写': 'LLM エージェントとモデルの書き換え',
    // Model Name
    '模型名称': '機種名',
    // Enter the model name
    '请输入模型名称': 'モデル名を入力してください',
    // Rewrites the model field of the request body when proxying, empty means no rewrite
    '代理转发时强制改写请求体中的 model 字段，留空则不改写': 'リクエスト本文のモデルフィールドは、プロキシによって転送されるときに強制的に書き換えられます。空白のままにすると書き換えられません。',
    // Base URL
    '基础地址': 'ベースアドレス',
    // Enter the base URL
    '请输入基础地址': 'ベースアドレスを入力してください',
    // e.g. https://api.openai.com/v1; OpenAI compatible LLM API base URL, empty disables the proxy
    '示例：https://api.openai.com/v1；OpenAI 兼容的 LLM API 基础地址，留空则禁用代理': '例：https://api.openai.com/v1。OpenAI 互換の LLM API ベース URL です。空欄の場合はプロキシを無効にします。',
    // API Key
    'API 密钥': 'APIキー',
    // Injected as a Bearer Authorization header when proxying
    '代理转发时以 Bearer 形式注入 Authorization 请求头': 'プロキシ経由で転送するときに、Authorization リクエスト ヘッダーを Bearer の形式で挿入します。',
    // Marketplace iframe Site Address
    '市场 iframe 站点地址': 'マーケット iframe サイトのアドレス',
    // Site URL
    '站点 URL': 'サイトURL',
    // Enter the marketplace URL
    '请输入应用市场 URL': 'アプリマーケットのURLを入力してください',
    // The marketplace page is embedded as an iframe and receives install events via postMessage
    '应用市场页面以 iframe 方式嵌入，并通过 postMessage 协议接收安装事件': 'アプリ マーケット ページは iframe に埋め込まれており、postMessage プロトコルを通じてインストール イベントを受け取ります。',
    // Navigation Links
    '导航链接': 'ナビゲーションリンク',
    // External links in the top toolbar
    '顶部工具栏外部链接': '上部のツールバーの外部リンク',
    // No links yet, click the button below to add one
    '暂无链接，点击下方按钮添加': 'まだリンクがありません。下のボタンをクリックして追加してください',
    // Link
    '链接': 'リンク',
    // Delete Link
    '删除链接': 'リンクを削除する',
    // e.g. Toolbox
    '如：工具箱': '例: ツールボックス',
    // Link URL
    '链接地址': 'リンクアドレス',
    // Icon
    '图标': 'アイコン',
    // Add Link
    '添加链接': 'リンクを追加',
    // Go template variables are supported; use {{.Title | json}} in JSON fields to handle quotes and newlines safely.
    '支持 Go 模板变量；在 JSON 字段中使用 {{.Title | json}} 可安全处理引号和换行。': 'Go テンプレート変数のサポート。 {{.タイトル | を使用してJSON フィールドの json}} は引用符と改行を安全に処理します。',
    // <event type>
    '<事件类型>': '<イベントの種類>',
    // <alert level>
    '<告警级别>': '<警報レベル>',
    // <alert title>
    '<告警标题>': '<アラームタイトル>',
    // <alert content>
    '<告警内容>': '＜警報内容＞',
    // DingTalk Bot
    '钉钉机器人': 'ディントークロボット',
    // Feishu Bot
    '飞书机器人': 'フェイシュロボット',
    // WeCom Bot
    '企业微信机器人': 'エンタープライズ WeChat ロボット',
    // Generic Webhook
    '通用 Webhook': 'ユニバーサル Webhook',
    // Monitoring Logs
    '监控日志': '監視ログ',
    // System and container monitoring collection
    '系统与容器监控采集': 'システムとコンテナの監視と収集',
    // Collection Interval
    '监控采集间隔': '監視収集間隔',
    // 5 seconds
    '5 秒': '5秒',
    // 15 seconds
    '15 秒': '15秒',
    // 30 seconds
    '30 秒': '30秒',
    // 60 seconds
    '60 秒': '60秒',
    // Collection frequency of system and container metrics; when disabled, history collection and resource threshold alerts stop; app fault alerts below still run independently
    '系统与容器监控数据的采集频率，禁用后停止历史采集与资源阈值告警；下方应用故障告警仍可独立运行': 'システムおよびコンテナの監視データの収集頻度。これを無効にすると、履歴収集とリソースしきい値アラームが停止します。以下のアプリケーション障害アラームは引き続き独立して実行できます。',
    // Notification Channels
    '通知通道': '通知チャネル',
    // Configure webhook URL and message format, resource and app fault alerts are pushed to all channels
    '配置 Webhook 地址与消息格式，资源和应用故障告警均推送到全部通道': 'Webhook アドレスとメッセージ形式を構成し、リソースとアプリケーションの障害アラームをすべてのチャネルにプッシュします',
    // No channel yet, click the button below to configure
    '暂无通知通道，点击下方按钮开始配置': '現在、通知チャネルはありません。下のボタンをクリックして設定を開始します。',
    // Channel
    '通道': 'チャンネル',
    // Delete Channel
    '删除通道': 'チャンネルの削除',
    // Channel Name
    '通道名称': 'チャンネル名',
    // e.g. DingTalk alerts
    '如：钉钉告警': '例: DingTalk アラート',
    // Webhook URL
    'Webhook 地址': 'Webhook アドレス',
    // Paste the HTTP(S) webhook URL provided by the channel
    '粘贴渠道提供的 HTTP(S) Webhook 地址': 'チャネルから提供された HTTP(S) Webhook アドレスを貼り付けます',
    // Channel Message Template
    '消息渠道模板': 'メッセージチャネルテンプレート',
    // Custom Request Body Template
    '自定义请求体模板': 'カスタムリクエスト本文テンプレート',
    // Use standard JSON (no template)
    '使用标准 JSON（无需模板）': '標準の JSON を使用します (テンプレートは必要ありません)',
    // Selecting it overwrites the request body below; templates with placeholders need the parameters described.
    '选择后会覆盖下方请求体；带占位符的模板需按提示补充对应参数。': '選択後、以下のリクエスト本文が上書きされます。プレースホルダーを含むテンプレートの場合は、プロンプトに従って対応するパラメーターを追加する必要があります。',
    // Standard Request Body Preview
    '标准请求体结构预览': '標準リクエストボディ構造のプレビュー',
    // Generated by the backend from the actual alert when sending, no template content is stored in the config.
    '发送时由后端根据实际告警自动生成，配置中无需保存模板内容。': '送信時には、実際のアラームに基づいてバックエンドによって自動的に生成されるため、テンプレートの内容を設定に保存する必要はありません。',
    // Request Body Template (JSON)
    '请求体模板（JSON）': 'リクエスト本文テンプレート（JSON）',
    // Leave empty to send the standard isrvd event JSON
    '留空则发送 isrvd 标准事件 JSON': 'isrvd 標準イベント JSON を送信するには空白のままにします',
    // Add Channel
    '添加通道': 'チャンネルを追加',
    // App Fault Alerts
    '应用故障告警': 'アプリケーション障害アラーム',
    // Independently checks containers and gateway certificates, notifies on cron job failures
    '独立检测容器与网关证书，计划任务失败时立即通知': 'コンテナとゲートウェイの証明書を個別に検出し、スケジュールされたタスクが失敗した場合はすぐに通知します',
    // Container Faults
    '容器故障': 'コンテナの障害',
    // Abnormal exit, out of memory, failed health checks, continuous and frequent restarts; notifies on recovery
    '异常退出、内存不足、健康检查失败、持续重启及频繁重启；恢复后通知': '異常終了、メモリ不足、ヘルスチェックの失敗、連続的な再起動と頻繁な再起動。回復後の通知',
    // Frequent Restart Count
    '频繁重启次数': '頻繁な再起動時間',
    // Restart Statistics Window (seconds)
    '重启统计窗口（秒）': '統計ウィンドウの再起動 (秒)',
    // Checks every 30 seconds, the same fault is not sent twice; normal exits and common stop signals are ignored. Add the isrvd.notify.ignore=true label to exclude a container.
    '每 30 秒检测一次，同一故障不重复发送；正常退出及常见停止信号不告警。为容器添加 isrvd.notify.ignore=true 标签可排除检测。': 'これは 30 秒ごとに検出され、同じ障害が繰り返し送信されることはありません。通常の終了信号と共通の停止信号は警報を発しません。 isrvd.notify.ignore=true タグをコンテナに追加して、検出を除外します。',
    // Cron Job Failures
    '计划任务失败': 'スケジュールされたタスクが失敗しました',
    // One notification per failed execution, including timeouts; see the execution history for details, notifications contain no script or output
    '每次失败执行发送一次通知，包括超时；详细原因在执行历史中查看，通知不含脚本或输出': 'タイムアウトを含む実行が失敗するたびに通知が送信されます。詳細な理由は実行履歴で確認できます。通知にはスクリプトや出力は含まれません。',
    // Certificate Expiry
    '证书到期': '証明書の有効期限が切れます',
    // Checks readable APISIX and Caddy certificates, reminds before expiry and notifies after renewal
    '检查 APISIX 与 Caddy 的可读取证书，到期前提醒，续期恢复后通知': 'APISIX と Caddy の読み取り可能な証明書を確認し、有効期限が切れる前に通知し、更新および回復後に通知します。',
    // Days to Remind in Advance
    '提前提醒天数': '事前に通知する日数',
    // Checks every hour, each certificate is reminded at most once a day; the level is raised when expired. Caddy file and cached certificates must be readable by the panel process.
    '每小时检查一次，同一证书每天最多提醒一次；检测到过期时升级告警级别。Caddy 文件与缓存证书需要面板进程能够读取。': '1 時間ごとに確認すると、同じ証明書が最大 1 日に 1 回通知されます。期限切れが検出されると、アラームレベルがアップグレードされます。 Caddy ファイルとキャッシュされた証明書は、パネル プロセスで読み取れる必要があります。',
    // Resource Alert Rules
    '资源告警规则': 'リソースアラートルール',
    // Resource thresholds and consecutive hits that trigger a push
    '资源阈值与持续次数，命中后触发推送': 'リソースのしきい値と期間、ヒット後のトリガーのプッシュ',
    // No rules yet, click the button below to add one
    '暂无规则，点击下方按钮添加': 'まだルールがありません。下のボタンをクリックして追加してください',
    // Rule
    '规则': 'ルール',
    // Delete Rule
    '删除规则': '削除ルール',
    // Metric
    '指标': 'インジケーター',
    // Memory Usage
    '内存使用率': 'メモリ使用量',
    // Disk Usage
    '磁盘使用率': 'ディスク使用量',
    // Threshold (%)
    '阈值(%)': 'しきい値(%)',
    // Consecutive Hits
    '持续次数': '継続時間の数',
    // Add Rule
    '添加规则': 'ルールの追加',
    // Consecutive hits is the number of collection cycles above the threshold before alerting, to suppress spikes.
    '持续次数指连续多少个采集周期超阈值才告警，用于抑制瞬时抖动。': '持続時間の数は、アラームが発行される前に、連続した収集サイクルがしきい値を超えた回数を指し、瞬間的なジッターを抑制するために使用されます。',
    // Password Login
    '密码登录': 'パスワードログイン',
    // Password Login Switch
    '密码登录开关': 'パスワードログインスイッチ',
    // Disable Password Login
    '禁用密码登录': 'パスワードログインを無効にする',
    // Only Passkey, OIDC or proxy header login is allowed when disabled; make sure at least one alternative is configured
    '禁用后仅允许 Passkey、OIDC 或代理 Header 登录；请确保已配置至少一种可用的替代方式': '無効にすると、パスキー、OIDC、またはプロキシ ヘッダーのログインのみが許可されます。利用可能な代替手段が少なくとも 1 つ設定されていることを確認してください',
    // Minimum Password Length
    '密码最小长度': 'パスワードの最小長',
    // Enter the minimum password length
    '请输入密码最小长度': 'パスワードの最小長を入力してください',
    // Minimum characters when creating members and changing passwords, defaults to 6
    '创建成员和修改密码时的最小字符数，默认 6': 'メンバーの作成およびパスワードの変更時の最小文字数、デフォルトは 6',
    // WebAuthn/FIDO2 Login
    'WebAuthn/FIDO2 登录': 'WebAuthn/FIDO2 ログイン',
    // Enable Passkey Login
    '启用 Passkey 登录': 'パスキーログインを有効にする',
    // Passwordless sign-in with WebAuthn/FIDO2
    '使用 WebAuthn/FIDO2 进行无密码登录': 'WebAuthn/FIDO2を使用したパスワードレスログイン',
    // Relying Party Name
    'Relying Party 名称': '依拠当事者名',
    // Enter the RP name
    '请输入 RP 名称': 'RP名を入力してください',
    // Name shown on the Passkey registration/login screen, e.g. iSrvd
    '显示在 Passkey 注册/登录界面上的名称，如 iSrvd': 'パスキー登録/ログイン インターフェイスに表示される名前 (iSrvd など)',
    // Domain only, e.g. example.com
    '纯域名，如 example.com': '純粋なドメイン名 (example.com など)',
    // Must be a domain without the https:// prefix, e.g. example.com; an address with a scheme is reduced to its domain
    '必须是纯域名，不含 https:// 前缀，如 example.com；填写带 scheme 的地址将自动提取域名部分': 'これは、https:// プレフィックスのない純粋なドメイン名である必要があります (example.com など)。スキームを使用してアドレスを入力すると、ドメイン名の部分が自動的に抽出されます',
    // Allowed Origins
    '允许的 Origin': '許可された原点',
    // Enter allowed origins, one per line
    '请输入允许的 Origin，每行一个': '許可する Origin を 1 行に 1 つずつ入力してください',
    // e.g. https://example.com, https://*.example.com; must match the access address
    '示例：https://example.com、https://*.example.com；必须与访问地址一致': '例：https://example.com、https://*.example.com。アクセス先のアドレスと一致する必要があります。',
    // Timeout (ms)
    '超时时间（毫秒）': 'タイムアウト (ミリ秒)',
    // Enter the timeout
    '请输入超时时间': 'タイムアウトを入力してください',
    // Timeout of Passkey operations, defaults to 60000 (60 seconds)
    'Passkey 操作的超时时间，默认 60000（60 秒）': 'パスキー操作のタイムアウト、デフォルトは 60000 (60 秒)',
    // Single Sign-on Provider Options
    '单点登录 Provider 参数': 'シングル サインオン プロバイダーのパラメーター',
    // Enable OIDC Login
    '启用 OIDC 登录': 'OIDC ログインを有効にする',
    // Single sign-on with OpenID Connect
    '使用 OpenID Connect 进行单点登录': 'OpenID Connectを使用したシングルサインオン',
    // Issuer URL
    '颁发者地址': '発行者の住所',
    // Enter the issuer URL
    '请输入颁发者地址': '発行者のアドレスを入力してください',
    // e.g. https://idp.example.com; used to discover authorization_endpoint, token_endpoint, jwks_uri and other metadata; takes effect right after saving
    '示例：https://idp.example.com；用于自动发现 authorization_endpoint、token_endpoint、jwks_uri 等元数据；保存后立即生效': '例：https://idp.example.com。authorization_endpoint、token_endpoint、jwks_uri などのメタデータを自動検出します。保存後すぐに反映されます。',
    // Client ID
    '客户端 ID': 'クライアントID',
    // Enter the client ID
    '请输入客户端 ID': 'クライアントIDを入力してください',
    // Client Secret
    '客户端密钥': 'クライアントキー',
    // Callback URL
    '回调地址': 'コールバックアドレス',
    // Enter the callback URL
    '请输入回调地址': 'コールバックアドレスを入力してください',
    // e.g. https://isrvd.example.com/api/account/oidc/callback; auto-generated when empty in development, a fixed HTTPS callback is recommended in production
    '示例：https://isrvd.example.com/api/account/oidc/callback；开发环境可留空自动生成，生产环境建议填写固定 HTTPS 回调地址': '例：https://isrvd.example.com/api/account/oidc/callback。開発環境では空欄にすると自動生成されます。本番環境では固定の HTTPS コールバック URL を指定してください。',
    // Username Claim
    '用户名字段': 'ユーザー名フィールド',
    // Enter the username claim
    '请输入用户名字段': 'ユーザー名フィールドを入力してください',
    // Claim used as the username, defaults to sub; its value must match members.username exactly, sign-in fails when the user does not exist
    'OIDC 用户信息中作为用户名的字段，默认 sub；该字段的值必须与 members.username 完全一致，用户不存在时登录失败': 'OIDC ユーザー情報のユーザー名として使用されるフィールド。デフォルトは sub です。このフィールドの値は members.username とまったく同じである必要があります。ユーザーが存在しない場合はログインに失敗します。',
    // Scopes
    '授权范围': '認可範囲',
    // Enter the scopes
    '请输入授权范围': '認可範囲を入力してください',
    // e.g. openid profile email; space separated, openid is always included
    '示例：openid profile email；以空格分隔，系统会自动确保包含 openid': '例: openid プロファイル電子メール。スペースで区切ると、システムは自動的に openid が含まれていることを確認します。',
    // Login Button Label
    '登录按钮名称': 'ログインボタン名',
    // Enter the login button label
    '请输入登录按钮名称': 'ログインボタン名を入力してください',
    // Custom label of the OIDC login button; empty uses the default "Sign in with OIDC"
    '自定义 OIDC 登录按钮显示名称；留空则使用默认文案「使用 OIDC 登录」': 'OIDC ログイン ボタンの表示名をカスタマイズします。デフォルトのコピー「OIDC でログイン」を使用するには、空白のままにします。',
    // Proxy Header Login
    '代理 Header 登录': 'エージェントヘッダーログイン',
    // Read the username from an upstream proxy header
    '从上游代理 Header 读取用户名': 'アップストリームのプロキシ ヘッダーからユーザー名を読み取る',
    // Enable Proxy Header Login
    '启用代理 Header 登录': 'プロキシヘッダーログインを有効にする',
    // Uses the username header passed by the upstream proxy when enabled
    '开启后使用上游代理传入的用户名 Header': '開いた後、アップストリームプロキシによって渡されたユーザー名ヘッダーを使用します。',
    // Username Header
    '用户名 Header': 'ユーザー名のヘッダー',
    // Enter the header name
    '请输入 Header 名称': 'ヘッダー名を入力してください',
    // When enabled the header value is used as the login username; empty disables it
    '启用时，将使用该 Header 值作为登录用户名；留空则禁用': '有効にすると、このヘッダー値がログイン ユーザー名として使用されます。無効にする場合は空白のままにします',
    // Trusted Proxy CIDRs
    '可信代理 CIDR': '信頼できるプロキシ CIDR',
    // Enter proxy source CIDRs, one per line
    '请输入代理来源 CIDR，每行一个': 'プロキシ ソース CIDR を 1 行に 1 つずつ入力してください',
    // e.g. 127.0.0.1/32, 10.0.0.0/8; only listed proxy source IPs may pass the username header; saving empty fills in the loopback addresses (127.0.0.1/32, ::1/128)
    '示例：127.0.0.1/32、10.0.0.0/8；仅列出的代理来源 IP 允许传入用户名 Header；留空保存时默认填充本机回环地址（127.0.0.1/32、::1/128）': '例: 127.0.0.1/32、10.0.0.0/8;リストされたプロキシ ソース IP のみが受信ユーザー名ヘッダーを許可します。空白のままにすると、ローカル ループバック アドレス (127.0.0.1/32、::1/128) がデフォルトで入力されます。',
    // APISIX Admin API address, defaults to http://127.0.0.1:9180
    'APISIX Admin API 地址，默认 http://127.0.0.1:9180': 'APISIX 管理 API アドレス、デフォルト http://127.0.0.1:9180',
    // Key used to access the APISIX Admin API
    '访问 APISIX Admin API 的密钥': 'APISIX 管理 API にアクセスするためのキー',
    // Caddy Admin API address, defaults to http://127.0.0.1:2019
    'Caddy Admin API 地址，默认 http://127.0.0.1:2019': 'Caddy Admin API アドレス、デフォルト http://127.0.0.1:2019',
    // Engine connection, TLS certificates and the container root directory
    '引擎连接、TLS 证书与容器根目录': 'エンジン接続、TLS 証明書、およびコンテナー ルート',
    // Enter the Docker host
    '请输入 Docker Host': 'Dockerホストを入力してください',
    // e.g. unix:///var/run/docker.sock, tcp://host:2375 (plain) or tcp://host:2376 (with TLS below); empty uses the DOCKER_HOST environment variable
    '示例：unix:///var/run/docker.sock、tcp://host:2375（明文）或 tcp://host:2376（配合下方 TLS）；留空则使用环境变量 DOCKER_HOST': '例: unix:///var/run/docker.sock、tcp://host:2375 (プレーンテキスト) または tcp://host:2376 (以下は TLS)。環境変数 DOCKER_HOST を使用するには、空白のままにします。',
    // Used when connecting to remote Docker / Swarm; the Docker host must be a tcp:// address. Swarm and Docker share this connection, the target node must be a manager
    '连接远程 Docker / Swarm 时使用；Docker Host 必须是 tcp:// 地址。Swarm 与 Docker 共用此连接，目标节点需为 manager': 'リモートの Docker / Swarm に接続するときに使用されます。 Docker ホストは tcp:// アドレスである必要があります。 Swarm と Docker はこの接続を共有し、ターゲット ノードはマネージャーである必要があります',
    // Skip Server Certificate Verification
    '跳过服务端证书校验': 'サーバー証明書の検証をスキップする',
    // For testing only; man-in-the-middle attacks are no longer prevented
    '仅用于测试环境；开启后无法防范中间人攻击': 'テスト環境でのみ使用されます。有効にすると、中間者攻撃を防ぐことはできません。',
    // CA Certificate
    'CA 证书': 'CA証明書',
    // CA used to verify the server certificate (PEM, ca.pem); empty uses the system root certificates
    '用于校验服务端证书的 CA（PEM，即 ca.pem）；留空则使用系统根证书': 'CA (PEM、ca.pem) はサーバー証明書の検証に使用されます。システムのルート証明書を使用する場合は空白のままにします',
    // Required for mutual TLS (daemon started with --tlsverify) (PEM, cert.pem), paired with the private key; clearing it also clears the key
    '双向认证（daemon 启用 --tlsverify）时填写（PEM，即 cert.pem），需与私钥成对；清空证书会同时清除私钥': '双方向認証 (デーモンが --tlsverify を有効にする) の場合は (PEM、cert.pem) を入力します。これは秘密キーと組み合わせる必要があります。証明書をクリアすると秘密キーもクリアされます',
    // Private key of the client certificate (PEM, key.pem); never shown again after saving
    '客户端证书对应的私钥（PEM，即 key.pem）；保存后不会再显示': 'クライアント証明書に対応する秘密キー (PEM、つまり key.pem)。保存後は再度表示されません。',
    // Container Data Root
    '容器数据根目录': 'コンテナデータのルートディレクトリ',
    // Enter the container data root directory
    '请输入容器数据根目录': 'コンテナデータのルートディレクトリを入力してください',
    // Base directory for container volumes (relative to the base directory), defaults to containers
    '用于存放容器数据卷的基础目录（相对于基础目录），默认 containers': 'コンテナ データ ボリューム (ベース ディレクトリを基準とした相対値)、デフォルト コンテナの保存に使用されるベース ディレクトリ',
    // Pending
    '待接入': 'アクセス待ちです',
    // Revoked
    '已吊销': '取り消された',
    // Online
    '在线': 'オンライン',
    // List nodes
    '查询节点列表': 'ノード一覧を表示',
    // Update node
    '更新节点': 'ノードを更新',
    // Revoke Node
    '吊销节点': 'ノードを取り消す',
    // Revoke node <strong class="text-slate-900">{0}</strong>? The token becomes invalid and disconnects immediately, the node must be deleted and registered again.
    '确定要吊销节点 <strong class="text-slate-900">{0}</strong> 吗？吊销后令牌立即失效并断开连接，需要删除后重新注册才能再次接入。': 'ノード <strong class="text-slate-900">{0}</strong> を取り消してもよろしいですか?失効後、トークンは直ちに無効になり、接続は切断されます。再度アクセスするには、削除して再登録する必要があります。',
    // Confirm Revoke
    '确认吊销': '失効の確認',
    // Node revoked
    '节点已吊销': 'ノードが取り消されました',
    // Delete node <strong class="text-slate-900">{0}</strong>? Online nodes are disconnected immediately.
    '确定要删除节点 <strong class="text-slate-900">{0}</strong> 吗？在线的节点会被立即断开。': 'ノード <strong class="text-slate-900">{0}</strong> を削除してもよろしいですか?オンライン ノードはすぐに切断されます。',
    // Node deleted
    '节点已删除': 'ノードが削除されました',
    // Connect managed hosts and switch to a node to manage it
    '接入受管机，并切换到节点进行管理': '管理対象マシンに接続し、管理用ノードに切り替えます',
    // Connect and Manage Hosts
    '接入并管理受管机': '管理対象マシンへのアクセスと管理',
    // No managed node yet
    '暂无受管节点': '管理対象ノードはまだありません',
    // Node renamed
    '节点已重命名': 'ノードの名前が変更されました',
    // Rename Node
    '重命名节点': 'ノード名の変更',
    // Node Management
    '节点管理': 'ノード管理',
    // Port Forwarding
    '端口转发': 'ポートフォワーディング',
    // The center listens on ports and forwards through the tunnel to nodes (TCP / UDP)
    '中控监听端口，经隧道转发到节点（TCP / UDP）': 'コントローラーがポートで待ち受け、トンネル経由でノードへ転送します（TCP / UDP）',
    // Node port forwarding
    '节点端口转发': 'ノードのポート転送',
    // Search listen port, target, node or note...
    '搜索监听端口、目标、节点或备注...': 'リッスンポート、転送先、ノード、メモを検索...',
    // Add Forward
    '添加转发': '転送を追加',
    // Edit Forward
    '编辑转发': '転送を編集',
    // Delete Forward
    '删除转发': '転送を削除',
    // Listen Port
    '监听端口': 'リッスンポート',
    // Target Address
    '目标地址': '転送先アドレス',
    // No forwarding rules
    '暂无转发规则': '転送ルールはありません',
    // Click "Add Forward" at the top right to forward a node port to the center
    '点击右上角「添加转发」，把节点端口转发到中控': '右上の「転送を追加」からノードのポートをコントローラーに転送します',
    // No matching rules found
    '未找到匹配规则': '一致するルールが見つかりません',
    // No node available. Add a node in "Node Management" first
    '暂无可用节点，请先在「节点管理」中接入节点': '利用可能なノードがありません。先に「ノード管理」でノードを追加してください',
    // The node or forwarding rule no longer exists. Refresh and try again
    '节点或转发规则已不存在，请刷新后重试': 'ノードまたは転送ルールが存在しません。更新して再試行してください',
    // Forwarding rule saved
    '转发规则已保存': '転送ルールを保存しました',
    // Forwarding rule deleted
    '转发规则已删除': '転送ルールを削除しました',
    // Delete the forwarding rule for listen port <strong class="text-slate-900">{1}/{2}</strong> on node <strong class="text-slate-900">{0}</strong>? Established connections will be closed immediately.
    '确定要删除节点 <strong class="text-slate-900">{0}</strong> 上监听端口 <strong class="text-slate-900">{1}/{2}</strong> 的转发规则吗？已建立的连接会被立即断开。': 'ノード <strong class="text-slate-900">{0}</strong> のリッスンポート <strong class="text-slate-900">{1}/{2}</strong> の転送ルールを削除しますか？確立済みの接続は直ちに切断されます。',
    // The center listens on the given port and connects through the tunnel to a target on the node. TCP and UDP are supported.
    '中控监听指定端口，经隧道连接到节点上的目标地址，支持 TCP 与 UDP。': 'コントローラーが指定ポートで待ち受け、トンネル経由でノード上の転送先に接続します。TCP と UDP に対応しています。',
    // Forwarded connections bypass isrvd authentication; the target application is responsible for it. Do not forward services without authentication.
    '转发的连接不经过 isrvd 认证，鉴权由目标应用负责；请勿转发无认证的服务。': '転送される接続は isrvd の認証を通りません。認証は転送先アプリケーションの責任です。認証のないサービスは転送しないでください。',
    // Node
    '所属节点': '対象ノード',
    // Center Listen Port
    '中控监听端口': 'コントローラーのリッスンポート',
    // Must be within the range allowed by portmap.range in the center config file
    '须在中控配置 portmap.range 允许的范围内': 'コントローラーの設定 portmap.range で許可された範囲内にしてください',
    // Node Target Address
    '节点目标地址': 'ノード側の転送先アドレス',
    // host:port; a bare port points to 127.0.0.1 on the node
    'host:port；只填端口时默认指向节点本机 127.0.0.1': 'host:port。ポートのみの場合はノード自身の 127.0.0.1 を指します',
    // e.g. SSH
    '如：SSH': '例：SSH',
    // Save Forward
    '保存转发': '転送を保存',
    // Enter the node name
    '请输入节点名称': 'ノード名を入力してください',
    // 1–64 characters, only used to tell nodes apart in the UI
    '1–64 个字符，仅用于在界面中区分节点': '1 ～ 64 文字。インターフェイス内のノードを区別するためにのみ使用されます。',
    // Command copied to clipboard
    '命令已复制到剪贴板': 'コマンドがクリップボードにコピーされました',
    // Install with Docker
    'Docker 安装': 'Docker でインストール',
    // Install on bare metal
    '裸机安装': 'ベアメタルにインストール',
    // Note
    '备注': '備考',
    // e.g. Hangzhou IDC web-01
    '如：杭州机房 web-01': '例: 杭州コンピューター室 web-01',
    // Generate token
    '生成 token': 'トークンを生成',
    // Click Add Node to generate a token, then start the managed host in agent mode
    '点击右上角「接入节点」生成 token，并在受管机上以 agent 模式启动': '右上の「ノードを追加」をクリックしてトークンを生成し、管理対象ホストを agent モードで起動してください',
    // Run mode
    '运行模式': '実行モード',
    // Standalone mode
    '单机模式': 'スタンドアロンモード',
    // Center mode
    '中控模式': 'コントローラーモード',
    // Managed mode
    '受控模式': '管理対象モード',
    // Server URL
    '服务器地址': 'サーバー URL',
    // HTTP(S) URL of the center, without a path or query parameters
    '中控服务器的 HTTP(S) 地址，不含路径或查询参数': 'コントローラーの HTTP(S) URL（パスやクエリパラメーターは含めません）',
    // Connection token
    '连接 token': '接続トークン',
    // Use the token generated by the center; only one client per token may be online
    '直接填写中控生成的 token；同一 token 仅允许一个客户端在线': 'コントローラーが生成したトークンを入力してください。同じトークンでオンラインにできるクライアントは 1 つだけです',
    // Managed mode exposes no public port. Verify the URL and token; after restarting, access is only available through the center
    '受控模式不开放对外端口；请确认地址与 token 正确，重启后只能通过中控访问': '管理対象モードでは外部ポートを公開しません。URL とトークンが正しいことを確認してください。再起動後はコントローラー経由でのみアクセスできます',
    // Config saved; run mode, connection and listen address changes require a restart
    '配置已保存，运行模式、连接参数和监听地址变更需重启生效': '設定を保存しました。実行モード、接続パラメーター、リッスンアドレスの変更は再起動後に反映されます',
    // The token is shown only when created. Save it securely; it does not expire and permits only one client online.
    'token 仅在创建时显示，请妥善保存；长期有效，仅允许一个客户端在线。': 'トークンは作成時にのみ表示されます。安全に保管してください。有効期限はなく、オンラインにできるクライアントは 1 つだけです。',
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
