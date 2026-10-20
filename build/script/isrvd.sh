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
ROLE_DIR="/etc/systemd/system/${SERVICE_NAME}.service.d"
ROLE_FILE="$ROLE_DIR/installer.conf"
DOCKER_NETWORK="sdnet"
DOCKER_DATA_DIR="/srv/data"
SERVICE_AREA="auto"
DOCKER_IMAGE=""
DOCKER_VARIANT=""
DEPLOYMENT_ENV=()

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

parse_options() {
    local command="$1"
    shift
    while [ "$#" -gt 0 ]; do
        case "$1" in
            --cn|--global|--auto) SERVICE_AREA="${1#--}" ;;
            --docker|--caddy|--apisix)
                [ "$command" = install ] || fail "Image options are only valid for install"
                [ -z "$DOCKER_VARIANT" ] || fail "Choose only one Docker image option"
                DOCKER_VARIANT="${1#--}"
                [ "$DOCKER_VARIANT" != docker ] || DOCKER_VARIANT=slim
                ;;
            --help|-h) print_usage; exit 0 ;;
            *) fail "Unknown $command option: $1" ;;
        esac
        shift
    done
}

detect_country_code() {
    local code

    code=$(curl -fsSL --connect-timeout 2 --max-time 5 -A "isrvd-installer" "https://ipip.rehi.org/country_code" 2>/dev/null) || return 1
    if [ -z "$code" ]; then
        return 1
    fi

    printf '%s' "$code" | tr -d '[:space:]' | tr '[:lower:]' '[:upper:]'
}

resolve_release_area() {
    RELEASE_AREA="$SERVICE_AREA"
    RELEASE_COUNTRY=""
    if [ "$RELEASE_AREA" = auto ]; then
        RELEASE_COUNTRY=$(detect_country_code || true)
        RELEASE_AREA=global
        [ "$RELEASE_COUNTRY" != CN ] || RELEASE_AREA=cn
    fi
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
    if [ -n "${DOCKER_IMAGE:-}" ]; then
        echo "  Image:   $DOCKER_IMAGE"
    fi
}

ARCH=$(uname -s | tr '[:upper:]' '[:lower:]')-$(get_arch)

# ------------------------------------------
# systemctl 服务管理
# ------------------------------------------

# 安装时读取当前进程环境；更新时沿用已有服务配置。
prepare_deployment_env() {
    DEPLOYMENT_MODE="${ISRVD_MODE:-server}"
    DEPLOYMENT_MODE="${DEPLOYMENT_MODE,,}"
    case "$DEPLOYMENT_MODE" in
        server|center) ;;
        agent)
            if [ -z "${ISRVD_CENTER_URL:-}" ] || [ -z "${ISRVD_NODE_TOKEN:-}" ]; then
                echo "[error] Agent mode requires ISRVD_CENTER_URL and ISRVD_NODE_TOKEN" >&2
                return 1
            fi
            ;;
        *)
            echo "[error] ISRVD_MODE must be server, center, or agent" >&2
            return 1
            ;;
    esac
    DEPLOYMENT_ENV=("ISRVD_MODE=$DEPLOYMENT_MODE")
    if [ "$DEPLOYMENT_MODE" = agent ]; then
        DEPLOYMENT_ENV+=("ISRVD_CENTER_URL=$ISRVD_CENTER_URL" "ISRVD_NODE_TOKEN=$ISRVD_NODE_TOKEN")
    fi
}

# systemd Environment 使用自己的引号与 specifier 规则，不能直接套用 shell 转义。
systemd_environment() {
    local value="$1"
    value="${value//\\/\\\\}"
    value="${value//\"/\\\"}"
    value="${value//%/%%}"
    value="${value//$'\n'/\\n}"
    value="${value//$'\r'/\\r}"
    printf 'Environment="%s"\n' "$value"
}

