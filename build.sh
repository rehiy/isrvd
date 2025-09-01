#!/bin/sh
#

set -e
set -o noglob

###########################################

export CGO_ENABLED=0
export GO111MODULE=on

build() {
    echo building for $1/$2
    target=build/isrvd-$1-$2
    if [ x"$1" = x"windows" ]; then
        target="${target}.exe"
    fi
    GOOS=$1 GOARCH=$2 go build -ldflags="-s -w" -o $target main.go
}

####################################################################

cd front/

npm i
npm run build

cd ../

####################################################################

build linux amd64
build linux arm64

####################################################################

for app in `ls build`; do
    gzip build/$app
done
