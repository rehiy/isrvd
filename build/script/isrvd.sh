#!/bin/bash
set -euo pipefail

# ------------------------------------------
# isrvd 服务管理脚本
# 用法:
#   isrvd.sh install [--docker|--caddy|--apisix] [--cn|--global|--auto]
#   isrvd.sh update [--cn|--global|--auto]
#   isrvd.sh download [--cn|--global|--auto]
#   isrvd.sh uninstall
# iSrvd 源: 默认按 IP country_code 自动选择；可用 --cn/--global 手动指定
# ------------------------------------------

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
DOCKER_VARIANT=""
DOCKER_IMAGE=""
DEPLOYMENT_ENV=()
DEPLOYMENT_MODE=""
DOWNLOAD_TMP_FILE=""
RELEASE_AREA=""
RELEASE_COUNTRY=""
RELEASE_SOURCE=""
RELEASE_URL=""
ARCH=""

trap '[ -z "$DOWNLOAD_TMP_FILE" ] || rm -f "$DOWNLOAD_TMP_FILE"' EXIT

# ------------------------------------------
# 通用
# ------------------------------------------

fail() {
    echo "[error] $*" >&2
    exit 1
}

container_exists() {
    docker container inspect "$1" >/dev/null 2>&1
}

has_docker() {
    command -v docker >/dev/null 2>&1
}

init_arch() {
    local machine
    case "$(uname -m)" in
        x86_64|amd64) machine=amd64 ;;
        aarch64|arm64) machine=arm64 ;;
        armv7l|armhf) machine=arm ;;
        *) fail "Unsupported architecture: $(uname -m)" ;;
    esac
    ARCH="$(uname -s | tr '[:upper:]' '[:lower:]')-$machine"
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

# 解析发布源（国内 CNB / 全球 GitHub 与 Docker Hub），同一次运行只探测一次。
resolve_release() {
    if [ -z "$RELEASE_AREA" ]; then
        RELEASE_AREA="$SERVICE_AREA"
        if [ "$RELEASE_AREA" = auto ]; then
            RELEASE_COUNTRY=$(curl -fsSL --connect-timeout 2 --max-time 5 -A "isrvd-installer" \
                "https://ipip.rehi.org/country_code" 2>/dev/null | tr -d '[:space:]' | tr '[:lower:]' '[:upper:]') || RELEASE_COUNTRY=""
            [[ "$RELEASE_COUNTRY" =~ ^[A-Z]{2}$ ]] || RELEASE_COUNTRY=""
            RELEASE_AREA=global
            [ "$RELEASE_COUNTRY" != CN ] || RELEASE_AREA=cn
        fi
    fi

    if [ "$RELEASE_AREA" = cn ]; then
        RELEASE_URL="https://cnb.cool/rehiy/isrvd/-/releases/latest/download/isrvd-$ARCH.tar.gz"
        RELEASE_SOURCE="CNB"
        DOCKER_IMAGE="docker.cnb.cool/rehiy/isrvd"
    else
        RELEASE_URL="https://github.com/rehiy/isrvd/releases/latest/download/isrvd-$ARCH.tar.gz"
        RELEASE_SOURCE="GitHub"
        DOCKER_IMAGE="rehiy/isrvd"
    fi
}

print_banner() {
    local area="$RELEASE_AREA"
    [ -z "$RELEASE_COUNTRY" ] || area="$area (country: $RELEASE_COUNTRY)"
    echo "=========================================="
    echo "  isrvd $1"
    echo "  Arch:    $ARCH"
    echo "  Area:    $area"
    echo "  Source:  $RELEASE_SOURCE"
    echo "=========================================="
}

download_package() {
    DOWNLOAD_TMP_FILE=$(mktemp)
    echo "[info] Downloading: $1"
    curl -fL --retry 3 --retry-delay 2 --connect-timeout 10 --progress-bar -o "$DOWNLOAD_TMP_FILE" "$1"
}

# ------------------------------------------
# 二进制部署（systemd）
# ------------------------------------------

# 安装时读取当前进程环境；更新时沿用已有配置。
prepare_deployment_env() {
    local mode="${ISRVD_MODE:-server}"
    mode="${mode,,}"
    case "$mode" in
        server|center) ;;
        agent) [ -n "${ISRVD_CENTER_URL:-}" ] && [ -n "${ISRVD_NODE_TOKEN:-}" ] \
                   || fail "Agent mode requires ISRVD_CENTER_URL and ISRVD_NODE_TOKEN" ;;
        *) fail "ISRVD_MODE must be server, center, or agent" ;;
    esac
    DEPLOYMENT_MODE="$mode"
    DEPLOYMENT_ENV=("ISRVD_MODE=$mode")
    if [ "$mode" = agent ]; then
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

get_bin_file() {
    local bin="$INSTALL_DIR/isrvd-$ARCH"
    [ -x "$bin" ] || fail "No executable found: $bin"
    printf '%s' "$bin"
}

