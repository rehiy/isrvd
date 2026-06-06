#!/bin/bash
set -e

# ------------------------------------------
# isrvd 服务管理脚本
# 用法: isrvd.sh install [--docker] [--caddy|--apisix] | update | uninstall | download
# ------------------------------------------

# 配置
SERVICE_NAME="isrvd"
INSTALL_DIR="/usr/local/isrvd"
CONFIG_FILE="$INSTALL_DIR/config.yml"
BIN_LINK="/usr/local/bin/isrvd"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
DOCKER_NETWORK="sdnet"
DOCKER_DATA_DIR="/srv/data"
DOCKER_CONF_DIR="$INSTALL_DIR/docker"

# ------------------------------------------
# 版本信息
# ------------------------------------------

get_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        armv7l|armhf) echo "arm" ;;
        *) uname -m ;;
    esac
}

get_latest_version() {
    curl -sI https://github.com/rehiy/isrvd/releases/latest 2>/dev/null | \
        grep -i "location:" | sed 's#.*/tag/##' | tr -d '\r\n' || echo "unknown"
}

ARCH=$(uname -s | tr '[:upper:]' '[:lower:]')-$(get_arch)
LATEST=$(get_latest_version)
DOWNLOAD_URL="https://github.com/rehiy/isrvd/releases/download/$LATEST/isrvd-$ARCH.tar.gz"

# ------------------------------------------
# systemctl 服务管理
# ------------------------------------------

service_install() {
    local bin_file=$(get_bin_file)

    cat > "$SERVICE_FILE" << EOF
[Unit]
Description=isrvd - Infrastructure Service Daemon
Documentation=https://github.com/rehiy/isrvd
After=network.target docker.service
Wants=docker.service

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
Environment="CONFIG_PATH=$CONFIG_FILE"
ExecStart=$bin_file
ExecReload=/bin/kill -HUP \$MAINPID
Restart=on-failure
RestartSec=5s

# 安全加固
NoNewPrivileges=true
LimitNOFILE=65536

# 日志
StandardOutput=journal
StandardError=journal
SyslogIdentifier=isrvd

[Install]
WantedBy=multi-user.target
EOF
    echo "[info] Installing systemd service"
    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME"
}

service_uninstall() {
    if [ ! -f "$SERVICE_FILE" ]; then
        return
    fi
    if systemctl is-enabled --quiet "$SERVICE_NAME" 2>/dev/null; then
        echo "[info] Disabling ${SERVICE_NAME} service..."
        systemctl disable "$SERVICE_NAME"
    fi
    echo "[info] Removing service file"
    rm -f "$SERVICE_FILE"
    systemctl daemon-reload
}

service_start() {
    echo "[info] Starting ${SERVICE_NAME} service..."
    systemctl start "$SERVICE_NAME"
}

service_stop() {
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        echo "[info] Stopping ${SERVICE_NAME} service..."
        systemctl stop "$SERVICE_NAME"
    fi
}

# ------------------------------------------
# 文件管理
# ------------------------------------------

get_bin_file() {
    find "$INSTALL_DIR" -type f -name "isrvd-*" -executable | head -1
}

files_install() {
    echo "[info] Creating directory: $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"

    echo "[info] Extracting to: $INSTALL_DIR"
    tar xzf "$1" -C "$INSTALL_DIR"

    # 创建命令链接
    local bin_file=$(get_bin_file)
    echo "[info] Creating symlink: $BIN_LINK -> $bin_file"
    ln -sf "$bin_file" "$BIN_LINK"
}

files_update() {
    echo "[info] Extracting to: $INSTALL_DIR"
    tar xzf "$1" -C "$INSTALL_DIR"

    # 更新符号链接
    local bin_file=$(get_bin_file)
    if [ ! -L "$BIN_LINK" ] || [ "$(readlink -f "$BIN_LINK")" != "$bin_file" ]; then
        echo "[info] Updating symlink: $BIN_LINK -> $bin_file"
        ln -sf "$bin_file" "$BIN_LINK"
    fi
}

files_uninstall() {
    if [ -L "$BIN_LINK" ]; then
        echo "[info] Removing symlink: $BIN_LINK"
        rm -f "$BIN_LINK"
    fi

    if [ -d "$INSTALL_DIR" ]; then
        echo "[info] Removing directory: $INSTALL_DIR"
        rm -rf "$INSTALL_DIR"
    fi
}

