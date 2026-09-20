#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

export CGO_ENABLED=0

build() {
  local goos="$1" goarch="$2" dest="$3" bin="$4"
  mkdir -p "$dest"
  echo "building ${goos}/${goarch} -> ${dest}/${bin}"
  GOOS="$goos" GOARCH="$goarch" go build -o "${dest}/${bin}" ./cmd/skillctl
  if [[ "$bin" != *.exe ]]; then
    chmod +x "${dest}/${bin}"
  fi
}

build darwin arm64 npm/skillctl-darwin-arm64/bin skillctl
build darwin amd64 npm/skillctl-darwin-amd64/bin skillctl
build linux amd64 npm/skillctl-linux-amd64/bin skillctl
build linux arm64 npm/skillctl-linux-arm64/bin skillctl
build windows amd64 npm/skillctl-windows-amd64/bin skillctl.exe

chmod +x npm/skillctl/bin/skillctl.js

if [[ "${1:-}" == "pack" ]]; then
  for dir in npm/skillctl-darwin-arm64 npm/skillctl-darwin-amd64 npm/skillctl-linux-amd64 npm/skillctl-linux-arm64 npm/skillctl-windows-amd64 npm/skillctl; do
    echo "packing ${dir}"
    (cd "$dir" && npm pack)
  done
fi