service_install() {
    local bin_file entry
    bin_file=$(get_bin_file)

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
NoNewPrivileges=true
LimitNOFILE=65536
StandardOutput=journal
StandardError=journal
SyslogIdentifier=isrvd

[Install]
WantedBy=multi-user.target
EOF

    mkdir -p "$ROLE_DIR"
    (
        umask 077
        echo '[Service]' > "$ROLE_FILE"
        for entry in "${DEPLOYMENT_ENV[@]}"; do
            systemd_environment "$entry" >> "$ROLE_FILE"
        done
    )
    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME"
}

service_stop() {
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        echo "[info] Stopping ${SERVICE_NAME} service..."
        systemctl stop "$SERVICE_NAME"
    fi
}

service_start() {
    echo "[info] Starting ${SERVICE_NAME} service..."
    systemctl start "$SERVICE_NAME"
}

# $1=update 时保留已有 config.yml
files_install() {
    local options=()
    [ "${1:-}" != update ] || options+=(--exclude=config.yml)
    mkdir -p "$INSTALL_DIR"
    echo "[info] Extracting to: $INSTALL_DIR"
    tar xzf "$DOWNLOAD_TMP_FILE" "${options[@]+"${options[@]}"}" -C "$INSTALL_DIR" || return 1
    ln -sf "$(get_bin_file)" "$BIN_LINK"
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
    resolve_release
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

update_binary() {
    [ -d "$INSTALL_DIR" ] || fail "Not installed; use install first"
    resolve_release
    print_banner Updater
    # 下载与归档检查在停服之前完成，失败不影响正在运行的服务。
    download_package "$RELEASE_URL"
    tar tzf "$DOWNLOAD_TMP_FILE" >/dev/null 2>&1 || fail "Downloaded package is corrupted"
    service_stop
    if ! files_install update; then
        service_start
        fail "Failed to extract update"
    fi
    service_start
    echo "[info] Update complete (configuration preserved)"
    print_binary_info
}

# ------------------------------------------
# Docker 部署
# ------------------------------------------

ensure_docker_ready() {
    if ! has_docker; then
        resolve_release
        echo "[info] Installing Docker..."
        download_package https://get.docker.com
        if [ "$RELEASE_AREA" = cn ]; then
            sh "$DOWNLOAD_TMP_FILE" --mirror Aliyun
            mkdir -p /etc/docker
            echo '{"registry-mirrors":["https://docker.1ms.run"]}' > /etc/docker/daemon.json
            systemctl restart docker
        else
            sh "$DOWNLOAD_TMP_FILE"
        fi
    fi
    docker info >/dev/null 2>&1 || fail "Docker is not running or current environment cannot access Docker"
}

ensure_swarm_and_network() {
    local state addr driver attachable

    state=$(docker info --format '{{.Swarm.LocalNodeState}}')
    case "$state" in
        active) ;;
        inactive)
            addr=$(ip route get 1.1.1.1 2>/dev/null | awk '{ for (i = 1; i <= NF; i++) if ($i == "src") { print $(i + 1); exit } }') || addr=""
            echo "[info] Initializing Docker Swarm..."
            docker swarm init ${addr:+--advertise-addr "$addr"}
            ;;
        *) fail "Docker Swarm state is $state; cannot initialize automatically" ;;
    esac

    if docker network inspect "$DOCKER_NETWORK" >/dev/null 2>&1; then
        read -r driver attachable <<< "$(docker network inspect --format '{{.Driver}} {{.Attachable}}' "$DOCKER_NETWORK")"
        [ "$driver" = overlay ] && [ "$attachable" = true ] \
            || fail "Docker network $DOCKER_NETWORK must be an attachable overlay network"
    else
        echo "[info] Creating attachable overlay network: $DOCKER_NETWORK"
        docker network create --driver=overlay --attachable "$DOCKER_NETWORK"
    fi
}

# 读取配置文件 portmap.range 给出的监听端口范围；未配置返回空，格式无效时终止。
portmap_range() {
    local file="$DOCKER_DATA_DIR/conf/isrvd.yml" value
    [ -f "$file" ] || return 0
    value=$(awk '
        /^[^[:space:]#]/ { in_portmap = ($0 ~ /^portmap:[[:space:]]*(#.*)?$/); next }
        in_portmap && /^[[:space:]]+range:/ {
            sub(/^[[:space:]]+range:[[:space:]]*/, ""); sub(/[[:space:]]*#.*$/, ""); gsub(/["'"'"']/, ""); print; exit
        }' "$file")
    [ -z "$value" ] || [[ "$value" =~ ^[0-9]+(-[0-9]+)?$ ]] || fail "portmap.range in $file must look like 20000-20999 or 20022"
    printf '%s' "$value"
}

run_isrvd_container() {
    local variant="$1" image="$2" mode=server entry range
    local ports=() environment=()

    for entry in "${DEPLOYMENT_ENV[@]}"; do
        environment+=(-e "$entry")
        case "$entry" in ISRVD_MODE=*) mode="${entry#*=}" ;; esac
    done
    [ "$mode" = agent ] || ports+=(-p 8080:8080)
    if [ "$mode" = center ]; then
        # 端口转发监听范围需发布到宿主机（TCP 与 UDP）；容器内须同时设置 portmap.bind: 0.0.0.0
        range=$(portmap_range)
        [ -z "$range" ] || ports+=(-p "$range:$range" -p "$range:$range/udp")
    fi
    case "$variant" in
        caddy)  ports+=(-p 80:80 -p 443:443) ;;
        apisix) ports+=(-p 80:9080 -p 443:9443) ;;
    esac

    docker run -d \
        --name "$SERVICE_NAME" \
        --restart unless-stopped \
        --network "$DOCKER_NETWORK" \
        "${ports[@]+"${ports[@]}"}" \
        -v "$DOCKER_DATA_DIR:/data" \
        -v /var/run/docker.sock:/var/run/docker.sock \
        "${environment[@]+"${environment[@]}"}" \
        "$image"
}

