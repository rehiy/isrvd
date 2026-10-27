<?php
/**
 * iSrvd 安装脚本代理
 *
 * 部署地址：https://isrvd.rehiy.com/install/
 * 功能说明：按安装方式返回系统服务或 Docker 安装脚本。
 *
 * 请求参数（GET）：
 *   script  安装方式，可选 systemd（默认）或 docker。
 *
 * 响应格式：
 *   Shell 脚本文本；非法参数返回 HTTP 400，下载失败且无有效缓存时返回 HTTP 502。
 *
 * 缓存策略：
 *   两种脚本独立缓存 1 小时；成功响应缓存 1 小时，错误响应不缓存。
 *   反向代理/CDN 缓存键须包含 script 查询参数。
 */

// ── 常量配置 ──────────────────────────────────────────────

const CONTENT_TYPE      = 'text/plain; charset=utf-8';
const CACHE_TTL         = 3600; // 1 小时
const DOWNLOAD_BASE_URL = 'https://raw.githubusercontent.com/rehiy/isrvd/refs/heads/master/build/script/';
const DOWNLOAD_FILES    = [
    'isrvd-v4-systemd.sh' => 'isrvd-system.sh',
    'isrvd-v4-docker.sh'  => 'isrvd-docker.sh',
];
const DOWNLOAD_HEADERS  = [];
const SCRIPT_FILES      = [
    'systemd' => 'isrvd-v4-systemd.sh',
    'docker'  => 'isrvd-v4-docker.sh',
];

// ── 入口 ──────────────────────────────────────────────────

/**
 * 校验安装方式，并返回对应的安装脚本内容。
 */
function main(): void
{
    $script = $_GET['script'] ?? 'systemd';
    if (!is_string($script) || !array_key_exists($script, SCRIPT_FILES)) {
        respond("Error: script must be systemd or docker.\n", 400);
    }

    $content = fetchFile(SCRIPT_FILES[$script]);
    if ($content === null) {
        respond("Error: unable to fetch upstream script and no cached copy available.\n", 502);
    }

    respond($content);
}

// ── 工具函数 ──────────────────────────────────────────────

/**
 * 输出响应并终止请求。
 * 仅允许缓存的 HTTP 200 响应使用 CACHE_TTL，其余响应禁止缓存。
 */
function respond(string $content, int $status = 200, bool $cacheable = true): void
{
    http_response_code($status);
    header('Content-Type: ' . CONTENT_TYPE);
    header('Cache-Control: ' . ($status === 200 && $cacheable
        ? 'public, max-age=' . CACHE_TTL
        : 'no-store'));
    header('Content-Length: ' . strlen($content));
    echo $content;
    exit;
}

/**
 * 下载白名单文件，优先使用 CACHE_TTL 秒内的有效缓存。
 * 拉取失败或内容无效时回退到有效旧缓存；有效新内容通过临时文件原子写入。
 * 缓存写入失败不影响返回已下载的有效内容。
 * $file 为 DOWNLOAD_FILES 的本地缓存文件名，$validate 为可选业务校验函数。
 */
function fetchFile(string $file, ?callable $validate = null): ?string
{
    if (!array_key_exists($file, DOWNLOAD_FILES)) {
        return null;
    }

    $cached = null;
    $cacheFile = __DIR__ . '/' . $file;
    clearstatcache(true, $cacheFile);

    if (is_file($cacheFile)) {
        $cached = @file_get_contents($cacheFile);
        if ($cached === false || !isValidFile($file, $cached, $validate)) {
            $cached = null;
        }
        $mtime = @filemtime($cacheFile);
        if ($cached !== null && $mtime !== false && (time() - $mtime) < CACHE_TTL) {
            return $cached;
        }
    }

    $ctx = stream_context_create([
        'http' => [
            'method'          => 'GET',
            'header'          => implode("\r\n", array_merge([
                'User-Agent: iSrvd-Proxy/1.0',
            ], DOWNLOAD_HEADERS)),
            'timeout'         => 15,
            'ignore_errors'   => true,
            'follow_location' => 0,
        ],
    ]);

    $body = @file_get_contents(DOWNLOAD_BASE_URL . DOWNLOAD_FILES[$file], false, $ctx);

    $status = 0;
    foreach ($http_response_header ?? [] as $h) {
        if (preg_match('#^HTTP/\S+\s+(\d+)#', $h, $m)) {
            $status = (int) $m[1];
        }
    }

    if ($body === false || $status !== 200 || !isValidFile($file, $body, $validate)) {
        return $cached;
    }

    $cacheDir = dirname($cacheFile);
    if (!is_dir($cacheDir) && !@mkdir($cacheDir, 0755, true) && !is_dir($cacheDir)) {
        return $body;
    }

    $tempFile = @tempnam($cacheDir, '.isrvd-');
    if ($tempFile === false) {
        return $body;
    }
    if (dirname($tempFile) !== $cacheDir
        || @file_put_contents($tempFile, $body, LOCK_EX) !== strlen($body)
        || !@chmod($tempFile, 0644)
        || !@rename($tempFile, $cacheFile)) {
        @unlink($tempFile);
    }
    clearstatcache(true, $cacheFile);

    return $body;
}

/**
 * 校验非空内容、PHP 语法及 JSON 对象或数组格式。
 * 基础校验通过后执行可选业务校验。
 */
function isValidFile(string $file, string $content, ?callable $validate = null): bool
{
    if ($content === '') {
        return false;
    }

    if (substr($file, -4) === '.php') {
        if (strncmp($content, '<?php', 5) !== 0) {
            return false;
        }
        try {
            token_get_all($content, TOKEN_PARSE);
        } catch (ParseError $e) {
            return false;
        }
    }

    if (substr($file, -5) === '.json') {
        $data = json_decode($content, true);
        if (!is_array($data) || json_last_error() !== JSON_ERROR_NONE) {
            return false;
        }
    }

    return $validate === null || $validate($content);
}

main();
