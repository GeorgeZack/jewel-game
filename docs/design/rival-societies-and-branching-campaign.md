# Rival societies and branching campaign

Lean design notes for GeorgeZack/jewel-game, 11 Nov 2026 (America/Chicago). Each ending is defined by a society: the Crown, a join-the-villain cult, or seizing a steampunk or occult Engine. Maps, the academy and the climax are shared across routes. Board rules change at the Act 3 lock. Companion doc: `society-academy-and-incremental-progression.md`.

## Brief and decisions

The shipped version of Morrowind doesn't let you join Dagoth Ur; the encounter ends in combat ([UESP](https://en.uesp.net/wiki/Morrowind:Dagoth_Ur_(god))). UESP lists unused House Dagoth assets, but not a finished quest to join him and take the Akulakhan ([UESP unused content](https://en.uesp.net/wiki/Morrowind:Unused_Content)). This campaign fills that gap in a steampunk occult-studies setting. You can join a forbidden society and drive out the Crown, rise in the establishment and pacify the rival colleges, or seize the Engine for your own society.

- [x] Three or four acts, with founding the academy mid-game as the first lean.
- [x] Endings include a join-the-villain route and a seize-the-Engine route.
- [x] Choices change match-3 rules or the gem economy, not only dialogue.
- [x] Rival secret societies, the Crown or establishment, and a steampunk or occult Engine.
- [ ] Exact society names (open; the names below are placeholders).
- [ ] Which act the founding quest occupies (open).

## What other games do

- **Fallout: New Vegas**: four endings (NCR, Legion, House, and Independent with Yes Man) on one Mojave map, locked in at Hoover Dam. The Legion is the join-the-antagonist route, and Yes Man seizes the apparatus ([endings](https://fallout.wiki/wiki/Endings_(Fallout:_New_Vegas)), [Strip endings](https://fallout.wiki/wiki/New_Vegas_Strip/Endings)).
- **Pillars of Eternity II: Deadfire**: four factions. You can work for several until a point of no return (The Coming Storm / Queen's Palace), and then the others turn hostile. Ukaizo can be taken independently ([factions](https://pillars.fandom.com/wiki/Factions_(Deadfire))).
- **Tyranny**: villainy is the premise. You can challenge Kyros or swear fealty by several routes, with epilogue slides ([endings](https://tyranny.fandom.com/wiki/Tyranny_endings)).
- **Arcanum**: siding with Kerghan is a true villain ending ([endings](https://arcanum.fandom.com/wiki/Endings)).
- **The Witcher 2**: choosing Roche or Iorveth gives largely different Act 2 maps (Vergen or the Kaedweni camp). That's too expensive to copy ([Vergen](https://witcher.fandom.com/wiki/Vergen)).
- **Dragon Age: Origins**: a linear spine with a Landsmeet bottleneck. Loghain can be recruited, and there are epilogue slides ([epilogue](https://dragonage.miraheze.org/wiki/Epilogue_(Origins))).
- **Pathfinder: Wrath of the Righteous**: Lich, Demon and Swarm paths let the player become the threat. Steal the idea, not Owlcat's volume of content ([mythic paths](https://pathfinderwrathoftherighteous.wiki.fextralife.com/Mythic+Paths)).
- **Disco Elysium**: almost no split maps; flags and vision quests carry the branching. That's cheap on maps and expensive on writing, and not enough when the board must also change ([political vision quest](https://discoelysium.fandom.com/wiki/Political_vision_quest)).
- **Alpha Protocol**: shared missions, with reputation and mission order doing the branching.
- **Planescape: Torment**: a shared late dungeon, where you merge with or kill the Transcendent One.
- Ultima IV and Darklands weren't used as templates.

## Scope and technique

Use branch-and-bottleneck structure, one hard lock in Act 3, state flags and reactive reports, shared maps reskinned by whoever holds them, epilogue slides, and affiliation that changes the mechanics (board rules and gem income).

Don't build a unique second act, a full mythic-path amount of content, or endings that are only slides. Plan four ending packages and one climax, not four campaigns. Flags, report text, `ApplyStep` modifiers and a last-act Settings card are cheaper than unique maps.

## Campaign skeleton

| Placeholder | Role | Analog |
|---|---|---|
| The Crown / establishment | Rise inside it, pacify the rival colleges and hold the ruin in the name of order. | Empire / NCR / Royal Deadfire Company |
| Forbidden society / ruin-cult | Join it, fight the Crown's governor or a local god-king, and drive the occupation out. | Dagoth Ur / Legion / a Tyranny overlord path |
| Independent claim | Seize the Engine for your society and answer to no one. | Yes Man / independent Ukaizo / seizing the Akulakhan |
| Rival college (optional) | Courted, pacified or broken. Gets a full ending or just a slide. | Telvanni / Príncipi |

- **Act 1**: you're a solo explorer and every society courts you. The academy is locked, and there are no unique maps.
- **Founding quest**: the first lean, about who helps you claim the headquarters or whether you take it alone. It isn't a hard lock yet.
- **Act 2**: the academy opens. You work for several societies on shared maps, and faculties, the refinery scored by play, and field chapters come online. Standing with one society can poison others.
- **Act 3**: a named lock. The societies you rejected turn hostile, and board rules shift.
- **Climax**: one Engine ruin. The Crown seals it, the cult wakes or serves it, and the independent route seizes it. An optional fourth route shatters it or hands it off. Then come epilogue slides for the societies, the academy and companions.

## Board and economy

Apply these at the Act 3 lock, on the existing jewel-game loop (`rpg/duel.go` `ApplyStep`, `board.Settings`, `assets/story/foes.yaml`, `startDuel`). The full mapping for combat and puzzle duels is in the academy doc. Numbers are deliberately left unset.

| Path | Board | Economy | Academy |
|---|---|---|---|
| Crown | Safer, defensive rules; maybe ban a volatile color. | Stable dues from wins; the refinery favors clean clears. | Field chapters run faster in Crown territory. |
| Join the cult | The forbidden color unlocks; longer chains, with backlash if mismanaged. | That residue buys unique grid nodes; Crown dues dry up. | Jobs in official territory become raids; some faculties are rewritten as cult holdings. |
| Seize the Engine | Once per fight, spend academy stock to rewrite one Settings flag mid-battle (lock, gravity, Force). The Engine is a tool. | Stock is scarce, and overuse starves the grid. | No regular Crown pay; seized stores fund the Engine. |
| Rival college | Allied expeditions add that faculty's element as a modifier. | Trade replaces dues; unusual recipes come earlier. | The academy seat is a bargain, and broken bargains weaken the modifier. |

Acts 1 and 2 use the default rules, and the founding quest can give a small, reversible lean. A New Game cycle is the place to try another ending; don't offer a mid-act allegiance respec.
