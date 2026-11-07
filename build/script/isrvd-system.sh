#!/bin/bash
set -euo pipefail

# ------------------------------------------
# isrvd 系统服务 + 组件容器管理脚本；全容器化请使用 isrvd-docker.sh。
# 用法:
#   isrvd-system.sh install [--components caddy|apisix] [--data PATH] [--cn|--global|--auto]
#   isrvd-system.sh update [--cn|--global|--auto]
#   isrvd-system.sh download [--cn|--global|--auto]
#   isrvd-system.sh uninstall
# iSrvd 源: 默认按 IP country_code 自动选择；可用 --cn/--global 手动指定
# ------------------------------------------

SERVICE_NAME="isrvd"
INSTALL_DIR="/usr/local/isrvd"
CONFIG_FILE="$INSTALL_DIR/config.yml"
BIN_LINK="/usr/local/bin/isrvd"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
ROLE_DIR="/etc/systemd/system/${SERVICE_NAME}.service.d"
ROLE_FILE="$ROLE_DIR/installer.conf"
DATA_DIR="/srv/isrvd"

SERVICE_AREA="auto"
SYSTEM_COMPONENTS=""
COMPONENT_NAMES=()
COMPONENT_CONTAINERS=()
declare -A COMPONENT_MOUNTS=(
    [etcd]="/etcd:/etcd-data"
    [caddy]="/caddy/caddy.json:/etc/caddy/caddy.json:ro /caddy:/data/caddy /caddy/config:/config"
    [apisix]="/apisix/apisix.yaml:/isrvd-config/apisix.yaml:ro /apisix/conf:/usr/local/apisix/conf /apisix/cert:/usr/local/apisix/cert /apisix/logs:/usr/local/apisix/logs"
)
# 旧版挂载后缀（按容器内目标路径索引），用于识别已有部署。
declare -A LEGACY_MOUNT_SUFFIX=(
    [/etc/caddy/caddy.json]="/conf/components/caddy.json"
    [/config]="/caddy-config"
    [/isrvd-config/apisix.yaml]="/conf/components/apisix.yaml"
    [/usr/local/apisix/conf/config.yaml]="/conf/components/apisix.yaml"
)
INSTALLER_LABEL="io.isrvd.installer"
CADDY_IMAGE="caddy:2.8.4"
APISIX_IMAGE="apache/apisix:3.16.0-debian"
ETCD_IMAGE="quay.io/coreos/etcd:v3.5.29"
DEPLOYMENT_ENV=()
DEPLOYMENT_MODE=""
DOWNLOAD_TMP_FILE=""
RELEASE_AREA=""
RELEASE_COUNTRY=""
RELEASE_SOURCE=""
RELEASE_URL=""
ARCH=""
INSTALL_IN_PROGRESS=0
INSTALL_DIR_CREATED=0
INSTALL_SERVICE_CREATED=0
INSTALL_ROLE_DIR_CREATED=0
INSTALL_COMPONENTS_CREATED=()
INSTALL_SYSCONF_CREATED=0

installer_exit() {
    local status="$1"
    trap - EXIT
    if [ "$INSTALL_IN_PROGRESS" -eq 1 ]; then
        rollback_install || echo "[error] Installation rollback was incomplete" >&2
        [ "$status" -ne 0 ] || status=1
    fi
    [ -z "$DOWNLOAD_TMP_FILE" ] || rm -f "$DOWNLOAD_TMP_FILE"
    exit "$status"
}

trap 'installer_exit "$?"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

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
    local command="$1" gateway="" data=""
    shift
    while [ "$#" -gt 0 ]; do
        case "$1" in
            --cn|--global|--auto) SERVICE_AREA="${1#--}" ;;
            --systemd)
                [ "$command" = install ] || fail "--systemd is only valid for install"
                ;;
            --caddy|--apisix)
                [ "$command" = install ] || fail "Gateway shortcuts are only valid for install"
                [ -z "$gateway" ] || fail "Choose only one gateway"
                gateway="${1#--}"
                ;;
            --components|--data)
                [ "$command" = install ] || fail "$1 is only valid for install"
                [ "$#" -ge 2 ] && [ -n "$2" ] || fail "$1 requires a value"
                case "$1" in
                    --components)
                        [ -z "$SYSTEM_COMPONENTS" ] || fail "Specify --components only once"
                        SYSTEM_COMPONENTS="$2" ;;
                    --data)
                        [ -z "$data" ] || fail "Specify --data only once"
                        data="$2" ;;
                esac
                shift
                ;;
            --docker|--tag|--variant) fail "Fully containerized deployment requires isrvd-docker.sh" ;;
            --help|-h) print_usage; exit 0 ;;
            *) fail "Unknown $command option: $1" ;;
        esac
        shift
    done
    [ "$command" = install ] || return 0
    [ -z "$gateway" ] || [ -z "$SYSTEM_COMPONENTS" ] || fail "Use a gateway shortcut or --components, not both"
    SYSTEM_COMPONENTS="${SYSTEM_COMPONENTS:-$gateway}"
    select_components
    set_component_data_dir "${data:-$(default_data_dir)}"
}

