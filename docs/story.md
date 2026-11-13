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
  cards.yaml        rule cards: a different board or payouts for a duel (optional)
  orrery.yaml       the Orrery: the upgrade grid heroes buy with residue (optional)
  facilities.yaml   the academy's facilities, built and upgraded with gold (optional)
  puzzles/*.yaml    puzzle boards: authored boards with an objective and routes, one per file
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
  residue: water             # color of the residue chapters here pay: fire, water, air, earth or aether
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
  card: sideways             # optional: the rule card (cards.yaml) this foe fights on
```

Stats are fire, water, air, earth, aether, battle, cunning and morale, as on the Hero screen. The
spell ids are in `rpg/spells.go`. Monsters can use any spell, but they're balanced around the monster
ones: bite, venom, frenzy, roar, bone_ward, drain_life, mana_burn, firebreath, dark_pact, and the two
that cost Aether, hollow_hymn and void_lash. A foe's Aether stat and spells only matter in a run that
has Aether (see [Aether](#aether)); without it the foe never gets Aether mana. In the slice the
Drowned Priest and the Hollow King have both.

`go test ./rpg` also runs a balance check that duels a geared hero against some of the bosses by
their ids. If you rename or remove those foes, update the list in `TestDuelsFinishAndAreFair`.

## cards.yaml

A rule card changes the board a duel is played on, or what each cascade pays. A foe names the
card it fights on with `card:`; a foe without one gets the standard duel, an 8x8 board falling
down (9x9 once the run has Aether). Anything a card leaves out stays as in the standard duel, so
an empty card changes nothing. Every card needs a `name` and a `desc`, its rule in one sentence:
the duel opens with both in a box, and shows them beside the foe while it lasts.

```yaml
- id: sideways
  name: Sideways
  desc: Gems fall to the left, and new ones arrive from the right.
  width: 10                  # board size, 4 to 10 (default 8, or 9 with Aether); it wins over Aether's size
  height: 8
  gravity: left              # the way gems fall: down (default), left, up or right
  chaos_gravity: true        # gravity turns at random after every move
  overcharge: true           # bigger bombs, wider bolts
  gold_pct: 50               # payout modifiers, in percent: 50 pays half again, -50 half
  xp_pct: 0
  mana_pct: -25              # mana from matches, before the pool caps it
  skull_pct: 20              # skull damage
```

The modifiers apply to what the board pays during the duel, for both sides, not to the bounty for
winning. `TestCardsStayFair` in `rpg/perks_test.go` plays every card, with and without Aether,
against the same matchups as the standard duel and fails a card that moves the win rate too far
or drags (`JEWEL_FULL_SIM=1` for the full run).

The slice's bosses fight on these cards, gentler first:

| foe | card | rule |
|---|---|---|
| barrow_lord | `barrow_hush` | skulls deal 25% less, mana matches pay 50% more |
| warchief | `war_drums` | overcharge: a +5 skull blasts a 5x5 square |
| ember_wyrm | `updraft` | gravity up |
| hollow_king | `hollow_halls` | a 10x8 board |

`standard`, `sideways` and `great_hall` (10x10, the largest board) are spare cards for trying out.

In a dev build, `JEWEL_CARD=sideways ./start.sh` plays every duel on that card, to try one out.

`go test ./rpg -run TestProgression -v` plays the main quests in order with the AI as the hero
and prints each class's level, gold, the residue earned in each color and the Orrery nodes owned
after every quest. It plays the quests in the order they are listed in quests.yaml.

## orrery.yaml

The Orrery is the upgrade grid. Its nodes sit on rings and are joined by links; each belongs to a
color's region and costs that color's **residue**. A hero earns residue on their own turns in a
duel: a match of four, a match of five or more, and matches later in a cascade leave residue of
their color (a plain three on the first step leaves none). It is kept after a lost duel, like
the gold and XP the board paid. A won duel also pays a bounty of residue in the color of the
hero's seat (`WIN_RESIDUE`), so a class that wins fast still grows. Colorless residue, made from colored residue at a loss with the
Convert buttons, pays for any node; when a color's pool is short, colorless makes up the rest.

Each class starts owning its **seat** and can buy any node linked to one it owns. A node gives
stats, a rite (a hero spell, from any class), a perk, or a mix. A class always knows its own rites up to
level 4 (its starting kit); every other rite must be bought here, and is castable from its
spell's level. Respec refunds the residue each bought node took, to the pools that paid it, for some gold
(free once a facility with `free_respec` is built).

The shipped grid has four regions of fifteen nodes on four rings (ring 4 outermost): a seat and
four cheap stat nodes outside, three rites and stat nodes on ring 3, three large perk nodes
on ring 2, and a strong stat node and a level 13 rite on ring 1. Neighbouring regions join on the
outer and the inner ring. Inside them, on ring 0, is Aether's core: twelve nodes in a loop (four
gates, each linked from a region's ring 1 stat node, Aether stats, four Aether rites and three
"forbidden" perks that cost you HP or another pool). It has no seat, and it stays locked, shown
grey, until the run has Aether.

```yaml
- { id: water_seat, name: Tide Seat, color: water, ring: 3, angle: 90, seat: Knight, links: [brine] }
- id: brine
  name: Brine                # optional for a rite node: the rite's name is shown
  color: water               # region: fire, water, air, earth or aether
  ring: 3                    # 1 is the innermost seated ring; 0 is Aether's core
  angle: 60                  # degrees clockwise from the top
  cost: 10                   # residue of the region's color
  stats: { water: 4, morale: 2 }
  rite: shield_bash          # optional spell ID
  links: [loam]              # joined both ways, so list each link once
- id: kindling
  name: Kindling
  color: fire
  ring: 2
  angle: 338
  cost: 10
  perk: { kind: strike, size: 3, amount: 2 }   # a large node: a rule for the hero's duels
```

A **perk** is a rule that changes the hero's side of a duel; foes never have one. It always works
in its node's region color (below, "the color"). The kinds and their parameters:

| kind | parameters | what it does |
|---|---|---|
| `strike` | `size` (3-5), `amount`, `trade` | a match of `size` or more of the color also deals `amount` damage; with `trade: true` (size 4 or more) it deals it instead of the extra turn |
| `mend` | `size` (3-5), `amount`, `trade` | a match of `size` or more of the color also heals `amount`; `trade` as for strike |
| `wild_four` | `amount` (2-3) | a match of exactly four of the color plants an x`amount` Wild on the board, and gives no extra turn |
| `skull_paint` | `amount` (1-4) | each skull match turns up to `amount` gems beside the skulls into the color |
| `overflow` | `to` (another color), `amount` (0-10) | the color's max mana is `amount` lower, and mana past it goes into `to` instead of being lost |
| `forsake` | `amount` | skull matches deal `amount` more per match, and matches of the color give no mana |
| `seed` | `special` (wild or bomb), `amount` (1-4) | the duel starts with `amount` x2 Wilds or +5 skulls on the board; the foe can take them too |
| `hunger` | `size` (3-5), `amount`, `to` (another color) | a match of `size` or more of the color also moves up to `amount` of your `to` mana into the color |
| `blood_price` | `amount` (Aether nodes only) | a rite short of Aether may take the rest from your HP, `amount` HP for each Aether missing; never your last HP |

`strike`, `mend` and `hunger` also take `backlash`: the HP the perk costs you each time it works
(never your last). The Aether perks use it: Void Lance is a strike with a backlash.

A perk that works writes a line in the battle log and lights up in the duel's perk list. Loading
checks the kind and its parameters.

Loading checks that ids and links exist, every class has a seat, rites are hero spells
not already in a starting kit, every node can be reached from every seat, and the four seated
regions can all be reached without passing through Aether's locked core. The other numbers
(residue per match, the conversion rate, the respec price) are placeholders, kept together at the
top of `rpg/orrery.go`.

## Aether

Aether is the fifth element, off at the start of every run. The story flag `aether_unlocked`
turns it on for good (a `set_flag` step; in the slice the done scene of "The Lake Archive",
`archive_done`, sets it, so the Drowned Priest, the Ember Wyrm and the Hollow King are fought with
it). Once it is on, its purple gem falls on every duel board, the standard board grows from 8x8 to
9x9 so that matches stay about as common with one more color, and its mana pool, Aether stat,
residue pool and Orrery core show. Before that, none of them appear. Spells cost Aether like any
other color; the cost list in `rpg/spells.go` is Fire, Water, Air, Earth, Aether. Board sizes
(`DUEL_SIZE`, `AETHER_DUEL_SIZE`) are in `rpg/cards.go`.

A run saved before Aether existed loads with it off and at zero (the old save is kept as
`.bak` first).

## facilities.yaml

Facilities are the academy's buildings. The academy itself opens when a scene sets the flag
`academy_founded` (for now the done scene of the first main quest does). A facility is available
once its `requires` conditions hold (the same conditions as everywhere else), and then each of its
levels is bought with gold, in order. A level's perks add to the levels below it.

```yaml
- id: treasury
  name: Treasury
  desc: More gold from every duel.   # shown under the name
  requires: [quest:toll]             # optional; available once these hold
  levels:
    - { cost: 150, gold_pct: 10 }    # cost is gold, and every level needs one
    - { cost: 300, gold_pct: 10 }
```

A level's perks: `gold_pct` and `xp_pct` (more from every duel), `hp`, `start_mana` (of every color
at the start of a duel), `skull_damage` (per skull match), `armor` (blocks skull damage),
`henchmen` (more henchmen can be hired), `chapter_pct` (completed chapters pay more) and
`free_respec: true` (an Orrery respec costs no gold; the Observatory has it). Facilities
give no stats: those come from the Orrery.

Loading checks that ids are unique, every facility has a level, every level has a cost, and the
conditions are valid. Henchmen, chapters and dues are tuned with the placeholder numbers at the
top of `rpg/academy.go`: henchmen are hired and trained with gold and hold one item the hero
owns; a henchman posted to an open site completes a chapter after a number of the hero's wins
at that site (recalling one loses its progress), which sets that site's holder to the academy and leaves a report of gold and residue in
the site's `residue` color (locations.yaml);
each later win at a held site adds dues to the ledger. Nothing moves while the game is closed.

## puzzles/*.yaml

A puzzle board is an authored board with an objective instead of a foe, and no AI turn. Each file
in `puzzles/` is one puzzle. Some gems are **marked**; the board follows them as they fall and
reports how each was cleared. Puzzles are prototypes: no story scene uses one yet.

```yaml
id: vault
name: The Society Vault
objective: Clear the three tumbler pins.   # shown to the player
moves: 5                     # moves allowed, swaps and tool uses alike; running out loses
refill: none                 # none (cleared cells stay empty), queue, or seed
queue: rbyg                  # refill: queue, colors dropped in order, then nothing
seed: 0                      # drives anything random, so the board always plays the same
gravity: down
board:                       # rows, top first; one cell per word
  - "y  b  g  p  g  b"       # r b y g p are the five colors, . is an empty cell
  - "p  r  y* p  r  r"       # after the color: # locked, * bomb, - or | bolt
  - "g  r  b# r  b# r"
marks:                       # name: [x, y], counting from the top left
  a: [1, 2]
goal: { cleared: [a] }       # finishes the puzzle
fail:                        # optional: any of these holding after a move loses
  - { cleared: [bell] }
tools:                       # rites used on the board, a few times a puzzle instead of for mana
  - { id: force, name: Force, kind: force, uses: 1, rites: [smite, cleave] }   # clear one gem
  - { id: lockpick, name: Lockpick, kind: paint, from: p, to: r, uses: 1, rites: [transmute] }   # repaint one gem
  - { id: dye, name: Dye, kind: convert, from: p, to: r, uses: 1, rites: [transmute] }          # repaint every gem of a color
routes:                      # the first route whose conditions hold when the goal does is taken
  - id: force
    name: Forced the lock
    flag: vault_alarm        # set on the hero
    when: { blasted: true }
    solution: ["swap 1,2 2,2", "swap 4,3 5,3"]
```

A hero can use a tool when they know one of its rites. Conditions, for `goal`, `fail` and `when`
alike, hold when all of their parts do:

- `cleared: [a, b]` these marks are cleared.
- `order: [a, b]` these marks are cleared, each in a later cascade step than the one before.
- `blasted: true` a mark was cleared by a bomb, a bolt or a Force; `false`, none was.
- `tool: true` or `false` a tool was used, or not.
- `max_match`, `min_match` the largest match, and `max_moves` the moves made (a tool use counts as one).

If the goal holds but no route does (the vault's pins matched out of order, say), the puzzle is
lost as jammed. A lost puzzle can be tried again at no cost.

Every route needs a stored `solution`: moves written `swap x,y x,y`, `tool id` (a convert) or `tool id x,y` (a Force or a paint, at that gem).
Loading the story replays each one and reports a problem if it no longer finishes the puzzle by
its route, so `go test ./rpg` catches a board edit that breaks a route.

To see every way a board can end, run the brute-force solver:
`SOLVE=vault SOLVE_DEPTH=4 go test ./rpg -run TestSolvePuzzle -v`. It tries every swap and tool
use up to that many moves and prints, for each route and each kind of loss (jammed, stuck), how
many sequences end there and the shortest one. Use it to check that every route is reachable and
none leaks into another; depth 4 on the vault takes about 15 seconds, depth 5 a few minutes.

In a dev build, `JEWEL_PUZZLE=vault ./start.sh` opens the puzzle once a hero is loaded, and
`JEWEL_PUZZLE_ROUTE=quiet` as well plays that route's stored solution by itself.

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
| `set_holder: { at: id, holder: crown }` | say who holds a location; `holder: academy` is the player's own, and an empty holder means no one. Sites the academy holds pay dues |
| `end: true` | mark the story beaten |

### Flow

| Field | Meaning |
|---|---|
| `goto: scene` | stop this scene and play that one instead |
| `run: scene` | play that scene, then come back here |
| `if: [...]` with `then:` / `else:` | branch on conditions |
| `if: [...]` alone | the rest of the step only happens when the conditions hold |
| `duel: { foe: id, level: 3, win: scene, lose: scene }` | fight, then continue |
| `puzzle: vault` | play a puzzle board (puzzles/), then continue whatever the result; check its route flags with `if` |

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
- items and the shop: `rpg/hero.go`
- henchmen, chapters and dues: `rpg/academy.go`
- duel rules and balance numbers: `rpg/duel.go`
- the scene player: `core/scene-view.go`; the loader and the checks: `rpg/story.go`

### Adding a new kind of step

1. Add a field to `Step` in `rpg/story.go`, with its YAML name in the tag.
2. If it refers to something by id, check it in `checkSteps` so mistakes are caught.
3. Make it do its thing in `SceneRunner.exec` in `core/scene-view.go`.
4. Document it here.
