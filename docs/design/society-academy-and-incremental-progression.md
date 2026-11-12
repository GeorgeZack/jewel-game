# Society academy and incremental progression

Lean design notes for GeorgeZack/jewel-game, 11 Nov 2026 (America/Chicago). The Citadel becomes the society academy. Progression is incremental, not idle. Duels split into adversarial and puzzle modes. The existing match-3 engine stays; it is not rebuilt.

## Decisions

- [x] Lock the academy (the Citadel nav that is always on today) behind a founding quest.
- [x] Incremental only: no overnight or offline timers.
- [x] Classes are starting templates. Every rite stays learnable. Respec is cheap early and free after an academy faculty.
- [x] Sphere grid plus gem alchemy. Elements unlock over the acts.
- [x] The board can grow. Area bosses change a rule. Three or four acts.
- [x] Steampunk / Victorian occult-studies theme. No intentionally cozy design. Mobile is first class.
- [x] Do not rebuild the match-3 engine.
- [x] Split duels: adversarial (combat gem set, the current fight) and puzzle (a different gem set, an authored board, an objective, several routes). Decided 11 Nov 2026.
- [ ] Which act the founding quest sits in (open).

## Classes and respec

Today, `rpg/hero.go` sets starting stats for Knight, Warrior, Druid and Wizard, and `KnownSpells` in `rpg/spells.go` only unlocks spells whose Class matches. Keep those four as starting templates for the first duels, and open `KnownSpells` so every rite or device is learnable. Respec is cheap early and free after an academy faculty.

