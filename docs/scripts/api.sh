#!/usr/bin/env bash
# =============================================================================
# isrvd API Harness — 认证持久化 + 通用 curl 封装
# 用法: source scripts/api.sh
# =============================================================================

ISRVD_CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/isrvd"
ISRVD_CONFIG_FILE="$ISRVD_CONFIG_DIR/profile.json"

ISRVD_BASE_URL="${ISRVD_BASE_URL:-}"
ISRVD_TOKEN="${ISRVD_TOKEN:-}"
ISRVD_USERNAME="${ISRVD_USERNAME:-}"

_red()    { printf '\033[0;31m%s\033[0m\n' "$*"; }
_green()  { printf '\033[0;32m%s\033[0m\n' "$*"; }
_yellow() { printf '\033[0;33m%s\033[0m\n' "$*"; }
_blue()   { printf '\033[0;34m%s\033[0m\n' "$*"; }

# ---------------------------------------------------------------------------
# 配置文件读写
# ---------------------------------------------------------------------------
_isrvd_save_config() {
  mkdir -p "$ISRVD_CONFIG_DIR"
  jq -n \
    --arg url "$ISRVD_BASE_URL" \
    --arg token "$ISRVD_TOKEN" \
    --arg user "$ISRVD_USERNAME" \
    '{base_url: $url, token: $token, username: $user}' > "$ISRVD_CONFIG_FILE"
  chmod 600 "$ISRVD_CONFIG_FILE"
}

_isrvd_load_config() {
  # 环境变量优先
  if [ -n "$ISRVD_BASE_URL" ] && [ -n "$ISRVD_TOKEN" ]; then
    return 0
  fi

  if [ -f "$ISRVD_CONFIG_FILE" ]; then
    ISRVD_BASE_URL="${ISRVD_BASE_URL:-$(jq -r '.base_url // empty' "$ISRVD_CONFIG_FILE")}"
    ISRVD_TOKEN="${ISRVD_TOKEN:-$(jq -r '.token // empty' "$ISRVD_CONFIG_FILE")}"
    ISRVD_USERNAME="${ISRVD_USERNAME:-$(jq -r '.username // empty' "$ISRVD_CONFIG_FILE")}"
    return 0
  fi

  return 1
}

