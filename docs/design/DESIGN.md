# Design: the society game

Draft, 12 Nov 2026. This is the single statement of what the next game is. It consolidates
`society-academy-and-incremental-progression.md` and `rival-societies-and-branching-campaign.md`,
which stay in this folder as research notes (comparable games, links, the four example puzzle
boards). Where this file and those notes disagree, this file wins.

Marks used below:

- **Settled**: decided by George in the earlier notes. Not reopened here.
- **Proposed**: a recommendation made in this draft that fills a gap in the notes. Not decided
  until George confirms it. Agents must not build on a Proposed item without checking.
- **Open**: nobody has an answer yet.

## 1. The game in one paragraph

A match-3 duelling RPG set among rival Victorian occult societies. You arrive alone, found your own
academy, work for several societies on one shared map, and then commit to one of them. The
commitment changes the rules of the board you fight on, not only the dialogue. Every fight feeds
the academy, and the academy changes what you bring to the next fight. Nothing advances while the
game is closed.

The hook that no comparable game has: **your allegiance changes the physics of the board.** The
engine already carries the switches for this (`board.Settings`: gravity, size, locks, Overcharge,
free swaps, colour count), so the most distinctive feature is also the cheapest one to build.
Everything else in the design should be judged by whether it serves that hook.

## 2. Pillars

1. **The board is the game.** Every system pays in or out through a board: fights, puzzle boards.
   No progression screen exists that is not fed by play. (Settled, as "incremental,
   not idle".)
2. **Rules are loot.** Bosses, faculties, elements and allegiance each change a board rule. A rule
   change is worth more than a stat bonus and is preferred wherever both would work. (Proposed as a
   pillar; follows from Settled items.)
3. **Commitment costs something.** Societies court you, then one choice makes enemies of the rest.
   Returning to the first event is how you see another path. (Settled.)
4. **One or two boards, then a short spend.** That is a phone session, and it is the unit every
   system is paced against. (Settled.)
5. **Dry, not cozy.** Tone is conspiratorial and a little dangerous: ledgers, oaths, rivals and
   forbidden work. (Settled as "no intentionally cozy design"; the positive wording is Proposed.)

## 3. The loop

```
fight or puzzle board  ->  residue, gold, reports, flags
        ^                              |
        |                              v
   new rules, rites,      <-   spend at the academy
   elements, map access        (grid, alchemy, faculties)
```

- **Minute loop:** a turn on the board.
- **Session loop:** one or two boards, then a spend at the academy.
- **Act loop:** a boss falls, a faculty and a rule open, the map changes hands.
- **Campaign loop:** finish an ending, return to the first event, choose differently.

The campaign loop is what makes "incremental" and "branching" one game instead of two. It is a
run-based structure with meta progression (Settled, 12 Nov 2026): the academy and grid are rebuilt
every run, and a permanent occult book carries over. See 5.8.

## 4. Scope budget

The notes set no size for the game. Without one, the campaign milestone cannot be estimated and
the economy cannot be tuned. These targets for the first release are Settled (12 Nov 2026), except where a row says Proposed:

| Thing | Today | Target |
|---|---|---|
| First playthrough | under 2 hours | 6 to 8 hours |
| Acts | 1 arc | 3, with the lock at the start of Act 3 |
| Endings | 1 | 3 full (Crown, cult, independent); rival college as an epilogue variant |
| Main quests | 8 | about 24 |
| Foes | see `foes.yaml` | about 30, of which 6 to 8 bosses carry a rule card |
| Puzzle boards | 0 | 8 to 10, each with two or three routes |
| Rites (spells) | 33 in `rpg/spells.go`, hero and monster | about 45; new ones mostly use the new element and rule hooks |
| Grid nodes | 0 | 60 to 80 |
| Occult book entries | 0 | 20 to 30 (Proposed) |
| Companions | 0 | 4 to 6 (Proposed) |
| Mana elements | 4 | 5 (the forbidden colour), 6 at most |
| Scene text | about 100 lines of YAML | the largest single cost; budget it per act before writing |

Code is the cheap part of this project. Writing, balance and consistent art are the expensive
parts, and the budget above is mostly a writing budget.

## 5. Systems

### 5.1 Fights

Settled: the adversarial duel stays as it is (shared board, four mana pools, skulls, extra turns,
Wilds). Classes become starting templates, every rite is learnable, respec is cheap and later free.

Requirements the notes do not state:

- **Rule cards.** A rule card is a named, data-defined `board.Settings` override plus optional
  payout modifiers for `ApplyStep`. Bosses, puzzle boards, locations and allegiance all refer to
  cards by id. One mechanism serves milestones 0, 3, 6 and 7 of the roadmap. (Proposed.)
- **The foe AI must read the card.** `rpg/ai.go` ranks swaps with `PreviewSwap`, so it survives
  gravity and size changes, but it knows nothing about a fifth element or a card's payout
  modifiers. Each new card needs a simulated-duel test like `TestDuelsFinishAndAreFair`.
- **A fifth element is a combat change, not content.** `NUM_MANA` is a fixed array size in costs,
  pools and stats. Each extra colour also makes matches rarer on an 8x8 board. So the forbidden
  colour and board growth belong together: the element arrives with a larger board. (Proposed.)
- **Board size on a phone has a hard cap.** `board.Settings` allows 16 wide, but ten columns on a
  360 px wide phone is already 36 px a gem. Treat 10x10 as the largest duel board until a device
  test says otherwise, and let puzzle boards be smaller than 8x8 as well as larger. (Proposed.)

### 5.2 Progression

Settled, 12 Nov 2026:

- The sphere grid replaces level-up stat points. Levels keep giving HP and gate rites; grid nodes
  give stats, rites and rule perks. Respec refunds grid nodes.
- Gold is spent on facilities (and gear). Residue is spent on the grid.
- Mounts are dropped, and the Stables with them.

A facility therefore has two gates: a boss or story beat makes it available, and gold builds it.
This replaces the earlier note's "not gold purchases".

Gold has three sinks, in the order the player meets them (Settled, 12 Nov 2026): gear for
yourself, henchmen (hiring, training, equipping), and facilities that you and your staff
both benefit from. See 5.5.

### 5.3 Alchemy, the Orrery and the Grimoire

Settled, 12 Nov 2026.

- **Alchemy** is the match-3 board: the main loop. A fight is alchemy done against someone. Gems
  are reagents. The board itself is the alchemy table.
- **The Orrery** is the circular upgrade grid, after Final Fantasy X's sphere grid: a brass model
  of orbits, with nodes as bodies on the rings and class seats at different points.
- **The Grimoire** is the permanent book that survives a return (5.8).

**The gems you match are the spheres you spend.** Residue comes in the element colours. Matching
a colour well in a fight adds to that colour's pool, and Orrery nodes cost residue of their
region's colour. How you play decides where you can grow. Rule perks (a colour that also heals, a
four that leaves a lock) are the large nodes. A new element in a later act opens a new region.

