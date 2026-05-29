#!/bin/sh
#

set -e

rm -rf dist

###########################################
# 尝试获取版本号
###########################################

last_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
prev_tag=$(git tag | sort -V | tail -n 2 | head -n 1)

echo "==> build version: $last_tag"

if [ -n "$last_tag$prev_tag" ]; then
    git log $prev_tag..$last_tag --pretty=format:"%s" | grep -v "^release" | sed 's/^/- /' | sort > RELEASE.md
fi

###########################################
# 替换版本号
###########################################

sed -i "s/Version = \".*\"/Version = \"$last_tag\"/" config/config.go

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
        -ldflags="-s -w -buildid=" -o "$target" cmd/server/main.go
}

###########################################
# 前端构建
###########################################

cd webview/

npm ci
npm run build

cd ../

###########################################
# 编译后端
###########################################

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

cp config.yml dist/
cp build/script/isrvd.sh dist/isrvd.sh

cd dist/
for app in isrvd-*; do
    archive="${app%.exe}.tar.gz"
    echo "==> packing $archive"
    tar czf "$archive" "$app" config.yml isrvd.sh
    rm -f "$app"
done

rm -f config.yml isrvd.sh
cd ../
