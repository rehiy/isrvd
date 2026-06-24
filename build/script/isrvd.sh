#!/bin/bash
set -euo pipefail

# 清理临时下载文件（若存在）
trap 'rm -f "${DOWNLOAD_TMP_FILE:-}" >/dev/null 2>&1 || true' EXIT

# ------------------------------------------
# isrvd 服务管理脚本
# 用法:
#   isrvd.sh install [--docker|--caddy|--apisix] [--cn|--global|--auto]
#   isrvd.sh update [--cn|--global|--auto]
#   isrvd.sh download [--cn|--global|--auto]
#   isrvd.sh uninstall
# iSrvd 源: 默认按 IP country_code 自动选择；可用 --cn/--global 手动指定
# ------------------------------------------

# 配置
SERVICE_NAME="isrvd"
INSTALL_DIR="/usr/local/isrvd"
CONFIG_FILE="$INSTALL_DIR/config.yml"
BIN_LINK="/usr/local/bin/isrvd"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
DOCKER_NETWORK="sdnet"
DOCKER_DATA_DIR="/srv/data"
SERVICE_AREA="auto"

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

parse_service_area_option() {
    case "$1" in
        --cn)     SERVICE_AREA="cn" ;;
        --global) SERVICE_AREA="global" ;;
        --auto)   SERVICE_AREA="auto" ;;
        *) return 1 ;;
    esac
    return 0
}

detect_country_code() {
    local code

    code=$(curl -fsSL --connect-timeout 2 --max-time 5 -A "isrvd-installer" "https://ipip.rehi.org/country_code" 2>/dev/null) || return 1
    if [ -z "$code" ]; then
        return 1
    fi

    printf '%s' "$code" | tr -d '[:space:]' | tr '[:lower:]' '[:upper:]'
}

resolve_service_area() {
    local country

    if [ "$SERVICE_AREA" = "auto" ]; then
        country=$(detect_country_code || true)
        if [ "$country" = "CN" ]; then
            printf 'cn:%s' "$country"
        else
            printf 'global:%s' "$country"
        fi
    else
        printf '%s:' "$SERVICE_AREA"
    fi
}


resolve_release_area() {
    local area_info
    area_info=$(resolve_service_area)
    RELEASE_AREA="${area_info%%:*}"
    RELEASE_COUNTRY="${area_info#*:}"
}

setup_release() {
    resolve_release_area

    if [ "$RELEASE_AREA" = "cn" ]; then
        RELEASE_URL="https://cnb.cool/rehiy/isrvd/-/releases/latest/download/isrvd-$ARCH.tar.gz"
        RELEASE_SOURCE="CNB"
    else
        RELEASE_URL="https://github.com/rehiy/isrvd/releases/latest/download/isrvd-$ARCH.tar.gz"
        RELEASE_SOURCE="GitHub"
    fi
}

print_release_info() {
    local area="$RELEASE_AREA"
    [ -n "$RELEASE_COUNTRY" ] && area="$area (country: $RELEASE_COUNTRY)"
    echo "  Arch:    $ARCH"
    echo "  Area:    $area"
    echo "  Source:  $RELEASE_SOURCE"
    [ -n "$DOCKER_IMAGE" ] && echo "  Image:   $DOCKER_IMAGE"
}

ARCH=$(uname -s | tr '[:upper:]' '[:lower:]')-$(get_arch)

# ------------------------------------------
# systemctl 服务管理
# ------------------------------------------

service_install() {
    local bin_file
    bin_file=$(get_bin_file) || exit 1

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
    local bin
    bin=$(find "$INSTALL_DIR" -type f -name "isrvd-*" -executable | head -1)
    if [ -z "$bin" ]; then
        echo "[error] No executable found in $INSTALL_DIR" >&2
        return 1
    fi
    printf '%s' "$bin"
}

files_install() {
    echo "[info] Creating directory: $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"

    echo "[info] Extracting to: $INSTALL_DIR"
    tar xzf "$1" -C "$INSTALL_DIR"

    # 创建命令链接
    local bin_file
    bin_file=$(get_bin_file) || exit 1
    echo "[info] Creating symlink: $BIN_LINK -> $bin_file"
    ln -sf "$bin_file" "$BIN_LINK"
}

