#!/bin/sh
# Builds the browser version and publishes it to https://georgezack.github.io/jewel-game-web/
# The public repo holds only the built game; each deploy replaces its contents.
set -e
cd "$(dirname "$0")"
REPO=GeorgeZack/jewel-game-web

./build-web.sh
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
cp web/index.html web/wasm_exec.js web/game.wasm "$tmp"/
touch "$tmp/.nojekyll" # serve files as-is

src=$(git rev-parse --short HEAD)
cd "$tmp"
git init -q -b main
git add -A
git commit -qm "Build from $src"
git push -qf "git@github.com:$REPO.git" main

# turn Pages on the first time
if ! gh api "repos/$REPO/pages" >/dev/null 2>&1; then
	gh api -X POST "repos/$REPO/pages" -f "source[branch]=main" -f "source[path]=/" >/dev/null
fi
echo "Deployed $src: https://georgezack.github.io/jewel-game-web/ (may take a minute to update)"