container_started() {
    sleep 3
    [ "$(docker inspect --format '{{.State.Running}}' "$SERVICE_NAME")" = true ] && return 0
    docker logs "$SERVICE_NAME" >&2 || true
    return 1
}

install_docker_deployment() {
    [ ! -f "$SERVICE_FILE" ] || fail "Binary service already installed; uninstall before switching deployments"
    ensure_docker_ready
    ! container_exists "$SERVICE_NAME" || fail "Container already exists; use update"
    ensure_swarm_and_network
    resolve_release
    DOCKER_IMAGE="$DOCKER_IMAGE:$DOCKER_VARIANT"
    print_banner "Docker Installer"
    echo "  Image:   $DOCKER_IMAGE"
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
        echo "  Portmap:   To enable port forwarding, set portmap.range and portmap.bind: 0.0.0.0 in the config, then run update"
    fi
}

update_docker_deployment() {
    local old_image variant entry
    local backup="$SERVICE_NAME-update-backup"
    ! container_exists "$backup" || fail "Container $backup already exists; resolve it before updating"

    while IFS= read -r entry; do
        case "$entry" in
            ISRVD_MODE=*|ISRVD_CENTER_URL=*|ISRVD_NODE_TOKEN=*) DEPLOYMENT_ENV+=("$entry") ;;
        esac
    done < <(docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "$SERVICE_NAME")

    old_image=$(docker inspect --format '{{.Config.Image}}' "$SERVICE_NAME")
    case "${old_image##*:}" in
        slim|latest) variant=slim ;;
        caddy|apisix) variant="${old_image##*:}" ;;
        *) fail "Cannot determine isrvd image variant from: $old_image" ;;
    esac

    ensure_swarm_and_network
    resolve_release
    DOCKER_IMAGE="$DOCKER_IMAGE:$variant"
    print_banner "Docker Updater"
    echo "  Image:   $DOCKER_IMAGE"
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
# 命令
# ------------------------------------------

install() {
    prepare_deployment_env
    if [ -n "$DOCKER_VARIANT" ]; then
        install_docker_deployment
    else
        install_binary
    fi
}

update() {
    if has_docker && container_exists "$SERVICE_NAME"; then
        update_docker_deployment
    else
        update_binary
    fi
}

uninstall() {
    local container removed=0

    echo "[info] Uninstalling isrvd"

    if has_docker && docker info >/dev/null 2>&1; then
        for container in "$SERVICE_NAME" isrvd-caddy isrvd-apisix isrvd-apisix-etcd; do
            if container_exists "$container"; then
                echo "[info] Removing Docker container: $container"
                docker rm -f "$container"
                removed=1
            fi
        done
        [ "$removed" -eq 0 ] || echo "[info] Docker data preserved: $DOCKER_DATA_DIR"
    fi

    if [ -f "$SERVICE_FILE" ] || [ -d "$INSTALL_DIR" ]; then
        service_stop
        if [ -f "$SERVICE_FILE" ]; then
            systemctl disable "$SERVICE_NAME" 2>/dev/null || true
            rm -f "$SERVICE_FILE" "$ROLE_FILE"
            rmdir "$ROLE_DIR" 2>/dev/null || true
            systemctl daemon-reload
        fi
        [ ! -L "$BIN_LINK" ] || rm -f "$BIN_LINK"
        rm -rf "$INSTALL_DIR"
        removed=1
    fi

    if [ "$removed" -eq 0 ]; then
        echo "[info] isrvd is not installed"
    else
        echo "[info] Uninstallation complete"
    fi
}

download() {
    local output="isrvd-$ARCH.tar.gz"
    resolve_release
    print_banner Downloader
    download_package "$RELEASE_URL"
    mv "$DOWNLOAD_TMP_FILE" "$output"
    DOWNLOAD_TMP_FILE=""
    chmod 644 "$output"
    echo "[info] Downloaded: $output"
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
Docker centers publish the TCP/UDP range from portmap.range in $DOCKER_DATA_DIR/conf/isrvd.yml (also set portmap.bind: 0.0.0.0).

Config: $CONFIG_FILE (binary), $DOCKER_DATA_DIR/conf/isrvd.yml (Docker)
EOF
}

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
    [ "$command" = download ] || [ "$EUID" -eq 0 ] || fail "Please run as root"
    [ "$command" = uninstall ] || init_arch
    "$command"
}

main "$@"
