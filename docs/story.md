# Writing the story

The Quest mode's world, monsters, quests, dialogue and cutscenes are plain text files in
`assets/story/`. You can change the story without touching Go code.

- **Desktop:** run `./start.sh`. Saving a story file reloads it straight away. If a file has a
  mistake, the game keeps the last version that worked and shows the error in the left panel; press
  F1 for the details.
- **Web:** the story is built into `game.wasm`. `./start.sh --dev` rebuilds it and reloads the
  page when you save. `./start.sh --web` builds once and serves that.
- **Checking:** `go test ./rpg` loads the story and fails on any broken reference, such as a quest
  naming a foe that doesn't exist or a scene that `goto`s a misspelled scene. It also checks that
  the main quests can be finished in order.

## Language

Interface text (menus, spells, items, classes, the battle log) is in `assets/lang/en.yaml`.
Edit a value and save. On the desktop it reloads immediately. On the web, `./start.sh --dev`
rebuilds and reloads the page.

`assets/lang.yaml` names the language, `en` by default. To add one, copy `en.yaml` to
`fr.yaml`, translate the values (leave the keys), and set `language: fr`. A missing key keeps
the English from the code, so a translation can be partial. Patterns are Go format strings
(`%s`, `%d`); `%[2]s` swaps the order of arguments.

Anything in the story files (place names, foe names, dialogue, quest text) stays in
`assets/story/`. A language can replace the whole story by shipping
`assets/lang/<language>/story/story.yaml` plus the rest of that folder. Pictures not in the
translation fall back to `assets/story/`.

Ids do not get translated: class names stored in the save, spell ids, item ids, and the `name`
of a gem type. Change the words next to those ids. The font is Noto Mono, which covers
Latin letters; another script needs a font that has those glyphs.

```
assets/story/
  story.yaml        title, start location, opening scene, final quest
  characters.yaml   who can speak in scenes
  locations.yaml    the world map
  foes.yaml         monsters
  quests.yaml       quests
  scenes/*.yaml     dialogue and cutscenes; split across as many files as you like
  portraits/        character art
  backgrounds/      cutscene backdrops
  foes/             monster art
```

## How the pieces fit

- **Locations** sit on the map, joined by **roads**. Some are open from the start; the rest open
  when a quest lists them in `opens`, when their `requires` conditions hold, or when a scene
  `unlock`s them.
- **Quests** are given out in a town (`giver`) and ask you to beat a foe some number of times at a
  location (`at`). Finishing one pays out and opens new locations.
- **Scenes** are the scripted parts: conversations, choices, title cards, fades and rewards. They
  play when:
  - a hero is created (`opening` in story.yaml)
  - a quest is offered (`offer_scene`) or finished (`done_scene`)
  - the hero first arrives somewhere (`enter_scene`)
  - the player talks to someone (`people`)
- **Flags** are named on/off switches that scenes set, for remembering things like "met the mayor".
  They're saved with the hero, so later scenes and quests can check them.

Every thing has an `id`: lowercase, no spaces, e.g. `old_mill`. Other files refer to it by that
id, and saves store ids, so renaming an id breaks old saves' references to it. Change `name`
instead whenever you can.

## YAML in two minutes

```yaml
key: value                 # text doesn't need quotes...
title: "Yes: it does"      # ...unless it has ": " in it, or starts with  ! & * [ { # | > or a quote
list: [a, b, c]            # a short list
items:                     # a list of things, one per "- "
  - id: a
    name: First
  - id: b
    name: Second
long: >-                   # a long paragraph; line breaks become spaces
  This can go on
  over several lines.
```

Indentation is spaces, never tabs. The one trap in this format: a condition starting with `!` must
be quoted, `if: ["!flag:met"]`, because a bare `!` means something else in YAML.

## story.yaml

```yaml
title: The Hollow King
map_name: The Marches        # heading on the world map
start: emberfield            # where new heroes begin
opening: opening             # scene played when a hero is created
final_quest: king            # finishing this marks the story beaten
blurb: >-                    # shown next to the class picker
  Something in the north has every creature in the land running south.
```

## characters.yaml

```yaml
- id: mayor
  name: Mayor Aldren
  color: [214, 170, 90]      # red, green, blue (0-255): the name's color and the placeholder portrait
  portrait: aldren           # optional: image name in portraits/, defaults to the id
```

