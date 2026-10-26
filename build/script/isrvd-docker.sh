#!/bin/bash
set -euo pipefail

# isrvd 全容器化安装与管理；系统服务部署请使用 isrvd-system.sh。
# 用法: isrvd-docker.sh install [--tag TAG] [--variant slim|caddy|apisix] [--cn|--global|--auto]

SERVICE_NAME="isrvd"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
INSTALL_DIR="/usr/local/isrvd"
DOCKER_NETWORK="sdnet"
DOCKER_DATA_DIR="/srv/data"
INSTALLER_LABEL="io.isrvd.installer"
SERVICE_AREA="auto"
DOCKER_VARIANT=""
DOCKER_TAG=""
DOCKER_IMAGE=""
DEPLOYMENT_ENV=()
DEPLOYMENT_MODE=""
DOWNLOAD_TMP_FILE=""
RELEASE_AREA=""
RELEASE_COUNTRY=""
RELEASE_SOURCE=""
ARCH=""

trap '[ -z "$DOWNLOAD_TMP_FILE" ] || rm -f "$DOWNLOAD_TMP_FILE"' EXIT

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

assert_isrvd_container() {
    local owner image
    owner=$(docker inspect --format "{{index .Config.Labels \"$INSTALLER_LABEL\"}}" "$SERVICE_NAME")
    [ "$owner" != deployment ] || return 0
    image=$(docker inspect --format '{{.Config.Image}}' "$SERVICE_NAME")
    case "$owner:$image" in
        ':rehiy/isrvd:'*|'<no value>:rehiy/isrvd:'*|':docker.cnb.cool/rehiy/isrvd:'*|'<no value>:docker.cnb.cool/rehiy/isrvd:'*) return 0 ;;
        *) fail "Container $SERVICE_NAME is not owned by this installer" ;;
    esac
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
    local command="$1" gateway=""
    shift
    while [ "$#" -gt 0 ]; do
        case "$1" in
            --cn|--global|--auto) SERVICE_AREA="${1#--}" ;;
            --docker)
                [ "$command" = install ] || fail "--docker is only valid for install"
                ;;
            --caddy|--apisix)
                [ "$command" = install ] || fail "Gateway shortcuts are only valid for install"
                [ -z "$gateway" ] || fail "Choose only one gateway"
                gateway="${1#--}"
                ;;
            --tag|--variant)
                [ "$command" = install ] || [ "$command" = update ] || fail "Image options require install or update"
                [ "$#" -ge 2 ] && [ -n "$2" ] || fail "$1 requires a value"
                if [ "$1" = --tag ]; then
                    [ -z "$DOCKER_TAG" ] || fail "Specify --tag only once"
                    DOCKER_TAG="$2"
                    [[ "$DOCKER_TAG" =~ ^[a-zA-Z0-9_][a-zA-Z0-9_.-]{0,127}$ ]] || fail "Invalid Docker tag"
                else
                    [ -z "$DOCKER_VARIANT" ] || fail "Specify --variant only once"
                    case "$2" in slim|caddy|apisix) DOCKER_VARIANT="$2" ;; *) fail "--variant must be slim, caddy, or apisix" ;; esac
                fi
                shift
                ;;
            --systemd|--components) fail "System service deployment requires isrvd-system.sh" ;;
            --help|-h) print_usage; exit 0 ;;
            *) fail "Unknown $command option: $1" ;;
        esac
        shift
    done
    if [ "$command" != install ]; then
        if [ -n "$DOCKER_TAG" ] && [ -n "$DOCKER_VARIANT" ]; then
            validate_image_layout "$DOCKER_TAG" "$DOCKER_VARIANT"
        fi
        return 0
    fi
    [ -z "$gateway" ] || [ -z "$DOCKER_VARIANT" ] || [ "$gateway" = "$DOCKER_VARIANT" ] || fail "Conflicting image variants"
    DOCKER_VARIANT="${DOCKER_VARIANT:-$gateway}"
    if [ -z "$DOCKER_VARIANT" ]; then
        case "$DOCKER_TAG" in caddy|apisix) DOCKER_VARIANT="$DOCKER_TAG" ;; *) DOCKER_VARIANT=slim ;; esac
    fi
    DOCKER_TAG="${DOCKER_TAG:-$DOCKER_VARIANT}"
    validate_image_layout "$DOCKER_TAG" "$DOCKER_VARIANT"
}

validate_image_layout() {
    local tag="$1" variant="$2" expected=""
    case "$tag" in
        slim|latest) expected=slim ;;
        caddy|apisix) expected="$tag" ;;
    esac
    [ -z "$expected" ] || [ "$variant" = "$expected" ] || fail "Tag $tag requires --variant $expected"
}

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
        RELEASE_SOURCE="CNB"
        DOCKER_IMAGE="docker.cnb.cool/rehiy/isrvd"
    else
        RELEASE_SOURCE="Docker Hub"
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

