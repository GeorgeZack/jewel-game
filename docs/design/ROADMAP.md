# Roadmap

Phased work for agents on GeorgeZack/jewel-game `master`, rewritten 12 Nov 2026 to follow
`DESIGN.md`. Milestones are in dependency order. Keep a PR to one task. Do not invent files that
are not marked **new**. Section numbers such as (5.6) refer to `DESIGN.md`.

Items marked **Decision needed** stop work: ask George. Numbers (costs, rates, payouts) are never
invented and shipped; they go in data as tunables and are set from the simulator in milestone 0.

Two review gates (milestones 1 and 2) come before any breadth. If either prototype is not fun or
not reliable, the design changes before more is built on it.

## Open decisions

- [ ] **Decision needed, when reached:** each Proposed item in `DESIGN.md`. George will rule on
      these as the milestone that needs them comes up: rule cards and the 10x10 cap (5.1, before
      milestones 0 and 5), rival college as an
      epilogue variant (5.7, before 11), early-return cost and first-time payouts (5.8, before 7).
- [ ] **Decision needed:** names for the player's society, the rival college, the entities and
      the meta currency. The two main societies have working names (5.7).
- [x] Decided 13 Nov 2026: returns are unlimited, meta stage 2 opens after at least three, and
      foes do not scale with returns.

Decided: no sound in the first release, and Steam is on hold. The release is the browser build.

## 0. Foundation

**Goal:** The plumbing every later milestone needs, with default play unchanged.

**Depends on:** nothing.

- [ ] Verify touch, spell buttons, swaps, dialogue and pause on a real phone against
      `core/phone.go`, `core/game-controller.go`, `rl/app.go`, `core/tile-manager.go`. Fix only
      what is broken. Do not reimplement finger-as-mouse.
- [x] Add a save version and a migration path (`rpg/world.go`, `rpg/store_file.go`,
      `rpg/store_js.go`). Old saves load or start fresh without crashing.
- [x] Split the save into a profile (Grimoire, endings seen, return count, tutorial done) and a
      run (today's `Hero`). The profile can be empty for now.
- [x] Rule cards (5.1): a named `board.Settings` override plus optional `ApplyStep` payout
      modifiers, defined in data (**new** YAML under `assets/`), loaded and id-checked with the
      story in `rpg/story.go`.
- [x] Plumb a card through `core/quest-view.go` `startDuel` and `rpg/duel.go` `BoardSettings`.
      The default card is today's 8x8 duel. Test that a non-default gravity, Overcharge or size
      reaches the board.
- [x] Progression simulator: extend the duel balance test (`rpg/rpg_test.go`
      `TestDuelsFinishAndAreFair`) so the AI plays a scripted quest sequence as the hero and
      reports level, gold and residue per stage.

**Acceptance:** The current campaign plays as before. A debug hook starts a duel on another
card. The simulator prints a progression table.

## 1. Gate: residue and a small Orrery, on the current slice

**Goal:** Find out whether coloured residue changes how a fight is played (5.2, 5.3).

**Depends on:** 0.

- [x] Coloured residue from match quality in `rpg/duel.go` `ApplyStep`, banked in `Settle`.
- [x] A small Orrery of about twenty nodes (**new** `rpg/orrery.go` and YAML), four class seats,
      nodes costing their region's colour. It replaces level-up stat points (`Hero.Points`,
      `RaiseStat`).
- [x] Open `KnownSpells` so rites are not locked to `Class` (`rpg/hero.go`, `rpg/spells.go`);
      rites come from Orrery nodes and level gates.
- [x] Conversion of coloured residue to colourless at a loss, and respec that refunds nodes.
- [x] A plain Orrery screen that works on the narrow layout (`core/quest-menus.go` or **new**
      view, `core/phone.go`).

**Acceptance:** Two classes start in different seats and can reach the same node. The Hollow
King slice is finishable. **Stop for George's review** before milestone 4.

## 2. Gate: one puzzle board

**Goal:** Prove that routes on an authored board can be detected reliably (5.6).

**Depends on:** 0.

- [x] Authored-board loader (**new** `assets/puzzles/` YAML into `board.Paint` or a prefilled
      board) with no refill, an authored refill queue, or a fixed seed.
