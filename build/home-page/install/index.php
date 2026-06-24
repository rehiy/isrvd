<?php
/**
 * 代理 / 缓存 isrvd.sh 安装脚本
 * 部署于 https://isrvd.rehiy.com/install/
 *
 * 用法：放到任意 PHP 可访问目录，访问该文件即可获取 isrvd.sh 内容。
 * 缓存 12 小时，过期后重新拉取。
 */

$url      = 'https://raw.githubusercontent.com/rehiy/isrvd/refs/heads/master/build/script/isrvd.sh';
$cacheDir = __DIR__ . '/cache';
$cacheFile = $cacheDir . '/isrvd.sh';
$cacheTTL  = 12 * 3600; // 12 小时

// 确保缓存目录存在
if (!is_dir($cacheDir)) {
    mkdir($cacheDir, 0755, true);
}

// 检查缓存是否有效
if (file_exists($cacheFile) && (time() - filemtime($cacheFile)) < $cacheTTL) {
    $content = file_get_contents($cacheFile);
} else {
    $ctx = stream_context_create([
        'http' => [
            'timeout'    => 15,
            'user_agent' => 'isrvd-cache-proxy/1.0',
        ],
    ]);

    $content = @file_get_contents($url, false, $ctx);

    if ($content === false) {
        // 拉取失败时，如果有旧缓存则降级使用
        if (file_exists($cacheFile)) {
            $content = file_get_contents($cacheFile);
        } else {
            http_response_code(502);
            header('Content-Type: text/plain');
            die("Error: Unable to fetch $url and no cached copy available.\n");
        }
    } else {
        file_put_contents($cacheFile, $content);
    }
}

// 输出
header('Content-Type: text/plain; charset=utf-8');
header('Content-Length: ' . strlen($content));
header('Cache-Control: public, max-age=' . $cacheTTL);
echo $content;