Coloured residue can be converted to colourless residue at a cost. Colourless pays for any node.
So a player who only ever matches fire can still reach the water region, slowly. The rate is a
tunable; it should be poor enough that playing the colour is clearly better.

The separate "alembic table" minigame from the earlier note is cut. Conversion is a button at
the Orrery.

### 5.4 Economy

Settled in shape, 12 Nov 2026. Reagents from the first draft are removed along with recipes.

| Currency | Earned by | Spent on | Kept between runs |
|---|---|---|---|
| Gold | Gold gems, quests, dues | Facilities, gear | No |
| Residue, per colour | Matching that colour well in fights you play (chains, fours and fives) | Orrery nodes of that colour; converts to colourless at a loss | No |
| XP | Star gems, quests | Levels | No |
| Standing (per society) | Quests, routes taken, the founding lean | Not spent. Gates quests, sets prices, decides who turns on you at the lock | No |
| Meta currency (name Open) | See 5.8 | The occult book | Yes |

- "Device charges" are a per-board allowance on puzzle boards, not a currency.
- "Insight", "reagents" and "academy stock" are not currencies. Field chapters and puzzle routes
  pay in gold, residue and standing. The independent path's rule rewrites spend residue.
- Dues are gold paid by sites you hold, claimed at the academy, as part of field chapters.

Standing needs a number. Story flags are on/off today (`Hero.Flags`), so standing needs a counter
type and a condition such as `standing:crown>=3` in the story format.

### 5.5 The academy

Three layers (the third depends on 5.3):

| Layer | What the player does there | Fed by |
|---|---|---|
| Facilities | Builds and upgrades with gold | Bosses and story beats make them available |
| Orrery | Spends residue on nodes; converts colours; respecs | Every fight |
| Chapters and ledger | Sends henchmen, reads reports, claims dues, sees who holds which site | Wins and objectives at a site |

The founding quest opens the academy with its first facility. All of it resets at the end of a
run.

**Staff.** Two kinds, which do different jobs (the two kinds are Settled, 12 Nov 2026; their details are Proposed).

*Henchmen* are unnamed and bought. Gold hires, trains and equips them. They are who you send on
field chapters, and a chapter's report and dues depend on how good they are. They are the middle
gold sink: gear for yourself, then henchmen, then facilities that serve both. They need no
writing and no portraits, only a rank and a gear slot.

*Companions* are a small cast of named characters, four to six. They are not bought. Each
favours a society, and you earn their favour through quests, routes taken and standing.

