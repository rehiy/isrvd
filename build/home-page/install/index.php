<?php
/**
 * iSrvd 安装脚本代理
 * 部署于 https://isrvd.rehiy.com/install/
 *
 * 用于代理 GitHub 上的 isrvd.sh 安装脚本，提供缓存加速。
 * 缓存 12 小时，过期后重新拉取。
 */

header('Content-Type: text/plain; charset=utf-8');

// ── 常量配置 ──────────────────────────────────────────────

const CACHE_FILE    = __DIR__ . '/isrvd.sh';
const CACHE_TTL     = 12 * 3600; // 12 小时

const SCRIPT_URL    = 'https://raw.githubusercontent.com/rehiy/isrvd/refs/heads/master/build/script/isrvd.sh';

// ── 入口 ──────────────────────────────────────────────────

function main(): void
{
    $content = fetchScript();

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
function fetchScript(): ?string
{
    $cached = null;

    if (file_exists(CACHE_FILE)) {
        $cached = file_get_contents(CACHE_FILE);
        if ($cached !== false && (time() - filemtime(CACHE_FILE)) < CACHE_TTL) {
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

    $body = @file_get_contents(SCRIPT_URL, false, $ctx);

    $status = 0;
    foreach ($http_response_header ?? [] as $h) {
        if (preg_match('#^HTTP/\S+\s+(\d+)#', $h, $m)) {
            $status = (int) $m[1];
        }
    }

    if (!$body || $status !== 200) {
        return $cached;
    }

    $cacheDir = dirname(CACHE_FILE);
    if (!is_dir($cacheDir)) {
        mkdir($cacheDir, 0755, true);
    }
    file_put_contents(CACHE_FILE, $body, LOCK_EX);

    return $body;
}

main();
