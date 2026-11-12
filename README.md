# Jewel Game

A fantasy RPG where every fight is a match-3 duel. Pick a hero, travel the world map, take on
quests, and battle monsters on a shared gem board.

**Play it in your browser:** https://georgezack.github.io/jewel-game-web/

This is a vertical slice: the first story arc, eight main quests through to the Hollow King,
playable start to finish. The pixel art was made with the help of an AI image tool (Retro
Diffusion), and some of it is still placeholder. See [attributions.md](attributions.md).

## How to play

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


Losing a duel costs nothing, but you only keep what you picked up on the board. Progress saves
automatically (in the browser's local storage, or `~/.config/jewel-game/hero.json` on the desktop; what outlasts a
run goes in `profile.json` beside it).

**Specials:** 4 in a row makes a **Bolt** (clears its row or column), an L or T makes a **Bomb** (3x3),
and 5 in a row makes a **Prism** (swap it with a gem to clear that color). Swapping two specials
together combines them.

| Input | Action |
| --- | --- |
| Left drag, or click two gems | Swap |
| 1-6 | Cast spells (duels) |
| H | Hint |
| F1 | Debug log |
| Esc | Deselect / quit |

## Building

You need [Go](https://go.dev/doc/install). The game draws with [Ebitengine](https://ebitengine.org),
so the same code runs on the desktop and in a browser through WebAssembly. Desktop builds on Linux
need the X11/OpenGL development headers; the web build needs no C compiler.

```sh
./start.sh           # dev build with the map editor (E) and hot reload (or: go run -tags dev .)
./start.sh --watch   # rebuild and restart on save, needs CompileDaemon
./start.sh --dev     # browser dev build; rebuilds the wasm and reloads the page on save
./start.sh --web     # the public browser build (no editor), on http://localhost:8000
./deploy-web.sh      # publish the public build to GitHub Pages
go test ./...        # game, RPG and story checks, no window needed
```

Images, fonts and `assets/` are embedded in the binary, but files on disk take priority, so dev
builds pick up edits live.

**Writing the story:** the world, foes, quests, characters, dialogue and cutscenes are YAML files in
`assets/story/`, with art in its `portraits/`, `backgrounds/` and `foes/` folders. They reload live
on the desktop. See [docs/story.md](docs/story.md) for the format. The wording of menus, spells,
items and classes is in `assets/lang/en.yaml` (set the language in `assets/lang.yaml`). The
numbers behind those spells and items stay in `rpg/`.

Gem looks come from `assets/gemtypes.yaml` and reload live when it changes.

## Layout

- `board/` pure game rules (matching, specials, gravity, locks); unit tested
- `rpg/` RPG rules (duels, spells, heroes, quests, saves) and the story loader; unit tested
- `assets/story/` the story as data, documented in `docs/story.md`
- `assets/lang/` the interface language files
- `core/` animation, input, and the game loop around a board
- `entities/` gem drawing
- `ui/` widgets and the debug log
- `rl/` the few raylib-style drawing and input calls the game uses, implemented on Ebitengine
- `web/` the page that loads the WebAssembly build

