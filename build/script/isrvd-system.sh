#!/bin/bash
set -euo pipefail

# ------------------------------------------
# isrvd 系统服务 + 组件容器管理脚本；全容器化请使用 isrvd-docker.sh。
# 用法:
#   isrvd-system.sh install [--components caddy|apisix|etcd] [--data PATH] [--cn|--global|--auto]
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
DOCKER_DATA_DIR="/srv/data"

SERVICE_AREA="auto"
SYSTEM_COMPONENTS=""
COMPONENT_NAMES=()
COMPONENT_CONTAINERS=()
COMPONENT_CONFIG_DIR="$DOCKER_DATA_DIR/conf/components"
declare -A COMPONENT_MOUNTS=(
    [etcd]="/etcd:/etcd-data"
    [caddy]="/conf/components/caddy.json:/etc/caddy/caddy.json:ro /caddy:/data/caddy /caddy-config:/config"
    [apisix]="/conf/components/apisix.yaml:/usr/local/apisix/conf/config.yaml:ro"
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
    SYSTEM_COMPONENTS="${SYSTEM_COMPONENTS:-${gateway:-none}}"
    select_components
    if [ "${#COMPONENT_NAMES[@]}" -gt 0 ]; then
        set_component_data_dir "${data:-$DOCKER_DATA_DIR}"
    else
        [ -z "$data" ] || fail "--data requires at least one component"
    fi
}

select_components() {
    local component caddy=0 apisix=0 etcd=0
    local requested=()
    IFS=, read -r -a requested <<< "$SYSTEM_COMPONENTS"
    [[ "$SYSTEM_COMPONENTS" != ,* && "$SYSTEM_COMPONENTS" != *, && "$SYSTEM_COMPONENTS" != *,,* ]] || fail "Invalid component list"
    for component in "${requested[@]}"; do
        case "$component" in
            none) [ "$SYSTEM_COMPONENTS" = none ] || fail "none cannot be combined with components" ;;
            caddy) caddy=1 ;;
            apisix) apisix=1; etcd=1 ;;
            etcd) etcd=1 ;;
            *) fail "Unknown component: $component" ;;
        esac
    done
    [ "$caddy$apisix" != 11 ] || fail "Caddy and APISIX both use ports 80/443; choose one gateway"
    COMPONENT_NAMES=()
    [ "$etcd" -eq 0 ] || COMPONENT_NAMES+=(etcd)
    [ "$caddy" -eq 0 ] || COMPONENT_NAMES+=(caddy)
    [ "$apisix" -eq 0 ] || COMPONENT_NAMES+=(apisix)
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
    local bin_file entry component
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
    echo "[error] Installation rolled back; component data preserved: $DOCKER_DATA_DIR" >&2
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
    if [ "${#COMPONENT_NAMES[@]}" -gt 0 ]; then
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