ensure_docker_ready() {
    if ! has_docker; then
        resolve_release
        echo "[info] Installing Docker..."
        DOWNLOAD_TMP_FILE=$(mktemp)
        curl -fL --retry 3 --retry-delay 2 --connect-timeout 10 --progress-bar \
            -o "$DOWNLOAD_TMP_FILE" https://get.docker.com
        if [ "$RELEASE_AREA" = cn ]; then
            sh "$DOWNLOAD_TMP_FILE" --mirror Aliyun
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
        --label "$INSTALLER_LABEL=deployment" \
        --label "io.isrvd.variant=$variant" \
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

install() {
    [ ! -e "$SERVICE_FILE" ] && [ ! -d "$INSTALL_DIR" ] || fail "System service already installed; use isrvd-system.sh to manage it"
    prepare_deployment_env
    ensure_docker_ready
    ! container_exists "$SERVICE_NAME" || fail "Container already exists; use update"
    ensure_swarm_and_network
    resolve_release
    DOCKER_IMAGE="$DOCKER_IMAGE:$DOCKER_TAG"
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
        echo "  Portmap:   Set portmap.range and portmap.bind: 0.0.0.0 in the config, then run update"
    fi
}

update() {
    local old_image variant entry environment tag
    local backup="$SERVICE_NAME-update-backup"
    has_docker && docker info >/dev/null 2>&1 || fail "Docker is unavailable"
    container_exists "$SERVICE_NAME" || fail "Not installed; use install first (system service deployments use isrvd-system.sh)"
    ! container_exists "$backup" || fail "Container $backup already exists; resolve it before updating"
    assert_isrvd_container
    environment=$(docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "$SERVICE_NAME") \
        || fail "Cannot read the previous isrvd container environment"
    while IFS= read -r entry; do
        case "$entry" in
            ISRVD_MODE=*|ISRVD_CENTER_URL=*|ISRVD_NODE_TOKEN=*) DEPLOYMENT_ENV+=("$entry") ;;
        esac
    done <<< "$environment"
    old_image=$(docker inspect --format '{{.Config.Image}}' "$SERVICE_NAME")
    variant="$DOCKER_VARIANT"
    if [ -z "$variant" ]; then
        case "$DOCKER_TAG" in
            slim|latest) variant=slim ;;
            caddy|apisix) variant="$DOCKER_TAG" ;;
            *) variant=$(docker inspect --format '{{index .Config.Labels "io.isrvd.variant"}}' "$SERVICE_NAME") ;;
        esac
    fi
    case "$variant" in
        slim|caddy|apisix) ;;
        *)
            case "${old_image##*:}" in
                slim|latest) variant=slim ;;
                caddy|apisix) variant="${old_image##*:}" ;;
                *) fail "Unknown image layout; specify --variant slim, caddy, or apisix" ;;
            esac
            ;;
    esac
    tag="${DOCKER_TAG:-${old_image##*:}}"
    validate_image_layout "$tag" "$variant"
    ensure_swarm_and_network
    resolve_release
    DOCKER_IMAGE="$DOCKER_IMAGE:$tag"
    print_banner "Docker Updater"
    echo "  Image:   $DOCKER_IMAGE"
    docker pull "$DOCKER_IMAGE"
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

uninstall() {
    if ! has_docker; then
        echo "[info] Docker deployment is not installed"
        return
    fi
    docker info >/dev/null 2>&1 || fail "Docker is unavailable; refusing an incomplete uninstall"
    if ! container_exists "$SERVICE_NAME"; then
        echo "[info] Docker deployment is not installed; system service deployments use isrvd-system.sh"
        return
    fi
    assert_isrvd_container
    docker rm -f "$SERVICE_NAME"
    echo "[info] Docker deployment removed; data preserved: $DOCKER_DATA_DIR"
}

print_usage() {
    cat <<EOF
Usage: $0 install [--tag TAG] [--variant slim|caddy|apisix] [--cn|--global|--auto]
       $0 update [--tag TAG] [--variant slim|caddy|apisix] [--cn|--global|--auto]
       $0 uninstall

Fully containerized deployment only. For systemd + component containers, use isrvd-system.sh.
Tags default to slim/caddy/apisix; --caddy/--apisix select the matching all-in-one image.
Custom tags must exist in the selected registry; use --variant for their port layout.
Built-in slim/latest/caddy/apisix tags must match --variant on both install and update.
Updates preserve the installed tag and role unless --tag changes the image.
Sources: --auto (default), --cn (CNB), --global (Docker Hub).
Installs initialize Swarm and create the attachable overlay network sdnet.

Install reads exported environment variables:
  ISRVD_MODE          server (default), center, or agent
  ISRVD_CENTER_URL    Required for agent
  ISRVD_NODE_TOKEN    Required for agent
Agents do not publish 8080. Centers publish portmap.range (TCP/UDP) from the config;
also set portmap.bind: 0.0.0.0 and run update.
Data: $DOCKER_DATA_DIR; config: $DOCKER_DATA_DIR/conf/isrvd.yml.
Uninstall removes only the isrvd container and preserves system services and components.
EOF
}

main() {
    local command="${1:-}"
    case "$command" in
        --help|-h) print_usage; return ;;
        install|update|uninstall) shift ;;
        download) fail "Binary downloads require isrvd-system.sh download" ;;
        *) print_usage; return 1 ;;
    esac
    if [ "$command" = uninstall ] && [ "$#" -gt 0 ]; then
        case "$1" in --help|-h) print_usage; return ;; esac
        fail "Uninstall does not accept options"
    fi
    parse_options "$command" "$@"
    [ "$EUID" -eq 0 ] || fail "Please run as root"
    [ "$command" = uninstall ] || init_arch
    "$command"
}

main "$@"
