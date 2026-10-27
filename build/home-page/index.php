<?php
/**
 * iSrvd 首页代理
 *
 * 部署地址：https://isrvd.rehiy.com/
 * 功能说明：同步主页和安装、升级入口文件，返回 home.html 内容。
 *
 * 请求参数（GET）：
 *   无。
 *
 * 响应格式：
 *   HTML；下载失败且无有效缓存时返回 HTTP 502。
 *
 * 缓存策略：
 *   下载文件及成功响应缓存 1 小时；错误响应不缓存。
 */

// ── 常量配置 ──────────────────────────────────────────────

const CONTENT_TYPE      = 'text/html; charset=utf-8';
const CACHE_TTL         = 3600; // 1 小时
const DOWNLOAD_BASE_URL = 'https://raw.githubusercontent.com/rehiy/isrvd/refs/heads/master/build/home-page/';
const DOWNLOAD_FILES    = [
    'home.html'         => 'home.html',
    'install/index.php' => 'install/index.php',
    'update/index.php'  => 'update/index.php',
];
const DOWNLOAD_HEADERS  = [];

// ── 入口 ──────────────────────────────────────────────────

/**
 * 同步下载列表中的文件，并返回首页内容。
 */
function main(): void
{
    $content = null;
    foreach (array_keys(DOWNLOAD_FILES) as $file) {
        $body = fetchFile($file);
        if ($file === 'home.html') {
            $content = $body;
        }
    }

    if ($content === null) {
        respond("Error: unable to fetch upstream page and no cached copy available.\n", 502);
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