- [x] `PuzzleGemTypes` beside `DuelGemTypes` in `core/data-manager.go`.
- [x] Objective and route conditions as data, evaluated over `board.StepResult` events: marked
      gems cleared, by match or blast, order, largest match, move count, tool used. **new** types
      in `rpg/`, wired from `core/quest-view.go`. No `Combatant.HP` win.
- [x] Tools are the hero's rites with a per-board allowance. Failure and free retry.
- [x] The society vault board with its three routes, each setting a flag.
- [x] A test that replays a stored solution for every route and checks the route detected.

**Acceptance:** The vault can be launched from a debug hook and finished three ways, each
detected correctly. Adversarial duels are unchanged. **Stop for George's review** before more
boards are authored.

## 3. The academy

**Goal:** A locked academy with facilities, henchmen and chapters (5.2, 5.5).

**Depends on:** 0. Uses a founding flag until milestone 9 supplies the quest.

- [x] Lock academy nav until the founding flag is set (`core/quest-view.go`, `core/phone.go`
      `drawPhoneNav`, `core/quest-menus.go` `drawCitadel`).
- [x] Facilities replace Buildings (`rpg/hero.go`): made available by a flag, built and upgraded
      with gold. Remove mounts and the Stables. Move flat stat bonuses to the Orrery.
- [x] Henchmen: hire, train and equip with gold (**new**). No names or portraits.
- [x] Field chapters and the dues ledger: send a henchman to a site; it resolves on wins or
      objectives there, never on real time; claim the report, dues and residue at the academy.
- [x] Location holder state (which society holds a site), shown in the ledger.

**Acceptance:** A new hero cannot open the academy. After the flag, gold flows gear, then
henchmen, then facilities. Closing the game advances nothing.

## 4. The full Orrery

**Goal:** The shipping grid (5.3).

**Depends on:** gate 1 passed, 3.

- [x] Grow the Orrery to its budgeted size, with regions by element.
- [x] Large nodes that carry rule perks, implemented as modifiers on the hero's side of
      `ApplyStep` or as card overrides.
- [x] One facility makes respec free.
- [x] Final Orrery screen for desktop and phone.

**Acceptance:** Every node is reachable from every seat. Simulated duels stay fair with each
large node taken.

## 5. Board growth, the fifth element, boss cards

**Goal:** Later fights differ in rules and size (5.1).

**Depends on:** 0.

- [x] Allow duel boards other than `GRID_LENGTH` (8), up to 10x10 (`utils/constants.go`,
      `rpg/duel.go`, `core/tile-manager.go`, `core/phone.go`). Verify gem size on a phone.
- [x] Generalise `NUM_MANA` so a fifth element can be enabled by flag: pools, costs, stats,
      residue colour, an Orrery region. It arrives together with a larger board.
- [x] A rule card id on a foe (`assets/story/foes.yaml`) and bosses that use one.
- [x] Teach `rpg/ai.go` to value the fifth element, and add a simulated-duel test per card.

**Acceptance:** A boss fight runs on a different card and the campaign returns to the default
after it. Every card has a passing fairness test.

## 6. Story format

**Goal:** The story YAML can express the campaign (5.5, 5.7). Document each addition in
`docs/story.md`.

**Depends on:** 0, 2, 3.

- [x] Standing counters per society, steps to change them, and comparisons in conditions.
- [x] A `puzzle:` step beside `duel:`, with a scene per route.
- [x] A rule card id on a quest and on a `duel:` step.
- [x] Location holders that scenes can change; foes, shop stock and description vary by holder.
- [x] Hostile societies: swap a location's roaming foes by flag.
- [x] Act markers for title cards and gating.
- [x] Companions: favour counters, joining and leaving, the one who comes into a fight and their
      rite or passive.
- [x] Conditions on the profile: return count and endings seen.

**Acceptance:** `go test ./rpg` validates all of the above and still checks that main quests
can be finished in order, for each allegiance.

## 7. The return and the Grimoire

**Goal:** The run-based loop and meta stages 1 and 2 (5.8).

**Depends on:** 0 (profile save), 6 (profile conditions).

- [x] Tutorial once per profile. Class choice moves to the first event (`core/title.go`).
- [x] Return: reset the run (facilities, Orrery, gold, level, map, quests, standing, staff) and
      restart at the first event. Available at an ending and, at a cost, mid-run.