files_update() {
    echo "[info] Extracting to: $INSTALL_DIR"
    tar xzf "$1" -C "$INSTALL_DIR"

    # 更新符号链接
    local bin_file
    bin_file=$(get_bin_file) || exit 1
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
    local download_url="$1"
    local tmp_file

    tmp_file=$(mktemp)
    echo "[info] Downloading: $download_url"

    if ! curl -fL --progress-bar -o "$tmp_file" "$download_url"; then
        echo "[error] Download failed"
        rm -f "$tmp_file"
        return 1
    fi

    DOWNLOAD_TMP_FILE="$tmp_file"
}


# ------------------------------------------
# Docker 版部署
# ------------------------------------------

install_docker() {
    if command -v docker >/dev/null 2>&1; then
        echo "[info] Docker already installed: $(docker --version)"
        return
    fi

    echo "[info] Installing Docker..."
    curl -fsSL https://get.docker.com -o get-docker.sh && \
        sh get-docker.sh --mirror Aliyun && \
        rm -f get-docker.sh && \
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

ensure_swarm_ready() {
    local state
    local advertise_addr

    state=$(docker info --format '{{.Swarm.LocalNodeState}}')
    if [ "$state" = "active" ]; then
        echo "[info] Docker Swarm already active"
        return
    fi
    if [ "$state" != "inactive" ]; then
        echo "[error] Docker Swarm state is $state; cannot initialize automatically"
        exit 1
    fi

    advertise_addr=$(ip route get 1.1.1.1 2>/dev/null | awk '{ for (i = 1; i <= NF; i++) if ($i == "src") { print $(i + 1); exit } }')
    echo "[info] Initializing Docker Swarm..."
    if [ -n "$advertise_addr" ]; then
        docker swarm init --advertise-addr "$advertise_addr"
    else
        docker swarm init
    fi
}

ensure_docker_network() {
    local driver
    local attachable

    if docker network inspect "$DOCKER_NETWORK" >/dev/null 2>&1; then
        driver=$(docker network inspect --format '{{.Driver}}' "$DOCKER_NETWORK")
        attachable=$(docker network inspect --format '{{.Attachable}}' "$DOCKER_NETWORK")
        if [ "$driver" != "overlay" ] || [ "$attachable" != "true" ]; then
            echo "[error] Docker network $DOCKER_NETWORK must be an attachable overlay network"
            exit 1
        fi
        echo "[info] Docker network exists: $DOCKER_NETWORK"
        return
    fi

    echo "[info] Creating attachable overlay network: $DOCKER_NETWORK"
    docker network create --driver=overlay --attachable "$DOCKER_NETWORK"
}

setup_docker_image() {
    local variant="$1"
    local repository

    resolve_release_area

    if [ "$RELEASE_AREA" = "cn" ]; then
        repository="docker.cnb.cool/rehiy/isrvd"
        RELEASE_SOURCE="CNB Docker"
    else
        repository="rehiy/isrvd"
        RELEASE_SOURCE="Docker Hub"
    fi
    DOCKER_IMAGE="$repository:$variant"
}

docker_variant_from_image() {
    local image="$1"
    local tag="${image##*:}"

    case "$tag" in
        slim|latest) echo "slim" ;;
        caddy)       echo "caddy" ;;
        apisix)      echo "apisix" ;;
        *)
            echo "[error] Cannot determine isrvd image variant from: $image" >&2
            return 1
            ;;
    esac
}

run_isrvd_container() {
    local variant="$1"
    local image="$2"
    local ports=(-p 8080:8080)

    case "$variant" in
        caddy)  ports+=(-p 80:80 -p 443:443) ;;
        apisix) ports+=(-p 80:9080 -p 443:9443) ;;
    esac

    docker run -d \
        --name "$SERVICE_NAME" \
        --restart unless-stopped \
        --network "$DOCKER_NETWORK" \
        "${ports[@]}" \
        -v "$DOCKER_DATA_DIR:/data" \
        -v /var/run/docker.sock:/var/run/docker.sock \
        "$image"
}