# ------------------------------------------
# 下载
# ------------------------------------------

download_package() {
    local tmp_file

    tmp_file=$(mktemp)
    echo "[info] Downloading: $DOWNLOAD_URL"

    if ! curl -fL --progress-bar -o "$tmp_file" "$DOWNLOAD_URL"; then
        echo "[error] Download failed"
        rm -f "$tmp_file"
        return 1
    fi

    DOWNLOAD_TMP_FILE="$tmp_file"
}


# ------------------------------------------
# Docker 组件安装
# ------------------------------------------

random_hex() {
    local bytes="${1:-16}"
    head -c "$bytes" /dev/urandom | od -An -tx1 | tr -d ' \n'
}

install_docker() {
    check_root

    if command -v docker >/dev/null 2>&1; then
        echo "[info] Docker already installed: $(docker --version)"
        return
    fi

    echo "[info] Installing Docker..."
    curl -fsSL https://get.docker.com -o get-docker.sh && \
        sh get-docker.sh --mirror Aliyun && \
        mkdir -p /etc/docker && \
        echo '{"registry-mirrors":["https://docker.1ms.run"]}' | tee /etc/docker/daemon.json && \
        systemctl restart docker
}

ensure_docker_ready() {
    install_docker

    if ! docker info >/dev/null 2>&1; then
        echo "[error] Docker is not running or current environment cannot access Docker"
        exit 1
    fi
}

ensure_docker_network() {
    if docker network inspect "$DOCKER_NETWORK" >/dev/null 2>&1; then
        echo "[info] Docker network exists: $DOCKER_NETWORK"
        return
    fi

    echo "[info] Creating Docker network: $DOCKER_NETWORK"
    docker network create --driver=bridge "$DOCKER_NETWORK"
}

ensure_isrvd_installed() {
    if [ -d "$INSTALL_DIR" ]; then
        echo "[info] isrvd already installed: $INSTALL_DIR"
        return
    fi
    install_binary
}

ensure_container_absent() {
    local name="$1"
    if docker ps -a --format '{{.Names}}' | grep -Fxq "$name"; then
        echo "[error] Docker container already exists: $name"
        echo "[info] Remove it first: docker rm -f $name"
        exit 1
    fi
}