- A companion in favour contributes to play: one comes into a fight and adds one thing, a rite
  on a long cooldown or a passive. They are not a second combatant with HP.
- Joining a society at the lock is permanent for that run. Companions who favour the others
  leave, and may be met again as foes.
- They are the companions of the epilogue slides.
- Both kinds reset on a return, except as the Grimoire allows (5.8).

Scope: each companion is a character to write, a portrait, a rite and an ending slide. Henchmen
are cheap. If the cast has to shrink, shrink companions.

### 5.6 Puzzle boards

Settled: a second mode with its own gem set, an authored board, an objective instead of HP, and at
least two routes with different consequences. Call them puzzle boards rather than duels, since
there is no opponent.

This is the riskiest system in the design, because match-3 refills are random and "which route
did the player take" has to be decided by the game, not asserted. Requirements (Settled, 12 Nov
2026):

- **Determinism.** A puzzle board either has no refill (cleared cells stay empty or fill from an
  authored queue) or a fixed seed. An authored board with random refills cannot guarantee a route.
- **An objective model.** Objectives and routes are conditions over events the board already
  reports in `StepResult`: which marked gems were cleared, by a match or by a blast, in what
  order, with what largest match, in how many moves, and whether a tool was used. A route is the
  first route whose conditions hold when the objective completes. This must be data, in the puzzle
  file, and checked by tests that replay a stored solution for every route.
- **Failure.** A puzzle can be lost (moves run out, a blowout) and retried at no cost, as duels
  are today.
- **Tools.** Rites the hero owns, limited to a per-board number of uses.
- **Consequences.** Each route sets a flag and may move standing. At least one route on every
  board should cost something later.

Build one board end to end, with a solver test, before the framework is generalised. If routes
cannot be detected reliably on the vault board, the design changes before four boards are authored.

### 5.7 Campaign

Settled: branch and bottleneck, shared maps, a founding quest as the first lean, one hard lock, an
Engine climax, epilogue slides, and allegiance that changes board rules and income.

- **Settled:** three acts. Act 1 ends with the founding quest, so the academy opens at the start
  of Act 2. This gives Act 1 a clear goal and keeps the first hour free of menus.
- **Working names (12 Nov 2026, may change):** the Crown's society is *The Royal Society for the
  Study of Alchemy and Natural Sciences*. The forbidden society is *The Society of Independent
  Alchemists*. The player's own society, the rival college, the entities and the meta currency
  are unnamed. Note for writing: "Independent" in the forbidden society's name sits close to the
  independent ending (seizing the Engine for your own society); this file calls that ending the
  player's own claim where the two could be confused.
- **Proposed:** each allegiance is one rule card and one payout modifier, applied to every fight
  after the lock, as in the table in the branching note. The independent path's mid-fight rule
  rewrite is the only one that needs new duel UI; build it last.
- **Proposed:** the rival college is an epilogue variant and an Act 2 ally, not a fourth ending.

What the story format needs that it does not have (see `docs/story.md`):

- standing counters and comparisons in conditions
- a `puzzle:` step beside `duel:`, with a scene per route
- a rule card id on a foe, a quest or a `duel:` step
- a holder for each location (which society holds it) that scenes can change, with foes, shop
  stock and description varying by holder. This is what "shared maps, reskinned" means in data.
- hostile societies after the lock: a way to swap a location's roaming foes by flag
- an act marker for title cards and for gating

### 5.8 The return

Settled, 12 Nov 2026. The game is run-based with meta progression.

- Finishing the game resets facilities, the grid, gold, level, the map, quests, standing and
  allegiance.
- A permanent upgrade path persists on the profile: an occult tree or book. Eldritch entities
  guide it, and their power is what returns you to the first event after the tutorial.
- The tutorial is played once per profile. Every later run starts at that first event.
- All faction routes are played on one profile. The point is to try another allegiance and
  another build cheaply.

Consequences for the rest of the design:

- **The entities are a fourth party in the story.** They sit above the societies and know you
  have done this before. That is a writing opportunity: scenes can check how many returns and
  which endings the profile has seen.
