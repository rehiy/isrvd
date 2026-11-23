#!/bin/sh
#

set -e

rm -rf dist
rm -rf public/assets
rm -rf public/index.html

###########################################
# 尝试获取版本号
###########################################

last_tag=$(git describe --tags --abbrev=0 2>/dev/null || true)
build_version=${last_tag:-v0.0.0}
release_range=${last_tag:-HEAD}
if [ -n "$last_tag" ]; then
    prev_tag=$(git describe --tags --abbrev=0 "${last_tag}^" 2>/dev/null || true)
    if [ -n "$prev_tag" ]; then
        release_range="$prev_tag..$last_tag"
    fi
fi

echo "==> build version: $build_version"
release_subjects=$(git log --no-merges --format='%s' "$release_range" --)
printf '%s\n' "$release_subjects" | awk '
    /^release/ || /^Merge pull request #[0-9]+ from / || /^$/ { next }
    { print "- " $0 }
' | sort > RELEASE.md

###########################################
# Go 编译环境
###########################################

export CGO_ENABLED=0
export GO111MODULE=on

build() {
    echo "==> building for $1/$2"
    target="dist/isrvd-$1-$2"
    if [ x"$1" = x"windows" ]; then
        target="${target}.exe"
    fi
    GOOS=$1 GOARCH=$2 go build \
        -tags netgo -trimpath -buildvcs=false \
        -ldflags="-s -w -buildid= -X isrvd/server/config.Version=$build_version" \
        -o "$target" ./server/cmd/server
}

###########################################
# 前端构建
###########################################

echo ""
echo "==> 构建前端"

cd webview/

npm ci
npm run build

cd ../

echo ""
echo "==> 复制前端构建结果"
cp -a webview/dist/* public/

###########################################
# 生成 OpenAPI 文档
###########################################

echo ""
echo "==> 生成 OpenAPI 文档"
go run ./server/cmd/openapi-gen/ -o public/openapi/data.json

###########################################
# 编译后端
###########################################

echo ""
echo "==> 编译后端"

# Linux
build linux amd64
build linux arm64

# macOS
build darwin amd64
build darwin arm64

# Windows
build windows amd64
build windows arm64

###########################################
# 打包分发
###########################################

echo ""
echo "==> 打包分发"

cp config.yml dist/
cp scripts/isrvd-system.sh scripts/isrvd-docker.sh dist/

cd dist/
for app in isrvd-linux-* isrvd-darwin-* isrvd-windows-*.exe; do
    [ -f "$app" ] || continue
    case "$app" in *.tar.gz) continue ;; esac
    archive="${app%.exe}.tar.gz"
    echo "==> packing $archive"
    tar czf "$archive" "$app" config.yml isrvd-system.sh isrvd-docker.sh
    rm -f "$app"
done

rm -f config.yml isrvd-system.sh isrvd-docker.sh
cd ../