`hero` always means the player and doesn't need an entry.

## locations.yaml

```yaml
- id: stonebridge
  name: Stonebridge
  town: true                 # towns have a shop and give quests
  open: false                # on the map from the start?
  requires: [flag:heard_of_bridge]   # optional: opens once these all hold
  x: 0.38                    # position on the map, 0 = left/top, 1 = right/bottom
  y: 0.74
  level: 2                   # level of the roaming foes here
  desc: A toll town grown fat on the river trade.
  roads: [mossgate, barrow]  # roads go both ways: list this one in theirs too
  foes: [bandit, rat]        # roaming foes; also who ambushes travellers on the way in
  enter_scene: stonebridge_arrival   # optional: plays the first time the hero arrives
  people:                    # optional: "Talk to ..." buttons
    - name: Tollkeeper Brann
      scene: brann_talk
      requires: [quest:toll] # optional: only there once these hold
```

## foes.yaml

```yaml
- id: goblin
  name: Goblin
  shape: humanoid            # placeholder drawing: blob, beast, humanoid, ghost, brute, dragon
  color: [120, 170, 70]
  image: goblin_chief        # optional: picture in foes/, defaults to the id
  hp_pct: 85                 # health: this % of (30 + 8 × level), plus 3 per morale (default 100)
  stats: { fire: 2, air: 2, battle: 2, cunning: 3 }   # at level 1; missing stats are 0
  growth: { fire: 2, water: 1, battle: 2 }            # added per 2 levels
  spells: [frenzy]
```

Stats are fire, water, air, earth, battle, cunning and morale, as on the Hero screen. The spell ids
are in `rpg/spells.go`. Monsters can use any spell, but they're balanced around the monster ones:
bite, venom, frenzy, roar, bone_ward, drain_life, mana_burn, firebreath and dark_pact.

`go test ./rpg` also runs a balance check that duels a geared hero against some of the bosses by
their ids. If you rename or remove those foes, update the list in `TestDuelsFinishAndAreFair`.

## quests.yaml

```yaml
- id: goblins
  title: Smoke on the Road
  main: true                 # story quest: starred in the list, and checked for reachability
  giver: emberfield          # the town that offers it
  requires: [quest:rats]     # optional: only offered once these hold
  at: mossgate               # where the fights are
  foe: goblin
  level: 1                   # the foe's level
  count: 2                   # wins needed (default 1)
  gold: 40
  xp: 50
  item: barrow_mail          # optional reward; item ids are in rpg/hero.go
  opens: [stonebridge]       # locations this opens
  intro: Goblins burned the Harlan farm last night...   # the offer text
  outro: The goblins weren't raiding. They were running.  # shown when it's done
  offer_scene: goblins_offer # optional: play this instead of the plain offer box
  done_scene: goblins_done   # optional: play this after the "Quest complete" box
```

An `offer_scene` replaces the box with the Accept button, so the scene has to accept the quest
itself with an `accept_quest` step, usually inside a choice. The plain box also lists the foe,
count and reward; with an `offer_scene`, say those in the dialogue.

## Scenes

A scenes file maps scene ids to lists of steps. Steps play in order, and each step does whatever
its fields say:

```yaml
mayor_intro:
  - fade_in: 1.0
  - background: town_square
  - who: mayor
    mood: worried
    text: You're awake. Good.
  - who: hero
    text: What happened?
  - text: Somewhere a bell starts ringing.      # no "who": narration
  - set_flag: met_mayor
```

The player clicks, or presses Space or Enter, to move past a line. The first click finishes the
typing, the next moves on. **Skip** runs ahead to the next choice. Everything a skipped scene does
still happens, such as flags, rewards and duels.

### Dialogue

| Field | Meaning |
|---|---|
| `text` | the line; without `who` it's narration |
| `who` | a character id, or `hero` |
| `mood` | picks the portrait `<portrait>_<mood>.png` if it exists |
| `choices` | buttons shown under the line (see below) |

```yaml
  - who: mayor
    text: Will you help us?
    choices:
      - text: I'll go.
        do:                          # steps run when picked
          - accept_quest: goblins
          - who: mayor
            text: Thank you.
      - text: Tell me more first.
        goto: mayor_details          # continue with another scene
      - text: (Show the seal)
        if: [item:royal_seal]        # only offered when this holds
        goto: mayor_seal
      - text: Not now.               # no do/goto: the scene just carries on
```