References:
- FFX Sphere Grid: characters start in different seats, and every node is eventually reachable ([guide](https://eip.gg/ffx-x2/guides/sphere-grid/)).
- Path of Exile: one shared tree with different starts. Don't copy its scarce refunds or class-locked Ascendancies ([forum](https://www.pathofexile.com/forum/view-thread/59640)).
- Diablo 3: unlimited respecs in town ([wiki](https://di.diablowiki.net/Respec)).
- Diablo 4: nearly free during the campaign, then expensive. Do the opposite after a faculty ([Polygon](https://www.polygon.com/23697507/diablo-4-respec-costs-build-flexibility/)).
- Hades Mirror of Night: refund upgrades for one Key ([wiki](https://hades.wiki.gg/wiki/Hades:Mirror_of_Night)).

## Academy layers

Today the Citadel nav is always on (`core/quest-view.go`), and buildings are gold purchases (`rpg/hero.go`: Treasury, Stables, Barracks, Library, Temple, Mage Tower, War Room). The proposals below retheme them as faculties that are earned in play.

| Layer | Rule | Hook |
|---|---|---|
| Founding quest | The academy stays locked until a fight in mid-Act 1 or at the start of Act 2. Winning it founds the society, opens the first faculties and allows the free-respec upgrade. | `core/quest-view.go` nav |
| Refinery | Residue comes only from fights you play. Output scales with chain length and match quality. Nothing cooks while the game is closed. | New spend action fed from `ApplyStep` |
| Field chapters | Expeditions resolve from wins or board objectives, not real time. Each returns a report plus reagents, and quality follows the fight that enabled it. | New layer; objectives can be puzzle duels |
| Faculties | Bosses unlock faculties and a board rule. One faculty makes respec free. Retheme the existing Building ids and add no timers. | `rpg/hero.go` Buildings |
| Alembic table | A short occult-studies session: combine pooled gems on a small board, using the existing gem language. | `board` package; home session |
| Dues | A site pays when you win there or finish a local objective, and you claim it at the academy. No calendar backlog. | Map ledger; Gold already in `Settle` |
| Board growth / prestige | Duels are 8x8 now (`BoardSettings`, `GRID_LENGTH`), and `board.Settings` already allows 4 to 16. Grow the board later. Optional New Game "Ace" restart that keeps upgrades. | `rpg/duel.go`; `board/board.go` |

## Combat and duel modes

Keep the current fight loop and split duels into two modes with different gem sets. Society affiliation can still change `ApplyStep` payouts and which colors are common; that's covered in the branching doc.

| Mode | What it is | Code |
|---|---|---|
| Adversarial | The current turn-based fight against a foe on one shared board, using the combat reagent set. Colored gems fill four mana pools (Fire, Water, Air and Earth today). Skulls deal damage, and +5 skulls are Bombs. Gold and stars bank after the fight. Matching 4 gives an extra turn, and matching 5 leaves a Wild. With no moves left, mana drains and the board reshuffles. Spells spend mana and can already Force, Convert or PlaceWild. Open `KnownSpells`. Retheme foes in YAML as cultists, automata, rival agents and ruin horrors. Bosses pass their own `board.Settings` (gravity, Overcharge, size) into `startDuel`. The flags exist, but every duel uses the same 8x8 Duel settings today. | `rpg/duel.go`; `rpg/hero.go`; `rpg/spells.go`; `rpg/ai.go`; `board/resolve.go`; `core/quest-view.go` `startDuel`; `core/data-manager.go` `DuelGemTypes`; `assets/story/foes.yaml`; `assets/lang/en.yaml` |
| Puzzle | A different gem set on a pre-built authored board, with a specific objective instead of HP. Several valid routes, each with a different payoff or consequence. This isn't a second engine: it uses the same matching, swaps and spell hooks. | See Puzzle duels below |

## Puzzle duels

Deus Ex was designed so every problem has multiple solutions: force, stealth, tools or talk. That's from Warren Spector's design commandments, and the 2002 postmortem notes the paths were often planned rather than fully emergent ([Spector](https://www.gamedeveloper.com/design/warren-spector-traces-i-deus-ex-i-s-development-back-to-a-game-of-d-d), [postmortem](https://www.gamedeveloper.com/design/postmortem-ion-storm-s-i-deus-ex-i-)). Steve Pearsall of Thief 2 said each mission has a set of objectives and several ways to achieve them, with combat as the last resort ([interview](https://web.archive.org/web/20000815220921/rpgvault.ign.com/features/interviews/thief2.shtml)). Puzzle boards should work the same way: one objective, at least two routes, different consequences. The four boards below are proposals.

| Board | Objective | Routes and payoffs |
|---|---|---|
| Society vault | Open the lock. Locked gems already exist in the `board` package. | **Force the lock** (blast it or Force-clear the tumbler): fast, but sets an alarm flag and hurts the ledger. **Quiet bypass** (route a color around the locks, or release them in order): clean dues. **Lockpick rite** (Convert or PlaceWild to complete the tumbler): spends a device charge and opens cleanly. |
| Cipher wheel | Reveal the message with a required color pattern or order. | **Deduce the pattern** (authored locks hold the runs): full text and bonus insight. **Force-blast the wheel**: partial text, and some insight gems are destroyed. **Known-key rite** (Convert to the cipher color): spends a charge and gives the full text. |
| Boiler / pressure valve | Vent the boiler without a blowout. Its own Settings can use gravity or Overcharge. | **Bleed the valves in sequence** without making a match of 4 or more: bonus residue and no backlash. **Overpressure** (clear all pressure gems at once): works, but costs a blast or HP. **Regulator device** (lock a column or change Gravity): spends a charge and vents safely. |
| Ritual seal | Close or break the seal, with no single required path. | **Close the ring cleanly** (locks isolate the seal, then you complete it): academy standing. **Break it** (destroy the seal gems): works, unlocks a forbidden color earlier, and taints the rules later. **Binding rite** (PlaceWild or Convert to close the circle): spends mana and closes cleanly. |

Implementation notes:
- Add `PuzzleGemTypes` beside `DuelGemTypes` in `core/data-manager.go`. `TileManager.Types` is already a palette hook.
- Add an authored-board loader, either new YAML fed into `board.Paint` or a prefilled `NewBoard`. Today `NewBoard` only fills randomly.
- Win by checking the objective, not by `Combatant.HP` or a damage-based `Duel.Winner`.
- Pass a separate `board.Settings` for puzzles. Enable locks, since `TileManager.Duel` currently turns them off, and set `FreeSwap`, `Gravity` and `MaxLocks` as needed.
- Tools are the existing Spell hooks: `Force`, `Convert`, `PlaceWild`.
- The route taken sets `Hero.Flags` for later payoffs.

## Comparables, avoid, mobile

Comparable games:
- 10000000: short runs buy persistent rooms. It had no pause, which is a warning ([TouchArcade](https://toucharcade.com/2012/07/26/10000000-review/)).
- You Must Build a Boat: permanent upgrades, and an Ace restart keeps them and marks the count. Use that for New Game, not a daily reset ([StackExchange](https://gaming.stackexchange.com/questions/223368/whats-the-changes-after-multiple-playthroughs-of-you-must-build-a-boat)).
- Melvor Idle: Mastery from repeating an action is useful. Don't copy its 24-hour offline simulation ([wiki](https://wiki.melvoridle.com/w/Offline_Progression)).
- Rogue Legacy: gold from a run buys permanent manor upgrades ([wiki](https://roguelegacy.wiki.gg/wiki/Upgrades)).
- Slay the Spire: score unlocks cards at 300, 1,000 and 2,000 and relics at 750 and 1,500, and the next character unlocks after a clear ([wiki](https://slaythespire.wiki.gg/wiki/Score)).
- Puzzle Quest: the Citadel's buildings are earned, with no build timer. Don't copy Puzzle Quest 3's paragon track or guild Kingdoms ([StrategyWiki](https://strategywiki.org/wiki/Puzzle_Quest:_Challenge_of_the_Warlords/The_Citadel)).

Avoid: AFK chests, build queues, stamina-for-cash, gacha as academy progress, a bolted-on comfort-shop layer, and class locks or late respec costs that force a reroll.

Mobile: the same Go + Ebitengine build already runs in the browser as WASM (`web/index.html`, deployed via jewel-game-web). `uiScale` in `core/game-controller.go` shrinks two 360px side panels from a design height of 860. The phone layout and touch input (a finger acting as the mouse) landed on `master` in `2fe0f137` on 11 Nov 2026, so verify them on a device instead of reimplementing them. Spell buttons and swaps must work under a finger. Saves use localStorage via `rpg/store_js.go`. A phone session is one or two boards, then a short spend at the table. Pause has to work, and nothing should cook while the game is closed.