service_role_install() {
    mkdir -p "$ROLE_DIR"
    (
        umask 077
        echo '[Service]' > "$ROLE_FILE"
        local entry
        for entry in "${DEPLOYMENT_ENV[@]}"; do
            systemd_environment "$entry" >> "$ROLE_FILE"
        done
    )
}

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
Environment="ISRVD_CONFIG=$CONFIG_FILE"
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
    service_role_install
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
    rm -f "$ROLE_FILE"
    rmdir "$ROLE_DIR" 2>/dev/null || true
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
    local bin="$INSTALL_DIR/isrvd-$ARCH"
    [ -x "$bin" ] || fail "No executable found: $bin"
    printf '%s' "$bin"
}

files_install() {
    local options=()
    [ "${1:-install}" != update ] || options+=(--exclude=config.yml)
    mkdir -p "$INSTALL_DIR" || return 1
    echo "[info] Extracting to: $INSTALL_DIR"
    tar xzf "$DOWNLOAD_TMP_FILE" "${options[@]}" -C "$INSTALL_DIR" || return 1
    local bin_file
    bin_file=$(get_bin_file) || return 1
    ln -sf "$bin_file" "$BIN_LINK"
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
    DOWNLOAD_TMP_FILE=$(mktemp)
    echo "[info] Downloading: $1"
    curl -fL --progress-bar -o "$DOWNLOAD_TMP_FILE" "$1"
}

# ------------------------------------------
# Docker 版部署
# ------------------------------------------

