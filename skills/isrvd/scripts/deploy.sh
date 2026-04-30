#!/usr/bin/env bash
# =============================================================================
# isrvd Deploy Harness — 一键部署快捷脚本
# 用法: ./scripts/deploy.sh <mode> <name> [compose-file] [options...]
#
# 模式:
#   container <name> <image>  [json-opts]    — 创建单个容器
#   compose   <name> <file>   [--init-url=]  — Docker Compose 部署（单机）
#   swarm     <name> <file>                  — Swarm Stack 部署（集群）
#   service   <name> <image>  [json-opts]    — 创建 Swarm 服务
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/api.sh"

# ---------------------------------------------------------------------------
# 用法提示
# ---------------------------------------------------------------------------
usage() {
  cat <<'EOF'
用法: ./scripts/deploy.sh <mode> <name> <args...>

模式:
  container <name> <image> [json-overrides]
    创建并启动一个 Docker 容器
    示例: ./scripts/deploy.sh container my-nginx nginx:latest '{"ports":{"80":"80"}}'

  compose <project-name> <compose-file> [--init-url=URL]
    通过 Docker Compose 部署（单机模式）
    示例: ./scripts/deploy.sh compose my-app ./docker-compose.yml

  swarm <stack-name> <compose-file>
    通过 Swarm Stack 部署（集群模式）
    示例: ./scripts/deploy.sh swarm my-stack ./docker-compose.yml

  service <name> <image> [json-overrides]
    创建一个 Swarm 服务
    示例: ./scripts/deploy.sh service web nginx:latest '{"replicas":3,"ports":[{"targetPort":80,"publishedPort":80}]}'

环境变量:
  ISRVD_BASE_URL   isrvd 服务地址（必须）
  ISRVD_TOKEN      认证 token（必须，或先执行 isrvd_init）

EOF
  exit 1
}

# ---------------------------------------------------------------------------
# deploy_container — 创建容器
# ---------------------------------------------------------------------------
deploy_container() {
  local name="${1:?缺少容器名}"
  local image="${2:?缺少镜像名}"
  local overrides="${3:-{}}"

  _blue "→ 部署容器: $name (镜像: $image)"

  # 合并默认配置和用户覆盖
  local body
  body=$(echo "$overrides" | jq --arg name "$name" --arg image "$image" \
    '. + {name: $name, image: $image}')

  local result
  result=$(isrvd_post "/docker/container/create" "$body")

  if echo "$result" | jq -e '.success == true' >/dev/null 2>&1; then
    _green "✓ 容器创建成功"
    echo "$result" | jq '.payload'
  else
    _red "✗ 容器创建失败"
    echo "$result" | jq '.message'
    return 1
  fi
}

# ---------------------------------------------------------------------------
# deploy_compose — Docker Compose 部署
# ---------------------------------------------------------------------------
deploy_compose() {
  local project_name="${1:?缺少项目名}"
  local compose_file="${2:?缺少 compose 文件路径}"
  local init_url="${3:-}"

  if [ ! -f "$compose_file" ]; then
    _red "✗ 文件不存在: $compose_file"
    return 1
  fi

  _blue "→ Compose 部署: $project_name (文件: $compose_file)"

  _isrvd_check || return 1

  local url="$ISRVD_BASE_URL/api/compose/docker/deploy"
  local args=(
    -s
    -X POST
    -H "Authorization: Bearer $ISRVD_TOKEN"
    -F "projectName=$project_name"
    -F "content=$(cat "$compose_file")"
  )

  # 附加初始化 URL
  if [ -n "$init_url" ]; then
    args+=(-F "initURL=$init_url")
  fi

  local result
  result=$(curl "${args[@]}" "$url")

  if echo "$result" | jq -e '.success == true' >/dev/null 2>&1; then
    _green "✓ Compose 部署成功"
    echo "$result" | jq '.payload'
  else
    _red "✗ Compose 部署失败"
    echo "$result" | jq '.'
    return 1
  fi
}

# ---------------------------------------------------------------------------
# deploy_swarm — Swarm Stack 部署
# ---------------------------------------------------------------------------
deploy_swarm() {
  local stack_name="${1:?缺少 Stack 名}"
  local compose_file="${2:?缺少 compose 文件路径}"

  if [ ! -f "$compose_file" ]; then
    _red "✗ 文件不存在: $compose_file"
    return 1
  fi

  _blue "→ Swarm Stack 部署: $stack_name (文件: $compose_file)"

  local content
  content=$(cat "$compose_file")

  local body
  body=$(jq -n --arg name "$stack_name" --arg content "$content" \
    '{projectName: $name, content: $content}')

  local result
  result=$(isrvd_post "/compose/swarm/deploy" "$body")

  if echo "$result" | jq -e '.success == true' >/dev/null 2>&1; then
    _green "✓ Swarm Stack 部署成功"
    echo "$result" | jq '.payload'
  else
    _red "✗ Swarm Stack 部署失败"
    echo "$result" | jq '.'
    return 1
  fi
}

# ---------------------------------------------------------------------------
# deploy_service — 创建 Swarm 服务
# ---------------------------------------------------------------------------
deploy_service() {
  local name="${1:?缺少服务名}"
  local image="${2:?缺少镜像名}"
  local overrides="${3:-{}}"

  _blue "→ 创建 Swarm 服务: $name (镜像: $image)"

  local body
  body=$(echo "$overrides" | jq --arg name "$name" --arg image "$image" \
    '. + {name: $name, image: $image}')

  local result
  result=$(isrvd_post "/swarm/service/create" "$body")

  if echo "$result" | jq -e '.success == true' >/dev/null 2>&1; then
    _green "✓ Swarm 服务创建成功"
    echo "$result" | jq '.payload'
  else
    _red "✗ Swarm 服务创建失败"
    echo "$result" | jq '.'
    return 1
  fi
}

# ---------------------------------------------------------------------------
# 主入口
# ---------------------------------------------------------------------------
mode="${1:-}"
shift || true

case "$mode" in
  container) deploy_container "$@" ;;
  compose)
    # 解析 --init-url 参数
    init_url=""
    args=()
    for arg in "$@"; do
      case "$arg" in
        --init-url=*) init_url="${arg#--init-url=}" ;;
        *) args+=("$arg") ;;
      esac
    done
    deploy_compose "${args[@]}" "$init_url"
    ;;
  swarm)   deploy_swarm "$@" ;;
  service) deploy_service "$@" ;;
  *)       usage ;;
esac