# 未指定 --data 且仅旧默认目录 /srv/data 有主配置时沿用它。
default_data_dir() {
    if [ ! -e "$DATA_DIR/conf/isrvd.yml" ] && [ -f /srv/data/conf/isrvd.yml ]; then
        echo "[info] Reusing existing data directory: /srv/data" >&2
        printf '%s' /srv/data
    else
        printf '%s' "$DATA_DIR"
    fi
}

select_components() {
    case "$SYSTEM_COMPONENTS" in
        ""|none) COMPONENT_NAMES=() ;;
        caddy) COMPONENT_NAMES=(caddy) ;;
        apisix) COMPONENT_NAMES=(etcd apisix) ;;
        *) fail "Choose --components caddy or apisix, or omit it for isrvd only; APISIX automatically includes etcd. Standalone etcd and component combinations are not supported" ;;
    esac
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
    else
        RELEASE_URL="https://github.com/rehiy/isrvd/releases/latest/download/isrvd-$ARCH.tar.gz"
        RELEASE_SOURCE="GitHub"
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
    [ -z "$DOWNLOAD_TMP_FILE" ] || rm -f "$DOWNLOAD_TMP_FILE"
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
    local bin_file entry component working_dir="${DATA_DIR//\"/\\\"}"
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
WorkingDirectory="${working_dir//%/%%}"
$(systemd_environment "ISRVD_CONFIG=$CONFIG_FILE")
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
        if [ "${#COMPONENT_NAMES[@]}" -gt 0 ]; then
            printf '[Unit]\nRequires=docker.service\nAfter=docker.service\n' > "$ROLE_FILE"
        else
            : > "$ROLE_FILE"
        fi
        echo '[Service]' >> "$ROLE_FILE"
        if [ "${#COMPONENT_NAMES[@]}" -gt 0 ]; then
            echo 'TimeoutStartSec=240' >> "$ROLE_FILE"
            for component in "${COMPONENT_NAMES[@]}"; do
                printf 'ExecStartPre=%s start isrvd-%s\n' "$(command -v docker)" "$component" >> "$ROLE_FILE"
                printf 'ExecStartPre=%s --noproxy * -fsS --retry 60 --retry-connrefused --retry-delay 1 --retry-max-time 60 --connect-timeout 1 --max-time 2 --output /dev/null %s\n' \
                    "$(command -v curl)" "$(component_url "$component")" >> "$ROLE_FILE"
            done
        fi
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

validate_package() {
    local entries entry binary=0
    entries=$(tar tzf "$DOWNLOAD_TMP_FILE") || fail "Downloaded package is corrupted"
    while IFS= read -r entry; do
        case "$entry" in
            "isrvd-$ARCH") binary=1 ;;
            config.yml|isrvd.sh|isrvd-system.sh|isrvd-docker.sh) ;;
            *) fail "Unsafe or unexpected package entry: $entry" ;;
        esac
    done <<< "$entries"
    [ "$binary" -eq 1 ] || fail "Package is missing isrvd-$ARCH"
    tar tvzf "$DOWNLOAD_TMP_FILE" | awk 'substr($0, 1, 1) != "-" { bad = 1 } END { exit bad }' \
        || fail "Package must contain only regular files"
}

# $1=update 时保留已有 config.yml
files_install() {
    local options=() entry
    validate_package
    [ "${1:-}" != update ] || options+=(--exclude=config.yml)
    mkdir -p "$INSTALL_DIR"
    for entry in "isrvd-$ARCH" config.yml isrvd.sh isrvd-system.sh isrvd-docker.sh; do
        [ "${1:-}" != update ] || [ "$entry" != config.yml ] || continue
        [ ! -L "$INSTALL_DIR/$entry" ] || fail "Refusing to extract over a symbolic link: $entry"
    done
    echo "[info] Extracting to: $INSTALL_DIR"
    tar xzf "$DOWNLOAD_TMP_FILE" --no-same-owner --no-same-permissions "${options[@]+"${options[@]}"}" -C "$INSTALL_DIR" || return 1
    ln -sf "$(get_bin_file)" "$BIN_LINK"
}

print_binary_info() {
    cat <<EOF
  Install: $INSTALL_DIR
  Binary:  $(get_bin_file)
  Config:  $CONFIG_FILE
EOF
}

