<?php
/**
 * iSrvd 安装脚本代理
 * 部署于 https://isrvd.rehiy.com/install/
 *
 * 默认代理 isrvd-system.sh（系统服务），?script=docker 代理 isrvd-docker.sh。
 * 两种脚本分别缓存 5 分钟；反向代理/CDN 缓存键须包含 script 查询参数。
 */

header('Content-Type: text/plain; charset=utf-8');

// ── 常量配置 ──────────────────────────────────────────────

const CACHE_TTL = 5 * 60;
const SCRIPT_FILES = [
    'systemd' => 'isrvd-system.sh',
    'docker'  => 'isrvd-docker.sh',
];
const SCRIPT_BASE_URL = 'https://raw.githubusercontent.com/rehiy/isrvd/refs/heads/master/build/script/';

// ── 入口 ──────────────────────────────────────────────────

function main(): void
{
    $script = $_GET['script'] ?? 'systemd';
    if (!is_string($script) || !array_key_exists($script, SCRIPT_FILES)) {
        http_response_code(400);
        header('Cache-Control: no-store');
        die("Error: script must be systemd or docker.\n");
    }
    $content = fetchScript($script);

    if (!$content) {
        http_response_code(502);
        die("Error: unable to fetch upstream script and no cached copy available.\n");
    }

    respond($content);
}

// ── 工具函数 ──────────────────────────────────────────────

function respond(string $content): void
{
    $ttl = CACHE_TTL;

    header("Cache-Control: public, max-age=$ttl");
    header('Content-Length: ' . strlen($content));
    echo $content;
    exit;
}

/**
 * 从 GitHub 获取安装脚本，结果缓存 CACHE_TTL 秒
 * 拉取失败时返回旧缓存（若有），确保用户始终能获取到脚本
 */
function fetchScript(string $script): ?string
{
    $cached = null;
    $cacheFile = __DIR__ . '/isrvd-v4-' . $script . '.sh';
    $scriptUrl = SCRIPT_BASE_URL . SCRIPT_FILES[$script];

    if (file_exists($cacheFile)) {
        $cached = file_get_contents($cacheFile);
        if ($cached === false) {
            $cached = null;
        }
        if ($cached !== null && (time() - filemtime($cacheFile)) < CACHE_TTL) {
            return $cached;
        }
    }

    $ctx = stream_context_create([
        'http' => [
            'method'        => 'GET',
            'header'        => 'User-Agent: iSrvd-Proxy/1.0',
            'timeout'       => 15,
            'ignore_errors' => true,
        ],
    ]);

    $body = @file_get_contents($scriptUrl, false, $ctx);

    $status = 0;
    foreach ($http_response_header ?? [] as $h) {
        if (preg_match('#^HTTP/\S+\s+(\d+)#', $h, $m)) {
            $status = (int) $m[1];
        }
    }

    if (!$body || $status !== 200) {
        return $cached;
    }

    file_put_contents($cacheFile, $body, LOCK_EX);

    return $body;
}

main();
