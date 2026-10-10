import type { LanguageModule } from '../types'

/**
 * 日本語訳：キーは中国語原文（簡体字）
 *
 * よく使われる UI 文言のみ登録済み（44/1897）。未登録の文案は中国語原文に
 * フォールバックするため、空白やキーの漏洩は発生しない。
 *
 * 訳文を追加するときは、未訳リストを書き出して埋めるのが早い：
 *   node scripts/check-locales.mjs --worklist ja > tmp/ja-ui.jsonl
 * 登録後は node scripts/check-locales.mjs で取りこぼしを確認する。
 * このファイルは手書きでもよいが、scripts/apply-translations.mjs で取り込むと
 * 英文コメントと並び順が保たれる。
 */

const messages: Record<string, string> = {
    '系统概览': '概要',
    '文件管理': 'ファイル管理',
    'Web 终端': 'Web ターミナル',
    '本机进程': 'ローカルプロセス',
    'SSH 远程管理': 'SSH リモート管理',
    '集中管理': '集中管理',
    'AI 助手': 'AI アシスタント',
    '计划任务': 'スケジュールタスク',
    'APISIX': 'APISIX',
    'Caddy': 'Caddy',
    'Docker': 'Docker',
    'Swarm': 'Swarm',
    'Compose': 'Compose',
    '成员管理': 'メンバー管理',
    '系统管理': 'システム管理',
    '账号安全': 'アカウントセキュリティ',
    'API Key': 'API キー',
    'Passkey': 'Passkey',
    '登录': 'ログイン',
    '退出': 'ログアウト',
    '保存': '保存する',
    '取消': 'キャンセル',
    '删除': '削除',
    '确定': '確定',
    '创建': '作成',
    '编辑': '編集',
    '刷新': '更新',
    '搜索': '検索',
    '启动': '起動',
    '停止': '停止する',
    '重启': '再起動',
    '暂停': '一時停止',
    '恢复': '再開',
    '容器': 'コンテナ',
    '镜像': 'イメージ',
    '网络': 'ネットワーク',
    '卷': 'ボリューム',
    '节点': 'ノード',
    '服务': 'サービス',
    '日志': 'ログ',
    '配置': '設定',
    '权限': '権限',
    '操作审计': '操作監査',
    '设置': '設定',
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