install_docker_deployment() {
    local variant="$1"

    check_root

    if [ -f "$SERVICE_FILE" ]; then
        echo "[error] Binary isrvd service is already installed"
        echo "[info] Run '$0 uninstall' before switching to the Docker deployment"
        exit 1
    fi

    ensure_docker_ready
    ensure_swarm_ready
    ensure_docker_network

    if docker ps -a --format '{{.Names}}' | grep -Fxq "$SERVICE_NAME"; then
        echo "[error] Docker container already exists: $SERVICE_NAME"
        echo "[info] Use '$0 update' to upgrade it"
        exit 1
    fi

    setup_docker_image "$variant"
    echo "=========================================="
    echo "  isrvd Docker Installer"
    echo "=========================================="
    print_release_info
    echo "=========================================="

    mkdir -p "$DOCKER_DATA_DIR"
    docker pull "$DOCKER_IMAGE"
    run_isrvd_container "$variant" "$DOCKER_IMAGE"

    echo "[info] Waiting for isrvd container..."
    sleep 3
    if ! docker ps --format '{{.Names}}' | grep -Fxq "$SERVICE_NAME"; then
        echo "[error] isrvd container exited unexpectedly"
        docker logs "$SERVICE_NAME" || true
        exit 1
    fi

    echo ""
    echo "=========================================="
    echo "  Installation Complete!"
    echo "=========================================="
    echo "  Container: $SERVICE_NAME"
    echo "  Image:     $DOCKER_IMAGE"
    echo "  Data:      $DOCKER_DATA_DIR"
    echo "  Network:   $DOCKER_NETWORK (overlay, attachable)"
    echo "  Web UI:    http://127.0.0.1:8080"
    echo "=========================================="
}

update_docker_deployment() {
    local old_image
    local old_image_id
    local variant

    old_image=$(docker inspect --format '{{.Config.Image}}' "$SERVICE_NAME")
    old_image_id=$(docker inspect --format '{{.Image}}' "$SERVICE_NAME")
    variant=$(docker_variant_from_image "$old_image") || exit 1

    ensure_docker_ready
    ensure_swarm_ready
    ensure_docker_network
    setup_docker_image "$variant"

    echo "=========================================="
    echo "  isrvd Docker Updater"
    echo "=========================================="
    print_release_info
    echo "=========================================="

    docker pull "$DOCKER_IMAGE"
    docker rm -f "$SERVICE_NAME"
    if ! run_isrvd_container "$variant" "$DOCKER_IMAGE"; then
        echo "[error] Failed to start updated container; restoring previous image"
        docker rm -f "$SERVICE_NAME" 2>/dev/null || true
        run_isrvd_container "$variant" "$old_image_id"
        exit 1
    fi

    sleep 3
    if ! docker ps --format '{{.Names}}' | grep -Fxq "$SERVICE_NAME"; then
        echo "[error] Updated container exited; restoring previous image"
        docker logs "$SERVICE_NAME" || true
        docker rm -f "$SERVICE_NAME" >/dev/null 2>&1 || true
        run_isrvd_container "$variant" "$old_image_id"
        exit 1
    fi

    echo "[info] Docker deployment updated: $DOCKER_IMAGE"
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

    setup_release

    echo "=========================================="
    echo "  isrvd Installer"
    echo "=========================================="
    print_release_info
    echo "=========================================="

    if [ -d "$INSTALL_DIR" ]; then
        echo "[error] Already installed: $INSTALL_DIR"
        echo "[info] Use 'update' to upgrade"
        exit 1
    fi

    if ! download_package "$RELEASE_URL"; then
        exit 1
    fi

    files_install "$DOWNLOAD_TMP_FILE"
    service_install
    service_start

    rm -f "$DOWNLOAD_TMP_FILE"

    local bin_file
    bin_file=$(get_bin_file) || bin_file="(unknown)"

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
    local variant=""

    while [ "$#" -gt 0 ]; do
        case "$1" in
            --cn|--global|--auto)
                parse_service_area_option "$1"
                ;;
            --docker|--caddy|--apisix)
                if [ -n "$variant" ]; then
                    echo "[error] Choose only one Docker image option: --docker, --caddy, or --apisix"
                    exit 1
                fi
                case "$1" in
                    --docker) variant="slim" ;;
                    --caddy)  variant="caddy" ;;
                    --apisix) variant="apisix" ;;
                esac
                ;;
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

    if [ -z "$variant" ]; then
        install_binary
        return
    fi
    install_docker_deployment "$variant"
}

