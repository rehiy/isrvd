#!/usr/bin/env bash
# =============================================================================
# isrvd Health Check — 系统健康检查与状态报告
# 用法: ./scripts/health-check.sh [--json]
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/api.sh"

JSON_MODE=false
[ "${1:-}" = "--json" ] && JSON_MODE=true

# ---------------------------------------------------------------------------
# 分隔线
# ---------------------------------------------------------------------------
_separator() {
  if ! $JSON_MODE; then
    echo "────────────────────────────────────────────────────"
  fi
}

# ---------------------------------------------------------------------------
# 检查服务可用性
# ---------------------------------------------------------------------------
check_probe() {
  _blue "▶ 服务可用性探测"
  local probe
  probe=$(isrvd_get "/system/probe" 2>/dev/null) || {
    _red "✗ 无法连接到 isrvd 服务"
    return 1
  }

  if $JSON_MODE; then
    echo "$probe"
    return
  fi

  local docker_ok swarm_ok apisix_ok
  docker_ok=$(echo "$probe" | jq -r '.payload.docker // false')
  swarm_ok=$(echo "$probe" | jq -r '.payload.swarm // false')
  apisix_ok=$(echo "$probe" | jq -r '.payload.apisix // false')

  [ "$docker_ok" = "true" ] && _green "  ✓ Docker: 可用" || _red "  ✗ Docker: 不可用"
  [ "$swarm_ok" = "true" ]  && _green "  ✓ Swarm:  可用" || _yellow "  ○ Swarm:  未启用"
  [ "$apisix_ok" = "true" ] && _green "  ✓ APISIX: 可用" || _yellow "  ○ APISIX: 未配置"
}

# ---------------------------------------------------------------------------
# 检查系统资源
# ---------------------------------------------------------------------------
check_stats() {
  _blue "▶ 系统资源统计"
  local stats
  stats=$(isrvd_get "/system/stats" 2>/dev/null) || {
    _red "✗ 无法获取系统统计"
    return 1
  }

  if $JSON_MODE; then
    echo "$stats"
    return
  fi

  echo "$stats" | jq -r '.payload | to_entries[] | "  \(.key): \(.value)"' 2>/dev/null || echo "  (无法解析)"
}

# ---------------------------------------------------------------------------
# 检查容器状态
# ---------------------------------------------------------------------------
check_containers() {
  _blue "▶ 容器状态概览"
  local containers
  containers=$(isrvd_get "/docker/containers?all=true" 2>/dev/null) || {
    _yellow "  ○ Docker 不可用，跳过容器检查"
    return 0
  }

  if $JSON_MODE; then
    echo "$containers"
    return
  fi

  local total running exited other
  total=$(echo "$containers" | jq '.payload | length')
  running=$(echo "$containers" | jq '[.payload[] | select(.state == "running")] | length')
  exited=$(echo "$containers" | jq '[.payload[] | select(.state == "exited")] | length')
  other=$((total - running - exited))

  _green "  运行中: $running"
  [ "$exited" -gt 0 ] && _yellow "  已停止: $exited" || echo "  已停止: $exited"
  [ "$other" -gt 0 ] && _red "  其他:   $other" || echo "  其他:   $other"
  echo "  总计:   $total"

  # 列出非运行状态的容器
  if [ "$exited" -gt 0 ] || [ "$other" -gt 0 ]; then
    echo ""
    _yellow "  非运行状态容器:"
    echo "$containers" | jq -r '.payload[] | select(.state != "running") | "    \(.name) [\(.state)] - \(.image)"'
  fi
}

# ---------------------------------------------------------------------------
# 检查 Swarm 服务状态
# ---------------------------------------------------------------------------
check_services() {
  _blue "▶ Swarm 服务状态"
  local services
  services=$(isrvd_get "/swarm/services" 2>/dev/null) || {
    _yellow "  ○ Swarm 不可用，跳过服务检查"
    return 0
  }

  if $JSON_MODE; then
    echo "$services"
    return
  fi

  local total healthy unhealthy
  total=$(echo "$services" | jq '.payload | length')

  if [ "$total" -eq 0 ]; then
    echo "  无 Swarm 服务"
    return
  fi

  echo "  服务总数: $total"
  echo ""

  # 逐个检查服务健康
  echo "$services" | jq -r '.payload[] | "\(.name)|\(.replicas)|\(.runningTasks)|\(.image)"' | while IFS='|' read -r name replicas running image; do
    if [ "$running" = "$replicas" ] && [ "$running" != "0" ]; then
      _green "  ✓ $name ($running/$replicas) - $image"
    elif [ "$running" = "0" ]; then
      _red "  ✗ $name ($running/$replicas) - $image"
    else
      _yellow "  △ $name ($running/$replicas) - $image"
    fi
  done
}

# ---------------------------------------------------------------------------
# 检查路由状态
# ---------------------------------------------------------------------------
check_routes() {
  _blue "▶ APISIX 路由状态"
  local routes
  routes=$(isrvd_get "/apisix/routes" 2>/dev/null) || {
    _yellow "  ○ APISIX 不可用，跳过路由检查"
    return 0
  }

  if $JSON_MODE; then
    echo "$routes"
    return
  fi

  local total enabled disabled
  total=$(echo "$routes" | jq '.payload | length')
  enabled=$(echo "$routes" | jq '[.payload[] | select(.status == 1)] | length')
  disabled=$((total - enabled))

  echo "  路由总数: $total"
  _green "  启用: $enabled"
  [ "$disabled" -gt 0 ] && _yellow "  禁用: $disabled"
}

# ---------------------------------------------------------------------------
# 主流程
# ---------------------------------------------------------------------------
if ! $JSON_MODE; then
  echo ""
  _blue "═══════════════════════════════════════════════════"
  _blue "         isrvd 健康检查报告"
  _blue "         $(date '+%Y-%m-%d %H:%M:%S')"
  _blue "═══════════════════════════════════════════════════"
  echo ""
fi

check_probe
_separator
check_stats
_separator
check_containers
_separator
check_services
_separator
check_routes

if ! $JSON_MODE; then
  echo ""
  _separator
  _green "✓ 健康检查完成"
  echo ""
fi
