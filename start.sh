#!/bin/bash
# Build and run the game.
#   ./start.sh           build once and run, with the map editor and hot reload
#   ./start.sh --watch   rebuild and restart on every save (needs CompileDaemon)
#   ./start.sh --web     build the public browser version (no editor) and serve it on http://localhost:8000 (PORT to change)
#   ./start.sh --dev     same as --web, and rebuild the wasm (then reload the page) on save
set -e

# assets are loaded relative to the project root
cd "$(dirname "$0")"

# A GOROOT left in the environment by a version manager can point at a different Go than the `go`
# on the PATH, and then nothing compiles. Go finds its own root when this is unset.
unset GOROOT

if ! command -v go >/dev/null; then
	echo "go is not installed: https://go.dev/doc/install" >&2
	exit 1
fi

watch=0
web=0
dev=0
for arg in "$@"; do
	case "$arg" in
		-w|--watch) watch=1 ;;
		--web) web=1 ;;
		--dev) dev=1 ;;
		*)
			echo "usage: $0 [--watch | --web | --dev]" >&2
			exit 1
			;;
	esac
done

if [ "$watch" = 1 ] && [ "$web$dev" != 00 ]; then
	echo "use --watch for the desktop app, or --dev for the browser" >&2
	exit 1
fi

if [ "$watch" = 1 ]; then
	if ! command -v CompileDaemon >/dev/null; then
		echo "CompileDaemon not found. Install it with:" >&2
		echo "  go install github.com/githubnemo/CompileDaemon@latest" >&2
		echo "and make sure \$(go env GOPATH)/bin is on your PATH." >&2
		exit 1
	fi
	exec CompileDaemon -directory="." -recursive=true -exclude-dir=".git" -exclude-dir="web" -log-prefix=false -color=true \
		-build="go build -tags dev -o jewel-game ." -command="./jewel-game"
fi

if [ "$dev" = 1 ]; then
	if ! command -v python3 >/dev/null; then
		echo "python3 is needed to serve the page." >&2
		exit 1
	fi
	exec python3 devserver.py
fi

if [ "$web" = 1 ]; then
	if ! command -v python3 >/dev/null; then
		echo "python3 is needed to serve the page; or serve the web/ folder with any static server." >&2
		exit 1
	fi
	./build-web.sh
	echo "Open http://localhost:${PORT:-8000}  (Ctrl+C to stop; rerun ./build-web.sh and reload after code changes)"
	JEWEL_WATCH=0 exec python3 devserver.py
fi

echo "Building (the first build takes a little while)..."
go build -tags dev -o jewel-game .
exec ./jewel-game