rollback_install() {
    local index container
    if [ "$INSTALL_SERVICE_CREATED" -eq 1 ]; then
        systemctl stop "$SERVICE_NAME" >/dev/null 2>&1 || true
        systemctl disable "$SERVICE_NAME" >/dev/null 2>&1 || true
        rm -f "$SERVICE_FILE" "$ROLE_FILE"
        if [ "$INSTALL_ROLE_DIR_CREATED" -eq 1 ]; then rmdir "$ROLE_DIR" 2>/dev/null || true; fi
        systemctl daemon-reload >/dev/null 2>&1 || true
    fi
    for ((index=${#INSTALL_COMPONENTS_CREATED[@]}-1; index>=0; index--)); do
        container="${INSTALL_COMPONENTS_CREATED[$index]}"
        if container_exists "$container" && managed_component "${container#isrvd-}"; then
            docker rm -f "$container" >/dev/null 2>&1 || echo "[error] Could not remove $container during rollback" >&2
        fi
    done
    if [ -L "$BIN_LINK" ] && [ "$(readlink "$BIN_LINK")" = "$INSTALL_DIR/isrvd-$ARCH" ]; then
        rm -f "$BIN_LINK"
    fi
    if [ "$INSTALL_DIR_CREATED" -eq 1 ]; then rm -rf -- "$INSTALL_DIR"; fi
    if [ "$INSTALL_SYSCONF_CREATED" -eq 1 ]; then
        rm -f /etc/sysctl.d/70-isrvd-unprivileged-ports.conf
    fi
    echo "[error] Installation rolled back; data preserved: $DATA_DIR" >&2
}

install_binary() {
    [ ! -e "$INSTALL_DIR" ] && [ ! -L "$INSTALL_DIR" ] && [ ! -e "$SERVICE_FILE" ] && [ ! -L "$SERVICE_FILE" ] || fail "Already installed; use update"
    [ ! -e "$BIN_LINK" ] && [ ! -L "$BIN_LINK" ] || fail "Existing command path found; refusing to overwrite $BIN_LINK"
    if [ -L "$ROLE_DIR" ] || [ -e "$ROLE_FILE" ] || [ -L "$ROLE_FILE" ] || [ -e "$ROLE_DIR/role.conf" ] || [ -L "$ROLE_DIR/role.conf" ]; then
        fail "Existing role configuration found; refusing to overwrite it"
    fi
    command -v systemctl >/dev/null 2>&1 || fail "Systemd is required for --systemd"
    if has_docker && container_exists "$SERVICE_NAME"; then
        fail "Docker deployment already installed; uninstall before switching deployments"
    fi
    prepare_component_credentials
    resolve_release
    print_banner "Systemd + component containers Installer"
    if [ "${#COMPONENT_NAMES[@]}" -gt 0 ]; then
        ensure_docker_ready
        preflight_components
    elif has_docker && docker info >/dev/null 2>&1; then
        check_component_backups
    fi
    download_package "$RELEASE_URL"
    validate_package
    mkdir "$INSTALL_DIR"
    INSTALL_DIR_CREATED=1
    INSTALL_IN_PROGRESS=1
    files_install
    install_data_config
    configure_root_directory
    if [ "${#COMPONENT_NAMES[@]}" -gt 0 ]; then
        [[ " ${COMPONENT_NAMES[*]} " = *" apisix "* ]] && prepare_unprivileged_ports
        install_components
        configure_binary_components
    fi
    [ -d "$ROLE_DIR" ] || INSTALL_ROLE_DIR_CREATED=1
    INSTALL_SERVICE_CREATED=1
    service_install
    service_start
    INSTALL_IN_PROGRESS=0
    echo "[info] Installation complete (mode: $DEPLOYMENT_MODE)"
    print_binary_info
    echo "  Manage: systemctl {start|stop|status|edit} $SERVICE_NAME"
    echo "  Logs:   journalctl -u $SERVICE_NAME -f"
}

update_binary() {
    [ -d "$INSTALL_DIR" ] || fail "Not installed; use install first"
    discover_binary_data
    discover_components
    resolve_release
    print_banner Updater
    # 下载与归档检查在停服之前完成，失败不影响正在运行的服务。
    download_package "$RELEASE_URL"
    validate_package
    update_components
    service_stop
    if ! files_install update; then
        service_start
        fail "Failed to extract update"
    fi
    configure_binary_components
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
        else
            sh "$DOWNLOAD_TMP_FILE"
        fi
    fi
    docker info >/dev/null 2>&1 || fail "Docker is not running or current environment cannot access Docker"
}

component_image() {
    case "$1" in
        etcd) printf '%s' "$ETCD_IMAGE" ;;
        caddy) printf '%s' "$CADDY_IMAGE" ;;
        apisix) printf '%s' "$APISIX_IMAGE" ;;
    esac
}

