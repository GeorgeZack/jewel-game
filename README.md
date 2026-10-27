# jewel-game
A **Bejewelled** clone.

## Requirements

- `go` to build the game from scratch

## Running

```sh
./start.sh           # build and run (or: go run .)
./start.sh --watch   # rebuild and restart on save, needs CompileDaemon
go test ./board      # game rules, no window needed
```

The background photo is stored with Git LFS (`git lfs pull`); without it the game uses a gradient.

## How to play

| Input | Action |
| --- | --- |
| Left drag, or click two gems | Swap |
| Right click | Lock / unlock a gem |
| Space | Release every lock at once |
| H | Hint |
| R | New board |
| Z / Backspace | Undo (sandbox) |
| Enter | Clear matches sitting on the board (sandbox) |
| Tab | Switch Classic / Sandbox |
| F1 | Debug log |
| Esc | Drop brush / deselect / quit |

**Specials:** 4 in a row makes a **Bolt** (clears its row or column), an L or T makes a **Bomb** (3x3),
5 in a row makes a **Prism** (swap it with a gem to clear that color). Swapping two specials together
combines them; Prism + Bolt/Bomb turns a whole color into that special.

**Locks:** a locked gem hangs in place: it doesn't fall, match, or break, and other gems fall past it.
Lock pieces of several lines, finish them, then release them all together. Making more than one
special in a single step pays a Synchro bonus.

**Sandbox:** presets (Chill / Playground / Insane), board size and color count, lock limit, free swaps,
diagonal swaps, overcharged specials, gravity direction, chaos gravity, special rain, animation speed,
and a paint brush for placing any gem or special by hand.

Gem looks come from `assets/gemtypes.yaml` and reload live when it changes.

## Layout

- `board/` pure game rules (matching, specials, gravity, locks); unit tested
- `core/` animation, input, and the game loop around a board
- `entities/` gem drawing
- `ui/` widgets and the debug log