config_replace_block() {
    local section="$1"
    local body="$2"
    local tmp_file

    mkdir -p "$(dirname "$CONFIG_FILE")"
    touch "$CONFIG_FILE"
    tmp_file=$(mktemp)

    awk -v section="$section" -v body="$body" '
        BEGIN { skip = 0; found = 0 }
        $0 == section ":" {
            print
            print body
            skip = 1
            found = 1
            next
        }
        skip && $0 ~ /^[^[:space:]#][^:]*:/ { skip = 0 }
        !skip { print }
        END {
            if (!found) {
                printf "\n%s:\n%s\n", section, body
            }
        }
    ' "$CONFIG_FILE" > "$tmp_file"

    mv "$tmp_file" "$CONFIG_FILE"
}
restart_isrvd() {
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        echo "[info] Restarting ${SERVICE_NAME} service..."
        systemctl restart "$SERVICE_NAME"
    fi
}

install_caddy_component() {
    ensure_docker_network
    ensure_container_absent isrvd-caddy

    mkdir -p "$DOCKER_CONF_DIR" "$DOCKER_DATA_DIR/caddy" "$DOCKER_DATA_DIR/caddy-config"
    cat > "$DOCKER_CONF_DIR/caddy.json" << 'EOF'
{
  "admin": {
    "config": {
      "persist": true
    },
    "listen": "0.0.0.0:2019"
  },
  "apps": {
    "http": {
      "servers": {
        "srv0": {
          "automatic_https": {
            "disable": true,
            "disable_redirects": true
          },
          "listen": [":80", ":443"],
          "routes": []
        }
      }
    }
  },
  "storage": {
    "module": "file_system",
    "root": "/data/caddy"
  }
}
EOF

    echo "[info] Starting Caddy container..."
    docker run -d \
        --name isrvd-caddy \
        --network "$DOCKER_NETWORK" \
        -p 80:80 \
        -p 443:443 \
        -p 127.0.0.1:2019:2019 \
        -v "$DOCKER_CONF_DIR/caddy.json:/etc/caddy/caddy.json" \
        -v "$DOCKER_DATA_DIR/caddy:/data/caddy" \
        -v "$DOCKER_DATA_DIR/caddy-config:/config" \
        caddy:2.8.4-alpine \
        caddy run --config /etc/caddy/caddy.json --adapter json

    config_replace_block "caddy" "  adminUrl: http://127.0.0.1:2019"
    restart_isrvd

    echo "[info] Caddy installed in Docker and configured for isrvd management."
    echo "[info] Caddy Admin API: http://127.0.0.1:2019"
}

install_apisix_component() {
    ensure_docker_network
    ensure_container_absent isrvd-apisix-etcd
    ensure_container_absent isrvd-apisix

    local admin_key
    admin_key=$(random_hex 16)

    mkdir -p "$DOCKER_CONF_DIR" "$DOCKER_DATA_DIR/apisix-etcd"
    cat > "$DOCKER_CONF_DIR/apisix.yaml" << EOF
apisix:
  node_listen: 9080
  enable_ipv6: false
  ssl:
    enable: true
    listen:
      - port: 9443
        enable_http3: false
  enable_control: true
  control:
    ip: 0.0.0.0
    port: 9090
  trusted_addresses:
    - 0.0.0.0/0
  dns_resolver:
    - 127.0.0.11
  dns_resolver_valid: 10

deployment:
  admin:
    allow_admin:
      - 0.0.0.0/0
    admin_key:
      - name: admin
        key: $admin_key
        role: admin
    admin_listen:
      ip: 0.0.0.0
      port: 9180
  etcd:
    host:
      - http://isrvd-apisix-etcd:2379
    prefix: /apisix
    timeout: 30

plugin_attr:
  prometheus:
    export_addr:
      ip: 0.0.0.0
      port: 9091
EOF

    echo "[info] Starting APISIX etcd container..."
    docker run -d \
        --name isrvd-apisix-etcd \
        --network "$DOCKER_NETWORK" \
        -e ALLOW_NONE_AUTHENTICATION=yes \
        -e ETCD_ADVERTISE_CLIENT_URLS=http://isrvd-apisix-etcd:2379 \
        -v "$DOCKER_DATA_DIR/apisix-etcd:/bitnami/etcd" \
        bitnami/etcd:3.5

    echo "[info] Waiting for APISIX etcd..."
    sleep 5

    echo "[info] Starting APISIX container..."
    docker run -d \
        --name isrvd-apisix \
        --network "$DOCKER_NETWORK" \
        -p 80:9080 \
        -p 443:9443 \
        -p 127.0.0.1:9180:9180 \
        -v "$DOCKER_CONF_DIR/apisix.yaml:/usr/local/apisix/conf/config.yaml" \
        apache/apisix:3.16.0-debian

    config_replace_block "apisix" "  adminUrl: http://127.0.0.1:9180/apisix/admin
  adminKey: $admin_key"
    restart_isrvd

    echo "[info] APISIX installed in Docker and configured for isrvd management."
    echo "[info] APISIX Admin API: http://127.0.0.1:9180/apisix/admin"
}

# ------------------------------------------
# 主流程
# ------------------------------------------

check_root() {
    if [ "$EUID" -ne 0 ]; then
        echo "[error] Please run as root"
        exit 1
    fi
}

install_binary() {
    check_root

    echo "=========================================="
    echo "  isrvd Installer"
    echo "=========================================="
    echo "  Version: $LATEST"
    echo "  Arch:    $ARCH"
    echo "=========================================="

    if [ -d "$INSTALL_DIR" ]; then
        echo "[error] Already installed: $INSTALL_DIR"
        echo "[info] Use 'update' to upgrade"
        exit 1
    fi

    if ! download_package; then
        exit 1
    fi

    files_install "$DOWNLOAD_TMP_FILE"
    service_install
    service_start

    rm -f "$DOWNLOAD_TMP_FILE"

    local bin_file=$(get_bin_file)

    echo ""
    echo "=========================================="
    echo "  Installation Complete!"
    echo "=========================================="
    echo "  Install: $INSTALL_DIR"
    echo "  Binary:  $bin_file"
    echo "  Config:  $CONFIG_FILE"
    echo ""
    echo "  Commands:"
    echo "    systemctl start $SERVICE_NAME"
    echo "    systemctl stop $SERVICE_NAME"
    echo "    systemctl status $SERVICE_NAME"
    echo "    journalctl -u $SERVICE_NAME -f"
    echo "=========================================="
}

install() {
    local with_docker=0
    local with_caddy=0
    local with_apisix=0

    if [ "$#" -eq 0 ]; then
        install_binary
        return
    fi

    while [ "$#" -gt 0 ]; do
        case "$1" in
            --docker) with_docker=1 ;;
            --caddy)  with_caddy=1 ;;
            --apisix) with_apisix=1 ;;
            --help|-h)
                print_usage
                return
                ;;
            *)
                echo "[error] Unknown install option: $1"
                print_usage
                exit 1
                ;;
        esac
        shift
    done

    if [ "$with_caddy" -eq 1 ] && [ "$with_apisix" -eq 1 ]; then
        echo "[error] --caddy and --apisix both expose ports 80/443; choose one gateway option"
        exit 1
    fi

    ensure_isrvd_installed

    if [ "$with_docker" -eq 1 ] || [ "$with_caddy" -eq 1 ] || [ "$with_apisix" -eq 1 ]; then
        ensure_docker_ready
    fi

    if [ "$with_caddy" -eq 1 ]; then
        install_caddy_component
    fi
    if [ "$with_apisix" -eq 1 ]; then
        install_apisix_component
    fi
}