managed_component() {
    [ "$(docker inspect --format "{{index .Config.Labels \"$INSTALLER_LABEL\"}}" "isrvd-$1" 2>/dev/null)" = component ]
}

check_component_backups() {
    local component backup action="${1:-update}"
    for component in etcd caddy apisix; do
        backup="isrvd-$component-update-backup"
        if container_exists "$backup"; then
            managed_component "$component-update-backup" || fail "Container $backup is not owned by this installer"
            [ "$action" = uninstall ] || fail "Interrupted component update detected: $backup; restore it or use isrvd-system.sh uninstall before retrying"
        fi
    done
}

preflight_components() {
    local component
    check_component_backups
    for component in "${COMPONENT_NAMES[@]}"; do
        ! container_exists "isrvd-$component" || fail "Container isrvd-$component already exists"
        docker pull "$(component_image "$component")"
    done
}

# APISIX 官方镜像默认以非 root 用户运行，--network host 下绑定 80/443
# 依赖宿主机放开低端口绑定限制；已放开或无 sysctl 时不改系统配置。
prepare_unprivileged_ports() {
    local conf="/etc/sysctl.d/70-isrvd-unprivileged-ports.conf" current
    command -v sysctl >/dev/null 2>&1 || return 0
    current=$(sysctl -n net.ipv4.ip_unprivileged_port_start 2>/dev/null) || current=""
    [ -n "$current" ] && [ "$current" -gt 0 ] 2>/dev/null || return 0
    echo "[info] APISIX container binds 80/443 as a non-root user; lowering the bind restriction (net.ipv4.ip_unprivileged_port_start: $current -> 0)"
    printf 'net.ipv4.ip_unprivileged_port_start = 0\n' > "$conf" \
        || fail "Cannot write $conf"
    INSTALL_SYSCONF_CREATED=1
    sysctl -p "$conf" >/dev/null 2>&1 \
        || fail "Cannot apply net.ipv4.ip_unprivileged_port_start=0; set it manually and rerun"
}

resolve_data_dir() {
    local data="$1" install_dir
    [[ "$data" = /* && "$data" != *:* && "$data" != *\\* && ! "$data" =~ [[:cntrl:]] && ! "$data" =~ (^|/)\.\.?(/|$) ]] \
        || fail "Data directory must be an absolute directory without dot segments, colons, backslashes or control characters"
    data=$(realpath -m -- "$data") || fail "Cannot resolve data directory (realpath is required)"
    install_dir=$(realpath -m -- "$INSTALL_DIR") || fail "Cannot resolve installation directory"
    [ "$data" != / ] || fail "Data directory must not use the root directory"
    [[ "$data" != "$install_dir" && "$data" != "$install_dir/"* && "$install_dir" != "$data/"* ]] \
        || fail "Data directory must not overlap the installation directory"
    [ ! -e "$data" ] || [ -d "$data" ] || fail "Data path is not a directory"
    DATA_DIR="$data"
}

set_component_data_dir() {
    local data resolved paths
    resolve_data_dir "$1"
    data="$DATA_DIR"
    paths=("$data/"{conf/isrvd.yml,etcd,caddy,caddy/caddy.json,caddy/config,apisix/apisix.yaml,apisix/conf,apisix/cert,apisix/logs})
    resolved=$(realpath -m -- "${paths[@]}") || fail "Cannot resolve component data paths"
    [ "$resolved" = "$(printf '%s\n' "${paths[@]}")" ] || fail "Component data paths must not be redirected by symbolic links"
}

restore_component_data_dir() {
    local container component mount destination suffix mode writable source root common=""
    local mounts=()
    [ "$CONFIG_FILE" = "$INSTALL_DIR/config.yml" ] || common="$DATA_DIR"
    for container in "${COMPONENT_CONTAINERS[@]}"; do
        component="${container#isrvd-}"
        component="${component%-update-backup}"
        [ -n "${COMPONENT_MOUNTS[$component]:-}" ] || fail "Unknown component container: $container"
        read -r -a mounts <<< "${COMPONENT_MOUNTS[$component]}"
        if [ "$component" = apisix ]; then
            source=$(docker inspect --format '{{range .Mounts}}{{if eq .Destination "/usr/local/apisix/conf/config.yaml"}}{{println .Source}}{{end}}{{end}}' "$container") \
                || fail "Cannot read the previous APISIX configuration mount"
            [ -z "$source" ] || mounts=("/apisix/apisix.yaml:/usr/local/apisix/conf/config.yaml:ro")
        fi
        for mount in "${mounts[@]}"; do
            IFS=: read -r suffix destination mode <<< "$mount"
            writable=true
            [ "$mode" != ro ] || writable=false
            source=$(docker inspect --format "{{range .Mounts}}{{if and (eq .Destination \"$destination\") (eq .Type \"bind\") (eq .RW $writable)}}{{println .Source}}{{end}}{{end}}" "$container") \
                || fail "Cannot read component data mount: $container"
            if [[ -n "$source" && "$source" != *"$suffix" && -n "${LEGACY_MOUNT_SUFFIX[$destination]:-}" ]]; then
                suffix="${LEGACY_MOUNT_SUFFIX[$destination]}"
            fi
            [[ -n "$source" && "$source" = *"$suffix" ]] || fail "Unexpected component data mount: $container ($destination)"
            root="${source%"$suffix"}"
            set_component_data_dir "$root"
            [ -z "$common" ] || [ "$common" = "$DATA_DIR" ] || fail "Component data mounts use different directories; refusing to change them"
            common="$DATA_DIR"
        done
    done
}

discover_components() {
    local component backup action="${1:-update}"
    COMPONENT_NAMES=()
    COMPONENT_CONTAINERS=()
    if has_docker; then
        if ! docker info >/dev/null 2>&1; then
            if [ -f "$ROLE_FILE" ] && grep -q '^ExecStartPre=.* start isrvd-' "$ROLE_FILE"; then
                fail "Docker is unavailable; cannot safely discover or update component containers"
            fi
            return 0
        fi
        check_component_backups "$action"
        for component in etcd caddy apisix; do
            backup="isrvd-$component-update-backup"
            if [ "$action" = uninstall ] && container_exists "$backup"; then
                COMPONENT_CONTAINERS+=("$backup")
            fi
            if container_exists "isrvd-$component"; then
                managed_component "$component" || fail "Container isrvd-$component is not owned by this installer"
                COMPONENT_NAMES+=("$component")
                COMPONENT_CONTAINERS+=("isrvd-$component")
            fi
        done
    fi
    restore_component_data_dir
}

validate_apisix_key() {
    local key="$1"
    [[ "$key" =~ ^[A-Za-z0-9_-]{32,128}$ ]] || fail "ISRVD_APISIX_ADMIN_KEY must contain 32-128 letters, digits, underscores or hyphens"
    [[ ! "$key" =~ ^[0-9]+$ ]] || fail "ISRVD_APISIX_ADMIN_KEY must not be numeric-only (APISIX converts numeric environment values)"
}

prepare_component_credentials() {
    if [[ " ${COMPONENT_NAMES[*]} " = *" apisix "* ]]; then
        export APISIX_ADMIN_KEY="${ISRVD_APISIX_ADMIN_KEY:-isrvd_$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]')}"
        validate_apisix_key "$APISIX_ADMIN_KEY"
    fi
}

prepare_component_configs() {
    local component storage_root="${DATA_DIR//\"/\\\"}/caddy"
    for component in "${COMPONENT_NAMES[@]}"; do
        case "$component" in
            etcd) mkdir -p "$DATA_DIR/etcd" ;;
            caddy)
                mkdir -p "$DATA_DIR/caddy/config"
                if [ ! -e "$DATA_DIR/caddy/caddy.json" ]; then
                    cat > "$DATA_DIR/caddy/caddy.json" <<EOF
{
  "admin": {"listen": "127.0.0.1:2019", "config": {"persist": true}},
  "apps": {"http": {"servers": {"srv0": {
    "listen": [":80", ":443"],
    "automatic_https": {"disable": true, "disable_redirects": true},
    "routes": []
  }}}},
  "storage": {"module": "file_system", "root": "$storage_root"}
}
EOF
                fi
                ;;
            apisix)
                [ -n "${APISIX_ADMIN_KEY:-}" ] || fail "APISIX credentials are not prepared"
                mkdir -p "$DATA_DIR/apisix"
                if [ ! -e "$DATA_DIR/apisix/apisix.yaml" ]; then
                    cat > "$DATA_DIR/apisix/apisix.yaml" <<'EOF'
apisix:
  node_listen: 80
  enable_ipv6: false
  ssl:
    enable: true
    listen:
      - port: 443
  enable_control: true
  control:
    ip: 127.0.0.1
    port: 9090
  trusted_addresses:
    - 127.0.0.1/32

deployment:
  admin:
    allow_admin:
      - 127.0.0.1/32
    admin_key:
      - name: admin
        key: ${{APISIX_ADMIN_KEY}}
        role: admin
    admin_listen:
      ip: 127.0.0.1
      port: 9180
  etcd:
    host:
      - http://127.0.0.1:2379
    prefix: /apisix
    timeout: 30
EOF
                fi
                ;;
        esac
    done
}

install_data_config() {
    local target="$DATA_DIR/conf/isrvd.yml"
    set_component_data_dir "$DATA_DIR"
    mkdir -p "$DATA_DIR/conf"
    if [ -e "$target" ]; then
        [ -f "$target" ] && [ ! -L "$target" ] || fail "Expected a regular config file: $target"
        rm -f "$INSTALL_DIR/config.yml"
    else
        mv "$INSTALL_DIR/config.yml" "$target"
    fi
    chmod 600 "$target"
    ln -s "$target" "$INSTALL_DIR/config.yml"
    CONFIG_FILE="$target"
}

discover_binary_data() {
    local legacy="$INSTALL_DIR/config.yml" target root
    if [ -L "$legacy" ]; then
        target=$(realpath -e -- "$legacy") || fail "Cannot resolve installed configuration"
        [[ "$target" = */conf/isrvd.yml ]] || fail "Unexpected installed configuration link"
        set_component_data_dir "${target%/conf/isrvd.yml}"
        CONFIG_FILE="$target"
    elif [ -f "$legacy" ]; then
        root=$(awk '
            /^[^[:space:]#]/ { section = $0 }
            section ~ /^server:/ && /^[[:space:]]+rootDirectory:/ {
                sub(/^[[:space:]]+rootDirectory:[[:space:]]*/, "")
                sub(/[[:space:]]+#[^"\047]*$/, "")
                sub(/[[:space:]]+$/, "")
                if ($0 ~ /^".*"$/) {
                    $0 = substr($0, 2, length($0) - 2)
                    gsub(/\\"/, "\"", $0)
                } else if ($0 ~ /^\047.*\047$/) {
                    $0 = substr($0, 2, length($0) - 2)
                    gsub(/\047\047/, "\047", $0)
                }
                print; exit
            }' "$legacy")
        [ -z "$root" ] || resolve_data_dir "$root"
        CONFIG_FILE="$legacy"
    fi
}

preserve_legacy_config() {
    local target="$DATA_DIR/conf/isrvd.yml"
    [ "$CONFIG_FILE" = "$INSTALL_DIR/config.yml" ] && [ -f "$CONFIG_FILE" ] && [ ! -L "$CONFIG_FILE" ] || return 0
    set_component_data_dir "$DATA_DIR"
    [ ! -e "$target" ] && [ ! -L "$target" ] || fail "Refusing to overwrite preserved configuration: $target"
    mkdir -p "$DATA_DIR/conf"
    cp -p -- "$CONFIG_FILE" "$target"
    chmod 600 "$target"
}

configure_root_directory() {
    local tmp
    [ -f "$CONFIG_FILE" ] && [ ! -L "$CONFIG_FILE" ] || fail "Expected a regular config file: $CONFIG_FILE"
    tmp=$(mktemp "$(dirname "$CONFIG_FILE")/.config.XXXXXX")
    cp -p -- "$CONFIG_FILE" "$tmp" || { rm -f "$tmp"; fail "Cannot copy config file attributes: $CONFIG_FILE"; }
    if ! ROOT_DIRECTORY="$DATA_DIR" awk '
        function root_config() {
            value = ENVIRON["ROOT_DIRECTORY"]
            gsub(/"/, "\\\"", value)
            print "  rootDirectory: \"" value "\""
        }
        /^[^[:space:]#]/ { section = $0 }
        section ~ /^server:/ && /^[[:space:]]+rootDirectory:/ { if (!done) root_config(); done = 1; next }
        { print }
        END { if (!done) exit 1 }
    ' "$CONFIG_FILE" > "$tmp"; then
        rm -f "$tmp"
        fail "Expected server.rootDirectory in $CONFIG_FILE"
    fi
    mv "$tmp" "$CONFIG_FILE"
}

configure_binary_components() {
    local tmp apisix=0 caddy=0
    [[ " ${COMPONENT_NAMES[*]} " != *" apisix "* ]] || apisix=1
    [[ " ${COMPONENT_NAMES[*]} " != *" caddy "* ]] || caddy=1
    [ "$apisix$caddy" != 00 ] || return 0
    [ -f "$CONFIG_FILE" ] && [ ! -L "$CONFIG_FILE" ] || fail "Expected a regular config file: $CONFIG_FILE"
    if [ "$apisix" -eq 1 ]; then validate_apisix_key "${APISIX_ADMIN_KEY:-}"; fi
    tmp=$(mktemp "$(dirname "$CONFIG_FILE")/.config.XXXXXX")
    if ! awk -v apisix="$apisix" -v caddy="$caddy" '
        function apisix_config() {
            print "  adminUrl: http://127.0.0.1:9180/apisix/admin"
            print "  adminKey: \"" ENVIRON["APISIX_ADMIN_KEY"] "\""
        }
        function caddy_config() { print "  adminUrl: http://127.0.0.1:2019" }
        /^[^[:space:]#]/ { section = $0 }
        apisix && /^apisix:/ && !/^apisix:[[:space:]]*(#.*)?$/ { invalid = 1; exit 1 }
        caddy && /^caddy:/ && !/^caddy:[[:space:]]*(#.*)?$/ { invalid = 1; exit 1 }
        apisix && section ~ /^apisix:/ && /^[[:space:]]+#?[[:space:]]*admin(Url|Key):/ { next }
        caddy && section ~ /^caddy:/ && /^[[:space:]]+#?[[:space:]]*adminUrl:/ { next }
        { print }
        apisix && /^apisix:/ { seen_apisix = 1; apisix_config() }
        caddy && /^caddy:/ { seen_caddy = 1; caddy_config() }
        END {
            if (invalid) exit 1
            if (apisix && !seen_apisix) { print "\napisix:"; apisix_config() }
            if (caddy && !seen_caddy) { print "\ncaddy:"; caddy_config() }
        }
    ' "$CONFIG_FILE" > "$tmp"; then
        rm -f "$tmp"
        fail "Expected block-style gateway sections in $CONFIG_FILE"
    fi
    chmod 600 "$tmp"
    mv "$tmp" "$CONFIG_FILE"
}

component_url() {
    case "$1" in
        etcd) printf '%s' 'http://127.0.0.1:2379/health' ;;
        caddy) printf '%s' 'http://127.0.0.1:2019/config/' ;;
        apisix) printf '%s' 'http://127.0.0.1:9090/v1/schema' ;;
    esac
}

component_admin_ready() {
    [ "$1" = apisix ] || return 0
    [ -n "${APISIX_ADMIN_KEY:-}" ] || return 1
    printf 'header = "X-API-KEY: %s"\n' "$APISIX_ADMIN_KEY" | \
        curl --config - --noproxy '*' -fsS --connect-timeout 1 --max-time 2 \
            http://127.0.0.1:9180/apisix/admin/routes >/dev/null 2>&1
}

component_ready() {
    local component="$1" attempt
    for ((attempt=0; attempt<60; attempt++)); do
        if [ "$(docker inspect --format '{{.State.Running}}' "isrvd-$component")" = true ] && \
            curl --noproxy '*' -fsS --connect-timeout 1 --max-time 2 "$(component_url "$component")" >/dev/null 2>&1 && \
            component_admin_ready "$component"; then
            return 0
        fi
        sleep 1
    done
    echo "[error] Component isrvd-$component is not ready" >&2
    return 1
}

prepare_apisix_data() {
    local image="$1" previous="isrvd-apisix-update-backup" source dir
    mkdir -p "$DATA_DIR/apisix/"{conf,cert,logs}
    if container_exists "$previous"; then
        source=$(docker inspect --format '{{range .Mounts}}{{if eq .Destination "/usr/local/apisix/conf/config.yaml"}}{{println .Source}}{{end}}{{end}}' "$previous") || return 1
        if [ -n "$source" ]; then
            for dir in conf logs; do
                docker cp "$previous:/usr/local/apisix/$dir/." "$DATA_DIR/apisix/$dir" || return 1
            done
            # 官方镜像没有 /usr/local/apisix/cert，旧版部署也没有挂载它，缺失属正常情况。
            docker cp "$previous:/usr/local/apisix/cert/." "$DATA_DIR/apisix/cert" 2>/dev/null || true
        fi
    fi
    docker run --rm --user root --network none --security-opt no-new-privileges:true \
        -v "$DATA_DIR/apisix:/isrvd-state" --entrypoint /bin/sh "$image" -ec '
        rm -f /isrvd-state/conf/config.yaml
        for dir in conf cert; do
            if [ -d "/usr/local/apisix/$dir" ]; then
                cp -an "/usr/local/apisix/$dir/." "/isrvd-state/$dir/"
            fi
        done
        rm -f /isrvd-state/conf/config.yaml
        ln -s /isrvd-config/apisix.yaml /isrvd-state/conf/config.yaml
        chown -hR apisix:apisix /isrvd-state/conf /isrvd-state/cert /isrvd-state/logs
    '
}

run_component_container() {
    local component="$1" image="$2" mount
    local options=(--name "isrvd-$component" --restart unless-stopped --network host
        --label "$INSTALLER_LABEL=component" --security-opt no-new-privileges:true)
    local mounts=() args=()
    read -r -a mounts <<< "${COMPONENT_MOUNTS[$component]:-}"
    for mount in "${mounts[@]}"; do options+=(-v "$DATA_DIR$mount"); done
    case "$component" in
        etcd)
            args=(/usr/local/bin/etcd --name isrvd-etcd --data-dir /etcd-data
                --listen-client-urls http://127.0.0.1:2379 --advertise-client-urls http://127.0.0.1:2379
                --listen-peer-urls http://127.0.0.1:2380 --initial-advertise-peer-urls http://127.0.0.1:2380
                --initial-cluster isrvd-etcd=http://127.0.0.1:2380)
            ;;
        caddy)
            [ "$DATA_DIR/caddy" = /data/caddy ] || options+=(-v "$DATA_DIR/caddy:$DATA_DIR/caddy")
            args=(caddy run --config /etc/caddy/caddy.json --resume)
            ;;
        apisix)
            prepare_apisix_data "$image" || return 1
            options+=(-e APISIX_ADMIN_KEY)
            ;;
        *) fail "Unknown component: $component" ;;
    esac
    docker run -d "${options[@]}" "$image" "${args[@]}"
}

install_components() {
    local component
    prepare_component_configs
    for component in "${COMPONENT_NAMES[@]}"; do
        INSTALL_COMPONENTS_CREATED+=("isrvd-$component")
        if ! run_component_container "$component" "$(component_image "$component")" || ! component_ready "$component"; then
            fail "Component installation failed; component data is preserved in $DATA_DIR"
        fi
    done
}

# 旧布局迁移：停容器前只复制，新容器就绪后才清理（失败回滚的旧容器仍需旧文件）。
migrate_legacy_component_data() {
    local component="$1" legacy="$DATA_DIR/conf/components" file
    case "$component" in
        caddy)
            mkdir -p "$DATA_DIR/caddy/config" || return 1
            [ ! -d "$DATA_DIR/caddy-config" ] || cp -an "$DATA_DIR/caddy-config/." "$DATA_DIR/caddy/config/" || return 1
            file=caddy.json
            ;;
        apisix)
            mkdir -p "$DATA_DIR/apisix" || return 1
            file=apisix.yaml
            ;;
        *) return 0 ;;
    esac
    if [ -f "$legacy/$file" ] && [ ! -L "$legacy/$file" ] && [ ! -e "$DATA_DIR/$component/$file" ]; then
        cp -p -- "$legacy/$file" "$DATA_DIR/$component/$file" || return 1
    fi
    [ -f "$DATA_DIR/$component/$file" ] && [ ! -L "$DATA_DIR/$component/$file" ] \
        || { echo "[error] Missing $component startup config: $DATA_DIR/$component/$file" >&2; return 1; }
}

