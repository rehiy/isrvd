# 镜像入口共用的配置初始化与首次启动提示。

install_default() {
    if [ ! -f "$2" ]; then
        echo "[init] Creating default config: $2"
        cp "$1" "$2"
    else
        echo "[init] Using existing config: $2"
    fi
}

print_default_password_notice() {
    if grep -q 'password: admin' "$1" 2>/dev/null; then
        echo "=========================================="
        echo "  首次启动 - 默认密码为 admin"
        echo "=========================================="
        echo "  isrvd 管理员用户: admin"
        echo "  isrvd 管理员密码: admin"
        echo ""
        echo "  请登录后立即修改密码"
        echo "=========================================="
    fi
}
