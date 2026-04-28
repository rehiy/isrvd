#!/usr/bin/env bash
# =============================================================================
# isrvd API Harness — 通用 curl 封装、认证、环境检测
# 用法: source scripts/api.sh
# =============================================================================
set -euo pipefail

# ---------------------------------------------------------------------------
# 全局变量
# ---------------------------------------------------------------------------
ISRVD_BASE_URL="${ISRVD_BASE_URL:-}"
ISRVD_TOKEN="${ISRVD_TOKEN:-}"
ISRVD_USERNAME="${ISRVD_USERNAME:-}"

# ---------------------------------------------------------------------------
# 颜色输出
# ---------------------------------------------------------------------------
_red()    { printf '\033[0;31m%s\033[0m\n' "$*"; }
_green()  { printf '\033[0;32m%s\033[0m\n' "$*"; }
_yellow() { printf '\033[0;33m%s\033[0m\n' "$*"; }
_blue()   { printf '\033[0;34m%s\033[0m\n' "$*"; }

# ---------------------------------------------------------------------------
# isrvd_init — 初始化连接并登录
# 参数: base_url username password
# ---------------------------------------------------------------------------
isrvd_init() {
  local base_url="${1:?用法: isrvd_init <base_url> <username> <password>}"
  local username="${2:?缺少 username}"
  local password="${3:?缺少 password}"

  ISRVD_BASE_URL="${base_url%/}"
  ISRVD_USERNAME="$username"

  _blue "→ 连接 $ISRVD_BASE_URL ..."

  # 登录获取 token
  local resp
  resp=$(curl -sf -X POST "$ISRVD_BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$username\",\"password\":\"$password\"}" 2>&1) || {
    _red "✗ 登录失败: $resp"
    return 1
  }

  local success
  success=$(echo "$resp" | jq -r '.success // false')
  if [ "$success" != "true" ]; then
    _red "✗ 登录失败: $(echo "$resp" | jq -r '.message // "未知错误"')"
    return 1
  fi

  ISRVD_TOKEN=$(echo "$resp" | jq -r '.payload.token')
  _green "✓ 登录成功 (用户: $username)"
}

# ---------------------------------------------------------------------------
# _isrvd_check — 检查是否已初始化
# ---------------------------------------------------------------------------
_isrvd_check() {
  if [ -z "$ISRVD_BASE_URL" ] || [ -z "$ISRVD_TOKEN" ]; then
    _red "✗ 未初始化，请先执行: isrvd_init <base_url> <username> <password>"
    return 1
  fi
}

# ---------------------------------------------------------------------------
# _isrvd_curl — 底层 curl 封装
# 参数: method path [data]
# ---------------------------------------------------------------------------
_isrvd_curl() {
  local method="$1"
  local path="$2"
  shift 2

  _isrvd_check || return 1

  local url="$ISRVD_BASE_URL/api${path}"
  local args=(
    -s -w "\n%{http_code}"
    -X "$method"
    -H "Authorization: Bearer $ISRVD_TOKEN"
    -H "Content-Type: application/json"
  )

  # 如果有 body 数据
  if [ $# -gt 0 ] && [ -n "$1" ]; then
    args+=(-d "$1")
  fi

  local output
  output=$(curl "${args[@]}" "$url" 2>&1)

  local http_code
  http_code=$(echo "$output" | tail -1)
  local body
  body=$(echo "$output" | sed '$d')

  # 输出格式化的 JSON
  if echo "$body" | jq . >/dev/null 2>&1; then
    echo "$body" | jq .
  else
    echo "$body"
  fi

  # 非 2xx 状态码时返回错误
  if [[ ! "$http_code" =~ ^2 ]]; then
    _red "✗ HTTP $http_code" >&2
    return 1
  fi
}

# ---------------------------------------------------------------------------
# 便捷方法
# ---------------------------------------------------------------------------
isrvd_get()    { _isrvd_curl GET    "$@"; }
isrvd_post()   { _isrvd_curl POST   "$@"; }
isrvd_put()    { _isrvd_curl PUT    "$@"; }
isrvd_patch()  { _isrvd_curl PATCH  "$@"; }
isrvd_delete() { _isrvd_curl DELETE "$@"; }

# ---------------------------------------------------------------------------
# isrvd_upload — multipart/form-data 上传
# 参数: path file_field file_path [extra_fields...]
# extra_fields 格式: "key=value"
# ---------------------------------------------------------------------------
isrvd_upload() {
  local path="${1:?用法: isrvd_upload <path> <file_field> <file_path> [key=value ...]}"
  local file_field="${2:?缺少 file_field}"
  local file_path="${3:?缺少 file_path}"
  shift 3

  _isrvd_check || return 1

  local url="$ISRVD_BASE_URL/api${path}"
  local args=(
    -s
    -X POST
    -H "Authorization: Bearer $ISRVD_TOKEN"
    -F "${file_field}=@${file_path}"
  )

  # 附加表单字段
  for field in "$@"; do
    args+=(-F "$field")
  done

  curl "${args[@]}" "$url" | jq .
}

# ---------------------------------------------------------------------------
# isrvd_probe — 快速探测服务可用性
# ---------------------------------------------------------------------------
isrvd_probe() {
  _isrvd_check || return 1
  _blue "→ 探测服务可用性..."
  isrvd_get "/system/probe"
}

# ---------------------------------------------------------------------------
# isrvd_stats — 查看系统资源统计
# ---------------------------------------------------------------------------
isrvd_stats() {
  _isrvd_check || return 1
  _blue "→ 获取系统资源统计..."
  isrvd_get "/system/stats"
}

# ---------------------------------------------------------------------------
# isrvd_whoami — 查看当前认证信息
# ---------------------------------------------------------------------------
isrvd_whoami() {
  _isrvd_check || return 1
  _blue "→ 当前用户: $ISRVD_USERNAME"
  isrvd_get "/auth/info"
}

# ---------------------------------------------------------------------------
# 帮助信息
# ---------------------------------------------------------------------------
isrvd_help() {
  cat <<'EOF'
isrvd API Harness — 可用函数:

  初始化:
    isrvd_init <base_url> <username> <password>   登录并初始化

  通用 API:
    isrvd_get    <path>                            GET 请求
    isrvd_post   <path> [json_body]                POST 请求
    isrvd_put    <path> [json_body]                PUT 请求
    isrvd_patch  <path> [json_body]                PATCH 请求
    isrvd_delete <path>                            DELETE 请求
    isrvd_upload <path> <field> <file> [k=v ...]   文件上传

  快捷命令:
    isrvd_probe                                    探测服务可用性
    isrvd_stats                                    系统资源统计
    isrvd_whoami                                   当前用户信息
    isrvd_help                                     显示此帮助

  示例:
    source scripts/api.sh
    isrvd_init "http://localhost:8080" "admin" "mypassword"
    isrvd_get "/docker/containers?all=true"
    isrvd_post "/docker/container/create" '{"image":"nginx:latest","name":"web"}'
    isrvd_post "/docker/container/abc123/action" '{"action":"start"}'
    isrvd_get "/swarm/services"
    isrvd_get "/apisix/routes"
EOF
}

_green "✓ isrvd API harness 已加载。执行 isrvd_help 查看可用命令。"