# 新位置文件存在时才删除旧副本；失败只告警。
cleanup_legacy_component_data() {
    local component="$1" legacy="$DATA_DIR/conf/components" file
    case "$component" in
        caddy)
            file=caddy.json
            [ ! -d "$DATA_DIR/caddy/config" ] || rm -rf -- "$DATA_DIR/caddy-config" || echo "[warn] Could not remove $DATA_DIR/caddy-config" >&2
            ;;
        apisix) file=apisix.yaml ;;
        *) return 0 ;;
    esac
    if [ -f "$DATA_DIR/$component/$file" ]; then
        rm -f -- "$legacy/$file" || echo "[warn] Could not remove $legacy/$file" >&2
    fi
    rmdir "$legacy" 2>/dev/null || true
}

update_components() {
    local component image entry backup environment
    check_component_backups
    # 先统一迁移并校验全部组件，任何一个失败都在停止容器之前中止。
    for component in "${COMPONENT_NAMES[@]}"; do
        migrate_legacy_component_data "$component" || fail "Cannot migrate legacy $component data into $DATA_DIR"
    done
    for component in "${COMPONENT_NAMES[@]}"; do
        image=$(docker inspect --format '{{.Config.Image}}' "isrvd-$component")
        backup="isrvd-$component-update-backup"
        ! container_exists "$backup" || fail "Container $backup already exists; resolve it before updating"
        if [ "$component" = apisix ]; then
            environment=$(docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "isrvd-$component") \
                || fail "Cannot read the previous APISIX container environment"
            unset APISIX_ADMIN_KEY
            while IFS= read -r entry; do
                case "$entry" in APISIX_ADMIN_KEY=*) export APISIX_ADMIN_KEY="${entry#*=}" ;; esac
            done <<< "$environment"
            validate_apisix_key "${APISIX_ADMIN_KEY:-}"
        fi
        docker pull "$image"
        docker rename "isrvd-$component" "$backup"
        if ! docker stop "$backup"; then
            docker rename "$backup" "isrvd-$component"
            fail "Cannot stop the previous $component container"
        fi
        if run_component_container "$component" "$image" && component_ready "$component"; then
            docker rm "$backup"
            cleanup_legacy_component_data "$component"
        else
            docker rm -f "isrvd-$component" >/dev/null 2>&1 || true
            docker rename "$backup" "isrvd-$component"
            docker start "isrvd-$component"
            fail "Component update failed; restored the previous $component container"
        fi
    done
}

