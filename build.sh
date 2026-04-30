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

sed -i "s/Version = \".*\"/Version = \"$last_tag\"/" config/load.go

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
    GOOS=$1 GOARCH=$2 go build -ldflags="-s -w" -o "$target" cmd/server/main.go
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

build linux amd64
build linux arm64

###########################################
# 打包分发
###########################################

cp config.yml dist/

cd dist/
for app in isrvd-*; do
    archive="${app}.tar.gz"
    echo "==> packing $archive"
    tar czf "$archive" "$app" config.yml
    rm -f "$app"
done

rm -f config.yml
cd ../
