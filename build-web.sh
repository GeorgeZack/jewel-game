#!/bin/sh
# Builds the browser version into web/. TAGS=dev includes the map editor. Serve that folder over HTTP, e.g.: python3 -m http.server -d web
set -e
cd "$(dirname "$0")"
tmp=web/game.wasm.tmp
if ! GOOS=js GOARCH=wasm go build -tags "${TAGS:-}" -trimpath -ldflags="-s -w" -o "$tmp" .; then
	rm -f "$tmp"
	exit 1
fi
mv -f "$tmp" web/game.wasm
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/
echo "Built web/game.wasm ($(du -h web/game.wasm | cut -f1))"
