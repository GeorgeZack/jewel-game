# jewel-game
A **Bejewelled** clone.

## Requirements

- `go` to build the game from scratch

## Running

```sh
./start.sh           # build and run (or: go run .)
./start.sh --watch   # rebuild and restart on save, needs CompileDaemon
go test ./...        # game and RPG rules, no window needed
./start.sh --web     # browser build, served on http://localhost:8000
./start.sh --dev     # same, and rebuild the wasm and reload the page on save
```

The game draws with [Ebitengine](https://ebitengine.org), so the same code runs on the desktop and in a
browser through WebAssembly. Desktop builds on Linux need the X11/OpenGL development headers; there is
no C compiler needed for the web build. Images, fonts and `assets/` are embedded in the binary, but files
on disk take priority, so editing `assets/gemtypes.yaml` still hot reloads on the desktop. In a browser the
hero is saved to `localStorage` instead of `~/.config/jewel-game/hero.json`.

The background photo is stored with Git LFS (`git lfs pull`); without it the game uses a gradient.

## How to play

| Input | Action |
| --- | --- |
| Left drag, or click two gems | Swap |
| Right click | Lock / unlock a gem (not in duels) |
| Space | Release every lock at once |
| H | Hint |
| R | New board (not in duels) |
| Z / Backspace | Undo (sandbox) |
| Enter | Clear matches sitting on the board (sandbox) |
| Tab | Switch Classic / Sandbox / Quest |
| 1-6 | Cast spells (duels) |
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

**Quest:** an RPG in the style of Puzzle Quest: Challenge of the Warlords, with its own world, story and art.

- *Duels:* you and a monster take turns on one 8x8 board. Red, blue, yellow and green gems fill your
  Fire, Water, Air and Earth mana. Skulls damage the foe, and +5 skulls hit harder and blast the gems
  around them. Gold and stars pay out after the duel. Whatever the other side's move sets up, they get to use.
- *Extra turns and Wilds:* match 4 and you go again. Match 5 and you also get a Wild, which completes
  any mana match and multiplies it. Running out of moves drains all mana and reshuffles.
- *Heroes:* Knight, Warrior, Druid or Wizard, each with six spells unlocked by level (six can be
  equipped). Levels give points for Fire, Water, Air, Earth, Battle, Cunning and Morale.
- *World:* walk the map between towns and wilds, take main and side quests, fight roaming foes and
  road ambushes. Quests open up the map through to the final boss.
- *Spending gold:* gear in five slots from town shops, mounts, and permanent citadel buildings.
- *Story:* dialogue with portraits and choices, title cards, people to talk to in towns, and story
  flags that quests and scenes can check.

Losing a duel costs nothing but keeps only what you picked up on the board. The hero is saved to
`~/.config/jewel-game/hero.json`.

**Writing the story:** the world, foes, quests, characters, dialogue and cutscenes are YAML files in
`assets/story/`, with art in its `portraits/`, `backgrounds/` and `foes/` folders. They reload live
on the desktop. See [docs/story.md](docs/story.md) for the format. Classes, spells and items are
still Go tables in `rpg/`.

Gem looks come from `assets/gemtypes.yaml` and reload live when it changes.

## Layout

- `board/` pure game rules (matching, specials, gravity, locks); unit tested
- `rpg/` RPG rules (duels, spells, heroes, quests, saves) and the story loader; unit tested
- `assets/story/` the story as data, documented in `docs/story.md`
- `core/` animation, input, and the game loop around a board
- `entities/` gem drawing
- `ui/` widgets and the debug log
- `rl/` the few raylib-style drawing and input calls the game uses, implemented on Ebitengine
- `web/` the page that loads the WebAssembly build

