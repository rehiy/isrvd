<?php
/**
 * iSrvd 自动升级服务端
 *
 * 部署地址：https://isrvd.rehiy.com/update/
 * 功能说明：比较客户端与最新 Release 版本，返回版本信息和匹配平台的升级包地址。
 *
 * 请求参数（GET）：
 *   ver   当前版本，如 v1.0.0。
 *   os    运行平台，如 linux、darwin 或 windows。
 *   arch  CPU 架构，如 amd64 或 arm64。
 *
 * 响应格式：
 *   UpdateInfo JSON，包含以下字段：
 *   version  最新版本号，成功获取 Release 数据时返回。
 *   release  GitHub Release 页面地址，成功获取 Release 数据时返回。
 *   package  升级包地址，有新版且存在匹配资源时返回，否则为空。
 *   message  提示信息。
 *   error    错误信息，非空时客户端终止更新。
 *
 * 缓存策略：
 *   Release 数据缓存 1 小时；按请求参数生成的 JSON 响应不缓存。
 */

// ── 常量配置 ──────────────────────────────────────────────

const CONTENT_TYPE      = 'application/json; charset=utf-8';
const CACHE_TTL         = 3600; // 1 小时
const DOWNLOAD_BASE_URL = 'https://api.github.com/repos/rehiy/isrvd/releases/';
const DOWNLOAD_FILES    = ['latest.json' => 'latest'];
const DOWNLOAD_HEADERS  = [
    'Accept: application/vnd.github+json',
    'X-GitHub-Api-Version: 2022-11-28',
];

// 操作系统和架构到升级包文件名的映射。
const ASSET_MAP = [
    'linux/amd64'   => 'isrvd-linux-amd64.tar.gz',
    'linux/arm64'   => 'isrvd-linux-arm64.tar.gz',
    'darwin/amd64'  => 'isrvd-darwin-amd64.tar.gz',
    'darwin/arm64'  => 'isrvd-darwin-arm64.tar.gz',
    'windows/amd64' => 'isrvd-windows-amd64.tar.gz',
    'windows/arm64' => 'isrvd-windows-arm64.tar.gz',
];

// ── 入口 ──────────────────────────────────────────────────

/**
 * 校验客户端参数，并返回版本信息和匹配平台的升级包地址。
 */
function main(): void
{
    $ver  = $_GET['ver']  ?? '';
    $os   = $_GET['os']   ?? '';
    $arch = $_GET['arch'] ?? '';

    if (!is_string($ver) || !is_string($os) || !is_string($arch)) {
        respondUpdate('', '', '', 'missing required parameters: ver, os, arch');
    }

    $ver  = trim($ver);
    $os   = trim($os);
    $arch = trim($arch);
    if (!$ver || !$os || !$arch) {
        respondUpdate('', '', '', 'missing required parameters: ver, os, arch');
    }

    $release = fetchLatestRelease();
    if ($release === null) {
        respondUpdate('', '', '', '', 'failed to fetch release info, please retry later');
    }

    $latest     = $release['tag_name'];
    $releaseUrl = $release['html_url'];
    $assets     = $release['assets'];

    // 当前版本无需升级时仅展示版本信息，不提供升级包地址。
    if (versionCompare($ver, $latest) >= 0) {
        respondUpdate($latest, $releaseUrl, '', 'already up to date');
    }

    // 根据操作系统和架构匹配升级包文件名。
    $key      = "$os/$arch";
    $filename = ASSET_MAP[$key] ?? null;

    if (!$filename) {
        respondUpdate($latest, $releaseUrl, '', '', "unsupported platform: $key");
    }

    $downloadUrl = '';
    foreach ($assets as $asset) {
        if (($asset['name'] ?? '') === $filename) {
            $downloadUrl = $asset['browser_download_url'] ?? '';
            break;
        }
    }

    if (!$downloadUrl) {
        respondUpdate($latest, $releaseUrl, '', '', "asset not found for platform: $key");
    }

    respondUpdate($latest, $releaseUrl, $downloadUrl, "update available: $latest");
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

/**
 * 构造 UpdateInfo JSON，并输出禁止缓存的动态响应。
 */
function respondUpdate(
    string $version,
    string $release,
    string $package,
    string $message,
    string $error = ''
): void {
    $content = json_encode([
        'version' => $version,
        'release' => $release,
        'package' => $package,
        'message' => $message,
        'error'   => $error,
    ], JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE | JSON_INVALID_UTF8_SUBSTITUTE);
    respond($content, 200, false);
}

/**
 * 获取经业务校验的最新 Release 数据，拉取失败时回退到有效旧缓存。
 */
function fetchLatestRelease(): ?array
{
    $content = fetchFile('latest.json', 'validateRelease');
    return $content === null ? null : json_decode($content, true);
}

/**
 * 校验 Release 版本号、发布地址和资源列表的字段类型。
 */
function validateRelease(string $content): bool
{
    $data = json_decode($content, true);
    if (!is_array($data)
        || !is_string($data['tag_name'] ?? null)
        || trim($data['tag_name']) === ''
        || !is_string($data['html_url'] ?? null)
        || !is_array($data['assets'] ?? null)) {
        return false;
    }

    foreach ($data['assets'] as $asset) {
        if (!is_array($asset)
            || !is_string($asset['name'] ?? '')
            || !is_string($asset['browser_download_url'] ?? '')) {
            return false;
        }
    }

    return true;
}

/**
 * 移除版本号开头的 v 前缀。
 */
function normalizeVer(string $v): string
{
    return ltrim($v, 'v');
}

/**
 * 按数字分段比较版本号，忽略 v 前缀，并将缺失分段视为 0。
 */
function versionCompare(string $a, string $b): int
{
    $pa = array_map('intval', explode('.', normalizeVer($a)));
    $pb = array_map('intval', explode('.', normalizeVer($b)));
    $len = max(count($pa), count($pb));

    for ($i = 0; $i < $len; $i++) {
        $va = $pa[$i] ?? 0;
        $vb = $pb[$i] ?? 0;
        if ($va > $vb) return 1;
        if ($va < $vb) return -1;
    }
    return 0;
}

main();