### Presentation

| Field | Meaning |
|---|---|
| `background: name` | full-screen image from backgrounds/; `none` removes it |
| `fade_in: 1.5` | fade from black over 1.5 seconds |
| `fade_out: 1.0` | fade to black; the next `fade_in` brings the picture back |
| `wait: 0.5` | pause |
| `title: ...`, `subtitle: ...` | a title card; it goes on a click or after a few seconds |

Without a background, scenes play over a darkened game screen. Backgrounds and fades are cleared
when a scene ends.

### Changes to the hero

| Field | Meaning |
|---|---|
| `set_flag: name` / `clear_flag: name` | story switches |
| `gold: 25` | give gold (negative takes it, down to 0) |
| `xp: 100` | give experience |
| `item: steel_sword` | give an item |
| `accept_quest: id` | start a quest |
| `unlock: id` | open a location on the map |
| `travel: id` | move the hero there |
| `end: true` | mark the story beaten |

### Flow

| Field | Meaning |
|---|---|
| `goto: scene` | stop this scene and play that one instead |
| `run: scene` | play that scene, then come back here |
| `if: [...]` with `then:` / `else:` | branch on conditions |
| `if: [...]` alone | the rest of the step only happens when the conditions hold |
| `duel: { foe: id, level: 3, win: scene, lose: scene }` | fight, then continue |

```yaml
bessa_talk:
  - if: [quest:barrow_lord]
    then:
      - who: bessa
        text: They say you put the Barrow Lord back in his hole.
    else:
      - who: bessa
        text: In seventy years I've never seen a goblin run.

ambush_at_the_mill:
  - who: bandit
    text: Your money or your life.
  - duel: { foe: bandit, level: 3, win: mill_won, lose: mill_lost }
```

A duel's `level` defaults to the hero's level. Without `win`, a win carries on with the next step.
Without `lose`, a loss ends the scene, so the player can come back and try again.

### Conditions

Used by `requires`, `if` and choices. A list means all of them must hold, and a `!` in front means
"not" (quote it: `"!flag:met"`).

| Condition | True when |
|---|---|
| `quest:id` | the quest is finished |
| `active:id` | the quest is in progress |
| `flag:name` | a scene has set the flag |
| `seen:scene` | the scene has played before |
| `item:id` | the hero owns the item |
| `level:5` | the hero is level 5 or higher |
| `class:knight` | the hero's class |
| `at:location` | the hero is there |

## Art

Put images in the folders and name them after what they're for. Any size works; PNG and JPG are
both fine.

| File | Used for |
|---|---|
| `portraits/<character>.png` | a character's dialogue portrait (or the `portrait` name) |
| `portraits/<character>_<mood>.png` | the same character with a `mood:`, e.g. `mayor_worried.png` |
| `portraits/hero_<class>.png`, `portraits/hero.png` | the player, e.g. `hero_knight.png` |
| `backgrounds/<name>.png` | `background: <name>` in scenes |
| `foes/<foe id>.png` | the monster in duels (or the foe's `image` name) |

Missing art isn't an error. The game draws a placeholder and uses the picture as soon as the file
appears. A missing mood falls back to the plain portrait. Images are scaled with
nearest-neighbour filtering, so pixel art such as Retro Diffusion output stays crisp; it looks best
at whole-number scales. Portraits are shown in a 200×200 box and foes in 130×130, while
backgrounds fill the window and crop to keep their shape.

## Still in Go code

These parts aren't story files yet, and changing them means editing Go:

- classes and their spells: `rpg/hero.go`, `rpg/spells.go`
- items, the shop and citadel buildings: `rpg/hero.go`
- duel rules and balance numbers: `rpg/duel.go`
- the scene player: `core/scene-view.go`; the loader and the checks: `rpg/story.go`

### Adding a new kind of step

1. Add a field to `Step` in `rpg/story.go`, with its YAML name in the tag.
2. If it refers to something by id, check it in `checkSteps` so mistakes are caught.
3. Make it do its thing in `SceneRunner.exec` in `core/scene-view.go`.
4. Document it here.