- [x] Meta currency: earned on first times, banked at each act's end, forfeited for the current
      act on an early return.
- [x] The Grimoire (**new** data and screen): entries that buy speed and options, and from stage
      2 the two kinds of entry that set the meta lean.
- [x] An entity scene at each return, varying with return count and endings seen.

**Acceptance:** A profile can finish, return, change class and allegiance, and keep its
Grimoire. An early return loses only the current act's unbanked earnings.

## 8. Retheme

**Goal:** All text reads as Victorian occult societies, once.

**Depends on:** names (working names exist for the two main societies). Systems from 1 to 5 should be in so their strings exist.

- [x] Gems as reagents, spells as rites and devices, classes, facilities, henchmen
      (`assets/gemtypes.yaml`, `assets/lang/en.yaml`, display names in `rpg/`).
- [x] UI colour and typography in `ui/theme.go` and `ui/widgets.go`; manuscript borders when the
      art exists.

**Acceptance:** No high-fantasy wording remains in menus. Ids and saves are unaffected.

## 9. Act 1

**Depends on:** 6, 8.

- [x] The tutorial, the first event, and Act 1 quests on the shared map, with every society
      courting the adept. Budget the scene text before writing. (`scenes/act1.yaml`; see "The
      shipped story" in `docs/story.md`.)
- [x] Companions introduced: Clemency Hart (Royal Society), Jory Fitch (Independents) and Ines
      Marlow (College), one on each society's quest.
- [x] Two or three puzzle boards: `assay`, `vault` and `reliquary`, each with routes that move
      standing.
- [x] The founding quest, which sets the first lean and opens the academy.

**Acceptance:** Act 1 is finishable by every class, on a phone. Backed by
`TestActOneByEveryClass` (simulated duels, every class) and `TestStoryTextFitsAPhone`; the
on-screen phone check is still to do.

## 10. Act 2

**Depends on:** 9, 3, 4, 5.

- [ ] Work for several societies; standing moves; sites change holders.
- [ ] Facilities and bosses with rule cards come online. The fifth element arrives.
- [ ] Three or four puzzle boards whose routes move standing.

**Acceptance:** A hero can reach the lock with at least three different standing outcomes.

## 11. Act 3 and the endings

**Depends on:** 10, 7.

- [ ] The lock: allegiance is permanent for the run, other societies turn hostile, their
      companions leave.
- [ ] One rule card and one payout modifier per allegiance (5.7). Build the independent path's
      mid-fight rule rewrite last; it needs new duel UI.
- [ ] The Engine climax and three endings (Crown, cult, independent), with the rival college as
      an epilogue variant. Epilogue slides for societies, the academy and companions.

**Acceptance:** Each ending is reachable on one profile across returns, and the board rules
change at the lock for each allegiance.

## 12. Art

**Depends on:** 8. Runs beside 9 to 11.

- [ ] Style pack made outside the repo: colored-pencil hatching for art, manuscript borders for
      UI. Keep a prompt and seed log (**new**, `docs/design/art/`).
- [ ] Gems, portraits, foes, backgrounds, the Orrery and the Grimoire.
- [ ] Legible on phone and desktop.
- [ ] Update `attributions.md` and the README's AI-art note.

**Acceptance:** No slice pixel art remains on player-facing screens.

## 13. Balance and playtest

**Depends on:** 11.

- [ ] Set costs, rates and payouts from the simulator, then from played runs.
- [ ] A first run of six to eight hours; a later run clearly shorter.
- [ ] Full playthroughs on phone and desktop for each ending and for an early return.

**Acceptance:** No allegiance, seat or large node is dominant or dead in the simulator.

## 14. Release

**Depends on:** 12, 13.

- [ ] `go test ./...`; WASM build via `build-web.sh`.
- [ ] Replace Hollow King copy in `README.md` and `core/title.go`.
- [ ] An AI-art note on the page and in `attributions.md`, listing every tool including Retro
      Diffusion. Store questionnaires wait until Steam is back on the table.

**Acceptance:** A new player can finish an ending and return, on a phone.

## Later: meta stages 3 and 4

Not in the first release. Binding companions and society leaders to the Grimoire, and the
reckoning with the entities (5.8). The profile save and the entity scenes in milestone 7 must
leave room for both. Budget separately before writing.
