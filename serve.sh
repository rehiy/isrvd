#!/bin/sh
#

go run ./main.go &

cd webview && npm run dev
