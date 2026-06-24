<?php
/**
 * iSrvd 自动升级服务端
 * 部署于 https://isrvd.rehiy.com/update/
 *
 * 请求参数（GET）：
 *   ver   当前版本，如 v1.0.0
 *   os    运行平台，如 linux / darwin / windows
 *   arch  CPU 架构，如 amd64 / arm64
 *
 * 响应（UpdateInfo JSON）：
 *   version  最新版本号（始终返回）
 *   release  GitHub Release 页面 URL（始终返回）
 *   package  下载地址（有对应 asset 且有新版时返回，否则为空）
 *   message  提示信息
 *   error    错误信息（非空时客户端终止更新）
 */

header('Content-Type: application/json; charset=utf-8');

// ── 常量配置 ──────────────────────────────────────────────

const CACHE_FILE    = __DIR__ . '/latest.json';
const CACHE_TTL     = 3600; // 1 小时

const GITHUB_REPO   = 'rehiy/isrvd';
const GITHUB_API    = 'https://api.github.com/repos/' . GITHUB_REPO . '/releases/latest';

// 平台/架构到 Release Asset 文件名的映射
const ASSET_MAP = [
    'linux/amd64'   => 'isrvd-linux-amd64.tar.gz',
    'linux/arm64'   => 'isrvd-linux-arm64.tar.gz',
    'darwin/amd64'  => 'isrvd-darwin-amd64.tar.gz',
    'darwin/arm64'  => 'isrvd-darwin-arm64.tar.gz',
    'windows/amd64' => 'isrvd-windows-amd64.tar.gz',
    'windows/arm64' => 'isrvd-windows-arm64.tar.gz',
];

// ── 入口 ──────────────────────────────────────────────────

function main(): void
{
    $ver  = trim($_GET['ver']  ?? '');
    $os   = trim($_GET['os']   ?? '');
    $arch = trim($_GET['arch'] ?? '');

    if (!$ver || !$os || !$arch) {
        respond('', '', '', 'missing required parameters: ver, os, arch');
        return;
    }

    $release = fetchLatestRelease();
    if (!$release) {
        respond('', '', '', '', 'failed to fetch release info, please retry later');
        return;
    }

    $latest     = $release['tag_name'] ?? '';
    $releaseUrl = $release['html_url']  ?? '';
    $assets     = $release['assets']    ?? [];

    if (!$latest) {
        respond('', '', '', '', 'no release tag found');
        return;
    }

    // 始终返回最新版本信息，由客户端判断是否需要更新
    // package 为空时客户端只做版本展示，不执行下载
    if (versionCompare($ver, $latest) >= 0) {
        respond($latest, $releaseUrl, '', 'already up to date');
        return;
    }

    // 查找对应平台的下载地址
    $key      = "$os/$arch";
    $filename = ASSET_MAP[$key] ?? null;

    if (!$filename) {
        respond($latest, $releaseUrl, '', '', "unsupported platform: $key");
        return;
    }

    $downloadUrl = '';
    foreach ($assets as $asset) {
        if (($asset['name'] ?? '') === $filename) {
            $downloadUrl = $asset['browser_download_url'] ?? '';
            break;
        }
    }

    if (!$downloadUrl) {
        respond($latest, $releaseUrl, '', '', "asset not found for platform: $key");
        return;
    }

    respond($latest, $releaseUrl, $downloadUrl, "update available: $latest");
}

// ── 工具函数 ──────────────────────────────────────────────

function respond(
    string $version,
    string $release,
    string $package,
    string $message,
    string $error = ''
): void {
    echo json_encode([
        'version' => $version,
        'release' => $release,
        'package' => $package,
        'message' => $message,
        'error'   => $error,
    ], JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE);
    exit;
}

/**
 * 从 GitHub API 获取最新 Release，结果缓存 CACHE_TTL 秒
 * GitHub 请求失败时返回旧缓存（若有），确保客户端始终能得到版本信息
 */
function fetchLatestRelease(): ?array
{
    $cached = null;

    if (file_exists(CACHE_FILE)) {
        $cached = json_decode(file_get_contents(CACHE_FILE), true) ?: null;
        if ($cached && (time() - filemtime(CACHE_FILE)) < CACHE_TTL) {
            return $cached;
        }
    }

    $ctx = stream_context_create([
        'http' => [
            'method'          => 'GET',
            'header'          => implode("\r\n", [
                'Accept: application/vnd.github+json',
                'User-Agent: iSrvd-update-server/1.0',
                'X-GitHub-Api-Version: 2022-11-28',
            ]),
            'timeout'         => 10,
            'ignore_errors'   => true,
        ],
    ]);

    $body = @file_get_contents(GITHUB_API, false, $ctx);

    $status = 0;
    foreach ($http_response_header ?? [] as $h) {
        if (preg_match('#^HTTP/\S+\s+(\d+)#', $h, $m)) {
            $status = (int) $m[1];
        }
    }

    if (!$body || $status !== 200) {
        return $cached;
    }

    $data = json_decode($body, true);
    if (!$data) {
        return $cached;
    }

    $cacheDir = dirname(CACHE_FILE);
    if (!is_dir($cacheDir)) {
        mkdir($cacheDir, 0755, true);
    }
    file_put_contents(CACHE_FILE, $body, LOCK_EX);

    return $data;
}

function normalizeVer(string $v): string
{
    return ltrim($v, 'v');
}

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
