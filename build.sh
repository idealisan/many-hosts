#!/usr/bin/env bash
# macOS / Linux 构建：无需 manifest 与 Windows GUI 子系统参数。
set -euo pipefail

export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"
export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
export GODEBUG="${GODEBUG:-http2client=0}"
export CGO_ENABLED=0

go build -o manyhosts .