update() {
    check_root

    echo "=========================================="
    echo "  isrvd Updater"
    echo "=========================================="
    echo "  Latest: $LATEST"
    echo "=========================================="

    if [ ! -d "$INSTALL_DIR" ]; then
        echo "[error] Not installed"
        echo "[info] Use 'install' first"
        exit 1
    fi

    service_stop

    if ! download_package; then
        service_start
        exit 1
    fi

    files_update "$DOWNLOAD_TMP_FILE"
    service_start

    rm -f "$DOWNLOAD_TMP_FILE"

    local bin_file=$(get_bin_file)

    echo ""
    echo "=========================================="
    echo "  Update Complete!"
    echo "=========================================="
    echo "  Install: $INSTALL_DIR"
    echo "  Binary:  $bin_file"
    echo "  Config:  $CONFIG_FILE (preserved)"
    echo "=========================================="
}

uninstall() {
    check_root

    echo "=========================================="
    echo "  isrvd Uninstaller"
    echo "=========================================="

    service_stop
    service_uninstall
    files_uninstall

    echo ""
    echo "=========================================="
    echo "  Uninstallation Complete!"
    echo "=========================================="
}

download() {
    echo "=========================================="
    echo "  isrvd Downloader"
    echo "=========================================="
    echo "  Version: $LATEST"
    echo "  Arch:    $ARCH"
    echo "  URL:     $DOWNLOAD_URL"
    echo "=========================================="

    if ! download_package; then
        exit 1
    fi

    mv "$DOWNLOAD_TMP_FILE" "isrvd-$ARCH.tar.gz"

    echo ""
    echo "[info] Downloaded: isrvd-$ARCH.tar.gz"
    ls -la "isrvd-$ARCH.tar.gz"
}

print_usage() {
    echo "Usage: $0 install [--docker] [--caddy|--apisix]"
    echo "       $0 {update|uninstall|download}"
    echo ""
    echo "Commands:"
    echo "  install               - Download and install isrvd systemd service"
    echo "  install --docker      - Install binary isrvd and Docker engine"
    echo "  install --caddy       - Install binary isrvd, Docker, Caddy container, and caddy.adminUrl"
    echo "  install --apisix      - Install binary isrvd, Docker, APISIX container, and apisix admin config"
    echo "  update                - Update systemd installation to latest version"
    echo "  uninstall             - Remove isrvd systemd service and config"
    echo "  download              - Download latest package to current directory"
    echo ""
    echo "Latest version: $LATEST"
}

# ------------------------------------------
# 入口
# ------------------------------------------

case "${1:-}" in
    install)
        shift
        install "$@"
        ;;
    update)         update ;;
    uninstall)      uninstall ;;
    download)       download ;;
    *)
        print_usage
        exit 1
        ;;
esac