# ---------------------------------------------------------------------------
# isrvd_login — 用账号密码登录，token 保存到配置文件
# ---------------------------------------------------------------------------
isrvd_login() {
  local base_url="${1:?用法: isrvd_login <base_url> <username> <password>}"
  local username="${2:?缺少 username}"
  local password="${3:?缺少 password}"

  ISRVD_BASE_URL="${base_url%/}"
  ISRVD_USERNAME="$username"

  _blue "→ 登录 $ISRVD_BASE_URL ..."

  local resp
  resp=$(curl -sf -X POST "$ISRVD_BASE_URL/api/account/login" \
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
  _isrvd_save_config
  _green "✓ 登录成功，已保存到 $ISRVD_CONFIG_FILE"
}

# ---------------------------------------------------------------------------
# isrvd_token — 直接用 token 认证，保存到配置文件
# ---------------------------------------------------------------------------
isrvd_token() {
  local base_url="${1:?用法: isrvd_token <base_url> <token>}"
  local token="${2:?缺少 token}"

  ISRVD_BASE_URL="${base_url%/}"
  ISRVD_TOKEN="$token"
  ISRVD_USERNAME=""

  # 验证 token 有效性
  _blue "→ 验证 token ..."
  local resp
  resp=$(curl -sf "$ISRVD_BASE_URL/api/account/info" \
    -H "Authorization: Bearer $ISRVD_TOKEN" 2>&1) || {
    _red "✗ token 无效或服务不可达"
    return 1
  }

  local success
  success=$(echo "$resp" | jq -r '.success // false')
  if [ "$success" != "true" ]; then
    _red "✗ token 无效: $(echo "$resp" | jq -r '.message // "未知错误"')"
    return 1
  fi

  ISRVD_USERNAME=$(echo "$resp" | jq -r '.payload.username // "unknown"')
  _isrvd_save_config
  _green "✓ token 有效 (用户: $ISRVD_USERNAME)，已保存到 $ISRVD_CONFIG_FILE"
}

# ---------------------------------------------------------------------------
# _isrvd_check — 确保已认证（自动加载配置文件）
# ---------------------------------------------------------------------------
_isrvd_check() {
  _isrvd_load_config 2>/dev/null || true

  if [ -z "$ISRVD_BASE_URL" ] || [ -z "$ISRVD_TOKEN" ]; then
    _red "✗ 未认证。请先执行:" >&2
    _red "  isrvd_login <base_url> <username> <password>" >&2
    _red "  isrvd_token <base_url> <token>" >&2
    return 1
  fi
}

# ---------------------------------------------------------------------------
# _isrvd_format — 数组对象自动转紧凑表格
# 输入: jq 输出（stdin），输出: 表格或原样
# 表格格式: [N]{field1,field2,...}:
#             val1,val2,...
# ---------------------------------------------------------------------------
_isrvd_format() {
  local input
  input=$(cat)
  [ -z "$input" ] && return

  local line_count first_line first_type
  line_count=$(printf '%s\n' "$input" | wc -l | tr -d ' ')
  first_line=$(printf '%s\n' "$input" | head -1)
  first_type=$(printf '%s' "$first_line" | jq -r 'type' 2>/dev/null) || {
    printf '%s\n' "$input"; return
  }

  local _fmt='keys_unsorted as $ks |
    [$ks[] as $k | .[$k]] |
    map(if . == null then "null"
        elif type == "string" then @json
        elif type == "number" or type == "boolean" then tostring
        else tojson end) |
    "  " + join(",")'

  # 多行对象 → 表格
  if [ "$line_count" -gt 1 ] && [ "$first_type" = "object" ]; then
    local keys
    keys=$(printf '%s' "$first_line" | jq -r 'keys_unsorted | join(",")')
    printf '[%s]{%s}:\n' "$line_count" "$keys"
    printf '%s\n' "$input" | jq -r "$_fmt"
    return
  fi

  # 单行数组（元素为对象）→ 表格
  if [ "$line_count" -eq 1 ] && [ "$first_type" = "array" ]; then
    local arr_len inner_type
    arr_len=$(printf '%s' "$input" | jq -r 'length')
    if [ "$arr_len" -gt 0 ]; then
      inner_type=$(printf '%s' "$input" | jq -r '.[0] | type')
      if [ "$inner_type" = "object" ]; then
        local keys
        keys=$(printf '%s' "$input" | jq -r '.[0] | keys_unsorted | join(",")')
        printf '[%s]{%s}:\n' "$arr_len" "$keys"
        printf '%s' "$input" | jq -r "(.[\$zero] | keys_unsorted) as \$ks |
          .[] | . as \$o |
          [\$ks[] as \$k | \$o[\$k]] |
          map(if . == null then \"null\"
              elif type == \"string\" then @json
              elif type == \"number\" or type == \"boolean\" then tostring
              else tojson end) |
          \"  \" + join(\",\")" --argjson zero 0
        return
      fi
    fi
  fi

  printf '%s\n' "$input"
}

# ---------------------------------------------------------------------------
# _isrvd_curl — 底层 curl 封装
# 参数: method path [body] [jq_filter]
# 默认输出紧凑 JSON（.payload），传 jq_filter 可按需提取字段
# 数组对象自动转紧凑表格格式，大幅减少 token 消耗
# ---------------------------------------------------------------------------
_isrvd_curl() {
  local method="$1"
  local path="$2"
  local body=""
  local jq_filter=""
  shift 2

  # GET/DELETE: 没有 body，剩余参数是 jq filter
  # POST/PUT/PATCH: 第一个参数是 body，第二个是 jq filter
  case "$method" in
    GET|DELETE)
      jq_filter="${1:-}"
      ;;
    *)
      body="${1:-}"
      jq_filter="${2:-}"
      ;;
  esac

  _isrvd_check || return 1

  local url="$ISRVD_BASE_URL/api${path}"
  local args=(
    -s -w "\n%{http_code}"
    -X "$method"
    -H "Authorization: Bearer $ISRVD_TOKEN"
    -H "Content-Type: application/json"
  )

  if [ -n "$body" ]; then
    args+=(-d "$body")
  fi

  local output
  output=$(curl "${args[@]}" "$url" 2>&1)

  local http_code
  http_code=$(echo "$output" | tail -1)
  local raw
  raw=$(echo "$output" | sed '$d')

  if [[ ! "$http_code" =~ ^2 ]]; then
    echo "$raw" | jq -rc . 2>/dev/null || echo "$raw"
    _red "✗ HTTP $http_code" >&2
    return 1
  fi

  # 提取 .payload（统一响应格式），若不存在则用原始 body
  local payload
  payload=$(echo "$raw" | jq -c '.payload // .' 2>/dev/null) || payload="$raw"

  local result
  if [ -n "$jq_filter" ]; then
    result=$(echo "$payload" | jq -rc "$jq_filter")
  else
    result=$(echo "$payload" | jq -rc .)
  fi

  printf '%s\n' "$result" | _isrvd_format
}