- **The save splits in two.** A profile (book, endings seen, return count, tutorial done) and a
  run (today's `Hero`). This is the first save change to make.
- **Later runs must be shorter.** Six to eight hours is right for a first run and too long for a
  loop. Book upgrades should mostly buy speed and options, not raw power: start with a facility
  built, start on a chosen grid seat, carry a rite, skip a known fight. Scene skipping already
  exists.
- **Class choice happens at the first event**, not at profile creation, so a return can change it.

Returning early (Settled that it is allowed and costs something; the cost is Proposed): the
meta currency a run earns is banked at the end of each act. Returning mid-act forfeits what the
current act has earned and the ending payout. What is banked is never taken. This makes an
abandoned experiment cost time, not past progress.

What pays the meta currency (Proposed): first times. The first defeat of each boss, the first
time each puzzle route is taken, the first lock with each society, the first finish with each
class, and, largest, each ending first seen. Repeats pay a small fraction. This rewards trying
another route and makes grinding one route pointless.

**The meta campaign** (Settled in outline, 12 Nov 2026; the stages are Proposed). The returns
are a story of their own. The entities are not neutral: each return costs you something of
yourself. Across runs the player is either being consumed by them or turning to fight them.

1. *Patronage.* Early returns. The entities guide, the Grimoire opens, nothing seems wrong.
2. *The price.* After a few returns the cost shows. The Grimoire offers two kinds of entry:
   ones that take more from the entities, and ones that resist them. Which the player buys is
   the meta lean, and it replaces a dialogue choice.
3. *Binding.* With enough meta currency and enough runs spent with a companion, that companion
   can be bound to the Grimoire. A bound companion remembers, is present from the first event of
   every later run, and does not leave at the lock. This deliberately breaks the run rule, and
   it is the strongest thing the Grimoire sells. Society leaders can later be swayed the same
   way, which changes what a society offers in a run.
4. *The reckoning.* A final run opens when enough is bound. It ends past the Engine, against the
   entities or as one of them.

How to afford this: the meta campaign is told through Grimoire entries, a short entity scene at
each return, and changed lines for bound companions. It uses no new maps. The reckoning is one
sequence in the Engine ruin with its own rule cards. Stages 3 and 4 are where the project could
double in size, so they need their own budget before any of it is written.

Open:

- Stages 1 and 2 ship in the first release with the three endings; 3 and 4 follow (Settled).
  The save and the entities' early scenes are designed for all four.
- [ ] How many returns before stage 2 and stage 4.
- [ ] Whether foes scale with the return count.

## 6. Technical requirements

- **Save version.** `hero.json` has no version field. Almost every system here changes the save.
  Add a version and a migration path before the first change, not during the fifth.
- **Rule cards as data**, loaded and checked with the story (`rpg/story.go` already validates ids).
- **A progression simulator.** The duel balance test exists. Extend it to play a scripted
  campaign with the AI as the hero and report level, residue and gold per act. Economy numbers
  come out of this, not out of guesses.
- **Rites stay in Go.** Spells are closures over the duel; moving them to YAML is not worth it.
  Grid nodes, book entries, cards and puzzles are data.
- **Sound.** None in the first release (Settled, 12 Nov 2026). Wanted later; do not add audio
  code or assets now.
- **Release target.** The browser build, desktop and phone. Steam is on hold (Settled, 12 Nov
  2026), so no native packaging, controller input or cloud saves are planned.

## 7. What this can become, and what it cannot

It can be a compact, replayable RPG of six to eight hours with a mechanic nobody else has, built
on an engine that already does most of the hard part. A solo author with code agents can finish
that.

It cannot be New Vegas with a gem board. The comparable games in the branching note carry tens of
hours of written reactivity. Here the reactivity has to come from rules, standing numbers and
reskinned sites, with writing spent on the founding quest, the lock, and the three endings. If the
budget in section 4 has to shrink, cut puzzle boards and grid nodes before cutting an ending, and
cut an ending before cutting the return.

## 8. Risks, in order

1. **Puzzle boards may not be fun or detectable.** Prototype one first.
2. **The incremental loop may not hold.** Residue from match quality has to change how people
   play a fight. Test the refinery and a twenty-node grid on the existing Hollow King slice before
   writing any new story.
3. **Writing volume.** Four endings, society voices and reports. Budget per act.
4. **Rule cards break balance.** Every card needs a simulated-duel test.
5. **Art done twice.** Do not produce final art until names and theme text are fixed.

## 9. Open decisions for George

Decided 12 Nov 2026: economy shape, grid replaces stat points, mounts dropped, gold to
facilities and residue to the grid, three acts with the founding quest ending Act 1, the return
as a core run-based loop, the scope budget, gear, henchmen and facilities as the gold sinks,
early return allowed at a cost, the names Alchemy, Orrery and Grimoire, coloured residue with
conversion to colourless, companions who leave at the lock, a meta campaign against or with the
entities.

- [ ] The early-return cost and the meta payouts (5.8).
- [ ] The detail of the meta campaign stages (5.8).
- [ ] Names for the player's society, the rival college, the entities and the meta currency.
- [ ] Remaining Proposed items, to be decided as each comes up in the roadmap: rule cards (5.1), board cap of 10x10 (5.1), rival college as an epilogue variant (5.7).

## 10. Roadmap

`ROADMAP.md` was rewritten on 12 Nov 2026 to follow this file. Meta stages 1 and 2 are in the
first release; stages 3 and 4 are held for the next (Settled).