update() {
    while [ "$#" -gt 0 ]; do
        case "$1" in
            --cn|--global|--auto)
                parse_service_area_option "$1"
                ;;
            --help|-h)
                print_usage
                return
                ;;
            *)
                echo "[error] Unknown update option: $1"
                print_usage
                exit 1
                ;;
        esac
        shift
    done

    check_root

    if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
        if docker ps -a --format '{{.Names}}' | grep -Fxq "$SERVICE_NAME"; then
            update_docker_deployment
            return
        fi
    fi

    setup_release

    echo "=========================================="
    echo "  isrvd Updater"
    echo "=========================================="
    print_release_info
    echo "=========================================="

    if [ ! -d "$INSTALL_DIR" ]; then
        echo "[error] Not installed"
        echo "[info] Use 'install' first"
        exit 1
    fi

    service_stop

    if ! download_package "$RELEASE_URL"; then
        service_start
        exit 1
    fi

    files_update "$DOWNLOAD_TMP_FILE"
    service_start

    rm -f "$DOWNLOAD_TMP_FILE"

    local bin_file
    bin_file=$(get_bin_file) || bin_file="(unknown)"

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
    local container
    local removed=0

    check_root
    echo "=========================================="
    echo "  isrvd Uninstaller"
    echo "=========================================="

    if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
        for container in "$SERVICE_NAME" isrvd-caddy isrvd-apisix isrvd-apisix-etcd; do
            if docker ps -a --format '{{.Names}}' | grep -Fxq "$container"; then
                echo "[info] Removing Docker container: $container"
                docker rm -f "$container"
                removed=1
            fi
        done
        [ "$removed" -eq 1 ] && echo "[info] Docker data preserved: $DOCKER_DATA_DIR"
    fi

    if [ -f "$SERVICE_FILE" ] || [ -d "$INSTALL_DIR" ]; then
        service_stop
        service_uninstall
        files_uninstall
        removed=1
    fi

    if [ "$removed" -eq 0 ]; then
        echo "[info] isrvd is not installed"
    fi

    echo ""
    echo "=========================================="
    echo "  Uninstallation Complete!"
    echo "=========================================="
}

download() {
    while [ "$#" -gt 0 ]; do
        case "$1" in
            --cn|--global|--auto)
                parse_service_area_option "$1"
                ;;
            --help|-h)
                print_usage
                return
                ;;
            *)
                echo "[error] Unknown download option: $1"
                print_usage
                exit 1
                ;;
        esac
        shift
    done

    setup_release

    echo "=========================================="
    echo "  isrvd Downloader"
    echo "=========================================="
    print_release_info
    echo "  URL:     $RELEASE_URL"
    echo "=========================================="

    if ! download_package "$RELEASE_URL"; then
        exit 1
    fi

    mv "$DOWNLOAD_TMP_FILE" "isrvd-$ARCH.tar.gz"

    echo ""
    echo "[info] Downloaded: isrvd-$ARCH.tar.gz"
    ls -la "isrvd-$ARCH.tar.gz"
}

print_usage() {
    echo "Usage: $0 install [--docker|--caddy|--apisix] [--cn|--global|--auto]"
    echo "       $0 update [--cn|--global|--auto]"
    echo "       $0 download [--cn|--global|--auto]"
    echo "       $0 uninstall"
    echo ""
    echo "Commands:"
    echo "  install               - Download and install isrvd systemd service"
    echo "  install --docker      - Install rehiy/isrvd:slim with Docker, Swarm, and sdnet"
    echo "  install --caddy       - Install rehiy/isrvd:caddy with Docker, Swarm, and sdnet"
    echo "  install --apisix      - Install rehiy/isrvd:apisix with Docker, Swarm, and sdnet"
    echo "  update                - Update the current binary or Docker deployment"
    echo "  uninstall             - Remove the current deployment (Docker data is preserved)"
    echo "  download              - Download latest package to current directory"
    echo ""
    echo "iSrvd source:"
    echo "  --auto               - Detect country_code via IP and use CNB when country_code=CN (default)"
    echo "  --cn                 - Use CNB releases or docker.cnb.cool/rehiy/isrvd images"
    echo "  --global             - Use GitHub releases or rehiy/isrvd images from Docker Hub"
    echo ""
    echo "Note: Docker installs initialize Swarm and create the attachable overlay network sdnet."
}

# ------------------------------------------
# 入口
# ------------------------------------------

case "${1:-}" in
    install)
        shift
        install "$@"
        ;;
    update)
        shift
        update "$@"
        ;;
    uninstall)      uninstall ;;
    download)
        shift
        download "$@"
        ;;
    *)
        print_usage
        exit 1
        ;;
esac
