#!/bin/bash
# Build and run the game.
#   ./start.sh           build once and run
#   ./start.sh --watch   rebuild and restart on every save (needs CompileDaemon)
set -e

# assets are loaded relative to the project root
cd "$(dirname "$0")"

if ! command -v go >/dev/null; then
	echo "go is not installed: https://go.dev/doc/install" >&2
	exit 1
fi

if head -c 40 _res/bkg.jpg 2>/dev/null | grep -q "git-lfs"; then
	echo "Note: background image not downloaded, run 'git lfs pull' to get it (a gradient is used meanwhile)."
fi

case "$1" in
	-w|--watch)
		if ! command -v CompileDaemon >/dev/null; then
			echo "CompileDaemon not found. Install it with:" >&2
			echo "  go install github.com/githubnemo/CompileDaemon@latest" >&2
			echo "and make sure \$(go env GOPATH)/bin is on your PATH." >&2
			exit 1
		fi
		exec CompileDaemon -directory="." -recursive=true -exclude-dir=".git" -log-prefix=false -color=true \
			-build="go build -o jewel-game ." -command="./jewel-game"
		;;
	"")
		echo "Building (the first build compiles raylib and takes a little while)..."
		go build -o jewel-game .
		exec ./jewel-game
		;;
	*)
		echo "usage: $0 [--watch]" >&2
		exit 1
		;;
esac