# ------------------------------------------
# 命令
# ------------------------------------------

install() {
    prepare_deployment_env
    install_binary
}

update() {
    if has_docker && container_exists "$SERVICE_NAME"; then
        fail "Docker deployment must be updated with isrvd-docker.sh"
    fi
    update_binary
}

uninstall() {
    local container index removed=0

    echo "[info] Uninstalling isrvd"

    if has_docker && container_exists "$SERVICE_NAME"; then
        fail "Docker deployment must be uninstalled with isrvd-docker.sh"
    fi
    discover_binary_data
    discover_components uninstall
    if [ -f "$SERVICE_FILE" ]; then service_stop; fi
    preserve_legacy_config
    if has_docker; then
        for ((index=${#COMPONENT_CONTAINERS[@]}-1; index>=0; index--)); do
            container="${COMPONENT_CONTAINERS[$index]}"
            echo "[info] Removing Docker container: $container"
            docker rm -f "$container"
            removed=1
        done
        [ "$removed" -eq 0 ] || echo "[info] Docker data preserved: $DATA_DIR"
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
        echo "[info] Uninstallation complete; configuration and data preserved: $DATA_DIR"
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
Usage: $0 install [--components caddy|apisix] [--data PATH] [--cn|--global|--auto]
       $0 update [--cn|--global|--auto]
       $0 download [--cn|--global|--auto]
       $0 uninstall

System service + component containers only. Fully containerized deployments use isrvd-docker.sh.
Commands:
  install     Install isrvd as a systemd service and optional gateway containers
  update      Update the binary and owned components; preserve configuration, role and data mounts
  download    Download the latest binary package to the current directory
  uninstall   Remove the system service and owned components; preserve their data directory

Sources: --auto (default), --cn (CNB), --global (GitHub).
No gateway is selected by default; omit --components to install only isrvd, without Docker.
--components optionally selects caddy or apisix (APISIX + etcd).
--components none is still accepted for compatibility with existing commands.
Standalone etcd and component combinations are not supported for new installations.
--data PATH selects the root of all installer-managed persistent files for install only (default: /srv/isrvd), with or without a gateway.
Main config: DATA/conf/isrvd.yml (0600); DATA is also server.rootDirectory and the service working directory.
Layout under DATA: conf/isrvd.yml, caddy/{caddy.json,config}, etcd/, apisix/{apisix.yaml,conf,cert,logs}.
Updates migrate the legacy layout (conf/components, caddy-config) and remove the old files after the new container is ready.
Without --data, a legacy /srv/data with conf/isrvd.yml is reused if /srv/isrvd has none.
The directory must be absolute, non-root and outside $INSTALL_DIR; managed subpaths must not redirect via symlinks.
Docker daemon storage, journal, explicit external paths and etcd key namespaces are not rewritten.
Components run in separate host-network containers; APISIX automatically installs its etcd dependency.
The APISIX image runs as a non-root user; when net.ipv4.ip_unprivileged_port_start is above 0, install writes /etc/sysctl.d/70-isrvd-unprivileged-ports.conf (=0) so 80/443 can bind, and removes the file on rollback.
Component admin APIs and etcd listen only on 127.0.0.1; gateways use ports 80/443.
--caddy/--apisix select the corresponding gateway option; --systemd is accepted but not required.
This script never updates or removes the fully containerized isrvd deployment.

Install reads and persists exported environment variables:
  ISRVD_MODE          server (default), center, or agent
  ISRVD_CENTER_URL    Required for agent
  ISRVD_NODE_TOKEN    Required for agent
  ISRVD_APISIX_ADMIN_KEY  Optional non-numeric-only 32-128 character key for systemd + APISIX; generated if unset
                         Passed to the APISIX container and saved as apisix.adminKey in root-only config.yml (0600)
Installation failures roll back new service files and containers, preserving component data.
Interrupted component updates block further updates; restore backups or use uninstall to remove owned backups.
Updates preserve the installed component key, role and actual bind mounts; inconsistent data paths abort before stopping components.
Changing --data in an install command does not migrate an existing deployment. Centers use portmap.range and portmap.bind in $CONFIG_FILE.

Config: $DATA_DIR/conf/isrvd.yml; data: $DATA_DIR.
$INSTALL_DIR/config.yml is a compatibility link for new installations.
Uninstall preserves configuration and data; legacy local configuration is copied to DATA/conf/isrvd.yml first.
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
