# The Halloran Engine

A Victorian occult RPG where every fight is a match-3 duel. You walk a map, take quests, and match reagents on a shared board.

A work in progress, playable start to finish. You arrive alone in Aldermoor with a key and a dead teacher's letter, three rival societies court you, and the dead below the streets lead down to an Engine. Some of the pixel art is still placeholder ([attributions.md](attributions.md)).

Play it in the browser: https://georgezack.github.io/jewel-game-web/

## Run it

[Go 1.25](https://go.dev/doc/install) or newer. On Linux, the desktop build needs the X11 and OpenGL development headers.

```sh
./start.sh          # desktop, with the map editor
./start.sh --web    # browser build at http://localhost:8000
go test ./...
```

Go, [Ebitengine](https://ebitengine.org/), and WebAssembly.
