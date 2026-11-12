# Design docs for jewel-game

These notes are for code agents working on [GeorgeZack/jewel-game](https://github.com/GeorgeZack/jewel-game) (Go + Ebitengine, branch `master`). They live here as an export. When they go into the repo, put them in `docs/design/`.

The repo already has `README.md` (current Hollow King vertical slice) and `docs/story.md` (YAML story format). There is no existing roadmap or TODO plan. `AGENTS.md` is only Hyprland debug-window placement. This folder is the design intent for the next game; `ROADMAP.md` is the work plan against it.

## Start with DESIGN.md

`DESIGN.md` (draft, 12 Nov 2026) is the consolidated design. It wins where it disagrees with the
two notes below, which remain as research and code mapping. Items it marks **Proposed** are not
decided until George confirms them.

## The two documents

| File | Read it when | It covers |
|---|---|---|
| `society-academy-and-incremental-progression.md` | Changing the Citadel, classes, spells, boards, gems, duels, progression, or mobile play. | Academy lock, incremental layers, classes as templates, combat retune, puzzle duels, board growth. |
| `rival-societies-and-branching-campaign.md` | Changing the campaign, factions, endings, story flags, or Act 3 board/economy modifiers. | Rival societies, branch-and-bottleneck acts, mechanical affiliation, shared maps. |

Read both if a change touches combat *and* the campaign (for example Act 3 Settings, or a puzzle that sets a story flag). Combat and puzzle-duel code mapping lives only in the academy doc. Campaign skeleton and ending packages live only in the branching doc.

## Source of truth

`DESIGN.md` marks each item Settled, Proposed or Open. Build only on Settled items. For Proposed
or Open items, and for any `ROADMAP.md` task marked **Decision needed**, ask George; do not
invent a default and ship it.

These docs override the current fantasy framing in the repo README. They do not override hard engine constraints in `board/` or the story YAML format in `docs/story.md`.

## Hard rules

- Do not rebuild the match-3 engine (`board/`, `board/resolve.go`). Extend it (Settings, authored boards, objectives, palettes).
- Incremental, not idle. Nothing progresses while the game is closed. No overnight timers, AFK chests, build queues, or stamina-for-cash.
- No intentionally cozy design.
- Mobile is first class. The same WASM build is the phone game.
- Theme is steampunk / Victorian occult secret-society academy. Not generic high fantasy.

## Current game vs this plan

HEAD on `master` as of 2fe0f137 (11 Nov 2026, 11:33 PM America/Chicago): a playable fantasy vertical slice. Eight main quests through the Hollow King. One 8x8 adversarial duel mode. Classes lock spells. The Citadel nav is always on; buildings are gold purchases. Phone layout and finger-as-mouse input landed in that commit (`core/phone.go`, `rl/app.go`). Verify on a real device; do not reimplement touch.

What changes, what is new and what is cut are in `DESIGN.md`; the order of work is in `ROADMAP.md`. The two older notes predate several decisions (the alembic table and mounts are cut, the "Ace" restart became the return and the Grimoire, buildings became gold-built facilities), so use them for research links and code mapping only.