set_component_data_dir() {
    local data="$1" install_dir resolved paths
    [[ "$data" = /* && "$data" != *:* && "$data" != *\\* && ! "$data" =~ [[:cntrl:]] && ! "$data" =~ (^|/)\.\.?(/|$) ]] \
        || fail "Component data must be an absolute directory without dot segments, colons, backslashes or control characters"
    data=$(realpath -m -- "$data") || fail "Cannot resolve component data directory (realpath is required)"
    install_dir=$(realpath -m -- "$INSTALL_DIR") || fail "Cannot resolve installation directory"
    [ "$data" != / ] || fail "Component data must not use the root directory"
    [[ "$data" != "$install_dir" && "$data" != "$install_dir/"* && "$install_dir" != "$data/"* ]] \
        || fail "Component data must not overlap the installation directory"
    [ ! -e "$data" ] || [ -d "$data" ] || fail "Component data path is not a directory"
    paths=("$data/"{etcd,caddy,caddy-config,conf/components/caddy.json,conf/components/apisix.yaml})
    resolved=$(realpath -m -- "${paths[@]}") || fail "Cannot resolve component data paths"
    [ "$resolved" = "$(printf '%s\n' "${paths[@]}")" ] || fail "Component data paths must not be redirected by symbolic links"
    DOCKER_DATA_DIR="$data"
    COMPONENT_CONFIG_DIR="$data/conf/components"
}

restore_component_data_dir() {
    local container component mount destination suffix mode writable source root common=""
    local mounts=()
    for container in "${COMPONENT_CONTAINERS[@]}"; do
        component="${container#isrvd-}"
        component="${component%-update-backup}"
        [ -n "${COMPONENT_MOUNTS[$component]:-}" ] || fail "Unknown component container: $container"
        read -r -a mounts <<< "${COMPONENT_MOUNTS[$component]}"
        for mount in "${mounts[@]}"; do
            IFS=: read -r suffix destination mode <<< "$mount"
            writable=true
            [ "$mode" != ro ] || writable=false
            source=$(docker inspect --format "{{range .Mounts}}{{if and (eq .Destination \"$destination\") (eq .Type \"bind\") (eq .RW $writable)}}{{println .Source}}{{end}}{{end}}" "$container") \
                || fail "Cannot read component data mount: $container"
            [[ -n "$source" && "$source" = *"$suffix" ]] || fail "Unexpected component data mount: $container ($destination)"
            root="${source%"$suffix"}"
            set_component_data_dir "$root"
            [ -z "$common" ] || [ "$common" = "$DOCKER_DATA_DIR" ] || fail "Component data mounts use different directories; refusing to change them"
            common="$DOCKER_DATA_DIR"
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
    local component
    mkdir -p "$COMPONENT_CONFIG_DIR"
    for component in "${COMPONENT_NAMES[@]}"; do
        case "$component" in
            etcd) mkdir -p "$DOCKER_DATA_DIR/etcd" ;;
            caddy)
                mkdir -p "$DOCKER_DATA_DIR/caddy" "$DOCKER_DATA_DIR/caddy-config"
                if [ ! -e "$COMPONENT_CONFIG_DIR/caddy.json" ]; then
                    cat > "$COMPONENT_CONFIG_DIR/caddy.json" <<'EOF'
{
  "admin": {"listen": "127.0.0.1:2019", "config": {"persist": true}},
  "apps": {"http": {"servers": {"srv0": {
    "listen": [":80", ":443"],
    "automatic_https": {"disable": true, "disable_redirects": true},
    "routes": []
  }}}},
  "storage": {"module": "file_system", "root": "/data/caddy"}
}
EOF
                fi
                ;;
            apisix)
                [ -n "${APISIX_ADMIN_KEY:-}" ] || fail "APISIX credentials are not prepared"
                if [ ! -e "$COMPONENT_CONFIG_DIR/apisix.yaml" ]; then
                    cat > "$COMPONENT_CONFIG_DIR/apisix.yaml" <<'EOF'
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

configure_binary_components() {
    local tmp apisix=0 caddy=0
    [[ " ${COMPONENT_NAMES[*]} " != *" apisix "* ]] || apisix=1
    [[ " ${COMPONENT_NAMES[*]} " != *" caddy "* ]] || caddy=1
    [ "$apisix$caddy" != 00 ] || return 0
    [ -f "$CONFIG_FILE" ] && [ ! -L "$CONFIG_FILE" ] || fail "Expected a regular config file: $CONFIG_FILE"
    if [ "$apisix" -eq 1 ]; then validate_apisix_key "${APISIX_ADMIN_KEY:-}"; fi
    tmp=$(mktemp "$INSTALL_DIR/.config.XXXXXX")
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

run_component_container() {
    local component="$1" image="$2" mount
    local options=(--name "isrvd-$component" --restart unless-stopped --network host
        --label "$INSTALLER_LABEL=component" --security-opt no-new-privileges:true)
    local mounts=() args=()
    read -r -a mounts <<< "${COMPONENT_MOUNTS[$component]:-}"
    for mount in "${mounts[@]}"; do options+=(-v "$DOCKER_DATA_DIR$mount"); done
    case "$component" in
        etcd)
            args=(/usr/local/bin/etcd --name isrvd-etcd --data-dir /etcd-data
                --listen-client-urls http://127.0.0.1:2379 --advertise-client-urls http://127.0.0.1:2379
                --listen-peer-urls http://127.0.0.1:2380 --initial-advertise-peer-urls http://127.0.0.1:2380
                --initial-cluster isrvd-etcd=http://127.0.0.1:2380)
            ;;
        caddy) args=(caddy run --config /etc/caddy/caddy.json --resume) ;;
        apisix) options+=(-e APISIX_ADMIN_KEY) ;;
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
            fail "Component installation failed; component data is preserved in $DOCKER_DATA_DIR"
        fi
    done
}

update_components() {
    local component image entry backup environment
    check_component_backups
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
    discover_components uninstall
    if [ -f "$SERVICE_FILE" ]; then service_stop; fi
    if has_docker; then
        for ((index=${#COMPONENT_CONTAINERS[@]}-1; index>=0; index--)); do
            container="${COMPONENT_CONTAINERS[$index]}"
            echo "[info] Removing Docker container: $container"
            docker rm -f "$container"
            removed=1
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
Usage: $0 install [--components none|caddy|apisix|etcd|caddy,etcd] [--data PATH] [--cn|--global|--auto]
       $0 update [--cn|--global|--auto]
       $0 download [--cn|--global|--auto]
       $0 uninstall

System service + component containers only. Fully containerized deployments use isrvd-docker.sh.
Commands:
  install     Install isrvd as a systemd service and optional component containers
  update      Update the binary and owned components; preserve configuration, role and data mounts
  download    Download the latest binary package to the current directory
  uninstall   Remove the system service and owned components; preserve their data directory

Sources: --auto (default), --cn (CNB), --global (GitHub).
--data PATH selects the component data directory for install only (default: /srv/data).
The directory must be absolute, non-root and outside $INSTALL_DIR; component subpaths must not redirect via symlinks.
Components run in separate host-network containers; APISIX includes etcd.
Component admin APIs and etcd listen only on 127.0.0.1; gateways use ports 80/443.
--caddy/--apisix select component containers; --systemd is accepted but not required.
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

Config: $CONFIG_FILE; component data: $DOCKER_DATA_DIR.
Uninstall deletes $INSTALL_DIR including config.yml; back it up first.
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
