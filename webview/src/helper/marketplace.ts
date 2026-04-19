// 应用市场 postMessage 协议与安装脚本生成

export interface MarketplaceInstallPayload {
    source: 'marketplace'
    protocol: number
    type: 'install'
    timestamp: number
    app: {
        key: string
        name: string
        title?: string
        tags?: string[]
        architectures?: string[]
        icon?: string
    }
    version: {
        value: string
        url: string
        path: string
    }
    instance: {
        name: string
        safeName: string
    }
    env: {
        system: Record<string, string>
        user: Record<string, string | number | boolean>
    }
    formFields?: unknown[]
    deploy: {
        network: string
    }
}

// 本地默认安装根目录，仅作脚本中 INSTALL_ROOT 的默认值
const LOCAL_INSTALL_ROOT = '/opt/apps'

// 校验 postMessage 数据是否为合法的安装 payload
export function isMarketplaceInstallPayload(data: unknown): data is MarketplaceInstallPayload {
    if (!data || typeof data !== 'object') return false
    const d = data as Record<string, unknown>
    if (d.source !== 'marketplace' || d.type !== 'install') return false
    if (typeof d.app !== 'object' || d.app === null) return false
    if (typeof d.version !== 'object' || d.version === null) return false
    if (typeof d.instance !== 'object' || d.instance === null) return false
    if (typeof d.env !== 'object' || d.env === null) return false
    if (typeof d.deploy !== 'object' || d.deploy === null) return false
    return true
}

// 生成安装 shell 脚本（与 build/marketplace/index.html 中 buildInstallScript 逻辑保持一致）
export function buildInstallScript(payload: MarketplaceInstallPayload): string {
    // 转义 shell 单引号：a'b -> 'a'\''b'
    const shq = (v: unknown) => `'${String(v).replace(/'/g, `'\\''`)}'`
    // 转义 .env 值：双引号包裹，转义 " \ $ `
    const envEsc = (v: unknown) => `"${String(v).replace(/[\\"`$]/g, '\\$&')}"`

    const { app, version, instance, env, deploy } = payload

    const mergedEnv: Record<string, unknown> = { ...env.user, ...env.system }

    const envLines = Object.entries(mergedEnv)
        .map(([k, v]) => `${k}=${envEsc(v == null ? '' : v)}`)
        .join('\n')

    return [
        `#!/usr/bin/env bash`,
        `# ${app.key} ${version.value} 安装脚本`,
        `set -euo pipefail`,
        ``,
        `# ============ 基础依赖检查 ============`,
        `command -v docker >/dev/null 2>&1 || { echo "✗ 未检测到 docker，请先安装 Docker" >&2; exit 1; }`,
        `docker compose version >/dev/null 2>&1 || { echo "✗ 未检测到 docker compose 插件（docker compose v2）" >&2; exit 1; }`,
        `command -v curl  >/dev/null 2>&1 || { echo "✗ 未检测到 curl"  >&2; exit 1; }`,
        `command -v unzip >/dev/null 2>&1 || { echo "✗ 未检测到 unzip" >&2; exit 1; }`,
        ``,
        `# ============ 配置 ============`,
        `APP_KEY=${shq(app.key)}`,
        `APP_NAME=${shq(instance.safeName)}`,
        `APP_VERSION=${shq(version.value)}`,
        `ZIP_URL=${shq(version.url)}`,
        `INSTALL_ROOT=${shq(LOCAL_INSTALL_ROOT)}`,
        `INSTALL_DIR="$INSTALL_ROOT/$APP_KEY/$APP_NAME"`,
        `NETWORK_NAME=${shq(deploy.network)}`,
        ``,
        `# ============ 确保应用所需网络存在 ============`,
        `if ! docker network inspect "$NETWORK_NAME" >/dev/null 2>&1; then`,
        `  echo "==> 创建网络 $NETWORK_NAME"`,
        `  docker network create "$NETWORK_NAME" >/dev/null`,
        `fi`,
        ``,
        `# ============ 准备目录 ============`,
        `if [ -d "$INSTALL_DIR" ]; then`,
        `  echo "✗ 目录已存在：$INSTALL_DIR" >&2`,
        `  echo "  请先移除或使用其它实例名" >&2`,
        `  exit 1`,
        `fi`,
        `mkdir -p "$INSTALL_DIR"`,
        `cd "$INSTALL_DIR"`,
        ``,
        `# ============ 下载并解压应用包 ============`,
        `echo "==> 下载 $APP_KEY:$APP_VERSION"`,
        `curl -fL --retry 3 -o app.zip "$ZIP_URL"`,
        `echo "==> 解压"`,
        `unzip -oq app.zip`,
        `rm -f app.zip`,
        ``,
        `# ============ 写入 .env ============`,
        `echo "==> 写入 .env"`,
        `cat > .env <<'MARKETPLACE_ENV_EOF'`,
        envLines,
        `MARKETPLACE_ENV_EOF`,
        `chmod 600 .env`,
        ``,
        `# ============ 启动 ============`,
        `echo "==> docker compose up -d"`,
        `docker compose -p "$APP_NAME" up -d`,
        ``,
        `echo ""`,
        `echo "✓ 安装完成"`,
        `echo "  应用：$APP_KEY ($APP_VERSION)"`,
        `echo "  实例：$APP_NAME"`,
        `echo "  目录：$INSTALL_DIR"`,
        ``,
    ].join('\n')
}