install_docker() {
    if command -v docker >/dev/null 2>&1; then
        return
    fi
    echo "[info] Installing Docker..."
    download_package https://get.docker.com
    sh "$DOWNLOAD_TMP_FILE" --mirror Aliyun
    mkdir -p /etc/docker
    echo '{"registry-mirrors":["https://docker.1ms.run"]}' > /etc/docker/daemon.json
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

    advertise_addr=$(ip route get 1.1.1.1 2>/dev/null | awk '{ for (i = 1; i <= NF; i++) if ($i == "src") { print $(i + 1); exit } }') || advertise_addr=""
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
        read -r driver attachable <<< "$(docker network inspect --format '{{.Driver}} {{.Attachable}}' "$DOCKER_NETWORK")"
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

# 读取 Docker 数据目录配置里的 forward.ports；未配置返回空，格式无效时终止。
docker_forward_ports() {
    local file="$DOCKER_DATA_DIR/conf/isrvd.yml" value
    [ -f "$file" ] || return 0
    value=$(awk '
        /^[^[:space:]#]/ { in_forward = ($0 ~ /^forward:[[:space:]]*(#.*)?$/); next }
        in_forward && /^[[:space:]]+ports:/ {
            sub(/^[[:space:]]+ports:[[:space:]]*/, ""); sub(/[[:space:]]*#.*$/, ""); gsub(/["'"'"']/, ""); print; exit
        }' "$file")
    [ -z "$value" ] || [[ "$value" =~ ^[0-9]+(-[0-9]+)?$ ]] || fail "forward.ports in $file must look like 20000-20999 or 20022"
    printf '%s' "$value"
}

run_isrvd_container() {
    local variant="$1"
    local image="$2"
    local ports=()
    local environment=()
    local mode="server"
    local entry
    for entry in "${DEPLOYMENT_ENV[@]}"; do
        environment+=(-e "$entry")
        case "$entry" in ISRVD_MODE=*) mode="${entry#*=}" ;; esac
    done
    mode="${mode,,}"
    if [ "$mode" != "agent" ]; then
        ports+=(-p 8080:8080)
    fi
    if [ "$mode" = "center" ]; then
        # 端口转发的监听范围（配置文件 forward.ports）需要发布到宿主机（TCP 与 UDP）；容器内须同时设置 forward.bind: 0.0.0.0
        local forward_ports
        forward_ports=$(docker_forward_ports)
        if [ -n "$forward_ports" ]; then
            ports+=(-p "$forward_ports:$forward_ports" -p "$forward_ports:$forward_ports/udp")
        fi
    fi

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
        "${environment[@]}" \
        "$image"
}

install_docker_deployment() {
    [ ! -f "$SERVICE_FILE" ] || fail "Binary service already installed; uninstall before switching deployments"
    ensure_docker_ready
    ! container_exists "$SERVICE_NAME" || fail "Container already exists; use update"
    ensure_swarm_ready
    ensure_docker_network
    setup_docker_image "$DOCKER_VARIANT"
    print_banner "Docker Installer"
    mkdir -p "$DOCKER_DATA_DIR"
    docker pull "$DOCKER_IMAGE"
    run_isrvd_container "$DOCKER_VARIANT" "$DOCKER_IMAGE"
    container_started || fail "isrvd container exited unexpectedly"
    cat <<EOF
[info] Installation complete
  Container: $SERVICE_NAME
  Data:      $DOCKER_DATA_DIR
  Config:    $DOCKER_DATA_DIR/conf/isrvd.yml
  Mode:      $DEPLOYMENT_MODE
EOF
    if [ "$DEPLOYMENT_MODE" = agent ]; then
        echo "  Access:    Open this node from the center console"
    else
        echo "  Web UI:    http://127.0.0.1:8080"
    fi
    if [ "$DEPLOYMENT_MODE" = center ]; then
        echo "  Forward:   To enable port forwarding, set forward.ports and forward.bind: 0.0.0.0 in the config, then run update"
    fi
}

update_docker_deployment() {
    local old_image variant old_env entry
    local backup="$SERVICE_NAME-update-backup"
    ! container_exists "$backup" || fail "Container $backup already exists; resolve it before updating"
    old_env=$(docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "$SERVICE_NAME")
    while IFS= read -r entry; do
        case "$entry" in
            ISRVD_MODE=*|ISRVD_CENTER_URL=*|ISRVD_NODE_TOKEN=*) DEPLOYMENT_ENV+=("$entry") ;;
        esac
    done <<< "$old_env"
    old_image=$(docker inspect --format '{{.Config.Image}}' "$SERVICE_NAME")
    variant=$(docker_variant_from_image "$old_image")
    ensure_swarm_ready
    ensure_docker_network
    setup_docker_image "$variant"
    print_banner "Docker Updater"
    docker pull "$DOCKER_IMAGE"

    # 保留旧容器本身，失败时恢复原有镜像、环境、挂载与端口。
    docker rename "$SERVICE_NAME" "$backup"
    if ! docker stop "$backup"; then
        docker rename "$backup" "$SERVICE_NAME"
        fail "Cannot stop the previous container"
    fi
    if run_isrvd_container "$variant" "$DOCKER_IMAGE" && container_started; then
        docker rm "$backup"
        echo "[info] Docker deployment updated: $DOCKER_IMAGE"
        return
    fi
    echo "[error] Update failed; restoring previous container" >&2
    docker rm -f "$SERVICE_NAME" >/dev/null 2>&1 || true
    docker rename "$backup" "$SERVICE_NAME"
    docker start "$SERVICE_NAME"
    return 1
}

# ------------------------------------------
# 主流程
# ------------------------------------------

fail() {
    echo "[error] $*" >&2
    exit 1
}

check_root() {
    [ "$EUID" -eq 0 ] || fail "Please run as root"
}

print_banner() {
    echo "=========================================="
    echo "  isrvd $1"
    print_release_info
    echo "=========================================="
}

container_exists() {
    docker container inspect "$1" >/dev/null 2>&1
}

container_started() {
    sleep 3
    if [ "$(docker inspect --format '{{.State.Running}}' "$SERVICE_NAME")" = true ]; then
        return 0
    fi
    docker logs "$SERVICE_NAME" >&2 || true
    return 1
}

print_binary_info() {
    cat <<EOF
  Install: $INSTALL_DIR
  Binary:  $(get_bin_file)
  Config:  $CONFIG_FILE
EOF
}

install_binary() {
    [ ! -e "$INSTALL_DIR" ] && [ ! -e "$SERVICE_FILE" ] || fail "Already installed; use update"
    if [ -e "$ROLE_FILE" ] || [ -L "$ROLE_FILE" ] || [ -e "$ROLE_DIR/role.conf" ] || [ -L "$ROLE_DIR/role.conf" ]; then
        fail "Existing role configuration found; refusing to overwrite it"
    fi
    setup_release
    print_banner Installer
    download_package "$RELEASE_URL"
    files_install
    service_install
    service_start
    echo "[info] Installation complete (mode: $DEPLOYMENT_MODE)"
    print_binary_info
    echo "  Manage: systemctl {start|stop|status|edit} $SERVICE_NAME"
    echo "  Logs:   journalctl -u $SERVICE_NAME -f"
}

install() {
    prepare_deployment_env
    if [ -n "$DOCKER_VARIANT" ]; then
        install_docker_deployment
    else
        install_binary
    fi
}

update() {
    if command -v docker >/dev/null 2>&1 && container_exists "$SERVICE_NAME"; then
        update_docker_deployment
        return
    fi
    [ -d "$INSTALL_DIR" ] || fail "Not installed; use install first"
    setup_release
    print_banner Updater
    # 下载与归档检查在停服之前完成，下载失败不影响正在运行的服务。
    download_package "$RELEASE_URL"
    tar tzf "$DOWNLOAD_TMP_FILE" >/dev/null
    service_stop
    if ! files_install update; then
        service_start
        fail "Failed to extract update"
    fi
    service_start
    echo "[info] Update complete (configuration preserved)"
    print_binary_info
}

uninstall() {
    local container
    local removed=0

    echo "=========================================="
    echo "  isrvd Uninstaller"
    echo "=========================================="

    if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
        for container in "$SERVICE_NAME" isrvd-caddy isrvd-apisix isrvd-apisix-etcd; do
            if container_exists "$container"; then
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
    setup_release
    print_banner Downloader
    download_package "$RELEASE_URL"
    mv "$DOWNLOAD_TMP_FILE" "isrvd-$ARCH.tar.gz"
    echo "[info] Downloaded: isrvd-$ARCH.tar.gz"
}

print_usage() {
    cat <<EOF
Usage: $0 install [--docker|--caddy|--apisix] [--cn|--global|--auto]
       $0 update [--cn|--global|--auto]
       $0 download [--cn|--global|--auto]
       $0 uninstall

Commands:
  install     Install the systemd service (default) or a Docker image variant
  update      Update the installed binary or Docker deployment
  download    Download the latest binary package to the current directory
  uninstall   Remove the deployment; Docker data is preserved

Source: --auto (default, CNB in CN), --cn (CNB), --global (GitHub/Docker Hub)
Docker variants: --docker (slim), --caddy, --apisix
Docker installs initialize Swarm and create the attachable overlay network sdnet.

Install reads and persists exported environment variables:
  ISRVD_MODE          server (default), center, or agent
  ISRVD_CENTER_URL    Required for agent
  ISRVD_NODE_TOKEN    Required for agent
Docker agents do not publish 8080. Updates preserve the installed role.
Docker centers publish the TCP/UDP range from forward.ports in $DOCKER_DATA_DIR/conf/isrvd.yml (also set forward.bind: 0.0.0.0).

Config: $CONFIG_FILE (binary), $DOCKER_DATA_DIR/conf/isrvd.yml (Docker)
EOF
}

# ------------------------------------------
# 入口
# ------------------------------------------

main() {
    local command="${1:-}"
    case "$command" in
        --help|-h) print_usage; return ;;
        install|update|download|uninstall) shift ;;
        *) print_usage; return 1 ;;
    esac
    if [ "$command" = uninstall ] && [ "$#" -gt 0 ]; then
        case "$1" in --help|-h) print_usage; return ;; esac
        fail "Uninstall does not accept options"
    fi
    parse_options "$command" "$@"
    [ "$command" = download ] || check_root
    "$command"
}

main "$@"