# ---------------------------------------------------------------------------
# 便捷方法
# GET/DELETE: isrvd_get <path> [jq_filter]
# POST/PUT/PATCH: isrvd_post <path> [body] [jq_filter]
# ---------------------------------------------------------------------------
isrvd_get()    { _isrvd_curl GET    "$@"; }
isrvd_post()   { _isrvd_curl POST   "$@"; }
isrvd_put()    { _isrvd_curl PUT    "$@"; }
isrvd_patch()  { _isrvd_curl PATCH  "$@"; }
isrvd_delete() { _isrvd_curl DELETE "$@"; }

# ---------------------------------------------------------------------------
# isrvd_upload — multipart/form-data 上传
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

  for field in "$@"; do
    args+=(-F "$field")
  done

  curl "${args[@]}" "$url" | jq .
}

# ---------------------------------------------------------------------------
# isrvd_whoami — 查看当前认证状态
# ---------------------------------------------------------------------------
isrvd_whoami() {
  _isrvd_check || return 1
  _blue "→ $ISRVD_BASE_URL (用户: ${ISRVD_USERNAME:-unknown})"
  isrvd_get "/account/info"
}

# ---------------------------------------------------------------------------
# isrvd_status — 查看配置文件状态
# ---------------------------------------------------------------------------
isrvd_status() {
  if [ -f "$ISRVD_CONFIG_FILE" ]; then
    _green "✓ 配置文件: $ISRVD_CONFIG_FILE"
    jq '{base_url, username, token: (.token[:20] + "...")}' "$ISRVD_CONFIG_FILE"
  else
    _yellow "○ 无配置文件"
  fi
}

# ---------------------------------------------------------------------------
# isrvd_logout — 清除配置文件
# ---------------------------------------------------------------------------
isrvd_logout() {
  if [ -f "$ISRVD_CONFIG_FILE" ]; then
    rm -f "$ISRVD_CONFIG_FILE"
    ISRVD_BASE_URL=""
    ISRVD_TOKEN=""
    ISRVD_USERNAME=""
    _green "✓ 已清除认证信息"
  else
    _yellow "○ 无需清除"
  fi
}

# ---------------------------------------------------------------------------
# 帮助
# ---------------------------------------------------------------------------
isrvd_help() {
  cat <<'EOF'
isrvd API Harness

  认证（持久化到 ~/.config/isrvd/profile.json）:
    isrvd_login  <url> <user> <pass>     账号密码登录
    isrvd_token  <url> <token>           直接用 token
    isrvd_logout                         清除认证
    isrvd_status                         查看当前配置
    isrvd_whoami                         当前用户信息

  API 调用（自动提取 .payload，数组对象转紧凑表格）:
    isrvd_get    <path>                          GET
    isrvd_post   <path> [body]                   POST
    isrvd_put    <path> [body]                    PUT
    isrvd_patch  <path> [body]                   PATCH
    isrvd_delete <path>                          DELETE
    isrvd_upload <path> <field> <file> [k=v...]  文件上传

  认证优先级: 环境变量 > 配置文件

  示例:
    isrvd_token "$ISRVD_APIURL" "$ISRVD_APITOKEN"
    isrvd_get "/docker/containers"
    isrvd_post "/docker/container" '{"image":"...","name":"..."}'
    isrvd_get "/swarm/services"
EOF
}

# ---------------------------------------------------------------------------
# 自动加载配置
# ---------------------------------------------------------------------------
if _isrvd_load_config 2>/dev/null; then
  _green "✓ isrvd harness 已加载 (${ISRVD_BASE_URL:-未配置})" >&2
else
  _yellow "○ isrvd harness 已加载，未找到配置。执行 isrvd_login 或 isrvd_token 初始化。" >&2
fi
